package service

import (
	"database/sql"
	"fmt"
)

type SettingItem struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	ValueType string `json:"valueType"`
	IsSecret  bool   `json:"isSecret"`
	Group     string `json:"group"`
	UpdatedAt string `json:"updatedAt"`
}

type SettingsService struct {
	db        *sql.DB
	cryptoSvc *CryptoService
}

func NewSettingsService(db *sql.DB, cryptoSvc *CryptoService) *SettingsService {
	return &SettingsService{
		db:        db,
		cryptoSvc: cryptoSvc,
	}
}

func (s *SettingsService) GetSetting(key string) (SettingItem, error) {
	var item SettingItem
	if s == nil || s.db == nil {
		return item, fmt.Errorf("settings service db is nil")
	}

	row := s.db.QueryRow(`
SELECT setting_key, setting_value, value_type, is_secret, setting_group, updated_at
FROM app_settings
WHERE setting_key = ?
`, key)

	var isSecret int
	if err := row.Scan(&item.Key, &item.Value, &item.ValueType, &isSecret, &item.Group, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return item, fmt.Errorf("setting not found: %s", key)
		}
		return item, fmt.Errorf("failed to get setting %s: %w", key, err)
	}

	item.IsSecret = isSecret == 1
	return item, nil
}

func (s *SettingsService) GetSettingsByGroup(group string) ([]SettingItem, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("settings service db is nil")
	}

	rows, err := s.db.Query(`
SELECT setting_key, setting_value, value_type, is_secret, setting_group, updated_at
FROM app_settings
WHERE setting_group = ?
ORDER BY setting_key
`, group)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings by group: %w", err)
	}
	defer rows.Close()

	var items []SettingItem
	for rows.Next() {
		var item SettingItem
		var isSecret int

		if err := rows.Scan(
			&item.Key,
			&item.Value,
			&item.ValueType,
			&isSecret,
			&item.Group,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan setting row: %w", err)
		}

		item.IsSecret = isSecret == 1

		// 민감값은 평문을 프론트에 주지 않음
		if item.IsSecret {
			item.Value = ""
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *SettingsService) SaveSetting(key, value, valueType string, isSecret bool, group string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("settings service db is nil")
	}

	secretInt := 0
	if isSecret {
		secretInt = 1
	}

	_, err := s.db.Exec(`
INSERT INTO app_settings (setting_key, setting_value, value_type, is_secret, setting_group, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(setting_key) DO UPDATE SET
  setting_value = excluded.setting_value,
  value_type = excluded.value_type,
  is_secret = excluded.is_secret,
  setting_group = excluded.setting_group,
  updated_at = excluded.updated_at
`, key, value, valueType, secretInt, group, nowText())
	if err != nil {
		return fmt.Errorf("failed to save setting %s: %w", key, err)
	}

	return nil
}

func (s *SettingsService) SaveSettings(items []SettingItem) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("settings service db is nil")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin settings tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
INSERT INTO app_settings (setting_key, setting_value, value_type, is_secret, setting_group, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(setting_key) DO UPDATE SET
  setting_value = excluded.setting_value,
  value_type = excluded.value_type,
  is_secret = excluded.is_secret,
  setting_group = excluded.setting_group,
  updated_at = excluded.updated_at
`)
	if err != nil {
		return fmt.Errorf("failed to prepare settings stmt: %w", err)
	}
	defer stmt.Close()

	for _, item := range items {
		secretInt := 0
		if item.IsSecret {
			secretInt = 1
		}

		if _, err := stmt.Exec(
			item.Key,
			item.Value,
			item.ValueType,
			secretInt,
			item.Group,
			nowText(),
		); err != nil {
			return fmt.Errorf("failed to save setting %s: %w", item.Key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit settings tx: %w", err)
	}

	return nil
}

// SaveSecretSettingWithPin encrypts a secret value using the current PIN-derived key.
func (s *SettingsService) SaveSecretSettingWithPin(key, plainValue, valueType, group, pin string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("settings service db is nil")
	}
	if s.cryptoSvc == nil {
		return fmt.Errorf("crypto service is nil")
	}

	if stringsTrim(plainValue) == "" {
		return s.SaveSetting(key, "", valueType, true, group)
	}

	enc, err := s.cryptoSvc.EncryptString(pin, plainValue)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret setting %s: %w", key, err)
	}

	return s.SaveSetting(key, enc, valueType, true, group)
}

// HasSecretSetting returns whether encrypted value exists in DB.
func (s *SettingsService) HasSecretSetting(key string) (bool, error) {
	item, err := s.GetSetting(key)
	if err != nil {
		return false, err
	}
	return stringsTrim(item.Value) != "", nil
}

// DecryptSecretSetting is optional internal helper.
// 프론트에는 보통 노출하지 않는 것을 권장.
func (s *SettingsService) DecryptSecretSetting(key, pin string) (string, error) {
	if s == nil || s.db == nil {
		return "", fmt.Errorf("settings service db is nil")
	}
	if s.cryptoSvc == nil {
		return "", fmt.Errorf("crypto service is nil")
	}

	item, err := s.GetSetting(key)
	if err != nil {
		return "", err
	}
	if stringsTrim(item.Value) == "" {
		return "", nil
	}

	plain, err := s.cryptoSvc.DecryptString(pin, item.Value)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret setting %s: %w", key, err)
	}

	return plain, nil
}

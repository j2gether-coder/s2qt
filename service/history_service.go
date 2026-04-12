package service

import (
	"database/sql"
	"fmt"
)

type HistoryMaster struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	BibleText  string `json:"bibleText"`
	Hymn       string `json:"hymn"`
	Preacher   string `json:"preacher"`
	ChurchName string `json:"churchName"`
	SermonDate string `json:"sermonDate"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

type HistoryQTJSON struct {
	ID           int64  `json:"id"`
	HistoryID    int64  `json:"historyId"`
	Audience     string `json:"audience"`
	QTResultJSON string `json:"qtResultJson"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type SaveHistoryRequest struct {
	Title        string `json:"title"`
	BibleText    string `json:"bibleText"`
	Hymn         string `json:"hymn"`
	Preacher     string `json:"preacher"`
	ChurchName   string `json:"churchName"`
	SermonDate   string `json:"sermonDate"`
	Audience     string `json:"audience"`
	QTResultJSON string `json:"qtResultJson"`
}

type HistoryService struct {
	db *sql.DB
}

func NewHistoryService(db *sql.DB) *HistoryService {
	return &HistoryService{db: db}
}

func (s *HistoryService) SaveHistory(req SaveHistoryRequest) (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("history service db is nil")
	}
	if stringsTrim(req.Title) == "" || stringsTrim(req.BibleText) == "" {
		return 0, fmt.Errorf("title and bible text are required")
	}
	if stringsTrim(req.Audience) == "" || stringsTrim(req.QTResultJSON) == "" {
		return 0, fmt.Errorf("audience and qt result json are required")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin history tx: %w", err)
	}
	defer tx.Rollback()

	now := nowText()

	res, err := tx.Exec(`
INSERT INTO history_master (title, bible_text, hymn, preacher, church_name, sermon_date, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, req.Title, req.BibleText, req.Hymn, req.Preacher, req.ChurchName, req.SermonDate, now, now)
	if err != nil {
		return 0, fmt.Errorf("failed to insert history master: %w", err)
	}

	historyID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get history master id: %w", err)
	}

	if _, err := tx.Exec(`
INSERT INTO history_qt_json (history_id, audience, qt_result_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
`, historyID, req.Audience, req.QTResultJSON, now, now); err != nil {
		return 0, fmt.Errorf("failed to insert history qt json: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit history tx: %w", err)
	}
	return historyID, nil
}

func (s *HistoryService) ListHistory() ([]HistoryMaster, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("history service db is nil")
	}

	rows, err := s.db.Query(`
SELECT id, title, bible_text, hymn, preacher, church_name, sermon_date, created_at, updated_at
FROM history_master
ORDER BY created_at DESC, id DESC
`)
	if err != nil {
		return nil, fmt.Errorf("failed to query history list: %w", err)
	}
	defer rows.Close()

	var items []HistoryMaster
	for rows.Next() {
		var item HistoryMaster
		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.BibleText,
			&item.Hymn,
			&item.Preacher,
			&item.ChurchName,
			&item.SermonDate,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan history row: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *HistoryService) GetHistory(historyID int64) (HistoryMaster, error) {
	var item HistoryMaster
	if s == nil || s.db == nil {
		return item, fmt.Errorf("history service db is nil")
	}

	row := s.db.QueryRow(`
SELECT id, title, bible_text, hymn, preacher, church_name, sermon_date, created_at, updated_at
FROM history_master
WHERE id = ?
`, historyID)

	if err := row.Scan(
		&item.ID,
		&item.Title,
		&item.BibleText,
		&item.Hymn,
		&item.Preacher,
		&item.ChurchName,
		&item.SermonDate,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return item, fmt.Errorf("history not found: %d", historyID)
		}
		return item, fmt.Errorf("failed to get history: %w", err)
	}

	return item, nil
}

func (s *HistoryService) GetHistoryQTJSON(historyID int64, audience string) (HistoryQTJSON, error) {
	var item HistoryQTJSON
	if s == nil || s.db == nil {
		return item, fmt.Errorf("history service db is nil")
	}

	row := s.db.QueryRow(`
SELECT id, history_id, audience, qt_result_json, created_at, updated_at
FROM history_qt_json
WHERE history_id = ? AND audience = ?
ORDER BY id DESC
LIMIT 1
`, historyID, audience)

	if err := row.Scan(
		&item.ID,
		&item.HistoryID,
		&item.Audience,
		&item.QTResultJSON,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return item, fmt.Errorf("history qt json not found: history_id=%d audience=%s", historyID, audience)
		}
		return item, fmt.Errorf("failed to get history qt json: %w", err)
	}

	return item, nil
}

func (s *HistoryService) DeleteHistory(historyID int64) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("history service db is nil")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin delete history tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM history_qt_json WHERE history_id = ?`, historyID); err != nil {
		return fmt.Errorf("failed to delete history qt json rows: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM history_master WHERE id = ?`, historyID); err != nil {
		return fmt.Errorf("failed to delete history master row: %w", err)
	}

	return tx.Commit()
}

func (s *HistoryService) UpsertHistoryQTJSON(historyID int64, audience, qtJSON string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("history service db is nil")
	}
	if historyID <= 0 {
		return fmt.Errorf("invalid history id")
	}

	_, err := s.db.Exec(`
INSERT INTO history_qt_json (history_id, audience, qt_result_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
`, historyID, audience, qtJSON, nowText(), nowText())
	if err != nil {
		return fmt.Errorf("failed to upsert history qt json: %w", err)
	}
	return nil
}

package service

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var dbInstance *sql.DB

// OpenSQLite opens or returns the singleton SQLite connection.
func OpenSQLite(dbPath string) (*sql.DB, error) {
	if dbInstance != nil {
		return dbInstance, nil
	}

	if stringsTrim(dbPath) == "" {
		return nil, fmt.Errorf("db path is empty")
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping sqlite: %w", err)
	}

	if err := enablePragmas(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	dbInstance = db
	return dbInstance, nil
}

// InitSQLite creates required tables and indexes.
func InitSQLite(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	stmts := []string{
		`
CREATE TABLE IF NOT EXISTS app_settings (
    setting_key   TEXT PRIMARY KEY,
    setting_value TEXT,
    value_type    TEXT NOT NULL DEFAULT 'text',
    is_secret     INTEGER NOT NULL DEFAULT 0,
    setting_group TEXT NOT NULL DEFAULT 'general',
    updated_at    TEXT NOT NULL
);`,
		`
CREATE TABLE IF NOT EXISTS history_master (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    title        TEXT NOT NULL,
    bible_text   TEXT NOT NULL,
    hymn         TEXT,
    preacher     TEXT,
    church_name  TEXT,
    sermon_date  TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);`,
		`
CREATE TABLE IF NOT EXISTS history_step1 (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    history_id        INTEGER NOT NULL,
    audience          TEXT NOT NULL,
    step1_result_json TEXT NOT NULL,
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL,
    FOREIGN KEY (history_id) REFERENCES history_master(id)
);`,
		`CREATE INDEX IF NOT EXISTS idx_history_master_created_at ON history_master(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_history_master_title ON history_master(title);`,
		`CREATE INDEX IF NOT EXISTS idx_history_step1_history_audience ON history_step1(history_id, audience);`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to exec init statement: %w", err)
		}
	}

	return EnsureSettingsDefaults(db)
}

// EnsureSettingsDefaults inserts missing setting keys.
func EnsureSettingsDefaults(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	type seed struct {
		Key       string
		Value     string
		ValueType string
		IsSecret  int
		Group     string
	}

	seeds := []seed{
		{"security.pin_enabled", "false", "boolean", 0, "security"},
		{"security.pin_hash", "", "password", 1, "security"},
		{"security.pin_length", "6", "number", 0, "security"},
		{"security.unlock_scope", "session", "text", 0, "security"},

		{"llm.provider", "openai", "text", 0, "llm"},
		{"llm.api_key", "", "password", 1, "llm"},
		{"llm.model", "", "text", 0, "llm"},
		{"llm.mode", "manual", "text", 0, "llm"},
		{"llm.enabled", "false", "boolean", 0, "llm"},

		{"smtp.enabled", "false", "boolean", 0, "smtp"},
		{"smtp.from_email", "", "text", 0, "smtp"},
		{"smtp.host", "", "text", 0, "smtp"},
		{"smtp.port", "587", "number", 0, "smtp"},
		{"smtp.username", "", "text", 0, "smtp"},
		{"smtp.password", "", "password", 1, "smtp"},
		{"smtp.security", "tls", "text", 0, "smtp"},

		{"church.name", "", "text", 0, "church"},
		{"church.logo_path", "", "text", 0, "church"},
		{"church.homepage_url", "", "url", 0, "church"},
		{"church.default_footer_text", "", "multiline", 0, "church"},

		{"license.type", "freeware", "text", 0, "license"},
		{"license.status", "active", "text", 0, "license"},
		{"license.expire_date", "", "text", 0, "license"},
		{"app.version", "1.0.0", "text", 0, "license"},
		{"app.guide_text", "", "multiline", 0, "license"},
	}

	now := nowText()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin settings seed tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
INSERT OR IGNORE INTO app_settings
(setting_key, setting_value, value_type, is_secret, setting_group, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		return fmt.Errorf("failed to prepare settings seed stmt: %w", err)
	}
	defer stmt.Close()

	for _, s := range seeds {
		if _, err := stmt.Exec(s.Key, s.Value, s.ValueType, s.IsSecret, s.Group, now); err != nil {
			return fmt.Errorf("failed to seed setting %s: %w", s.Key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit settings seed tx: %w", err)
	}
	return nil
}

func enablePragmas(db *sql.DB) error {
	pragmas := []string{
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA synchronous = NORMAL;`,
		`PRAGMA busy_timeout = 5000;`,
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("failed to apply pragma %q: %w", p, err)
		}
	}
	return nil
}

func nowText() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func stringsTrim(s string) string {
	return strings.TrimSpace(s)
}

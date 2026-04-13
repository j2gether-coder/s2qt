package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"s2qt/util"
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

type ReworkPrepareResponse struct {
	Success      bool   `json:"success"`
	HistoryID    int64  `json:"historyId"`
	Audience     string `json:"audience"`
	Title        string `json:"title"`
	BibleText    string `json:"bibleText"`
	Hymn         string `json:"hymn"`
	Preacher     string `json:"preacher"`
	ChurchName   string `json:"churchName"`
	SermonDate   string `json:"sermonDate"`
	Message      string `json:"message"`
	TempJSONPath string `json:"tempJsonPath"`
}

type flatQTStep2Data struct {
	Audience        string `json:"audience"`
	Title           string `json:"title"`
	BibleText       string `json:"bibleText"`
	Hymn            string `json:"hymn"`
	Preacher        string `json:"preacher"`
	ChurchName      string `json:"churchName"`
	SermonDate      string `json:"sermonDate"`
	SourceURL       string `json:"sourceURL"`
	SummaryTitle    string `json:"summaryTitle"`
	SummaryBody     string `json:"summaryBody"`
	MessageTitle1   string `json:"messageTitle1"`
	MessageBody1    string `json:"messageBody1"`
	MessageTitle2   string `json:"messageTitle2"`
	MessageBody2    string `json:"messageBody2"`
	MessageTitle3   string `json:"messageTitle3"`
	MessageBody3    string `json:"messageBody3"`
	ReflectionItem1 string `json:"reflectionItem1"`
	ReflectionItem2 string `json:"reflectionItem2"`
	ReflectionItem3 string `json:"reflectionItem3"`
	PrayerTitle     string `json:"prayerTitle"`
	PrayerBody      string `json:"prayerBody"`
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
ORDER BY updated_at DESC, id DESC
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

func (s *HistoryService) PrepareReworkFromHistory(historyID int64, audience string) (ReworkPrepareResponse, error) {
	var resp ReworkPrepareResponse

	if s == nil || s.db == nil {
		return resp, fmt.Errorf("history service db is nil")
	}
	if historyID <= 0 {
		return resp, fmt.Errorf("invalid history id")
	}
	if stringsTrim(audience) == "" {
		return resp, fmt.Errorf("audience is required")
	}

	master, err := s.GetHistory(historyID)
	if err != nil {
		return resp, err
	}

	qtRow, err := s.GetHistoryQTJSON(historyID, audience)
	if err != nil {
		return resp, err
	}

	flat, err := parseFlatQTStep2JSON(qtRow.QTResultJSON, audience)
	if err != nil {
		return resp, err
	}

	doc := buildQTSectionDocFromFlat(flat, audience)

	tempPath, err := writeQTSectionDocToTempJSON(doc)
	if err != nil {
		return resp, err
	}

	title := getStringFromMap(doc.Metadata, "title")
	bibleText := getStringFromMap(doc.Metadata, "bible_text")
	hymn := getStringFromMap(doc.Metadata, "hymn")
	preacher := getStringFromMap(doc.Metadata, "preacher")
	churchName := getStringFromMap(doc.Metadata, "church_name")
	sermonDate := getStringFromMap(doc.Metadata, "sermon_date")

	if stringsTrim(title) == "" {
		title = master.Title
	}
	if stringsTrim(bibleText) == "" {
		bibleText = master.BibleText
	}
	if stringsTrim(hymn) == "" {
		hymn = master.Hymn
	}
	if stringsTrim(preacher) == "" {
		preacher = master.Preacher
	}
	if stringsTrim(churchName) == "" {
		churchName = master.ChurchName
	}
	if stringsTrim(sermonDate) == "" {
		sermonDate = master.SermonDate
	}

	resp = ReworkPrepareResponse{
		Success:      true,
		HistoryID:    master.ID,
		Audience:     audience,
		Title:        title,
		BibleText:    bibleText,
		Hymn:         hymn,
		Preacher:     preacher,
		ChurchName:   churchName,
		SermonDate:   sermonDate,
		Message:      "temp.json restored successfully",
		TempJSONPath: tempPath,
	}

	return resp, nil
}

func parseFlatQTStep2JSON(jsonText string, expectedAudience string) (*flatQTStep2Data, error) {
	if stringsTrim(jsonText) == "" {
		return nil, fmt.Errorf("qt result json is empty")
	}

	var flat flatQTStep2Data
	if err := json.Unmarshal([]byte(jsonText), &flat); err != nil {
		return nil, fmt.Errorf("invalid flat qt result json: %w", err)
	}

	if stringsTrim(flat.Audience) == "" {
		flat.Audience = stringsTrim(expectedAudience)
	}
	if stringsTrim(flat.Audience) == "" {
		return nil, fmt.Errorf("flat qt result json audience is empty")
	}
	if stringsTrim(expectedAudience) != "" && stringsTrim(flat.Audience) != stringsTrim(expectedAudience) {
		return nil, fmt.Errorf("flat qt result json audience mismatch: expected=%s actual=%s", expectedAudience, flat.Audience)
	}

	if stringsTrim(flat.Title) == "" &&
		stringsTrim(flat.BibleText) == "" &&
		stringsTrim(flat.SummaryBody) == "" &&
		stringsTrim(flat.MessageBody1) == "" &&
		stringsTrim(flat.PrayerBody) == "" {
		return nil, fmt.Errorf("flat qt result json fields are empty")
	}

	return &flat, nil
}

func buildQTSectionDocFromFlat(flat *flatQTStep2Data, expectedAudience string) *QTSectionDoc {
	audience := stringsTrim(flat.Audience)
	if audience == "" {
		audience = stringsTrim(expectedAudience)
	}

	return &QTSectionDoc{
		Version:  "1.0",
		DocType:  "qt",
		Audience: audience,
		Template: "qt_classic",
		Metadata: map[string]any{
			"title":       strings.TrimSpace(flat.Title),
			"bible_text":  strings.TrimSpace(flat.BibleText),
			"hymn":        strings.TrimSpace(flat.Hymn),
			"preacher":    strings.TrimSpace(flat.Preacher),
			"church_name": strings.TrimSpace(flat.ChurchName),
			"sermon_date": strings.TrimSpace(flat.SermonDate),
			"source_url":  strings.TrimSpace(flat.SourceURL),
		},
		Sections: []QTSectionData{
			{
				Type:  "summary",
				Title: step2firstNonEmpty(flat.SummaryTitle, "🌿 말씀의 창: 본문 요약"),
				Blocks: []QTBlockData{
					{Type: "paragraph", Text: strings.TrimSpace(flat.SummaryBody)},
				},
			},
			{
				Type:  "message",
				Title: "✨ 오늘의 메시지",
				Blocks: []QTBlockData{
					{Type: "message_title", Text: strings.TrimSpace(flat.MessageTitle1)},
					{Type: "paragraph", Text: strings.TrimSpace(flat.MessageBody1)},
					{Type: "message_title", Text: strings.TrimSpace(flat.MessageTitle2)},
					{Type: "paragraph", Text: strings.TrimSpace(flat.MessageBody2)},
					{Type: "message_title", Text: strings.TrimSpace(flat.MessageTitle3)},
					{Type: "paragraph", Text: strings.TrimSpace(flat.MessageBody3)},
				},
			},
			{
				Type:  "reflection",
				Title: "🔍 깊은 묵상과 적용",
				Blocks: []QTBlockData{
					{
						Type: "list",
						Items: []string{
							strings.TrimSpace(flat.ReflectionItem1),
							strings.TrimSpace(flat.ReflectionItem2),
							strings.TrimSpace(flat.ReflectionItem3),
						},
					},
				},
			},
			{
				Type:  "prayer",
				Title: step2firstNonEmpty(flat.PrayerTitle, "🙏 오늘의 기도"),
				Blocks: []QTBlockData{
					{Type: "paragraph", Text: strings.TrimSpace(flat.PrayerBody)},
				},
			},
		},
	}
}

func writeQTSectionDocToTempJSON(doc *QTSectionDoc) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("qt section doc is nil")
	}

	paths, err := util.GetAppPaths()
	if err != nil {
		return "", fmt.Errorf("failed to get app paths: %w", err)
	}

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal qt section doc: %w", err)
	}

	if err := os.WriteFile(paths.TempJson, b, 0o644); err != nil {
		return "", fmt.Errorf("failed to write temp.json: %w", err)
	}

	return paths.TempJson, nil
}

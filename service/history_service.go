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

type HistoryListQuery struct {
	Keyword  string `json:"keyword"`
	Audience string `json:"audience"`
	SortKey  string `json:"sortKey"`
	SortDir  string `json:"sortDir"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
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
	Series       string `json:"series"`
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
	Audience          string   `json:"audience"`
	Title             string   `json:"title"`
	BibleText         string   `json:"bibleText"`
	BiblePassageText  string   `json:"bible_passage_text"`
	Hymn              string   `json:"hymn"`
	Preacher          string   `json:"preacher"`
	ChurchName        string   `json:"churchName"`
	SermonDate        string   `json:"sermonDate"`
	SourceURL         string   `json:"sourceURL"`
	SupportScriptures []string `json:"support_scriptures"`
	SummaryTitle      string   `json:"summaryTitle"`
	SummaryBody       string   `json:"summaryBody"`
	MessageTitle1     string   `json:"messageTitle1"`
	MessageBody1      string   `json:"messageBody1"`
	MessageTitle2     string   `json:"messageTitle2"`
	MessageBody2      string   `json:"messageBody2"`
	MessageTitle3     string   `json:"messageTitle3"`
	MessageBody3      string   `json:"messageBody3"`
	ReflectionItem1   string   `json:"reflectionItem1"`
	ReflectionItem2   string   `json:"reflectionItem2"`
	ReflectionItem3   string   `json:"reflectionItem3"`
	PrayerTitle       string   `json:"prayerTitle"`
	PrayerBody        string   `json:"prayerBody"`
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

func (s *HistoryService) CountHistory(query HistoryListQuery) (int, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("history service db is nil")
	}

	q := normalizeHistoryListQuery(query)
	whereSQL, args := buildHistoryListWhere(q)

	sqlText := `
SELECT COUNT(*)
FROM history_master hm
` + whereSQL

	var total int
	if err := s.db.QueryRow(sqlText, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("failed to count history: %w", err)
	}

	return total, nil
}

func (s *HistoryService) ListHistoryPaged(query HistoryListQuery) ([]HistoryMaster, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("history service db is nil")
	}

	q := normalizeHistoryListQuery(query)
	whereSQL, args := buildHistoryListWhere(q)

	sortColumn := resolveHistorySortColumn(q.SortKey)
	sortDir := resolveHistorySortDir(q.SortDir)
	offset := (q.Page - 1) * q.PageSize

	sqlText := `
SELECT
  hm.id,
  hm.title,
  hm.bible_text,
  hm.hymn,
  hm.preacher,
  hm.church_name,
  hm.sermon_date,
  hm.created_at,
  hm.updated_at
FROM history_master hm
` + whereSQL + `
ORDER BY ` + sortColumn + ` ` + sortDir + `, hm.id DESC
LIMIT ? OFFSET ?
`

	args = append(args, q.PageSize, offset)

	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query paged history list: %w", err)
	}
	defer rows.Close()

	items := make([]HistoryMaster, 0, q.PageSize)
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
			return nil, fmt.Errorf("failed to scan paged history row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading paged history rows: %w", err)
	}

	return items, nil
}

func normalizeHistoryListQuery(q HistoryListQuery) HistoryListQuery {
	q.Keyword = strings.TrimSpace(q.Keyword)
	q.Audience = strings.TrimSpace(q.Audience)
	q.SortKey = strings.TrimSpace(q.SortKey)
	q.SortDir = strings.TrimSpace(q.SortDir)

	if q.Audience == "" {
		q.Audience = "all"
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 10
	}

	switch q.SortKey {
	case "createdAt", "title", "bibleText":
	default:
		q.SortKey = "createdAt"
	}

	switch strings.ToLower(q.SortDir) {
	case "asc", "desc":
		q.SortDir = strings.ToLower(q.SortDir)
	default:
		q.SortDir = "desc"
	}

	return q
}

func buildHistoryListWhere(q HistoryListQuery) (string, []any) {
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 3)

	if q.Keyword != "" {
		pattern := "%" + q.Keyword + "%"
		clauses = append(clauses, `(hm.title LIKE ? OR hm.bible_text LIKE ?)`)
		args = append(args, pattern, pattern)
	}

	if q.Audience != "" && q.Audience != "all" {
		clauses = append(clauses, `
EXISTS (
  SELECT 1
  FROM history_qt_json hq
  WHERE hq.history_id = hm.id
    AND hq.audience = ?
)
`)
		args = append(args, q.Audience)
	}

	if len(clauses) == 0 {
		return "", args
	}

	return "WHERE " + strings.Join(clauses, " AND "), args
}

func resolveHistorySortColumn(sortKey string) string {
	switch sortKey {
	case "title":
		return "hm.title"
	case "bibleText":
		return "hm.bible_text"
	case "createdAt":
		fallthrough
	default:
		return "hm.created_at"
	}
}

func resolveHistorySortDir(sortDir string) string {
	if strings.EqualFold(strings.TrimSpace(sortDir), "asc") {
		return "ASC"
	}
	return "DESC"
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

	doc, llmDoc, err := restoreQTSectionDoc(qtRow.QTResultJSON, audience)
	if err != nil {
		return resp, err
	}

	tempPath, err := writeQTSectionDocToTempJSON(doc)
	if err != nil {
		return resp, err
	}

	// 인포그래픽은 저장된 원본에서 다시 렌더한다.
	// 구버전 이력(llmDoc == nil)이면 파일을 비우기만 한다.
	restoreInfographicFile(llmDoc, audience, master)

	// series는 metadata에만 있다. history_master.title에는 시리즈가 합쳐진
	// 라벨이 들어 있어 거기서 되뽑을 수 없다.
	series := getStringFromMap(doc.Metadata, "series")

	title := getStringFromMap(doc.Metadata, "title")
	bibleText := getStringFromMap(doc.Metadata, "bible_text")
	hymn := getStringFromMap(doc.Metadata, "hymn")
	preacher := getStringFromMap(doc.Metadata, "preacher")
	churchName := getStringFromMap(doc.Metadata, "church_name")
	sermonDate := getStringFromMap(doc.Metadata, "sermon_date")

	// master.Title은 "시리즈|||제목" 합친 라벨이므로 그대로 쓰면 제목에 시리즈가 섞인다.
	labelSeries, labelTitle := splitHistoryTitle(master.Title)

	if stringsTrim(series) == "" {
		series = labelSeries
	}
	if stringsTrim(title) == "" {
		title = labelTitle
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
		Series:       series,
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

// restoreQTSectionDoc는 작업내역에 저장된 JSON을 temp.json용 문서로 되살린다.
//
// qt_result_json 컬럼에는 두 가지 포맷이 시간순으로 섞여 있다.
//
//	신규(2026-08-31~) : Step1이 저장한 LLM 원본. sections 배열이 있다.
//	구버전            : Step2가 저장한 평면 페이로드. sections가 없다.
//
// sections 유무로 갈리며, 구버전 경로는 지금까지 쌓인 이력의 유일한 복원 수단이므로
// 그대로 유지한다.
//
// 두 번째 반환값은 신규 포맷일 때만 채워지며, 인포그래픽 복원에 쓴다.
// 구버전 이력에는 인포그래픽 데이터가 없으므로 nil이다.
func restoreQTSectionDoc(jsonText string, audience string) (*QTSectionDoc, *QTLLMDoc, error) {
	if stringsTrim(jsonText) == "" {
		return nil, nil, fmt.Errorf("qt result json is empty")
	}

	var probe struct {
		Sections []json.RawMessage `json:"sections"`
	}
	_ = json.Unmarshal([]byte(jsonText), &probe)

	if len(probe.Sections) == 0 {
		flat, err := parseFlatQTStep2JSON(jsonText, audience)
		if err != nil {
			return nil, nil, err
		}
		return buildQTSectionDocFromFlat(flat, audience), nil, nil
	}

	var doc QTLLMDoc
	if err := json.Unmarshal([]byte(jsonText), &doc); err != nil {
		return nil, nil, fmt.Errorf("invalid llm result json: %w", err)
	}

	qt := doc.QTSectionDoc
	qt.Version = "1.0"
	if stringsTrim(qt.DocType) == "" {
		qt.DocType = "qt"
	}
	if stringsTrim(qt.Audience) == "" {
		qt.Audience = stringsTrim(audience)
	}
	if stringsTrim(qt.Template) == "" {
		qt.Template = "qt_classic"
	}

	return &qt, &doc, nil
}

// restoreInfographicFile은 재작업 시 sermon_summary.md를 다시 만든다.
// 조건을 만족하지 않으면 파일을 0바이트로 비워, 이전 설교의 인포그래픽이
// 이번 작업의 산출물로 오인되지 않게 한다.
//
// 재작업에서도 인포그래픽이 살아나는 것은 md를 전사문이 아니라
// 저장된 JSON에서 렌더하기 때문이다. (전사문은 작업내역에 저장되지 않는다)
//
// 제목과 성경본문은 이력 마스터에 저장된 값(= Step1 당시 화면 기본정보)을 쓴다.
func restoreInfographicFile(doc *QTLLMDoc, audience string, master HistoryMaster) {
	paths, err := util.GetAppPaths()
	if err != nil || paths == nil {
		LogError("rework: app paths 조회 실패로 sermon_summary.md를 갱신하지 못했습니다")
		return
	}

	truncateFile(paths.TempSermonSummary)

	if doc == nil || stringsTrim(audience) != AudienceAdult {
		return
	}

	if reasons := ValidateInfographic(doc.Infographic); len(reasons) > 0 {
		LogWarn("rework: infographic 생성 건너뜀 — " + strings.Join(reasons, ", "))
		return
	}

	// 시리즈와 제목은 복원된 문서의 metadata에서 가져온다.
	// master.Title은 "시리즈|||제목" 합친 라벨이므로 되나눠 폴백으로만 쓴다.
	series, title := splitHistoryTitle(master.Title)
	bibleText := master.BibleText

	if doc.Metadata != nil {
		if s := strings.TrimSpace(getStringFromMap(doc.Metadata, "series")); s != "" {
			series = s
		}
		if t := strings.TrimSpace(getStringFromMap(doc.Metadata, "title")); t != "" {
			title = t
		}
		if b := strings.TrimSpace(getStringFromMap(doc.Metadata, "bible_text")); b != "" {
			bibleText = b
		}
	}

	content := RenderInfographicMD(doc.Infographic, series, title, bibleText)
	if err := os.WriteFile(paths.TempSermonSummary, []byte(content), 0o644); err != nil {
		LogError("rework: sermon_summary.md 저장 실패: " + err.Error())
		return
	}

	LogInfo("rework: sermon_summary.md 복원 완료 path=" + paths.TempSermonSummary)
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

	supportScriptures := cleanStringSlice(flat.SupportScriptures)
	if supportScriptures == nil {
		supportScriptures = []string{}
	}

	return &QTSectionDoc{
		Version:  "1.0",
		DocType:  "qt",
		Audience: audience,
		Template: "qt_classic",
		Metadata: map[string]any{
			"title":              strings.TrimSpace(flat.Title),
			"bible_text":         strings.TrimSpace(flat.BibleText),
			"bible_passage_text": strings.TrimSpace(flat.BiblePassageText),
			"hymn":               strings.TrimSpace(flat.Hymn),
			"support_scriptures": supportScriptures,
			"preacher":           strings.TrimSpace(flat.Preacher),
			"church_name":        strings.TrimSpace(flat.ChurchName),
			"sermon_date":        strings.TrimSpace(flat.SermonDate),
			"source_url":         strings.TrimSpace(flat.SourceURL),
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

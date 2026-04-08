package service

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"s2qt/util"
)

type OfficeResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	DocxFile string `json:"docxFile"`
	PptxFile string `json:"pptxFile"`
}

type OfficeService struct {
	Paths *util.AppPaths
}

type QTDocument struct {
	Title            string
	BibleText        string
	Hymn             string
	Audience         string
	SermonDate       string
	Preacher         string
	ChurchName       string
	Summary          []string
	Messages         []QTMessage
	ReflectionItems  []string
	PrayerTitle      string
	PrayerParagraphs []string
	FooterText       string
}

type QTMessage struct {
	Title      string
	Paragraphs []string
}

type OfficeExportDoc struct {
	Title            string                `json:"title"`
	BibleText        string                `json:"bible_text"`
	Hymn             string                `json:"hymn"`
	Audience         string                `json:"audience,omitempty"`
	SermonDate       string                `json:"sermon_date,omitempty"`
	Preacher         string                `json:"preacher,omitempty"`
	ChurchName       string                `json:"church_name,omitempty"`
	Summary          []string              `json:"summary"`
	Messages         []OfficeExportMessage `json:"messages"`
	ReflectionItems  []string              `json:"reflection_items"`
	PrayerTitle      string                `json:"prayer_title"`
	PrayerParagraphs []string              `json:"prayer_paragraphs"`
	FooterText       string                `json:"footer_text,omitempty"`
}

type OfficeExportMessage struct {
	Title      string   `json:"title"`
	Paragraphs []string `json:"paragraphs"`
}

type OfficeExportEnvelope struct {
	Format     string          `json:"format"`
	OutputPath string          `json:"output_path"`
	Document   OfficeExportDoc `json:"document"`
}

func NewOfficeService() (*OfficeService, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return &OfficeService{
		Paths: paths,
	}, nil
}

func (s *OfficeService) LoadQTDocumentFromJSON() (*QTDocument, error) {
	b, err := os.ReadFile(s.Paths.TempJson)
	if err != nil {
		return nil, fmt.Errorf("temp.json 읽기 실패: %w", err)
	}

	var src QTSectionDoc
	if err := json.Unmarshal(b, &src); err != nil {
		return nil, fmt.Errorf("temp.json 파싱 실패: %w", err)
	}

	return s.BuildQTDocumentFromJSON(&src)
}

func (s *OfficeService) BuildQTDocumentFromJSON(src *QTSectionDoc) (*QTDocument, error) {
	if src == nil {
		return nil, fmt.Errorf("QTSectionDoc가 nil 입니다")
	}

	doc := &QTDocument{
		Title:            officeGetMetadataString(src.Metadata, "title"),
		BibleText:        officeGetMetadataString(src.Metadata, "bible_text"),
		Hymn:             officeGetMetadataString(src.Metadata, "hymn"),
		Audience:         officeFirstNonEmpty(strings.TrimSpace(src.Audience), officeGetMetadataString(src.Metadata, "audience")),
		SermonDate:       officeGetMetadataString(src.Metadata, "sermon_date"),
		Preacher:         officeGetMetadataString(src.Metadata, "preacher"),
		ChurchName:       officeGetMetadataString(src.Metadata, "church_name"),
		Summary:          nil,
		Messages:         nil,
		ReflectionItems:  nil,
		PrayerTitle:      "",
		PrayerParagraphs: nil,
		FooterText:       officeGetMetadataString(src.Metadata, "footer_text"),
	}

	summarySec := officeFindSection(src.Sections, "summary")
	if summarySec != nil {
		doc.Summary = officeExtractParagraphTexts(summarySec)
	}

	messageSec := officeFindSection(src.Sections, "message")
	if messageSec != nil {
		doc.Messages = officeExtractMessages(messageSec)
	}

	reflectionSec := officeFindSection(src.Sections, "reflection")
	if reflectionSec != nil {
		doc.ReflectionItems = officeExtractReflectionItems(reflectionSec)
	}

	prayerSec := officeFindSection(src.Sections, "prayer")
	if prayerSec != nil {
		doc.PrayerTitle = officeFirstNonEmpty(strings.TrimSpace(prayerSec.Title), "오늘의 기도")
		doc.PrayerParagraphs = officeExtractParagraphTexts(prayerSec)
	}

	doc.Title = officeFirstNonEmpty(doc.Title, "QT")

	return doc, nil
}

func (s *OfficeService) GetTempDocxPath() string {
	dir := filepath.Dir(s.Paths.TempHtml)
	return filepath.Join(dir, "temp.docx")
}

func (s *OfficeService) GetTempPptxPath() string {
	dir := filepath.Dir(s.Paths.TempHtml)
	return filepath.Join(dir, "temp.pptx")
}

func (s *OfficeService) GetTempOfficeExportJSONPath(format string) string {
	dir := filepath.Dir(s.Paths.TempHtml)
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "office"
	}
	return filepath.Join(dir, "temp_office_"+format+".json")
}

func (s *OfficeService) GetOfficeExporterPath() string {
	return filepath.Join("bin", "office_export.exe")
}

func (s *OfficeService) BuildOfficeExportDoc(doc *QTDocument) *OfficeExportDoc {
	if doc == nil {
		return nil
	}

	out := &OfficeExportDoc{
		Title:            strings.TrimSpace(doc.Title),
		BibleText:        strings.TrimSpace(doc.BibleText),
		Hymn:             strings.TrimSpace(doc.Hymn),
		Audience:         strings.TrimSpace(doc.Audience),
		SermonDate:       strings.TrimSpace(doc.SermonDate),
		Preacher:         strings.TrimSpace(doc.Preacher),
		ChurchName:       strings.TrimSpace(doc.ChurchName),
		Summary:          officeCompactLines(doc.Summary),
		Messages:         make([]OfficeExportMessage, 0, len(doc.Messages)),
		ReflectionItems:  officeCompactLines(doc.ReflectionItems),
		PrayerTitle:      strings.TrimSpace(doc.PrayerTitle),
		PrayerParagraphs: officeCompactLines(doc.PrayerParagraphs),
		FooterText:       strings.TrimSpace(doc.FooterText),
	}

	for _, msg := range doc.Messages {
		title := strings.TrimSpace(msg.Title)
		paragraphs := officeCompactLines(msg.Paragraphs)

		if title == "" && len(paragraphs) == 0 {
			continue
		}

		out.Messages = append(out.Messages, OfficeExportMessage{
			Title:      title,
			Paragraphs: paragraphs,
		})
	}

	return out
}

func (s *OfficeService) SaveOfficeExportJSON(format string, doc *QTDocument, outputPath string) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("QTDocument가 nil 입니다")
	}

	jsonPath := s.GetTempOfficeExportJSONPath(format)

	envelope := OfficeExportEnvelope{
		Format:     strings.ToLower(strings.TrimSpace(format)),
		OutputPath: strings.TrimSpace(outputPath),
		Document:   *s.BuildOfficeExportDoc(doc),
	}

	b, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return "", fmt.Errorf("office export json 직렬화 실패: %w", err)
	}

	if err := os.WriteFile(jsonPath, b, 0644); err != nil {
		return "", fmt.Errorf("office export json 저장 실패: %w", err)
	}

	return jsonPath, nil
}

func (s *OfficeService) RunOfficeExporter(format, inputPath, outputPath string) error {
	format = strings.ToLower(strings.TrimSpace(format))
	inputPath = strings.TrimSpace(inputPath)
	outputPath = strings.TrimSpace(outputPath)

	if format == "" {
		return fmt.Errorf("format이 비어 있습니다")
	}
	if inputPath == "" {
		return fmt.Errorf("inputPath가 비어 있습니다")
	}
	if outputPath == "" {
		return fmt.Errorf("outputPath가 비어 있습니다")
	}

	exporterPath := s.GetOfficeExporterPath()
	if _, err := os.Stat(exporterPath); err != nil {
		return fmt.Errorf("office_export.exe를 찾을 수 없습니다: %w", err)
	}

	if _, err := os.Stat(inputPath); err != nil {
		return fmt.Errorf("office export input json을 찾을 수 없습니다: %w", err)
	}

	_ = os.Remove(outputPath)

	cmd := exec.Command(
		exporterPath,
		"--format", format,
		"--input", inputPath,
		"--output", outputPath,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("office export 실행 실패: %v / output=%s", err, strings.TrimSpace(string(out)))
	}

	if err := officeWaitForFile(outputPath, 5*time.Second); err != nil {
		return err
	}

	return nil
}

func (s *OfficeService) ExportDOCXWithExe(doc *QTDocument, outPath string) error {
	jsonPath, err := s.SaveOfficeExportJSON("docx", doc, outPath)
	if err != nil {
		return err
	}
	return s.RunOfficeExporter("docx", jsonPath, outPath)
}

func (s *OfficeService) ExportPPTXWithExe(doc *QTDocument, outPath string) error {
	jsonPath, err := s.SaveOfficeExportJSON("pptx", doc, outPath)
	if err != nil {
		return err
	}
	return s.RunOfficeExporter("pptx", jsonPath, outPath)
}

func officeFindSection(sections []QTSectionData, sectionType string) *QTSectionData {
	sectionType = strings.TrimSpace(sectionType)
	for i := range sections {
		if strings.TrimSpace(sections[i].Type) == sectionType {
			return &sections[i]
		}
	}
	return nil
}

func officeExtractParagraphTexts(sec *QTSectionData) []string {
	if sec == nil {
		return nil
	}

	var out []string
	for _, blk := range sec.Blocks {
		if strings.TrimSpace(blk.Type) != "paragraph" {
			continue
		}
		text := strings.TrimSpace(blk.Text)
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func officeExtractMessages(sec *QTSectionData) []QTMessage {
	if sec == nil {
		return nil
	}

	var messages []QTMessage

	for i := 0; i < len(sec.Blocks)-1; i++ {
		titleBlk := sec.Blocks[i]
		bodyBlk := sec.Blocks[i+1]

		if strings.TrimSpace(titleBlk.Type) != "message_title" {
			continue
		}
		if strings.TrimSpace(bodyBlk.Type) != "paragraph" {
			continue
		}

		title := strings.TrimSpace(titleBlk.Text)
		body := strings.TrimSpace(bodyBlk.Text)

		if title == "" && body == "" {
			continue
		}

		messages = append(messages, QTMessage{
			Title:      title,
			Paragraphs: officeCompactLines([]string{body}),
		})

		i++
	}

	return messages
}

func officeExtractReflectionItems(sec *QTSectionData) []string {
	if sec == nil {
		return nil
	}

	for _, blk := range sec.Blocks {
		if strings.TrimSpace(blk.Type) != "list" {
			continue
		}
		return officeCompactLines(blk.Items)
	}

	return nil
}

func officeGetMetadataString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}

	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

func officeCompactLines(lines []string) []string {
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func officeFirstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func officeWaitForFile(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		info, err := os.Stat(path)
		if err == nil && info.Size() > 0 {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("산출물 파일이 생성되지 않았습니다: %s", path)
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, `'`, "&apos;")
	return s
}

func buildCoreXML(title string) string {
	now := time.Now().UTC().Format(time.RFC3339)

	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
 xmlns:dc="http://purl.org/dc/elements/1.1/"
 xmlns:dcterms="http://purl.org/dc/terms/"
 xmlns:dcmitype="http://purl.org/dc/dcmitype/"
 xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <dc:title>` + xmlEscape(title) + `</dc:title>
  <dc:creator>S2QT</dc:creator>
  <cp:lastModifiedBy>S2QT</cp:lastModifiedBy>
  <dcterms:created xsi:type="dcterms:W3CDTF">` + now + `</dcterms:created>
  <dcterms:modified xsi:type="dcterms:W3CDTF">` + now + `</dcterms:modified>
</cp:coreProperties>`
}

package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode"

	"s2qt/util"
)

type QTSectionDoc struct {
	Version  string          `json:"version"`
	DocType  string          `json:"doc_type"`
	Audience string          `json:"audience"`
	Template string          `json:"template_id"`
	Metadata map[string]any  `json:"metadata"`
	Sections []QTSectionData `json:"sections"`
}

// QTLLMDoc는 LLM이 생성한 원본 문서(v1.1)다.
// QTSectionDoc를 임베딩하므로, QTSectionDoc만 마샬하면 infographic이 자동으로 빠진다.
// Step1은 이 성질을 이용해 QT(temp.json)와 인포그래픽(sermon_summary.md)을 분리한다.
type QTLLMDoc struct {
	QTSectionDoc
	Infographic *InfographicData `json:"infographic,omitempty"`
}

type QTSectionData struct {
	Type   string        `json:"type"`
	Title  string        `json:"title"`
	Blocks []QTBlockData `json:"blocks"`
}

type QTBlockData struct {
	Type  string   `json:"type"`
	Text  string   `json:"text,omitempty"`
	Title string   `json:"title,omitempty"`
	Items []string `json:"items,omitempty"`
}

type QTStep2Service struct {
	Paths *util.AppPaths
}

func NewQTStep2Service() (*QTStep2Service, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}
	return &QTStep2Service{Paths: paths}, nil
}

func (s *QTStep2Service) Load() (*QTStep2Data, error) {
	b, err := os.ReadFile(s.Paths.TempJson)
	if err != nil {
		return nil, fmt.Errorf("temp.json 읽기 실패: %w", err)
	}

	var doc QTSectionDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("temp.json 파싱 실패: %w", err)
	}

	out := &QTStep2Data{
		Audience: strings.TrimSpace(doc.Audience),
	}

	// metadata 복원
	if doc.Metadata != nil {
		out.Series = strings.TrimSpace(getStringFromMap(doc.Metadata, "series"))
		out.Title = ensureQTTitlePrefix(step2firstNonEmpty(getStringFromMap(doc.Metadata, "title")))
		out.BibleText = normalizeBibleReference(getStringFromMap(doc.Metadata, "bible_text"))
		out.BiblePassageText = getStringFromMap(doc.Metadata, "bible_passage_text")
		out.Hymn = normalizeHymnText(getStringFromMap(doc.Metadata, "hymn"))
		out.Preacher = getStringFromMap(doc.Metadata, "preacher")
		out.ChurchName = getStringFromMap(doc.Metadata, "church_name")
		out.SermonDate = getStringFromMap(doc.Metadata, "sermon_date")
		out.SourceURL = getStringFromMap(doc.Metadata, "source_url")
		out.SupportScriptures = normalizeBibleRefSlice(getStringSliceFromMap(doc.Metadata, "support_scriptures"))
	}

	for _, sec := range doc.Sections {
		switch strings.TrimSpace(sec.Type) {
		case "summary":
			out.SummaryTitle = step2firstNonEmpty(sec.Title, "말씀의 길잡이")
			out.SummaryBody = firstParagraphText(sec.Blocks)

		case "message":
			msgIdx := 0
			for i := 0; i < len(sec.Blocks)-1; i++ {
				if sec.Blocks[i].Type == "message_title" && sec.Blocks[i+1].Type == "paragraph" {
					msgIdx++
					title := strings.TrimSpace(sec.Blocks[i].Text)
					body := strings.TrimSpace(sec.Blocks[i+1].Text)

					switch msgIdx {
					case 1:
						out.MessageTitle1 = title
						out.MessageBody1 = body
					case 2:
						out.MessageTitle2 = title
						out.MessageBody2 = body
					case 3:
						out.MessageTitle3 = title
						out.MessageBody3 = body
					}
					i++
				}
			}

		case "reflection":
			items := firstListItems(sec.Blocks)
			if len(items) > 0 {
				out.ReflectionItem1 = items[0]
			}
			if len(items) > 1 {
				out.ReflectionItem2 = items[1]
			}
			if len(items) > 2 {
				out.ReflectionItem3 = items[2]
			}

		case "prayer":
			out.PrayerTitle = step2firstNonEmpty(sec.Title, "오늘의 기도")
			out.PrayerBody = firstParagraphText(sec.Blocks)
		}
	}

	return out, nil
}

// preserveInternalMetaFields는 기존 temp.json의 내부 전용 필드를 새로 저장할 metadata 맵에
// 복사해 유실을 방지한다. Step2 UI가 편집하지 않는 값들이며, Save()가 doc을 통째로 새로
// 만들기 때문에 여기서 옮겨 주지 않으면 저장 한 번에 사라진다.
//
//	support_scriptures_full : extended.html이 쓰는 관련 성구 전체 본문
//	infographic             : sermon_summary.md를 다시 렌더하는 데 쓰는 원본
func preserveInternalMetaFields(tempJsonPath string, metadata map[string]any) {
	if metadata == nil {
		return
	}

	b, err := os.ReadFile(tempJsonPath)
	if err != nil {
		return
	}

	var prev QTSectionDoc
	if err := json.Unmarshal(b, &prev); err != nil {
		return
	}
	if prev.Metadata == nil {
		return
	}

	if v, ok := prev.Metadata["support_scriptures_full"]; ok {
		if arr, ok := v.([]any); ok && len(arr) > 0 {
			metadata["support_scriptures_full"] = arr
		}
	}

	if v, ok := prev.Metadata["infographic"]; ok && v != nil {
		metadata["infographic"] = v
	}
}

func ensureQTTitlePrefix(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "[QT]"
	}

	if strings.HasPrefix(title, "[QT]") {
		return title
	}

	return "[QT] " + title
}

func (s *QTStep2Service) Save(req *QTStep2Data) error {
	if req == nil {
		return fmt.Errorf("step2 data가 비어 있습니다")
	}

	finalTitle := ensureQTTitlePrefix(step2firstNonEmpty(req.Title, "QT"))
	finalBibleText := normalizeBibleReference(req.BibleText)
	finalSupportScriptures := normalizeBibleRefSlice(req.SupportScriptures)

	doc := QTSectionDoc{
		Version:  "1.0",
		DocType:  "qt",
		Audience: strings.TrimSpace(req.Audience),
		Template: "qt_classic",
		Metadata: map[string]any{
			// series는 Step2 UI에서 편집하지만, Save()가 doc을 통째로 새로 만들므로
			// 여기에 넣지 않으면 저장 한 번에 유실된다.
			"series":             strings.TrimSpace(req.Series),
			"title":              finalTitle,
			"bible_text":         finalBibleText,
			"bible_passage_text": strings.TrimSpace(req.BiblePassageText),
			"hymn":               normalizeHymnText(req.Hymn),
			"support_scriptures": finalSupportScriptures,
			"preacher":           strings.TrimSpace(req.Preacher),
			"church_name":        strings.TrimSpace(req.ChurchName),
			"sermon_date":        strings.TrimSpace(req.SermonDate),
			"source_url":         strings.TrimSpace(req.SourceURL),
		},
		Sections: []QTSectionData{
			{
				Type:  "summary",
				Title: step2firstNonEmpty(req.SummaryTitle, "📖 말씀의 길잡이"),
				Blocks: []QTBlockData{
					{Type: "paragraph", Text: strings.TrimSpace(req.SummaryBody)},
				},
			},
			{
				Type:  "message",
				Title: "✨ 오늘의 메시지",
				Blocks: []QTBlockData{
					{Type: "message_title", Text: strings.TrimSpace(req.MessageTitle1)},
					{Type: "paragraph", Text: strings.TrimSpace(req.MessageBody1)},
					{Type: "message_title", Text: strings.TrimSpace(req.MessageTitle2)},
					{Type: "paragraph", Text: strings.TrimSpace(req.MessageBody2)},
					{Type: "message_title", Text: strings.TrimSpace(req.MessageTitle3)},
					{Type: "paragraph", Text: strings.TrimSpace(req.MessageBody3)},
				},
			},
			{
				Type:  "reflection",
				Title: "🔍 깊은 묵상과 적용",
				Blocks: []QTBlockData{
					{
						Type: "list",
						Items: []string{
							strings.TrimSpace(req.ReflectionItem1),
							strings.TrimSpace(req.ReflectionItem2),
							strings.TrimSpace(req.ReflectionItem3),
						},
					},
				},
			},
			{
				Type:  "prayer",
				Title: step2firstNonEmpty(req.PrayerTitle, "🙏 오늘의 기도"),
				Blocks: []QTBlockData{
					{Type: "paragraph", Text: strings.TrimSpace(req.PrayerBody)},
				},
			},
		},
	}

	// 내부 전용 필드(support_scriptures_full, infographic)는 Step2 UI에서 편집하지
	// 않으므로, 기존 temp.json에 있던 값을 그대로 보존한다.
	// 빠뜨리면 첫 저장은 정상이고 두 번째 저장부터 조용히 망가진다.
	preserveInternalMetaFields(s.Paths.TempJson, doc.Metadata)

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("temp.json 직렬화 실패: %w", err)
	}

	if err := os.WriteFile(s.Paths.TempJson, b, 0644); err != nil {
		return fmt.Errorf("temp.json 저장 실패: %w", err)
	}

	// Step2 편집 내용을 sermon_summary.md에도 반영한다.
	// temp.json 저장이 성공한 뒤에만 수행한다.
	s.rewriteSermonSummary(doc)

	// 저장 버튼은 필수 절차이므로 temp.html도 함께 최신 상태로 갱신
	htmlReq := *req
	htmlReq.Title = finalTitle
	htmlReq.BibleText = finalBibleText
	htmlReq.BiblePassageText = strings.TrimSpace(req.BiblePassageText)
	htmlReq.Hymn = normalizeHymnText(req.Hymn)
	htmlReq.SupportScriptures = finalSupportScriptures

	if _, err := s.BuildHTML(&htmlReq); err != nil {
		return fmt.Errorf("temp.html 저장 실패: %w", err)
	}

	return nil
}

// rewriteSermonSummary는 Step2 저장 내용으로 sermon_summary.md를 다시 렌더한다.
//
// 조립이 끝난 doc을 통째로 받는다. 길잡이·적용·기도를 sections에서 가져오므로
// Step2에서 고친 QT 본문이 그대로 md에 반영된다.
//
// metadata.infographic이 없으면 파일을 건드리지 않는다. 비장년이거나 Step1의
// 인포그래픽 검증이 실패한 경우이며, 그때 md는 이미 0바이트다.
//
// 실패해도 저장 자체를 되돌리지 않는다 — temp.json과 temp.html은 이미 정상이고
// md는 부수 산출물이다.
func (s *QTStep2Service) rewriteSermonSummary(doc QTSectionDoc) {
	metadata := doc.Metadata

	data := getInfographicFromMap(metadata, "infographic")
	if data == nil {
		return
	}

	content := RenderInfographicMD(data, resolveSermonSummaryQT(data, doc.Sections), SermonSummaryMeta{
		Series:            getStringFromMap(metadata, "series"),
		Title:             getStringFromMap(metadata, "title"),
		BibleText:         getStringFromMap(metadata, "bible_text"),
		SupportScriptures: getStringSliceFromMap(metadata, "support_scriptures"),
	})

	if err := os.WriteFile(s.Paths.TempSermonSummary, []byte(content), 0o644); err != nil {
		LogError("step2: sermon_summary.md 갱신 실패: " + err.Error())
		return
	}

	LogInfo("step2: sermon_summary.md 갱신 완료 path=" + s.Paths.TempSermonSummary)
}

func (s *QTStep2Service) BuildHTML(req *QTStep2Data) (string, error) {
	if req == nil {
		return "", fmt.Errorf("step2 data가 비어 있습니다")
	}

	bodyHTML := buildQTStep2HTML(req)
	if strings.TrimSpace(bodyHTML) == "" {
		return "", fmt.Errorf("html 생성 결과가 비어 있습니다")
	}

	fullHTML := s.wrapQTStep2HTMLDocument(bodyHTML)

	if err := os.WriteFile(s.Paths.TempHtml, []byte(fullHTML), 0644); err != nil {
		return "", fmt.Errorf("temp.html 저장 실패: %w", err)
	}

	return s.Paths.TempHtml, nil
}

func buildQTStep2HTML(req *QTStep2Data) string {
	titleText := ensureQTTitlePrefix(step2firstNonEmpty(req.Title, "QT"))
	bibleText := normalizeBibleReference(req.BibleText)
	hymnText := normalizeHymnText(req.Hymn)
	supportScriptures := normalizeBibleRefSlice(req.SupportScriptures)
	biblePassageText := formatBiblePassageForOutput(req.BiblePassageText)

	biblePassageTitle := "성경본문"
	biblePassageClass := "qt-bible-passage"

	if isBiblePassageAbbreviated(req.BiblePassageText) {
		biblePassageTitle = "성경본문(처음절과 마지막절)"
		biblePassageClass += " is-abbreviated"
	}

	subboxParts := make([]string, 0)

	if bibleText != "" {
		subboxParts = append(subboxParts, "본문 성구: "+escapeHTML(bibleText))
	}

	if hymnText != "" {
		subboxParts = append(subboxParts, "찬송: "+escapeHTML(hymnText))
	}

	if len(supportScriptures) > 0 {
		subboxParts = append(
			subboxParts,
			"관련 성구: "+escapeHTML(strings.Join(supportScriptures, ", ")),
		)
	}

	subbox := ""
	if len(subboxParts) > 0 {
		subbox = `<div class="qt-subbox">` + strings.Join(subboxParts, "<br />") + `</div>`
	}

	passageHTML := ""
	if strings.TrimSpace(biblePassageText) != "" {
		passageHTML = `
  <div class="` + biblePassageClass + `">
    <div class="qt-bible-passage-title">` + escapeHTML(biblePassageTitle) + `</div>
    <p>` + nl2br(escapeHTML(biblePassageText)) + `</p>
  </div>`
	}

	prayerTitle := strings.TrimSpace(req.PrayerTitle)
	showPrayerInnerTitle := prayerTitle != "" && prayerTitle != "오늘의 기도" && prayerTitle != "🙏 오늘의 기도"

	prayerTitleHTML := ""
	if showPrayerInnerTitle {
		prayerTitleHTML = `<div class="qt-prayer-title">` + escapeHTML(prayerTitle) + `</div>`
	}

	// 시리즈가 없으면 블록 자체를 넣지 않는다.
	// 빈 div를 남기면 margin만큼 여백이 생겨 레이아웃이 달라진다.
	seriesHTML := ""
	if s := strings.TrimSpace(req.Series); s != "" {
		seriesHTML = `<div class="qt-series">` + escapeHTML(s) + `</div>
  `
	}

	return `
<div class="qt-wrap">
  ` + seriesHTML + `<div class="qt-title">` + escapeHTML(titleText) + `</div>
  ` + subbox + `
  ` + passageHTML + `

  <h2 class="qt-section-title">📖 말씀의 길잡이</h2>
  <div class="qt-body">
    <p>` + nl2br(escapeHTML(req.SummaryBody)) + `</p>
  </div>

  <h2 class="qt-section-title">✨ 오늘의 메시지</h2>

  <h3 class="qt-message-title">` + escapeHTML(req.MessageTitle1) + `</h3>
  <div class="qt-body"><p>` + nl2br(escapeHTML(req.MessageBody1)) + `</p></div>

  <h3 class="qt-message-title">` + escapeHTML(req.MessageTitle2) + `</h3>
  <div class="qt-body"><p>` + nl2br(escapeHTML(req.MessageBody2)) + `</p></div>

  <h3 class="qt-message-title">` + escapeHTML(req.MessageTitle3) + `</h3>
  <div class="qt-body"><p>` + nl2br(escapeHTML(req.MessageBody3)) + `</p></div>

  <h2 class="qt-section-title">🔍 깊은 묵상과 적용</h2>
  <div class="qt-box qt-reflection">
    <ul class="qt-list">
      <li>` + escapeHTML(req.ReflectionItem1) + `</li>
      <li>` + escapeHTML(req.ReflectionItem2) + `</li>
      <li>` + escapeHTML(req.ReflectionItem3) + `</li>
    </ul>
  </div>

  <h2 class="qt-section-title">🙏 오늘의 기도</h2>
  <div class="qt-box qt-prayer">
    ` + prayerTitleHTML + `
    <p>` + nl2br(escapeHTML(req.PrayerBody)) + `</p>
  </div>
</div>`
}

func step2firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func getStringFromMap(m map[string]any, key string) string {
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

func getStringSliceFromMap(m map[string]any, key string) []string {
	if m == nil {
		return []string{}
	}

	v, ok := m[key]
	if !ok || v == nil {
		return []string{}
	}

	switch x := v.(type) {
	case []string:
		return cleanStringSlice(x)

	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			out = append(out, strings.TrimSpace(fmt.Sprint(item)))
		}
		return cleanStringSlice(out)

	case string:
		parts := strings.FieldsFunc(x, func(r rune) bool {
			return r == ',' || r == '\n' || r == '\r'
		})
		return cleanStringSlice(parts)

	default:
		return []string{}
	}
}

// resolveSermonSummaryQT는 sermon_summary.md가 재활용할 본문을 정한다.
//
// 2026-10-01 이전 이력에는 infographic에 guide/apply/prayer가 들어 있다.
// 있으면 그것이 이긴다 — 재작업했을 때 당시 산출물이 그대로 재현된다.
// 없으면(= 그 이후 JSON) QT 섹션에서 가져오므로 Step2 편집이 md에 반영된다.
//
// 폴백은 여기 한 곳에만 둔다. 호출부 3곳에 흩어 놓으면 한 곳이 빠졌을 때
// 그 경로에서만 옛 이력이 다르게 나온다.
func resolveSermonSummaryQT(data *InfographicData, sections []QTSectionData) SermonSummaryQT {
	out := extractSermonSummaryQT(sections)

	if data == nil {
		return out
	}

	if g := strings.TrimSpace(data.Guide); g != "" {
		out.Guide = g
	}
	if a := cleanStringSlice(data.Apply); len(a) > 0 {
		out.Apply = a
	}
	if p := strings.TrimSpace(data.Prayer); p != "" {
		out.Prayer = p
	}

	return out
}

// extractSermonSummaryQT는 QT 섹션에서 md가 재활용할 본문을 꺼낸다.
// Step1·Step2·재작업이 모두 조립된 QTSectionDoc을 갖고 있으므로 경로는 하나다.
func extractSermonSummaryQT(sections []QTSectionData) SermonSummaryQT {
	var out SermonSummaryQT

	for _, sec := range sections {
		switch strings.TrimSpace(sec.Type) {
		case "summary":
			out.Guide = firstParagraphText(sec.Blocks)
		case "reflection":
			out.Apply = firstListItems(sec.Blocks)
		case "prayer":
			out.Prayer = firstParagraphText(sec.Blocks)
		}
	}

	return out
}

// firstParagraphText는 첫 paragraph 블록의 본문을 돌려준다.
func firstParagraphText(blocks []QTBlockData) string {
	for _, blk := range blocks {
		if strings.TrimSpace(blk.Type) == "paragraph" {
			return strings.TrimSpace(blk.Text)
		}
	}
	return ""
}

// firstListItems는 첫 list 블록의 항목을 돌려준다.
func firstListItems(blocks []QTBlockData) []string {
	for _, blk := range blocks {
		if strings.TrimSpace(blk.Type) == "list" {
			return cleanStringSlice(blk.Items)
		}
	}
	return nil
}

// copyMetadata는 metadata 맵을 얕게 복사한다.
// 맵은 참조 타입이라 구조체를 복사해도 같은 맵을 가리킨다. 한쪽에만 키를 더하고
// 싶을 때 이것을 거치지 않으면 다른 쪽도 조용히 함께 바뀐다.
func copyMetadata(m map[string]any) map[string]any {
	out := make(map[string]any, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	return out
}

// getInfographicFromMap은 metadata에 실린 인포그래픽을 꺼낸다.
// 값은 방금 넣은 *InfographicData일 수도, 파일에서 읽은 map[string]any일 수도 있어
// 재마샬로 한 가지 형태로 통일한다.
func getInfographicFromMap(m map[string]any, key string) *InfographicData {
	if m == nil || m[key] == nil {
		return nil
	}

	b, err := json.Marshal(m[key])
	if err != nil {
		return nil
	}

	var out InfographicData
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}

	return &out
}

func cleanStringSlice(items []string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{})

	for _, item := range items {
		s := strings.TrimSpace(item)
		if s == "" {
			continue
		}
		if _, exists := seen[s]; exists {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	return out
}

func escapeHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return r.Replace(s)
}

func nl2br(s string) string {
	return strings.ReplaceAll(s, "\n", "<br />")
}

func normalizeBibleReference(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	bookPart, restPart := splitBibleBookAndRest(s)
	if bookPart == "" {
		return s
	}

	normalizedBook := normalizeBibleBookName(bookPart)
	if normalizedBook == "" {
		normalizedBook = strings.TrimSpace(bookPart)
	}

	restPart = strings.TrimSpace(restPart)
	if restPart == "" {
		return normalizedBook
	}

	return normalizedBook + " " + restPart
}

func normalizeBibleRefSlice(items []string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{})

	for _, item := range items {
		s := normalizeBibleReference(item)
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, exists := seen[s]; exists {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	return out
}

func splitBibleBookAndRest(s string) (string, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}

	for i, r := range s {
		if unicode.IsDigit(r) {
			return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i:])
		}
	}

	return s, ""
}

func normalizeBibleBookName(book string) string {
	key := strings.ToLower(strings.TrimSpace(book))
	key = strings.ReplaceAll(key, " ", "")

	if v, ok := bibleBookNameMap[key]; ok {
		return v
	}
	return strings.TrimSpace(book)
}

var bibleBookNameMap = map[string]string{
	"창": "창세기", "창세": "창세기", "창세기": "창세기",
	"출": "출애굽기", "출애굽": "출애굽기", "출애굽기": "출애굽기",
	"레": "레위기", "레위": "레위기", "레위기": "레위기",
	"민": "민수기", "민수": "민수기", "민수기": "민수기",
	"신": "신명기", "신명": "신명기", "신명기": "신명기",

	"수": "여호수아", "수아": "여호수아", "여호수아": "여호수아",
	"삿": "사사기", "사사": "사사기", "사사기": "사사기",
	"룻": "룻기", "룻기": "룻기",

	"삼상": "사무엘상", "사무엘상": "사무엘상",
	"삼하": "사무엘하", "사무엘하": "사무엘하",
	"왕상": "열왕기상", "열왕기상": "열왕기상",
	"왕하": "열왕기하", "열왕기하": "열왕기하",
	"대상": "역대상", "역대상": "역대상",
	"대하": "역대하", "역대하": "역대하",
	"스": "에스라", "에스라": "에스라",
	"느": "느헤미야", "느헤미야": "느헤미야",
	"에": "에스더", "에스더": "에스더",

	"욥": "욥기", "욥기": "욥기",
	"시": "시편", "시편": "시편",
	"잠": "잠언", "잠언": "잠언",
	"전": "전도서", "전도서": "전도서",
	"아": "아가", "아가": "아가",

	"사": "이사야", "이사야": "이사야",
	"렘": "예레미야", "예레미야": "예레미야",
	"애": "예레미야애가", "예레미야애가": "예레미야애가",
	"겔": "에스겔", "에스겔": "에스겔",
	"단": "다니엘", "다니엘": "다니엘",

	"호": "호세아", "호세아": "호세아",
	"욜": "요엘", "요엘": "요엘",
	"암": "아모스", "아모스": "아모스",
	"옵": "오바댜", "오바댜": "오바댜",
	"욘": "요나", "요나": "요나",
	"미": "미가", "미가": "미가",
	"나": "나훔", "나훔": "나훔",
	"합": "하박국", "하박국": "하박국",
	"습": "스바냐", "스바냐": "스바냐",
	"학": "학개", "학개": "학개",
	"슥": "스가랴", "스가랴": "스가랴",
	"말": "말라기", "말라기": "말라기",

	"마": "마태복음", "마태": "마태복음", "마태복음": "마태복음",
	"막": "마가복음", "마가": "마가복음", "마가복음": "마가복음",
	"눅": "누가복음", "누가": "누가복음", "누가복음": "누가복음",
	"요": "요한복음", "요한": "요한복음", "요한복음": "요한복음",
	"행": "사도행전", "사도행전": "사도행전",

	"롬": "로마서", "로마서": "로마서",
	"고전": "고린도전서", "고린도전서": "고린도전서",
	"고후": "고린도후서", "고린도후서": "고린도후서",
	"갈": "갈라디아서", "갈라디아서": "갈라디아서",
	"엡": "에베소서", "에베소서": "에베소서",
	"빌": "빌립보서", "빌립보서": "빌립보서",
	"골": "골로새서", "골로새서": "골로새서",

	"살전": "데살로니가전서", "데살로니가전서": "데살로니가전서",
	"살후": "데살로니가후서", "데살로니가후서": "데살로니가후서",
	"딤전": "디모데전서", "디모데전서": "디모데전서",
	"딤후": "디모데후서", "디모데후서": "디모데후서",
	"딛": "디도서", "디도서": "디도서",
	"몬": "빌레몬서", "빌레몬서": "빌레몬서",

	"히": "히브리서", "히브리서": "히브리서",
	"약": "야고보서", "야고보서": "야고보서",
	"벧전": "베드로전서", "베드로전서": "베드로전서",
	"벧후": "베드로후서", "베드로후서": "베드로후서",
	"요일": "요한일서", "요한일서": "요한일서",
	"요이": "요한이서", "요한이서": "요한이서",
	"요삼": "요한삼서", "요한삼서": "요한삼서",
	"유": "유다서", "유다서": "유다서",
	"계": "요한계시록", "요한계시록": "요한계시록",
}

func normalizeHymnText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	prefixes := []string{
		"찬송가:",
		"찬송:",
		"찬송가 :",
		"찬송 :",
	}

	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			s = strings.TrimSpace(strings.TrimPrefix(s, p))
			break
		}
	}

	return strings.TrimSpace(s)
}

func splitBiblePassageLines(text string) []string {
	raw := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(raw))

	for _, line := range raw {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func formatBiblePassageForOutput(text string) string {
	lines := splitBiblePassageLines(text)
	if len(lines) == 0 {
		return ""
	}
	if len(lines) <= 5 {
		return strings.Join(lines, "\n")
	}
	return lines[0] + "\n...\n" + lines[len(lines)-1]
}

func isBiblePassageAbbreviated(text string) bool {
	lines := splitBiblePassageLines(text)
	return len(lines) > 5
}

func (s *QTStep2Service) wrapQTStep2HTMLDocument(body string) string {
	cssText := loadQTHTMLStyle()

	return `<!doctype html>
<html lang="ko">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>S2QT Preview</title>
</head>
<body>
<style>
` + cssText + `
</style>
` + body + `
</body>
</html>`
}

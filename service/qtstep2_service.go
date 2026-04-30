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
			out.SummaryTitle = step2firstNonEmpty(sec.Title, "말씀의 창")
			if len(sec.Blocks) > 0 {
				out.SummaryBody = strings.TrimSpace(sec.Blocks[0].Text)
			}

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
			for _, blk := range sec.Blocks {
				if blk.Type == "list" {
					if len(blk.Items) > 0 {
						out.ReflectionItem1 = strings.TrimSpace(blk.Items[0])
					}
					if len(blk.Items) > 1 {
						out.ReflectionItem2 = strings.TrimSpace(blk.Items[1])
					}
					if len(blk.Items) > 2 {
						out.ReflectionItem3 = strings.TrimSpace(blk.Items[2])
					}
					break
				}
			}

		case "prayer":
			out.PrayerTitle = step2firstNonEmpty(sec.Title, "오늘의 기도")
			if len(sec.Blocks) > 0 {
				out.PrayerBody = strings.TrimSpace(sec.Blocks[0].Text)
			}
		}
	}

	return out, nil
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
				Title: step2firstNonEmpty(req.SummaryTitle, "🌿 말씀의 창"),
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

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("temp.json 직렬화 실패: %w", err)
	}

	if err := os.WriteFile(s.Paths.TempJson, b, 0644); err != nil {
		return fmt.Errorf("temp.json 저장 실패: %w", err)
	}

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
		biblePassageTitle = "성경본문(축약)"
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

	return `
<div class="qt-wrap">
  <div class="qt-title">` + escapeHTML(titleText) + `</div>
  ` + subbox + `
  ` + passageHTML + `

  <h2 class="qt-section-title">🌿 말씀의 창</h2>
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

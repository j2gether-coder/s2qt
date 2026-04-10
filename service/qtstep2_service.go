package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
		out.Title = step2firstNonEmpty(getStringFromMap(doc.Metadata, "title"))
		out.BibleText = getStringFromMap(doc.Metadata, "bible_text")
		out.Hymn = getStringFromMap(doc.Metadata, "hymn")
		out.Preacher = getStringFromMap(doc.Metadata, "preacher")
		out.ChurchName = getStringFromMap(doc.Metadata, "church_name")
		out.SermonDate = getStringFromMap(doc.Metadata, "sermon_date")
		out.SourceURL = getStringFromMap(doc.Metadata, "source_url")
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

func (s *QTStep2Service) Save(req *QTStep2Data) error {
	if req == nil {
		return fmt.Errorf("step2 data가 비어 있습니다")
	}

	finalTitle := step2firstNonEmpty(req.Title, "QT")
	finalBibleText := strings.TrimSpace(req.BibleText)

	doc := QTSectionDoc{
		Version:  "1.0",
		DocType:  "qt",
		Audience: strings.TrimSpace(req.Audience),
		Template: "qt_classic",
		Metadata: map[string]any{
			"title":       finalTitle,
			"bible_text":  finalBibleText,
			"hymn":        strings.TrimSpace(req.Hymn),
			"preacher":    strings.TrimSpace(req.Preacher),
			"church_name": strings.TrimSpace(req.ChurchName),
			"sermon_date": strings.TrimSpace(req.SermonDate),
			"source_url":  strings.TrimSpace(req.SourceURL),
		},
		Sections: []QTSectionData{
			{
				Type:  "summary",
				Title: step2firstNonEmpty(req.SummaryTitle, "🌿 말씀의 창: 본문 요약"),
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

	return os.WriteFile(s.Paths.TempJson, b, 0644)
}

func (s *QTStep2Service) BuildHTML(req *QTStep2Data) (string, error) {
	if req == nil {
		return "", fmt.Errorf("step2 data가 비어 있습니다")
	}

	html := buildQTStep2HTML(req)
	if strings.TrimSpace(html) == "" {
		return "", fmt.Errorf("html 생성 결과가 비어 있습니다")
	}

	if err := os.WriteFile(s.Paths.TempHtml, []byte(html), 0644); err != nil {
		return "", fmt.Errorf("temp.html 저장 실패: %w", err)
	}

	return s.Paths.TempHtml, nil
}

func buildQTStep2HTML(req *QTStep2Data) string {
	titleText := step2firstNonEmpty(req.Title, "QT")
	bibleText := strings.TrimSpace(req.BibleText)

	subboxParts := make([]string, 0)
	if bibleText != "" {
		subboxParts = append(subboxParts, "본문 성구: "+escapeHTML(bibleText))
	}
	if strings.TrimSpace(req.Hymn) != "" {
		subboxParts = append(subboxParts, "찬송: "+escapeHTML(strings.TrimSpace(req.Hymn)))
	}

	subbox := ""
	if len(subboxParts) > 0 {
		subbox = `<div class="qt-subbox">` + strings.Join(subboxParts, " &nbsp;|&nbsp; ") + `</div>`
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

  <h2 class="qt-section-title">🌿 말씀의 창: 본문 요약</h2>
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

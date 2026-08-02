package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"s2qt/util"
)

// blog.html은 기존 HTML/PDF/PNG와 완전히 분리된 산출물이다.
// temp.json의 내부 전용 필드(bible_passage_full, support_scriptures_full)를 사용해
// 성구 전체 본문을 그대로 담는다. 기존 산출물 경로는 이 필드를 읽지 않는다.

type BlogSupportScripture struct {
	Reference string `json:"reference"`
	Text      string `json:"text"`
}

type blogMetadata struct {
	Title                 string                 `json:"title"`
	BibleText             string                 `json:"bible_text"`
	BiblePassageText      string                 `json:"bible_passage_text"`
	Hymn                  string                 `json:"hymn"`
	SupportScriptures     []string               `json:"support_scriptures"`
	SupportScripturesFull []BlogSupportScripture `json:"support_scriptures_full"`
	Preacher              string                 `json:"preacher"`
	ChurchName            string                 `json:"church_name"`
	SermonDate            string                 `json:"sermon_date"`
	SourceURL             string                 `json:"source_url"`
}

type blogDoc struct {
	Metadata blogMetadata    `json:"metadata"`
	Sections []QTSectionData `json:"sections"`
}

type BlogService struct {
	Paths *util.AppPaths
}

func NewBlogService() (*BlogService, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}
	return &BlogService{Paths: paths}, nil
}

// BuildBlogHTML는 temp.json을 읽어 blog.html을 생성하고 경로를 반환한다.
func (s *BlogService) BuildBlogHTML() (string, error) {
	b, err := os.ReadFile(s.Paths.TempJson)
	if err != nil {
		return "", fmt.Errorf("temp.json 읽기 실패: %w", err)
	}

	var doc blogDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return "", fmt.Errorf("temp.json 파싱 실패: %w", err)
	}

	body := buildBlogBody(&doc)
	full := wrapBlogHTMLDocument(body)

	if err := os.WriteFile(s.Paths.TempBlog, []byte(full), 0644); err != nil {
		return "", fmt.Errorf("blog.html 저장 실패: %w", err)
	}

	return s.Paths.TempBlog, nil
}

// blogPassageText는 blog 전용 전체 본문을 반환한다.
// bible_passage_text는 전체 본문을 보관하며(기존 산출물만 렌더링 시 축약),
// 블로그는 축약 없이 전체를 그대로 사용한다.
func blogPassageText(m blogMetadata) string {
	return strings.TrimSpace(m.BiblePassageText)
}

// blogSupportScriptures는 전체 본문이 있으면 그것을, 없으면 참조만 반환한다.
func blogSupportScriptures(m blogMetadata) []BlogSupportScripture {
	if len(m.SupportScripturesFull) > 0 {
		out := make([]BlogSupportScripture, 0, len(m.SupportScripturesFull))
		for _, item := range m.SupportScripturesFull {
			ref := normalizeBibleReference(item.Reference)
			text := strings.TrimSpace(item.Text)
			if ref == "" && text == "" {
				continue
			}
			out = append(out, BlogSupportScripture{Reference: ref, Text: text})
		}
		if len(out) > 0 {
			return out
		}
	}

	// 폴백: 참조만 표시
	refs := normalizeBibleRefSlice(m.SupportScriptures)
	out := make([]BlogSupportScripture, 0, len(refs))
	for _, ref := range refs {
		out = append(out, BlogSupportScripture{Reference: ref})
	}
	return out
}

func buildBlogBody(doc *blogDoc) string {
	m := doc.Metadata

	titleText := ensureQTTitlePrefix(step2firstNonEmpty(m.Title, "QT"))
	bibleText := normalizeBibleReference(m.BibleText)
	hymnText := normalizeHymnText(m.Hymn)
	passageText := blogPassageText(m)
	supports := blogSupportScriptures(m)

	var sb strings.Builder

	// temp.html과 동일한 qt-* 클래스 구조를 사용해 loadQTHTMLStyle() CSS를 그대로 적용받는다.
	sb.WriteString(`<div class="qt-wrap">`)
	sb.WriteString(`<div class="qt-title">` + escapeHTML(titleText) + `</div>`)

	// 상단 메타 정보 (본문 성구 / 찬송)
	subboxParts := make([]string, 0)
	if bibleText != "" {
		subboxParts = append(subboxParts, "본문 성구: "+escapeHTML(bibleText))
	}
	if hymnText != "" {
		subboxParts = append(subboxParts, "찬송: "+escapeHTML(hymnText))
	}
	if len(subboxParts) > 0 {
		sb.WriteString(`<div class="qt-subbox">` + strings.Join(subboxParts, "<br />") + `</div>`)
	}

	// 섹션(요약/메시지/묵상/기도)
	sb.WriteString(buildBlogSections(doc.Sections))

	// 성경본문/관련 성구는 오늘의 기도 아래에 배치한다.
	// 본문 성구 전체 (관련 성구와 동일한 구조: 섹션 타이틀 → 성경 장:절 → 본문)
	if passageText != "" {
		sb.WriteString(`<h2 class="qt-section-title">성경본문</h2>`)
		sb.WriteString(`<div class="qt-bible-passage">`)
		if bibleText != "" {
			sb.WriteString(`<div class="qt-bible-passage-title">` + escapeHTML(bibleText) + `</div>`)
		}
		sb.WriteString(`<p>` + nl2br(escapeHTML(passageText)) + `</p>`)
		sb.WriteString(`</div>`)
	}

	// 관련 성구 전체 본문 (본문 성구와 동일한 passage 박스를 재사용)
	if len(supports) > 0 {
		sb.WriteString(`<h2 class="qt-section-title">관련 성구</h2>`)
		for _, item := range supports {
			sb.WriteString(`<div class="qt-bible-passage">`)
			if item.Reference != "" {
				sb.WriteString(`<div class="qt-bible-passage-title">` + escapeHTML(item.Reference) + `</div>`)
			}
			if item.Text != "" {
				sb.WriteString(`<p>` + nl2br(escapeHTML(item.Text)) + `</p>`)
			}
			sb.WriteString(`</div>`)
		}
	}

	sb.WriteString(`</div>`)
	return sb.String()
}

func buildBlogSections(sections []QTSectionData) string {
	var sb strings.Builder

	for _, sec := range sections {
		switch strings.TrimSpace(sec.Type) {
		case "summary":
			body := ""
			if len(sec.Blocks) > 0 {
				body = strings.TrimSpace(sec.Blocks[0].Text)
			}
			if body == "" {
				continue
			}
			sb.WriteString(`<h2 class="qt-section-title">📖 말씀의 길잡이</h2>`)
			sb.WriteString(`<div class="qt-body"><p>` + nl2br(escapeHTML(body)) + `</p></div>`)

		case "message":
			sb.WriteString(`<h2 class="qt-section-title">✨ 오늘의 메시지</h2>`)
			for i := 0; i < len(sec.Blocks); i++ {
				blk := sec.Blocks[i]
				text := strings.TrimSpace(blk.Text)
				if text == "" {
					continue
				}
				if blk.Type == "message_title" {
					sb.WriteString(`<h3 class="qt-message-title">` + escapeHTML(text) + `</h3>`)
				} else {
					sb.WriteString(`<div class="qt-body"><p>` + nl2br(escapeHTML(text)) + `</p></div>`)
				}
			}

		case "reflection":
			items := make([]string, 0)
			for _, blk := range sec.Blocks {
				if blk.Type == "list" {
					for _, it := range blk.Items {
						if v := strings.TrimSpace(it); v != "" {
							items = append(items, v)
						}
					}
					break
				}
			}
			if len(items) == 0 {
				continue
			}
			sb.WriteString(`<h2 class="qt-section-title">🔍 깊은 묵상과 적용</h2>`)
			sb.WriteString(`<div class="qt-box qt-reflection"><ul class="qt-list">`)
			for _, it := range items {
				sb.WriteString(`<li>` + escapeHTML(it) + `</li>`)
			}
			sb.WriteString(`</ul></div>`)

		case "prayer":
			body := ""
			if len(sec.Blocks) > 0 {
				body = strings.TrimSpace(sec.Blocks[0].Text)
			}
			if body == "" {
				continue
			}
			sb.WriteString(`<h2 class="qt-section-title">🙏 오늘의 기도</h2>`)
			sb.WriteString(`<div class="qt-box qt-prayer"><p>` + nl2br(escapeHTML(body)) + `</p></div>`)
		}
	}

	return sb.String()
}

// wrapBlogHTMLDocument는 temp.html과 동일한 QT HTML 스타일(loadQTHTMLStyle)을 임베드한다.
func wrapBlogHTMLDocument(body string) string {
	cssText := loadQTHTMLStyle()

	return `<!doctype html>
<html lang="ko">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>S2QT Blog</title>
</head>
<body>
<style>
` + cssText + `
</style>
` + body + `
</body>
</html>`
}

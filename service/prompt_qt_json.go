package service

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultQTPromptBase = `[Role]
QT Writing Assistant

[Task]
ASR 설교 전사문을 교회 주보/묵상/블로그용 QT JSON 원고로 작성한다.

[Language]
Output must be written in natural Korean.

[Metadata]
Title={{title}}
Bible Passage={{bible_text}}
Hymn={{hymn}}
Preacher={{preacher}}
Church={{church_name}}
Date={{sermon_date}}
URL={{source_url}}
Audience={{audience}}

[Critical Scope]
- Use only {{bible_text}} as the QT basis.
- Exclude sermon content outside {{bible_text}} from summary, message, reflection, application, and prayer.
- Use verse references only within {{bible_text}}. If uncertain, omit verse references.

[Output Rules]
- JSON object only.
- No code blocks, HTML, markdown fence, or explanations.
- Keep JSON syntax valid.

[Content Rules]
- ASR errors may exist. Fix meaning from context and repeated key messages.
- Do not quote the transcript literally.
- Do not overstate uncertain details; generalize when needed.
- Clarify the main theme, exhortation, and application.
- Summarize illustrations only when they directly support the main message.
- Rewrite awkward sentences into natural Korean.
- Reflect repeated emphases.

[Field Rules]
- metadata.title must start with "[QT]".
- metadata.bible_text must equal {{bible_text}}.
- metadata.bible_passage_text: passage text only; include verse numbers and one verse per line if possible.
- metadata.hymn: use {{hymn}} if given; otherwise recommend one hymn; use "-" if unsure.
- metadata.support_scriptures: string array, 0-3 related references, excluding {{bible_text}} itself; [] if none.`

const defaultQTPromptJSONSchema = `[JSON Contract]
- Output exactly one JSON object matching the schema below.
- version="1.0", doc_type="qt", template_id="qt_classic".
- sections must be exactly 4 and in this order: summary, message, reflection, prayer.
- message.blocks must be exactly 6: message_title/paragraph repeated 3 times.
- reflection.blocks[0].items must contain exactly 3 items.
- prayer.blocks must contain exactly 1 paragraph block.
- Do not use verses outside {{bible_text}}.
- Fill empty strings with natural content whenever possible.
- metadata.support_scriptures must be an array, never a single string.

[JSON Schema]
{
  "version": "1.0",
  "doc_type": "qt",
  "audience": "{{audience}}",
  "template_id": "qt_classic",
  "metadata": {
    "title": "[QT] title",
    "bible_text": "{{bible_text}}",
    "bible_passage_text": "",
    "hymn": "",
    "support_scriptures": [],
    "preacher": "{{preacher}}",
    "church_name": "{{church_name}}",
    "sermon_date": "{{sermon_date}}",
    "source_url": "{{source_url}}",
    "month_name": "{{month_name}}",
    "month_accent": "{{month_accent}}"
  },
  "sections": [
    {
      "type": "summary",
      "title": "🌿 말씀의 창",
      "blocks": [
        { "type": "paragraph", "text": "" }
      ]
    },
    {
      "type": "message",
      "title": "✨ 오늘의 메시지",
      "blocks": [
        { "type": "message_title", "text": "" },
        { "type": "paragraph", "text": "" },
        { "type": "message_title", "text": "" },
        { "type": "paragraph", "text": "" },
        { "type": "message_title", "text": "" },
        { "type": "paragraph", "text": "" }
      ]
    },
    {
      "type": "reflection",
      "title": "🔍 깊은 묵상과 적용",
      "blocks": [
        { "type": "list", "items": ["", "", ""] }
      ]
    },
    {
      "type": "prayer",
      "title": "🙏 오늘의 기도",
      "blocks": [
        { "type": "paragraph", "text": "" }
      ]
    }
  ]
}`

const defaultQTPromptTranscript = `[Transcript]
{{raw_text}}`

func loadAppConfig() (*AppConfig, error) {
	configPath := "var/conf/app.yaml"

	b, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("app.yaml 읽기 실패: %w", err)
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("app.yaml 파싱 실패: %w", err)
	}

	return &cfg, nil
}

func loadQTPromptBaseTemplate() string {
	cfg, err := loadAppConfig()
	if err != nil {
		return defaultQTPromptBase
	}

	promptPath := strings.TrimSpace(cfg.PromptQTJSONFile)
	if promptPath == "" {
		return defaultQTPromptBase
	}

	b, err := os.ReadFile(promptPath)
	if err != nil {
		return defaultQTPromptBase
	}

	text := strings.TrimSpace(string(b))
	if text == "" {
		return defaultQTPromptBase
	}

	return text
}

func GetMonthAccentColor(t time.Time) string {
	switch t.Month() {
	case time.January:
		return "#3E5A78"
	case time.February:
		return "#6B5B95"
	case time.March:
		return "#4C8C4A"
	case time.April:
		return "#C97B84"
	case time.May:
		return "#3F7D4E"
	case time.June:
		return "#3E8E8E"
	case time.July:
		return "#4A78C2"
	case time.August:
		return "#C97A3D"
	case time.September:
		return "#7A8B4C"
	case time.October:
		return "#8A5A44"
	case time.November:
		return "#7A4A5A"
	case time.December:
		return "#4E6A4A"
	default:
		return "#27ae60"
	}
}

func GetMonthNameKorean(t time.Time) string {
	switch t.Month() {
	case time.January:
		return "1월"
	case time.February:
		return "2월"
	case time.March:
		return "3월"
	case time.April:
		return "4월"
	case time.May:
		return "5월"
	case time.June:
		return "6월"
	case time.July:
		return "7월"
	case time.August:
		return "8월"
	case time.September:
		return "9월"
	case time.October:
		return "10월"
	case time.November:
		return "11월"
	case time.December:
		return "12월"
	default:
		return "이번 달"
	}
}

func GetQTPromptTemplate() string {
	return loadQTPromptBaseTemplate()
}

func getAudiencePromptRules(audience string) string {
	switch strings.TrimSpace(audience) {
	case "adult":
		return `[Audience: adult]
- audience="adult".
- Keep the input title.
- summary: 5-6 sentences.
- message paragraphs: 3-5 sentences each.
- reflection: 3 concrete items leading to self-examination and obedience.
- prayer: 5-6 sentences with repentance, gratitude, and obedience.
- Style: calm, reverent, warm; avoid casual language.`

	case "young_adult":
		return `[Audience: young_adult]
- audience="young_adult".
- You may refine the title if it better expresses the sermon; otherwise keep it.
- summary: 4-5 sentences.
- message paragraphs: 2-4 sentences each.
- Connect naturally with calling, choices, relationships, identity, and life direction.
- reflection: 3 practical choices for today.
- prayer: 4-5 honest, sincere sentences.
- Style: warm, clear, realistic; not overly heavy.`

	case "teen":
		return `[Audience: teen]
- audience="teen".
- You may simplify the title; otherwise keep it.
- summary: 3-4 short sentences.
- message paragraphs: 2-3 sentences each.
- Use simple words; reduce abstract or difficult theological terms.
- Connect with school, friends, family, emotions, and habits.
- reflection: 3 specific actions students can practice.
- prayer: about 4 easy, natural sentences.
- Style: empathetic and clear, not merely didactic.`

	case "child":
		return `[Audience: child]
- audience="child".
- You may rewrite the title in very easy words; otherwise keep it.
- summary: 2-3 very simple sentences.
- message paragraphs: 1-2 short sentences each.
- Avoid difficult theology and abstract terms.
- reflection: 3 small, easy actions.
- prayer: 3-4 short and gentle sentences.
- Style: warm, simple, not scary or overly heavy.`

	default:
		return `[Audience: default]
- audience="{{audience}}".
- Prefer the input title; refine only if needed.
- Match tone and length to the audience as naturally as possible.`
	}
}

func BuildQTPromptJSON(meta QTMeta) string {
	now := time.Now()

	basePrompt := strings.TrimSpace(loadQTPromptBaseTemplate())
	audienceRules := strings.TrimSpace(getAudiencePromptRules(meta.Audience))
	schemaPrompt := strings.TrimSpace(defaultQTPromptJSONSchema)
	transcriptPrompt := strings.TrimSpace(defaultQTPromptTranscript)

	prompt := strings.Join([]string{
		basePrompt,
		audienceRules,
		schemaPrompt,
		transcriptPrompt,
	}, "\n\n")

	replacer := strings.NewReplacer(
		"{{title}}", meta.Title,
		"{{bible_text}}", meta.BibleText,
		"{{hymn}}", meta.Hymn,
		"{{preacher}}", meta.Preacher,
		"{{church_name}}", meta.ChurchName,
		"{{sermon_date}}", meta.SermonDate,
		"{{source_url}}", meta.SourceURL,
		"{{raw_text}}", meta.RawText,
		"{{audience}}", meta.Audience,
		"{{month_accent}}", GetMonthAccentColor(now),
		"{{month_name}}", GetMonthNameKorean(now),
	)

	return replacer.Replace(prompt)
}

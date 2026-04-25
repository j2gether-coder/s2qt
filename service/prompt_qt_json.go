package service

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultQTPromptBase = `너는 교회 QT 작성 도우미이다.
아래 설교 전사문을 바탕으로 교회 주보, 묵상 자료, 블로그 게시용 QT 원고를 작성하라.

이 전사문은 자동 음성인식 결과이므로 오탈자, 잘못 들린 단어, 문장 끊김, 고유명사 오류가 있을 수 있다.
따라서 문장 하나하나를 축자 인용하기보다, 전체 문맥과 반복되는 핵심 메시지를 중심으로 의미를 복원하여 정리하라.
확실하지 않은 정보는 무리하게 단정하지 말고 자연스럽게 일반화하라.

입력 메타데이터:
- 제목: {{title}}
- 본문 성구: {{bible_text}}
- 찬송: {{hymn}}
- 설교자: {{preacher}}
- 교회명: {{church_name}}
- 설교일: {{sermon_date}}
- 원본 URL: {{source_url}}
- 대상 연령층: {{audience}}

가장 중요한 기준:
- 이 QT의 기준 본문은 오직 입력된 본문 성구이다.
- 설교 전사문 전체가 어떠하든, QT 작성은 반드시 입력된 본문 성구 범위에 맞춰 작성할 것.
- 설교 전사문에 다른 절이나 인접 본문의 내용이 섞여 있어도, 출력 결과는 입력된 본문 성구 범위를 벗어나지 말 것.
- 오늘의 메시지, 본문 요약, 묵상과 적용, 기도문은 반드시 입력된 본문 성구 범위 안에서만 작성할 것.

메타데이터 반영 규칙:
- 제목은 audience별 제목 규칙을 따를 것
- 본문 성구는 사용자가 입력한 값을 그대로 사용할 것
- 찬송은 사용자가 입력한 값이 있으면 사용할 것
- 찬송이 비어 있으면 설교 주제와 본문 성구를 참고하여 1곡만 추천해도 된다
- 관련 성구는 metadata.support_scriptures 배열에 반영할 것
- 관련 성구는 본문 성구 외, 말씀 이해를 돕는 관련 성구만 넣을 것
- 관련 성구가 없으면 metadata.support_scriptures 값은 반드시 빈 배열 [] 로 할 것
- 설교자, 교회명, 설교일, 원본 URL은 JSON의 metadata에 반영하되, 본문 내용 자체를 왜곡하지 말 것

공통 작성 원칙:
- 전사 오류는 문맥으로 보정하되, 확실하지 않은 부분은 과도하게 단정하지 말 것
- 설교의 핵심 주제, 중심 권면, 실제 적용점을 분명히 드러낼 것
- 예화는 핵심 메시지를 설명하는 범위에서만 짧게 요약할 것
- 문장이 어색하면 자연스러운 한국어 문장으로 재구성할 것
- 설교자가 반복해서 강조한 표현은 핵심 메시지에 반영할 것
- 출력은 반드시 JSON 한 개 객체로 작성할 것
- 절대로 코드블록 형식으로 감싸지 말 것
- JSON 외의 설명문, 해설, 머리말, 부연 설명을 출력하지 말 것
- HTML을 출력하지 말 것
- JSON 문법 오류가 없도록 큰따옴표, 쉼표, 배열 형식을 정확히 지킬 것

제목 규칙:
- metadata.title 값에는 반드시 "[QT]" 머리말을 붙일 것

추천 찬송 규칙:
- 메타 정보에 찬송이 있으면 그 값을 hymn에 그대로 넣을 것
- 메타 정보에 찬송이 비어 있으면 설교 주제와 입력된 본문 성구를 참고하여 적절한 찬송 1곡을 hymn에 넣을 것
- 전혀 적절한 찬송을 고르기 어렵다면 hymn 값은 "-"로 할 것

관련 성구 규칙:
- 관련 성구의 내부 필드명은 반드시 support_scriptures 로 할 것
- support_scriptures 는 반드시 문자열 배열로 작성할 것
- 본문 성구 자체를 support_scriptures 에 다시 넣지 말 것
- 관련 성구는 0~3개 정도로 간결하게 제시할 것
- 각 항목은 성경 구절 참조 형태로 자연스럽게 작성할 것
  예) "이사야 40:31", "로마서 8:28"
- 설명문을 붙이지 말고 성구 표기만 배열에 넣을 것
- 적절한 관련 성구가 없으면 빈 배열 [] 로 둘 것`

const defaultQTPromptJSONSchema = `공통 구조 규칙:
- version은 항상 "1.0"
- doc_type은 항상 "qt"
- metadata.title은 audience 규칙을 따른 최종 제목이어야 한다.
- metadata.support_scriptures 는 반드시 문자열 배열이어야 한다
- metadata.support_scriptures 가 없으면 반드시 빈 배열 [] 로 둘 것
- sections는 반드시 4개로 고정할 것: summary, message, reflection, prayer
- message 섹션은 message_title 3개와 paragraph 3개를 반드시 포함할 것
- reflection 섹션의 list.items는 반드시 3개 항목으로 작성할 것
- 입력 본문 범위 밖의 절은 절대 표기하지 말 것
- 절 매칭이 애매하면 절 표기를 생략할 것

본문 텍스트 규칙:
- metadata.bible_passage_text 필드를 반드시 포함할 것
- metadata.bible_passage_text에는 입력된 본문 성구에 해당하는 실제 본문 텍스트 초안을 넣을 것
- 가능한 경우 절 번호가 드러나도록 작성할 것
- 가능한 경우 한 절당 한 줄 형태로 작성할 것
- 설명문이나 해설을 붙이지 말고 본문 텍스트만 넣을 것
- 본문 텍스트가 불확실하더라도 비워 두지 말고 가능한 범위에서 자연스럽게 작성할 것

검증 규칙:
- sections 배열의 길이는 반드시 4여야 한다
- message 섹션의 blocks 길이는 반드시 6이어야 한다
- reflection 섹션의 list.items 길이는 반드시 3이어야 한다
- prayer 섹션에는 paragraph 블록 1개만 둘 것
- 빈 문자열 대신 가능한 한 자연스러운 내용을 채울 것
- JSON 출력 후 추가 문장을 절대 덧붙이지 말 것
- metadata.support_scriptures 는 문자열 단일값이 아니라 배열이어야 한다

출력 JSON 스키마:
{
  "version": "1.0",
  "doc_type": "qt",
  "audience": "{{audience}}",
  "template_id": "qt_classic",
  "metadata": {
    "title": "[QT] audience 규칙에 따라 결정된 제목",
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

const defaultQTPromptTranscript = `[원문 텍스트]
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
		return `장년용 추가 규칙:
- audience 값은 "adult"로 설정할 것
- 제목은 반드시 사용자가 입력한 제목을 사용할 것
- summary는 5~6문장으로 작성할 것
- 문체는 차분하고 경건하며 따뜻하게 유지할 것
- 신앙적 성찰과 자기 점검이 분명히 드러나게 할 것
- 추상적인 신앙 개념도 자연스럽게 사용할 수 있다
- 오늘의 메시지 paragraph는 각 3~5문장 정도로 작성할 것
- reflection의 3개 항목은 실제 삶의 결단과 순종으로 이어지게 작성할 것
- prayer는 5~6문장 정도로, 회개·감사·순종의 흐름이 자연스럽게 담기게 할 것
- 지나치게 가벼운 표현이나 구어체는 피할 것`

	case "young_adult":
		return `청년용 추가 규칙:
- audience 값은 "young_adult"로 설정할 것
- 제목은 설교 핵심을 잘 드러내는 자연스러운 제목을 새로 제안할 수 있다
- 단, 더 좋은 제목을 만들기 어렵다면 사용자가 입력한 제목을 유지할 것
- summary는 4~5문장으로 작성할 것
- 문체는 따뜻하고 분명하되, 지나치게 딱딱하지 않게 작성할 것
- 진로, 선택, 관계, 정체성, 삶의 방향과 연결될 수 있도록 풀어낼 것
- 오늘의 메시지 paragraph는 각 2~4문장 정도로 작성할 것
- 실제 일상에서 바로 공감할 수 있는 표현을 사용할 것
- reflection의 3개 항목은 오늘 바로 실천할 수 있는 선택 중심으로 작성할 것
- prayer는 4~5문장 정도로, 솔직하고 진실한 고백이 느껴지게 작성할 것
- 지나치게 무거운 표현보다는 담백하고 현실적인 문장을 사용할 것`

	case "teen":
		return `중고등부용 추가 규칙:
- audience 값은 "teen"로 설정할 것
- 제목은 학생이 바로 이해할 수 있도록 더 쉽고 분명한 말로 다시 제안할 수 있다
- 단, 적절한 새 제목이 떠오르지 않으면 사용자가 입력한 제목을 유지할 것
- summary는 3~4문장으로 작성할 것
- 문장은 짧고 분명하게 작성할 것
- 어려운 신학 용어, 추상적인 표현, 긴 설명은 줄일 것
- 학교생활, 친구관계, 가정생활, 감정, 습관과 연결될 수 있게 작성할 것
- 오늘의 메시지 paragraph는 각 2~3문장 정도로 작성할 것
- message_title은 이해하기 쉽게 짧고 분명하게 쓸 것
- reflection의 3개 항목은 학생이 바로 실천할 수 있게 구체적으로 작성할 것
- prayer는 4문장 정도로 쉽고 자연스럽게 작성할 것
- 교훈적이기만 한 말투보다 공감되고 이해되는 말투를 사용할 것`

	case "child":
		return `어린이용 추가 규칙:
- audience 값은 "child"로 설정할 것
- 제목은 어린이가 바로 이해할 수 있는 아주 쉬운 말로 다시 제안할 수 있다
- 단, 적절한 새 제목이 떠오르지 않으면 사용자가 입력한 제목을 유지할 것
- summary는 2~3문장으로 작성할 것
- 아주 쉬운 단어만 사용할 것
- 한 문장은 짧고 단순하게 작성할 것
- 어려운 신학 용어와 추상 표현은 사용하지 말 것
- 오늘의 메시지 paragraph는 각 1~2문장 정도로 작성할 것
- message_title도 어린이가 바로 이해할 수 있는 쉬운 말로 쓸 것
- reflection의 3개 항목은 아주 작고 쉬운 실천으로 작성할 것
- prayer는 3~4문장 정도로 짧고 쉬운 기도로 작성할 것
- 부드럽고 따뜻한 말투를 사용할 것
- 무섭거나 지나치게 무거운 표현은 피할 것`

	default:
		return `추가 규칙:
- audience 값은 입력된 값을 그대로 사용할 것
- 제목은 기본적으로 사용자가 입력한 제목을 우선 사용하되, 필요한 경우에만 더 자연스럽게 다듬을 것`
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

	prompt = strings.ReplaceAll(prompt, "{{title}}", meta.Title)
	prompt = strings.ReplaceAll(prompt, "{{bible_text}}", meta.BibleText)
	prompt = strings.ReplaceAll(prompt, "{{hymn}}", meta.Hymn)
	prompt = strings.ReplaceAll(prompt, "{{preacher}}", meta.Preacher)
	prompt = strings.ReplaceAll(prompt, "{{church_name}}", meta.ChurchName)
	prompt = strings.ReplaceAll(prompt, "{{sermon_date}}", meta.SermonDate)
	prompt = strings.ReplaceAll(prompt, "{{source_url}}", meta.SourceURL)
	prompt = strings.ReplaceAll(prompt, "{{raw_text}}", meta.RawText)
	prompt = strings.ReplaceAll(prompt, "{{audience}}", meta.Audience)
	prompt = strings.ReplaceAll(prompt, "{{month_accent}}", GetMonthAccentColor(now))
	prompt = strings.ReplaceAll(prompt, "{{month_name}}", GetMonthNameKorean(now))

	return prompt
}

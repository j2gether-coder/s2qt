package service

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ===== COPILOT FIX: Korean Output Only =====
// This version includes enhanced Language rules to prevent
// Copilot (and other LLMs) from outputting English text
// ============================================

const defaultQTPromptBase = `[Role]
QT Writing Assistant (Korean Church)

[Task]
Convert ASR sermon transcript into structured QT (Quiet Time) JSON 
content for church bulletin, meditation material, and blog publication.

[Important]
- This prompt works with any LLM (Claude, GPT-4o, Gemini, Mistral, etc.)
- Optimize output for accuracy, not model-specific features
- Ensure JSON validity for all parsers

[Language - CRITICAL for ALL LLMs]

Input Language: Korean (ASR transcript, may contain errors)
Output Language: NATURAL KOREAN ONLY (완벽한 자연스러운 한글)

MANDATORY OUTPUT RULES:
1. ✓ Every single value MUST be in Korean (완벽한 한글)
2. ✓ NO English words, phrases, or sentences anywhere
3. ✓ NO romanization (ㄱㄴㄷ only, not abc)
4. ✓ NO mixing languages in any field
5. ✓ If uncertain, generalize using simpler Korean (do NOT fall back to English)

SPECIFIC FIELD REQUIREMENTS:

metadata.title: 
  Must start with "[QT] " + Korean title
  Example: "[QT] 주님의 사랑에 응답하는 삶"
  ❌ Wrong: "[QT] Love God's Word" or "[QT] God's Love (신의 사랑)"

sections[*].blocks[*].text (paragraph blocks): 
  Natural Korean sentences with proper grammar
  Example: "여호사밧은 압도적인 적군을 앞두고도 자신을 바라보지 않았습니다."
  ❌ Wrong: "Jehoshaphat did not look at his abilities..."

sections[*].blocks[*].items (list items - reflection, etc): 
  Format: "주어/목적어 + 동사 + 고 있는지 생각해봅시다/살펴봅시다/확인해봅시다"
  Example 1: "주님을 사랑하는 마음으로 고백하고 있는지 생각해봅시다."
  Example 2: "내 섬김이 의무가 아닌 진정한 사랑에서 비롯되는지 살펴봅시다."
  Example 3: "내 사랑이 주변 공동체를 세우고 강하게 하는지 확인해봅시다."
  ❌ Wrong: "Consider whether you confess your love for the Lord..."

metadata.hymn: 
  Korean hymn name or standard Korean hymn format
  Example: "새찬송가 324장"
  Example: "주의 사랑을 담아"
  ❌ Wrong: "Hymn 456" or "Love Song (사랑의 노래)"

metadata.support_scriptures: 
  Korean Bible verse references
  Example: ["마태복음 11:28", "요한복음 14:1"]
  ❌ Wrong: ["Matthew 11:28", "존11:1"]

metadata.support_scriptures_full[*].text: 
  Complete Korean Bible verse text (from Korean Bible translation)
  Example: "수고하고 무거운 짐진 자들아 다 내게로 오라 내가 너희를 쉬게 하리라"
  ❌ Wrong: "Come to me, all you who are weary..."

CHINESE CHARACTERS:
✓ OK to use in Korean context: 기도(禱告), 회개(悔改), 영적(靈的), 적용(適用)
✗ NOT OK as shortcut: 心, 愛, 聖 (use Korean instead: 마음, 사랑, 거룩한)

TONE AND STYLE:
- Respectful, reverent, warm Korean
- Write like professional church bulletin or meditation material
- Natural Korean: use ~습니다, ~세요, ~해주세요 (polite endings)
- NOT formal/stiff Korean: avoid ~하라, ~할지어다
- NOT casual/slangy Korean: avoid 야, 근데, 뭐 등

WHAT TO DO IF UNSURE:
→ Use simpler, more common Korean words
→ Avoid complex theological jargon in English
→ Look for Korean equivalents
→ Do NOT fall back to English explanations

COPILOT USERS SPECIFIC NOTE:
If you see English output despite this rule, immediately request:
"Please regenerate in KOREAN ONLY. Every word must be in Korean.
Format: '주님을...고 있는지 생각해봅시다.' No English words allowed."
`

// Enhanced JSON schema with Korean examples
const defaultQTPromptJSONSchema = `[JSON Output Contract]

MANDATORY:
- Exactly one JSON object (no array, no multiple objects)
- All values are strings or arrays (no nested objects unless specified)
- No code blocks: JSON starts with { and ends with }
- UTF-8 encoded Korean text ONLY

STRUCTURE:
1. sections: exactly 4 sections in this exact order:
   a. "summary" (말씀의 길잡이)
   b. "message" (오늘의 메시지)
   c. "reflection" (깊은 묵상과 적용)
   d. "prayer" (오늘의 기도)

2. message.blocks: exactly 6 blocks (3 pairs of title + paragraph)
   [message_title, paragraph, message_title, paragraph, message_title, paragraph]

3. reflection.blocks: exactly 1 block with type "list" and 3 items

4. prayer.blocks: exactly 1 block with type "paragraph"

METADATA RULES:
- title: Must start with "[QT] " + Korean title
  Example: "[QT] 주님의 사랑에 응답하는 삶"
  
- bible_text: Must exactly equal input {{bible_text}}
  Example: "마가복음 5:1-20"
  
- bible_passage_text: Complete passage with ALL verses
  Format: "1절 [verse text]\n2절 [verse text]"
  Example:
    "1절 예수께서 건너편으로 가시매...\n2절 나가기를 원하여 공동묘지에 머물렀던 자..."
  
- hymn: Use {{hymn}} if provided; recommend 1 Korean hymn if empty; use "-" if unsure
  Example: "새찬송가 456장"
  
- support_scriptures: Array of 0-3 Korean Bible references (strings), excluding {{bible_text}}
  Example: ["마태복음 11:28", "요한복음 14:1"] or []
  
- support_scriptures_full: MUST be array of objects [{reference, text}]
  matching support_scriptures order
  Example: [
    {"reference": "마태복음 11:28", "text": "수고하고 무거운 짐진 자들아 다 내게로 오라..."},
    {"reference": "요한복음 14:1", "text": "너희 마음에 근심하지 말라..."}
  ]

[JSON Schema Template - KOREAN EXAMPLE]
{
  "version": "1.0",
  "doc_type": "qt",
  "audience": "{{audience}}",
  "template_id": "qt_classic",
  "metadata": {
    "title": "[QT] 주님의 사랑에 응답하는 삶",
    "bible_text": "{{bible_text}}",
    "bible_passage_text": "1절 설명\n2절 설명",
    "hymn": "새찬송가 456장",
    "support_scriptures": ["마태복음 11:28", "요한복음 14:1"],
    "support_scriptures_full": [
      {"reference": "마태복음 11:28", "text": "수고하고..."},
      {"reference": "요한복음 14:1", "text": "너희 마음에..."}
    ],
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
      "title": "말씀의 길잡이",
      "blocks": [
        {
          "type": "paragraph",
          "text": "여호사밧은 압도적인 적군을 앞두고도 자신의 능력을 바라보지 않았습니다. 오직 주만 바라봤던 그 믿음의 고백 앞에 하나님은 응답하셨습니다."
        }
      ]
    },
    {
      "type": "message",
      "title": "오늘의 메시지",
      "blocks": [
        {
          "type": "message_title",
          "text": "생각은 말씀으로 지키기"
        },
        {
          "type": "paragraph",
          "text": "사탄은 가장 먼저 우리의 생각을 흔듭니다. 의심을 심고 두려움을 키우며 하나님의 사랑까지 의심하게 만듭니다. 생각의 전쟁에서 승리하는 방법은 감정이 아니라 하나님의 말씀을 결론으로 삼는 것입니다."
        },
        {
          "type": "message_title",
          "text": "생활은 하나님을 기쁘시게 하기"
        },
        {
          "type": "paragraph",
          "text": "문제보다 먼저 하나님과의 관계를 회복하는 것이 승리의 시작입니다. 회개와 순종은 하나님께서 일하실 자리를 마련합니다."
        },
        {
          "type": "message_title",
          "text": "입술은 믿음과 찬양으로 선포하기"
        },
        {
          "type": "paragraph",
          "text": "찬양은 두려움을 바라보는 것이 아닌 하나님을 바라보는 믿음의 선언입니다. 원망은 마음을 무너뜨리지만 감사와 찬양은 영혼을 다시 일으켜 세웁니다."
        }
      ]
    },
    {
      "type": "reflection",
      "title": "깊은 묵상과 적용",
      "blocks": [
        {
          "type": "list",
          "items": [
            "주님을 사랑하는 마음으로 진정하게 고백하고 있는지 생각해봅시다.",
            "내 섬김이 의무가 아닌 진정한 사랑에서 비롯되는지 살펴봅시다.",
            "내 사랑이 주변 공동체를 세우고 강하게 하는지 확인해봅시다."
          ]
        }
      ]
    },
    {
      "type": "prayer",
      "title": "오늘의 기도",
      "blocks": [
        {
          "type": "paragraph",
          "text": "주님, 나의 생각을 말씀으로 지켜주세요. 세상의 의심과 두려움에 흔들리지 않고 오직 당신만 바라보는 믿음을 원합니다. 내 생활 속에서 당신을 기쁘시게 하는 순종을 이루어 주시고, 내 입술에서 언제나 감사와 찬양이 흘러나오게 해주세요. 아멘."
        }
      ]
    }
  ]
}

QUALITY CHECKS:
✓ All Korean text is grammatically correct
✓ All verses cited are within {{bible_text}}
✓ Arrays are properly formatted (not single strings)
✓ No English words or phrases anywhere
✓ No markdown, HTML, or escape characters
✓ No trailing commas in JSON
✓ Quotation marks in text are escaped properly
✓ Reflection items use Korean format: "XXX을/를 하고 있는지 생각해봅시다"
✓ Prayer is natural Korean with reverent tone`

const defaultQTPromptTranscript = `[Sermon Transcript]
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

// Enhanced audience-specific rules with Korean examples
func getAudiencePromptRules(audience string) string {
	switch strings.TrimSpace(audience) {
	case "adult":
		return `[Audience: adult]

Target: Church members 40+, professionals, mature believers

Content Guidelines (KOREAN ONLY):

Title: MUST reuse the user-provided input title EXACTLY (제목은 입력값 그대로 사용)
- Set metadata.title to the input {{title}} verbatim — do NOT rewrite, shorten, rephrase, or "improve" it
- Preserve every word; only ensure it starts with "[QT] " (add the prefix if missing, otherwise change nothing)
- Create a new title ONLY when {{title}} is empty
- Format: "[QT] 제목"
- Example: "[QT] 주님의 사랑에 응답하는 삶"

Bible Passage: MUST reuse the user-provided input passage EXACTLY (본문성구는 입력값 그대로 사용)
- Set metadata.bible_text to the input {{bible_text}} verbatim — do NOT change the book, chapter, or verse range
- The QT foundation stays fixed to the originally entered passage

Summary: 5-6 sentences, reverent and thoughtful (모두 한글)
- Example: "여호사밧은 압도적인 적군을 앞두고도 자신을 바라보지 않았습니다."

Message: 3 points with 3-5 sentences each (완벽한 한글만)
- Each point connects to personal faith maturity
- Use theological depth appropriate to audience
- Examples: "삶의 경험에서 배우는 신앙", "리더십의 책임감", "가족 속의 신앙"

Reflection: 3 items for personal examination (한글 포맷 필수)
- Format: "주어/목적어 + 동사 + 고 있는지 생각해봅시다/살펴봅시다/확인해봅시다"
- Example 1: "주님을 사랑하는 마음으로 진정하게 고백하고 있는지 생각해봅시다."
- Example 2: "내 섬김이 의무가 아닌 진정한 사랑에서 비롯되는지 살펴봅시다."
- Example 3: "내 사랑이 주변 공동체를 세우고 강하게 하는지 확인해봅시다."

Prayer: 5-6 sentences including repentance, gratitude, petition (한글만)
- Tone: Serious, humble, warm
- Structure: 경배 → 회개 → 감사 → 간구
- Example: "주님, 나의 생각을 말씀으로 지켜주세요..."

Style: Warm, reverent, profound; avoid casual or trendy language
- Use: ~습니다, ~세요, ~해주세요 (polite Korean)
- Avoid: ~하라, ~할지어다 (formal old Korean)
- Avoid: 야, 근데, 뭐 (casual Korean)

⚠️ CRITICAL: No English words anywhere. If output contains English, regenerate in KOREAN ONLY.`

	case "young_adult":
		return `[Audience: young_adult]

Target: Ages 18-35, career-focused, relational, identity-seeking

Content Guidelines (KOREAN ONLY):

Title: Refine if needed for relevance to this age group (한글만)
- Format: "[QT] 제목"

Summary: 4-5 sentences, connecting to life stage (모두 한글)
- Make it relevant to young adults

Message: 3 points with 2-4 sentences each (완벽한 한글)
- Topics: 직업, 관계, 정체성, 미래, 신앙
- Make biblical truth practical and personal

Reflection: 3 practical choices for today (한글 포맷)
- Format: "나는...할 것이다" or "이번 주 나는...할 수 있다"
- Example 1: "주님의 뜻을 찾기 위해 오늘 기도의 시간을 가질 것이다."
- Example 2: "내 삶의 결정에서 하나님을 먼저 생각해야 한다는 것을 기억할 것이다."
- Example 3: "힘들어하는 친구에게 주님의 사랑을 전할 기회를 찾을 것이다."

Prayer: 4-5 honest, sincere sentences (한글만)
- Acknowledge struggles and hopes
- Example: "주님, 이 시대 속에서 나의 신앙이 흔들리지 않게 해주세요..."

Style: Clear, realistic, warm; acknowledge questions and doubts
- Use natural Korean that young adults speak
- Not overly formal, not casual

⚠️ CRITICAL: No English phrases. All Korean only.`

	case "teen":
		return `[Audience: teen]

Target: Ages 12-18, developing identity, peer-influenced, experiential

Content Guidelines (KOREAN ONLY):

Title: Simplify if using difficult words or abstract concepts (단순한 한글)
- Format: "[QT] 쉬운 제목"

Summary: 3-4 short sentences, concrete examples (모두 한글)
- Use simple, direct language (no jargon)

Message: 3 points with 2-3 sentences each (쉬운 한글)
- Use simple words; reduce abstract theology
- Connect to: school, friends, family, emotions, habits, social media
- Avoid complex theological English terms

Reflection: 3 specific, achievable actions (한글만)
- Format: "나는...할 것이다" (simple actions)
- Example 1: "친구와 다투었을 때 먼저 용서를 구할 것이다."
- Example 2: "어려울 때 기도로 응답을 구할 것이다."
- Example 3: "학교에서 정직하게 행동할 것이다."

Prayer: About 4 sentences, easy and natural (쉬운 한글)
- Honest language, not formal prayer words
- Example: "주님, 내 마음을 도와주세요..."

Style: Empathetic, clear, not preachy
- Use words teenagers use (naturally Korean)
- Respect their growing independence

⚠️ CRITICAL: No English words at all. Simplify in Korean instead.`

	case "child":
		return `[Audience: child]

Target: Ages 6-11, concrete thinking, visual learners, playful

Content Guidelines (KOREAN ONLY):

Title: Rewrite in very easy words if original is abstract (매우 쉬운 한글)
- Format: "[QT] 아주 쉬운 제목"

Summary: 2-3 very simple sentences with clear subject/action (간단한 한글)
- Use only simple, everyday Korean words

Message: 3 points with 1-2 simple sentences each (아동용 한글)
- Avoid difficult theology and abstract terms
- Use stories, characters, actions
- Topics: 착함, 정직함, 용감함, 하나님의 사랑, 가족

Reflection: 3 small, easy actions (한글만)
- Format: "나는...할 수 있다"
- Example 1: "엄마 말씀을 잘 들을 것이다."
- Example 2: "친구에게 친절하게 할 것이다."
- Example 3: "하나님께 감사할 것이다."

Prayer: 3-4 short, gentle sentences (아주 간단한 한글)
- Simple vocabulary
- Positive tone
- Direct talk to God
- Example: "하나님, 좋은 날 주셔서 감사합니다..."

Style: Warm, simple, encouraging
- Avoid scary or heavy themes
- Use words children know (very simple Korean)

⚠️ CRITICAL: No English at all. Simplify everything in Korean.`

	default:
		return `[Audience: {{audience}}]

- Apply appropriate tone and depth
- Keep title unless unclear
- Adjust vocabulary and examples to match audience
- All content MUST be in Korean (한글만)
- No English words or phrases anywhere
- 3 message points with 2-5 sentences each
- 3 reflection items (in Korean format)
- 4-6 prayer sentences (natural Korean)

⚠️ CRITICAL: If any English appears in output, regenerate in KOREAN ONLY.`
	}
}

// BuildQTPromptJSON constructs the complete prompt for LLM processing
// COPILOT FIX: Enhanced Language rules prevent English output
func BuildQTPromptJSON(meta QTMeta) string {
	now := time.Now()

	basePrompt := strings.TrimSpace(loadQTPromptBaseTemplate())
	audienceRules := strings.TrimSpace(getAudiencePromptRules(meta.Audience))
	schemaPrompt := strings.TrimSpace(defaultQTPromptJSONSchema)
	transcriptPrompt := strings.TrimSpace(defaultQTPromptTranscript)

	// Construct prompt in optimal order for all LLMs
	prompt := strings.Join([]string{
		basePrompt,
		audienceRules,
		schemaPrompt,
		transcriptPrompt,
	}, "\n\n")

	// Template replacement
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

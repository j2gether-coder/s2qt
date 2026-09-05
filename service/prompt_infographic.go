package service

import (
	"os"
	"path/filepath"
	"strings"

	"s2qt/util"
)

// ┌─────────────────────────────────────────────────────────────────────┐
// │ 사용하지 않는 코드 (2026-08-31 통합 프롬프트 도입)                    │
// │                                                                     │
// │ 인포그래픽은 이제 QT와 함께 하나의 통합 프롬프트로 생성되며,           │
// │ sermon_summary.md는 Step1에서 LLM 원본 JSON을 렌더해 만든다.            │
// │   - 프롬프트 조립: defaultQTPromptInfographic (prompt_qt_json.go)     │
// │   - MD 렌더링   : RenderInfographicMD() (infographic_service.go)     │
// │                                                                     │
// │ 이 파일의 함수는 어디에서도 호출하지 않는다.                          │
// │ var/conf/prompt_sermon_summary.md와 함께 작성 규칙 참고용으로만 남긴다.  │
// │ (확정 8 — 코드에서는 참조하지 않음)                                   │
// └─────────────────────────────────────────────────────────────────────┘
//
// 아래는 이전 방식의 설명이다.
// 인포그래픽용 프롬프트는 var/conf/prompt_sermon_summary.md에서 읽는다.
// app.yaml의 prompt_infographic_file로 경로를 바꿀 수 있으며,
// 파일이 없으면 아래 기본 프롬프트로 새로 생성한다(var/는 배포 시 비어 있을 수 있음).

const defaultInfographicPromptFile = "prompt_sermon_summary.md"

const defaultInfographicPrompt = `# [역할]

말씀 요약문을 작성합니다. 이 문서는 말씀의 주요 내용과 흐름을 간략히 정리하여, 읽는 사람이 말씀을 이해하고 묵상하며 삶에 적용할 수 있도록 돕는 것을 목표로 합니다.

# [전처리 규칙]

- ASR 오류는 문맥상 자연스럽게 보정합니다.
- 의미를 변경하지 않습니다.
- 없는 내용을 추가하지 않습니다.
- 성경 인명, 지명, 용어, 성경구절은 가능한 정확하게 보정합니다.
- 불확실한 경우에는 원문을 유지합니다.

# [문서 목표]

- 사용자가 위에서 아래로 자연스럽게 읽으며 말씀을 이해하고, 묵상하고, 오늘의 적용까지 이어질 수 있도록 구성합니다.
- 내용 전달에 집중하며, 간결하고 명료한 표현을 사용합니다.

# [용어 규칙]

반드시 아래 용어를 사용합니다.

- 말씀의 길잡이
- 말씀을 따라
- 더하는 말씀
- 말씀의 핵심
- 오늘의 적용
- 오늘의 기도

출력 문서에는 "설교", "설교 흐름", "설교 예화" 등의 표현을 사용하지 않습니다.

# [출력 순서]

1. 제목
2. 성경본문
3. 말씀의 길잡이
4. 말씀을 따라
5. 더하는 말씀
6. 말씀의 핵심
7. 오늘의 적용
8. 오늘의 기도

# [작성 규칙]

## 제목

- 메타정보의 제목을 사용합니다.
- 짧고 명확하게 작성합니다.
- 필요한 경우 원어를 함께 표기합니다. (예: 여호와 샬롬 / יְהוָה שָׁלוֹם / YHWH Shalom)
- 원어는 말씀의 핵심 의미일 때만 사용합니다.

## 성경본문

- 본문을 정확히 표기합니다. (예: 사사기 6:24)

## 말씀의 길잡이

- 본문의 배경과 상황을 2~3문장으로 정리합니다.
- 왜 이 말씀이 필요한지를 설명합니다.

## 말씀을 따라

- 말씀의 주요 흐름과 요점을 3~5개의 단계로 정리합니다.
- 각 단계는 한 문장으로 간략히 표현합니다.
- 단계가 자연스럽게 이어지며 말씀의 중심 메시지를 전달할 수 있도록 합니다.

## 더하는 말씀

- 설교에 사용된 예화를 간략히 정리합니다.
- 예화가 본문 메시지를 어떻게 뒷받침하는지 설명합니다.

## 말씀의 핵심

- 말씀의 핵심 메시지를 한 문장으로 정리합니다.
- 예: "하나님의 평강은 두려움 가운데서도 우리를 붙드신다."

## 오늘의 적용

- 말씀을 삶에 적용하기 위한 2~3개의 실천 방안을 제시합니다.
- 구체적이고 실천 가능한 내용으로 작성합니다.

## 오늘의 기도

- 말씀을 붙들고 실천하기를 다짐하는 기도문을 2~3문장으로 작성합니다.

# [메타정보]

설교제목: {{title}}
성경본문: {{bible_text}}

# [전사문]

{{raw_text}}
`

// infographicPromptPath는 app.yaml 설정값 또는 기본 경로를 절대 경로로 반환한다.
// 실행 위치(개발/배포)에 상관없이 같은 파일을 가리키도록 프로젝트 루트 기준으로 해석한다.
func infographicPromptPath() string {
	paths, err := util.GetAppPaths()
	if err != nil || paths == nil {
		return defaultInfographicPromptFile
	}

	confPath := ""
	if cfg, err := loadAppConfig(); err == nil && cfg != nil {
		confPath = strings.TrimSpace(cfg.PromptInfographicFile)
	}

	if confPath == "" {
		return filepath.Join(paths.Conf, defaultInfographicPromptFile)
	}

	if filepath.IsAbs(confPath) {
		return confPath
	}

	return filepath.Join(paths.Root, filepath.FromSlash(confPath))
}

// loadInfographicPromptTemplate는 프롬프트 파일을 읽고,
// 파일이 없거나 비어 있으면 기본 프롬프트로 파일을 생성한 뒤 그 내용을 반환한다.
func loadInfographicPromptTemplate() string {
	path := infographicPromptPath()

	b, err := os.ReadFile(path)
	if err == nil {
		if text := strings.TrimSpace(string(b)); text != "" {
			return text
		}
	}

	writeDefaultInfographicPrompt(path)
	return strings.TrimSpace(defaultInfographicPrompt)
}

func writeDefaultInfographicPrompt(path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		LogError("infographic: 프롬프트 디렉터리 생성 실패: " + err.Error())
		return
	}

	if err := os.WriteFile(path, []byte(defaultInfographicPrompt), 0o644); err != nil {
		LogError("infographic: 기본 프롬프트 저장 실패: " + err.Error())
		return
	}

	LogInfo("infographic: 기본 프롬프트 생성 path=" + path)
}

// BuildInfographicPrompt는 프롬프트 템플릿에 메타정보와 전사문을 합쳐 완성된 MD를 반환한다.
func BuildInfographicPrompt(title string, bibleText string, rawText string) string {
	template := loadInfographicPromptTemplate()

	replacer := strings.NewReplacer(
		"{{title}}", strings.TrimSpace(title),
		"{{bible_text}}", strings.TrimSpace(bibleText),
		"{{raw_text}}", strings.TrimSpace(rawText),
	)

	return replacer.Replace(template)
}

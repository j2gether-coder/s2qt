package service

import (
	"os"
	"path/filepath"
	"strings"

	"s2qt/util"
)

// 인포그래픽용 프롬프트는 var/conf/prompt_infographic.md에서 읽는다.
// app.yaml의 prompt_infographic_file로 경로를 바꿀 수 있으며,
// 파일이 없으면 아래 기본 프롬프트로 새로 생성한다(var/는 배포 시 비어 있을 수 있음).

const defaultInfographicPromptFile = "prompt_infographic.md"

const defaultInfographicPrompt = `# [Role]

인포그래픽용 Markdown(MD)을 작성한다.

이 문서는 최종 인포그래픽 생성을 위한 중간 산출물이다.
ASR 오류는 문맥상 자연스럽게 보정한다.

단,

- 의미를 변경하지 않는다.
- 없는 내용을 추가하지 않는다.
- 성경 인명, 지명, 용어, 성경구절은 가능한 정확하게 보정한다.
- 불확실한 경우에는 원문을 유지한다.

사용자가 위에서 아래로 자연스럽게 읽으며

* 말씀을 이해하고
* 말씀을 묵상하고
* 오늘의 적용까지 이어질 수 있도록 구성한다.

디자인보다 가독성을 우선한다.

---

# [용어 규칙]

반드시 아래 용어를 사용한다.

* 말씀의 길잡이
* 말씀을 따라
* 더하는 말씀
* 말씀의 핵심
* 오늘의 적용
* 오늘의 기도

"설교", "설교 흐름", "설교 예화" 등의 표현은 사용하지 않는다.

---

# [출력 순서]

반드시 다음 순서를 유지한다.

1. 제목
2. 성경본문
3. 말씀의 길잡이
4. 말씀을 따라
5. 더하는 말씀
6. 말씀의 핵심
7. 오늘의 적용
8. 오늘의 기도

순서를 변경하지 않는다.

---

# [작성 규칙]

## 1. 제목

* 메타정보의 제목을 사용한다.
* 짧고 명확하게 작성한다.
* 필요한 경우 원어를 함께 표기한다.

예)

여호와 샬롬

יְהוָה שָׁלוֹם

YHWH Shalom

"여호와는 평강이시다"

원어는 말씀의 핵심 의미일 때만 사용한다.

---

## 2. 성경본문

본문을 정확히 표기한다.

예)

사사기 6:24

---

## 3. 말씀의 길잡이

* 본문의 배경과 상황을 2~3문장으로 정리한다.
* 왜 이 말씀이 필요한지를 설명한다.

---

## 4. 말씀을 따라

가장 중요한 영역이다.

단순 요약이 아니라 말씀의 흐름을 단계별로 정리한다.

형식:

① 상황

↓

② 하나님의 말씀

↓

③ 믿음의 반응

↓

④ 하나님의 역사

규칙:

* 3~5단계
* 각 단계는 한 문장
* 자연스럽게 이어질 것

---

## 5. 더하는 말씀

* 예화를 1개만 사용한다.
* 본문 이해를 돕는 역할만 한다.
* 짧게 정리한다.

---

## 6. 말씀의 핵심

한 문장으로 정리한다.

예)

"하나님의 평강은 두려움 가운데서도 우리를 붙드신다."

---

## 7. 오늘의 적용

2~3개의 실천 질문과 실천으로 작성한다.

예)

- 오늘 내가 하나님께 맡겨야 할 두려움은 무엇인가?
- 하나님의 평강을 신뢰하는 행동을 하나 실천해 보자.

---

## 8. 오늘의 기도

2~3문장으로 작성한다.

---

# [출력 형식]

반드시 Markdown 형식으로 출력한다.

예)

# 제목

## 성경본문

## 말씀의 길잡이

## 말씀을 따라

### ①

↓

### ②

↓

### ③

## 더하는 말씀

## 말씀의 핵심

## 오늘의 적용

## 오늘의 기도

---

# [중요]

이 문서는

"정보를 나열하는 문서"가 아니다.

말씀을 따라
독자가 한 걸음씩 묵상할 수 있도록

흐름을 만드는 것이 가장 중요한 목표이다.

# [메타정보]

설교제목: {{title}}
성경본문: {{bible_text}}

---

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

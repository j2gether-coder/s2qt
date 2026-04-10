1. HTML에서 QT 문서 구조를 파싱
2. 중간 모델(`QTDocument`)로 정리
3. 그 모델을 DOCX로 렌더링

이 3단계 구조로 가야 합니다.

---

# 1. 설계 목표

`office_service.go`의 1차 목표는 아래입니다.

* Step3 최종 HTML을 입력받는다
* QT 문서 구조를 안정적으로 추출한다
* DOCX를 텍스트 덤프가 아니라 **문서형 구조**로 만든다
* 나중에 PPT에서도 재사용 가능한 중간 모델을 만든다

즉 DOCX 생성보다 더 중요한 것은 **중간 구조 설계**입니다.

---

# 2. 권장 구조

## 파일 역할

`office_service.go`는 아래 두 역할을 가집니다.

### A. 파싱

HTML → QT 구조 데이터

### B. 렌더링

QT 구조 데이터 → DOCX

---

# 3. 핵심 데이터 모델

가장 먼저 중간 모델을 정의합니다.

```go
type QTDocument struct {
	Title           string
	BibleText       string
	Hymn            string
	Summary         []string
	Messages        []QTMessage
	ReflectionItems []string
	PrayerTitle     string
	PrayerParagraphs []string
	FooterText      string
}

type QTMessage struct {
	Title      string
	Paragraphs []string
}
```

## 왜 이렇게 나누나

### `Summary []string`

본문 요약은 문단이 1개일 수도, 여러 개일 수도 있으므로 배열이 안전합니다.

### `Messages []QTMessage`

오늘의 메시지는:

* 메시지 제목
* 메시지 본문

으로 나뉘기 때문에 구조체로 두는 게 맞습니다.

### `ReflectionItems []string`

묵상과 적용은 리스트 구조이므로 문자열 배열이 적절합니다.

### `PrayerParagraphs []string`

기도문도 문단 단위로 보관하면 나중에 Word/PPT에 재사용하기 쉽습니다.

---

# 4. 서비스 구조

권장 구조는 아래입니다.

```go
type OfficeService struct {
	Paths *util.AppPaths
}
```

그리고 메서드는 크게 4개로 나눕니다.

```go
func NewOfficeService() (*OfficeService, error)

func (s *OfficeService) SaveHtmlAndMakeDOCX(html string) (*OfficeResult, error)

func (s *OfficeService) ParseQTDocument(html string) (*QTDocument, error)

func (s *OfficeService) BuildDOCX(doc *QTDocument, outPath string) error
```

즉 흐름은:

```text
HTML
 → ParseQTDocument
 → QTDocument
 → BuildDOCX
 → temp.docx
```

---

# 5. 내부 함수 설계

## 5.1 HTML 전처리

HTML은 먼저 약간 정리합니다.

```go
func stripStyleBlock(content string) string
func decodeHTMLText(text string) string
func normalizeWhitespace(text string) string
```

역할:

* `<style>` 제거
* 엔티티 복원
* 공백 정리

---

## 5.2 구조 추출 함수

QT 문서에서 각 영역을 뽑는 함수가 필요합니다.

```go
func extractTitle(html string) string
func extractBibleTextAndHymn(html string) (string, string)
func extractSummary(html string) []string
func extractMessages(html string) []QTMessage
func extractReflectionItems(html string) []string
func extractPrayer(html string) (string, []string)
func extractFooterText(html string) string
```

### 예시 역할

#### `extractTitle`

* `<h1 class="qt-title">...</h1>` 추출

#### `extractBibleTextAndHymn`

* `.qt-subbox` 안에서

  * `본문: ...`
  * `찬송: ...`
    추출

#### `extractSummary`

* `🌿 말씀의 창: 본문 요약` 다음의 `.qt-body p` 들 추출

#### `extractMessages`

* `✨ 오늘의 메시지` 아래
* 각 `.qt-message-title` + 뒤따르는 `.qt-body p` 묶기

#### `extractReflectionItems`

* `.qt-reflection` 내부 `li` 추출

#### `extractPrayer`

* `.qt-prayer-title`
* `.qt-prayer .qt-body p`

#### `extractFooterText`

* `.qt-footer-text`

---

# 6. 파서 설계 원칙

여기서 중요한 점은 **exact string 방식은 피해야 한다**는 것입니다.

이미 Step3에서 경험하셨듯이,

* 인라인 스타일이 붙거나
* 속성 순서가 바뀌거나
* 여백/줄바꿈이 달라지면
  exact string 파서는 금방 깨집니다.

그래서 `office_service.go`도 아래 원칙으로 가야 합니다.

## 권장

* 정규식은 `class 포함 여부` 중심
* 태그 전체 exact match 금지
* `style=""` 같은 추가 속성 허용
* 내부 텍스트는 strip 후 사용

예:

* `<div class="qt-box qt-prayer" style="...">`
* `<h2 class="qt-section-title" style="...">`

모두 인식 가능해야 합니다.

---

# 7. DOCX 렌더링 설계

## 7.1 렌더링 원칙

DOCX는 HTML 디자인을 흉내 내기보다, **Word 문서다운 구조**로 만드는 게 좋습니다.

즉 아래처럼 매핑합니다.

### 제목

* 큰 글씨
* 굵게
* 중앙 정렬

### 본문/찬송

* 일반 문단
* 또는 강조 문단

### 섹션 제목

* Heading 스타일 느낌
* 굵게, 약간 크게

### 메시지 제목

* 소제목
* 굵게

### 본문

* 일반 문단

### 묵상과 적용

* bullet list

### 기도문

* 별도 문단 블록
* 약간 강조

### footer

* 작은 글씨
* 중앙 정렬

---

## 7.2 문서 생성 함수 예시

```go
func (s *OfficeService) BuildDOCX(doc *QTDocument, outPath string) error
```

내부에서 순서대로:

1. 제목 문단
2. 본문/찬송 문단
3. 섹션 제목 + summary 문단
4. 섹션 제목 + messages
5. 섹션 제목 + reflection list
6. 기도 제목 + 기도문
7. footer 문단

이 구조로 생성하면 됩니다.

---

# 8. 단계적 구현 전략

## 1단계

우선 DOCX를 이 정도 품질로 만듭니다.

* 제목
* 본문/찬송
* 섹션 제목
* 일반 문단
* bullet list
* footer

이 정도만 해도 지금보다 훨씬 문서답게 나옵니다.

## 2단계

스타일 고도화

* 제목 크기/정렬
* 섹션 간 여백
* 기도문 강조
* footer 작은 글씨

## 3단계

템플릿화

* `app.yaml`
* `style/docx template`
* 교회별 스타일 선택

---

# 9. 결과 구조체

DOCX 우선이므로 결과 구조체는 단순해도 됩니다.

```go
type OfficeResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	DocxFile string `json:"docxFile"`
	PptxFile string `json:"pptxFile"`
}
```

지금은 `DocxFile` 중심으로 쓰고,
PPT는 이후 확장합니다.

---

# 10. 추천 메서드 목록

최종적으로 `office_service.go` 안에는 아래 정도가 있으면 좋습니다.

```go
type OfficeService struct {
	Paths *util.AppPaths
}

type OfficeResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	DocxFile string `json:"docxFile"`
	PptxFile string `json:"pptxFile"`
}

type QTDocument struct {
	Title            string
	BibleText        string
	Hymn             string
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

func NewOfficeService() (*OfficeService, error)

func (s *OfficeService) SaveHtmlAndMakeDOCX(html string) (*OfficeResult, error)

func (s *OfficeService) ParseQTDocument(html string) (*QTDocument, error)

func (s *OfficeService) BuildDOCX(doc *QTDocument, outPath string) error

func extractTitle(html string) string
func extractBibleTextAndHymn(html string) (string, string)
func extractSummary(html string) []string
func extractMessages(html string) []QTMessage
func extractReflectionItems(html string) []string
func extractPrayer(html string) (string, []string)
func extractFooterText(html string) string
```

---

# 11. 설계 결론

DOCX 우선 구조의 핵심은 이것입니다.

* **HTML → QTDocument → DOCX**
* 텍스트 덤프가 아니라 문서 구조 렌더링
* 나중에 PPT도 같은 `QTDocument`를 재사용

즉 지금 `office_service.go`는 단순 변환기가 아니라
**QT 문서 렌더링 서비스의 시작점**이 되어야 합니다.


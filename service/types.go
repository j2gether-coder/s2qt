package service

// AudienceAdult는 장년용 연령대 식별자다.
// 인포그래픽은 장년 실행에서만 생성하므로 여러 곳에서 이 값을 비교한다.
const AudienceAdult = "adult"

type VideoPipelineResult struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	AudioFile      string `json:"audioFile"`
	WavFile        string `json:"wavFile"`
	TranscriptFile string `json:"transcriptFile"`
	TranscriptText string `json:"transcriptText"`
	MarkdownFile   string `json:"markdownFile"`
	Log            string `json:"log"`

	CharCount       int `json:"charCount"`
	WordCount       int `json:"wordCount"`
	LineCount       int `json:"lineCount"`
	EstimatedTokens int `json:"estimatedTokens"`

	DownloadMs   int64 `json:"downloadMs"`
	ConvertMs    int64 `json:"convertMs"`
	TranscribeMs int64 `json:"transcribeMs"`
	TotalMs      int64 `json:"totalMs"`

	TranscribeModel string `json:"transcribeModel,omitempty"`
	FallbackModel   string `json:"fallbackModel,omitempty"`
	FallbackUsed    bool   `json:"fallbackUsed,omitempty"`
	RetryReason     string `json:"retryReason,omitempty"`
}

type SourcePrepareMetrics struct {
	DownloadMs   int64 `json:"downloadMs,omitempty"`
	ConvertMs    int64 `json:"convertMs,omitempty"`
	TranscribeMs int64 `json:"transcribeMs,omitempty"`
	TotalMs      int64 `json:"totalMs,omitempty"`

	CharCount       int `json:"charCount,omitempty"`
	WordCount       int `json:"wordCount,omitempty"`
	LineCount       int `json:"lineCount,omitempty"`
	EstimatedTokens int `json:"estimatedTokens,omitempty"`

	TranscribeModel string `json:"transcribeModel,omitempty"`
	FallbackModel   string `json:"fallbackModel,omitempty"`
	FallbackUsed    bool   `json:"fallbackUsed,omitempty"`
	RetryReason     string `json:"retryReason,omitempty"`
}

type ProgressEvent struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

type AppConfig struct {
	PromptQTJSONFile      string `yaml:"prompt_qt_json_file"`
	PromptInfographicFile string `yaml:"prompt_infographic_file"`
	StyleQTHTMLFile       string `yaml:"style_qt_html_file"`
	StyleQTPDFFile        string `yaml:"style_qt_pdf_file"`
}

type QTMeta struct {
	// Series는 시리즈 설교의 시리즈명이다(예: "본받고 싶은 교회(1)").
	// 선택 항목이며, 비어 있으면 산출물에 표시하지 않는다.
	Series     string `json:"series"`
	Title      string `json:"title"`
	BibleText  string `json:"bibleText"`
	Hymn       string `json:"hymn"`
	Preacher   string `json:"preacher"`
	ChurchName string `json:"churchName"`
	SermonDate string `json:"sermonDate"`
	SourceURL  string `json:"sourceUrl"`
	RawText    string `json:"rawText"`
	Audience   string `json:"audience"`
}

// QT 준비용: temp.txt까지만 생성
type SourcePrepareRequest struct {
	SourceType  string `json:"sourceType"`  // video | audio | text
	InputMode   string `json:"inputMode"`   // url | file | paste
	SourceURL   string `json:"sourceUrl"`   // video url
	SourcePath  string `json:"sourcePath"`  // audio/text/video local file
	TextContent string `json:"textContent"` // pasted text
}

type SourcePrepareResult struct {
	Success    bool                 `json:"success"`
	Message    string               `json:"message"`
	Status     string               `json:"status"`
	SourceType string               `json:"sourceType"`
	RawText    string               `json:"rawText"`
	TxtFile    string               `json:"txtFile"`
	Steps      []string             `json:"steps"`
	Metrics    SourcePrepareMetrics `json:"metrics,omitempty"`
}

// audience Step1용: temp.json 생성
type LLMPrepareRequest struct {
	Audience   string `json:"audience"`
	Series     string `json:"series"`
	Title      string `json:"title"`
	BibleText  string `json:"bibleText"`
	Hymn       string `json:"hymn"`
	Preacher   string `json:"preacher"`
	ChurchName string `json:"churchName"`
	SermonDate string `json:"sermonDate"`
	SourceURL  string `json:"sourceUrl"`
}

type LLMPrepareResult struct {
	Success  bool     `json:"success"`
	Message  string   `json:"message"`
	Status   string   `json:"status"`
	JSONFile string   `json:"jsonFile"`
	JSONText string   `json:"jsonText,omitempty"`
	Steps    []string `json:"steps"`
}

type QTStep2Data struct {
	Audience string `json:"audience"`

	Series           string `json:"series"`
	Title            string `json:"title"`
	BibleText        string `json:"bibleText"`
	BiblePassageText string `json:"bible_passage_text"`
	Hymn             string `json:"hymn"`
	Preacher         string `json:"preacher"`
	ChurchName       string `json:"churchName"`
	SermonDate       string `json:"sermonDate"`
	SourceURL        string `json:"sourceURL"`

	SupportScriptures []string `json:"support_scriptures"`

	SummaryTitle string `json:"summaryTitle"`
	SummaryBody  string `json:"summaryBody"`

	MessageTitle1 string `json:"messageTitle1"`
	MessageBody1  string `json:"messageBody1"`
	MessageTitle2 string `json:"messageTitle2"`
	MessageBody2  string `json:"messageBody2"`
	MessageTitle3 string `json:"messageTitle3"`
	MessageBody3  string `json:"messageBody3"`

	ReflectionItem1 string `json:"reflectionItem1"`
	ReflectionItem2 string `json:"reflectionItem2"`
	ReflectionItem3 string `json:"reflectionItem3"`

	PrayerTitle string `json:"prayerTitle"`
	PrayerBody  string `json:"prayerBody"`
}

// InfographicData는 LLM이 생성한 인포그래픽 전용 필드 묶음이다.
// temp.json에는 저장하지 않고, Step1에서 sermon_summary.md로 렌더한 뒤 버린다.
// 원본은 작업내역 DB에 JSON 전문으로 보관된다.
// InfographicData는 LLM이 QT 본문과 별개로 새로 쓰는 부분을 담는다.
type InfographicData struct {
	Follow []string `json:"follow"`
	Extra  []string `json:"extra"`
	Core   string   `json:"core"`

	// 아래 3개는 2026-10-01부터 프롬프트가 생성하지 않는다.
	// 그 이전에 저장된 이력을 재작업할 때 당시 산출물을 재현하려고 읽기만 한다.
	//
	// 지우지 말 것 — encoding/json은 모르는 키를 말없이 버리므로,
	// 필드를 없애면 옛 이력의 폴백이 에러 없이 조용히 사라진다.
	Guide  string   `json:"guide,omitempty"`
	Apply  []string `json:"apply,omitempty"`
	Prayer string   `json:"prayer,omitempty"`
}

// SermonSummaryQT는 sermon_summary.md가 QT 섹션에서 재활용하는 본문이다.
// LLM이 따로 쓰지 않으므로 Step2 편집이 그대로 md에 반영된다.
//
// 값을 정하는 곳은 resolveSermonSummaryQT() 하나뿐이다.
type SermonSummaryQT struct {
	Guide  string   // sections[summary]    — 말씀의 길잡이
	Apply  []string // sections[reflection] — 깊은 묵상과 적용
	Prayer string   // sections[prayer]     — 오늘의 기도
}

// SermonSummaryMeta는 sermon_summary.md의 머리말과 꼬리말에 들어가는 값이다.
// 인포그래픽 본문(InfographicData)과 달리 이 값들은 LLM 출력이 아니거나,
// 사람이 나중에 고칠 수 있는 값이다.
//
// 인자를 나열하면 문자열 3개가 연달아 순서 실수를 잡아내지 못하므로 구조체로 받는다.
type SermonSummaryMeta struct {
	Series            string   // 화면 입력. 비면 줄 자체를 생략
	Title             string   // 장년은 화면 입력, 비장년은 LLM 제목
	BibleText         string   // 화면 입력
	SupportScriptures []string // LLM 출력 → Step2에서 편집 가능. 0개가 정상
}

// QTStep1SaveRequest는 Step1 결과저장의 입력이다.
// 화면 기본정보를 함께 받아 LLM이 날조한 메타정보 대신 사용하고,
// 작업내역 DB 저장에 필요한 필수값(Title/BibleText/Audience)을 확보한다.
type QTStep1SaveRequest struct {
	Audience   string `json:"audience"`
	Series     string `json:"series"`
	Title      string `json:"title"`
	BibleText  string `json:"bibleText"`
	Hymn       string `json:"hymn"`
	Preacher   string `json:"preacher"`
	ChurchName string `json:"churchName"`
	SermonDate string `json:"sermonDate"`
	SourceURL  string `json:"sourceURL"`
	JSONText   string `json:"jsonText"`
}

// QTStep1SaveResult의 Warnings는 화면에 표시하지 않고 로그로만 사용한다(결정 5).
type QTStep1SaveResult struct {
	TempJSONPath    string   `json:"tempJsonPath"`
	InfographicPath string   `json:"infographicPath"`
	HistoryID       int64    `json:"historyId"`
	Warnings        []string `json:"-"`
}

type QTStep2PreviewResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	HtmlFile string `json:"htmlFile"`
}

type QTStep3Request struct {
	MakeHTML bool `json:"makeHtml"`
	MakePDF  bool `json:"makePdf"`
	MakeDOCX bool `json:"makeDocx"`
	MakePPTX bool `json:"makePptx"`
	MakePNG  bool `json:"makePng"`
	DPI      int  `json:"dpi"`
}

type QTStep3FileResult struct {
	Success  bool   `json:"success"`
	Status   string `json:"status"`
	FilePath string `json:"filePath,omitempty"`
	Error    string `json:"error,omitempty"`
}

// 설교요약(sermon_summary.md) 필드는 없다. Step1에서 생성하며
// 화면에도 표시하지 않는다(결정 4).
type QTStep3Result struct {
	HTML QTStep3FileResult `json:"html"`
	PDF  QTStep3FileResult `json:"pdf"`
	DOCX QTStep3FileResult `json:"docx"`
	PPTX QTStep3FileResult `json:"pptx"`
	PNG  QTStep3FileResult `json:"png"`

	// Extended는 성구 전체 본문을 포함한 확장판 QT다(extended.html).
	Extended QTStep3FileResult `json:"extended"`
}

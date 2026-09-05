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
type InfographicData struct {
	Guide  string   `json:"guide"`
	Follow []string `json:"follow"`
	Extra  []string `json:"extra"`
	Core   string   `json:"core"`
	Apply  []string `json:"apply"`
	Prayer string   `json:"prayer"`
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

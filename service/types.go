package service

type VideoPipelineResult struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	VideoFile      string `json:"videoFile"`
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
}

type ProgressEvent struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

type AppConfig struct {
	PromptQTJSONFile string `yaml:"prompt_qt_json_file"`
	StyleQTHTMLFile  string `yaml:"style_qt_html_file"`
	StyleQTPDFFile   string `yaml:"style_qt_pdf_file"`
}

type QTMeta struct {
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
	Success    bool     `json:"success"`
	Message    string   `json:"message"`
	Status     string   `json:"status"`
	SourceType string   `json:"sourceType"`
	RawText    string   `json:"rawText"`
	TxtFile    string   `json:"txtFile"`
	Steps      []string `json:"steps"`
}

// audience Step1용: temp.json 생성
type LLMPrepareRequest struct {
	Audience   string `json:"audience"`
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

	Title      string `json:"title"`
	BibleText  string `json:"bibleText"`
	Hymn       string `json:"hymn"`
	Preacher   string `json:"preacher"`
	ChurchName string `json:"churchName"`
	SermonDate string `json:"sermonDate"`
	SourceURL  string `json:"sourceURL"`

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

type QTStep3Result struct {
	HTML QTStep3FileResult `json:"html"`
	PDF  QTStep3FileResult `json:"pdf"`
	DOCX QTStep3FileResult `json:"docx"`
	PPTX QTStep3FileResult `json:"pptx"`
	PNG  QTStep3FileResult `json:"png"`
}

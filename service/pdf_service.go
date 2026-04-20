package service

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"s2qt/util"
)

type PDFResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	MdFile   string `json:"mdFile"`
	HtmlFile string `json:"htmlFile"`
	PdfFile  string `json:"pdfFile"`
}

type PDFService struct {
	Paths *util.AppPaths
}

type QTJSONDoc struct {
	Version  string          `json:"version"`
	DocType  string          `json:"doc_type"`
	Title    string          `json:"title"`
	Subbox   string          `json:"subbox,omitempty"`
	Sections []QTJSONSection `json:"sections"`
}

type QTJSONSection struct {
	Title  string        `json:"title"`
	Blocks []QTJSONBlock `json:"blocks"`
}

type QTJSONBlock struct {
	Type  string   `json:"type"`
	Text  string   `json:"text,omitempty"`
	Title string   `json:"title,omitempty"`
	Items []string `json:"items,omitempty"`
}

const qtFooterMessage = "말씀을 묵상으로, 묵상을 삶으로"
const qtPDFScript = `
(function() {
  function fitQTPage() {
    var frame = document.querySelector('.qt-page-frame');
    var scaled = document.querySelector('.qt-page-content-scaled');
    if (!frame || !scaled) {
      return;
    }

    scaled.style.transform = 'scale(1)';
    scaled.style.width = '100%';
    scaled.style.maxWidth = '';

    var availableHeight = frame.clientHeight;
    var contentHeight = scaled.scrollHeight;
    if (!availableHeight || !contentHeight) {
      return;
    }

    var scale = Math.min(1, availableHeight / contentHeight);
    if (scale < 1) {
      scaled.style.transform = 'scale(' + scale + ')';
      scaled.style.width = (100 / scale) + '%';
      scaled.style.maxWidth = 'none';
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', fitQTPage, { once: true });
  } else {
    fitQTPage();
  }

  window.addEventListener('load', fitQTPage, { once: true });
})();
`

const qtPDFLayoutStyle = `
@page{
  size: A4;
  margin: 0;
}

html, body{
  margin: 0 !important;
  padding: 0 !important;
  background: #ffffff !important;
  width: 210mm !important;
  height: 297mm !important;
  overflow: hidden !important;
}

body{
  position: relative !important;
  box-sizing: border-box !important;
  padding: 10mm 12mm var(--qt-safe-area, 36mm) 12mm !important;
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
}

.qt-page-frame{
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.qt-page-content-scaled{
  width: 100%;
  transform-origin: top center;
}

.qt-wrap{
  width: 100% !important;
  max-width: 186mm !important;
  margin: 0 auto !important;
  padding: 0 !important;
}

.qt-footer{
  position: fixed !important;
  left: 12mm !important;
  right: 12mm !important;
  bottom: var(--qt-footer-bottom, 8mm) !important;
  height: 14mm !important;
  color: #4b5563 !important;
  z-index: 20 !important;
  pointer-events: none;
}

.qt-footer-line{
  position: absolute !important;
  left: 0 !important;
  right: calc(var(--qt-qr-reserved, 34mm) + 4mm) !important;
  top: 0 !important;
  height: 0 !important;
  border-top: 1px solid #d1d5db !important;
}

.qt-footer-brand{
  position: absolute !important;
  left: 0 !important;
  top: 3.5mm !important;
  max-width: 58mm !important;
  display: flex !important;
  align-items: center !important;
  gap: 2.5mm !important;
}

.qt-footer-logo{
  width: 8mm !important;
  height: 8mm !important;
  object-fit: contain !important;
  flex: 0 0 auto !important;
}

.qt-footer-brand-image{
  max-width: 58mm !important;
  max-height: 8mm !important;
  width: auto !important;
  height: auto !important;
  object-fit: contain !important;
  display: block !important;
}

.qt-footer-church{
  font-size: 10px !important;
  font-weight: 700 !important;
  line-height: 1.2 !important;
  white-space: nowrap !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
}

.qt-footer-text{
  position: absolute !important;
  left: 0 !important;
  right: 0 !important;
  top: 4mm !important;
  font-size: 11px !important;
  font-weight: 700 !important;
  line-height: 1.2 !important;
  text-align: center !important;
}

.qt-page-qr{
  position: fixed !important;
  bottom: var(--qt-qr-bottom, 9mm) !important;
  width: var(--qt-qr-size, 27mm) !important;
  height: var(--qt-qr-size, 27mm) !important;
  max-width: var(--qt-qr-size, 27mm) !important;
  max-height: var(--qt-qr-size, 27mm) !important;
  object-fit: contain !important;
  display: block !important;
  z-index: 30 !important;
  pointer-events: none;
}

.qt-page-qr.right-bottom{
  right: 12mm !important;
}

.qt-page-qr.left-bottom{
  left: 12mm !important;
}

h1,h2,h3,blockquote,ul,li,.qt-box,.qt-subbox{
  page-break-inside: avoid;
  break-inside: avoid;
}
`

const defaultQTHTMLStyle = `
.qt-wrap{
  --qt-bg: #ffffff;
  --qt-text: #1f2937;
  --qt-muted: #4b5563;
  --qt-title: #1f3b2f;
  --qt-green: #1f8f55;
  --qt-blue: #1d4ed8;
  --qt-blue-bg: #eaf4ff;
  --qt-blue-border: #3b82f6;
  --qt-yellow-bg: #fff8d9;
  --qt-yellow-border: #f4c542;
  --qt-purple-bg: #f4efff;
  --qt-purple-border: #b39ddb;
  --qt-line: #d1d5db;

  max-width: 760px;
  margin: 0 auto;
  padding: 0;
  font-family: 'Nanum Gothic','Apple SD Gothic Neo',sans-serif;
  line-height: 1.6;
  color: var(--qt-text);
  background: var(--qt-bg);
  font-size: 14px;
  word-break: keep-all;
}

.qt-title{
  text-align: center;
  color: var(--qt-title);
  font-size: 24px;
  font-weight: 700;
  border-bottom: 2px solid var(--qt-green);
  padding-bottom: 8px;
  margin: 0 0 10px 0;
}

.qt-subbox{
  margin: 12px 0;
  padding: 10px 14px;
  background: var(--qt-blue-bg);
  border-left: 4px solid var(--qt-blue-border);
  border-radius: 6px;
  font-weight: 700;
  font-size: 13px;
  color: var(--qt-text);
}

.qt-section-title{
  color: var(--qt-green);
  border-left: 4px solid var(--qt-green);
  padding-left: 8px;
  margin: 18px 0 10px 0;
  font-size: 18px;
  font-weight: 700;
}

.qt-message-title{
  color: var(--qt-blue);
  margin: 14px 0 6px 0;
  font-size: 16px;
  font-weight: 700;
}

.qt-box{
  margin: 8px 0;
  padding: 10px 14px;
  border-radius: 6px;
  color: var(--qt-text);
}

.qt-reflection{
  background: var(--qt-yellow-bg);
  border-left: 4px solid var(--qt-yellow-border);
}

.qt-prayer{
  background: var(--qt-purple-bg);
  border-left: 4px solid var(--qt-purple-border);
}

.qt-list{
  margin: 0;
  padding-left: 18px;
}

.qt-list li{
  margin: 5px 0;
}

.qt-body p{
  margin: 0 0 8px 0;
  color: var(--qt-text);
}

.qt-prayer-title{
  font-weight: 700;
  margin-bottom: 6px;
  color: var(--qt-text);
}

.qt-footer{
  margin-top: 28px;
  color: var(--qt-muted);
  text-align: center;
}

.qt-footer-line{
  height: 1px;
  background: var(--qt-line);
  margin-bottom: 10px;
}

.qt-footer-text{
  font-size: 13px;
}

h1,h2,h3,blockquote,ul{
  page-break-inside: avoid;
  break-inside: avoid;
}
`

const defaultQTPDFStyle = `
.qt-wrap{
  --qt-bg: #ffffff;
  --qt-text: #1f2937;
  --qt-muted: #4b5563;
  --qt-title: #1f3b2f;
  --qt-green: #1f8f55;
  --qt-blue: #1d4ed8;
  --qt-blue-bg: #eaf4ff;
  --qt-blue-border: #3b82f6;
  --qt-yellow-bg: #fff8d9;
  --qt-yellow-border: #f4c542;
  --qt-purple-bg: #f4efff;
  --qt-purple-border: #b39ddb;
  --qt-line: #d1d5db;

  width: 100%;
  max-width: 186mm;
  margin: 0 auto;
  padding: 0;
  font-family: 'Nanum Gothic','Apple SD Gothic Neo',sans-serif;
  line-height: 1.45;
  color: var(--qt-text);
  background: var(--qt-bg);
  font-size: 12px;
  word-break: keep-all;
}

.qt-main{
  width: 100%;
}

.qt-title{
  text-align: center;
  color: var(--qt-title);
  font-size: 20px;
  font-weight: 700;
  border-bottom: 2px solid var(--qt-green);
  padding-bottom: 6px;
  margin: 0 0 8px 0;
  line-height: 1.25;
}

.qt-subbox{
  margin: 8px 0 10px 0;
  padding: 8px 12px;
  background: var(--qt-blue-bg);
  border-left: 4px solid var(--qt-blue-border);
  border-radius: 6px;
  font-weight: 700;
  font-size: 12px;
  color: var(--qt-text);
  line-height: 1.4;
}

.qt-section-title{
  color: var(--qt-green);
  border-left: 4px solid var(--qt-green);
  padding-left: 8px;
  margin: 14px 0 8px 0;
  font-size: 15px;
  font-weight: 700;
  line-height: 1.3;
}

.qt-message-title{
  color: var(--qt-blue);
  margin: 10px 0 4px 0;
  font-size: 13px;
  font-weight: 700;
  line-height: 1.3;
}

.qt-box{
  margin: 6px 0;
  padding: 8px 12px;
  border-radius: 6px;
  color: var(--qt-text);
}

.qt-reflection{
  background: var(--qt-yellow-bg);
  border-left: 4px solid var(--qt-yellow-border);
}

.qt-prayer{
  background: var(--qt-purple-bg);
  border-left: 4px solid var(--qt-purple-border);
}

.qt-list{
  margin: 0;
  padding-left: 16px;
}

.qt-list li{
  margin: 4px 0;
}

.qt-body p{
  margin: 0 0 6px 0;
  color: var(--qt-text);
  line-height: 1.45;
}

.qt-prayer-title{
  font-weight: 700;
  margin-bottom: 4px;
  color: var(--qt-text);
}

.qt-footer{
  color: var(--qt-muted);
}

.qt-footer-text{
  font-size: 11px;
  font-weight: 700;
  line-height: 1.2;
}

h1,h2,h3,blockquote,ul,li,.qt-box,.qt-subbox{
  page-break-inside: avoid;
  break-inside: avoid;
}
`

func NewPDFService() (*PDFService, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return &PDFService{
		Paths: paths,
	}, nil
}

func (s *PDFService) SaveHtmlAndMakePDF(html string) (*PDFResult, error) {
	return s.SaveHtmlAndMakePDFWithFooter(html, nil)
}

func (s *PDFService) SaveHtmlAndMakePDFWithFooter(html string, footerOverride *QTFooterConfig) (*PDFResult, error) {
	html = strings.TrimSpace(html)
	if html == "" {
		return nil, fmt.Errorf("html 내용이 비어 있습니다")
	}

	if err := s.cleanupPDFTemps(); err != nil {
		return nil, err
	}

	if _, err := s.SaveHtmlAndMakeJSON(html); err != nil {
		return nil, fmt.Errorf("temp.json 저장 실패: %w", err)
	}

	fragment := stripStyleBlock(html)

	mdContent := buildMarkdownSnapshot(fragment)
	if err := os.WriteFile(s.Paths.TempMd, []byte(mdContent), 0644); err != nil {
		return nil, fmt.Errorf("temp.md 저장 실패: %w", err)
	}

	htmlContent, err := s.wrapHTMLForHTML(fragment)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(s.Paths.TempHtml, []byte(htmlContent), 0644); err != nil {
		return nil, fmt.Errorf("temp.html 저장 실패: %w", err)
	}

	pdfHTMLContent, err := s.wrapHTMLForPDF(fragment, footerOverride)
	if err != nil {
		return nil, err
	}

	pdfSourcePath := buildPDFSourcePath(s.Paths.TempHtml)
	if err := os.WriteFile(pdfSourcePath, []byte(pdfHTMLContent), 0644); err != nil {
		return nil, fmt.Errorf("pdf source html 저장 실패: %w", err)
	}
	defer os.Remove(pdfSourcePath)

	if err := s.makePDFWithEdge(pdfSourcePath, s.Paths.TempPdf); err != nil {
		return nil, err
	}

	return &PDFResult{
		Success:  true,
		Message:  "temp.md, temp.html, temp.pdf 생성이 완료되었습니다.",
		MdFile:   s.Paths.TempMd,
		HtmlFile: s.Paths.TempHtml,
		PdfFile:  s.Paths.TempPdf,
	}, nil
}

func (s *PDFService) cleanupPDFTemps() error {
	files := []string{
		s.Paths.TempMd,
		s.Paths.TempHtml,
		s.Paths.TempPdf,
		buildPDFSourcePath(s.Paths.TempHtml),
		buildJSONPath(s.Paths.TempHtml),
	}

	for _, f := range files {
		_ = os.Remove(f)
	}
	return nil
}

func buildPDFSourcePath(tempHTMLPath string) string {
	dir := filepath.Dir(tempHTMLPath)
	return filepath.Join(dir, "temp_pdf_source.html")
}

func loadQTHTMLStyle() string {
	cfg, err := loadAppConfig()
	if err != nil {
		return defaultQTHTMLStyle
	}

	stylePath := strings.TrimSpace(cfg.StyleQTHTMLFile)
	if stylePath == "" {
		return defaultQTHTMLStyle
	}

	b, err := os.ReadFile(stylePath)
	if err != nil {
		return defaultQTHTMLStyle
	}

	text := strings.TrimSpace(string(b))
	if text == "" {
		return defaultQTHTMLStyle
	}

	return text
}

func loadQTPDFStyle() string {
	cfg, err := loadAppConfig()
	if err != nil {
		return mergeQTPDFStyle(defaultQTPDFStyle)
	}

	stylePath := strings.TrimSpace(cfg.StyleQTPDFFile)
	if stylePath == "" {
		return mergeQTPDFStyle(defaultQTPDFStyle)
	}

	b, err := os.ReadFile(stylePath)
	if err != nil {
		return mergeQTPDFStyle(defaultQTPDFStyle)
	}

	text := strings.TrimSpace(string(b))
	if text == "" {
		return mergeQTPDFStyle(defaultQTPDFStyle)
	}

	return mergeQTPDFStyle(text)
}

func mergeQTPDFStyle(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = defaultQTPDFStyle
	}
	return base + "\n\n" + qtPDFLayoutStyle
}

func (s *PDFService) wrapHTMLForHTML(content string) (string, error) {
	htmlStyle := loadQTHTMLStyle()
	cleaned := normalizeHTMLFragment(content)

	return wrapHTMLDocumentForHTML("S2QT HTML", htmlStyle, cleaned), nil
}

func (s *PDFService) wrapHTMLForPDF(content string, footerOverride *QTFooterConfig) (string, error) {
	pdfStyle := loadQTPDFStyle()
	cleaned := normalizeHTMLFragment(content)

	qrSvc, err := NewQRService()
	if err != nil {
		return "", err
	}

	mode := QTFooterModeDefault
	if footerOverride != nil && footerOverride.Mode != "" {
		mode = footerOverride.Mode
	}

	resolvedFooter, err := qrSvc.PrepareFooterAssets(mode, footerOverride)
	if err != nil {
		return "", err
	}

	layoutBody := buildQTFixedPageLayout(cleaned, resolvedFooter)
	pdfStyle = mergeQTFooterRuntimeStyle(pdfStyle, resolvedFooter)

	return wrapHTMLDocumentForPDF("S2QT PDF", pdfStyle, layoutBody), nil
}

func wrapHTMLDocumentForHTML(title, css, body string) string {
	return `<!DOCTYPE html>
<html lang="ko">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + title + `</title>
</head>
<body>
<style>
` + css + `
</style>
` + body + `
</body>
</html>`
}

func wrapHTMLDocumentForPDF(title, css, body string) string {
	return `<!DOCTYPE html>
<html lang="ko">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + title + `</title>
<style>
` + css + `
</style>
</head>
<body>
` + body + `
<script>
` + qtPDFScript + `
</script>
</body>
</html>`
}

func buildQTFixedPageLayout(bodyHTML string, footerCfg *QTFooterConfig) string {
	bodyHTML = strings.TrimSpace(bodyHTML)

	return `
<div class="qt-page-frame">
  <div class="qt-page-content-scaled">
` + bodyHTML + `
  </div>
</div>
` + buildQTFooterHTML(footerCfg) + buildQTFooterQRHTML(footerCfg)
}

func mergeQTFooterRuntimeStyle(base string, footerCfg *QTFooterConfig) string {
	if footerCfg == nil {
		return base
	}

	qrReserved := 0.0
	if footerCfg.ShowQR && footerCfg.QRSizeMM > 0 {
		qrReserved = footerCfg.QRSizeMM
	}

	runtime := fmt.Sprintf(`
:root{
  --qt-safe-area: %.2fmm;
  --qt-qr-size: %.2fmm;
  --qt-qr-reserved: %.2fmm;
  --qt-footer-bottom: 8mm;
  --qt-qr-bottom: 9mm;
}
`, footerCfg.SafeAreaMM, footerCfg.QRSizeMM, qrReserved)

	return base + "\n\n" + runtime
}

func buildQTFooterHTML(cfg *QTFooterConfig) string {
	if cfg == nil || !cfg.ShowFooter {
		return ""
	}

	var parts []string
	parts = append(parts, `<div class="qt-footer" aria-hidden="true">`)
	if cfg.ShowDivider {
		parts = append(parts, `<div class="qt-footer-line"></div>`)
	}

	brandHTML := buildQTFooterBrandHTML(cfg)
	if brandHTML != "" {
		parts = append(parts, brandHTML)
	}

	footerText := strings.TrimSpace(cfg.FooterText)
	if footerText == "" {
		footerText = qtFooterMessage
	}
	if footerText != "" {
		parts = append(parts, `<div class="qt-footer-text">`+footerText+`</div>`)
	}

	parts = append(parts, `</div>`)

	return "\n" + strings.Join(parts, "\n")
}

func buildQTFooterBrandHTML(cfg *QTFooterConfig) string {
	if cfg == nil {
		return ""
	}

	brandImageData := encodeImageAsDataURI(cfg.BrandImagePath)
	if brandImageData != "" {
		return `<div class="qt-footer-brand"><img class="qt-footer-brand-image" src="` + brandImageData + `" alt="church brand" /></div>`
	}

	logoData := encodeImageAsDataURI(cfg.LogoPath)
	if logoData != "" {
		return `<div class="qt-footer-brand"><img class="qt-footer-brand-image" src="` + logoData + `" alt="church logo" /></div>`
	}

	churchName := strings.TrimSpace(cfg.ChurchName)
	if churchName == "" {
		return ""
	}

	return `<div class="qt-footer-brand"><div class="qt-footer-church">` + churchName + `</div></div>`
}

func buildQTFooterQRHTML(cfg *QTFooterConfig) string {
	if cfg == nil || !cfg.ShowQR {
		return ""
	}

	dataURI := encodeImageAsDataURI(cfg.QRImagePath)
	if dataURI == "" {
		return ""
	}

	posClass := "right-bottom"
	if strings.EqualFold(strings.TrimSpace(cfg.QRPosition), "left-bottom") {
		posClass = "left-bottom"
	}

	return `
<img class="qt-page-qr ` + posClass + `" src="` + dataURI + `" alt="footer qr" />`
}

func stripStyleBlock(content string) string {
	re := regexp.MustCompile(`(?is)<style.*?</style>`)
	cleaned := re.ReplaceAllString(content, "")
	return strings.TrimSpace(cleaned)
}

func buildMarkdownSnapshot(content string) string {
	text := content

	replacer := strings.NewReplacer(
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
		"</p>", "\n\n",
		"</div>", "\n",
		"</li>", "\n",
		"</ul>", "\n",
		"</h1>", "\n\n",
		"</h2>", "\n\n",
		"</h3>", "\n\n",
	)
	text = replacer.Replace(text)

	tagRe := regexp.MustCompile(`(?is)<[^>]+>`)
	text = tagRe.ReplaceAllString(text, "")

	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", `"`)
	text = strings.ReplaceAll(text, "&#39;", `'`)

	lines := strings.Split(text, "\n")
	var cleaned []string
	blankCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			blankCount++
			if blankCount > 1 {
				continue
			}
			cleaned = append(cleaned, "")
			continue
		}

		blankCount = 0
		cleaned = append(cleaned, line)
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func (s *PDFService) makePDFWithEdge(htmlPath, pdfPath string) error {
	edgePath, err := findEdgePath()
	if err != nil {
		return err
	}

	absHTML, err := filepath.Abs(htmlPath)
	if err != nil {
		return fmt.Errorf("HTML 절대경로 변환 실패: %w", err)
	}

	absPDF, err := filepath.Abs(pdfPath)
	if err != nil {
		return fmt.Errorf("PDF 절대경로 변환 실패: %w", err)
	}

	fileURL := "file:///" + filepath.ToSlash(absHTML)

	cmd := exec.Command(
		edgePath,
		"--headless",
		"--disable-gpu",
		"--no-pdf-header-footer",
		"--virtual-time-budget=1500",
		"--print-to-pdf="+filepath.ToSlash(absPDF),
		fileURL,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("PDF 변환 실패: %v\n%s", err, string(out))
	}

	if err := waitForFile(absPDF, 5*time.Second); err != nil {
		return err
	}

	return nil
}

func waitForFile(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		info, err := os.Stat(path)
		if err == nil && info.Size() > 0 {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("PDF 파일이 생성되지 않았습니다: %s", path)
}

func findEdgePath() (string, error) {
	candidates := []string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("Microsoft Edge 실행 파일을 찾지 못했습니다")
}

func (s *PDFService) SaveHtmlAndMakeJSON(html string) (string, error) {
	html = strings.TrimSpace(html)
	if html == "" {
		return "", fmt.Errorf("html 내용이 비어 있습니다")
	}

	fragment := stripStyleBlock(html)
	if fragment == "" {
		return "", fmt.Errorf("style 제거 후 html fragment가 비어 있습니다")
	}

	doc, err := parseQTHTMLToJSON(fragment)
	if err != nil {
		return "", err
	}

	jsonPath := buildJSONPath(s.Paths.TempHtml)
	_ = os.Remove(jsonPath)

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("temp.json 직렬화 실패: %w", err)
	}

	if err := os.WriteFile(jsonPath, b, 0644); err != nil {
		return "", fmt.Errorf("temp.json 저장 실패: %w", err)
	}

	return jsonPath, nil
}

func buildJSONPath(tempHTMLPath string) string {
	dir := filepath.Dir(tempHTMLPath)
	return filepath.Join(dir, "temp.json")
}

func parseQTHTMLToJSON(fragment string) (*QTJSONDoc, error) {
	doc := &QTJSONDoc{
		Version:  "1.0",
		DocType:  "qt",
		Sections: make([]QTJSONSection, 0),
	}

	working := strings.TrimSpace(fragment)
	doc.Title = extractFirstClassText(working, "qt-title")
	doc.Subbox = extractFirstClassText(working, "qt-subbox")

	sectionMatches := extractSectionMatches(working)
	for _, m := range sectionMatches {
		sectionTitle := cleanHTMLText(m[1])
		bodyHTML := m[2]

		section := QTJSONSection{
			Title:  sectionTitle,
			Blocks: parseSectionBlocks(sectionTitle, bodyHTML),
		}
		if section.Title == "" && len(section.Blocks) == 0 {
			continue
		}
		doc.Sections = append(doc.Sections, section)
	}

	if len(doc.Sections) == 0 {
		bodyOnly := removeKnownTopLevelBlocks(working)
		blocks := parseSectionBlocks("", bodyOnly)
		if len(blocks) > 0 {
			doc.Sections = append(doc.Sections, QTJSONSection{Title: "", Blocks: blocks})
		}
	}

	doc.Sections = normalizeQTSections(doc.Sections)
	return doc, nil
}

func extractSectionMatches(fragment string) [][]string {
	h2Re := regexp.MustCompile(`(?is)<h2[^>]*>(.*?)</h2>`)
	matches := h2Re.FindAllStringSubmatchIndex(fragment, -1)
	if len(matches) == 0 {
		return nil
	}

	type sectionPos struct {
		fullStart  int
		fullEnd    int
		titleStart int
		titleEnd   int
		title      string
	}

	sections := make([]sectionPos, 0)
	for _, m := range matches {
		if len(m) < 4 {
			continue
		}
		titleText := cleanHTMLText(fragment[m[2]:m[3]])
		if qtSectionKey(titleText) == "" {
			continue
		}
		sections = append(sections, sectionPos{
			fullStart:  m[0],
			fullEnd:    m[1],
			titleStart: m[2],
			titleEnd:   m[3],
			title:      titleText,
		})
	}

	if len(sections) == 0 {
		return nil
	}

	results := make([][]string, 0, len(sections))
	for i, s := range sections {
		bodyStart := s.fullEnd
		bodyEnd := len(fragment)
		if i+1 < len(sections) {
			bodyEnd = sections[i+1].fullStart
		}
		results = append(results, []string{
			fragment[s.fullStart:bodyEnd],
			fragment[s.titleStart:s.titleEnd],
			fragment[bodyStart:bodyEnd],
		})
	}
	return results
}

func parseSectionBlocks(sectionTitle, sectionHTML string) []QTJSONBlock {
	blocks := make([]QTJSONBlock, 0)
	remaining := sectionHTML

	for {
		loc, typ, title, full := findNextKnownBlock(remaining)
		if loc == nil {
			break
		}

		prefix := strings.TrimSpace(remaining[:loc[0]])
		blocks = append(blocks, parseSimpleParagraphs(prefix)...)

		switch typ {
		case "message_title":
			text := cleanHTMLText(full)
			if text != "" {
				blocks = append(blocks, QTJSONBlock{Type: "message_title", Text: text})
			}
		case "list":
			items := extractListItems(full)
			if len(items) > 0 {
				blocks = append(blocks, QTJSONBlock{Type: "list", Items: items})
			}
		case "reflection":
			blockText := cleanHTMLText(full)
			if blockText != "" {
				blocks = append(blocks, QTJSONBlock{
					Type:  "reflection",
					Title: firstNonEmpty(title, "깊은 묵상과 적용"),
					Text:  strings.TrimSpace(blockText),
				})
			}
		case "prayer":
			prayerTitle := extractClassInner(full, "qt-prayer-title")
			prayerText := extractParagraphText(full)
			if prayerText == "" {
				prayerText = cleanHTMLText(full)
				if prayerTitle != "" {
					prayerText = strings.TrimSpace(strings.Replace(prayerText, prayerTitle, "", 1))
				}
			}
			blocks = append(blocks, QTJSONBlock{
				Type:  "prayer",
				Title: firstNonEmpty(prayerTitle, title, "오늘의 기도"),
				Text:  strings.TrimSpace(prayerText),
			})
		}

		remaining = remaining[loc[1]:]
	}

	blocks = append(blocks, parseSimpleParagraphs(remaining)...)
	return compactBlocks(blocks)
}

func findNextKnownBlock(html string) ([]int, string, string, string) {
	candidates := []struct {
		Type string
		Re   *regexp.Regexp
	}{
		{Type: "message_title", Re: regexp.MustCompile(`(?is)<[^>]*class="[^"]*qt-message-title[^"]*"[^>]*>.*?</[^>]+>`)},
		{Type: "message_title", Re: regexp.MustCompile(`(?is)<h3[^>]*>.*?</h3>`)},
		{Type: "list", Re: regexp.MustCompile(`(?is)<[^>]*class="[^"]*qt-list[^"]*"[^>]*>.*?</ul>`)},
		{Type: "reflection", Re: regexp.MustCompile(`(?is)<[^>]*class="[^"]*qt-box[^"]*qt-reflection[^"]*"[^>]*>.*?</div>`)},
		{Type: "prayer", Re: regexp.MustCompile(`(?is)<[^>]*class="[^"]*qt-box[^"]*qt-prayer[^"]*"[^>]*>.*?</div>`)},
	}

	var bestLoc []int
	var bestType, bestTitle, bestFull string
	for _, c := range candidates {
		loc := c.Re.FindStringIndex(html)
		if loc == nil {
			continue
		}
		if bestLoc == nil || loc[0] < bestLoc[0] {
			bestLoc = loc
			bestType = c.Type
			bestFull = html[loc[0]:loc[1]]
			switch c.Type {
			case "reflection":
				bestTitle = "깊은 묵상과 적용"
			case "prayer":
				bestTitle = "오늘의 기도"
			}
		}
	}
	return bestLoc, bestType, bestTitle, bestFull
}

func parseSimpleParagraphs(html string) []QTJSONBlock {
	blocks := make([]QTJSONBlock, 0)

	pRe := regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
	matches := pRe.FindAllStringSubmatch(html, -1)
	if len(matches) > 0 {
		for _, m := range matches {
			text := cleanHTMLText(m[1])
			if text != "" {
				blocks = append(blocks, QTJSONBlock{Type: "paragraph", Text: text})
			}
		}
		return blocks
	}

	text := cleanHTMLText(html)
	if text != "" {
		blocks = append(blocks, QTJSONBlock{Type: "paragraph", Text: text})
	}
	return blocks
}

func extractListItems(html string) []string {
	liRe := regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`)
	matches := liRe.FindAllStringSubmatch(html, -1)
	items := make([]string, 0, len(matches))
	for _, m := range matches {
		text := cleanHTMLText(m[1])
		if text != "" {
			items = append(items, text)
		}
	}
	return items
}

func extractFirstClassText(html, className string) string {
	pattern := `(?is)<[^>]*class="[^"]*` + regexp.QuoteMeta(className) + `[^"]*"[^>]*>(.*?)</[^>]+>`
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(html)
	if len(m) < 2 {
		return ""
	}
	return cleanHTMLText(m[1])
}

func extractClassInner(html, className string) string {
	return extractFirstClassText(html, className)
}

func extractParagraphText(html string) string {
	pRe := regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
	matches := pRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return ""
	}

	parts := make([]string, 0, len(matches))
	for _, m := range matches {
		text := cleanHTMLText(m[1])
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func removeKnownTopLevelBlocks(html string) string {
	patterns := []string{
		`(?is)<[^>]*class="[^"]*qt-title[^"]*"[^>]*>.*?</[^>]+>`,
		`(?is)<[^>]*class="[^"]*qt-subbox[^"]*"[^>]*>.*?</[^>]+>`,
		`(?is)<[^>]*class="[^"]*qt-footer[^"]*"[^>]*>.*?</div>`,
		`(?is)<img[^>]*class="[^"]*qt-page-qr[^"]*"[^>]*>`,
	}
	result := html
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		result = re.ReplaceAllString(result, "")
	}
	return strings.TrimSpace(result)
}

func compactBlocks(blocks []QTJSONBlock) []QTJSONBlock {
	result := make([]QTJSONBlock, 0, len(blocks))
	for _, b := range blocks {
		if b.Type == "" {
			continue
		}
		if b.Type == "list" && len(b.Items) == 0 {
			continue
		}
		if b.Type != "list" && strings.TrimSpace(b.Text) == "" && strings.TrimSpace(b.Title) == "" {
			continue
		}
		result = append(result, b)
	}
	return result
}

func normalizeQTSections(sections []QTJSONSection) []QTJSONSection {
	if len(sections) == 0 {
		return sections
	}

	result := make([]QTJSONSection, 0, len(sections)+1)
	for _, sec := range sections {
		key := qtSectionKey(sec.Title)
		if key != "reflection" {
			result = append(result, sec)
			continue
		}

		refBlocks := make([]QTJSONBlock, 0)
		prayerBlocks := make([]QTJSONBlock, 0)

		for _, b := range sec.Blocks {
			if b.Type == "prayer" {
				if strings.TrimSpace(b.Text) != "" {
					prayerBlocks = append(prayerBlocks, QTJSONBlock{
						Type: "paragraph",
						Text: strings.TrimSpace(b.Text),
					})
				}
				continue
			}
			refBlocks = append(refBlocks, b)
		}

		if len(refBlocks) > 0 {
			sec.Blocks = refBlocks
			result = append(result, sec)
		}

		if len(prayerBlocks) > 0 {
			result = append(result, QTJSONSection{
				Title:  "🙏 오늘의 기도",
				Blocks: prayerBlocks,
			})
		}
	}
	return result
}

func qtSectionKey(title string) string {
	t := normalizeQTHeading(title)
	switch {
	case strings.Contains(t, "말씀의창"):
		return "summary"
	case strings.Contains(t, "오늘의메시지"):
		return "message"
	case strings.Contains(t, "깊은묵상과적용"):
		return "reflection"
	case strings.Contains(t, "오늘의기도"):
		return "prayer"
	default:
		return ""
	}
}

func normalizeQTHeading(s string) string {
	s = cleanHTMLText(s)
	re := regexp.MustCompile(`[^가-힣A-Za-z0-9]+`)
	return re.ReplaceAllString(s, "")
}

func cleanHTMLText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
		"</p>", "\n",
		"</div>", "\n",
		"</li>", "\n",
	)
	s = replacer.Replace(s)

	tagRe := regexp.MustCompile(`(?is)<[^>]+>`)
	s = tagRe.ReplaceAllString(s, "")

	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", `'`)

	lines := strings.Split(s, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(strings.TrimSpace(line)), " ")
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func normalizeHTMLFragment(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	content = stripStyleBlock(content)

	bodyRe := regexp.MustCompile(`(?is)<body[^>]*>(.*)</body>`)
	if m := bodyRe.FindStringSubmatch(content); len(m) >= 2 {
		content = strings.TrimSpace(m[1])
	}

	htmlReplacer := regexp.MustCompile(`(?is)</?(html|head|body)[^>]*>`)
	content = htmlReplacer.ReplaceAllString(content, "")

	return strings.TrimSpace(content)
}

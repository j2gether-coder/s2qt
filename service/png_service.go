package service

import (
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	"image/png"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"s2qt/util"
)

var (
	pngKernel32          = syscall.NewLazyDLL("kernel32.dll")
	pngProcRtlMoveMemory = pngKernel32.NewProc("RtlMoveMemory")
)

type pngPDFiumAPI struct {
	dll *syscall.LazyDLL

	initLibrary      *syscall.LazyProc
	destroyLibrary   *syscall.LazyProc
	getLastError     *syscall.LazyProc
	loadMemDocument  *syscall.LazyProc
	closeDocument    *syscall.LazyProc
	getPageCount     *syscall.LazyProc
	getPageSizeIndex *syscall.LazyProc
	loadPage         *syscall.LazyProc
	closePage        *syscall.LazyProc

	bitmapCreate    *syscall.LazyProc
	bitmapDestroy   *syscall.LazyProc
	bitmapFillRect  *syscall.LazyProc
	bitmapGetBuffer *syscall.LazyProc
	bitmapGetStride *syscall.LazyProc

	renderPageBitmap *syscall.LazyProc
}

type PNGService struct {
	Paths *util.AppPaths
}

func NewPNGService() (*PNGService, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return &PNGService{
		Paths: paths,
	}, nil
}

type PNGGenerateResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	HTMLFile string `json:"htmlFile"`
	PNGFile  string `json:"pngFile"`
	DPI      int    `json:"dpi"`
	WidthPx  int    `json:"widthPx"`
	HeightPx int    `json:"heightPx"`
}

func (s *PNGService) GenerateFromTempHTML(dpi int) (*PNGGenerateResult, error) {
	return s.GenerateFromTempHTMLWithFooter(dpi, nil)
}

func (s *PNGService) GenerateFromTempPDF(dpi int) (*PNGGenerateResult, error) {
	return s.GenerateFromPDFFile(s.Paths.TempPdf, s.Paths.TempPng, dpi)
}

func (s *PNGService) GenerateFromPDFFile(pdfPath, pngPath string, dpi int) (*PNGGenerateResult, error) {
	pdfPath = strings.TrimSpace(pdfPath)
	pngPath = strings.TrimSpace(pngPath)

	if pdfPath == "" {
		return nil, fmt.Errorf("pdf 경로가 비어 있습니다")
	}
	if pngPath == "" {
		return nil, fmt.Errorf("png 경로가 비어 있습니다")
	}
	if dpi <= 0 {
		dpi = 300
	}

	if _, err := os.Stat(pdfPath); err != nil {
		return nil, fmt.Errorf("pdf 파일을 찾을 수 없습니다: %w", err)
	}

	pdfiumPath := filepath.Join(s.Paths.Bin, "pdfium.dll")
	if _, err := os.Stat(pdfiumPath); err != nil {
		return nil, fmt.Errorf("pdfium.dll을 찾을 수 없습니다: %w", err)
	}

	absPNG, err := filepath.Abs(pngPath)
	if err != nil {
		return nil, fmt.Errorf("png 절대경로 변환 실패: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(absPNG), 0o755); err != nil {
		return nil, fmt.Errorf("출력 폴더 생성 실패: %w", err)
	}

	if err := renderPDFPageToPNGWithPDFium(pdfiumPath, pdfPath, absPNG, dpi, 0); err != nil {
		return nil, err
	}

	widthPx, heightPx := a4PixelSize(dpi)

	return &PNGGenerateResult{
		Success:  true,
		Message:  "PNG 생성이 완료되었습니다. (PDFium)",
		HTMLFile: s.Paths.TempHtml,
		PNGFile:  absPNG,
		DPI:      dpi,
		WidthPx:  widthPx,
		HeightPx: heightPx,
	}, nil
}

func (s *PNGService) GenerateFromTempHTMLWithFooter(dpi int, footerOverride *QTFooterConfig) (*PNGGenerateResult, error) {
	return s.GenerateFromTempHTMLWithFooterAndBG(dpi, footerOverride, false)
}

func (s *PNGService) GenerateFromHTMLFile(htmlPath, pngPath string, dpi int, transparentBG bool) (*PNGGenerateResult, error) {
	htmlPath = strings.TrimSpace(htmlPath)
	pngPath = strings.TrimSpace(pngPath)

	if htmlPath == "" {
		return nil, fmt.Errorf("html 경로가 비어 있습니다")
	}
	if pngPath == "" {
		return nil, fmt.Errorf("png 경로가 비어 있습니다")
	}

	if dpi <= 0 {
		dpi = 300
	}

	widthPx, heightPx := a4PixelSize(dpi)
	viewportWidth, viewportHeight := a4ViewportSizeCSSPx()
	deviceScaleFactor := float64(dpi) / 96.0

	if _, err := os.Stat(htmlPath); err != nil {
		return nil, fmt.Errorf("html 파일을 찾을 수 없습니다: %w", err)
	}

	absPNG, err := filepath.Abs(pngPath)
	if err != nil {
		return nil, fmt.Errorf("png 절대경로 변환 실패: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(absPNG), 0o755); err != nil {
		return nil, fmt.Errorf("출력 폴더 생성 실패: %w", err)
	}

	browserPath, err := findBrowserExecutable()
	if err != nil {
		return nil, err
	}

	fileURL, err := toFileURL(htmlPath)
	if err != nil {
		return nil, fmt.Errorf("html 파일 URL 변환 실패: %w", err)
	}

	_ = os.Remove(absPNG)

	args := []string{
		"--headless=new",
		"--disable-gpu",
		"--hide-scrollbars",
		"--force-device-scale-factor=" + strconv.FormatFloat(deviceScaleFactor, 'f', 4, 64),
		fmt.Sprintf("--window-size=%d,%d", viewportWidth, viewportHeight),
		fmt.Sprintf("--screenshot=%s", absPNG),
		fileURL,
	}

	cmd := exec.Command(browserPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("PNG 생성 실패: %w / output=%s", err, string(output))
	}

	if err := waitForGeneratedPNG(absPNG, 15*time.Second); err != nil {
		return nil, err
	}

	if transparentBG {
		if err := transparentizePNGBackground(absPNG, 248); err != nil {
			return nil, fmt.Errorf("PNG 배경 투명화 실패: %w", err)
		}
	}

	return &PNGGenerateResult{
		Success:  true,
		Message:  "PNG 생성이 완료되었습니다.",
		HTMLFile: htmlPath,
		PNGFile:  absPNG,
		DPI:      dpi,
		WidthPx:  widthPx,
		HeightPx: heightPx,
	}, nil
}

func waitForGeneratedPNG(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		info, err := os.Stat(path)
		if err == nil && info.Size() > 0 {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("PNG 파일 생성 확인 실패: %s", path)
}

func (s *PNGService) buildPNGSourceFromHTMLFile(htmlPath string, footerOverride *QTFooterConfig, transparentBG bool) (string, error) {
	b, err := os.ReadFile(htmlPath)
	if err != nil {
		return "", fmt.Errorf("html 읽기 실패: %w", err)
	}

	sourcePath := buildPNGSourcePath(htmlPath)
	sourceHTML, err := s.wrapHTMLForPNG(string(b), footerOverride, transparentBG)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(sourcePath, []byte(sourceHTML), 0o644); err != nil {
		return "", fmt.Errorf("png source html 저장 실패: %w", err)
	}

	return sourcePath, nil
}

func buildPNGSourcePath(tempHTMLPath string) string {
	dir := filepath.Dir(tempHTMLPath)
	return filepath.Join(dir, "temp_png_source.html")
}

func (s *PNGService) wrapHTMLForPNG(content string, footerOverride *QTFooterConfig, transparentBG bool) (string, error) {
	pngStyle := loadQTPNGStyle(transparentBG)
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
	pngStyle = mergeQTFooterRuntimeStylePNG(pngStyle, resolvedFooter)

	return wrapHTMLDocument("S2QT PNG", pngStyle, layoutBody), nil
}

func mergeQTFooterRuntimeStylePNG(base string, footerCfg *QTFooterConfig) string {
	if footerCfg == nil {
		return base
	}

	safeArea := footerCfg.SafeAreaMM
	if safeArea <= 0 {
		safeArea = 36.0
	}

	qrSize := footerCfg.QRSizeMM
	if qrSize <= 0 {
		qrSize = 27.0
	}

	// PNG는 PDF보다 footer 침범 방지를 위해 조금 더 여유를 둡니다.
	// 최소 40mm 확보, 또는 QR 크기 + 13mm 중 큰 값 사용
	minSafeArea := qrSize + 13.0
	if minSafeArea < 40.0 {
		minSafeArea = 40.0
	}
	if safeArea < minSafeArea {
		safeArea = minSafeArea
	}

	runtime := fmt.Sprintf(`
:root{
  --qt-page-top: 10mm;
  --qt-safe-area: %.2fmm;
  --qt-qr-size: %.2fmm;
  --qt-footer-bottom: 2mm;
  --qt-qr-bottom: 2mm;
}
`, safeArea, qrSize)

	return base + "\n\n" + runtime
}

func wrapHTMLDocument(title, css, body string) string {
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

func loadQTBaseStyle() string {
	return `
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

  --qt-passage-bg: #f8fafc;
  --qt-passage-border: #60a5fa;
  --qt-passage-title: #1f2937;
  --qt-passage-text: #374151;

  --qt-passage-abbr-bg: #fffaf0;
  --qt-passage-abbr-border: #f59e0b;

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

.qt-main,
.qt-body,
.qt-body p,
.qt-list,
.qt-list li,
.qt-subbox,
.qt-prayer-title{
  color: var(--qt-text);
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
  line-height: 1.4;
}

.qt-bible-passage{
  margin: 10px 0 16px 0;
  padding: 10px 12px;
  background: var(--qt-passage-bg);
  border-left: 4px solid var(--qt-passage-border);
  border-radius: 6px;
}

.qt-bible-passage-title{
  margin: 0 0 6px 0;
  font-size: 13px;
  font-weight: 700;
  color: var(--qt-passage-title);
}

.qt-bible-passage p{
  margin: 0;
  font-size: 12.5px;
  line-height: 1.65;
  color: var(--qt-passage-text);
  white-space: pre-line;
}

.qt-bible-passage.is-abbreviated{
  background: var(--qt-passage-abbr-bg);
  border-left-color: var(--qt-passage-abbr-border);
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

.qt-reflection,
.qt-reflection *,
.qt-prayer,
.qt-prayer *{
  color: var(--qt-text);
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
  line-height: 1.45;
}

.qt-prayer-title{
  font-weight: 700;
  margin-bottom: 4px;
}

h1,h2,h3,blockquote,ul,li,.qt-box,.qt-subbox{
  page-break-inside: avoid;
  break-inside: avoid;
}
`
}

func loadQTPNGStyle(transparentBG bool) string {
	bgColor := "#ffffff"
	if transparentBG {
		bgColor = "transparent"
	}

	return loadQTBaseStyle() + `

@page{
  size: A4;
  margin: 0;
}

html, body{
  margin: 0 !important;
  padding: 0 !important;
  background: ` + bgColor + ` !important;
  width: 210mm !important;
  height: 297mm !important;
  overflow: hidden !important;
}

body{
  position: relative !important;
  width: 210mm !important;
  height: 297mm !important;
  margin: 0 !important;
  padding:
    var(--qt-page-top, 10mm)
    12mm
    var(--qt-safe-area, 40mm)
    12mm !important;
  box-sizing: border-box !important;
  background: ` + bgColor + ` !important;
  overflow: hidden !important;
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
}

.qt-page-frame{
  position: relative !important;
  width: 100% !important;
  height: calc(297mm - var(--qt-page-top, 10mm) - var(--qt-safe-area, 40mm)) !important;
  overflow: hidden !important;
}

.qt-page-content-scaled{
  width: 100% !important;
  transform-origin: top center !important;
}

.qt-wrap{
  width: 100% !important;
  max-width: 186mm !important;
  margin: 0 auto !important;
  padding: 0 !important;
}

/* footer는 PDF 기준과 동일 */
.qt-footer{
  position: fixed !important;
  left: 12mm !important;
  right: 12mm !important;
  bottom: var(--qt-footer-bottom, 2mm) !important;
  height: 29mm !important;
  color: #4b5563 !important;
  z-index: 20 !important;
  pointer-events: none;
}

.qt-footer-line{
  position: absolute !important;
  left: 0 !important;
  right: 0 !important;
  top: 0 !important;
  height: 0 !important;
  border-top: 2px solid var(--qt-green) !important;
}

.qt-footer-grid{
  position: absolute !important;
  left: 0 !important;
  right: 0 !important;
  top: 2mm !important;
  height: var(--qt-qr-size, 27mm) !important;
  display: grid !important;
  grid-template-columns: 50mm 1fr 50mm !important;
  column-gap: 0 !important;
  align-items: center !important;
}

.qt-footer-logo-cell{
  width: 50mm !important;
  height: var(--qt-qr-size, 27mm) !important;
  display: flex !important;
  align-items: center !important;
  justify-content: flex-start !important;
  overflow: hidden !important;
}

.qt-footer-brand{
  position: static !important;
  width: 50mm !important;
  height: 18mm !important;
  display: flex !important;
  align-items: center !important;
  justify-content: flex-start !important;
  text-align: left !important;
  max-width: 50mm !important;
  overflow: hidden !important;
}

.qt-footer-brand-image{
  max-width: 50mm !important;
  max-height: 18mm !important;
  width: auto !important;
  height: auto !important;
  object-fit: contain !important;
  display: block !important;
}

.qt-footer-church{
  font-size: 9px !important;
  font-weight: 600 !important;
  line-height: 1.1 !important;
  text-align: left !important;
  white-space: nowrap !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
}

.qt-footer-text-cell{
  height: var(--qt-qr-size, 27mm) !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
  padding: 0 4mm !important;
  box-sizing: border-box !important;
  overflow: hidden !important;
}

.qt-footer-text{
  position: static !important;
  width: 100% !important;
  margin: 0 !important;
  font-size: 9.5px !important;
  font-weight: 700 !important;
  line-height: 1.25 !important;
  text-align: center !important;
  white-space: normal !important;
  word-break: keep-all !important;
}

.qt-footer-qr-cell{
  width: 50mm !important;
  height: var(--qt-qr-size, 27mm) !important;
  display: flex !important;
  align-items: center !important;
  justify-content: flex-end !important;
  overflow: hidden !important;
}

.qt-footer-qr-image{
  width: var(--qt-qr-size, 27mm) !important;
  height: var(--qt-qr-size, 27mm) !important;
  max-width: var(--qt-qr-size, 27mm) !important;
  max-height: var(--qt-qr-size, 27mm) !important;
  object-fit: contain !important;
  display: block !important;
}

.qt-page-qr,
.qt-page-qr.left-bottom,
.qt-page-qr.right-bottom{
  display: none !important;
}

.qt-subbox-line{
  display: block;
}

.qt-subbox-line + .qt-subbox-line{
  margin-top: 4px;
}
`
}

func a4PixelSize(dpi int) (int, int) {
	width := int(8.2677165*float64(dpi) + 0.5)
	height := int(11.692913*float64(dpi) + 0.5)
	return width, height
}

func a4ViewportSizeCSSPx() (int, int) {
	cssDPI := 96.0
	width := int(float64(8.2677165)*cssDPI + 0.5)
	height := int(float64(11.692913)*cssDPI + 0.5)
	return width, height
}

func toFileURL(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	slashed := filepath.ToSlash(abs)

	u := &url.URL{
		Scheme: "file",
		Path:   "/" + slashed,
	}
	return u.String(), nil
}

func findBrowserExecutable() (string, error) {
	candidates := []string{
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	for _, name := range []string{"msedge.exe", "chrome.exe"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("headless 실행 가능한 Edge/Chrome 브라우저를 찾을 수 없습니다")
}

func transparentizePNGBackground(pngPath string, whiteThreshold uint8) error {
	f, err := os.Open(pngPath)
	if err != nil {
		return err
	}
	defer f.Close()

	srcImg, err := png.Decode(f)
	if err != nil {
		return err
	}

	b := srcImg.Bounds()
	rgba := image.NewRGBA(b)
	stddraw.Draw(rgba, b, srcImg, b.Min, stddraw.Src)

	w := b.Dx()
	h := b.Dy()
	if w <= 0 || h <= 0 {
		return fmt.Errorf("invalid png size: %dx%d", w, h)
	}

	visited := make([]bool, w*h)

	type pt struct {
		x int
		y int
	}

	indexOf := func(x, y int) int {
		return y*w + x
	}

	isNearWhite := func(c color.RGBA) bool {
		return c.A > 0 &&
			c.R >= whiteThreshold &&
			c.G >= whiteThreshold &&
			c.B >= whiteThreshold
	}

	queue := make([]pt, 0, w+h)

	enqueueIfBackground := func(x, y int) {
		if x < 0 || x >= w || y < 0 || y >= h {
			return
		}
		idx := indexOf(x, y)
		if visited[idx] {
			return
		}

		c := rgba.RGBAAt(b.Min.X+x, b.Min.Y+y)
		if !isNearWhite(c) {
			return
		}

		visited[idx] = true
		queue = append(queue, pt{x: x, y: y})
	}

	// 테두리에서 시작: 가장자리와 연결된 흰 배경만 제거
	for x := 0; x < w; x++ {
		enqueueIfBackground(x, 0)
		enqueueIfBackground(x, h-1)
	}
	for y := 0; y < h; y++ {
		enqueueIfBackground(0, y)
		enqueueIfBackground(w-1, y)
	}

	for head := 0; head < len(queue); head++ {
		p := queue[head]

		px := b.Min.X + p.x
		py := b.Min.Y + p.y

		c := rgba.RGBAAt(px, py)
		c.A = 0
		rgba.SetRGBA(px, py, c)

		enqueueIfBackground(p.x-1, p.y)
		enqueueIfBackground(p.x+1, p.y)
		enqueueIfBackground(p.x, p.y-1)
		enqueueIfBackground(p.x, p.y+1)
	}

	out, err := os.Create(pngPath)
	if err != nil {
		return err
	}
	defer out.Close()

	return png.Encode(out, rgba)
}

func (s *PNGService) GenerateFromTempHTMLWithFooterAndBG(dpi int, footerOverride *QTFooterConfig, transparentBG bool) (*PNGGenerateResult, error) {
	// 1순위: PDF 기준 PNG 생성
	// 단, transparentBG는 기존 HTML 기반 경로를 유지
	if !transparentBG {
		if _, err := os.Stat(s.Paths.TempPdf); err == nil {
			result, pdfErr := s.GenerateFromTempPDF(dpi)
			if pdfErr == nil {
				return result, nil
			}
			// PDFium 실패 시 fallback
			LogError("png: PDFium render failed, fallback to HTML screenshot: " + pdfErr.Error())
		}
	}

	// 2순위: 기존 HTML screenshot fallback
	sourcePath, err := s.buildPNGSourceFromHTMLFile(s.Paths.TempHtml, footerOverride, transparentBG)
	if err != nil {
		return nil, err
	}
	defer os.Remove(sourcePath)

	return s.GenerateFromHTMLFile(sourcePath, s.Paths.TempPng, dpi, transparentBG)
}

const (
	pngFPDFAnnot   = 0x01
	pngFPDFLCDText = 0x02
)

func renderPDFPageToPNGWithPDFium(dllPath, pdfPath, pngPath string, dpi int, pageIndex int) error {
	dllPath = strings.TrimSpace(dllPath)
	pdfPath = strings.TrimSpace(pdfPath)
	pngPath = strings.TrimSpace(pngPath)

	if dllPath == "" {
		return fmt.Errorf("pdfium.dll 경로가 비어 있습니다")
	}
	if pdfPath == "" {
		return fmt.Errorf("pdf 경로가 비어 있습니다")
	}
	if pngPath == "" {
		return fmt.Errorf("png 경로가 비어 있습니다")
	}
	if dpi <= 0 {
		dpi = 300
	}
	if pageIndex < 0 {
		return fmt.Errorf("page index는 0 이상이어야 합니다")
	}

	absDLL, err := filepath.Abs(dllPath)
	if err != nil {
		return fmt.Errorf("pdfium.dll 절대경로 변환 실패: %w", err)
	}
	absPDF, err := filepath.Abs(pdfPath)
	if err != nil {
		return fmt.Errorf("pdf 절대경로 변환 실패: %w", err)
	}
	absPNG, err := filepath.Abs(pngPath)
	if err != nil {
		return fmt.Errorf("png 절대경로 변환 실패: %w", err)
	}

	if err := verifyPNGInputFile(absDLL); err != nil {
		return fmt.Errorf("pdfium.dll 확인 실패: %w", err)
	}
	if err := verifyPNGInputFile(absPDF); err != nil {
		return fmt.Errorf("pdf 확인 실패: %w", err)
	}

	api, err := loadPNGPDFium(absDLL)
	if err != nil {
		return err
	}

	api.initLibrary.Call()
	defer api.destroyLibrary.Call()

	pdfBytes, err := os.ReadFile(absPDF)
	if err != nil {
		return fmt.Errorf("pdf 읽기 실패: %w", err)
	}
	if len(pdfBytes) == 0 {
		return fmt.Errorf("pdf 파일 크기가 0입니다")
	}

	doc, err := api.loadDocumentFromMemory(pdfBytes)
	if err != nil {
		return err
	}
	defer api.closeDocument.Call(doc)
	defer runtime.KeepAlive(pdfBytes)

	pageCount := api.pageCount(doc)
	if pageCount <= 0 {
		return fmt.Errorf("pdf 페이지 수가 0입니다")
	}
	if pageIndex >= pageCount {
		return fmt.Errorf("page index 범위 오류: page=%d, pageCount=%d", pageIndex, pageCount)
	}

	widthPt, heightPt, err := api.pageSize(doc, pageIndex)
	if err != nil {
		return err
	}

	widthPx := int(math.Round(widthPt / 72.0 * float64(dpi)))
	heightPx := int(math.Round(heightPt / 72.0 * float64(dpi)))
	if widthPx <= 0 || heightPx <= 0 {
		return fmt.Errorf("렌더링 크기 계산 실패: %.2fpt x %.2fpt, dpi=%d", widthPt, heightPt, dpi)
	}

	page, err := api.loadPDFPage(doc, pageIndex)
	if err != nil {
		return err
	}
	defer api.closePage.Call(page)

	bitmap, err := api.createBitmap(widthPx, heightPx)
	if err != nil {
		return err
	}
	defer api.bitmapDestroy.Call(bitmap)

	api.bitmapFillRect.Call(
		bitmap,
		0,
		0,
		uintptr(widthPx),
		uintptr(heightPx),
		0xFFFFFFFF,
	)

	flags := uintptr(pngFPDFAnnot | pngFPDFLCDText)

	api.renderPageBitmap.Call(
		bitmap,
		page,
		0,
		0,
		uintptr(widthPx),
		uintptr(heightPx),
		0,
		flags,
	)

	img, err := api.bitmapToRGBA(bitmap, widthPx, heightPx)
	if err != nil {
		return err
	}

	_ = os.Remove(absPNG)

	out, err := os.Create(absPNG)
	if err != nil {
		return fmt.Errorf("png 생성 실패: %w", err)
	}
	defer out.Close()

	if err := png.Encode(out, img); err != nil {
		return fmt.Errorf("png 인코딩 실패: %w", err)
	}

	return nil
}

func loadPNGPDFium(dllPath string) (*pngPDFiumAPI, error) {
	dll := syscall.NewLazyDLL(dllPath)

	api := &pngPDFiumAPI{
		dll: dll,

		initLibrary:      dll.NewProc("FPDF_InitLibrary"),
		destroyLibrary:   dll.NewProc("FPDF_DestroyLibrary"),
		getLastError:     dll.NewProc("FPDF_GetLastError"),
		loadMemDocument:  dll.NewProc("FPDF_LoadMemDocument"),
		closeDocument:    dll.NewProc("FPDF_CloseDocument"),
		getPageCount:     dll.NewProc("FPDF_GetPageCount"),
		getPageSizeIndex: dll.NewProc("FPDF_GetPageSizeByIndex"),
		loadPage:         dll.NewProc("FPDF_LoadPage"),
		closePage:        dll.NewProc("FPDF_ClosePage"),

		bitmapCreate:    dll.NewProc("FPDFBitmap_Create"),
		bitmapDestroy:   dll.NewProc("FPDFBitmap_Destroy"),
		bitmapFillRect:  dll.NewProc("FPDFBitmap_FillRect"),
		bitmapGetBuffer: dll.NewProc("FPDFBitmap_GetBuffer"),
		bitmapGetStride: dll.NewProc("FPDFBitmap_GetStride"),

		renderPageBitmap: dll.NewProc("FPDF_RenderPageBitmap"),
	}

	required := map[string]*syscall.LazyProc{
		"FPDF_InitLibrary":        api.initLibrary,
		"FPDF_DestroyLibrary":     api.destroyLibrary,
		"FPDF_GetLastError":       api.getLastError,
		"FPDF_LoadMemDocument":    api.loadMemDocument,
		"FPDF_CloseDocument":      api.closeDocument,
		"FPDF_GetPageCount":       api.getPageCount,
		"FPDF_GetPageSizeByIndex": api.getPageSizeIndex,
		"FPDF_LoadPage":           api.loadPage,
		"FPDF_ClosePage":          api.closePage,
		"FPDFBitmap_Create":       api.bitmapCreate,
		"FPDFBitmap_Destroy":      api.bitmapDestroy,
		"FPDFBitmap_FillRect":     api.bitmapFillRect,
		"FPDFBitmap_GetBuffer":    api.bitmapGetBuffer,
		"FPDFBitmap_GetStride":    api.bitmapGetStride,
		"FPDF_RenderPageBitmap":   api.renderPageBitmap,
	}

	for name, proc := range required {
		if err := proc.Find(); err != nil {
			return nil, fmt.Errorf("pdfium export 함수 확인 실패: %s: %w", name, err)
		}
	}

	return api, nil
}

func (api *pngPDFiumAPI) loadDocumentFromMemory(pdfBytes []byte) (uintptr, error) {
	if len(pdfBytes) == 0 {
		return 0, fmt.Errorf("pdf 메모리 버퍼가 비어 있습니다")
	}

	r1, _, _ := api.loadMemDocument.Call(
		uintptr(unsafe.Pointer(&pdfBytes[0])),
		uintptr(len(pdfBytes)),
		0,
	)

	if r1 == 0 {
		return 0, fmt.Errorf("FPDF_LoadMemDocument 실패: pdfium_error=%d", api.lastError())
	}

	return r1, nil
}

func (api *pngPDFiumAPI) pageCount(doc uintptr) int {
	r1, _, _ := api.getPageCount.Call(doc)
	return int(r1)
}

func (api *pngPDFiumAPI) pageSize(doc uintptr, pageIndex int) (float64, float64, error) {
	var width float64
	var height float64

	r1, _, _ := api.getPageSizeIndex.Call(
		doc,
		uintptr(pageIndex),
		uintptr(unsafe.Pointer(&width)),
		uintptr(unsafe.Pointer(&height)),
	)

	if r1 == 0 {
		return 0, 0, fmt.Errorf("FPDF_GetPageSizeByIndex 실패: page=%d pdfium_error=%d", pageIndex, api.lastError())
	}

	return width, height, nil
}

func (api *pngPDFiumAPI) loadPDFPage(doc uintptr, pageIndex int) (uintptr, error) {
	r1, _, _ := api.loadPage.Call(doc, uintptr(pageIndex))
	if r1 == 0 {
		return 0, fmt.Errorf("FPDF_LoadPage 실패: page=%d pdfium_error=%d", pageIndex, api.lastError())
	}
	return r1, nil
}

func (api *pngPDFiumAPI) createBitmap(width, height int) (uintptr, error) {
	r1, _, _ := api.bitmapCreate.Call(
		uintptr(width),
		uintptr(height),
		1,
	)
	if r1 == 0 {
		return 0, fmt.Errorf("FPDFBitmap_Create 실패: %dx%d", width, height)
	}
	return r1, nil
}

func (api *pngPDFiumAPI) bitmapToRGBA(bitmap uintptr, width, height int) (*image.RGBA, error) {
	bufferPtr, _, _ := api.bitmapGetBuffer.Call(bitmap)
	if bufferPtr == 0 {
		return nil, fmt.Errorf("FPDFBitmap_GetBuffer 실패")
	}

	strideRaw, _, _ := api.bitmapGetStride.Call(bitmap)
	stride := int(strideRaw)
	if stride <= 0 {
		return nil, fmt.Errorf("FPDFBitmap_GetStride 실패: %d", stride)
	}
	if stride < width*4 {
		return nil, fmt.Errorf("bitmap stride가 너무 작습니다: stride=%d width=%d", stride, width)
	}

	rawSize := stride * height
	raw, err := copyFromPNGNativeBuffer(bufferPtr, rawSize)
	if err != nil {
		return nil, err
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		srcRow := y * stride
		dstRow := y * img.Stride

		for x := 0; x < width; x++ {
			src := srcRow + x*4
			dst := dstRow + x*4

			b := raw[src+0]
			g := raw[src+1]
			r := raw[src+2]
			a := raw[src+3]

			if a == 0 {
				a = 255
			}

			img.Pix[dst+0] = r
			img.Pix[dst+1] = g
			img.Pix[dst+2] = b
			img.Pix[dst+3] = a
		}
	}

	return img, nil
}

func (api *pngPDFiumAPI) lastError() uintptr {
	r1, _, _ := api.getLastError.Call()
	return r1
}

func copyFromPNGNativeBuffer(src uintptr, size int) ([]byte, error) {
	if src == 0 {
		return nil, fmt.Errorf("native buffer pointer is null")
	}
	if size <= 0 {
		return nil, fmt.Errorf("native buffer size is invalid: %d", size)
	}

	if err := pngProcRtlMoveMemory.Find(); err != nil {
		return nil, fmt.Errorf("RtlMoveMemory 확인 실패: %w", err)
	}

	dst := make([]byte, size)

	syscall.SyscallN(
		pngProcRtlMoveMemory.Addr(),
		uintptr(unsafe.Pointer(&dst[0])),
		src,
		uintptr(size),
	)

	return dst, nil
}

func verifyPNGInputFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("디렉토리입니다: %s", path)
	}
	if info.Size() <= 0 {
		return fmt.Errorf("파일 크기가 0입니다: %s", path)
	}
	return nil
}

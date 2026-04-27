package service

import (
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	"image/png"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"s2qt/util"
)

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

	zoneHeight := footerCfg.SafeAreaMM
	if zoneHeight <= 0 {
		zoneHeight = 32.0
	}

	qrSize := footerCfg.QRSizeMM
	if qrSize <= 0 {
		qrSize = 27.0
	}

	runtime := fmt.Sprintf(`
:root{
  --qt-page-top: 10mm;
  --qt-footer-zone-height: %.2fmm;
  --qt-footer-zone-bottom: 0mm;
  --qt-footer-side-padding: 12mm;

  --qt-footer-text-bottom: 2mm;
  --qt-footer-brand-bottom: 0mm;

  --qt-footer-qr-bottom: -2.5mm;
  --qt-footer-qr-right: 12mm;
  --qt-footer-qr-size: %.2fmm;
}
`, zoneHeight, qrSize)

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

html{
  width: 210mm;
  height: 297mm;
  margin: 0 !important;
  padding: 0 !important;
  background: ` + bgColor + ` !important;
  overflow: hidden !important;
}

body{
  position: relative !important;
  width: 210mm;
  height: 297mm;
  margin: 0 !important;
  padding:
    var(--qt-page-top, 10mm)
    var(--qt-footer-side-padding, 12mm)
    var(--qt-footer-zone-height, 32mm)
    var(--qt-footer-side-padding, 12mm) !important;
  box-sizing: border-box !important;
  background: ` + bgColor + ` !important;
  overflow: hidden !important;
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
}

.qt-page-frame{
  position: relative;
  width: 100%;
  height: 100% !important;
  overflow: hidden;
}

.qt-page-content-scaled{
  width: 100%;
  transform-origin: top center;
}

/* PNG footer zone */
.qt-footer{
  position: fixed !important;
  left: var(--qt-footer-side-padding, 12mm) !important;
  right: var(--qt-footer-side-padding, 12mm) !important;
  bottom: var(--qt-footer-zone-bottom, 0mm) !important;
  height: var(--qt-footer-zone-height, 32mm) !important;
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

.qt-footer-text{
  position: absolute !important;
  left: 50% !important;
  bottom: var(--qt-footer-text-bottom, 2mm) !important;
  top: auto !important;
  transform: translateX(-50%) !important;
  font-size: 10px !important;
  font-weight: 700 !important;
  line-height: 1.2 !important;
  text-align: center !important;
  white-space: nowrap !important;
}

.qt-footer-brand{
  position: absolute !important;
  left: 50% !important;
  bottom: var(--qt-footer-brand-bottom, 0mm) !important;
  top: auto !important;
  transform: translateX(-50%) !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
  text-align: center !important;
  gap: 1.5mm !important;
  max-width: none !important;
}

.qt-footer-brand-image{
  max-width: 40mm !important;
  max-height: 4mm !important;
  width: auto !important;
  height: auto !important;
  object-fit: contain !important;
  display: block !important;
}

.qt-footer-church{
  font-size: 9px !important;
  font-weight: 600 !important;
  line-height: 1.1 !important;
  text-align: center !important;
  white-space: nowrap !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
}

.qt-page-qr{
  position: fixed !important;
  right: var(--qt-footer-qr-right, 12mm) !important;
  bottom: var(--qt-footer-qr-bottom, 1mm) !important;
  top: auto !important;
  width: var(--qt-footer-qr-size, 27mm) !important;
  height: var(--qt-footer-qr-size, 27mm) !important;
  max-width: var(--qt-footer-qr-size, 27mm) !important;
  max-height: var(--qt-footer-qr-size, 27mm) !important;
  object-fit: contain !important;
  display: block !important;
  z-index: 30 !important;
  pointer-events: none;
}

.qt-page-qr.left-bottom{
  left: var(--qt-footer-side-padding, 12mm) !important;
  right: auto !important;
}

.qt-page-qr.right-bottom{
  right: var(--qt-footer-qr-right, 12mm) !important;
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
	sourcePath, err := s.buildPNGSourceFromHTMLFile(s.Paths.TempHtml, footerOverride, transparentBG)
	if err != nil {
		return nil, err
	}
	defer os.Remove(sourcePath)

	return s.GenerateFromHTMLFile(sourcePath, s.Paths.TempPng, dpi, transparentBG)
}

package service

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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
	sourcePath, err := s.buildPNGSourceFromHTMLFile(s.Paths.TempHtml, footerOverride)
	if err != nil {
		return nil, err
	}
	defer os.Remove(sourcePath)

	return s.GenerateFromHTMLFile(sourcePath, s.Paths.TempPng, dpi)
}

func (s *PNGService) GenerateFromHTMLFile(htmlPath, pngPath string, dpi int) (*PNGGenerateResult, error) {
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

	if err := os.MkdirAll(filepath.Dir(pngPath), 0755); err != nil {
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

	_ = os.Remove(pngPath)

	args := []string{
		"--headless=new",
		"--disable-gpu",
		"--hide-scrollbars",
		"--force-device-scale-factor=" + strconv.FormatFloat(deviceScaleFactor, 'f', 4, 64),
		fmt.Sprintf("--window-size=%d,%d", viewportWidth, viewportHeight),
		fmt.Sprintf("--screenshot=%s", pngPath),
		fileURL,
	}

	cmd := exec.Command(browserPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("PNG 생성 실패: %w / output=%s", err, string(output))
	}

	if _, err := os.Stat(pngPath); err != nil {
		return nil, fmt.Errorf("PNG 파일 생성 확인 실패: %w", err)
	}

	return &PNGGenerateResult{
		Success:  true,
		Message:  "PNG 생성이 완료되었습니다.",
		HTMLFile: htmlPath,
		PNGFile:  pngPath,
		DPI:      dpi,
		WidthPx:  widthPx,
		HeightPx: heightPx,
	}, nil
}

func (s *PNGService) buildPNGSourceFromHTMLFile(htmlPath string, footerOverride *QTFooterConfig) (string, error) {
	b, err := os.ReadFile(htmlPath)
	if err != nil {
		return "", fmt.Errorf("html 읽기 실패: %w", err)
	}

	sourcePath := buildPNGSourcePath(htmlPath)
	sourceHTML, err := s.wrapHTMLForPNG(string(b), footerOverride)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(sourcePath, []byte(sourceHTML), 0644); err != nil {
		return "", fmt.Errorf("png source html 저장 실패: %w", err)
	}

	return sourcePath, nil
}

func buildPNGSourcePath(tempHTMLPath string) string {
	dir := filepath.Dir(tempHTMLPath)
	return filepath.Join(dir, "temp_png_source.html")
}

func (s *PNGService) wrapHTMLForPNG(content string, footerOverride *QTFooterConfig) (string, error) {
	pngStyle := loadQTPNGStyle()
	cleaned := normalizeHTMLFragment(content)
	qrSvc, err := NewQRService()
	if err != nil {
		return "", err
	}
	resolvedFooter, err := qrSvc.PrepareFooterAssets(QTFooterModeDefault, footerOverride)
	if err != nil {
		return "", err
	}
	layoutBody := buildQTFixedPageLayout(cleaned, resolvedFooter)
	pngStyle = mergeQTFooterRuntimeStyle(pngStyle, resolvedFooter)

	return wrapHTMLDocument("S2QT PNG", pngStyle, layoutBody), nil
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
</body>
</html>`
}

func loadQTPNGStyle() string {
	return loadQTPDFStyle() + `

html{
  width: 210mm;
  height: 297mm;
  margin: 0;
  padding: 0;
  background: #ffffff;
  overflow: hidden;
}

body{
  width: 210mm;
  height: 297mm;
  margin: 0 !important;
  padding: 0 !important;
  box-sizing: border-box;
  background: #ffffff;
  overflow: hidden;
}

.qt-page-frame{
  height: calc(297mm - var(--qt-safe-area, 36mm) - 10mm) !important;
}

.qt-wrap{
  width: 100%;
  max-width: 186mm;
  margin: 0 auto;
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

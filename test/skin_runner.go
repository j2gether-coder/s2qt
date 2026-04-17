package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"mime"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"s2qt/service"
	"s2qt/util"
)

type SkinTestInput struct {
	Enabled           bool    `json:"enabled"`
	SkinImagePath     string  `json:"skin_image_path"`
	OutputHTMLPath    string  `json:"output_html_path"`
	OutputPDFPath     string  `json:"output_pdf_path"`
	OutputPNGPath     string  `json:"output_png_path"`
	SafeTopMM         float64 `json:"safe_top_mm"`
	SafeRightMM       float64 `json:"safe_right_mm"`
	SafeBottomMM      float64 `json:"safe_bottom_mm"`
	SafeLeftMM        float64 `json:"safe_left_mm"`
	ContentMaxWidthMM float64 `json:"content_max_width_mm"`
	DebugSafeArea     bool    `json:"debug_safe_area"`
}

func loadSkinTestJSON(path string) (*SkinTestInput, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read test_skin.json: %w", err)
	}

	var in SkinTestInput
	if err := json.Unmarshal(b, &in); err != nil {
		return nil, fmt.Errorf("failed to parse test_skin.json: %w", err)
	}

	if !in.Enabled {
		return &in, nil
	}

	in.SkinImagePath = strings.TrimSpace(in.SkinImagePath)
	in.OutputHTMLPath = strings.TrimSpace(in.OutputHTMLPath)
	in.OutputPDFPath = strings.TrimSpace(in.OutputPDFPath)
	in.OutputPNGPath = strings.TrimSpace(in.OutputPNGPath)

	if in.SkinImagePath == "" {
		return nil, fmt.Errorf("skin_image_path is empty")
	}
	if in.OutputHTMLPath == "" {
		in.OutputHTMLPath = filepath.Join("var", "temp", "temp_skin_source.html")
	}
	if in.OutputPDFPath == "" {
		in.OutputPDFPath = filepath.Join("var", "temp", "temp_skin.pdf")
	}
	if in.OutputPNGPath == "" {
		in.OutputPNGPath = filepath.Join("var", "temp", "temp_skin.png")
	}
	if in.SafeTopMM <= 0 {
		in.SafeTopMM = 28
	}
	if in.SafeRightMM <= 0 {
		in.SafeRightMM = 18
	}
	if in.SafeBottomMM <= 0 {
		in.SafeBottomMM = 46
	}
	if in.SafeLeftMM <= 0 {
		in.SafeLeftMM = 18
	}
	if in.ContentMaxWidthMM <= 0 {
		in.ContentMaxWidthMM = 172
	}

	return &in, nil
}

// 테스트는 산출물 일부에 이슈가 있어도 가능한 범위까지 계속 진행한다.
func runSkinTest(paths *util.AppPaths, db *sql.DB, skin *SkinTestInput) []string {
	var outputs []string

	if paths == nil {
		fmt.Println("[WARN] skin test skipped: paths is nil")
		return outputs
	}
	if db == nil {
		fmt.Println("[WARN] skin test skipped: db is nil")
		return outputs
	}
	if skin == nil || !skin.Enabled {
		fmt.Println("[INFO] skin test skipped: disabled")
		return outputs
	}

	htmlBytes, err := os.ReadFile(paths.TempHtml)
	if err != nil {
		fmt.Printf("[WARN] skin temp.html read failed: %v\n", err)
		return outputs
	}

	footerSvc, err := service.NewFooterService(db)
	if err != nil {
		fmt.Printf("[WARN] skin footer service create failed: %v\n", err)
		return outputs
	}

	footerCfg, err := footerSvc.PrepareFooterConfigFromDB(service.QTFooterModeSubscriber)
	if err != nil {
		fmt.Printf("[WARN] skin footer config build failed: %v\n", err)
		return outputs
	}

	cleaned := extractBodyFragment(stripStyleBlocks(string(htmlBytes)))
	renderHTML := buildSkinHTML(cleaned, footerCfg, skin)

	if err := os.MkdirAll(filepath.Dir(skin.OutputHTMLPath), 0o755); err != nil {
		fmt.Printf("[WARN] skin html dir create failed: %v\n", err)
		return outputs
	}
	if err := os.MkdirAll(filepath.Dir(skin.OutputPDFPath), 0o755); err != nil {
		fmt.Printf("[WARN] skin pdf dir create failed: %v\n", err)
	}
	if err := os.MkdirAll(filepath.Dir(skin.OutputPNGPath), 0o755); err != nil {
		fmt.Printf("[WARN] skin png dir create failed: %v\n", err)
	}

	if err := os.WriteFile(skin.OutputHTMLPath, []byte(renderHTML), 0o644); err != nil {
		fmt.Printf("[WARN] skin html write failed: %v\n", err)
		return outputs
	}
	fmt.Printf("[OK] skin html : %s\n", skin.OutputHTMLPath)
	outputs = append(outputs, skin.OutputHTMLPath)

	if err := makeSkinPDF(skin.OutputHTMLPath, skin.OutputPDFPath); err != nil {
		fmt.Printf("[WARN] skin pdf failed: %v\n", err)
	} else {
		fmt.Printf("[OK] skin pdf  : %s\n", skin.OutputPDFPath)
		outputs = append(outputs, skin.OutputPDFPath)
	}

	if err := makeSkinPNGByScreenshot(skin.OutputHTMLPath, skin.OutputPNGPath, 300); err != nil {
		fmt.Printf("[WARN] skin png failed: %v\n", err)
	} else {
		fmt.Printf("[OK] skin png  : %s\n", skin.OutputPNGPath)
		outputs = append(outputs, skin.OutputPNGPath)
	}

	return outputs
}

func buildSkinHTML(bodyHTML string, footerCfg *service.QTFooterConfig, skin *SkinTestInput) string {
	skinData := encodeFileAsDataURI(skin.SkinImagePath)

	safeBottom := skin.SafeBottomMM
	if footerCfg != nil && footerCfg.SafeAreaMM > safeBottom {
		safeBottom = footerCfg.SafeAreaMM
	}

	qrReserved := 0.0
	qrSize := 27.0
	qrBottom := 12.0
	footerBottom := 12.0
	qrPosition := "right-bottom"

	if footerCfg != nil {
		if footerCfg.ShowQR && footerCfg.QRSizeMM > 0 {
			qrReserved = footerCfg.QRSizeMM
			qrSize = footerCfg.QRSizeMM
		}
		if strings.TrimSpace(footerCfg.QRPosition) != "" {
			qrPosition = strings.TrimSpace(footerCfg.QRPosition)
		}
	}

	debugHTML := ""
	if skin.DebugSafeArea {
		debugHTML = `<div class="skin-safe-guide" aria-hidden="true"></div>`
	}

	return `<!DOCTYPE html>
<html lang="ko">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>S2QT Skin Test</title>
<style>
@page{
  size:A4;
  margin:0;
}

html, body{
  margin:0 !important;
  padding:0 !important;
  width:210mm !important;
  height:297mm !important;
  overflow:hidden !important;
  background:#ffffff !important;
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
}

body{
  position:relative;
  box-sizing:border-box;
  color:#1f2937;
  font-family:'Nanum Gothic','Malgun Gothic','Apple SD Gothic Neo',sans-serif;
}

.skin-page{
  position:relative;
  width:210mm;
  height:297mm;
  overflow:hidden;
  background-image:url('` + skinData + `');
  background-repeat:no-repeat;
  background-size:210mm 297mm;
  background-position:center center;
}

.skin-safe-guide{
  position:absolute;
  left:` + formatMM(skin.SafeLeftMM) + `;
  top:` + formatMM(skin.SafeTopMM) + `;
  right:` + formatMM(skin.SafeRightMM) + `;
  bottom:` + formatMM(safeBottom) + `;
  border:1px dashed rgba(220,38,38,.8);
  box-sizing:border-box;
  z-index:8;
  pointer-events:none;
}

.qt-page-frame{
  position:absolute;
  left:` + formatMM(skin.SafeLeftMM) + `;
  top:` + formatMM(skin.SafeTopMM) + `;
  right:` + formatMM(skin.SafeRightMM) + `;
  bottom:` + formatMM(safeBottom) + `;
  overflow:hidden;
  z-index:10;
}

.qt-page-content-scaled{
  width:100%;
  transform-origin:top center;
}

.qt-wrap{
  width:100%;
  max-width:` + formatMM(skin.ContentMaxWidthMM) + `;
  margin:0 auto;
  padding:0;
  box-sizing:border-box;
  line-height:1.45;
  color:#1f2937;
  font-size:12px;
  word-break:keep-all;
}

.qt-main{
  width:100%;
}

.qt-title{
  text-align:center;
  color:#1f3b2f;
  font-size:20px;
  font-weight:700;
  border-bottom:2px solid #1f8f55;
  padding-bottom:6px;
  margin:0 0 8px 0;
  line-height:1.25;
}

.qt-subbox{
  margin:8px 0 10px 0;
  padding:8px 12px;
  background:#eaf4ff;
  border-left:4px solid #3b82f6;
  border-radius:6px;
  font-weight:700;
  font-size:12px;
  color:#1f2937;
  line-height:1.4;
}

.qt-section-title{
  color:#1f8f55;
  border-left:4px solid #1f8f55;
  padding-left:8px;
  margin:14px 0 8px 0;
  font-size:15px;
  font-weight:700;
  line-height:1.3;
}

.qt-message-title{
  color:#1d4ed8;
  margin:10px 0 4px 0;
  font-size:13px;
  font-weight:700;
  line-height:1.3;
}

.qt-box{
  margin:6px 0;
  padding:8px 12px;
  border-radius:6px;
  color:#1f2937;
}

.qt-reflection{
  background:#fff8d9;
  border-left:4px solid #f4c542;
}

.qt-prayer{
  background:#f4efff;
  border-left:4px solid #b39ddb;
}

.qt-list{
  margin:0;
  padding-left:16px;
}

.qt-list li{
  margin:4px 0;
}

.qt-body p{
  margin:0 0 6px 0;
  line-height:1.45;
}

.qt-prayer-title{
  font-weight:700;
  margin-bottom:4px;
}

.qt-footer{
  position:absolute;
  left:` + formatMM(skin.SafeLeftMM) + `;
  right:` + formatMM(skin.SafeRightMM) + `;
  bottom:` + formatMM(footerBottom) + `;
  height:14mm;
  color:#4b5563;
  z-index:20;
  pointer-events:none;
}

.qt-footer-line{
  position:absolute;
  left:0;
  right:calc(` + formatMM(qrReserved) + ` + 4mm);
  top:0;
  height:0;
  border-top:1px solid #d1d5db;
}

.qt-footer-brand{
  position:absolute;
  left:0;
  top:3.5mm;
  max-width:58mm;
  display:flex;
  align-items:center;
  gap:2.5mm;
}

.qt-footer-brand-image{
  max-width:58mm;
  max-height:8mm;
  width:auto;
  height:auto;
  object-fit:contain;
  display:block;
}

.qt-footer-church{
  font-size:10px;
  font-weight:700;
  line-height:1.2;
  white-space:nowrap;
  overflow:hidden;
  text-overflow:ellipsis;
}

.qt-footer-text{
  position:absolute;
  left:0;
  right:0;
  top:4mm;
  font-size:11px;
  font-weight:700;
  line-height:1.2;
  text-align:center;
}

.qt-page-qr{
  position:absolute;
  bottom:` + formatMM(qrBottom) + `;
  width:` + formatMM(qrSize) + `;
  height:` + formatMM(qrSize) + `;
  object-fit:contain;
  z-index:30;
  pointer-events:none;
}

.qt-page-qr.right-bottom{
  right:` + formatMM(skin.SafeRightMM) + `;
}

.qt-page-qr.left-bottom{
  left:` + formatMM(skin.SafeLeftMM) + `;
}

h1,h2,h3,blockquote,ul,li,.qt-box,.qt-subbox{
  page-break-inside:avoid;
  break-inside:avoid;
}
</style>
</head>
<body>
<div class="skin-page">
` + debugHTML + `
  <div class="qt-page-frame">
    <div class="qt-page-content-scaled">
` + bodyHTML + `
    </div>
  </div>
` + buildTestFooterHTML(footerCfg) + `
` + buildTestQRHTML(footerCfg, qrPosition) + `
</div>
<script>
(function() {
  function fitQTPage() {
    var frame = document.querySelector('.qt-page-frame');
    var scaled = document.querySelector('.qt-page-content-scaled');
    if (!frame || !scaled) return;

    scaled.style.transform = 'scale(1)';
    scaled.style.width = '100%';
    scaled.style.maxWidth = '';

    var availableHeight = frame.clientHeight;
    var contentHeight = scaled.scrollHeight;
    if (!availableHeight || !contentHeight) return;

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
</script>
</body>
</html>`
}

func buildTestFooterHTML(cfg *service.QTFooterConfig) string {
	if cfg == nil || !cfg.ShowFooter {
		return ""
	}

	var parts []string
	parts = append(parts, `<div class="qt-footer" aria-hidden="true">`)
	if cfg.ShowDivider {
		parts = append(parts, `<div class="qt-footer-line"></div>`)
	}

	brandHTML := buildTestFooterBrandHTML(cfg)
	if brandHTML != "" {
		parts = append(parts, brandHTML)
	}

	footerText := strings.TrimSpace(cfg.FooterText)
	if footerText == "" {
		footerText = "말씀을 묵상으로, 묵상을 삶으로"
	}
	parts = append(parts, `<div class="qt-footer-text">`+escapeHTML(footerText)+`</div>`)
	parts = append(parts, `</div>`)

	return strings.Join(parts, "\n")
}

func buildTestFooterBrandHTML(cfg *service.QTFooterConfig) string {
	if cfg == nil {
		return ""
	}

	if data := encodeFileAsDataURI(cfg.BrandImagePath); data != "" {
		return `<div class="qt-footer-brand"><img class="qt-footer-brand-image" src="` + data + `" alt="church brand" /></div>`
	}

	if data := encodeFileAsDataURI(cfg.LogoPath); data != "" {
		return `<div class="qt-footer-brand"><img class="qt-footer-brand-image" src="` + data + `" alt="church logo" /></div>`
	}

	churchName := strings.TrimSpace(cfg.ChurchName)
	if churchName != "" {
		return `<div class="qt-footer-brand"><div class="qt-footer-church">` + escapeHTML(churchName) + `</div></div>`
	}

	return ""
}

func buildTestQRHTML(cfg *service.QTFooterConfig, posClass string) string {
	if cfg == nil || !cfg.ShowQR {
		return ""
	}

	data := encodeFileAsDataURI(cfg.QRImagePath)
	if data == "" {
		return ""
	}

	if posClass == "" {
		posClass = "right-bottom"
	}

	return `<img class="qt-page-qr ` + posClass + `" src="` + data + `" alt="footer qr" />`
}

func stripStyleBlocks(v string) string {
	re := regexp.MustCompile(`(?is)<style.*?</style>`)
	return strings.TrimSpace(re.ReplaceAllString(v, ""))
}

func extractBodyFragment(v string) string {
	re := regexp.MustCompile(`(?is)<body[^>]*>(.*?)</body>`)
	m := re.FindStringSubmatch(v)
	if len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(v)
}

func formatMM(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64) + "mm"
}

func encodeFileAsDataURI(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	ext := strings.ToLower(filepath.Ext(path))
	mtype := mime.TypeByExtension(ext)
	if mtype == "" {
		switch ext {
		case ".png":
			mtype = "image/png"
		case ".jpg", ".jpeg":
			mtype = "image/jpeg"
		case ".webp":
			mtype = "image/webp"
		default:
			mtype = "application/octet-stream"
		}
	}

	return "data:" + mtype + ";base64," + base64.StdEncoding.EncodeToString(b)
}

func escapeHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return r.Replace(s)
}

func makeSkinPDF(htmlPath, pdfPath string) error {
	browserPath, err := findHeadlessBrowser()
	if err != nil {
		return err
	}

	fileURL, err := toFileURL(htmlPath)
	if err != nil {
		return err
	}

	absPDF, err := filepath.Abs(pdfPath)
	if err != nil {
		return err
	}

	_ = os.Remove(absPDF)

	cmd := exec.Command(
		browserPath,
		"--headless",
		"--disable-gpu",
		"--no-pdf-header-footer",
		"--virtual-time-budget=1500",
		"--print-to-pdf="+filepath.ToSlash(absPDF),
		fileURL,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pdf render failed: %v / output=%s", err, string(out))
	}

	return waitForFile(absPDF, 5*time.Second)
}

func makeSkinPNGByScreenshot(htmlPath, pngPath string, dpi int) error {
	if dpi <= 0 {
		dpi = 300
	}

	browserPath, err := findHeadlessBrowser()
	if err != nil {
		return err
	}

	fileURL, err := toFileURL(htmlPath)
	if err != nil {
		return err
	}

	viewportWidth, viewportHeight := a4ViewportSizeCSSPx()
	deviceScaleFactor := float64(dpi) / 96.0

	absPNG, err := filepath.Abs(pngPath)
	if err != nil {
		return err
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
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("png render failed: %v / output=%s", err, string(out))
	}

	if err := waitForFile(absPNG, 5*time.Second); err != nil {
		return err
	}

	targetW, targetH := a4TargetPixelSize300DPI()
	if err := normalizePNGSize(absPNG, targetW, targetH); err != nil {
		return fmt.Errorf("png size normalize failed: %w", err)
	}

	return nil
}

func a4ViewportSizeCSSPx() (int, int) {
	cssDPI := 96.0
	width := int(float64(8.2677165)*cssDPI + 0.5)
	height := int(float64(11.692913)*cssDPI + 0.5)
	return width, height
}

func a4TargetPixelSize300DPI() (int, int) {
	return 2480, 3508
}

func findHeadlessBrowser() (string, error) {
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

func waitForFile(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		info, err := os.Stat(path)
		if err == nil && info.Size() > 0 {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("파일이 생성되지 않았습니다: %s", path)
}

func normalizePNGSize(path string, targetW, targetH int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return err
	}

	b := img.Bounds()
	srcW := b.Dx()
	srcH := b.Dy()

	if srcW == targetW && srcH == targetH {
		return nil
	}

	if srcW < targetW || srcH < targetH {
		return fmt.Errorf("captured png is smaller than target: got=%dx%d target=%dx%d", srcW, srcH, targetW, targetH)
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.Draw(dst, dst.Bounds(), img, image.Point{X: b.Min.X, Y: b.Min.Y}, draw.Src)

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	return png.Encode(out, dst)
}

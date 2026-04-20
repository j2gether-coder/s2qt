package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	xdraw "golang.org/x/image/draw"

	_ "image/jpeg"

	"s2qt/util"
)

const (
	TemplateCategoryAll        = "all"
	TemplateCategoryMonthly    = "monthly"
	TemplateCategorySeasonal   = "seasonal"
	TemplateCategoryLiturgical = "liturgical"

	templateSettingEnabledKey  = "template.enabled"
	templateSettingSelectedKey = "template.selected_id"
	templateSettingCategoryKey = "template.selected_category"
)

type TemplateSettings struct {
	Enabled          bool
	SelectedID       string
	SelectedCategory string
}

type TemplatePlacement struct {
	ForegroundLeftPX   int
	ForegroundTopPX    int
	ForegroundWidthPX  int
	ForegroundHeightPX int
	FitMode            string
	AlignX             string
	AlignY             string
	DebugRect          bool
}

type TemplateManifest struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Category           string `json:"category"`
	PreviewImage       string `json:"preview_image"`
	BackgroundImage    string `json:"background_image"`
	PDFBackgroundImage string `json:"pdf_background_image"`
	PNGBackgroundImage string `json:"png_background_image"`
	SkinImagePath      string `json:"skin_image_path"`

	ForegroundLeftPX   int    `json:"foreground_left_px"`
	ForegroundTopPX    int    `json:"foreground_top_px"`
	ForegroundWidthPX  int    `json:"foreground_width_px"`
	ForegroundHeightPX int    `json:"foreground_height_px"`
	FitMode            string `json:"fit_mode"`
	AlignX             string `json:"align_x"`
	AlignY             string `json:"align_y"`
	DebugRect          bool   `json:"debug_rect"`
}

type TemplateItem struct {
	ID                  string
	Name                string
	Category            string
	Dir                 string
	PreviewImagePath    string
	BackgroundImagePath string
	PDFBackgroundPath   string
	PNGBackgroundPath   string
	PNGPlacement        TemplatePlacement
}

type TemplateApplyRequest struct {
	ApplyPDF       bool
	ApplyPNG       bool
	DPI            int
	FooterOverride *QTFooterConfig
}

type TemplateApplyResult struct {
	Enabled    bool
	TemplateID string
	PDFApplied bool
	PNGApplied bool
	PDFError   string
	PNGError   string
}

func (r *TemplateApplyResult) HasError() bool {
	if r == nil {
		return false
	}
	return strings.TrimSpace(r.PDFError) != "" || strings.TrimSpace(r.PNGError) != ""
}

type TemplateService struct {
	Paths *util.AppPaths
	DB    *sql.DB
}

func NewTemplateService() (*TemplateService, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}
	return &TemplateService{Paths: paths}, nil
}

func NewTemplateServiceWithDB(db *sql.DB) (*TemplateService, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return &TemplateService{
		Paths: paths,
		DB:    db,
	}, nil
}

func (s *TemplateService) ApplySelectedTemplate(req *TemplateApplyRequest) (*TemplateApplyResult, error) {
	if req == nil {
		return nil, fmt.Errorf("template apply request가 비어 있습니다")
	}
	if s.Paths == nil {
		return nil, fmt.Errorf("paths가 nil 입니다")
	}

	settings, err := s.LoadTemplateSettings()
	if err != nil {
		return nil, err
	}

	result := &TemplateApplyResult{
		Enabled:    settings.Enabled,
		TemplateID: strings.TrimSpace(settings.SelectedID),
	}

	if !settings.Enabled {
		return result, nil
	}
	if strings.TrimSpace(settings.SelectedID) == "" {
		return nil, fmt.Errorf("템플릿 사용이 켜져 있지만 선택된 템플릿 ID가 없습니다")
	}

	item, err := s.ResolveSelectedTemplate()
	if err != nil {
		return nil, err
	}

	if req.DPI <= 0 {
		req.DPI = 300
	}

	if req.ApplyPDF {
		if err := s.ApplyTemplateToPDF(item, req.FooterOverride); err != nil {
			result.PDFError = err.Error()
		} else {
			result.PDFApplied = true
		}
	}

	if req.ApplyPNG {
		if err := s.ApplyTemplateToPNG(item); err != nil {
			result.PNGError = err.Error()
		} else {
			result.PNGApplied = true
		}
	}

	if result.HasError() {
		return result, fmt.Errorf("템플릿 적용 중 일부 실패가 발생했습니다")
	}
	return result, nil
}

func (s *TemplateService) LoadTemplateSettings() (*TemplateSettings, error) {
	if s.Paths == nil {
		return nil, fmt.Errorf("paths가 nil 입니다")
	}
	if s.DB == nil {
		return nil, fmt.Errorf("template db is nil")
	}

	keys := []string{
		templateSettingEnabledKey,
		templateSettingSelectedKey,
		templateSettingCategoryKey,
	}

	placeholders := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys))
	for _, k := range keys {
		placeholders = append(placeholders, "?")
		args = append(args, k)
	}

	query := `
SELECT setting_key, setting_value
FROM app_settings
WHERE setting_key IN (` + strings.Join(placeholders, ",") + `)
`

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("template settings 조회 실패: %w", err)
	}
	defer rows.Close()

	settings := &TemplateSettings{SelectedCategory: TemplateCategoryAll}

	for rows.Next() {
		var key string
		var value sql.NullString
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("template settings scan 실패: %w", err)
		}

		v := ""
		if value.Valid {
			v = strings.TrimSpace(value.String)
		}

		switch key {
		case templateSettingEnabledKey:
			settings.Enabled = parseBoolText(v)
		case templateSettingSelectedKey:
			settings.SelectedID = v
		case templateSettingCategoryKey:
			settings.SelectedCategory = normalizeTemplateCategory(v)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("template settings 읽기 실패: %w", err)
	}

	return settings, nil
}

func (s *TemplateService) ResolveSelectedTemplate() (*TemplateItem, error) {
	settings, err := s.LoadTemplateSettings()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(settings.SelectedID) == "" {
		return nil, fmt.Errorf("선택된 템플릿 ID가 없습니다")
	}
	return s.GetTemplateByID(settings.SelectedID)
}

func (s *TemplateService) GetTemplateByID(templateID string) (*TemplateItem, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return nil, fmt.Errorf("template id가 비어 있습니다")
	}

	dir := filepath.Join(s.templateRootDir(), templateID)
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("template directory를 찾을 수 없습니다: %s", dir)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("template path가 디렉토리가 아닙니다: %s", dir)
	}

	item, err := s.loadTemplateItemFromDir(dir)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("template item을 읽지 못했습니다: %s", templateID)
	}
	return item, nil
}

func (s *TemplateService) ListTemplates(category string) ([]TemplateItem, error) {
	root := s.templateRootDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []TemplateItem{}, nil
		}
		return nil, fmt.Errorf("template directory 목록 조회 실패: %w", err)
	}

	category = normalizeTemplateCategory(category)
	items := make([]TemplateItem, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		item, err := s.loadTemplateItemFromDir(filepath.Join(root, entry.Name()))
		if err != nil || item == nil {
			continue
		}

		if category != TemplateCategoryAll && item.Category != category {
			continue
		}
		items = append(items, *item)
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func (s *TemplateService) ApplyTemplateToPDF(item *TemplateItem, footerOverride *QTFooterConfig) error {
	if item == nil {
		return fmt.Errorf("template item이 nil 입니다")
	}

	html, err := s.readBaseHTML()
	if err != nil {
		return err
	}

	pdfSvc, err := NewPDFService()
	if err != nil {
		return err
	}

	wrapped, err := s.wrapHTMLForTemplatePDF(html, item, footerOverride)
	if err != nil {
		return err
	}

	sourcePath := buildPDFSourcePath(s.Paths.TempHtml)
	outPath := s.buildTemplatedPDFPath()

	if err := EnsureParentDir(sourcePath); err != nil {
		return err
	}
	if err := os.WriteFile(sourcePath, []byte(wrapped), 0o644); err != nil {
		return fmt.Errorf("template pdf source html 저장 실패: %w", err)
	}
	defer os.Remove(sourcePath)
	defer os.Remove(outPath)

	if err := pdfSvc.makePDFWithEdge(sourcePath, outPath); err != nil {
		return err
	}

	if err := templateReplaceOutputFile(outPath, s.Paths.TempPdf); err != nil {
		return fmt.Errorf("templated pdf 교체 실패: %w", err)
	}
	return nil
}

func (s *TemplateService) ApplyTemplateToPNG(item *TemplateItem) error {
	if item == nil {
		return fmt.Errorf("template item이 nil 입니다")
	}
	if s.Paths == nil {
		return fmt.Errorf("paths가 nil 입니다")
	}
	if !FileExists(s.Paths.TempPng) {
		return fmt.Errorf("temp.png가 없습니다: %s", s.Paths.TempPng)
	}

	templatePath := item.templateBackgroundPathForPNG()
	if strings.TrimSpace(templatePath) == "" {
		return fmt.Errorf("png template image가 없습니다")
	}

	outPath := s.buildTemplatedPNGPath()
	defer os.Remove(outPath)

	if err := templateComposePNGToPath(templatePath, s.Paths.TempPng, outPath, item.PNGPlacement); err != nil {
		return err
	}

	if err := templateReplaceOutputFile(outPath, s.Paths.TempPng); err != nil {
		return fmt.Errorf("templated png 교체 실패: %w", err)
	}
	return nil
}

func (s *TemplateService) wrapHTMLForTemplatePDF(content string, item *TemplateItem, footerOverride *QTFooterConfig) (string, error) {
	cleaned := normalizeHTMLFragment(content)
	resolvedFooter, err := s.resolveTemplateFooterConfig(footerOverride)
	if err != nil {
		return "", err
	}

	pdfStyle := loadQTPDFStyle()
	pdfStyle = mergeQTFooterRuntimeStyle(pdfStyle, resolvedFooter)
	pdfStyle += "\n\n" + buildTemplateRuntimeStyle(item.templateBackgroundPathForPDF())

	layoutBody := buildQTTemplateLayerLayout(cleaned, item.templateBackgroundPathForPDF(), resolvedFooter)
	return wrapHTMLDocumentForPDF("S2QT PDF Template", pdfStyle, layoutBody), nil
}

func (s *TemplateService) resolveTemplateFooterConfig(footerOverride *QTFooterConfig) (*QTFooterConfig, error) {
	qrSvc, err := NewQRService()
	if err != nil {
		return nil, err
	}

	mode := QTFooterModeDefault
	if footerOverride != nil && footerOverride.Mode != "" {
		mode = footerOverride.Mode
	}

	return qrSvc.PrepareFooterAssets(mode, footerOverride)
}

func buildQTTemplateLayerLayout(bodyHTML, backgroundImagePath string, footerCfg *QTFooterConfig) string {
	bodyHTML = strings.TrimSpace(bodyHTML)
	bgDataURI := EncodeImageAsDataURI(backgroundImagePath)

	if bgDataURI == "" {
		return buildQTFixedPageLayout(bodyHTML, footerCfg)
	}

	return `
<div class="qt-template-layer" aria-hidden="true">
  <img class="qt-template-bg" src="` + bgDataURI + `" alt="template background" />
</div>
<div class="qt-template-content">` + buildQTFixedPageLayout(bodyHTML, footerCfg) + `</div>`
}

func buildTemplateRuntimeStyle(backgroundImagePath string) string {
	if strings.TrimSpace(backgroundImagePath) == "" {
		return ""
	}

	return `
body{
  background: transparent !important;
}

.qt-template-layer{
  position: fixed !important;
  left: 0 !important;
  top: 0 !important;
  width: 210mm !important;
  height: 297mm !important;
  z-index: 0 !important;
  pointer-events: none !important;
}

.qt-template-bg{
  width: 210mm !important;
  height: 297mm !important;
  display: block !important;
  object-fit: fill !important;
}

.qt-template-content{
  position: relative !important;
  z-index: 10 !important;
}

.qt-wrap{
  background: transparent !important;
}
`
}

func (s *TemplateService) readBaseHTML() (string, error) {
	b, err := os.ReadFile(s.Paths.TempHtml)
	if err != nil {
		return "", fmt.Errorf("temp.html 읽기 실패: %w", err)
	}
	cleaned := normalizeHTMLFragment(string(b))
	if cleaned == "" {
		return "", fmt.Errorf("temp.html 내용이 비어 있습니다")
	}
	return cleaned, nil
}

func (s *TemplateService) templateRootDir() string {
	tempDir := filepath.Dir(s.Paths.TempHtml)
	varDir := filepath.Dir(tempDir)
	return filepath.Join(varDir, "template")
}

func (s *TemplateService) buildTemplatedPDFPath() string {
	dir := filepath.Dir(s.Paths.TempPdf)
	return filepath.Join(dir, "temp_templated.pdf")
}

func (s *TemplateService) buildTemplatedPNGPath() string {
	dir := filepath.Dir(s.Paths.TempPng)
	return filepath.Join(dir, "temp_templated.png")
}

func (s *TemplateService) loadTemplateItemFromDir(dir string) (*TemplateItem, error) {
	manifest, _ := loadTemplateManifest(dir)

	item := &TemplateItem{
		ID:           filepath.Base(dir),
		Name:         filepath.Base(dir),
		Category:     TemplateCategoryAll,
		Dir:          dir,
		PNGPlacement: defaultTemplatePlacement(),
	}

	if manifest != nil {
		if strings.TrimSpace(manifest.ID) != "" {
			item.ID = strings.TrimSpace(manifest.ID)
		}
		if strings.TrimSpace(manifest.Name) != "" {
			item.Name = strings.TrimSpace(manifest.Name)
		}
		item.Category = normalizeTemplateCategory(manifest.Category)
		item.PreviewImagePath = resolveTemplateAssetPath(dir, manifest.PreviewImage)
		item.BackgroundImagePath = resolveTemplateAssetPath(dir, manifest.BackgroundImage)
		item.PDFBackgroundPath = resolveTemplateAssetPath(dir, manifest.PDFBackgroundImage)
		item.PNGBackgroundPath = resolveTemplateAssetPath(dir, manifest.PNGBackgroundImage)
		if item.PNGBackgroundPath == "" {
			item.PNGBackgroundPath = resolveTemplateAssetPath(dir, manifest.SkinImagePath)
		}
		item.PNGPlacement = normalizeTemplatePlacement(TemplatePlacement{
			ForegroundLeftPX:   manifest.ForegroundLeftPX,
			ForegroundTopPX:    manifest.ForegroundTopPX,
			ForegroundWidthPX:  manifest.ForegroundWidthPX,
			ForegroundHeightPX: manifest.ForegroundHeightPX,
			FitMode:            manifest.FitMode,
			AlignX:             manifest.AlignX,
			AlignY:             manifest.AlignY,
			DebugRect:          manifest.DebugRect,
		})
	}

	if item.PreviewImagePath == "" {
		item.PreviewImagePath = findFirstExistingFile(dir, []string{
			"preview.png", "preview.jpg", "preview.jpeg", "preview.webp",
		})
	}
	if item.BackgroundImagePath == "" {
		item.BackgroundImagePath = findFirstExistingFile(dir, []string{
			"template.png", "template.jpg", "template.jpeg", "template.webp",
			"background.png", "background.jpg", "background.jpeg", "background.webp",
			"bg.png", "bg.jpg", "bg.jpeg", "bg.webp",
			"skin.png", "skin.jpg", "skin.jpeg", "skin.webp",
			"frame.png", "frame.jpg", "frame.jpeg", "frame.webp",
		})
	}
	if item.PDFBackgroundPath == "" {
		item.PDFBackgroundPath = findFirstExistingFile(dir, []string{
			"pdf.png", "pdf.jpg", "pdf.jpeg", "pdf.webp",
			"pdf_background.png", "pdf_background.jpg", "pdf_background.jpeg", "pdf_background.webp",
		})
	}
	if item.PNGBackgroundPath == "" {
		item.PNGBackgroundPath = findFirstExistingFile(dir, []string{
			"png.png", "png.jpg", "png.jpeg", "png.webp",
			"png_background.png", "png_background.jpg", "png_background.jpeg", "png_background.webp",
			"skin.png", "skin.jpg", "skin.jpeg", "skin.webp",
			"template.png", "template.jpg", "template.jpeg", "template.webp",
		})
	}

	item.PNGPlacement = normalizeTemplatePlacement(item.PNGPlacement)
	return item, nil
}

func loadTemplateManifest(dir string) (*TemplateManifest, error) {
	manifestPath := filepath.Join(dir, "manifest.json")
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest TemplateManifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		return nil, fmt.Errorf("manifest parse 실패 (%s): %w", manifestPath, err)
	}
	return &manifest, nil
}

func resolveTemplateAssetPath(dir, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		if FileExists(value) {
			return value
		}
		return ""
	}
	path := filepath.Clean(filepath.Join(dir, value))
	if FileExists(path) {
		return path
	}
	return ""
}

func findFirstExistingFile(dir string, names []string) string {
	for _, name := range names {
		path := filepath.Join(dir, name)
		if FileExists(path) {
			return path
		}
	}
	return ""
}

func normalizeTemplateCategory(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", TemplateCategoryAll:
		return TemplateCategoryAll
	case TemplateCategoryMonthly:
		return TemplateCategoryMonthly
	case TemplateCategorySeasonal:
		return TemplateCategorySeasonal
	case TemplateCategoryLiturgical:
		return TemplateCategoryLiturgical
	default:
		return TemplateCategoryAll
	}
}

func defaultTemplatePlacement() TemplatePlacement {
	return TemplatePlacement{
		FitMode: "contain",
		AlignX:  "center",
		AlignY:  "top",
	}
}

func normalizeTemplatePlacement(p TemplatePlacement) TemplatePlacement {
	if p.ForegroundLeftPX < 0 {
		p.ForegroundLeftPX = 0
	}
	if p.ForegroundTopPX < 0 {
		p.ForegroundTopPX = 0
	}

	p.FitMode = strings.ToLower(strings.TrimSpace(p.FitMode))
	if p.FitMode != "cover" {
		p.FitMode = "contain"
	}

	p.AlignX = strings.ToLower(strings.TrimSpace(p.AlignX))
	switch p.AlignX {
	case "left", "center", "right":
	default:
		p.AlignX = "center"
	}

	p.AlignY = strings.ToLower(strings.TrimSpace(p.AlignY))
	switch p.AlignY {
	case "top", "center", "bottom":
	default:
		p.AlignY = "top"
	}

	return p
}

func (t *TemplateItem) templateBackgroundPathForPDF() string {
	if t == nil {
		return ""
	}
	if strings.TrimSpace(t.PDFBackgroundPath) != "" {
		return t.PDFBackgroundPath
	}
	return t.BackgroundImagePath
}

func (t *TemplateItem) templateBackgroundPathForPNG() string {
	if t == nil {
		return ""
	}
	if strings.TrimSpace(t.PNGBackgroundPath) != "" {
		return t.PNGBackgroundPath
	}
	return t.BackgroundImagePath
}

func templateComposePNGToPath(templatePath, foregroundPath, outputPath string, placement TemplatePlacement) error {
	bgImg, err := templateDecodeImageFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to load template image: %w", err)
	}

	fgImg, err := templateDecodeImageFile(foregroundPath)
	if err != nil {
		return fmt.Errorf("failed to load foreground image: %w", err)
	}

	fgBounds := fgImg.Bounds()
	fgW := fgBounds.Dx()
	fgH := fgBounds.Dy()
	if fgW <= 0 || fgH <= 0 {
		return fmt.Errorf("invalid foreground image size: %dx%d", fgW, fgH)
	}

	canvas := image.NewRGBA(image.Rect(0, 0, fgW, fgH))
	xdraw.CatmullRom.Scale(canvas, canvas.Bounds(), bgImg, bgImg.Bounds(), xdraw.Src, nil)

	targetRect := templateBuildTargetRect(canvas.Bounds(), placement)
	if targetRect.Empty() {
		return fmt.Errorf("foreground rect is empty or out of bounds: %v", targetRect)
	}

	dstRect, srcRect := templateCalcPlacement(fgW, fgH, targetRect, placement.FitMode, placement.AlignX, placement.AlignY)
	xdraw.CatmullRom.Scale(canvas, dstRect, fgImg, srcRect, xdraw.Over, nil)

	if placement.DebugRect {
		templateDrawRectBorder(canvas, targetRect, color.RGBA{R: 220, G: 38, B: 38, A: 255}, 3)
	}

	if err := EnsureParentDir(outputPath); err != nil {
		return err
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output png: %w", err)
	}
	defer out.Close()

	if err := png.Encode(out, canvas); err != nil {
		return fmt.Errorf("failed to encode output png: %w", err)
	}
	return nil
}

func templateBuildTargetRect(canvasBounds image.Rectangle, placement TemplatePlacement) image.Rectangle {
	if placement.ForegroundWidthPX <= 0 || placement.ForegroundHeightPX <= 0 {
		return canvasBounds
	}

	rect := image.Rect(
		placement.ForegroundLeftPX,
		placement.ForegroundTopPX,
		placement.ForegroundLeftPX+placement.ForegroundWidthPX,
		placement.ForegroundTopPX+placement.ForegroundHeightPX,
	)
	return rect.Intersect(canvasBounds)
}

func templateCalcPlacement(fgW, fgH int, target image.Rectangle, fitMode, alignX, alignY string) (image.Rectangle, image.Rectangle) {
	tw := target.Dx()
	th := target.Dy()
	srcRect := image.Rect(0, 0, fgW, fgH)

	if fitMode == "cover" {
		scale := math.Max(float64(tw)/float64(fgW), float64(th)/float64(fgH))
		scaledW := int(math.Round(float64(fgW) * scale))
		scaledH := int(math.Round(float64(fgH) * scale))

		dstX := templateCalcAlignedOffset(target.Min.X, tw, scaledW, alignX)
		dstY := templateCalcAlignedOffset(target.Min.Y, th, scaledH, alignY)
		return image.Rect(dstX, dstY, dstX+scaledW, dstY+scaledH), srcRect
	}

	scale := math.Min(float64(tw)/float64(fgW), float64(th)/float64(fgH))
	scaledW := int(math.Round(float64(fgW) * scale))
	scaledH := int(math.Round(float64(fgH) * scale))

	dstX := templateCalcAlignedOffset(target.Min.X, tw, scaledW, alignX)
	dstY := templateCalcAlignedOffset(target.Min.Y, th, scaledH, alignY)
	return image.Rect(dstX, dstY, dstX+scaledW, dstY+scaledH), srcRect
}

func templateCalcAlignedOffset(start, outer, inner int, align string) int {
	switch align {
	case "left", "top":
		return start
	case "right", "bottom":
		return start + (outer - inner)
	default:
		return start + (outer-inner)/2
	}
}

func templateDecodeImageFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

func templateDrawRectBorder(img *image.RGBA, rect image.Rectangle, c color.Color, thickness int) {
	if thickness <= 0 {
		thickness = 1
	}
	templateFillRect(img, image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+thickness), c)
	templateFillRect(img, image.Rect(rect.Min.X, rect.Max.Y-thickness, rect.Max.X, rect.Max.Y), c)
	templateFillRect(img, image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+thickness, rect.Max.Y), c)
	templateFillRect(img, image.Rect(rect.Max.X-thickness, rect.Min.Y, rect.Max.X, rect.Max.Y), c)
}

func templateFillRect(img *image.RGBA, rect image.Rectangle, c color.Color) {
	r := rect.Intersect(img.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.Set(x, y, c)
		}
	}
}

func templateReplaceOutputFile(src, dst string) error {
	if strings.TrimSpace(src) == "" || strings.TrimSpace(dst) == "" {
		return fmt.Errorf("src 또는 dst 경로가 비어 있습니다")
	}
	if !FileExists(src) {
		return fmt.Errorf("교체할 원본 파일이 없습니다: %s", src)
	}

	_ = os.Remove(dst)
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	return nil
}

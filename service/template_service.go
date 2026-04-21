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
	_ "image/png"

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

	templateThumbFileName = "thumb.png"
)

type TemplateSettings struct {
	Enabled          bool   `json:"enabled"`
	SelectedCategory string `json:"selectedCategory"`
	SelectedID       string `json:"selectedId"`
}

type TemplateSettingsSaveRequest struct {
	Enabled          bool   `json:"enabled"`
	SelectedCategory string `json:"selectedCategory"`
	SelectedID       string `json:"selectedId"`
}

type TemplatePlacement struct {
	LeftPX    int    `json:"left_px"`
	TopPX     int    `json:"top_px"`
	WidthPX   int    `json:"width_px"`
	HeightPX  int    `json:"height_px"`
	FitMode   string `json:"fit_mode"`
	AlignX    string `json:"align_x"`
	AlignY    string `json:"align_y"`
	DebugRect bool   `json:"debug_rect,omitempty"`
}

type TemplateManifest struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Enabled  *bool  `json:"enabled,omitempty"`

	// 권장: 단일 공통 자산
	TemplateImage string `json:"template_image,omitempty"`

	// 구버전/확장 호환 fallback
	PreviewImage       string `json:"preview_image,omitempty"`
	BackgroundImage    string `json:"background_image,omitempty"`
	PDFBackgroundImage string `json:"pdf_background_image,omitempty"`
	PNGBackgroundImage string `json:"png_background_image,omitempty"`
	SkinImagePath      string `json:"skin_image_path,omitempty"`

	PNGPlacement TemplatePlacement `json:"png_placement,omitempty"`
}

type TemplateListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	PreviewPath string `json:"previewPath"`

	HasPDFAsset bool `json:"hasPdfAsset"`
	HasPNGAsset bool `json:"hasPngAsset"`
	IsValid     bool `json:"isValid"`
}

type TemplateItem struct {
	ID           string
	Name         string
	Category     string
	Dir          string
	PreviewPath  string
	ThumbPath    string
	CommonPath   string
	PDFPath      string
	PNGPath      string
	PNGPlacement TemplatePlacement
}

type TemplateApplyRequest struct {
	ApplyPDF       bool
	ApplyPNG       bool
	DPI            int
	FooterOverride *QTFooterConfig
}

type TemplateApplyResult struct {
	Enabled    bool   `json:"enabled"`
	TemplateID string `json:"templateId"`
	PDFApplied bool   `json:"pdfApplied"`
	PNGApplied bool   `json:"pngApplied"`
	PDFError   string `json:"pdfError"`
	PNGError   string `json:"pngError"`
}

func (r *TemplateApplyResult) HasError() bool {
	if r == nil {
		return false
	}
	return strings.TrimSpace(r.PDFError) != "" || strings.TrimSpace(r.PNGError) != ""
}

type TemplateService struct {
	DB    *sql.DB
	Paths *util.AppPaths
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
		DB:    db,
		Paths: paths,
	}, nil
}

func (s *TemplateService) LoadTemplateSettings() (*TemplateSettings, error) {
	if s == nil || s.DB == nil {
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

	result := &TemplateSettings{
		Enabled:          false,
		SelectedCategory: TemplateCategoryAll,
		SelectedID:       "",
	}

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
			result.Enabled = parseBoolText(v)
		case templateSettingSelectedKey:
			result.SelectedID = strings.TrimSpace(v)
		case templateSettingCategoryKey:
			result.SelectedCategory = normalizeTemplateCategory(v)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("template settings rows 읽기 실패: %w", err)
	}

	return normalizeTemplateSettings(result), nil
}

func (s *TemplateService) SaveTemplateSettings(req TemplateSettingsSaveRequest) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("template db is nil")
	}

	normalized := normalizeTemplateSettings(&TemplateSettings{
		Enabled:          req.Enabled,
		SelectedCategory: req.SelectedCategory,
		SelectedID:       req.SelectedID,
	})

	if err := s.upsertTemplateSetting(templateSettingEnabledKey, templateBoolToText(normalized.Enabled), "boolean"); err != nil {
		return err
	}
	if err := s.upsertTemplateSetting(templateSettingCategoryKey, normalized.SelectedCategory, "text"); err != nil {
		return err
	}
	if err := s.upsertTemplateSetting(templateSettingSelectedKey, normalized.SelectedID, "text"); err != nil {
		return err
	}

	return nil
}

func (s *TemplateService) upsertTemplateSetting(key, value, valueType string) error {
	_, err := s.DB.Exec(`
INSERT INTO app_settings (setting_key, setting_value, value_type, is_secret, setting_group, updated_at)
VALUES (?, ?, ?, 0, 'template', datetime('now', 'localtime'))
ON CONFLICT(setting_key) DO UPDATE SET
	setting_value = excluded.setting_value,
	value_type = excluded.value_type,
	updated_at = excluded.updated_at
`, key, value, valueType)
	if err != nil {
		return fmt.Errorf("template app_setting 저장 실패 (%s): %w", key, err)
	}
	return nil
}

// ListTemplates는 절대 이미지 디코드/썸네일 생성 없이,
// var/template 직하 디렉터리와 manifest만 읽습니다.
func (s *TemplateService) ListTemplates() ([]TemplateListItem, error) {
	if s == nil || s.Paths == nil {
		return []TemplateListItem{}, nil
	}

	root := s.resolveTemplateRootDir()
	if strings.TrimSpace(root) == "" {
		return []TemplateListItem{}, nil
	}

	dirs, err := s.scanTemplateDirectories(root)
	if err != nil {
		return nil, err
	}

	items := make([]TemplateListItem, 0, len(dirs))
	for _, dir := range dirs {
		manifest, err := s.loadTemplateManifest(dir)
		if err != nil {
			continue
		}

		item, err := s.buildTemplateListItem(dir, manifest)
		if err != nil || item == nil {
			continue
		}
		items = append(items, *item)
	}

	sortTemplateListItems(items)
	return items, nil
}

func (s *TemplateService) ResolveSelectedTemplate() (*TemplateItem, error) {
	settings, err := s.LoadTemplateSettings()
	if err != nil {
		return nil, err
	}

	selectedID := strings.TrimSpace(settings.SelectedID)
	if selectedID == "" {
		return nil, fmt.Errorf("선택된 템플릿 ID가 없습니다")
	}

	return s.GetTemplateByID(selectedID)
}

func (s *TemplateService) GetTemplateByID(templateID string) (*TemplateItem, error) {
	if s == nil || s.Paths == nil {
		return nil, fmt.Errorf("template paths가 nil 입니다")
	}

	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return nil, fmt.Errorf("template id가 비어 있습니다")
	}

	root := s.resolveTemplateRootDir()
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("template root가 비어 있습니다")
	}

	dir := filepath.Join(root, templateID)
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("template directory를 찾을 수 없습니다: %s", dir)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("template path가 디렉토리가 아닙니다: %s", dir)
	}

	return s.loadTemplateItemFromDir(dir)
}

func (s *TemplateService) EnsureTemplateThumbnail(item *TemplateItem) (string, error) {
	if item == nil {
		return "", fmt.Errorf("template item이 nil 입니다")
	}

	srcPath := strings.TrimSpace(item.CommonPath)
	if srcPath == "" {
		srcPath = strings.TrimSpace(item.templateBackgroundPathForPNG())
	}
	if srcPath == "" {
		srcPath = strings.TrimSpace(item.templateBackgroundPathForPDF())
	}
	if srcPath == "" {
		return "", fmt.Errorf("template 원본 이미지가 없습니다")
	}
	if !FileExists(srcPath) {
		return "", fmt.Errorf("template 원본 파일이 없습니다: %s", srcPath)
	}

	dstPath := s.buildTemplateThumbPath(item.Dir)
	if FileExists(dstPath) {
		return dstPath, nil
	}

	if err := s.generateTemplateThumbnail(srcPath, dstPath, 360, 480); err != nil {
		return "", err
	}
	return dstPath, nil
}

func (s *TemplateService) buildTemplateThumbPath(dir string) string {
	return filepath.Join(dir, "thumb.png")
}

func (s *TemplateService) generateTemplateThumbnail(srcPath, dstPath string, maxWidth, maxHeight int) error {
	if maxWidth <= 0 {
		maxWidth = 360
	}
	if maxHeight <= 0 {
		maxHeight = 480
	}

	srcImg, err := templateDecodeImageFile(srcPath)
	if err != nil {
		return fmt.Errorf("template 이미지 디코드 실패: %w", err)
	}

	b := srcImg.Bounds()
	srcW := b.Dx()
	srcH := b.Dy()
	if srcW <= 0 || srcH <= 0 {
		return fmt.Errorf("유효하지 않은 template 이미지 크기: %dx%d", srcW, srcH)
	}

	scale := math.Min(float64(maxWidth)/float64(srcW), float64(maxHeight)/float64(srcH))
	if scale > 1.0 {
		scale = 1.0
	}

	dstW := int(math.Round(float64(srcW) * scale))
	dstH := int(math.Round(float64(srcH) * scale))
	if dstW <= 0 {
		dstW = 1
	}
	if dstH <= 0 {
		dstH = 1
	}

	dstImg := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	xdraw.CatmullRom.Scale(dstImg, dstImg.Bounds(), srcImg, b, xdraw.Src, nil)

	if err := EnsureParentDir(dstPath); err != nil {
		return err
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("thumb.png 생성 실패: %w", err)
	}
	defer f.Close()

	if err := png.Encode(f, dstImg); err != nil {
		return fmt.Errorf("thumb.png 저장 실패: %w", err)
	}

	return nil
}

func (s *TemplateService) ApplySelectedTemplate(req *TemplateApplyRequest) (*TemplateApplyResult, error) {
	if req == nil {
		return nil, fmt.Errorf("template apply request가 비어 있습니다")
	}
	if s == nil || s.Paths == nil {
		return nil, fmt.Errorf("template service가 초기화되지 않았습니다")
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
	if result.TemplateID == "" {
		return result, nil
	}

	item, err := s.ResolveSelectedTemplate()
	if err != nil {
		return nil, err
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

func (s *TemplateService) ApplyTemplateToPDF(item *TemplateItem, footerOverride *QTFooterConfig) error {
	if item == nil {
		return fmt.Errorf("template item이 nil 입니다")
	}
	if s.Paths == nil {
		return fmt.Errorf("template paths가 nil 입니다")
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

	return templateReplaceOutputFile(outPath, s.Paths.TempPdf)
}

func (s *TemplateService) ApplyTemplateToPNG(item *TemplateItem) error {
	if item == nil {
		return fmt.Errorf("template item이 nil 입니다")
	}
	if s.Paths == nil {
		return fmt.Errorf("template paths가 nil 입니다")
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

	return templateReplaceOutputFile(outPath, s.Paths.TempPng)
}

func normalizeTemplateSettings(v *TemplateSettings) *TemplateSettings {
	if v == nil {
		return &TemplateSettings{
			Enabled:          false,
			SelectedCategory: TemplateCategoryAll,
			SelectedID:       "",
		}
	}

	v.SelectedID = strings.TrimSpace(v.SelectedID)
	v.SelectedCategory = normalizeTemplateCategory(v.SelectedCategory)
	if v.SelectedCategory == "" {
		v.SelectedCategory = TemplateCategoryAll
	}
	return v
}

func templateBoolToText(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func (s *TemplateService) resolveTemplateRootDir() string {
	if s == nil || s.Paths == nil {
		return ""
	}
	return strings.TrimSpace(s.Paths.Template)
}

func (s *TemplateService) templateNoImagePath() string {
	if s == nil || s.Paths == nil {
		return ""
	}
	return strings.TrimSpace(s.Paths.TemplateNoImage)
}

// scanTemplateDirectories는 var/template 직하 1단계 디렉터리만 읽습니다.
func (s *TemplateService) scanTemplateDirectories(root string) ([]string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return []string{}, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("template directory 목록 조회 실패: %w", err)
	}

	dirs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, ".") {
			continue
		}

		dirs = append(dirs, filepath.Join(root, name))
	}

	sort.Strings(dirs)
	return dirs, nil
}

func (s *TemplateService) loadTemplateManifest(dir string) (*TemplateManifest, error) {
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

func (s *TemplateService) buildTemplateListItem(dir string, manifest *TemplateManifest) (*TemplateListItem, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest is nil")
	}

	if manifest.Enabled != nil && !*manifest.Enabled {
		return nil, fmt.Errorf("disabled template")
	}

	id := strings.TrimSpace(manifest.ID)
	if id == "" {
		id = filepath.Base(dir)
	}

	name := strings.TrimSpace(manifest.Name)
	if name == "" {
		name = id
	}

	category := normalizeTemplateCategory(manifest.Category)
	templatePath := resolveTemplateCommonImage(dir, manifest)

	return &TemplateListItem{
		ID:          id,
		Name:        name,
		Category:    category,
		PreviewPath: "",
		HasPDFAsset: strings.TrimSpace(templatePath) != "",
		HasPNGAsset: strings.TrimSpace(templatePath) != "",
		IsValid:     strings.TrimSpace(templatePath) != "",
	}, nil
}

func (s *TemplateService) GetTemplatePreview(templateID string) (string, error) {
	if s == nil || s.Paths == nil {
		return "", fmt.Errorf("template service가 초기화되지 않았습니다")
	}

	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		noImage := s.templateNoImagePath()
		if noImage != "" && FileExists(noImage) {
			return noImage, nil
		}
		return "", fmt.Errorf("template id가 비어 있습니다")
	}

	item, err := s.GetTemplateByID(templateID)
	if err != nil {
		noImage := s.templateNoImagePath()
		if noImage != "" && FileExists(noImage) {
			return noImage, nil
		}
		return "", err
	}

	thumbPath, err := s.EnsureTemplateThumbnail(item)
	if err == nil && strings.TrimSpace(thumbPath) != "" && FileExists(thumbPath) {
		return thumbPath, nil
	}

	noImage := s.templateNoImagePath()
	if noImage != "" && FileExists(noImage) {
		return noImage, nil
	}

	if item.CommonPath != "" && FileExists(item.CommonPath) {
		return item.CommonPath, nil
	}

	return "", fmt.Errorf("미리보기 이미지를 찾을 수 없습니다: %s", templateID)
}

func (s *TemplateService) loadTemplateItemFromDir(dir string) (*TemplateItem, error) {
	manifest, err := s.loadTemplateManifest(dir)
	if err != nil {
		return nil, err
	}

	if manifest.Enabled != nil && !*manifest.Enabled {
		return nil, fmt.Errorf("disabled template: %s", filepath.Base(dir))
	}

	item := &TemplateItem{
		ID:           filepath.Base(dir),
		Name:         filepath.Base(dir),
		Category:     TemplateCategoryAll,
		Dir:          dir,
		PNGPlacement: defaultTemplatePlacement(),
	}

	if strings.TrimSpace(manifest.ID) != "" {
		item.ID = strings.TrimSpace(manifest.ID)
	}
	if strings.TrimSpace(manifest.Name) != "" {
		item.Name = strings.TrimSpace(manifest.Name)
	}

	item.Category = normalizeTemplateCategory(manifest.Category)
	item.CommonPath = resolveTemplateCommonImage(dir, manifest)
	item.PDFPath = resolveTemplatePDFImage(dir, manifest)
	item.PNGPath = resolveTemplatePNGImage(dir, manifest)
	item.PNGPlacement = normalizeTemplatePlacement(manifest.PNGPlacement)

	if item.CommonPath == "" {
		item.CommonPath = findFirstExistingFile(dir, []string{
			"template.png", "template.jpg", "template.jpeg", "template.webp",
		})
	}

	if item.PDFPath == "" {
		item.PDFPath = item.CommonPath
	}
	if item.PNGPath == "" {
		item.PNGPath = item.CommonPath
	}

	return item, nil
}

func resolveTemplateThumbImage(dir string) string {
	path := filepath.Join(dir, templateThumbFileName)
	if FileExists(path) {
		return path
	}
	return ""
}

func resolveTemplateCommonImage(dir string, manifest *TemplateManifest) string {
	if manifest == nil {
		return ""
	}

	if p := resolveTemplateAssetPath(dir, manifest.TemplateImage); p != "" {
		return p
	}
	if p := resolveTemplateAssetPath(dir, manifest.BackgroundImage); p != "" {
		return p
	}
	if p := resolveTemplateAssetPath(dir, manifest.PDFBackgroundImage); p != "" {
		return p
	}
	if p := resolveTemplateAssetPath(dir, manifest.PNGBackgroundImage); p != "" {
		return p
	}
	if p := resolveTemplateAssetPath(dir, manifest.SkinImagePath); p != "" {
		return p
	}
	if p := resolveTemplateAssetPath(dir, manifest.PreviewImage); p != "" {
		return p
	}
	return ""
}

func resolveTemplatePreviewImage(dir string, manifest *TemplateManifest) string {
	if manifest == nil {
		return ""
	}
	if p := resolveTemplateThumbImage(dir); p != "" {
		return p
	}
	if p := resolveTemplateAssetPath(dir, manifest.PreviewImage); p != "" {
		return p
	}
	if p := resolveTemplateAssetPath(dir, manifest.TemplateImage); p != "" {
		return p
	}
	if p := resolveTemplateAssetPath(dir, manifest.BackgroundImage); p != "" {
		return p
	}
	return ""
}

func resolveTemplatePDFImage(dir string, manifest *TemplateManifest) string {
	if manifest == nil {
		return ""
	}
	if p := resolveTemplateAssetPath(dir, manifest.TemplateImage); p != "" {
		return p
	}
	if p := resolveTemplateAssetPath(dir, manifest.PDFBackgroundImage); p != "" {
		return p
	}
	if p := resolveTemplateAssetPath(dir, manifest.BackgroundImage); p != "" {
		return p
	}
	return ""
}

func resolveTemplatePNGImage(dir string, manifest *TemplateManifest) string {
	if manifest == nil {
		return ""
	}
	if p := resolveTemplateAssetPath(dir, manifest.TemplateImage); p != "" {
		return p
	}
	if p := resolveTemplateAssetPath(dir, manifest.PNGBackgroundImage); p != "" {
		return p
	}
	if p := resolveTemplateAssetPath(dir, manifest.SkinImagePath); p != "" {
		return p
	}
	if p := resolveTemplateAssetPath(dir, manifest.BackgroundImage); p != "" {
		return p
	}
	return ""
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

func sortTemplateListItems(items []TemplateListItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Category != items[j].Category {
			return items[i].Category < items[j].Category
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
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
	if p.LeftPX < 0 {
		p.LeftPX = 0
	}
	if p.TopPX < 0 {
		p.TopPX = 0
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
	if strings.TrimSpace(t.PDFPath) != "" {
		return t.PDFPath
	}
	return t.CommonPath
}

func (t *TemplateItem) templateBackgroundPathForPNG() string {
	if t == nil {
		return ""
	}
	if strings.TrimSpace(t.PNGPath) != "" {
		return t.PNGPath
	}
	return t.CommonPath
}

func (t *TemplateItem) sourcePathForThumbnail() string {
	if t == nil {
		return ""
	}
	if strings.TrimSpace(t.CommonPath) != "" {
		return t.CommonPath
	}
	if strings.TrimSpace(t.PDFPath) != "" {
		return t.PDFPath
	}
	if strings.TrimSpace(t.PNGPath) != "" {
		return t.PNGPath
	}
	return ""
}

func (t *TemplateItem) safePreviewPathForList(noImagePath string) string {
	if t == nil {
		return noImagePath
	}
	if strings.TrimSpace(t.ThumbPath) != "" {
		return t.ThumbPath
	}
	return noImagePath
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

func (s *TemplateService) buildTemplatedPDFPath() string {
	dir := filepath.Dir(s.Paths.TempPdf)
	return filepath.Join(dir, "temp_templated.pdf")
}

func (s *TemplateService) buildTemplatedPNGPath() string {
	dir := filepath.Dir(s.Paths.TempPng)
	return filepath.Join(dir, "temp_templated.png")
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
	if placement.WidthPX <= 0 || placement.HeightPX <= 0 {
		return canvasBounds
	}

	rect := image.Rect(
		placement.LeftPX,
		placement.TopPX,
		placement.LeftPX+placement.WidthPX,
		placement.TopPX+placement.HeightPX,
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

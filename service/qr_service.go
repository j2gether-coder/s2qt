package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	qrcode "github.com/skip2/go-qrcode"

	"s2qt/util"
)

type QRStyle string
type QTFooterMode string

const (
	QRStyleSquare QRStyle = "square"
	QRStyleDot    QRStyle = "dot"

	QTFooterModeDefault    QTFooterMode = "default"
	QTFooterModeSubscriber QTFooterMode = "subscriber"
	QTFooterModeCustom     QTFooterMode = "custom"
)

type FooterPolicy struct {
	AlwaysRegenerateQR  bool `json:"always_regenerate_qr"`
	OverwriteExistingQR bool `json:"overwrite_existing_qr"`
}

type QTFooterConfig struct {
	Mode        QTFooterMode `json:"mode"`
	ShowFooter  bool         `json:"show_footer"`
	ShowDivider bool         `json:"show_divider"`
	ShowQR      bool         `json:"show_qr"`

	FooterText     string `json:"footer_text,omitempty"`
	ChurchName     string `json:"church_name,omitempty"`
	LogoPath       string `json:"logo_path,omitempty"`
	BrandImagePath string `json:"brand_image_path,omitempty"`
	HomepageURL    string `json:"homepage_url,omitempty"`
	QRImagePath    string `json:"qr_image_path,omitempty"`

	QRPosition string  `json:"qr_position,omitempty"`
	QRSizeMM   float64 `json:"qr_size_mm,omitempty"`
	SafeAreaMM float64 `json:"safe_area_mm,omitempty"`
}

type FooterConfigFile struct {
	Version          string          `json:"version"`
	FooterPolicy     FooterPolicy    `json:"footer_policy"`
	DefaultFooter    QTFooterConfig  `json:"default_footer"`
	QRDefaults       QRRenderOptions `json:"qr_defaults"`
	SubscriberFooter QTFooterConfig  `json:"subscriber_footer"`
}

type QRRenderOptions struct {
	SizePx         int      `json:"size_px"`
	MarginModules  int      `json:"margin_modules"`
	Style          QRStyle  `json:"style"`
	DotScale       float64  `json:"dot_scale"`
	KeepFinderBox  bool     `json:"keep_finder_box"`
	BackgroundRGBA [4]uint8 `json:"background_rgba"`
	ForegroundRGBA [4]uint8 `json:"foreground_rgba"`
}

type QRGenerateResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Text     string `json:"text"`
	FilePath string `json:"filePath"`
	SizePx   int    `json:"sizePx"`
	Style    string `json:"style"`
}

type QRService struct {
	Paths *util.AppPaths
}

func NewQRService() (*QRService, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return &QRService{
		Paths: paths,
	}, nil
}

func DefaultFooterConfigFile() FooterConfigFile {
	return FooterConfigFile{
		Version: "1.0",
		FooterPolicy: FooterPolicy{
			AlwaysRegenerateQR:  true,
			OverwriteExistingQR: true,
		},
		DefaultFooter: QTFooterConfig{
			Mode:        QTFooterModeDefault,
			ShowFooter:  true,
			ShowDivider: true,
			ShowQR:      true,
			FooterText:  "말씀을 묵상으로, 묵상을 삶으로",
			HomepageURL: "https://s2gt.blogspot.com/",
			QRImagePath: filepath.Join("var", "image", "s2qt_link.png"),
			QRPosition:  "right-bottom",
			QRSizeMM:    27.0,
			SafeAreaMM:  32.0,
		},
		QRDefaults: DefaultQRRenderOptions(),
		SubscriberFooter: QTFooterConfig{
			Mode:        QTFooterModeSubscriber,
			ShowFooter:  true,
			ShowDivider: true,
			ShowQR:      true,
			QRImagePath: filepath.Join("var", "image", "church_qr.png"),
			QRPosition:  "right-bottom",
			QRSizeMM:    27.0,
			SafeAreaMM:  32.0,
		},
	}
}

func DefaultQRRenderOptions() QRRenderOptions {
	return QRRenderOptions{
		SizePx:         768,
		MarginModules:  2, // 기존 4 -> 2 (1차 테스트 권장)
		Style:          QRStyleDot,
		DotScale:       0.82,
		KeepFinderBox:  true,
		BackgroundRGBA: [4]uint8{0, 0, 0, 0},   // 완전 투명 배경
		ForegroundRGBA: [4]uint8{0, 0, 0, 255}, // 검정
	}
}

func loadFooterConfigFile() (*FooterConfigFile, error) {
	defaults := DefaultFooterConfigFile()
	path, err := resolveFooterConfigPath()
	if err != nil {
		return &defaults, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return &defaults, nil
	}

	var cfg FooterConfigFile
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("footer.json 파싱 실패: %w", err)
	}

	cfg = mergeFooterConfigFile(defaults, cfg)
	return &cfg, nil
}

func resolveFooterConfigPath() (string, error) {
	paths, err := util.GetAppPaths()
	if err == nil && paths != nil && strings.TrimSpace(paths.TempHtml) != "" {
		tempDir := filepath.Dir(paths.TempHtml)
		varDir := filepath.Dir(tempDir)
		p := filepath.Join(varDir, "conf", "footer.json")
		if _, statErr := os.Stat(p); statErr == nil {
			return p, nil
		}
	}

	p := filepath.Join("var", "conf", "footer.json")
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("footer.json을 찾지 못했습니다")
}

func mergeFooterConfigFile(base, loaded FooterConfigFile) FooterConfigFile {
	if strings.TrimSpace(loaded.Version) != "" {
		base.Version = loaded.Version
	}
	base.FooterPolicy = mergeFooterPolicy(base.FooterPolicy, loaded.FooterPolicy)
	base.DefaultFooter = mergeQTFooterConfig(base.DefaultFooter, loaded.DefaultFooter)
	base.QRDefaults = mergeQRRenderOptions(base.QRDefaults, loaded.QRDefaults)
	base.SubscriberFooter = mergeQTFooterConfig(base.SubscriberFooter, loaded.SubscriberFooter)
	return base
}

func mergeFooterPolicy(base, override FooterPolicy) FooterPolicy {
	if override.AlwaysRegenerateQR {
		base.AlwaysRegenerateQR = true
	}
	if override.OverwriteExistingQR {
		base.OverwriteExistingQR = true
	}
	return base
}

func mergeQTFooterConfig(base, override QTFooterConfig) QTFooterConfig {
	if override.Mode != "" {
		base.Mode = override.Mode
	}
	if override.ShowFooter {
		base.ShowFooter = true
	}
	if override.ShowDivider {
		base.ShowDivider = true
	}
	if override.ShowQR {
		base.ShowQR = true
	}
	if strings.TrimSpace(override.FooterText) != "" {
		base.FooterText = override.FooterText
	}
	if strings.TrimSpace(override.ChurchName) != "" {
		base.ChurchName = override.ChurchName
	}
	if strings.TrimSpace(override.LogoPath) != "" {
		base.LogoPath = override.LogoPath
	}
	if strings.TrimSpace(override.BrandImagePath) != "" {
		base.BrandImagePath = override.BrandImagePath
	}
	if strings.TrimSpace(override.HomepageURL) != "" {
		base.HomepageURL = override.HomepageURL
	}
	if strings.TrimSpace(override.QRImagePath) != "" {
		base.QRImagePath = override.QRImagePath
	}
	if strings.TrimSpace(override.QRPosition) != "" {
		base.QRPosition = override.QRPosition
	}
	if override.QRSizeMM > 0 {
		base.QRSizeMM = override.QRSizeMM
	}
	if override.SafeAreaMM > 0 {
		base.SafeAreaMM = override.SafeAreaMM
	}
	return base
}

func resolveFooterConfig(mode QTFooterMode, override *QTFooterConfig) (*QTFooterConfig, *FooterConfigFile, error) {
	fileCfg, err := loadFooterConfigFile()
	if err != nil {
		return nil, nil, err
	}

	var resolved QTFooterConfig
	switch mode {
	case QTFooterModeSubscriber:
		resolved = fileCfg.SubscriberFooter
	case QTFooterModeCustom:
		resolved = fileCfg.DefaultFooter
	default:
		resolved = fileCfg.DefaultFooter
	}

	if override != nil {
		resolved = mergeQTFooterConfig(resolved, *override)
		if override.Mode != "" {
			resolved.Mode = override.Mode
		}
	}

	resolved = normalizeResolvedFooterConfig(resolved)
	return &resolved, fileCfg, nil
}

func normalizeResolvedFooterConfig(cfg QTFooterConfig) QTFooterConfig {
	if cfg.Mode == "" {
		cfg.Mode = QTFooterModeDefault
	}
	if cfg.QRPosition == "" {
		cfg.QRPosition = "right-bottom"
	}
	if cfg.QRSizeMM <= 0 {
		cfg.QRSizeMM = 27.0
	}

	cfg.FooterText = strings.TrimSpace(cfg.FooterText)
	cfg.ChurchName = strings.TrimSpace(cfg.ChurchName)
	cfg.LogoPath = strings.TrimSpace(cfg.LogoPath)
	cfg.BrandImagePath = strings.TrimSpace(cfg.BrandImagePath)
	cfg.HomepageURL = strings.TrimSpace(cfg.HomepageURL)
	cfg.QRImagePath = strings.TrimSpace(cfg.QRImagePath)

	if cfg.HomepageURL == "" && cfg.QRImagePath == "" {
		cfg.ShowQR = false
		cfg.QRImagePath = ""
	} else if cfg.QRImagePath != "" {
		cfg.ShowQR = true
	}

	if !cfg.ShowFooter && cfg.FooterText == "" && cfg.ChurchName == "" && cfg.LogoPath == "" && cfg.BrandImagePath == "" {
		cfg.ShowFooter = true
		cfg.ShowDivider = true
		if cfg.FooterText == "" {
			cfg.FooterText = "말씀을 묵상으로, 묵상을 삶으로"
		}
	}

	if cfg.SafeAreaMM <= 0 {
		cfg.SafeAreaMM = resolveFooterSafeAreaMM(cfg)
	}
	return cfg
}

func resolveFooterSafeAreaMM(cfg QTFooterConfig) float64 {
	if cfg.SafeAreaMM > 0 {
		return cfg.SafeAreaMM
	}
	if cfg.ShowQR && (strings.TrimSpace(cfg.BrandImagePath) != "" || strings.TrimSpace(cfg.LogoPath) != "") {
		return 40.0
	}
	if cfg.ShowQR {
		return 36.0
	}
	return 24.0
}

func resolveFooterImagePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}

	paths, err := util.GetAppPaths()
	if err == nil && paths != nil && strings.TrimSpace(paths.TempHtml) != "" {
		tempDir := filepath.Dir(paths.TempHtml)
		rootDir := filepath.Dir(filepath.Dir(tempDir))
		return filepath.Join(rootDir, filepath.FromSlash(path))
	}
	return filepath.Clean(path)
}

func EncodeImageAsDataURI(path string) string {
	return encodeImageAsDataURI(path)
}

func encodeImageAsDataURI(path string) string {
	path = resolveFooterImagePath(path)
	if strings.TrimSpace(path) == "" {
		return ""
	}

	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 {
		return ""
	}

	ext := strings.ToLower(filepath.Ext(path))
	mime := "image/png"
	if ext == ".jpg" || ext == ".jpeg" {
		mime = "image/jpeg"
	} else if ext == ".webp" {
		mime = "image/webp"
	}

	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
}

func (s *QRService) WriteQRCode(text, outPath string, opts *QRRenderOptions) (*QRGenerateResult, error) {
	text = strings.TrimSpace(text)
	outPath = resolveFooterImagePath(outPath)

	if text == "" {
		return nil, fmt.Errorf("QR 내용이 비어 있습니다")
	}
	if strings.TrimSpace(outPath) == "" {
		return nil, fmt.Errorf("QR 출력 경로가 비어 있습니다")
	}

	finalOpts := DefaultQRRenderOptions()
	if opts != nil {
		finalOpts = mergeQRRenderOptions(finalOpts, *opts)
	}

	if finalOpts.SizePx <= 0 {
		finalOpts.SizePx = 768
	}
	if finalOpts.MarginModules < 0 {
		finalOpts.MarginModules = 2
	}
	if finalOpts.DotScale <= 0 || finalOpts.DotScale > 1.0 {
		finalOpts.DotScale = 0.82
	}
	if finalOpts.Style == "" {
		finalOpts.Style = QRStyleDot
	}

	qr, err := qrcode.New(text, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("QR 데이터 생성 실패: %w", err)
	}
	qr.DisableBorder = true

	bitmap := qr.Bitmap()
	if len(bitmap) == 0 || len(bitmap[0]) == 0 {
		return nil, fmt.Errorf("QR 비트맵 생성 결과가 비어 있습니다")
	}

	img, err := renderQRCodeBitmap(bitmap, finalOpts)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, fmt.Errorf("QR 출력 폴더 생성 실패: %w", err)
	}

	file, err := os.Create(outPath)
	if err != nil {
		return nil, fmt.Errorf("QR PNG 파일 생성 실패: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return nil, fmt.Errorf("QR PNG 인코딩 실패: %w", err)
	}

	return &QRGenerateResult{
		Success:  true,
		Message:  "QR 코드 PNG 생성이 완료되었습니다.",
		Text:     text,
		FilePath: outPath,
		SizePx:   finalOpts.SizePx,
		Style:    string(finalOpts.Style),
	}, nil
}

func (s *QRService) WriteDefaultS2QTLinkQRCode() (*QRGenerateResult, error) {
	cfg, fileCfg, err := resolveFooterConfig(QTFooterModeDefault, nil)
	if err != nil {
		return nil, err
	}

	qrText := strings.TrimSpace(cfg.HomepageURL)
	if qrText == "" {
		qrText = "https://s2gt.blogspot.com/"
	}

	return s.WriteQRCode(normalizeURLIfNeeded(qrText), cfg.QRImagePath, &fileCfg.QRDefaults)
}

func (s *QRService) WriteChurchURLQRCode(churchURL, outPath string, opts *QRRenderOptions) (*QRGenerateResult, error) {
	churchURL = normalizeURLIfNeeded(churchURL)
	if strings.TrimSpace(churchURL) == "" {
		return nil, fmt.Errorf("교회 홈페이지 URL이 비어 있습니다")
	}

	if strings.TrimSpace(outPath) == "" {
		cfg, _, err := resolveFooterConfig(QTFooterModeSubscriber, nil)
		if err != nil {
			return nil, err
		}
		outPath = cfg.QRImagePath
	}

	return s.WriteQRCode(churchURL, outPath, opts)
}

func (s *QRService) PrepareFooterAssets(mode QTFooterMode, override *QTFooterConfig) (*QTFooterConfig, error) {
	resolved, fileCfg, err := resolveFooterConfig(mode, override)
	if err != nil {
		return nil, err
	}

	resolved.LogoPath = resolveFooterImagePath(resolved.LogoPath)
	resolved.BrandImagePath = resolveFooterImagePath(resolved.BrandImagePath)
	resolved.QRImagePath = resolveFooterImagePath(resolved.QRImagePath)

	if resolved.ShowQR {
		qrText := strings.TrimSpace(resolved.HomepageURL)
		if qrText == "" && resolved.Mode == QTFooterModeDefault {
			qrText = "https://s2gt.blogspot.com/"
		}

		// 핵심:
		// 1) HomepageURL이 없어도 QRImagePath가 이미 있으면 fallback QR 유지
		// 2) 둘 다 없을 때만 QR 비활성화
		if qrText == "" {
			if strings.TrimSpace(resolved.QRImagePath) == "" {
				resolved.ShowQR = false
				resolved.QRImagePath = ""
			}
		} else {
			result, err := s.WriteQRCode(normalizeURLIfNeeded(qrText), resolved.QRImagePath, &fileCfg.QRDefaults)
			if err != nil {
				return nil, err
			}
			resolved.QRImagePath = result.FilePath
		}
	}

	resolved.SafeAreaMM = resolveFooterSafeAreaMM(*resolved)
	return resolved, nil
}

func mergeQRRenderOptions(base, override QRRenderOptions) QRRenderOptions {
	if override.SizePx > 0 {
		base.SizePx = override.SizePx
	}
	if override.MarginModules >= 0 {
		base.MarginModules = override.MarginModules
	}
	if override.Style != "" {
		base.Style = override.Style
	}
	if override.DotScale > 0 {
		base.DotScale = override.DotScale
	}
	if override.KeepFinderBox {
		base.KeepFinderBox = true
	}
	if override.BackgroundRGBA != [4]uint8{} {
		base.BackgroundRGBA = override.BackgroundRGBA
	}
	if override.ForegroundRGBA != [4]uint8{} {
		base.ForegroundRGBA = override.ForegroundRGBA
	}
	return base
}

func renderQRCodeBitmap(bitmap [][]bool, opts QRRenderOptions) (*image.RGBA, error) {
	moduleCount := len(bitmap)
	if moduleCount == 0 {
		return nil, fmt.Errorf("QR 비트맵이 비어 있습니다")
	}

	totalModules := float64(moduleCount + (opts.MarginModules * 2))
	moduleSize := float64(opts.SizePx) / totalModules
	if moduleSize < 1 {
		return nil, fmt.Errorf("QR 크기가 너무 작습니다")
	}

	bg := rgbaFromArray(opts.BackgroundRGBA)
	fg := rgbaFromArray(opts.ForegroundRGBA)

	img := image.NewRGBA(image.Rect(0, 0, opts.SizePx, opts.SizePx))
	fillRect(img, 0, 0, opts.SizePx, opts.SizePx, bg)

	for row := 0; row < moduleCount; row++ {
		for col := 0; col < moduleCount; col++ {
			if !bitmap[row][col] {
				continue
			}

			if opts.KeepFinderBox && isFinderModule(row, col, moduleCount) {
				continue
			}

			x0, y0, x1, y1 := moduleRect(col, row, opts.MarginModules, moduleSize)

			switch opts.Style {
			case QRStyleSquare:
				fillRect(img, x0, y0, x1, y1, fg)
			case QRStyleDot:
				fillCircleInRect(img, x0, y0, x1, y1, fg, opts.DotScale)
			default:
				fillRect(img, x0, y0, x1, y1, fg)
			}
		}
	}

	if opts.KeepFinderBox {
		drawFinderPatterns(img, moduleCount, opts.MarginModules, moduleSize, bg, fg)
	}

	return img, nil
}

func drawFinderPatterns(img *image.RGBA, moduleCount, marginModules int, moduleSize float64, bg, fg color.RGBA) {
	type point struct{ col, row int }
	starts := []point{
		{0, 0},
		{moduleCount - 7, 0},
		{0, moduleCount - 7},
	}
	for _, p := range starts {
		drawFinderBox(img, p.col, p.row, marginModules, moduleSize, bg, fg)
	}
}

func drawFinderBox(img *image.RGBA, startCol, startRow, marginModules int, moduleSize float64, bg, fg color.RGBA) {
	paintModuleBlock(img, startCol, startRow, 7, marginModules, moduleSize, fg)
	paintModuleBlock(img, startCol+1, startRow+1, 5, marginModules, moduleSize, bg)
	paintModuleBlock(img, startCol+2, startRow+2, 3, marginModules, moduleSize, fg)
}

func paintModuleBlock(img *image.RGBA, startCol, startRow, size, marginModules int, moduleSize float64, c color.RGBA) {
	x0, y0, _, _ := moduleRect(startCol, startRow, marginModules, moduleSize)
	_, _, x1, y1 := moduleRect(startCol+size-1, startRow+size-1, marginModules, moduleSize)
	fillRect(img, x0, y0, x1, y1, c)
}

func isFinderModule(row, col, moduleCount int) bool {
	inTopLeft := row >= 0 && row < 7 && col >= 0 && col < 7
	inTopRight := row >= 0 && row < 7 && col >= moduleCount-7 && col < moduleCount
	inBottomLeft := row >= moduleCount-7 && row < moduleCount && col >= 0 && col < 7
	return inTopLeft || inTopRight || inBottomLeft
}

func moduleRect(col, row, marginModules int, moduleSize float64) (int, int, int, int) {
	x0 := int(math.Round(float64(col+marginModules) * moduleSize))
	y0 := int(math.Round(float64(row+marginModules) * moduleSize))
	x1 := int(math.Round(float64(col+marginModules+1) * moduleSize))
	y1 := int(math.Round(float64(row+marginModules+1) * moduleSize))
	return x0, y0, x1, y1
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	b := img.Bounds()
	if x0 < b.Min.X {
		x0 = b.Min.X
	}
	if y0 < b.Min.Y {
		y0 = b.Min.Y
	}
	if x1 > b.Max.X {
		x1 = b.Max.X
	}
	if y1 > b.Max.Y {
		y1 = b.Max.Y
	}

	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func fillCircleInRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA, dotScale float64) {
	if x1 <= x0 || y1 <= y0 {
		return
	}

	cx := float64(x0+x1) / 2.0
	cy := float64(y0+y1) / 2.0
	w := float64(x1 - x0)
	h := float64(y1 - y0)
	r := math.Min(w, h) * dotScale / 2.0
	r2 := r * r

	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			dx := (float64(x) + 0.5) - cx
			dy := (float64(y) + 0.5) - cy
			if dx*dx+dy*dy <= r2 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func rgbaFromArray(v [4]uint8) color.RGBA {
	return color.RGBA{R: v[0], G: v[1], B: v[2], A: v[3]}
}

func normalizeURLIfNeeded(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return s
	}
	if strings.Contains(s, ".") {
		return "https://" + s
	}
	return s
}

func (s *QRService) GetSideNavQRDataURI() (string, error) {
	if s == nil || s.Paths == nil {
		return "", fmt.Errorf("QRService 또는 Paths가 nil 입니다")
	}

	qrPath := strings.TrimSpace(s.Paths.DefaultQRFile)

	// footer 설정에 기본 QR 경로가 있으면 우선 사용
	if cfg, _, err := resolveFooterConfig(QTFooterModeDefault, nil); err == nil && cfg != nil {
		if strings.TrimSpace(cfg.QRImagePath) != "" {
			qrPath = strings.TrimSpace(cfg.QRImagePath)
		}
	}

	// 먼저 기존 파일을 바로 data URI로 변환 시도
	dataURI := EncodeImageAsDataURI(qrPath)
	if dataURI != "" {
		return dataURI, nil
	}

	// 없으면 기본 S2QT 링크 QR 생성 후 다시 변환
	result, err := s.WriteDefaultS2QTLinkQRCode()
	if err != nil {
		return "", err
	}
	if result == nil || strings.TrimSpace(result.FilePath) == "" {
		return "", nil
	}

	return EncodeImageAsDataURI(result.FilePath), nil
}

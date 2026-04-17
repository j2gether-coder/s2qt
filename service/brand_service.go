package service

import (
	"database/sql"
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"s2qt/util"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type BrandSettings struct {
	ChurchName         string
	Denomination       string
	ChurchOnlyName     string
	LogoPath           string
	BrandImageIncluded bool
}

type BrandPrepareResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	BrandFile string `json:"brandFile"`
	Source    string `json:"source"`
}

type BrandService struct {
	DB    *sql.DB
	Paths *util.AppPaths
}

func NewBrandService(db *sql.DB) (*BrandService, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return &BrandService{
		DB:    db,
		Paths: paths,
	}, nil
}

func (s *BrandService) PrepareBrandImageFromDB() (*BrandPrepareResult, error) {
	settings, err := s.LoadBrandSettings()
	if err != nil {
		return nil, err
	}

	sourcePath := resolveFooterImagePath(settings.LogoPath)
	if strings.TrimSpace(sourcePath) == "" {
		return nil, fmt.Errorf("church.logo_path 가 비어 있습니다")
	}
	if !FileExists(sourcePath) {
		return nil, fmt.Errorf("로고 원본 이미지를 찾을 수 없습니다: %s", sourcePath)
	}

	brandPath := s.defaultBrandOutputPath() // 내부 중간 파일
	finalPath := s.defaultFinalOutputPath() // 최종 site_logo.png

	if err := EnsureParentDir(brandPath); err != nil {
		return nil, err
	}
	if err := EnsureParentDir(finalPath); err != nil {
		return nil, err
	}

	if settings.BrandImageIncluded {
		if err := s.writeNormalizedPNG(sourcePath, brandPath); err != nil {
			return nil, err
		}
		if err := CopyFile(brandPath, finalPath); err != nil {
			return nil, fmt.Errorf("최종 site_logo.png 반영 실패: %w", err)
		}
		_ = os.Remove(brandPath)

		return &BrandPrepareResult{
			Success:   true,
			Message:   "완성형 로고 이미지를 정규화한 뒤 site_logo.png로 반영했습니다.",
			BrandFile: finalPath,
			Source:    sourcePath,
		}, nil
	}

	if err := s.writeComposedBrandPNG(sourcePath, brandPath, settings); err != nil {
		return nil, err
	}
	if err := CopyFile(brandPath, finalPath); err != nil {
		return nil, fmt.Errorf("합성 결과를 site_logo.png로 반영 실패: %w", err)
	}
	_ = os.Remove(brandPath)

	return &BrandPrepareResult{
		Success:   true,
		Message:   "로고와 교회명을 합성한 뒤 site_logo.png로 반영했습니다.",
		BrandFile: finalPath,
		Source:    sourcePath,
	}, nil
}

func (s *BrandService) LoadBrandSettings() (*BrandSettings, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("db is nil")
	}

	keys := []string{
		"church.name",
		"church.logo_path",
		"church.brand_image_included",
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
		return nil, fmt.Errorf("brand settings 조회 실패: %w", err)
	}
	defer rows.Close()

	result := &BrandSettings{}

	for rows.Next() {
		var key string
		var value sql.NullString

		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("brand settings scan 실패: %w", err)
		}

		v := ""
		if value.Valid {
			v = strings.TrimSpace(value.String)
		}

		switch key {
		case "church.name":
			result.ChurchName = normalizeChurchDisplayName(v)
			result.Denomination, result.ChurchOnlyName = splitChurchDisplayName(v)
			if strings.TrimSpace(result.ChurchOnlyName) == "" {
				result.ChurchOnlyName = strings.TrimSpace(v)
			}
		case "church.logo_path":
			result.LogoPath = v
		case "church.brand_image_included":
			result.BrandImageIncluded = parseBoolText(v)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("brand settings 읽기 실패: %w", err)
	}

	return result, nil
}

func (s *BrandService) defaultBrandOutputPath() string {
	if s.Paths != nil && strings.TrimSpace(s.Paths.SiteLogoFile) != "" {
		dir := filepath.Dir(s.Paths.SiteLogoFile)
		return filepath.Join(dir, "site_brand.png")
	}
	return resolveFooterImagePath("var/image/site_brand.png")
}

func (s *BrandService) defaultFinalOutputPath() string {
	if s.Paths != nil && strings.TrimSpace(s.Paths.SiteLogoFile) != "" {
		return s.Paths.SiteLogoFile
	}
	return resolveFooterImagePath("var/image/site_logo.png")
}

func (s *BrandService) writeNormalizedPNG(srcPath, outPath string) error {
	img, err := decodeBrandImageFile(srcPath)
	if err != nil {
		return err
	}

	b := img.Bounds()
	canvas := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	stddraw.Draw(canvas, canvas.Bounds(), img, b.Min, stddraw.Src)

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("brand 파일 생성 실패: %w", err)
	}
	defer f.Close()

	if err := png.Encode(f, canvas); err != nil {
		return fmt.Errorf("brand PNG 인코딩 실패: %w", err)
	}

	return nil
}

func (s *BrandService) writeComposedBrandPNG(srcPath, outPath string, settings *BrandSettings) error {
	logoImg, err := decodeBrandImageFile(srcPath)
	if err != nil {
		return err
	}

	denomText := strings.TrimSpace(settings.Denomination)
	churchText := strings.TrimSpace(settings.ChurchOnlyName)
	if churchText == "" {
		churchText = strings.TrimSpace(settings.ChurchName)
	}
	if churchText == "" {
		return fmt.Errorf("교회명/브랜드명이 비어 있습니다")
	}

	denomFace, err := loadSystemFontFace(12)
	if err != nil {
		return fmt.Errorf("교단명 폰트 로드 실패: %w", err)
	}
	churchFace, err := loadSystemFontFace(16)
	if err != nil {
		return fmt.Errorf("교회명 폰트 로드 실패: %w", err)
	}

	denomW := measureTextWidth(denomFace, denomText)
	churchW := measureTextWidth(churchFace, churchText)
	textBlockW := maxInt(denomW, churchW)

	denomH := faceLineHeight(denomFace)
	churchH := faceLineHeight(churchFace)

	textGap := 4
	textBlockH := churchH
	if denomText != "" {
		textBlockH = denomH + textGap + churchH
	}

	logoBounds := logoImg.Bounds()
	logoW := logoBounds.Dx()
	logoH := logoBounds.Dy()
	if logoW <= 0 || logoH <= 0 {
		return fmt.Errorf("로고 이미지 크기가 올바르지 않습니다")
	}

	targetLogoH := textBlockH
	if targetLogoH < 20 {
		targetLogoH = 20
	}
	targetLogoW := int(float64(logoW) * (float64(targetLogoH) / float64(logoH)))
	if targetLogoW < 1 {
		targetLogoW = 1
	}

	resizedLogo := image.NewRGBA(image.Rect(0, 0, targetLogoW, targetLogoH))
	draw.ApproxBiLinear.Scale(resizedLogo, resizedLogo.Bounds(), logoImg, logoBounds, draw.Over, nil)

	paddingX := 12
	paddingY := 8
	gap := 12

	canvasW := paddingX + targetLogoW + gap + textBlockW + paddingX
	canvasH := maxInt(targetLogoH+(paddingY*2), textBlockH+(paddingY*2))

	canvas := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
	stddraw.Draw(canvas, canvas.Bounds(), image.Transparent, image.Point{}, stddraw.Src)

	logoX := paddingX
	logoY := (canvasH - targetLogoH) / 2
	stddraw.Draw(
		canvas,
		image.Rect(logoX, logoY, logoX+targetLogoW, logoY+targetLogoH),
		resizedLogo,
		image.Point{},
		stddraw.Over,
	)

	textX := paddingX + targetLogoW + gap
	textTop := (canvasH - textBlockH) / 2

	denomColor := color.RGBA{R: 90, G: 90, B: 90, A: 255}
	churchColor := color.RGBA{R: 20, G: 20, B: 20, A: 255}

	if denomText != "" {
		drawTextLine(canvas, denomFace, textX, textTop+faceAscent(denomFace), denomText, denomColor)

		churchX := textX + (textBlockW-churchW)/2
		drawTextLine(
			canvas,
			churchFace,
			churchX,
			textTop+denomH+textGap+faceAscent(churchFace),
			churchText,
			churchColor,
		)
	} else {
		drawTextLine(canvas, churchFace, textX, textTop+faceAscent(churchFace), churchText, churchColor)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("합성 로고 파일 생성 실패: %w", err)
	}
	defer f.Close()

	if err := png.Encode(f, canvas); err != nil {
		return fmt.Errorf("합성 로고 PNG 인코딩 실패: %w", err)
	}

	return nil
}

func decodeBrandImageFile(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("이미지 열기 실패: %w", err)
	}
	defer file.Close()

	ext := strings.ToLower(path)
	switch {
	case strings.HasSuffix(ext, ".png"):
		img, err := png.Decode(file)
		if err != nil {
			return nil, fmt.Errorf("png decode 실패: %w", err)
		}
		return img, nil
	case strings.HasSuffix(ext, ".jpg"), strings.HasSuffix(ext, ".jpeg"):
		img, err := jpeg.Decode(file)
		if err != nil {
			return nil, fmt.Errorf("jpeg decode 실패: %w", err)
		}
		return img, nil
	default:
		img, _, err := image.Decode(file)
		if err != nil {
			return nil, fmt.Errorf("image decode 실패: %w", err)
		}
		return img, nil
	}
}

func loadSystemFontFace(size float64) (font.Face, error) {
	for _, path := range candidateSystemFontPaths() {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		ft, err := opentype.Parse(b)
		if err != nil {
			continue
		}

		face, err := opentype.NewFace(ft, &opentype.FaceOptions{
			Size:    size,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err == nil {
			return face, nil
		}
	}

	return nil, fmt.Errorf("사용 가능한 시스템 폰트를 찾지 못했습니다")
}

func candidateSystemFontPaths() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			`C:\Windows\Fonts\NanumGothic.ttf`,
			`C:\Windows\Fonts\NanumGothicBold.ttf`,
			`C:\Windows\Fonts\malgun.ttf`,
			`C:\Windows\Fonts\malgunbd.ttf`,
		}
	case "darwin":
		return []string{
			`/Library/Fonts/NanumGothic.ttf`,
			`/System/Library/Fonts/Supplemental/Apple SD Gothic Neo.ttc`,
			`/System/Library/Fonts/Supplemental/AppleGothic.ttf`,
		}
	default:
		return []string{
			`/usr/share/fonts/truetype/nanum/NanumGothic.ttf`,
			`/usr/share/fonts/truetype/nanum/NanumGothicBold.ttf`,
			`/usr/share/fonts/truetype/malgun/malgun.ttf`,
			`/usr/share/fonts/truetype/msttcorefonts/malgun.ttf`,
		}
	}
}

func drawTextLine(dst stddraw.Image, face font.Face, x, y int, text string, clr color.Color) {
	if strings.TrimSpace(text) == "" {
		return
	}

	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(clr),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

func measureTextWidth(face font.Face, text string) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	d := &font.Drawer{Face: face}
	return d.MeasureString(text).Ceil()
}

func faceLineHeight(face font.Face) int {
	return face.Metrics().Height.Ceil()
}

func faceAscent(face font.Face) int {
	return face.Metrics().Ascent.Ceil()
}

func maxInt(a, b int) int {
	if a >= b {
		return a
	}
	return b
}
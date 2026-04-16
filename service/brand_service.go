package service

import (
	"database/sql"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"strings"

	"s2qt/util"
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
	if !ifileExists(sourcePath) {
		return nil, fmt.Errorf("로고/브랜드 원본 이미지를 찾을 수 없습니다: %s", sourcePath)
	}

	outPath := s.defaultBrandOutputPath()

	if err := os.MkdirAll(filepathDir(outPath), 0o755); err != nil {
		return nil, fmt.Errorf("brand 출력 디렉토리 생성 실패: %w", err)
	}

	if err := s.writeNormalizedPNG(sourcePath, outPath); err != nil {
		return nil, err
	}

	msg := "로고 이미지를 기준으로 church_brand.png를 생성했습니다."
	if settings.BrandImageIncluded {
		msg = "완성형 브랜드 이미지를 church_brand.png로 저장했습니다."
	}

	return &BrandPrepareResult{
		Success:   true,
		Message:   msg,
		BrandFile: outPath,
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
	if s.Paths != nil && strings.TrimSpace(s.Paths.ChurchBrandFile) != "" {
		return s.Paths.ChurchBrandFile
	}
	return resolveFooterImagePath("var/image/church_brand.png")
}

func (s *BrandService) writeNormalizedPNG(srcPath, outPath string) error {
	img, err := decodeBrandImageFile(srcPath)
	if err != nil {
		return err
	}

	b := img.Bounds()
	canvas := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(canvas, canvas.Bounds(), img, b.Min, draw.Src)

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

func filepathDir(path string) string {
	i := strings.LastIndexAny(path, `/\`)
	if i < 0 {
		return "."
	}
	return path[:i]
}

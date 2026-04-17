package service

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"s2qt/util"
)

type FooterSettings struct {
	ChurchName         string
	Denomination       string
	ChurchOnlyName     string
	LogoPath           string
	HomepageURL        string
	FooterText         string
	BrandImageIncluded bool
}

type FooterService struct {
	DB    *sql.DB
	Paths *util.AppPaths
	QR    *QRService
}

func NewFooterService(db *sql.DB) (*FooterService, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	qrSvc, err := NewQRService()
	if err != nil {
		return nil, err
	}

	return &FooterService{
		DB:    db,
		Paths: paths,
		QR:    qrSvc,
	}, nil
}

func (s *FooterService) PrepareFooterConfigFromDB(mode QTFooterMode) (*QTFooterConfig, error) {
	settings, err := s.LoadFooterSettings()
	if err != nil {
		return nil, err
	}

	cfg := &QTFooterConfig{
		Mode:           mode,
		ShowFooter:     true,
		ShowDivider:    true,
		ShowQR:         false,
		FooterText:     strings.TrimSpace(settings.FooterText),
		ChurchName:     strings.TrimSpace(settings.ChurchName),
		LogoPath:       resolveFooterImagePath(settings.LogoPath),
		BrandImagePath: "",
		HomepageURL:    strings.TrimSpace(settings.HomepageURL),
		QRImagePath:    "",
		QRPosition:     "right-bottom",
		QRSizeMM:       27.0,
	}

	if cfg.FooterText == "" {
		cfg.FooterText = "말씀을 묵상으로, 묵상을 삶으로"
	}

	// 1) brand_service.go 시도
	if brandSvc, err := NewBrandService(s.DB); err == nil && brandSvc != nil {
		if brandRes, err := brandSvc.PrepareBrandImageFromDB(); err == nil && brandRes != nil {
			brandFile := strings.TrimSpace(brandRes.BrandFile)
			if brandFile != "" && ifileExists(brandFile) {
				cfg.BrandImagePath = brandFile
			}
		}
	}

	// 2) brand_service.go 결과가 없으면 fallback
	if strings.TrimSpace(cfg.BrandImagePath) == "" {
		brandImagePath := s.resolveBrandImagePath(settings)
		if brandImagePath != "" {
			cfg.BrandImagePath = brandImagePath
		}
	}

	// 3) 홈페이지 URL이 있으면 church_qr.png 생성, 없으면 생략
	homepageURL := strings.TrimSpace(settings.HomepageURL)
	if homepageURL != "" {
		qrOut := s.defaultQRImagePath()

		if _, err := s.QR.WriteChurchURLQRCode(homepageURL, qrOut, nil); err == nil {
			if ifileExists(qrOut) {
				cfg.ShowQR = true
				cfg.QRImagePath = qrOut
			}
		} else {
			cfg.ShowQR = false
			cfg.QRImagePath = ""
		}
	}

	cfg.SafeAreaMM = resolveFooterSafeAreaMM(*cfg)
	return cfg, nil
}

func (s *FooterService) LoadFooterSettings() (*FooterSettings, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("db is nil")
	}

	keys := []string{
		"church.name",
		"church.logo_path",
		"church.brand_image_included",
		"church.homepage_url",
		"church.default_footer_text",
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
		return nil, fmt.Errorf("failed to query footer settings: %w", err)
	}
	defer rows.Close()

	result := &FooterSettings{}

	for rows.Next() {
		var key string
		var value sql.NullString

		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan footer setting: %w", err)
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
		case "church.homepage_url":
			result.HomepageURL = v
		case "church.default_footer_text":
			result.FooterText = v
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading footer settings: %w", err)
	}

	return result, nil
}

func (s *FooterService) resolveBrandImagePath(settings *FooterSettings) string {
	if settings == nil {
		return ""
	}

	logoPath := resolveFooterImagePath(settings.LogoPath)
	if settings.BrandImageIncluded {
		if ifileExists(logoPath) {
			return logoPath
		}
		return ""
	}

	if ifileExists(logoPath) {
		return logoPath
	}

	return ""
}

func (s *FooterService) defaultQRImagePath() string {
	if s.Paths != nil && strings.TrimSpace(s.Paths.SiteQRFile) != "" {
		return s.Paths.SiteQRFile
	}
	return resolveFooterImagePath("var/image/church_qr.png")
}

func splitChurchDisplayName(v string) (string, string) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", ""
	}

	parts := strings.SplitN(v, ",", 2)
	if len(parts) < 2 {
		return "", v
	}

	denom := strings.TrimSpace(parts[0])
	church := strings.TrimSpace(parts[1])
	return denom, church
}

func normalizeChurchDisplayName(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	denom, church := splitChurchDisplayName(v)
	if denom == "" {
		return church
	}
	if church == "" {
		return denom
	}
	return denom + " " + church
}

func parseBoolText(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "y", "yes", "on":
		return true
	default:
		return false
	}
}

func ifileExists(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

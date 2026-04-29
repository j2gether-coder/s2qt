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

	// site_logo.png는 환경설정 저장 시점에 이미 footer용 최종 이미지로 정규화되어 있어야 한다.
	// Step3 산출물 생성 시점에서는 BrandService를 다시 호출하지 않는다.
	// 이유: site_logo.png를 다시 읽어 재합성하면 교회명/브랜드명이 중복 합성될 수 있다.
	logoConfigured := strings.TrimSpace(cfg.LogoPath) != "" && ifileExists(cfg.LogoPath)

	if logoConfigured {
		cfg.BrandImagePath = cfg.LogoPath
		cfg.ChurchName = ""
		cfg.LogoPath = ""
	}

	homepageURL := strings.TrimSpace(settings.HomepageURL)
	if homepageURL != "" {
		qrOut := s.defaultQRImagePath()

		if _, err := s.QR.WriteChurchURLQRCode(homepageURL, qrOut, nil); err == nil {
			if ifileExists(qrOut) {
				cfg.ShowQR = true
				cfg.QRImagePath = qrOut
				cfg.QRPosition = "right-bottom"
				cfg.QRSizeMM = 27.0
			}
		}
	}

	if strings.TrimSpace(cfg.QRImagePath) == "" {
		fallbackQR := s.resolveDefaultQRFallbackPath()
		if fallbackQR != "" {
			cfg.ShowQR = true
			cfg.QRImagePath = fallbackQR
			cfg.QRPosition = "right-bottom"
			cfg.QRSizeMM = 27.0
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
		"church.logo_with_name",
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
		case "church.logo_with_name":
			if !result.BrandImageIncluded {
				result.BrandImageIncluded = parseBoolText(v)
			}
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

	// fallback은 완성형 로고인 경우에만 허용
	if !settings.BrandImageIncluded {
		return ""
	}

	logoPath := resolveFooterImagePath(settings.LogoPath)
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

func (s *FooterService) resolveDefaultBrandFallbackPath() string {
	if s == nil || s.Paths == nil {
		return ""
	}

	if strings.TrimSpace(s.Paths.DefaultQRFile) != "" && ifileExists(s.Paths.DefaultQRFile) {
		return s.Paths.DefaultQRFile
	}

	fallback := resolveFooterImagePath("var/image/s2qt_link.png")
	if ifileExists(fallback) {
		return fallback
	}

	return ""
}

func (s *FooterService) resolveDefaultQRFallbackPath() string {
	if s == nil || s.Paths == nil {
		return ""
	}

	if strings.TrimSpace(s.Paths.DefaultQRFile) != "" && ifileExists(s.Paths.DefaultQRFile) {
		return s.Paths.DefaultQRFile
	}

	fallback := resolveFooterImagePath("var/image/s2qt_link.png")
	if ifileExists(fallback) {
		return fallback
	}

	return ""
}

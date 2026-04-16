package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"s2qt/service"
	"s2qt/util"
)

type FooterTestInput struct {
	ChurchName         string `json:"church_name"`
	LogoPath           string `json:"logo_path"`
	BrandImageIncluded bool   `json:"brand_image_included"`
	HomepageURL        string `json:"homepage_url"`
	FooterText         string `json:"footer_text"`
}

type appSettingBackup struct {
	Key   string
	Found bool
	Value string
}

func main() {
	if err := run(); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[DONE] footer step2->step3 test completed")
}

func run() error {
	input, err := loadFooterTestJSON("test/footer_test.json")
	if err != nil {
		return err
	}

	printChurchNameRuleInfo(input.ChurchName)

	paths, err := util.GetAppPaths()
	if err != nil {
		return fmt.Errorf("app paths load failed: %w", err)
	}

	// temp.json 선행 확인
	if _, err := os.Stat(paths.TempJson); err != nil {
		return fmt.Errorf("temp.json이 없습니다. Step1/Step2 선행 데이터가 필요합니다: %w", err)
	}

	// 테스트 원본 로고를 고정 경로(church_logo.png)로 준비
	if err := prepareTestLogoFile(input.LogoPath, paths.ChurchLogoFile); err != nil {
		return fmt.Errorf("test logo prepare failed: %w", err)
	}

	db, err := service.OpenSQLite(paths.DBFile)
	if err != nil {
		return fmt.Errorf("db open failed: %w", err)
	}

	if err := service.InitSQLite(db); err != nil {
		return fmt.Errorf("db init failed: %w", err)
	}

	backups, err := upsertFooterTestSettings(db, input, paths.ChurchLogoFile)
	if err != nil {
		return fmt.Errorf("failed to save test settings: %w", err)
	}
	defer restoreAppSettings(db, backups)

	step2Svc, err := service.NewQTStep2Service()
	if err != nil {
		return fmt.Errorf("new QTStep2Service failed: %w", err)
	}

	step2Data, err := step2Svc.Load()
	if err != nil {
		return fmt.Errorf("step2 load failed: %w", err)
	}

	htmlPath, err := step2Svc.BuildHTML(step2Data)
	if err != nil {
		return fmt.Errorf("step2 build html failed: %w", err)
	}

	fmt.Println("=== Step2 Result ===")
	fmt.Printf("temp.json : %s\n", paths.TempJson)
	fmt.Printf("temp.html : %s\n", htmlPath)

	step3Svc, err := service.NewQTStep3Service()
	if err != nil {
		return fmt.Errorf("new QTStep3Service failed: %w", err)
	}

	req := &service.QTStep3Request{
		MakeHTML: false,
		MakePDF:  true,
		MakePNG:  true,
		MakeDOCX: false,
		MakePPTX: false,
		DPI:      300,
	}

	result, err := step3Svc.Run(req)
	if err != nil {
		return fmt.Errorf("step3 run failed: %w", err)
	}

	fmt.Println("=== Step3 Result ===")
	fmt.Printf("PDF  : success=%v, status=%s, file=%s, error=%s\n",
		result.PDF.Success, result.PDF.Status, result.PDF.FilePath, result.PDF.Error)
	fmt.Printf("PNG  : success=%v, status=%s, file=%s, error=%s\n",
		result.PNG.Success, result.PNG.Status, result.PNG.FilePath, result.PNG.Error)

	checkFiles(
		paths.ChurchLogoFile,
		paths.ChurchBrandFile,
		paths.ChurchQRFile,
		paths.DefaultQRFile,
		paths.TempJson,
		paths.TempHtml,
		paths.TempPdf,
		paths.TempPng,
	)

	return nil
}

func loadFooterTestJSON(path string) (*FooterTestInput, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read footer_test.json: %w", err)
	}

	var input FooterTestInput
	if err := json.Unmarshal(b, &input); err != nil {
		return nil, fmt.Errorf("failed to parse footer_test.json: %w", err)
	}

	input.ChurchName = strings.TrimSpace(input.ChurchName)
	input.LogoPath = strings.TrimSpace(input.LogoPath)
	input.HomepageURL = strings.TrimSpace(input.HomepageURL)
	input.FooterText = strings.TrimSpace(input.FooterText)

	if input.ChurchName == "" {
		return nil, fmt.Errorf("church_name is empty")
	}
	if input.LogoPath == "" {
		return nil, fmt.Errorf("logo_path is empty")
	}

	return &input, nil
}

func prepareTestLogoFile(srcPath, dstPath string) error {
	srcPath = strings.TrimSpace(srcPath)
	dstPath = strings.TrimSpace(dstPath)

	if srcPath == "" {
		return fmt.Errorf("source logo path is empty")
	}
	if dstPath == "" {
		return fmt.Errorf("destination logo path is empty")
	}

	srcAbs, err := filepath.Abs(srcPath)
	if err != nil {
		return fmt.Errorf("source abs path failed: %w", err)
	}

	srcInfo, err := os.Stat(srcAbs)
	if err != nil {
		return fmt.Errorf("source logo file not found: %w", err)
	}
	if srcInfo.IsDir() {
		return fmt.Errorf("source logo path is directory: %s", srcAbs)
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("failed to create destination dir: %w", err)
	}

	in, err := os.Open(srcAbs)
	if err != nil {
		return fmt.Errorf("failed to open source logo: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination logo: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy logo file: %w", err)
	}

	return nil
}

func printChurchNameRuleInfo(v string) {
	denom, church := splitChurchNameForTest(v)

	fmt.Println("=== Church Name Rule Check ===")
	fmt.Printf("Raw Input        : %s\n", v)

	if denom == "" {
		fmt.Println("[WARN] church_name 에 쉼표(,)가 없습니다.")
		fmt.Println("[WARN] 권장 입력 형식: 교단명, 교회명")
		fmt.Printf("Church Only      : %s\n", church)
		return
	}

	fmt.Printf("Denomination     : %s\n", denom)
	fmt.Printf("Church Only Name : %s\n", church)
}

func splitChurchNameForTest(v string) (string, string) {
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

func upsertFooterTestSettings(db *sql.DB, input *FooterTestInput, fixedLogoPath string) ([]appSettingBackup, error) {
	items := map[string]string{
		"church.name":                 input.ChurchName,
		"church.logo_path":            fixedLogoPath,
		"church.brand_image_included": boolToText(input.BrandImageIncluded),
		"church.homepage_url":         input.HomepageURL,
		"church.default_footer_text":  input.FooterText,
	}

	backups := make([]appSettingBackup, 0, len(items))

	for key, newValue := range items {
		backup, err := backupAppSetting(db, key)
		if err != nil {
			return nil, err
		}
		backups = append(backups, backup)

		if err := upsertAppSetting(db, key, newValue); err != nil {
			return nil, err
		}
	}

	return backups, nil
}

func backupAppSetting(db *sql.DB, key string) (appSettingBackup, error) {
	var value string
	err := db.QueryRow(`
SELECT setting_value
FROM app_settings
WHERE setting_key = ?
`, key).Scan(&value)

	if err == sql.ErrNoRows {
		return appSettingBackup{
			Key:   key,
			Found: false,
			Value: "",
		}, nil
	}
	if err != nil {
		return appSettingBackup{}, fmt.Errorf("failed to backup app_setting %s: %w", key, err)
	}

	return appSettingBackup{
		Key:   key,
		Found: true,
		Value: value,
	}, nil
}

func restoreAppSettings(db *sql.DB, backups []appSettingBackup) {
	for _, b := range backups {
		if b.Found {
			_, _ = db.Exec(`
UPDATE app_settings
SET setting_value = ?, updated_at = datetime('now', 'localtime')
WHERE setting_key = ?
`, b.Value, b.Key)
		} else {
			_, _ = db.Exec(`
DELETE FROM app_settings
WHERE setting_key = ?
`, b.Key)
		}
	}
}

func upsertAppSetting(db *sql.DB, key, value string) error {
	valueType := "text"
	if key == "church.brand_image_included" {
		valueType = "boolean"
	}
	if key == "church.homepage_url" {
		valueType = "url"
	}
	if key == "church.default_footer_text" {
		valueType = "multiline"
	}

	_, err := db.Exec(`
INSERT INTO app_settings (setting_key, setting_value, value_type, is_secret, setting_group, updated_at)
VALUES (?, ?, ?, 0, 'church', datetime('now', 'localtime'))
ON CONFLICT(setting_key) DO UPDATE SET
	setting_value = excluded.setting_value,
	value_type = excluded.value_type,
	updated_at = excluded.updated_at
`, key, value, valueType)
	if err != nil {
		return fmt.Errorf("failed to upsert app_setting %s: %w", key, err)
	}
	return nil
}

func boolToText(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func checkFiles(paths ...string) {
	fmt.Println("=== File Check ===")
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			fmt.Printf("[OK] %s (%d bytes)\n", p, info.Size())
		} else {
			fmt.Printf("[MISS] %s\n", p)
		}
	}
}

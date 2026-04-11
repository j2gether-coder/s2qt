package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"s2qt/service"
	"s2qt/util"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	db *sql.DB

	outputFileService *service.OutputFileService
	cryptoSvc         *service.CryptoService
	settingsSvc       *service.SettingsService
	historySvc        *service.HistoryService
}

func NewApp() *App {
	return &App{
		outputFileService: service.NewOutputFileService(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.outputFileService.SetContext(ctx)

	if err := a.initLocalServices(); err != nil {
		panic(err)
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		_ = a.db.Close()
	}
}

func (a *App) initLocalServices() error {
	paths, err := util.GetAppPaths()
	if err != nil {
		return fmt.Errorf("failed to get app paths: %w", err)
	}

	db, err := service.OpenSQLite(paths.DBFile)
	if err != nil {
		return fmt.Errorf("failed to open sqlite: %w", err)
	}

	if err := service.InitSQLite(db); err != nil {
		return fmt.Errorf("failed to init sqlite: %w", err)
	}

	cryptoSvc, err := service.NewCryptoService(paths.SecurityFile)
	if err != nil {
		return fmt.Errorf("failed to init crypto service: %w", err)
	}

	a.db = db
	a.cryptoSvc = cryptoSvc
	a.settingsSvc = service.NewSettingsService(db, cryptoSvc)
	a.historySvc = service.NewHistoryService(db)

	return nil
}

func (a *App) resolveDBPath() (string, error) {
	// util.GetAppPaths() 구조를 따르는 편이 기존 프로젝트와 더 잘 맞음
	paths, err := util.GetAppPaths()
	if err != nil {
		return "", err
	}

	dbDir := filepath.Dir(paths.DBFile)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return "", err
	}

	return paths.DBFile, nil
}

func (a *App) SelectTextFile() (string, error) {
	if a.ctx == nil {
		return "", errors.New("context is not initialized")
	}

	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "텍스트 파일 선택",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "텍스트 파일",
				Pattern:     "*.txt;*.md",
			},
		},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(path), nil
}

func (a *App) SelectAudioFile() (string, error) {
	if a.ctx == nil {
		return "", errors.New("context is not initialized")
	}

	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "오디오 파일 선택",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "오디오 파일",
				Pattern:     "*.mp3;*.wav;*.m4a;*.aac;*.flac;*.ogg",
			},
		},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(path), nil
}

func (a *App) LoadTextFile(path string) (string, error) {
	svc := service.NewTxtService()
	return svc.LoadTextFile(path)
}

// video URL 메타 조회 (yt-dlp --dump-json)
func (a *App) GetVideoMeta(url string) (*service.VideoMeta, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, errors.New("URL이 비어 있습니다")
	}

	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return service.FetchVideoMeta(paths.YtDlpExe, url)
}

func (a *App) RunSourcePrepare(req service.SourcePrepareRequest) (*service.SourcePrepareResult, error) {
	pipeline, err := service.NewPipelineService(nil)
	if err != nil {
		return nil, err
	}
	return pipeline.RunSourcePrepare(&req)
}

func (a *App) RunLLMPrepare(req service.LLMPrepareRequest) (*service.LLMPrepareResult, error) {
	pipeline, err := service.NewPipelineService(nil)
	if err != nil {
		return nil, err
	}
	return pipeline.RunLLMPrepare(&req)
}

func (a *App) BuildQTPrompt(req service.LLMPrepareRequest) (string, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return "", err
	}

	rawBytes, err := os.ReadFile(paths.TempTxt)
	if err != nil {
		return "", err
	}

	rawText := strings.TrimSpace(string(rawBytes))
	if rawText == "" {
		return "", errors.New("temp.txt 내용이 비어 있습니다")
	}

	meta := service.QTMeta{
		Title:      strings.TrimSpace(req.Title),
		BibleText:  strings.TrimSpace(req.BibleText),
		Hymn:       strings.TrimSpace(req.Hymn),
		Preacher:   strings.TrimSpace(req.Preacher),
		ChurchName: strings.TrimSpace(req.ChurchName),
		SermonDate: strings.TrimSpace(req.SermonDate),
		SourceURL:  strings.TrimSpace(req.SourceURL),
		RawText:    rawText,
		Audience:   strings.TrimSpace(req.Audience),
	}

	if meta.Title == "" {
		return "", errors.New("제목이 비어 있습니다")
	}
	if meta.BibleText == "" {
		return "", errors.New("본문 성구가 비어 있습니다")
	}
	if meta.Audience == "" {
		return "", errors.New("대상 연령층이 비어 있습니다")
	}

	llmSvc := &service.LLMService{}
	return llmSvc.BuildPrompt(meta), nil
}

func (a *App) SaveManualLLMResult(jsonText string) error {
	paths, err := util.GetAppPaths()
	if err != nil {
		return err
	}

	jsonText = strings.TrimSpace(jsonText)
	if jsonText == "" {
		return errors.New("저장할 JSON 결과가 비어 있습니다")
	}

	var js any
	if err := json.Unmarshal([]byte(jsonText), &js); err != nil {
		return errors.New("유효한 JSON 형식이 아닙니다")
	}

	return os.WriteFile(paths.TempJson, []byte(jsonText), 0644)
}

func (a *App) LoadQTStep2Data() (*service.QTStep2Data, error) {
	svc, err := service.NewQTStep2Service()
	if err != nil {
		return nil, err
	}
	return svc.Load()
}

func (a *App) SaveQTStep2Data(req service.QTStep2Data) error {
	svc, err := service.NewQTStep2Service()
	if err != nil {
		return err
	}
	return svc.Save(&req)
}

func (a *App) PreviewQTStep2HTML(req service.QTStep2Data) (*service.QTStep2PreviewResult, error) {
	svc, err := service.NewQTStep2Service()
	if err != nil {
		return nil, err
	}

	htmlFile, err := svc.BuildHTML(&req)
	if err != nil {
		return nil, err
	}

	return &service.QTStep2PreviewResult{
		Success:  true,
		Message:  "temp.html 생성이 완료되었습니다.",
		HtmlFile: htmlFile,
	}, nil
}

func (a *App) OpenTempHTMLPreview() error {
	paths, err := util.GetAppPaths()
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(paths.TempHtml)
	if err != nil {
		return err
	}

	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("temp.html 파일이 없습니다: %w", err)
	}

	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", absPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("기본 브라우저로 미리보기 열기 실패: %w", err)
	}

	return nil
}

func (a *App) RunQTStep3(req service.QTStep3Request) (*service.QTStep3Result, error) {
	svc, err := service.NewQTStep3Service()
	if err != nil {
		return nil, err
	}
	return svc.Run(&req)
}

func (a *App) GeneratePNG(dpi int) (*service.PNGGenerateResult, error) {
	pngSvc, err := service.NewPNGService()
	if err != nil {
		return nil, err
	}
	return pngSvc.GenerateFromTempHTML(dpi)
}

func (a *App) OpenGeneratedFile(filePath string) error {
	return a.outputFileService.OpenGeneratedFile(filePath)
}

func (a *App) SaveGeneratedFile(filePath, audienceID, formatKey string) (string, error) {
	return a.outputFileService.SaveGeneratedFile(filePath, audienceID, formatKey)
}

//
// Settings bindings
//

func (a *App) LoadAppSettingsByGroup(group string) ([]service.SettingItem, error) {
	if a.settingsSvc == nil {
		return nil, fmt.Errorf("settings service is not initialized")
	}
	return a.settingsSvc.GetSettingsByGroup(group)
}

func (a *App) SaveAppSettings(items []service.SettingItem) error {
	if a.settingsSvc == nil {
		return fmt.Errorf("settings service is not initialized")
	}
	return a.settingsSvc.SaveSettings(items)
}

func (a *App) SaveSecretSettingWithPin(key string, plainValue string, valueType string, group string, pin string) error {
	if a.settingsSvc == nil {
		return fmt.Errorf("settings service is not initialized")
	}
	return a.settingsSvc.SaveSecretSettingWithPin(key, plainValue, valueType, group, pin)
}

func (a *App) HasSecretValue(key string) (bool, error) {
	if a.settingsSvc == nil {
		return false, fmt.Errorf("settings service is not initialized")
	}
	return a.settingsSvc.HasSecretSetting(key)
}

//
// Security / PIN bindings
//

func (a *App) IsPinEnabled() (bool, error) {
	if a.cryptoSvc == nil {
		return false, fmt.Errorf("crypto service is not initialized")
	}
	return a.cryptoSvc.IsPinEnabled(), nil
}

func (a *App) SetupPin(pin string) error {
	if a.cryptoSvc == nil {
		return fmt.Errorf("crypto service is not initialized")
	}
	return a.cryptoSvc.SetupPin(pin)
}

func (a *App) ChangePin(oldPin string, newPin string) error {
	if a.cryptoSvc == nil {
		return fmt.Errorf("crypto service is not initialized")
	}
	return a.cryptoSvc.ChangePin(oldPin, newPin)
}

func (a *App) VerifyPin(pin string) (bool, error) {
	if a.cryptoSvc == nil {
		return false, fmt.Errorf("crypto service is not initialized")
	}
	return a.cryptoSvc.VerifyPin(pin)
}

func (a *App) GetPinLength() (int, error) {
	if a.cryptoSvc == nil {
		return 6, fmt.Errorf("crypto service is not initialized")
	}
	return a.cryptoSvc.GetPinLength(), nil
}

//
// History bindings
//

func (a *App) SaveHistory(req service.SaveHistoryRequest) (int64, error) {
	if a.historySvc == nil {
		return 0, fmt.Errorf("history service is not initialized")
	}
	return a.historySvc.SaveHistory(req)
}

func (a *App) ListHistory() ([]service.HistoryMaster, error) {
	if a.historySvc == nil {
		return nil, fmt.Errorf("history service is not initialized")
	}
	return a.historySvc.ListHistory()
}

func (a *App) GetHistory(historyID int64) (service.HistoryMaster, error) {
	if a.historySvc == nil {
		return service.HistoryMaster{}, fmt.Errorf("history service is not initialized")
	}
	return a.historySvc.GetHistory(historyID)
}

func (a *App) GetHistoryStep1(historyID int64, audience string) (service.HistoryStep1, error) {
	if a.historySvc == nil {
		return service.HistoryStep1{}, fmt.Errorf("history service is not initialized")
	}
	return a.historySvc.GetHistoryStep1(historyID, audience)
}

func (a *App) DeleteHistory(historyID int64) error {
	if a.historySvc == nil {
		return fmt.Errorf("history service is not initialized")
	}
	return a.historySvc.DeleteHistory(historyID)
}

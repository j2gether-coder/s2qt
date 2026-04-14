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
	smtpSvc           *service.SMTPService
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
		return
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
		return err
	}

	db, err := service.OpenSQLite(paths.DBFile)
	if err != nil {
		return err
	}

	cryptoSvc, err := service.NewCryptoService(paths.SecurityFile)
	if err != nil {
		return err
	}

	a.db = db
	a.cryptoSvc = cryptoSvc
	a.settingsSvc = service.NewSettingsService(db, cryptoSvc)
	a.smtpSvc = service.NewSMTPService(a.settingsSvc)
	a.historySvc = service.NewHistoryService(db)

	return nil
}

func (a *App) resolveDBPath() (string, error) {
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

func (a *App) GetVideoMeta(url string) (*service.VideoMeta, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		err := errors.New("URL이 비어 있습니다")
		service.LogError("qt_prepare: video meta url empty")
		return nil, err
	}

	service.LogInfo("qt_prepare: video meta fetch started")

	paths, err := util.GetAppPaths()
	if err != nil {
		service.LogError("qt_prepare: get app paths failed: " + err.Error())
		return nil, err
	}

	meta, err := service.FetchVideoMeta(paths.YtDlpExe, url)
	if err != nil {
		service.LogError("qt_prepare: video meta fetch failed: " + err.Error())
		return nil, err
	}

	service.LogInfo("qt_prepare: video meta fetch completed")
	return meta, nil
}

func (a *App) PrepareRuntimeForInput(inputType string) (*service.UtilCheckResult, error) {
	inputType = strings.TrimSpace(strings.ToLower(inputType))

	_, err := service.StartEventLog(inputType)
	if err != nil {
		return nil, err
	}

	service.LogInfo("qt_prepare: runtime prepare requested")

	var result *service.UtilCheckResult

	switch inputType {
	case "text":
		result, err = service.CheckRuntimeForText()
	case "audio":
		result, err = service.CheckRuntimeForAudio(true)
	case "video", "url", "youtube":
		result, err = service.CheckRuntimeForVideo(true)
	default:
		err = fmt.Errorf("unsupported input type: %s", inputType)
	}

	if err != nil {
		service.LogError("qt_prepare: runtime prepare failed: " + err.Error())
		service.EndEventLog("FAILED")
		return nil, err
	}

	if result != nil && !result.OK {
		service.LogError("qt_prepare: runtime prepare failed: " + result.Message)
		service.EndEventLog("FAILED")
		return result, nil
	}

	service.LogInfo("qt_prepare: runtime prepare completed")
	return result, nil
}

func (a *App) RunSourcePrepare(req service.SourcePrepareRequest) (*service.SourcePrepareResult, error) {
	service.LogInfo("qt_prepare: source prepare started")

	pipeline, err := service.NewPipelineService(nil)
	if err != nil {
		service.LogError("qt_prepare: pipeline service create failed: " + err.Error())
		return nil, err
	}

	result, err := pipeline.RunSourcePrepare(&req)
	if err != nil {
		service.LogError("qt_prepare: source prepare failed: " + err.Error())
		service.EndEventLog("FAILED")
		return nil, err
	}

	service.LogInfo("qt_prepare: source prepare completed")
	return result, nil
}

func (a *App) RunLLMPrepare(req service.LLMPrepareRequest) (*service.LLMPrepareResult, error) {
	service.LogInfo("step1: llm prepare started")

	pipeline, err := service.NewPipelineService(nil)
	if err != nil {
		service.LogError("step1: pipeline service create failed: " + err.Error())
		return nil, err
	}

	result, err := pipeline.RunLLMPrepare(&req)
	if err != nil {
		service.LogError("step1: llm prepare failed: " + err.Error())
		return nil, err
	}

	service.LogInfo("step1: llm prepare completed")
	return result, nil
}

func (a *App) BuildQTPrompt(req service.LLMPrepareRequest) (string, error) {
	service.LogInfo("step1: build prompt started")

	paths, err := util.GetAppPaths()
	if err != nil {
		service.LogError("step1: get app paths failed: " + err.Error())
		return "", err
	}

	rawBytes, err := os.ReadFile(paths.TempTxt)
	if err != nil {
		service.LogError("step1: temp.txt read failed: " + err.Error())
		return "", err
	}

	rawText := strings.TrimSpace(string(rawBytes))
	if rawText == "" {
		err := errors.New("temp.txt 내용이 비어 있습니다")
		service.LogError("step1: temp.txt empty")
		return "", err
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
		err := errors.New("제목이 비어 있습니다")
		service.LogError("step1: title empty")
		return "", err
	}
	if meta.BibleText == "" {
		err := errors.New("본문 성구가 비어 있습니다")
		service.LogError("step1: bible text empty")
		return "", err
	}
	if meta.Audience == "" {
		err := errors.New("대상 연령층이 비어 있습니다")
		service.LogError("step1: audience empty")
		return "", err
	}

	llmSvc := &service.LLMService{}
	prompt := llmSvc.BuildPrompt(meta)

	service.LogInfo("step1: build prompt completed")
	return prompt, nil
}

func (a *App) SaveManualLLMResult(jsonText string) error {
	service.LogInfo("step1: manual result save started")

	paths, err := util.GetAppPaths()
	if err != nil {
		service.LogError("step1: get app paths failed: " + err.Error())
		return err
	}

	jsonText = strings.TrimSpace(jsonText)
	if jsonText == "" {
		err := errors.New("저장할 JSON 결과가 비어 있습니다")
		service.LogError("step1: manual result empty")
		return err
	}

	var js any
	if err := json.Unmarshal([]byte(jsonText), &js); err != nil {
		service.LogError("step1: invalid json result")
		return errors.New("유효한 JSON 형식이 아닙니다")
	}

	if err := os.WriteFile(paths.TempJson, []byte(jsonText), 0o644); err != nil {
		service.LogError("step1: temp.json write failed: " + err.Error())
		return err
	}

	service.LogInfo("step1: manual result save completed")
	return nil
}

func (a *App) LoadQTStep2Data() (*service.QTStep2Data, error) {
	service.LogInfo("step2: load started")

	svc, err := service.NewQTStep2Service()
	if err != nil {
		service.LogError("step2: service create failed: " + err.Error())
		return nil, err
	}

	data, err := svc.Load()
	if err != nil {
		service.LogError("step2: load failed: " + err.Error())
		return nil, err
	}

	service.LogInfo("step2: load completed")
	return data, nil
}

func (a *App) SaveQTStep2Data(req service.QTStep2Data) error {
	service.LogInfo("step2: save started")

	svc, err := service.NewQTStep2Service()
	if err != nil {
		service.LogError("step2: service create failed: " + err.Error())
		return err
	}

	if err := svc.Save(&req); err != nil {
		service.LogError("step2: save failed: " + err.Error())
		return err
	}

	service.LogInfo("step2: save completed")
	return nil
}

func (a *App) PreviewQTStep2HTML(req service.QTStep2Data) (*service.QTStep2PreviewResult, error) {
	service.LogInfo("step2: preview started")

	svc, err := service.NewQTStep2Service()
	if err != nil {
		service.LogError("step2: service create failed: " + err.Error())
		return nil, err
	}

	htmlFile, err := svc.BuildHTML(&req)
	if err != nil {
		service.LogError("step2: preview build failed: " + err.Error())
		return nil, err
	}

	service.LogInfo("step2: preview completed")

	return &service.QTStep2PreviewResult{
		Success:  true,
		Message:  "temp.html 생성이 완료되었습니다.",
		HtmlFile: htmlFile,
	}, nil
}

func (a *App) OpenTempHTMLPreview() error {
	service.LogInfo("step2: open temp html preview started")

	paths, err := util.GetAppPaths()
	if err != nil {
		service.LogError("step2: get app paths failed: " + err.Error())
		return err
	}

	absPath, err := filepath.Abs(paths.TempHtml)
	if err != nil {
		service.LogError("step2: temp html abs path failed: " + err.Error())
		return err
	}

	if _, err := os.Stat(absPath); err != nil {
		service.LogError("step2: temp.html not found: " + err.Error())
		return fmt.Errorf("temp.html 파일이 없습니다: %w", err)
	}

	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", absPath)
	if err := cmd.Start(); err != nil {
		service.LogError("step2: browser preview open failed: " + err.Error())
		return fmt.Errorf("기본 브라우저로 미리보기 열기 실패: %w", err)
	}

	service.LogInfo("step2: open temp html preview completed")
	return nil
}

func (a *App) RunQTStep3(req service.QTStep3Request) (*service.QTStep3Result, error) {
	service.LogInfo("step3: output generation started")

	svc, err := service.NewQTStep3Service()
	if err != nil {
		service.LogError("step3: service create failed: " + err.Error())
		return nil, err
	}

	result, err := svc.Run(&req)
	if err != nil {
		service.LogError("step3: output generation failed: " + err.Error())
		return nil, err
	}

	service.LogInfo("step3: output generation completed")
	return result, nil
}

func (a *App) GeneratePNG(dpi int) (*service.PNGGenerateResult, error) {
	service.LogInfo("step3: png generation started")

	pngSvc, err := service.NewPNGService()
	if err != nil {
		service.LogError("step3: png service create failed: " + err.Error())
		return nil, err
	}

	result, err := pngSvc.GenerateFromTempHTML(dpi)
	if err != nil {
		service.LogError("step3: png generation failed: " + err.Error())
		return nil, err
	}

	service.LogInfo("step3: png generation completed")
	return result, nil
}

func (a *App) OpenGeneratedFile(filePath string) error {
	service.LogInfo("step3: open generated file requested")

	if err := a.outputFileService.OpenGeneratedFile(filePath); err != nil {
		service.LogError("step3: open generated file failed: " + err.Error())
		return err
	}

	service.LogInfo("step3: open generated file completed")
	return nil
}

func (a *App) SaveGeneratedFile(filePath, audienceID, formatKey string) (string, error) {
	service.LogInfo("step3: save generated file requested format=" + strings.TrimSpace(formatKey))

	savedPath, err := a.outputFileService.SaveGeneratedFile(filePath, audienceID, formatKey)
	if err != nil {
		service.LogError("step3: save generated file failed: " + err.Error())
		return "", err
	}

	service.LogInfo("step3: save generated file completed")
	return savedPath, nil
}

func (a *App) FinishCurrentRun() error {
	service.LogInfo("step3: flow end requested")
	service.EndEventLog("COMPLETED")
	return nil
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

	service.LogInfo("step2: history save started")

	historyID, err := a.historySvc.SaveHistory(req)
	if err != nil {
		service.LogError("step2: history save failed: " + err.Error())
		return 0, err
	}

	service.LogInfo("step2: history save completed")
	return historyID, nil
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

func (a *App) GetHistoryQTJSON(historyID int64, audience string) (service.HistoryQTJSON, error) {
	if a.historySvc == nil {
		return service.HistoryQTJSON{}, fmt.Errorf("history service is not initialized")
	}
	return a.historySvc.GetHistoryQTJSON(historyID, audience)
}

func (a *App) DeleteHistory(historyID int64) error {
	if a.historySvc == nil {
		return fmt.Errorf("history service is not initialized")
	}
	return a.historySvc.DeleteHistory(historyID)
}

func (a *App) PrepareReworkFromHistory(historyID int64, audience string) (service.ReworkPrepareResponse, error) {
	if a.historySvc == nil {
		return service.ReworkPrepareResponse{}, fmt.Errorf("history service is not initialized")
	}
	return a.historySvc.PrepareReworkFromHistory(historyID, audience)
}

func (a *App) LoadGuideDocument(sectionId string) (string, error) {
	return service.LoadGuideDocument(sectionId)
}

func (a *App) TestSMTPSettings(pin string) (*service.SMTPTestResult, error) {
	if a.smtpSvc == nil {
		return nil, fmt.Errorf("smtp service is not initialized")
	}
	return a.smtpSvc.TestSendToSelf(pin)
}

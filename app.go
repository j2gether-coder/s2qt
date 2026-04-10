package main

import (
	"context"
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
	ctx               context.Context
	outputFileService *service.OutputFileService
}

func NewApp() *App {
	return &App{
		outputFileService: service.NewOutputFileService(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.outputFileService.SetContext(ctx)
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

// TODO(step3-legacy-output):
// 기존 출력 열기/다른 저장 방식은 현재 Step3에서 사용하지 않음.
// 필요 시 이후 구조 정리 후 재사용 검토.
// func (a *App) OpenOutputFile(path string) error {
// 	path = strings.TrimSpace(path)
// 	if path == "" {
// 		return errors.New("파일 경로가 비어 있습니다")
// 	}
//
// 	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
// 	if err := cmd.Start(); err != nil {
// 		return fmt.Errorf("파일 열기 실패: %w", err)
// 	}
// 	return nil
// }
//
// func (a *App) SaveQTOutputAs(req service.SaveOutputAsRequest) (*service.SaveOutputAsResult, error) {
// 	svc, err := service.NewFileSaveService()
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	return svc.SaveOutputAs(&req, func(defaultFilename string, filters []service.DialogFileFilter) (string, error) {
// 		return "", nil
// 	})
// }

func (a *App) OpenGeneratedFile(filePath string) error {
	return a.outputFileService.OpenGeneratedFile(filePath)
}

func (a *App) SaveGeneratedFile(filePath, audienceID, formatKey string) (string, error) {
	return a.outputFileService.SaveGeneratedFile(filePath, audienceID, formatKey)
}

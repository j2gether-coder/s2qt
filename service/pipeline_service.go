package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"s2qt/util"
)

type PipelineService struct {
	Paths      *util.AppPaths
	OnProgress func(stage, message string)
}

func NewPipelineService(onProgress func(stage, message string)) (*PipelineService, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return &PipelineService{
		Paths:      paths,
		OnProgress: onProgress,
	}, nil
}

func (s *PipelineService) progress(stage, message string) {
	if s.OnProgress != nil {
		s.OnProgress(stage, message)
	}
}

func (s *PipelineService) cleanupSourcePrepareTempFiles() {
	files := []string{
		s.Paths.TempTxt,
		s.Paths.TempVideo,
		s.Paths.TempWav,
	}
	for _, f := range files {
		if strings.TrimSpace(f) != "" {
			_ = os.Remove(f)
		}
	}
}

func (s *PipelineService) cleanupLLMTempFiles() {
	if strings.TrimSpace(s.Paths.TempJson) != "" {
		_ = os.Remove(s.Paths.TempJson)
	}
}

func (s *PipelineService) saveTempText(rawText string) error {
	rawText = strings.TrimSpace(rawText)
	if rawText == "" {
		return fmt.Errorf("raw text가 비어 있습니다")
	}
	return os.WriteFile(s.Paths.TempTxt, []byte(rawText), 0644)
}

func (s *PipelineService) saveTempJSON(jsonText string) error {
	jsonText = strings.TrimSpace(jsonText)
	if jsonText == "" {
		return fmt.Errorf("json 결과가 비어 있습니다")
	}
	return os.WriteFile(s.Paths.TempJson, []byte(jsonText), 0644)
}

func (s *PipelineService) RunSourcePrepare(req *SourcePrepareRequest) (*SourcePrepareResult, error) {
	if req == nil {
		return nil, fmt.Errorf("source prepare request가 nil입니다")
	}

	steps := []string{}
	addStep := func(stage, msg string) {
		steps = append(steps, fmt.Sprintf("[%s] %s", stage, msg))
		s.progress(stage, msg)
	}

	addStep("init", "QT 준비 시작")
	s.cleanupSourcePrepareTempFiles()

	var rawText string
	var err error

	switch strings.TrimSpace(req.SourceType) {
	case "text":
		txtSvc := NewTxtService()
		addStep("text", "텍스트 원문 확인 중")
		rawText, err = txtSvc.ResolveRawText(req.InputMode, req.SourcePath, req.TextContent)

	case "audio":
		audioSvc, svcErr := NewAudioService(s.OnProgress)
		if svcErr != nil {
			return &SourcePrepareResult{
				Success: false,
				Message: svcErr.Error(),
				Status:  "FAILED",
				Steps:   steps,
			}, svcErr
		}
		addStep("audio", "오디오 원문 추출 중")
		rawText, err = audioSvc.ResolveRawText(req.SourcePath)

	case "video":
		videoSvc, svcErr := NewVideoService(s.OnProgress)
		if svcErr != nil {
			return &SourcePrepareResult{
				Success: false,
				Message: svcErr.Error(),
				Status:  "FAILED",
				Steps:   steps,
			}, svcErr
		}

		if strings.TrimSpace(req.InputMode) != "url" {
			err = fmt.Errorf("video는 현재 url 입력만 지원합니다")
		} else {
			addStep("video", "동영상 원문 추출 중")
			result, runErr := videoSvc.Run(req.SourceURL)
			if runErr != nil {
				err = runErr
			} else {
				rawText = result.TranscriptText
				if strings.TrimSpace(rawText) == "" {
					rawTextBytes, readErr := os.ReadFile(s.Paths.TempTxt)
					if readErr != nil {
						err = fmt.Errorf("video temp.txt 읽기 실패: %w", readErr)
					} else {
						rawText = strings.TrimSpace(string(rawTextBytes))
					}
				}
			}
		}

	default:
		err = fmt.Errorf("지원하지 않는 source type: %s", req.SourceType)
	}

	if err != nil {
		addStep("error", err.Error())
		return &SourcePrepareResult{
			Success:    false,
			Message:    err.Error(),
			Status:     "FAILED",
			SourceType: req.SourceType,
			Steps:      steps,
		}, err
	}

	rawText = strings.TrimSpace(rawText)
	if rawText == "" {
		err = fmt.Errorf("추출된 원문 텍스트가 비어 있습니다")
		addStep("error", err.Error())
		return &SourcePrepareResult{
			Success:    false,
			Message:    err.Error(),
			Status:     "FAILED",
			SourceType: req.SourceType,
			Steps:      steps,
		}, err
	}

	addStep("save", "temp.txt 저장 중")
	if err := s.saveTempText(rawText); err != nil {
		addStep("error", err.Error())
		return &SourcePrepareResult{
			Success:    false,
			Message:    err.Error(),
			Status:     "FAILED",
			SourceType: req.SourceType,
			Steps:      steps,
		}, err
	}

	addStep("done", "QT 준비 완료 (temp.txt 생성)")
	return &SourcePrepareResult{
		Success:    true,
		Message:    "QT 준비가 완료되었습니다.",
		Status:     "COMPLETED",
		SourceType: req.SourceType,
		RawText:    rawText,
		TxtFile:    s.Paths.TempTxt,
		Steps:      steps,
	}, nil
}

func (s *PipelineService) RunLLMPrepare(req *LLMPrepareRequest) (*LLMPrepareResult, error) {
	if req == nil {
		return nil, fmt.Errorf("llm prepare request가 nil입니다")
	}

	steps := []string{}
	addStep := func(stage, msg string) {
		steps = append(steps, fmt.Sprintf("[%s] %s", stage, msg))
		s.progress(stage, msg)
	}

	addStep("init", "LLM 준비 시작")
	s.cleanupLLMTempFiles()

	rawBytes, err := os.ReadFile(s.Paths.TempTxt)
	if err != nil {
		addStep("error", "temp.txt 읽기 실패")
		return &LLMPrepareResult{
			Success: false,
			Message: fmt.Sprintf("temp.txt 읽기 실패: %v", err),
			Status:  "FAILED",
			Steps:   steps,
		}, err
	}

	rawText := strings.TrimSpace(string(rawBytes))
	if rawText == "" {
		err = fmt.Errorf("temp.txt 내용이 비어 있습니다")
		addStep("error", err.Error())
		return &LLMPrepareResult{
			Success: false,
			Message: err.Error(),
			Status:  "FAILED",
			Steps:   steps,
		}, err
	}

	meta := QTMeta{
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

	if strings.TrimSpace(meta.Title) == "" {
		return nil, fmt.Errorf("제목이 비어 있습니다")
	}
	if strings.TrimSpace(meta.BibleText) == "" {
		return nil, fmt.Errorf("본문 성구가 비어 있습니다")
	}
	if strings.TrimSpace(meta.Audience) == "" {
		return nil, fmt.Errorf("대상 연령층이 비어 있습니다")
	}

	addStep("llm", "LLM 서비스 초기화 중")
	llm, err := NewLLMService()
	if err != nil {
		addStep("error", err.Error())
		return &LLMPrepareResult{
			Success: false,
			Message: err.Error(),
			Status:  "FAILED",
			Steps:   steps,
		}, err
	}

	addStep("llm", "QT JSON 생성 중")
	jsonText, err := llm.GenerateQTJSON(meta)
	if err != nil {
		addStep("error", err.Error())
		return &LLMPrepareResult{
			Success: false,
			Message: err.Error(),
			Status:  "FAILED",
			Steps:   steps,
		}, err
	}

	if !json.Valid([]byte(jsonText)) {
		err = fmt.Errorf("LLM 결과가 유효한 JSON이 아닙니다")
		addStep("error", err.Error())
		return &LLMPrepareResult{
			Success: false,
			Message: err.Error(),
			Status:  "FAILED",
			Steps:   steps,
		}, err
	}

	addStep("save", "temp.json 저장 중")
	if err := s.saveTempJSON(jsonText); err != nil {
		addStep("error", err.Error())
		return &LLMPrepareResult{
			Success: false,
			Message: err.Error(),
			Status:  "FAILED",
			Steps:   steps,
		}, err
	}

	addStep("done", "temp.json 생성 완료")
	return &LLMPrepareResult{
		Success:  true,
		Message:  "QT JSON 생성이 완료되었습니다.",
		Status:   "COMPLETED",
		JSONFile: s.Paths.TempJson,
		JSONText: jsonText,
		Steps:    steps,
	}, nil
}

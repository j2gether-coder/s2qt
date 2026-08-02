package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

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

func countPreparedText(text string) (charCount, wordCount, lineCount, estimatedTokens int) {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	charCount = utf8RuneCount(normalized)
	wordCount = len(strings.Fields(normalized))
	if strings.TrimSpace(normalized) == "" {
		lineCount = 0
	} else {
		lineCount = len(strings.Split(normalized, "\n"))
	}
	estimatedTokens = charCount / 2
	if estimatedTokens == 0 && charCount > 0 {
		estimatedTokens = 1
	}
	return
}

func formatHMS(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	totalSeconds := ms / 1000
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func logSourcePrepareSummary(sourceType, status, txtFile string, metrics SourcePrepareMetrics) {
	LogInfo(fmt.Sprintf(
		"qt_prepare: [result] source=%s status=%s chars=%d words=%d lines=%d estimated_tokens=%d transcribe_model=%s fallback_model=%s fallback_used=%t",
		strings.TrimSpace(sourceType),
		strings.TrimSpace(status),
		metrics.CharCount,
		metrics.WordCount,
		metrics.LineCount,
		metrics.EstimatedTokens,
		strings.TrimSpace(metrics.TranscribeModel),
		strings.TrimSpace(metrics.FallbackModel),
		metrics.FallbackUsed,
	))
	if strings.TrimSpace(metrics.RetryReason) != "" {
		LogWarn(fmt.Sprintf("qt_prepare: [result] retry_reason=%s", strings.TrimSpace(metrics.RetryReason)))
	}
	LogInfo(fmt.Sprintf(
		"qt_prepare: [time] download=%s convert=%s transcribe=%s total=%s",
		formatHMS(metrics.DownloadMs),
		formatHMS(metrics.ConvertMs),
		formatHMS(metrics.TranscribeMs),
		formatHMS(metrics.TotalMs),
	))
	LogInfo(fmt.Sprintf("qt_prepare: [file] temp_txt=%s", strings.TrimSpace(txtFile)))
}

func (s *PipelineService) RunSourcePrepare(req *SourcePrepareRequest) (*SourcePrepareResult, error) {
	if req == nil {
		return nil, fmt.Errorf("source prepare request가 nil입니다")
	}
	prepareStart := time.Now()

	steps := []string{}
	addStep := func(stage, msg string) {
		steps = append(steps, fmt.Sprintf("[%s] %s", stage, msg))
		s.progress(stage, msg)
	}

	addStep("init", "QT 준비 시작")
	s.cleanupSourcePrepareTempFiles()

	var rawText string
	var err error
	metrics := SourcePrepareMetrics{}

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
		if err == nil && audioSvc.LastTranscription != nil {
			metrics.TranscribeMs = audioSvc.LastTranscription.TotalMs
			metrics.TranscribeModel = audioSvc.LastTranscription.ModelName
			metrics.FallbackModel = audioSvc.LastTranscription.FallbackModelName
			metrics.FallbackUsed = audioSvc.LastTranscription.FallbackUsed
			metrics.RetryReason = audioSvc.LastTranscription.RetryReason
		}

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
				metrics = SourcePrepareMetrics{
					DownloadMs:      result.DownloadMs,
					ConvertMs:       result.ConvertMs,
					TranscribeMs:    result.TranscribeMs,
					TotalMs:         result.TotalMs,
					CharCount:       result.CharCount,
					WordCount:       result.WordCount,
					LineCount:       result.LineCount,
					EstimatedTokens: result.EstimatedTokens,
					TranscribeModel: result.TranscribeModel,
					FallbackModel:   result.FallbackModel,
					FallbackUsed:    result.FallbackUsed,
					RetryReason:     result.RetryReason,
				}
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

	if metrics.CharCount == 0 {
		charCount, wordCount, lineCount, estimatedTokens := countPreparedText(rawText)
		metrics.CharCount = charCount
		metrics.WordCount = wordCount
		metrics.LineCount = lineCount
		metrics.EstimatedTokens = estimatedTokens
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
	if metrics.TotalMs == 0 {
		metrics.TotalMs = time.Since(prepareStart).Milliseconds()
	}
	logSourcePrepareSummary(req.SourceType, "COMPLETED", s.Paths.TempTxt, metrics)

	return &SourcePrepareResult{
		Success:    true,
		Message:    "QT 준비가 완료되었습니다.",
		Status:     "COMPLETED",
		SourceType: req.SourceType,
		RawText:    rawText,
		TxtFile:    s.Paths.TempTxt,
		Steps:      steps,
		Metrics:    metrics,
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

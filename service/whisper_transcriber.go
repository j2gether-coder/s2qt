package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"s2qt/util"
)

type WhisperTranscriptionResult struct {
	Text              string
	ModelPath         string
	ModelName         string
	FallbackModelPath string
	FallbackModelName string
	FallbackUsed      bool
	RetryReason       string
	PrimaryMs         int64
	FallbackMs        int64
	TotalMs           int64
	Log               string
}

type WhisperTranscriber struct {
	Paths      *util.AppPaths
	OnProgress func(stage, message string)
}

func NewWhisperTranscriber(paths *util.AppPaths, onProgress func(stage, message string)) *WhisperTranscriber {
	return &WhisperTranscriber{
		Paths:      paths,
		OnProgress: onProgress,
	}
}

func (t *WhisperTranscriber) progress(stage, message string) {
	if t.OnProgress != nil {
		t.OnProgress(stage, message)
	}
}

func (t *WhisperTranscriber) CheckRequiredFiles() error {
	required := []string{
		t.Paths.WhisperExe,
		t.Paths.WhisperModel,
	}
	if t.hasFallbackModel() {
		required = append(required, t.Paths.WhisperFallbackModel)
	}

	for _, f := range required {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("필수 파일이 없습니다: %s", f)
		}
	}
	return nil
}

func (t *WhisperTranscriber) Transcribe(inputAudioPath string) (*WhisperTranscriptionResult, error) {
	start := time.Now()
	primaryName := filepath.Base(t.Paths.WhisperModel)

	LogInfo(fmt.Sprintf("qt_prepare: [transcribe] model=%s", primaryName))
	t.progress("transcribe", fmt.Sprintf("전사 모델: %s", primaryName))

	primaryStart := time.Now()
	primaryOut, err := t.run(inputAudioPath, t.Paths.WhisperModel)
	primaryMs := time.Since(primaryStart).Milliseconds()
	if err != nil {
		return nil, err
	}

	text, retryReason, err := t.readTranscript()
	if err != nil {
		return nil, err
	}

	result := &WhisperTranscriptionResult{
		Text:      text,
		ModelPath: t.Paths.WhisperModel,
		ModelName: primaryName,
		PrimaryMs: primaryMs,
		TotalMs:   time.Since(start).Milliseconds(),
		Log:       fmt.Sprintf("=== whisper model=%s (%d ms) ===\n%s", primaryName, primaryMs, primaryOut),
	}

	if retryReason == "" {
		return result, nil
	}

	result.RetryReason = retryReason
	if !t.hasFallbackModel() {
		return nil, fmt.Errorf("전사 품질 확인 실패: %s. fallback 모델이 설정되어 있지 않습니다", retryReason)
	}

	fallbackName := filepath.Base(t.Paths.WhisperFallbackModel)
	LogWarn(fmt.Sprintf("qt_prepare: [transcribe] model=%s retry_reason=%s fallback_model=%s", primaryName, retryReason, fallbackName))
	t.progress("transcribe", fmt.Sprintf("전사 품질 재확인 중: %s 모델로 재전사합니다", fallbackName))

	fallbackStart := time.Now()
	fallbackOut, fallbackErr := t.run(inputAudioPath, t.Paths.WhisperFallbackModel)
	fallbackMs := time.Since(fallbackStart).Milliseconds()
	if fallbackErr != nil {
		return nil, fmt.Errorf("%s. %s 모델 재전사 실패: %w", retryReason, fallbackName, fallbackErr)
	}

	fallbackText, fallbackReason, readErr := t.readTranscript()
	if readErr != nil {
		return nil, readErr
	}
	if fallbackReason != "" {
		return nil, fmt.Errorf("전사 품질 확인 실패: %s=%s, %s=%s", primaryName, retryReason, fallbackName, fallbackReason)
	}

	result.Text = fallbackText
	result.FallbackModelPath = t.Paths.WhisperFallbackModel
	result.FallbackModelName = fallbackName
	result.FallbackUsed = true
	result.FallbackMs = fallbackMs
	result.TotalMs = time.Since(start).Milliseconds()
	result.Log += fmt.Sprintf("\n\n=== whisper fallback model=%s (%d ms) ===\n%s", fallbackName, fallbackMs, fallbackOut)

	LogInfo(fmt.Sprintf("qt_prepare: [transcribe] selected_model=%s fallback_used=true", fallbackName))
	return result, nil
}

func (t *WhisperTranscriber) hasFallbackModel() bool {
	return strings.TrimSpace(t.Paths.WhisperFallbackModel) != "" && t.Paths.WhisperFallbackModel != t.Paths.WhisperModel
}

func (t *WhisperTranscriber) run(inputAudioPath, modelPath string) (string, error) {
	_ = os.Remove(t.Paths.TempTxt)

	args := []string{
		"-m", modelPath,
		"-f", inputAudioPath,
		"-l", "ko",
		"-otxt",
		"-of", strings.TrimSuffix(t.Paths.TempTxt, ".txt"),
		"-pp",
	}

	lastPct := -1
	out, err := runHiddenCommandStreaming(func(line string) {
		if pct, ok := parseWhisperProgress(line); ok && pct != lastPct {
			lastPct = pct
			t.progress("transcribe", fmt.Sprintf("전사 중 %d%%", pct))
		}
	}, t.Paths.WhisperExe, args...)
	if err != nil {
		return out, fmt.Errorf("전사 실패: %w", err)
	}

	return out, nil
}

func (t *WhisperTranscriber) readTranscript() (string, string, error) {
	txtBytes, err := os.ReadFile(t.Paths.TempTxt)
	if err != nil {
		return "", "", fmt.Errorf("전사 결과 읽기 실패: %w", err)
	}

	text := strings.TrimSpace(string(txtBytes))
	if text == "" {
		return "", "전사 결과 텍스트가 비어 있습니다", nil
	}
	if ok, reason := looksHallucinatedTranscript(text); ok {
		return text, reason, nil
	}
	return text, "", nil
}

const (
	transcriptTailRepeatFailureRatio = 0.25
	transcriptMinTailRepeatLines     = 4
)

var transcriptLineCleaner = regexp.MustCompile(`[^\p{L}\p{N}]+`)

func looksHallucinatedTranscript(text string) (bool, string) {
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return true, "전사 결과가 비어 있습니다"
	}

	lines := transcriptLines(normalized)
	if len(lines) == 0 {
		return true, "전사 결과가 비어 있습니다"
	}

	tailRepeatLines := trailingRepeatedTranscriptLines(lines)
	tailRepeatRatio := float64(tailRepeatLines) / float64(len(lines))
	if tailRepeatLines >= transcriptMinTailRepeatLines && tailRepeatRatio >= transcriptTailRepeatFailureRatio {
		return true, fmt.Sprintf("마지막 구간에서 유사 문장이 전체 라인의 %.0f%% 반복되었습니다", tailRepeatRatio*100)
	}

	return false, ""
}

func transcriptLines(text string) []string {
	rawLines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func trailingRepeatedTranscriptLines(lines []string) int {
	if len(lines) == 0 {
		return 0
	}

	lastLine := lines[len(lines)-1]
	count := 1
	for i := len(lines) - 2; i >= 0; i-- {
		if !similarTranscriptLine(lines[i], lastLine) {
			break
		}
		count++
	}
	return count
}

func similarTranscriptLine(a, b string) bool {
	aTokens := canonicalTranscriptLineTokens(a)
	bTokens := canonicalTranscriptLineTokens(b)
	if len(aTokens) == 0 || len(bTokens) == 0 {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}

	if strings.Join(aTokens, " ") == strings.Join(bTokens, " ") {
		return utf8RuneCount(strings.Join(aTokens, "")) >= 8
	}

	overlap := transcriptTokenOverlap(aTokens, bTokens)
	shorter := len(aTokens)
	if len(bTokens) < shorter {
		shorter = len(bTokens)
	}
	return shorter >= 4 && float64(overlap)/float64(shorter) >= 0.7
}

func canonicalTranscriptLineTokens(line string) []string {
	cleaned := transcriptLineCleaner.ReplaceAllString(strings.ToLower(line), " ")
	return strings.Fields(cleaned)
}

func transcriptTokenOverlap(a, b []string) int {
	counts := map[string]int{}
	for _, token := range a {
		counts[token]++
	}
	overlap := 0
	for _, token := range b {
		if counts[token] > 0 {
			overlap++
			counts[token]--
		}
	}
	return overlap
}

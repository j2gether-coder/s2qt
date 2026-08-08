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
		// fallback 모델이 없으면 재전사할 방법이 없다.
		// 전사 자체는 끝났으므로 경고만 남기고 1차 결과를 그대로 사용한다.
		LogWarn(fmt.Sprintf("qt_prepare: [transcribe] model=%s retry_reason=%s fallback_model=none", primaryName, retryReason))
		t.progress("transcribe", "전사 품질 경고: "+retryReason)
		return result, nil
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
	if strings.TrimSpace(fallbackText) == "" {
		// 재전사 결과가 비면 1차 결과라도 살려서 돌려준다.
		// temp.txt는 재전사 때 비워졌으므로 1차 전사문을 다시 써 준다.
		LogWarn("qt_prepare: [transcribe] fallback transcript is empty, keep primary transcript")
		if writeErr := os.WriteFile(t.Paths.TempTxt, []byte(result.Text), 0o644); writeErr != nil {
			return nil, fmt.Errorf("1차 전사문 복원 실패: %w", writeErr)
		}
		result.TotalMs = time.Since(start).Milliseconds()
		return result, nil
	}
	if fallbackReason != "" {
		// 재전사 결과에도 반복 구간이 남아 있으면 경고만 남기고 진행한다.
		// 여기서 실패로 끊으면 사용자는 편집할 원문조차 얻지 못한다.
		result.RetryReason = fmt.Sprintf("%s=%s, %s=%s", primaryName, retryReason, fallbackName, fallbackReason)
		LogWarn("qt_prepare: [transcribe] fallback transcript still suspicious: " + result.RetryReason)
		t.progress("transcribe", "재전사 후에도 반복 구간이 남아 있습니다. 원문을 확인해 주세요")
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
	// 연속 반복 구간이 이 길이 이상이면 전체 길이와 무관하게 환각으로 본다.
	// 긴 설교는 전사 단위가 수백 개라 비율 기준만으로는 잡히지 않는다.
	transcriptAbsoluteRepeatUnits = 5

	// 짧은 전사에서는 비율 기준을 함께 본다.
	transcriptMinRepeatUnits     = 4
	transcriptRepeatFailureRatio = 0.25

	// 연속이 아니라 흩어져 반복되는 경우(A B A B A B)를 잡는 기준.
	transcriptDuplicateFailureRatio = 0.5
	transcriptMinUnitsForDuplicate  = 8

	transcriptReasonSampleRunes = 30
)

var (
	transcriptLineCleaner     = regexp.MustCompile(`[^\p{L}\p{N}]+`)
	transcriptSentenceSpliter = regexp.MustCompile(`[.!?…]+\s*`)
)

func looksHallucinatedTranscript(text string) (bool, string) {
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return true, "전사 결과가 비어 있습니다"
	}

	units := transcriptUnits(normalized)
	if len(units) == 0 {
		return true, "전사 결과가 비어 있습니다"
	}

	// 1) 완전히 같은 문장이 연달아 나오는 구간은 전사문 어디에 있어도, 비율과 무관하게 감지한다.
	//    (기존 로직은 마지막 문장과 이어지는 구간만 확인해서, 반복 뒤에 다른 문장이
	//     한 줄이라도 붙으면 환각을 놓쳤다. whisper의 반복 루프는 대부분 완전 일치다.)
	exactRun, exactSample, exactAtTail := longestRepeatedTranscriptRun(units, sameTranscriptLine)
	if exactRun >= transcriptAbsoluteRepeatUnits {
		return true, fmt.Sprintf(
			"%s 구간에서 같은 문장이 %d회 연속 반복되었습니다(전체의 %.0f%%): %q",
			transcriptRunPosition(exactAtTail), exactRun,
			float64(exactRun)/float64(len(units))*100, truncateTranscriptSample(exactSample),
		)
	}

	// 2) 조금씩 다른 문장이 반복되는 경우는 오탐 여지가 있어 비율 기준을 함께 본다.
	similarRun, similarSample, similarAtTail := longestRepeatedTranscriptRun(units, similarTranscriptLine)
	similarRatio := float64(similarRun) / float64(len(units))
	if similarRun >= transcriptMinRepeatUnits && similarRatio >= transcriptRepeatFailureRatio {
		return true, fmt.Sprintf(
			"%s 구간에서 유사 문장이 %d회 연속 반복되었습니다(전체의 %.0f%%): %q",
			transcriptRunPosition(similarAtTail), similarRun,
			similarRatio*100, truncateTranscriptSample(similarSample),
		)
	}

	// 3) 연속은 아니지만 같은 문장이 전사문 전반에 반복되는 경우.
	if len(units) >= transcriptMinUnitsForDuplicate {
		dupRatio, dupSample := duplicateTranscriptRatio(units)
		if dupRatio >= transcriptDuplicateFailureRatio {
			return true, fmt.Sprintf(
				"중복 문장이 전체의 %.0f%%를 차지합니다: %q",
				dupRatio*100, truncateTranscriptSample(dupSample),
			)
		}
	}

	return false, ""
}

// transcriptUnits는 전사문을 비교 단위(문장)로 나눈다.
// whisper는 보통 세그먼트마다 한 줄을 쓰지만, 반복 구간이 한 줄에 몰려 나오는
// 경우도 있어 문장 부호 기준으로 한 번 더 나눈다.
func transcriptUnits(text string) []string {
	units := []string{}
	for _, line := range transcriptLines(text) {
		for _, sentence := range transcriptSentenceSpliter.Split(line, -1) {
			sentence = strings.TrimSpace(sentence)
			if sentence == "" {
				continue
			}
			units = append(units, sentence)
		}
	}
	return units
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

// longestRepeatedTranscriptRun은 가장 긴 연속 반복 구간의 길이와 대표 문장,
// 그리고 그 구간이 전사문 끝에 붙어 있는지를 돌려준다.
// 비교는 구간의 첫 문장(anchor) 기준으로 해서 조금씩 흘러가는 문장이
// 하나의 반복 구간으로 이어 붙지 않도록 한다.
func longestRepeatedTranscriptRun(units []string, equal func(a, b string) bool) (int, string, bool) {
	if len(units) == 0 {
		return 0, "", false
	}

	best, bestSample, bestEnd := 1, units[0], 0
	anchor, run := 0, 1

	for i := 1; i < len(units); i++ {
		if equal(units[i], units[anchor]) {
			run++
		} else {
			anchor, run = i, 1
		}
		if run > best {
			best, bestSample, bestEnd = run, units[anchor], i
		}
	}

	return best, bestSample, bestEnd == len(units)-1
}

func transcriptRunPosition(atTail bool) string {
	if atTail {
		return "마지막"
	}
	return "중간"
}

// duplicateTranscriptRatio는 앞서 나온 문장과 같은 문장이 차지하는 비율을 돌려준다.
func duplicateTranscriptRatio(units []string) (float64, string) {
	seen := map[string]int{}
	duplicates := 0
	topKey, topCount := "", 0

	for _, unit := range units {
		key := strings.Join(canonicalTranscriptLineTokens(unit), " ")
		if key == "" {
			key = unit
		}
		seen[key]++
		if seen[key] > 1 {
			duplicates++
		}
		if seen[key] > topCount {
			topKey, topCount = unit, seen[key]
		}
	}

	if topCount < 2 {
		return 0, ""
	}
	return float64(duplicates) / float64(len(units)), topKey
}

func truncateTranscriptSample(sample string) string {
	sample = strings.TrimSpace(sample)
	runes := []rune(sample)
	if len(runes) <= transcriptReasonSampleRunes {
		return sample
	}
	return string(runes[:transcriptReasonSampleRunes]) + "..."
}

// sameTranscriptLine은 문장 부호/대소문자만 다른 완전 일치를 판단한다.
// 길이 조건을 두지 않는다. ("감사합니다", "네" 처럼 짧은 문구가 whisper 환각의
// 대표 패턴이라, 기존의 8자 이상 조건 때문에 정작 흔한 환각을 놓쳤다.)
func sameTranscriptLine(a, b string) bool {
	aTokens := canonicalTranscriptLineTokens(a)
	bTokens := canonicalTranscriptLineTokens(b)
	if len(aTokens) == 0 || len(bTokens) == 0 {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}
	return strings.Join(aTokens, " ") == strings.Join(bTokens, " ")
}

func similarTranscriptLine(a, b string) bool {
	aTokens := canonicalTranscriptLineTokens(a)
	bTokens := canonicalTranscriptLineTokens(b)
	if len(aTokens) == 0 || len(bTokens) == 0 {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}

	if strings.Join(aTokens, " ") == strings.Join(bTokens, " ") {
		return true
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

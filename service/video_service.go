package service

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"s2qt/util"
)

type VideoService struct {
	Paths      *util.AppPaths
	OnProgress func(stage, message string)
}

func NewVideoService(onProgress func(stage, message string)) (*VideoService, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return &VideoService{
		Paths:      paths,
		OnProgress: onProgress,
	}, nil
}

func (s *VideoService) progress(stage, message string) {
	if s.OnProgress != nil {
		s.OnProgress(stage, message)
	}
}

func (s *VideoService) cleanupTempFiles() error {
	files := []string{
		s.Paths.TempVideo,
		s.Paths.TempWav,
		s.Paths.TempTxt,
	}

	for _, f := range files {
		_ = os.Remove(f)
	}
	return nil
}

func (s *VideoService) checkRequiredFiles() error {
	required := []string{
		s.Paths.YtDlpExe,
		s.Paths.FfmpegExe,
		s.Paths.WhisperExe,
		s.Paths.WhisperModel,
	}

	for _, f := range required {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("필수 파일이 없습니다: %s", f)
		}
	}
	return nil
}

func (s *VideoService) downloadVideo(url string) (string, error) {
	args := []string{
		"-f", "mp4/bestvideo+bestaudio/best",
		"--merge-output-format", "mp4",
		"-o", s.Paths.TempVideo,
		url,
	}

	out, err := newHiddenCommand(s.Paths.YtDlpExe, args...).CombinedOutput()
	return string(out), err
}

func (s *VideoService) convertToWav() (string, error) {
	args := []string{
		"-y",
		"-i", s.Paths.TempVideo,
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		s.Paths.TempWav,
	}

	out, err := newHiddenCommand(s.Paths.FfmpegExe, args...).CombinedOutput()
	return string(out), err
}

func (s *VideoService) transcribe() (string, error) {
	args := []string{
		"-m", s.Paths.WhisperModel,
		"-f", s.Paths.TempWav,
		"-l", "ko",
		"-otxt",
		"-of", strings.TrimSuffix(s.Paths.TempTxt, ".txt"),
	}

	out, err := newHiddenCommand(s.Paths.WhisperExe, args...).CombinedOutput()
	return string(out), err
}

func (s *VideoService) countText(text string) (charCount, wordCount, lineCount, estimatedTokens int) {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	charCount = utf8.RuneCountInString(normalized)
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

func (s *VideoService) Run(url string) (*VideoPipelineResult, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("URL이 비어 있습니다")
	}

	var logs []string
	totalStart := time.Now()

	s.progress("init", "초기화 중...")

	if err := s.cleanupTempFiles(); err != nil {
		return nil, err
	}
	logs = append(logs, "[OK] temp 파일 초기화 완료")
	s.progress("init", "임시 파일 정리 완료")

	if err := s.checkRequiredFiles(); err != nil {
		return nil, err
	}
	logs = append(logs, "[OK] 필수 파일 확인 완료")
	s.progress("check", "필수 파일 확인 완료")

	s.progress("download", "동영상 다운로드 중...")
	downloadStart := time.Now()
	out, err := s.downloadVideo(url)
	downloadMs := time.Since(downloadStart).Milliseconds()
	logs = append(logs, fmt.Sprintf("=== yt-dlp (%d ms) ===\n%s", downloadMs, out))
	if err != nil {
		return nil, fmt.Errorf("동영상 다운로드 실패\n%s", out)
	}
	s.progress("download", fmt.Sprintf("다운로드 완료 (%d ms)", downloadMs))

	s.progress("convert", "WAV 변환 중...")
	convertStart := time.Now()
	out, err = s.convertToWav()
	convertMs := time.Since(convertStart).Milliseconds()
	logs = append(logs, fmt.Sprintf("=== ffmpeg (%d ms) ===\n%s", convertMs, out))
	if err != nil {
		return nil, fmt.Errorf("WAV 변환 실패\n%s", out)
	}
	s.progress("convert", fmt.Sprintf("WAV 변환 완료 (%d ms)", convertMs))

	s.progress("transcribe", "전사 중...")
	transcribeStart := time.Now()
	out, err = s.transcribe()
	transcribeMs := time.Since(transcribeStart).Milliseconds()
	logs = append(logs, fmt.Sprintf("=== whisper (%d ms) ===\n%s", transcribeMs, out))
	if err != nil {
		return nil, fmt.Errorf("전사 실패\n%s", out)
	}
	s.progress("transcribe", fmt.Sprintf("전사 완료 (%d ms)", transcribeMs))

	s.progress("finalize", "결과 정리 중...")

	txtBytes, err := os.ReadFile(s.Paths.TempTxt)
	if err != nil {
		return nil, fmt.Errorf("전사 결과 읽기 실패: %w", err)
	}

	text := string(txtBytes)
	charCount, wordCount, lineCount, estimatedTokens := s.countText(text)

	logs = append(logs, fmt.Sprintf("[COUNT] chars=%d, words=%d, lines=%d, estimatedTokens=%d",
		charCount, wordCount, lineCount, estimatedTokens))

	totalMs := time.Since(totalStart).Milliseconds()
	logs = append(logs, fmt.Sprintf("[TIME] download=%d ms, convert=%d ms, transcribe=%d ms, total=%d ms",
		downloadMs, convertMs, transcribeMs, totalMs))

	s.progress("done", fmt.Sprintf("전체 완료 (%d ms)", totalMs))

	return &VideoPipelineResult{
		Success:         true,
		Message:         "정상 처리되었습니다.",
		VideoFile:       s.Paths.TempVideo,
		WavFile:         s.Paths.TempWav,
		TranscriptFile:  s.Paths.TempTxt,
		TranscriptText:  text,
		MarkdownFile:    "",
		Log:             strings.Join(logs, "\n\n"),
		CharCount:       charCount,
		WordCount:       wordCount,
		LineCount:       lineCount,
		EstimatedTokens: estimatedTokens,
		DownloadMs:      downloadMs,
		ConvertMs:       convertMs,
		TranscribeMs:    transcribeMs,
		TotalMs:         totalMs,
	}, nil
}

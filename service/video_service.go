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
		s.Paths.TempAudioSrc,
		s.Paths.TempWav,
		s.Paths.TempTxt,
	}
	files = append(files, s.Paths.LegacyTempFiles...)

	for _, f := range files {
		_ = os.Remove(f)
	}
	return nil
}

func (s *VideoService) checkRequiredFiles() error {
	required := []string{
		s.Paths.YtDlpExe,
		s.Paths.FfmpegExe,
	}

	for _, f := range required {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("필수 파일이 없습니다: %s", f)
		}
	}

	return NewWhisperTranscriber(s.Paths, s.OnProgress).CheckRequiredFiles()
}

// downloadAudio는 URL에서 오디오 트랙만 내려받는다.
// 이후 단계가 전사뿐이라 영상은 쓰이지 않으므로, 영상까지 받으면
// 다운로드 용량과 시간만 수십 배로 늘어난다.
func (s *VideoService) downloadAudio(url string) (string, error) {
	args := buildYtDlpArgs(
		// ba(bestaudio)를 우선하되, 오디오 전용 포맷이 없는 사이트를 위해
		// b(영상+오디오 통합)로 물러선다. 어느 쪽이든 오디오는 반드시 포함된다.
		// 확장자만 적은 `-f mp4`는 yt-dlp가 "영상 전용" mp4로 대체 선택할 수 있어
		// 오디오 없는 파일을 받게 되므로 쓰지 않는다.
		"-f", "ba[ext=m4a]/ba/b",
		// m4a 컨테이너 보정(FixupM4a)에 ffmpeg가 필요하다.
		// PATH에 ffmpeg가 없는 배포 환경을 위해 번들 실행 파일을 지정한다.
		"--ffmpeg-location", s.Paths.FfmpegExe,
		// 재생목록 URL이 들어와도 영상 한 개만 받는다.
		"--no-playlist",
		"--newline", "--no-part",
		"-o", s.Paths.TempAudioSrc,
		url,
	)

	lastPct := -1
	return runHiddenCommandStreaming(func(line string) {
		if pct, ok := parseYtDlpProgress(line); ok && pct != lastPct {
			lastPct = pct
			s.progress("download", fmt.Sprintf("오디오 다운로드 중 %d%%", pct))
		}
	}, s.Paths.YtDlpExe, args...)
}

func (s *VideoService) convertToWav() (string, error) {
	return convertMediaToWav(s.Paths, s.Paths.TempAudioSrc, s.Paths.TempWav)
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
	url = NormalizeVideoURL(url)
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

	// yt-dlp는 버전이 오래되면 YouTube 다운로드가 곧바로 실패하므로,
	// 다운로드 직전에 최신 버전을 확인하고 필요하면 먼저 갱신한다.
	// 확인/갱신에 실패해도 기존 실행 파일로 다운로드를 시도한다.
	ytdlpStart := time.Now()
	upd, err := EnsureYtDlpLatest(func(message string) {
		s.progress("download", message)
	})
	if err != nil {
		logs = append(logs, fmt.Sprintf("[WARN] yt-dlp 버전 확인 실패: %v", err))
	} else {
		logs = append(logs, fmt.Sprintf("[OK] yt-dlp %s (%d ms)",
			upd.Message, time.Since(ytdlpStart).Milliseconds()))
		if upd.Updated {
			s.progress("download", "yt-dlp 업데이트 완료 ("+upd.Version+")")
		}
	}

	s.progress("download", "오디오 다운로드 중...")
	downloadStart := time.Now()
	out, err := s.downloadAudio(url)
	downloadMs := time.Since(downloadStart).Milliseconds()
	logs = append(logs, fmt.Sprintf("=== yt-dlp (%d ms) ===\n%s", downloadMs, out))
	if err != nil {
		// JS 런타임이 없으면 YouTube가 HTTP 403을 돌려주므로, 원인을 함께 안내한다.
		if hint := ytDlpJSRuntimeHint(); hint != "" {
			return nil, fmt.Errorf("오디오 다운로드 실패 - %s\n%s", hint, out)
		}
		return nil, fmt.Errorf("오디오 다운로드 실패\n%s", out)
	}
	s.progress("download", fmt.Sprintf("다운로드 완료 (%d ms)", downloadMs))

	s.progress("convert", "WAV 변환 중...")
	convertStart := time.Now()
	out, err = s.convertToWav()
	convertMs := time.Since(convertStart).Milliseconds()
	logs = append(logs, fmt.Sprintf("=== ffmpeg (%d ms) ===\n%s", convertMs, out))
	if err != nil {
		// convertMediaToWav가 이미 원인을 담은 메시지를 만들어 준다.
		// 배너로 가득한 원본 출력을 그대로 노출하면 실제 오류가 묻힌다.
		return nil, err
	}
	s.progress("convert", fmt.Sprintf("WAV 변환 완료 (%d ms)", convertMs))

	s.progress("transcribe", "전사 중...")
	transcription, err := NewWhisperTranscriber(s.Paths, s.OnProgress).Transcribe(s.Paths.TempWav)
	if err != nil {
		return nil, err
	}
	transcribeMs := transcription.TotalMs
	text := transcription.Text
	logs = append(logs, transcription.Log)

	s.progress("transcribe", fmt.Sprintf("전사 완료 (%d ms)", transcribeMs))
	s.progress("finalize", "결과 정리 중...")

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
		AudioFile:       s.Paths.TempAudioSrc,
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
		TranscribeModel: transcription.ModelName,
		FallbackModel:   transcription.FallbackModelName,
		FallbackUsed:    transcription.FallbackUsed,
		RetryReason:     transcription.RetryReason,
	}, nil
}

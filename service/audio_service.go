package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"s2qt/util"
)

type AudioService struct {
	Paths      *util.AppPaths
	OnProgress func(stage, message string)
}

func NewAudioService(onProgress func(stage, message string)) (*AudioService, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return &AudioService{
		Paths:      paths,
		OnProgress: onProgress,
	}, nil
}

func (s *AudioService) progress(stage, message string) {
	if s.OnProgress != nil {
		s.OnProgress(stage, message)
	}
}

func (s *AudioService) ValidateAudioFile(path string) error {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return fmt.Errorf("오디오 파일 경로가 비어 있습니다")
	}

	if _, err := os.Stat(cleanPath); err != nil {
		return fmt.Errorf("오디오 파일이 존재하지 않습니다: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(cleanPath))
	switch ext {
	case ".mp3", ".wav", ".m4a", ".aac", ".flac", ".ogg":
		return nil
	default:
		return fmt.Errorf("지원하지 않는 오디오 파일 형식입니다: %s", ext)
	}
}

func (s *AudioService) checkRequiredFiles() error {
	required := []string{
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

func (s *AudioService) convertToWav(inputAudioPath string) (string, error) {
	// whisper-cli의 입력 포맷 지원 여부는 빌드 옵션에 따라 달라질 수 있으므로
	// 입력 파일이 mp3/wav/m4a 등 무엇이든 간에 내부적으로 16kHz mono PCM WAV로
	// 표준화한 뒤 전사한다. 이렇게 하면 Windows 배포 환경에서 포맷별 예외를 줄일 수 있다.
	args := []string{
		"-y",
		"-i", inputAudioPath,
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		s.Paths.TempWav,
	}

	out, err := newHiddenCommand(s.Paths.FfmpegExe, args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("WAV 변환 실패: %w", err)
	}

	return string(out), nil
}

func (s *AudioService) transcribe() (string, error) {
	args := []string{
		"-m", s.Paths.WhisperModel,
		"-f", s.Paths.TempWav,
		"-l", "ko",
		"-otxt",
		"-of", strings.TrimSuffix(s.Paths.TempTxt, ".txt"),
	}

	out, err := newHiddenCommand(s.Paths.WhisperExe, args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("전사 실패: %w", err)
	}

	return string(out), nil
}

func (s *AudioService) ResolveRawText(audioPath string) (string, error) {
	audioPath = strings.TrimSpace(audioPath)
	if err := s.ValidateAudioFile(audioPath); err != nil {
		return "", err
	}

	if err := s.checkRequiredFiles(); err != nil {
		return "", err
	}

	s.progress("convert", "오디오를 WAV로 변환 중...")
	if _, err := s.convertToWav(audioPath); err != nil {
		return "", err
	}

	s.progress("transcribe", "오디오 전사 중...")
	if _, err := s.transcribe(); err != nil {
		return "", err
	}

	txtBytes, err := os.ReadFile(s.Paths.TempTxt)
	if err != nil {
		return "", fmt.Errorf("전사 결과 읽기 실패: %w", err)
	}

	rawText := strings.TrimSpace(string(txtBytes))
	if rawText == "" {
		return "", fmt.Errorf("전사 결과 텍스트가 비어 있습니다")
	}

	return rawText, nil
}

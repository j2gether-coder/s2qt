package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"s2qt/util"
)

type AudioService struct {
	Paths             *util.AppPaths
	OnProgress        func(stage, message string)
	LastTranscription *WhisperTranscriptionResult
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
	return NewWhisperTranscriber(s.Paths, s.OnProgress).CheckRequiredFiles()
}

func (s *AudioService) checkFFmpeg() error {
	if _, err := os.Stat(s.Paths.FfmpegExe); err != nil {
		return fmt.Errorf("필수 파일이 없습니다: %s", s.Paths.FfmpegExe)
	}
	return nil
}

func isWhisperSupportedAudio(path string) bool {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".wav", ".mp3", ".m4a", ".aac", ".flac", ".ogg":
		return true
	default:
		return false
	}
}

func (s *AudioService) convertToWav(inputAudioPath string) (string, error) {
	// whisper-cli의 입력 포맷 지원 여부는 빌드 옵션에 따라 달라질 수 있으므로
	// 입력 파일이 mp3/wav/m4a 등 무엇이든 간에 내부적으로 16kHz mono PCM WAV로
	// 표준화한 뒤 전사한다. 이렇게 하면 Windows 배포 환경에서 포맷별 예외를 줄일 수 있다.
	return convertMediaToWav(s.Paths, inputAudioPath, s.Paths.TempWav)
}

func (s *AudioService) ResolveRawText(audioPath string) (string, error) {
	audioPath = strings.TrimSpace(audioPath)
	if err := s.ValidateAudioFile(audioPath); err != nil {
		return "", err
	}

	if err := s.checkRequiredFiles(); err != nil {
		return "", err
	}

	inputAudioPath := audioPath
	converted := false

	if !isWhisperSupportedAudio(audioPath) {
		if err := s.checkFFmpeg(); err != nil {
			return "", err
		}

		s.progress("convert", "오디오를 WAV로 변환 중...")
		if _, err := s.convertToWav(audioPath); err != nil {
			return "", err
		}
		inputAudioPath = s.Paths.TempWav
		converted = true
	}

	s.progress("transcribe", "오디오 전사 중...")
	transcriber := NewWhisperTranscriber(s.Paths, s.OnProgress)
	transcription, err := transcriber.Transcribe(inputAudioPath)
	if err != nil && !converted {
		if ffmpegErr := s.checkFFmpeg(); ffmpegErr != nil {
			return "", err
		}
		s.progress("convert", "직접 전사 실패, WAV로 변환 중...")
		if _, convertErr := s.convertToWav(audioPath); convertErr != nil {
			return "", fmt.Errorf("%v\nWAV 변환도 실패했습니다: %w", err, convertErr)
		}
		inputAudioPath = s.Paths.TempWav
		converted = true
		transcription, err = transcriber.Transcribe(inputAudioPath)
	}
	if err != nil {
		return "", err
	}
	s.LastTranscription = transcription

	rawText := strings.TrimSpace(transcription.Text)
	if rawText == "" {
		return "", fmt.Errorf("전사 결과 텍스트가 비어 있습니다")
	}

	return rawText, nil
}

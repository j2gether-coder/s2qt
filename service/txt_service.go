package service

import (
	"fmt"
	"os"
	"strings"
)

type TxtService struct{}

func NewTxtService() *TxtService {
	return &TxtService{}
}

func (s *TxtService) LoadTextFile(path string) (string, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return "", fmt.Errorf("텍스트 파일 경로가 비어 있습니다")
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("텍스트 파일 읽기 실패: %w", err)
	}

	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", fmt.Errorf("텍스트 파일 내용이 비어 있습니다")
	}

	return text, nil
}

func (s *TxtService) ResolveRawText(inputMode, sourcePath, textContent string) (string, error) {
	switch strings.TrimSpace(inputMode) {
	case "paste":
		text := strings.TrimSpace(textContent)
		if text == "" {
			return "", fmt.Errorf("붙여넣은 텍스트가 비어 있습니다")
		}
		return text, nil

	case "file":
		return s.LoadTextFile(sourcePath)

	default:
		return "", fmt.Errorf("지원하지 않는 text inputMode: %s", inputMode)
	}
}

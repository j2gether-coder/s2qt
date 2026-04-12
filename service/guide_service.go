package service

import (
	"fmt"
	"os"
	"path/filepath"
	"s2qt/util"
	"strings"
)

func LoadGuideDocument(sectionId string) (string, error) {
	section := strings.TrimSpace(strings.ToLower(sectionId))

	var fileName string
	switch section {
	case "license":
		fileName = "license.md"
	case "guide":
		fileName = "user_guide.md"
	default:
		return "", fmt.Errorf("지원하지 않는 안내 문서 구분입니다: %s", sectionId)
	}

	paths, err := util.GetAppPaths()
	if err != nil {
		return "", fmt.Errorf("앱 경로를 확인하는 중 오류가 발생했습니다: %w", err)
	}

	docPath := filepath.Join(paths.Doc, fileName)

	data, err := os.ReadFile(docPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("안내 문서 파일을 찾을 수 없습니다: %s", fileName)
		}
		return "", fmt.Errorf("안내 문서를 읽는 중 오류가 발생했습니다: %w", err)
	}

	return string(data), nil
}

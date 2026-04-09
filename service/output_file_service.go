package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type OutputFileService struct {
	ctx context.Context
}

func NewOutputFileService() *OutputFileService {
	return &OutputFileService{}
}

func (s *OutputFileService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *OutputFileService) OpenGeneratedFile(filePath string) error {
	path := strings.TrimSpace(filePath)
	if path == "" {
		return fmt.Errorf("file path is empty")
	}

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", "", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func (s *OutputFileService) SaveGeneratedFile(filePath, audienceID, formatKey string) (string, error) {
	src := strings.TrimSpace(filePath)
	if src == "" {
		return "", fmt.Errorf("source file path is empty")
	}

	if _, err := os.Stat(src); err != nil {
		return "", fmt.Errorf("source file not found: %w", err)
	}

	ext := normalizeExt(src, formatKey)
	if ext == "" {
		return "", fmt.Errorf("cannot determine file extension")
	}

	defaultName := buildDefaultQTFileName(audienceID, ext)

	targetPath, err := wruntime.SaveFileDialog(s.ctx, wruntime.SaveDialogOptions{
		Title:                "산출물 저장",
		DefaultDirectory:     filepath.Dir(src),
		DefaultFilename:      defaultName,
		CanCreateDirectories: true,
		Filters: []wruntime.FileFilter{
			{
				DisplayName: strings.ToUpper(ext) + " 파일",
				Pattern:     "*." + ext,
			},
		},
	})
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(targetPath) == "" {
		return "", nil
	}

	if filepath.Ext(targetPath) == "" {
		targetPath += "." + ext
	}

	if err := renameFile(src, targetPath); err != nil {
		return "", err
	}

	return targetPath, nil
}

func normalizeExt(filePath, formatKey string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(strings.TrimSpace(filePath)), "."))
	if ext != "" {
		return ext
	}
	return strings.ToLower(strings.TrimSpace(formatKey))
}

func buildDefaultQTFileName(audienceID, ext string) string {
	yyMMdd := time.Now().Format("060102")
	return fmt.Sprintf("QT_%s_%s.%s", audienceLabel(audienceID), yyMMdd, ext)
}

func audienceLabel(audienceID string) string {
	switch strings.TrimSpace(audienceID) {
	case "adult":
		return "장년"
	case "young_adult":
		return "청년"
	case "teen":
		return "중고등부"
	case "child":
		return "어린이"
	default:
		return "공통"
	}
}

// TODO(step3-save):
// temp 산출물은 유지하고, 사용자 지정 경로로 복사 저장한다.
func renameFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

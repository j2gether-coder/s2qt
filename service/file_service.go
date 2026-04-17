package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"s2qt/util"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type FileService struct {
	ctx   context.Context
	Paths *util.AppPaths
}

type SaveOutputAsRequest struct {
	SourcePath string `json:"sourcePath"`
	Format     string `json:"format"`
	SermonDate string `json:"sermonDate"`
	Audience   string `json:"audience"`
}

type SaveOutputAsResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	SourcePath   string `json:"sourcePath"`
	SavedPath    string `json:"savedPath"`
	FileName     string `json:"fileName"`
	Format       string `json:"format"`
	DialogOpened bool   `json:"dialogOpened"`
}

type DialogFileFilter struct {
	DisplayName string
	Pattern     string
}

func NewFileService() (*FileService, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return &FileService{
		Paths: paths,
	}, nil
}

func (s *FileService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *FileService) OpenGeneratedFile(filePath string) error {
	path := strings.TrimSpace(filePath)
	if path == "" {
		return fmt.Errorf("file path is empty")
	}

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		return newHiddenCommand("cmd", "/c", "start", "", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

// Wails SaveFileDialog 기반 저장
func (s *FileService) SaveGeneratedFile(filePath, audienceID, formatKey string) (string, error) {
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

	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		return "", nil
	}

	if filepath.Ext(targetPath) == "" {
		targetPath += "." + ext
	}

	if err := CopyFile(src, targetPath); err != nil {
		return "", err
	}

	return targetPath, nil
}

// 커스텀 saveDialog 주입 기반 저장
func (s *FileService) SaveOutputAs(
	req *SaveOutputAsRequest,
	saveDialog func(defaultFilename string, filters []DialogFileFilter) (string, error),
) (*SaveOutputAsResult, error) {
	if req == nil {
		return nil, fmt.Errorf("save output request가 비어 있습니다")
	}

	sourcePath := strings.TrimSpace(req.SourcePath)
	format := strings.ToLower(strings.TrimSpace(req.Format))

	if sourcePath == "" {
		return nil, fmt.Errorf("sourcePath가 비어 있습니다")
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("원본 파일을 찾을 수 없습니다: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("원본 경로가 파일이 아닙니다")
	}

	defaultName := buildDefaultFileName(req)
	filters := buildDialogFilters(format, sourcePath)

	targetPath, err := saveDialog(defaultName, filters)
	if err != nil {
		return nil, fmt.Errorf("파일 저장 창 처리 중 오류가 발생했습니다: %w", err)
	}

	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		return &SaveOutputAsResult{
			Success:      false,
			Message:      "파일 저장이 취소되었습니다.",
			SourcePath:   sourcePath,
			SavedPath:    "",
			FileName:     "",
			Format:       format,
			DialogOpened: true,
		}, nil
	}

	targetPath = ensureExtension(targetPath, sourcePath, format)

	if err := CopyFile(sourcePath, targetPath); err != nil {
		return nil, fmt.Errorf("파일 저장 실패: %w", err)
	}

	return &SaveOutputAsResult{
		Success:      true,
		Message:      "파일 저장이 완료되었습니다.",
		SourcePath:   sourcePath,
		SavedPath:    targetPath,
		FileName:     filepath.Base(targetPath),
		Format:       format,
		DialogOpened: true,
	}, nil
}

func buildDefaultFileName(req *SaveOutputAsRequest) string {
	if req == nil {
		return "QT_qt_" + time.Now().Format("20060102") + ".txt"
	}

	sourcePath := strings.TrimSpace(req.SourcePath)
	format := strings.ToLower(strings.TrimSpace(req.Format))
	sermonDate := strings.TrimSpace(req.SermonDate)
	audience := audienceFileLabelEnglish(req.Audience)

	targetExt := extFromFormat(format)
	if targetExt == "" {
		targetExt = strings.ToLower(filepath.Ext(sourcePath))
	}
	if targetExt == "" {
		targetExt = ".txt"
	}

	datePart := normalizeDateForFileName(sermonDate)
	if datePart == "" {
		datePart = time.Now().Format("20060102")
	}

	return "QT_" + audience + "_" + datePart + targetExt
}

func buildDefaultQTFileName(audienceID, ext string) string {
	yyMMdd := time.Now().Format("060102")
	return fmt.Sprintf("QT_%s_%s.%s", audienceLabelKorean(audienceID), yyMMdd, ext)
}

func audienceFileLabelEnglish(audience string) string {
	switch strings.ToLower(strings.TrimSpace(audience)) {
	case "adult":
		return "adult"
	case "young_adult":
		return "young"
	case "teen":
		return "teen"
	case "child":
		return "child"
	default:
		return "qt"
	}
}

func audienceLabelKorean(audienceID string) string {
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

func normalizeDateForFileName(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	reDigits := regexp.MustCompile(`\d`)
	digits := strings.Join(reDigits.FindAllString(v, -1), "")
	if len(digits) >= 8 {
		return digits[:8]
	}

	layouts := []string{
		"2006-01-02",
		"2006.01.02",
		"2006/01/02",
		"20060102",
		"2006-1-2",
		"2006.1.2",
		"2006/1/2",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t.Format("20060102")
		}
	}

	return ""
}

func buildDialogFilters(format, sourcePath string) []DialogFileFilter {
	ext := extFromFormat(format)
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(strings.TrimSpace(sourcePath)))
	}

	switch ext {
	case ".html":
		return []DialogFileFilter{
			{DisplayName: "HTML 파일", Pattern: "*.html"},
			{DisplayName: "모든 파일", Pattern: "*.*"},
		}
	case ".pdf":
		return []DialogFileFilter{
			{DisplayName: "PDF 파일", Pattern: "*.pdf"},
			{DisplayName: "모든 파일", Pattern: "*.*"},
		}
	case ".docx":
		return []DialogFileFilter{
			{DisplayName: "Word 파일", Pattern: "*.docx"},
			{DisplayName: "모든 파일", Pattern: "*.*"},
		}
	case ".pptx":
		return []DialogFileFilter{
			{DisplayName: "PowerPoint 파일", Pattern: "*.pptx"},
			{DisplayName: "모든 파일", Pattern: "*.*"},
		}
	case ".png":
		return []DialogFileFilter{
			{DisplayName: "PNG 파일", Pattern: "*.png"},
			{DisplayName: "모든 파일", Pattern: "*.*"},
		}
	default:
		return []DialogFileFilter{
			{DisplayName: "모든 파일", Pattern: "*.*"},
		}
	}
}

func extFromFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "html":
		return ".html"
	case "pdf":
		return ".pdf"
	case "docx":
		return ".docx"
	case "pptx":
		return ".pptx"
	case "png":
		return ".png"
	default:
		return ""
	}
}

func ensureExtension(targetPath, sourcePath, format string) string {
	targetExt := strings.ToLower(filepath.Ext(strings.TrimSpace(targetPath)))
	if targetExt != "" {
		return targetPath
	}

	wantExt := extFromFormat(format)
	if wantExt == "" {
		wantExt = strings.ToLower(filepath.Ext(strings.TrimSpace(sourcePath)))
	}
	if wantExt == "" {
		return targetPath
	}

	return targetPath + wantExt
}

func normalizeExt(filePath, formatKey string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(strings.TrimSpace(filePath)), "."))
	if ext != "" {
		return ext
	}
	return strings.ToLower(strings.TrimSpace(formatKey))
}

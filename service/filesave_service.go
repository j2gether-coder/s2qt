package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"s2qt/util"
)

type FileSaveService struct {
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

func NewFileSaveService() (*FileSaveService, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return &FileSaveService{
		Paths: paths,
	}, nil
}

func (s *FileSaveService) SaveOutputAs(
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

	defaultName := s.buildDefaultFileName(req)
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

	if err := copyFile(sourcePath, targetPath); err != nil {
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

func (s *FileSaveService) buildDefaultFileName(req *SaveOutputAsRequest) string {
	if req == nil {
		return "QT_qt_" + time.Now().Format("20060102") + ".txt"
	}

	sourcePath := strings.TrimSpace(req.SourcePath)
	format := strings.ToLower(strings.TrimSpace(req.Format))
	sermonDate := strings.TrimSpace(req.SermonDate)
	audience := audienceFileLabel(req.Audience)

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

func audienceFileLabel(audience string) string {
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

func normalizeDateForFileName(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	// 1) YYYY-MM-DD / YYYY.MM.DD / YYYY/MM/DD / YYYYMMDD 대응
	reDigits := regexp.MustCompile(`\d`)
	digits := strings.Join(reDigits.FindAllString(v, -1), "")

	if len(digits) >= 8 {
		return digits[:8]
	}

	// 2) Go date parse 시도
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

func copyFile(sourcePath, targetPath string) error {
	if strings.TrimSpace(sourcePath) == "" {
		return fmt.Errorf("원본 파일 경로가 비어 있습니다")
	}
	if strings.TrimSpace(targetPath) == "" {
		return fmt.Errorf("대상 파일 경로가 비어 있습니다")
	}

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("원본 파일 열기 실패: %w", err)
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("대상 폴더 생성 실패: %w", err)
	}

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("대상 파일 생성 실패: %w", err)
	}
	defer func() {
		_ = targetFile.Close()
	}()

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return fmt.Errorf("파일 복사 실패: %w", err)
	}

	if err := targetFile.Sync(); err != nil {
		return fmt.Errorf("파일 저장 확정 실패: %w", err)
	}

	return nil
}

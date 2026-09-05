package util

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type AppPaths struct {
	Root string

	Bin          string
	Var          string
	Temp         string
	Reports      string
	Conf         string
	Data         string
	Doc          string
	DB           string
	Model        string
	Log          string
	EventLogFile string
	Image        string

	SiteLogoFile  string
	SiteQRFile    string
	DefaultQRFile string

	// executables in bin
	YtDlpExe   string
	FfmpegExe  string
	FfprobeExe string
	WhisperExe string

	// fixed db file
	DBFile       string
	SecurityFile string

	// fixed model file
	WhisperModel         string
	WhisperFallbackModel string
	ModelConfigFile      string
	WhisperModels        []WhisperModelConfig

	// fixed temp files
	// TempAudioSrc는 URL에서 내려받은 원본 오디오다(전사에 영상은 필요 없다).
	TempAudioSrc    string
	TempWav         string
	TempTxt         string
	TempJson        string
	TempMd          string
	TempHtml        string
	TempDocx        string
	TempPptx        string
	// TempExtendedHtml은 성구 전체 본문을 포함한 확장판 QT다(구 blog.html).
	// TempSermonSummary는 인포그래픽 제작에 넣을 설교요약문이다(구 infographic.md).
	TempExtendedHtml  string
	TempSermonSummary string

	// LegacyTempFiles는 이전 버전이 남긴 임시 파일이다(예: 영상 전체를 받던 시절의 video.mp4).
	// 정리 대상으로만 사용한다.
	LegacyTempFiles []string

	// fixed report files (프로젝트 루트 reports 폴더)
	ReportPdf string
	ReportPng string

	// Template files
	Template        string
	TemplateNoImage string
}

type WhisperModelConfig struct {
	Name string `yaml:"name"`
	File string `yaml:"file"`
	URL  string `yaml:"url"`
}

type ModelConfig struct {
	DefaultModel  string               `yaml:"default_model"`
	FallbackModel string               `yaml:"fallback_model"`
	Models        []WhisperModelConfig `yaml:"models"`
}

func GetAppPaths() (*AppPaths, error) {
	root, err := FindProjectRoot()
	if err != nil {
		return nil, err
	}

	binDir := filepath.Join(root, "bin")
	varDir := filepath.Join(root, "var")
	tempDir := filepath.Join(varDir, "temp")
	reportsDir := filepath.Join(root, "reports")
	modelDir := filepath.Join(varDir, "model")
	confDir := filepath.Join(varDir, "conf")
	dbDir := filepath.Join(varDir, "db")
	docDir := filepath.Join(varDir, "doc")
	logDir := filepath.Join(varDir, "log")
	imageDir := filepath.Join(varDir, "image")
	templateDir := filepath.Join(varDir, "template")

	p := &AppPaths{
		Root:     root,
		Bin:      binDir,
		Var:      varDir,
		Temp:     tempDir,
		Reports:  reportsDir,
		Conf:     confDir,
		Data:     filepath.Join(varDir, "data"),
		Doc:      docDir,
		DB:       dbDir,
		Log:      logDir,
		Model:    modelDir,
		Image:    imageDir,
		Template: templateDir,

		YtDlpExe:   filepath.Join(binDir, "yt-dlp.exe"),
		FfmpegExe:  filepath.Join(binDir, "ffmpeg.exe"),
		FfprobeExe: filepath.Join(binDir, "ffprobe.exe"),
		WhisperExe: filepath.Join(binDir, "whisper-cli.exe"),

		ModelConfigFile: filepath.Join(confDir, "model.yaml"),
		DBFile:          filepath.Join(dbDir, "s2qt.db"),
		SecurityFile:    filepath.Join(confDir, "security.json"),
		EventLogFile:    filepath.Join(logDir, "event.log"),

		TempAudioSrc: filepath.Join(tempDir, "source.m4a"),
		TempWav:      filepath.Join(tempDir, "audio.wav"),
		TempTxt:      filepath.Join(tempDir, "temp.txt"),
		TempJson:     filepath.Join(tempDir, "temp.json"),
		TempMd:       filepath.Join(tempDir, "temp.md"),
		TempHtml:     filepath.Join(tempDir, "temp.html"),
		TempDocx:     filepath.Join(tempDir, "temp.docx"),
		TempPptx:     filepath.Join(tempDir, "temp.pptx"),
		TempExtendedHtml: filepath.Join(tempDir, "extended.html"),

		LegacyTempFiles: []string{
			filepath.Join(tempDir, "video.mp4"),
			// 2026-09-05 이름 변경 이전 산출물. 새 작업 시작 시 정리한다.
			filepath.Join(tempDir, "blog.html"),
			filepath.Join(tempDir, "infographic.md"),
		},

		ReportPdf: filepath.Join(reportsDir, "report.pdf"),
		ReportPng: filepath.Join(reportsDir, "report.png"),

		TempSermonSummary: filepath.Join(tempDir, "sermon_summary.md"),

		SiteLogoFile:    filepath.Join(imageDir, "site_logo.png"),
		SiteQRFile:      filepath.Join(imageDir, "site_qr.png"),
		DefaultQRFile:   filepath.Join(imageDir, "s2qt_link.png"),
		TemplateNoImage: filepath.Join(templateDir, "no_image.png"),
	}

	if err := EnsureDirs(p); err != nil {
		return nil, err
	}

	modelConfig, err := LoadModelConfig(p.ModelConfigFile)
	if err != nil {
		return nil, err
	}
	p.WhisperModels = modelConfig.Models
	p.WhisperModel = filepath.Join(modelDir, modelConfig.DefaultModel)
	p.WhisperFallbackModel = filepath.Join(modelDir, modelConfig.FallbackModel)

	return p, nil
}

func LoadModelConfig(path string) (*ModelConfig, error) {
	if !fileExists(path) {
		cfg := DefaultModelConfig()
		if err := writeModelConfig(path, cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ModelConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	normalizeModelConfig(&cfg)
	return &cfg, nil
}

func DefaultModelConfig() *ModelConfig {
	cfg := &ModelConfig{}
	normalizeModelConfig(cfg)
	return cfg
}

func normalizeModelConfig(cfg *ModelConfig) {
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = "ggml-tiny.bin"
	}
	if cfg.FallbackModel == "" {
		cfg.FallbackModel = "ggml-base.bin"
	}
	if len(cfg.Models) == 0 {
		cfg.Models = []WhisperModelConfig{
			{
				Name: "tiny",
				File: "ggml-tiny.bin",
				URL:  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin",
			},
			{
				Name: "base",
				File: "ggml-base.bin",
				URL:  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin",
			},
		}
	}
	if !modelConfigHasFile(cfg.Models, "ggml-tiny.bin") {
		cfg.Models = append(cfg.Models, WhisperModelConfig{
			Name: "tiny",
			File: "ggml-tiny.bin",
			URL:  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin",
		})
	}
	if !modelConfigHasFile(cfg.Models, "ggml-base.bin") {
		cfg.Models = append(cfg.Models, WhisperModelConfig{
			Name: "base",
			File: "ggml-base.bin",
			URL:  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin",
		})
	}
}

func modelConfigHasFile(models []WhisperModelConfig, file string) bool {
	for _, model := range models {
		if model.File == file {
			return true
		}
	}
	return false
}

func writeModelConfig(path string, cfg *ModelConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0o644)
}

// 개발용: 현재 작업 경로에서 상위로 올라가며 go.mod / wails.json 탐색
// 배포용: 실패 시 실행 파일 위치 기준으로 bin 상위 폴더를 루트로 사용
func FindProjectRoot() (string, error) {
	if root, err := findRootFromWorkingDir(); err == nil {
		return root, nil
	}

	if root, err := findRootFromExecutable(); err == nil {
		return root, nil
	}

	return "", errors.New("project root not found")
}

func findRootFromWorkingDir() (string, error) {
	start, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := filepath.Clean(start)

	for {
		if isProjectRoot(dir) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", errors.New("project root not found from working directory")
}

func findRootFromExecutable() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return "", err
	}

	exeDir := filepath.Dir(exePath)
	root := filepath.Dir(exeDir) // bin 상위를 루트로 간주

	if root == "" {
		return "", errors.New("invalid executable root")
	}

	return root, nil
}

func isProjectRoot(dir string) bool {
	return fileExists(filepath.Join(dir, "go.mod")) ||
		fileExists(filepath.Join(dir, "wails.json"))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func EnsureDirs(p *AppPaths) error {
	dirs := []string{
		p.Bin,
		p.Var,
		p.Temp,
		p.Reports,
		p.Conf,
		p.Data,
		p.DB,
		p.Model,
		p.Log,
		p.Image,
		p.Template,
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

package util

import (
	"errors"
	"os"
	"path/filepath"
)

type AppPaths struct {
	Root string

	Bin          string
	Var          string
	Temp         string
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
	WhisperModel string

	// fixed temp files
	TempVideo string
	TempWav   string
	TempTxt   string
	TempJson  string
	TempMd    string
	TempHtml  string
	TempPdf   string
	TempDocx  string
	TempPptx  string
	TempPng   string
}

func GetAppPaths() (*AppPaths, error) {
	root, err := FindProjectRoot()
	if err != nil {
		return nil, err
	}

	binDir := filepath.Join(root, "bin")
	varDir := filepath.Join(root, "var")
	tempDir := filepath.Join(varDir, "temp")
	modelDir := filepath.Join(varDir, "model")
	confDir := filepath.Join(varDir, "conf")
	dbDir := filepath.Join(varDir, "db")
	docDir := filepath.Join(varDir, "doc")
	logDir := filepath.Join(varDir, "log")
	imageDir := filepath.Join(varDir, "image")

	p := &AppPaths{
		Root:  root,
		Bin:   binDir,
		Var:   varDir,
		Temp:  tempDir,
		Conf:  confDir,
		Data:  filepath.Join(varDir, "data"),
		Doc:   docDir,
		DB:    dbDir,
		Log:   logDir,
		Model: modelDir,
		Image: imageDir,

		YtDlpExe:   filepath.Join(binDir, "yt-dlp.exe"),
		FfmpegExe:  filepath.Join(binDir, "ffmpeg.exe"),
		FfprobeExe: filepath.Join(binDir, "ffprobe.exe"),
		WhisperExe: filepath.Join(binDir, "whisper-cli.exe"),

		WhisperModel: filepath.Join(modelDir, "ggml-tiny.bin"),
		DBFile:       filepath.Join(dbDir, "s2qt.db"),
		SecurityFile: filepath.Join(confDir, "security.json"),
		EventLogFile: filepath.Join(logDir, "event.log"),

		TempVideo: filepath.Join(tempDir, "video.mp4"),
		TempWav:   filepath.Join(tempDir, "audio.wav"),
		TempTxt:   filepath.Join(tempDir, "temp.txt"),
		TempJson:  filepath.Join(tempDir, "temp.json"),
		TempMd:    filepath.Join(tempDir, "temp.md"),
		TempHtml:  filepath.Join(tempDir, "temp.html"),
		TempPdf:   filepath.Join(tempDir, "temp.pdf"),
		TempDocx:  filepath.Join(tempDir, "temp.docx"),
		TempPptx:  filepath.Join(tempDir, "temp.pptx"),
		TempPng:   filepath.Join(tempDir, "temp.png"),

		SiteLogoFile:  filepath.Join(imageDir, "site_logo.png"),
		SiteQRFile:    filepath.Join(imageDir, "site_qr.png"),
		DefaultQRFile: filepath.Join(imageDir, "s2qt_link.png"),
	}

	return p, EnsureDirs(p)
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
		p.Conf,
		p.Data,
		p.DB,
		p.Model,
		p.Log,
		p.Image,
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

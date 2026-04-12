package util

import (
	"errors"
	"os"
	"path/filepath"
)

type AppPaths struct {
	Root string

	Bin   string
	Var   string
	Temp  string
	Conf  string
	Data  string
	Doc   string
	DB    string
	Model string

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

	p := &AppPaths{
		Root:  root,
		Bin:   binDir,
		Var:   varDir,
		Temp:  tempDir,
		Conf:  filepath.Join(varDir, "conf"),
		Data:  filepath.Join(varDir, "data"),
		Doc:   docDir,
		DB:    filepath.Join(varDir, "db"),
		Model: modelDir,

		YtDlpExe:   filepath.Join(binDir, "yt-dlp.exe"),
		FfmpegExe:  filepath.Join(binDir, "ffmpeg.exe"),
		FfprobeExe: filepath.Join(binDir, "ffprobe.exe"),
		WhisperExe: filepath.Join(binDir, "whisper-cli.exe"),

		WhisperModel: filepath.Join(modelDir, "ggml-tiny.bin"),
		DBFile:       filepath.Join(dbDir, "s2qt.db"),
		SecurityFile: filepath.Join(confDir, "security.json"),

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
	}

	return p, EnsureDirs(p)
}

// 현재 작업 위치에서 시작해서 상위로 올라가며
// go.mod 또는 wails.json 이 있는 폴더를 프로젝트 루트로 판단
func FindProjectRoot() (string, error) {
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

	return "", errors.New("project root not found")
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
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

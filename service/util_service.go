package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"s2qt/util"
)

type UtilComponent struct {
	Key          string
	FileName     string
	TargetPath   string
	Downloadable bool
	Versioned    bool
	URL          string
	Description  string
}

type UtilCheckOptions struct {
	NeedFFmpeg bool
	NeedYtDlp  bool
	NeedModel  bool
	AutoRepair bool
}

type UtilCheckResult struct {
	CheckedAt time.Time         `json:"checked_at"`
	Mode      string            `json:"mode"`
	OK        bool              `json:"ok"`
	Checked   []string          `json:"checked"`
	Missing   []string          `json:"missing"`
	Installed []string          `json:"installed"`
	Versions  map[string]string `json:"versions"`
	Message   string            `json:"message"`
}

type UtilVersionInfo struct {
	LastCheckedAt  string            `json:"last_checked_at"`
	LastCheckMode  string            `json:"last_check_mode"`
	LastCheckOK    bool              `json:"last_check_ok"`
	Installed      map[string]bool   `json:"installed"`
	Versions       map[string]string `json:"versions"`
	ModelInstalled bool              `json:"model_installed"`
}

func CheckRuntimeForText() (*UtilCheckResult, error) {
	return EnsureRuntime(UtilCheckOptions{
		NeedFFmpeg: false,
		NeedYtDlp:  false,
		NeedModel:  false,
		AutoRepair: false,
	}, "text")
}

func CheckRuntimeForAudio(autoRepair bool) (*UtilCheckResult, error) {
	return EnsureRuntime(UtilCheckOptions{
		NeedFFmpeg: true,
		NeedYtDlp:  false,
		NeedModel:  true,
		AutoRepair: autoRepair,
	}, "audio")
}

func CheckRuntimeForVideo(autoRepair bool) (*UtilCheckResult, error) {
	return EnsureRuntime(UtilCheckOptions{
		NeedFFmpeg: true,
		NeedYtDlp:  true,
		NeedModel:  true,
		AutoRepair: autoRepair,
	}, "video")
}

func EnsureRuntime(opts UtilCheckOptions, mode string) (*UtilCheckResult, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	result := &UtilCheckResult{
		CheckedAt: time.Now(),
		Mode:      mode,
		OK:        true,
		Checked:   []string{},
		Missing:   []string{},
		Installed: []string{},
		Versions:  map[string]string{},
	}

	if err := os.MkdirAll(paths.Conf, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(paths.Data, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(paths.Bin, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(paths.Model, 0o755); err != nil {
		return nil, err
	}

	components := buildComponents(paths, opts)

	for _, c := range components {
		result.Checked = append(result.Checked, c.Key)

		if fileExists(c.TargetPath) {
			if c.Versioned && c.Key == "yt-dlp" {
				result.Versions[c.Key] = getYtDlpVersion(c.TargetPath)
			}
			continue
		}

		result.Missing = append(result.Missing, c.Key)

		if !opts.AutoRepair || !c.Downloadable {
			result.OK = false
			continue
		}

		if err := installComponent(paths.Data, c); err != nil {
			result.OK = false
			result.Message = fmt.Sprintf("%s 설치 실패: %v", c.Key, err)
			continue
		}

		result.Installed = append(result.Installed, c.Key)

		if c.Versioned && c.Key == "yt-dlp" && fileExists(c.TargetPath) {
			result.Versions[c.Key] = getYtDlpVersion(c.TargetPath)
		}
	}

	if !result.OK && result.Message == "" {
		result.Message = "필수 런타임 구성요소가 누락되었습니다."
	}
	if result.OK && result.Message == "" {
		result.Message = "런타임 준비가 완료되었습니다."
	}

	_ = saveUtilVersion(paths.Conf, result)
	_ = cleanupDataDir(paths.Data)

	return result, nil
}

func buildComponents(paths *util.AppPaths, opts UtilCheckOptions) []UtilComponent {
	items := []UtilComponent{}

	if opts.NeedFFmpeg {
		items = append(items,
			UtilComponent{
				Key:          "ffmpeg",
				FileName:     "ffmpeg.exe",
				TargetPath:   paths.FfmpegExe,
				Downloadable: true,
				Versioned:    false,
				URL:          "",
				Description:  "오디오 변환",
			},
			UtilComponent{
				Key:          "ffprobe",
				FileName:     "ffprobe.exe",
				TargetPath:   paths.FfprobeExe,
				Downloadable: true,
				Versioned:    false,
				URL:          "",
				Description:  "미디어 정보 확인",
			},
		)
	}

	if opts.NeedYtDlp {
		items = append(items, UtilComponent{
			Key:          "yt-dlp",
			FileName:     "yt-dlp.exe",
			TargetPath:   paths.YtDlpExe,
			Downloadable: true,
			Versioned:    true,
			URL:          "",
			Description:  "동영상 다운로드",
		})
	}

	if opts.NeedModel {
		items = append(items, UtilComponent{
			Key:          "ggml-tiny.bin",
			FileName:     "ggml-tiny.bin",
			TargetPath:   paths.WhisperModel,
			Downloadable: true,
			Versioned:    false,
			URL:          "",
			Description:  "Whisper model",
		})
	}

	return items
}

func installComponent(dataDir string, c UtilComponent) error {
	if strings.TrimSpace(c.URL) == "" {
		return fmt.Errorf("download url not configured")
	}

	stagedPath, err := downloadToDataDir(dataDir, c)
	if err != nil {
		return err
	}

	if err := verifyDownloadedFile(stagedPath); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(c.TargetPath), 0o755); err != nil {
		return err
	}

	return moveFile(stagedPath, c.TargetPath)
}

func downloadToDataDir(dataDir string, c UtilComponent) (string, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", err
	}

	stagedPath := filepath.Join(dataDir, c.FileName)

	resp, err := http.Get(c.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http status: %s", resp.Status)
	}

	out, err := os.Create(stagedPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", err
	}
	if err := out.Sync(); err != nil {
		return "", err
	}

	return stagedPath, nil
}

func verifyDownloadedFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("downloaded path is directory: %s", path)
	}
	if info.Size() <= 0 {
		return fmt.Errorf("downloaded file size is zero: %s", path)
	}
	return nil
}

func saveUtilVersion(confDir string, result *UtilCheckResult) error {
	confPath := filepath.Join(confDir, "util_ver.json")

	info, err := loadUtilVersionInfo(confPath)
	if err != nil {
		info = &UtilVersionInfo{
			Installed: map[string]bool{},
			Versions:  map[string]string{},
		}
	}

	if info.Installed == nil {
		info.Installed = map[string]bool{}
	}
	if info.Versions == nil {
		info.Versions = map[string]string{}
	}

	info.LastCheckedAt = result.CheckedAt.Format(time.RFC3339)
	info.LastCheckMode = result.Mode
	info.LastCheckOK = result.OK

	for _, key := range result.Checked {
		info.Installed[key] = false
	}
	for _, key := range result.Checked {
		if !contains(result.Missing, key) || contains(result.Installed, key) {
			info.Installed[key] = true
		}
	}

	for k, v := range result.Versions {
		if strings.TrimSpace(v) != "" {
			info.Versions[k] = v
		}
	}

	info.ModelInstalled = info.Installed["ggml-tiny.bin"]

	return writeJSON(confPath, info)
}

func loadUtilVersionInfo(path string) (*UtilVersionInfo, error) {
	if !fileExists(path) {
		return &UtilVersionInfo{
			Installed: map[string]bool{},
			Versions:  map[string]string{},
		}, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var info UtilVersionInfo
	if err := json.Unmarshal(b, &info); err != nil {
		return nil, err
	}

	if info.Installed == nil {
		info.Installed = map[string]bool{}
	}
	if info.Versions == nil {
		info.Versions = map[string]string{}
	}

	return &info, nil
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func cleanupDataDir(dataDir string) error {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dataDir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func getYtDlpVersion(binPath string) string {
	if !fileExists(binPath) {
		return ""
	}
	cmd := exec.Command(binPath, "--version")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func moveFile(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(src, dst)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

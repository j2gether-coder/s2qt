package service

import (
	"archive/zip"
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

const (
	defaultFFmpegPackageURL = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
	defaultYtDlpURL         = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	defaultModelURL         = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin"

	ffmpegPackageFileName = "ffmpeg-release-essentials.zip"
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
		LogError("util: get app paths failed: " + err.Error())
		return nil, err
	}

	LogInfo("util: runtime check started mode=" + mode)

	result := &UtilCheckResult{
		CheckedAt: time.Now(),
		Mode:      mode,
		OK:        true,
		Checked:   []string{},
		Missing:   []string{},
		Installed: []string{},
		Versions:  map[string]string{},
		Message:   "",
	}

	if err := ensureDir(paths.Conf); err != nil {
		LogError("util: ensure conf dir failed: " + err.Error())
		return nil, err
	}
	if err := ensureDir(paths.Data); err != nil {
		LogError("util: ensure data dir failed: " + err.Error())
		return nil, err
	}
	if err := ensureDir(paths.Bin); err != nil {
		LogError("util: ensure bin dir failed: " + err.Error())
		return nil, err
	}
	if err := ensureDir(paths.Model); err != nil {
		LogError("util: ensure model dir failed: " + err.Error())
		return nil, err
	}

	if opts.NeedFFmpeg {
		LogInfo("util: ffmpeg package check started")

		result.Checked = append(result.Checked, "ffmpeg", "ffprobe")

		ffmpegMissing := !fileExists(paths.FfmpegExe)
		ffprobeMissing := !fileExists(paths.FfprobeExe)

		if ffmpegMissing || ffprobeMissing {
			LogInfo("util: ffmpeg package missing detected")

			if opts.AutoRepair {
				LogInfo("util: ffmpeg package install started")
				if err := installFFmpegPackage(paths.Data, paths.Bin); err != nil {
					result.OK = false
					LogError("util: ffmpeg package install failed: " + err.Error())
					if result.Message == "" {
						result.Message = fmt.Sprintf("ffmpeg 패키지 설치 실패: %v", err)
					}
				} else {
					LogInfo("util: ffmpeg package install completed")
				}
			} else {
				result.OK = false
			}
		}

		if fileExists(paths.FfmpegExe) {
			if ffmpegMissing {
				result.Installed = appendIfMissing(result.Installed, "ffmpeg")
				LogInfo("util: ffmpeg.exe ready")
			}
		} else {
			result.Missing = appendIfMissing(result.Missing, "ffmpeg")
			result.OK = false
			LogError("util: ffmpeg.exe missing")
		}

		if fileExists(paths.FfprobeExe) {
			if ffprobeMissing {
				result.Installed = appendIfMissing(result.Installed, "ffprobe")
				LogInfo("util: ffprobe.exe ready")
			}
		} else {
			result.Missing = appendIfMissing(result.Missing, "ffprobe")
			result.OK = false
			LogError("util: ffprobe.exe missing")
		}
	}

	for _, c := range buildDirectComponents(paths, opts) {
		LogInfo("util: component check started key=" + c.Key)

		result.Checked = append(result.Checked, c.Key)

		existedBefore := fileExists(c.TargetPath)
		if existedBefore {
			if c.Versioned && c.Key == "yt-dlp" {
				result.Versions[c.Key] = getYtDlpVersion(c.TargetPath)
			}
			LogInfo("util: component already exists key=" + c.Key)
			continue
		}

		if opts.AutoRepair && c.Downloadable {
			LogInfo("util: component install started key=" + c.Key)
			if err := installDirectComponent(paths.Data, c); err != nil {
				result.OK = false
				LogError("util: component install failed key=" + c.Key + " err=" + err.Error())
				if result.Message == "" {
					result.Message = fmt.Sprintf("%s 설치 실패: %v", c.Key, err)
				}
			}
		}

		if fileExists(c.TargetPath) {
			result.Installed = appendIfMissing(result.Installed, c.Key)
			LogInfo("util: component ready key=" + c.Key)

			if c.Versioned && c.Key == "yt-dlp" {
				result.Versions[c.Key] = getYtDlpVersion(c.TargetPath)
			}
		} else {
			result.Missing = appendIfMissing(result.Missing, c.Key)
			result.OK = false
			LogError("util: component missing key=" + c.Key)
		}
	}

	if !result.OK && result.Message == "" {
		result.Message = "필수 런타임 구성요소가 누락되었거나 설치에 실패했습니다."
	}
	if result.OK && result.Message == "" {
		result.Message = "런타임 준비가 완료되었습니다."
	}

	if err := saveUtilVersion(paths.Conf, result); err != nil {
		LogError("util: util_ver.json save failed: " + err.Error())
	} else {
		LogInfo("util: util_ver.json saved")
	}

	LogInfo("util: cleanup data dir started")
	if err := cleanupDataDir(paths.Data); err != nil {
		LogError("util: cleanup data dir failed: " + err.Error())
	} else {
		LogInfo("util: cleanup data dir completed")
	}

	if result.OK {
		LogInfo("util: runtime check completed")
	} else {
		LogError("util: runtime check completed with failure")
	}

	return result, nil
}

func buildDirectComponents(paths *util.AppPaths, opts UtilCheckOptions) []UtilComponent {
	items := []UtilComponent{}

	if opts.NeedYtDlp {
		items = append(items, UtilComponent{
			Key:          "yt-dlp",
			FileName:     "yt-dlp.exe",
			TargetPath:   paths.YtDlpExe,
			Downloadable: true,
			Versioned:    true,
			URL:          defaultYtDlpURL,
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
			URL:          defaultModelURL,
			Description:  "Whisper model",
		})
	}

	return items
}

func installDirectComponent(dataDir string, c UtilComponent) error {
	if strings.TrimSpace(c.URL) == "" {
		return fmt.Errorf("download url not configured")
	}

	LogInfo("util: download started key=" + c.Key)

	stagedPath, err := downloadFile(dataDir, c.FileName, c.URL)
	if err != nil {
		LogError("util: download failed key=" + c.Key + " err=" + err.Error())
		return err
	}

	if err := verifyDownloadedFile(stagedPath); err != nil {
		LogError("util: verification failed key=" + c.Key + " err=" + err.Error())
		return err
	}

	if err := ensureDir(filepath.Dir(c.TargetPath)); err != nil {
		LogError("util: target dir ensure failed key=" + c.Key + " err=" + err.Error())
		return err
	}

	if err := copyFile(stagedPath, c.TargetPath); err != nil {
		LogError("util: copy failed key=" + c.Key + " err=" + err.Error())
		return err
	}

	LogInfo("util: download completed key=" + c.Key)
	return nil
}

func installFFmpegPackage(dataDir, binDir string) error {
	if strings.TrimSpace(defaultFFmpegPackageURL) == "" {
		return fmt.Errorf("ffmpeg package url not configured")
	}

	LogInfo("util: ffmpeg zip download started")

	zipPath, err := downloadFile(dataDir, ffmpegPackageFileName, defaultFFmpegPackageURL)
	if err != nil {
		LogError("util: ffmpeg zip download failed: " + err.Error())
		return err
	}

	LogInfo("util: ffmpeg zip download completed")

	if err := verifyDownloadedFile(zipPath); err != nil {
		LogError("util: ffmpeg zip verification failed: " + err.Error())
		return err
	}

	extractDir := filepath.Join(dataDir, "ffmpeg_extract")
	if err := os.RemoveAll(extractDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := ensureDir(extractDir); err != nil {
		return err
	}

	LogInfo("util: ffmpeg zip extract started")
	if err := unzipFile(zipPath, extractDir); err != nil {
		LogError("util: ffmpeg zip extract failed: " + err.Error())
		return err
	}
	LogInfo("util: ffmpeg zip extract completed")

	ffmpegSrc, err := findFileRecursive(extractDir, "ffmpeg.exe")
	if err != nil {
		LogError("util: ffmpeg.exe search failed: " + err.Error())
		return err
	}

	ffprobeSrc, err := findFileRecursive(extractDir, "ffprobe.exe")
	if err != nil {
		LogError("util: ffprobe.exe search failed: " + err.Error())
		return err
	}

	if err := copyFile(ffmpegSrc, filepath.Join(binDir, "ffmpeg.exe")); err != nil {
		LogError("util: ffmpeg.exe copy failed: " + err.Error())
		return err
	}
	LogInfo("util: ffmpeg.exe copied")

	if err := copyFile(ffprobeSrc, filepath.Join(binDir, "ffprobe.exe")); err != nil {
		LogError("util: ffprobe.exe copy failed: " + err.Error())
		return err
	}
	LogInfo("util: ffprobe.exe copied")

	return nil
}

func downloadFile(dataDir, fileName, url string) (string, error) {
	if err := ensureDir(dataDir); err != nil {
		return "", err
	}

	targetPath := filepath.Join(dataDir, fileName)

	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http status: %s", resp.Status)
	}

	out, err := os.Create(targetPath)
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

	return targetPath, nil
}

func unzipFile(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)

		// zip slip 방지
		cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)
		cleanTarget := filepath.Clean(targetPath)
		if !strings.HasPrefix(cleanTarget, cleanDest) {
			return fmt.Errorf("invalid zip entry path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := ensureDir(cleanTarget); err != nil {
				return err
			}
			continue
		}

		if err := ensureDir(filepath.Dir(cleanTarget)); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.Create(cleanTarget)
		if err != nil {
			rc.Close()
			return err
		}

		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return err
		}

		if err := out.Close(); err != nil {
			rc.Close()
			return err
		}
		if err := rc.Close(); err != nil {
			return err
		}
	}

	return nil
}

func findFileRecursive(rootDir, fileName string) (string, error) {
	var found string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil || info.IsDir() {
			return nil
		}
		if strings.EqualFold(info.Name(), fileName) {
			found = path
			return io.EOF
		}
		return nil
	})

	if err != nil && err != io.EOF {
		return "", err
	}
	if strings.TrimSpace(found) == "" {
		return "", fmt.Errorf("%s not found in extracted package", fileName)
	}

	return found, nil
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
		info.Installed[key] = !contains(result.Missing, key)
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
		fullPath := filepath.Join(dataDir, entry.Name())
		if err := os.RemoveAll(fullPath); err != nil {
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

func utilcopyFile(src, dst string) error {
	if err := ensureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

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

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
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

func appendIfMissing(items []string, target string) []string {
	if contains(items, target) {
		return items
	}
	return append(items, target)
}

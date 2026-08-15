package service

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"s2qt/util"
)

const (
	defaultFFmpegPackageURL = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
	defaultYtDlpURL         = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"

	ffmpegPackageFileName = "ffmpeg-release-essentials.zip"

	defaultPDFiumPackageURL  = "https://www.nuget.org/api/v2/package/bblanchon.PDFium.Win32/149.0.7811"
	pdfiumPackageFileName    = "bblanchon.PDFium.Win32.149.0.7811.nupkg"
	pdfiumDllFileName        = "pdfium.dll"
	pdfiumRuntimeDocFileName = "pdfium_runtime.md"

	// yt-dlp는 YouTube 변경에 맞춰 수시로 갱신되므로, 버전이 오래되면 다운로드 자체가 실패한다.
	// 따라서 다른 유틸과 달리 "파일이 있으면 통과"가 아니라 사용 시점마다 최신 버전을 확인한다.
	ytDlpLatestReleaseAPI = "https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest"
	utilVersionFileName   = "util_ver.json"
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
	NeedPDFium bool
	// NeedJSRuntime은 yt-dlp의 YouTube 추출에 필요한 JavaScript 런타임 확보 여부다.
	NeedJSRuntime bool
	AutoRepair    bool
}

type UtilCheckResult struct {
	CheckedAt string            `json:"checked_at"`
	Mode      string            `json:"mode"`
	OK        bool              `json:"ok"`
	Checked   []string          `json:"checked"`
	Missing   []string          `json:"missing"`
	Installed []string          `json:"installed"`
	Updated   []string          `json:"updated"`
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
		NeedFFmpeg:    true,
		NeedYtDlp:     true,
		NeedModel:     true,
		NeedJSRuntime: true,
		AutoRepair:    autoRepair,
	}, "video")
}

func CheckRuntimeForPNG(autoRepair bool) (*UtilCheckResult, error) {
	return EnsureRuntime(UtilCheckOptions{
		NeedFFmpeg: false,
		NeedYtDlp:  false,
		NeedModel:  false,
		NeedPDFium: true,
		AutoRepair: autoRepair,
	}, "png")
}

func EnsureRuntime(opts UtilCheckOptions, mode string) (*UtilCheckResult, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		LogError("util: get app paths failed: " + err.Error())
		return nil, err
	}

	LogInfo("util: runtime check started mode=" + mode)

	result := &UtilCheckResult{
		CheckedAt: time.Now().Format(time.RFC3339),
		Mode:      mode,
		OK:        true,
		Checked:   []string{},
		Missing:   []string{},
		Installed: []string{},
		Updated:   []string{},
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

	if opts.NeedPDFium {
		LogInfo("util: pdfium.dll check started")

		result.Checked = append(result.Checked, "pdfium.dll")

		pdfiumPath := filepath.Join(paths.Bin, pdfiumDllFileName)
		pdfiumMissing := !fileExists(pdfiumPath)

		if pdfiumMissing {
			LogInfo("util: pdfium.dll missing detected")

			if opts.AutoRepair {
				LogInfo("util: pdfium.dll install started")
				if err := installPDFiumDLL(paths.Data, paths.Bin); err != nil {
					result.OK = false
					LogError("util: pdfium.dll install failed: " + err.Error())
					if result.Message == "" {
						result.Message = fmt.Sprintf("pdfium.dll 설치 실패: %v", err)
					}
				} else {
					LogInfo("util: pdfium.dll install completed")
				}
			} else {
				result.OK = false
			}
		}

		if fileExists(pdfiumPath) {
			if pdfiumMissing {
				result.Installed = appendIfMissing(result.Installed, "pdfium.dll")
				LogInfo("util: pdfium.dll ready")
			}
		} else {
			result.Missing = appendIfMissing(result.Missing, "pdfium.dll")
			result.OK = false
			LogError("util: pdfium.dll missing")
		}
	}

	if opts.NeedJSRuntime {
		LogInfo("util: js runtime check started")

		result.Checked = append(result.Checked, jsRuntimeKey)

		name, runtimePath := findYtDlpJSRuntime()

		// 사용할 수 있는 런타임이 이미 있으면(bin 동봉 또는 PATH 설치) 새로 받지 않는다.
		if name == "" && opts.AutoRepair {
			LogInfo("util: js runtime missing, deno install started")

			if err := installDenoRuntime(paths.Data, paths.Bin); err != nil {
				LogError("util: deno install failed: " + err.Error())
				if result.Message == "" {
					result.Message = fmt.Sprintf("JavaScript 런타임 설치 실패: %v", err)
				}
			} else {
				LogInfo("util: deno install completed")
				if name, runtimePath = findYtDlpJSRuntime(); name != "" {
					result.Installed = appendIfMissing(result.Installed, jsRuntimeKey)
				}
			}
		}

		if name != "" {
			// deno는 "deno 2.9.5 ...", node는 "v22.x" 형태라 이름 중복을 피해 정리한다.
			version := strings.TrimSpace(jsRuntimeVersion(runtimePath))
			switch {
			case version == "":
				version = name
			case !strings.HasPrefix(strings.ToLower(version), name):
				version = name + " " + version
			}

			result.Versions[jsRuntimeKey] = version
			LogInfo("util: js runtime ready " + version + " path=" + runtimePath)
		} else {
			result.Missing = appendIfMissing(result.Missing, jsRuntimeKey)
			result.OK = false
			LogError("util: js runtime missing")
		}
	}

	for _, c := range buildDirectComponents(paths, opts) {
		LogInfo("util: component check started key=" + c.Key)

		result.Checked = append(result.Checked, c.Key)

		existedBefore := fileExists(c.TargetPath)
		if existedBefore {
			LogInfo("util: component already exists key=" + c.Key)

			// 이미 설치된 파일이라도 yt-dlp는 최신 버전 여부를 확인해 필요하면 교체한다.
			if c.Versioned && c.Key == "yt-dlp" {
				ensureYtDlpUpToDate(paths, c, opts, result)
			}
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
		if len(result.Updated) > 0 {
			result.Message = fmt.Sprintf("런타임 준비가 완료되었습니다. (업데이트: %s)", strings.Join(result.Updated, ", "))
		} else {
			result.Message = "런타임 준비가 완료되었습니다."
		}
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

func ytDlpComponent(paths *util.AppPaths) UtilComponent {
	return UtilComponent{
		Key:          "yt-dlp",
		FileName:     "yt-dlp.exe",
		TargetPath:   paths.YtDlpExe,
		Downloadable: true,
		Versioned:    true,
		URL:          defaultYtDlpURL,
		Description:  "동영상 다운로드",
	}
}

func buildDirectComponents(paths *util.AppPaths, opts UtilCheckOptions) []UtilComponent {
	items := []UtilComponent{}

	if opts.NeedYtDlp {
		items = append(items, ytDlpComponent(paths))
	}

	if opts.NeedModel {
		for _, model := range paths.WhisperModels {
			fileName := strings.TrimSpace(model.File)
			if fileName == "" {
				continue
			}
			items = append(items, UtilComponent{
				Key:          fileName,
				FileName:     fileName,
				TargetPath:   filepath.Join(paths.Model, fileName),
				Downloadable: true,
				Versioned:    false,
				URL:          model.URL,
				Description:  "Whisper model",
			})
		}
	}

	return items
}

// YtDlpUpdateResult는 yt-dlp 최신 버전 확인/교체 결과다.
type YtDlpUpdateResult struct {
	Updated         bool   `json:"updated"` // 최신본으로 교체했는지
	PreviousVersion string `json:"previous_version"`
	Version         string `json:"version"`
	LatestVersion   string `json:"latest_version"`
	Message         string `json:"message"`
}

// EnsureYtDlpLatest는 yt-dlp.exe가 최신 버전인지 확인하고, 오래된 경우 최신본을 받아 교체한다.
// yt-dlp는 버전이 오래되면 YouTube 다운로드가 곧바로 실패하므로,
// 동영상 다운로드 직전처럼 yt-dlp를 실제로 쓰기 직전에 호출한다.
// onProgress는 nil이어도 되며, 확인/다운로드 진행 상황을 호출자에게 알린다.
func EnsureYtDlpLatest(onProgress func(message string)) (*YtDlpUpdateResult, error) {
	notify := func(message string) {
		if onProgress != nil {
			onProgress(message)
		}
	}

	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	c := ytDlpComponent(paths)

	// 아직 설치 전이면 최신본을 새로 내려받는다.
	if !fileExists(c.TargetPath) {
		notify("yt-dlp 설치 중...")
		LogInfo("util: yt-dlp not installed, installing latest")

		if err := installDirectComponent(paths.Data, c); err != nil {
			return nil, fmt.Errorf("yt-dlp 설치 실패: %w", err)
		}

		res := &YtDlpUpdateResult{
			Updated: true,
			Version: getYtDlpVersion(c.TargetPath),
		}
		res.Message = "yt-dlp 설치 완료 (" + res.Version + ")"
		saveYtDlpVersion(paths.Conf, res)
		return res, nil
	}

	res := runYtDlpUpdate(paths, c, notify)
	saveYtDlpVersion(paths.Conf, res)

	return res, nil
}

// runYtDlpUpdate는 버전 비교와 교체의 공통 구현이다.
// 최신 버전 조회나 교체에 실패해도 기존 실행 파일은 그대로 남으므로 에러로 올리지 않는다.
func runYtDlpUpdate(paths *util.AppPaths, c UtilComponent, notify func(string)) *YtDlpUpdateResult {
	current := getYtDlpVersion(c.TargetPath)

	res := &YtDlpUpdateResult{
		PreviousVersion: current,
		Version:         current,
	}

	if notify == nil {
		notify = func(string) {}
	}

	notify("yt-dlp 최신 버전 확인 중...")
	LogInfo("util: yt-dlp latest version lookup started current=" + current)

	latest, err := fetchLatestYtDlpVersion()
	if err != nil {
		res.Message = "최신 버전 조회 실패: " + err.Error()
		LogError("util: yt-dlp latest version lookup failed: " + err.Error())
		return res
	}

	res.LatestVersion = latest

	if current != "" && normalizeVersionTag(current) == normalizeVersionTag(latest) {
		res.Message = "이미 최신 버전 (" + current + ")"
		LogInfo("util: yt-dlp is up to date version=" + current)
		return res
	}

	notify("yt-dlp 업데이트 중... (" + latest + ")")
	LogInfo("util: yt-dlp update started current=" + current + " latest=" + latest)

	if err := installDirectComponent(paths.Data, c); err != nil {
		res.Message = "업데이트 실패: " + err.Error()
		LogError("util: yt-dlp update failed: " + err.Error())
		return res
	}

	if updated := getYtDlpVersion(c.TargetPath); updated != "" {
		res.Version = updated
	} else {
		res.Version = latest
	}

	res.Updated = true
	res.Message = fmt.Sprintf("업데이트 완료 (%s → %s)", current, res.Version)
	LogInfo("util: yt-dlp update completed version=" + res.Version)

	return res
}

// ensureYtDlpUpToDate는 런타임 점검(EnsureRuntime) 중 이미 설치된 yt-dlp를 갱신하고,
// 그 결과를 점검 결과에 반영한다.
func ensureYtDlpUpToDate(paths *util.AppPaths, c UtilComponent, opts UtilCheckOptions, result *UtilCheckResult) {
	if !opts.AutoRepair || !c.Downloadable {
		if v := getYtDlpVersion(c.TargetPath); v != "" {
			result.Versions[c.Key] = v
		}
		return
	}

	upd := runYtDlpUpdate(paths, c, nil)

	if upd.Version != "" {
		result.Versions[c.Key] = upd.Version
	}
	if upd.Updated {
		result.Updated = appendIfMissing(result.Updated, c.Key)
	}
}

// saveYtDlpVersion은 EnsureRuntime 밖에서 확인한 yt-dlp 버전을 util_ver.json에 반영한다.
func saveYtDlpVersion(confDir string, res *YtDlpUpdateResult) {
	if strings.TrimSpace(res.Version) == "" {
		return
	}

	confPath := filepath.Join(confDir, utilVersionFileName)

	info, err := loadUtilVersionInfo(confPath)
	if err != nil || info == nil {
		LogError("util: util_ver.json load failed for yt-dlp version record")
		return
	}

	info.Versions["yt-dlp"] = res.Version
	info.Installed["yt-dlp"] = true

	if err := writeJSON(confPath, info); err != nil {
		LogError("util: util_ver.json yt-dlp version record save failed: " + err.Error())
	}
}

func fetchLatestYtDlpVersion() (string, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, ytDlpLatestReleaseAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	// GitHub API는 User-Agent가 없으면 403을 반환한다.
	req.Header.Set("User-Agent", "s2qt")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http status: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&release); err != nil {
		return "", err
	}

	tag := strings.TrimSpace(release.TagName)
	if tag == "" {
		return "", fmt.Errorf("latest release tag is empty")
	}

	return tag, nil
}

// normalizeVersionTag는 릴리스 태그(v2026.02.04 등)와 --version 출력(2026.02.04)을
// 같은 형태로 맞춘다.
func normalizeVersionTag(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
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

	if err := replaceFile(stagedPath, c.TargetPath); err != nil {
		LogError("util: copy failed key=" + c.Key + " err=" + err.Error())
		return err
	}

	LogInfo("util: download completed key=" + c.Key)
	return nil
}

// replaceFile은 대상 파일이 이미 있어도 안전하게 교체한다.
// Windows에서는 잠긴 exe를 os.Create로 열 수 없는 경우가 있어,
// 기존 파일을 .old로 옮긴 뒤 새 파일을 복사하고 마지막에 .old를 정리한다.
func replaceFile(srcPath, dstPath string) error {
	if !fileExists(dstPath) {
		return CopyFile(srcPath, dstPath)
	}

	backupPath := dstPath + ".old"
	_ = os.Remove(backupPath)

	if err := os.Rename(dstPath, backupPath); err != nil {
		// 이름 변경이 불가능하면 직접 덮어쓰기를 시도한다.
		return CopyFile(srcPath, dstPath)
	}

	if err := CopyFile(srcPath, dstPath); err != nil {
		// 복사 실패 시 기존 파일을 되돌려 사용 불가 상태를 막는다.
		_ = os.Rename(backupPath, dstPath)
		return err
	}

	_ = os.Remove(backupPath)
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

	if err := CopyFile(ffmpegSrc, filepath.Join(binDir, "ffmpeg.exe")); err != nil {
		LogError("util: ffmpeg.exe copy failed: " + err.Error())
		return err
	}
	LogInfo("util: ffmpeg.exe copied")

	if err := CopyFile(ffprobeSrc, filepath.Join(binDir, "ffprobe.exe")); err != nil {
		LogError("util: ffprobe.exe copy failed: " + err.Error())
		return err
	}
	LogInfo("util: ffprobe.exe copied")

	return nil
}

func installPDFiumDLL(dataDir, binDir string) error {
	if strings.TrimSpace(defaultPDFiumPackageURL) == "" {
		return fmt.Errorf("pdfium package url not configured")
	}

	LogInfo("util: pdfium package download started")

	packagePath, err := downloadFile(dataDir, pdfiumPackageFileName, defaultPDFiumPackageURL)
	if err != nil {
		LogError("util: pdfium package download failed: " + err.Error())
		return err
	}

	LogInfo("util: pdfium package download completed")

	if err := verifyDownloadedFile(packagePath); err != nil {
		LogError("util: pdfium package verification failed: " + err.Error())
		return err
	}

	extractDir := filepath.Join(dataDir, "pdfium_extract")
	if err := os.RemoveAll(extractDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := ensureDir(extractDir); err != nil {
		return err
	}

	LogInfo("util: pdfium package extract started")
	if err := unzipFile(packagePath, extractDir); err != nil {
		LogError("util: pdfium package extract failed: " + err.Error())
		return err
	}
	LogInfo("util: pdfium package extract completed")

	pdfiumSrc, err := findPDFiumDLLInExtractDir(extractDir)
	if err != nil {
		LogError("util: pdfium.dll search failed: " + err.Error())
		return err
	}

	if err := ensureDir(binDir); err != nil {
		return err
	}

	targetPath := filepath.Join(binDir, pdfiumDllFileName)
	if err := CopyFile(pdfiumSrc, targetPath); err != nil {
		LogError("util: pdfium.dll copy failed: " + err.Error())
		return err
	}

	LogInfo("util: pdfium.dll copied to bin")

	if err := writePDFiumRuntimeDoc(dataDir, targetPath, pdfiumSrc); err != nil {
		// 문서 작성 실패는 런타임 설치 실패로 보지 않음
		LogError("util: pdfium runtime doc write failed: " + err.Error())
	} else {
		LogInfo("util: pdfium runtime doc saved")
	}

	return nil
}

func findPDFiumDLLInExtractDir(extractDir string) (string, error) {
	preferred := []string{
		filepath.Join(extractDir, "runtimes", "win-x64", "native", pdfiumDllFileName),
		filepath.Join(extractDir, "build", "native", "x64", pdfiumDllFileName),
		filepath.Join(extractDir, "x64", pdfiumDllFileName),
	}

	for _, p := range preferred {
		if fileExists(p) {
			return p, nil
		}
	}

	var foundX64 string
	var foundAny string

	err := filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.EqualFold(info.Name(), pdfiumDllFileName) {
			return nil
		}

		normalized := strings.ToLower(filepath.ToSlash(path))

		if foundAny == "" {
			foundAny = path
		}

		if strings.Contains(normalized, "win-x64") ||
			strings.Contains(normalized, "/x64/") ||
			strings.Contains(normalized, "x64") {
			foundX64 = path
			return io.EOF
		}

		return nil
	})

	if err != nil && err != io.EOF {
		return "", err
	}

	if foundX64 != "" {
		return foundX64, nil
	}
	if foundAny != "" {
		return foundAny, nil
	}

	return "", fmt.Errorf("pdfium.dll not found in extracted package")
}

func writePDFiumRuntimeDoc(dataDir, dllPath, sourcePath string) error {
	varDir := filepath.Dir(dataDir)
	docDir := filepath.Join(varDir, "doc")

	if err := ensureDir(docDir); err != nil {
		return err
	}

	docPath := filepath.Join(docDir, pdfiumRuntimeDocFileName)

	content := strings.TrimSpace(fmt.Sprintf(`
# PDFium Runtime

## Purpose

S2QT uses PDFium as the preferred runtime for converting generated PDF files into PNG images.

## Installed File

- bin/pdfium.dll

Current installed path:

%s

## Source File

The DLL was extracted from:

%s

## Downloaded Package

- Package: bblanchon.PDFium.Win32
- URL: %s

## S2QT PNG Policy

S2QT output policy:

- HTML: review/edit/preview
- PDF: official document
- PNG: shared image generated from PDF

The PDFium path should be used first for PDF-to-PNG conversion.
The existing HTML screenshot-based PNG generation remains as fallback.

## Packaging Policy

Current runtime placement:

- bin/pdfium.dll

Future candidate:

- s2qt.exe internal PDFium rendering using bin/pdfium.dll

## License Notice

Keep the relevant PDFium license and third-party notices with the application package before redistribution.
`, dllPath, sourcePath, defaultPDFiumPackageURL))

	return os.WriteFile(docPath, []byte(content+"\n"), 0o644)
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
	confPath := filepath.Join(confDir, utilVersionFileName)

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

	info.LastCheckedAt = result.CheckedAt
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

	checkedModel := false
	modelInstalled := true
	for _, key := range result.Checked {
		if strings.HasPrefix(key, "ggml-") && strings.HasSuffix(key, ".bin") {
			checkedModel = true
			if contains(result.Missing, key) {
				modelInstalled = false
			}
		}
	}
	if checkedModel {
		info.ModelInstalled = modelInstalled
	}

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

	cmd := newHiddenCommand(binPath, "--version")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
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

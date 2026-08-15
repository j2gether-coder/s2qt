package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"s2qt/util"
)

const (
	denoDownloadURL     = "https://github.com/denoland/deno/releases/latest/download/deno-x86_64-pc-windows-msvc.zip"
	denoPackageFileName = "deno-x86_64-pc-windows-msvc.zip"
	denoExeFileName     = "deno.exe"

	// jsRuntimeKey는 런타임 점검 결과/util_ver.json에서 JS 런타임을 가리키는 키다.
	jsRuntimeKey = "js-runtime"
)

// yt-dlp 2026.x부터 YouTube 추출에 JavaScript 런타임이 필요하다.
// 런타임이 없으면 일부 포맷을 얻지 못해 다운로드가 HTTP 403으로 실패한다.
// yt-dlp가 기본으로 찾는 런타임은 deno뿐이므로, 사용할 수 있는 런타임을 직접 찾아
// --js-runtimes 로 명시한다. (https://github.com/yt-dlp/yt-dlp/wiki/EJS)
var ytDlpJSRuntimeNames = []string{"deno", "node", "bun"}

// ytDlpJSRuntimeArgs는 사용 가능한 JS 런타임을 찾아 yt-dlp 인자로 반환한다.
// 찾지 못하면 nil을 반환하며, 이 경우 yt-dlp 기본 동작에 맡긴다.
func ytDlpJSRuntimeArgs() []string {
	name, path := findYtDlpJSRuntime()
	if name == "" {
		return nil
	}
	return []string{"--js-runtimes", name + ":" + path}
}

// findYtDlpJSRuntime은 앱에 동봉된 런타임을 먼저 찾고, 없으면 PATH에서 탐색한다.
func findYtDlpJSRuntime() (name string, path string) {
	if paths, err := util.GetAppPaths(); err == nil {
		for _, candidate := range ytDlpJSRuntimeNames {
			bundled := filepath.Join(paths.Bin, candidate+".exe")
			if FileExists(bundled) {
				return candidate, bundled
			}
		}
	}

	for _, candidate := range ytDlpJSRuntimeNames {
		found, err := exec.LookPath(candidate)
		if err != nil || strings.TrimSpace(found) == "" {
			continue
		}
		return candidate, found
	}

	return "", ""
}

// ytDlpJSRuntimeHint는 JS 런타임이 없을 때 사용자에게 보여줄 안내 문구다.
// 런타임이 있으면 빈 문자열을 반환한다.
func ytDlpJSRuntimeHint() string {
	if name, _ := findYtDlpJSRuntime(); name != "" {
		return ""
	}
	return "JavaScript 런타임(deno 또는 node)을 찾을 수 없습니다. " +
		"YouTube 다운로드에는 JS 런타임이 필요합니다."
}

// buildYtDlpArgs는 공통 인자(JS 런타임 지정)를 앞에 붙여 yt-dlp 인자를 구성한다.
func buildYtDlpArgs(args ...string) []string {
	return append(ytDlpJSRuntimeArgs(), args...)
}

// jsRuntimeVersion은 런타임의 버전 문자열 첫 줄을 반환한다.
func jsRuntimeVersion(path string) string {
	out, err := newHiddenCommand(path, "--version").Output()
	if err != nil {
		return ""
	}

	lines := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)
	return strings.TrimSpace(lines[0])
}

// installDenoRuntime은 deno 압축 파일을 받아 bin/deno.exe로 설치한다.
// yt-dlp가 기본으로 탐색하는 런타임이 deno이므로, 별도 설치가 필요한 환경에서는 deno를 받는다.
func installDenoRuntime(dataDir, binDir string) error {
	if strings.TrimSpace(denoDownloadURL) == "" {
		return fmt.Errorf("deno download url not configured")
	}

	LogInfo("util: deno package download started")

	zipPath, err := downloadFile(dataDir, denoPackageFileName, denoDownloadURL)
	if err != nil {
		LogError("util: deno package download failed: " + err.Error())
		return err
	}

	LogInfo("util: deno package download completed")

	if err := verifyDownloadedFile(zipPath); err != nil {
		LogError("util: deno package verification failed: " + err.Error())
		return err
	}

	extractDir := filepath.Join(dataDir, "deno_extract")
	if err := os.RemoveAll(extractDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := ensureDir(extractDir); err != nil {
		return err
	}

	LogInfo("util: deno package extract started")
	if err := unzipFile(zipPath, extractDir); err != nil {
		LogError("util: deno package extract failed: " + err.Error())
		return err
	}
	LogInfo("util: deno package extract completed")

	denoSrc, err := findFileRecursive(extractDir, denoExeFileName)
	if err != nil {
		LogError("util: deno.exe search failed: " + err.Error())
		return err
	}

	if err := ensureDir(binDir); err != nil {
		return err
	}

	if err := replaceFile(denoSrc, filepath.Join(binDir, denoExeFileName)); err != nil {
		LogError("util: deno.exe copy failed: " + err.Error())
		return err
	}

	LogInfo("util: deno.exe copied to bin")
	return nil
}

package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultPDFiumVersion = "149.0.7811"
	defaultPackageName   = "bblanchon.PDFium.Win32"
	defaultDLLName       = "pdfium.dll"
)

func main() {
	version := flag.String("version", defaultPDFiumVersion, "bblanchon.PDFium NuGet package version")
	outPath := flag.String("out", filepath.Join("bin", defaultDLLName), "target pdfium.dll path")
	stageDir := flag.String("stage", filepath.Join("var", "data", "pdfium_test"), "temporary download directory")
	docPath := flag.String("doc", filepath.Join("var", "doc", "pdfium_runtime_test.md"), "runtime test document path")
	flag.Parse()

	packageURL := fmt.Sprintf("https://www.nuget.org/api/v2/package/%s/%s", defaultPackageName, strings.TrimSpace(*version))
	packageFile := fmt.Sprintf("%s.%s.nupkg", defaultPackageName, strings.TrimSpace(*version))
	packagePath := filepath.Join(*stageDir, packageFile)

	fmt.Println("PDFium DLL download test")
	fmt.Println("package url :", packageURL)
	fmt.Println("stage path  :", packagePath)
	fmt.Println("target dll  :", *outPath)

	if err := ensureDir(*stageDir); err != nil {
		fail(err)
	}

	if err := downloadFile(packageURL, packagePath); err != nil {
		fail(fmt.Errorf("download failed: %w", err))
	}

	if err := verifyFile(packagePath); err != nil {
		fail(fmt.Errorf("package verification failed: %w", err))
	}

	selectedEntry, err := extractPDFiumDLLFromNuGet(packagePath, *outPath)
	if err != nil {
		fail(fmt.Errorf("pdfium.dll extract failed: %w", err))
	}

	if err := verifyFile(*outPath); err != nil {
		fail(fmt.Errorf("target dll verification failed: %w", err))
	}

	if err := writeRuntimeDoc(*docPath, packageURL, packagePath, *outPath, selectedEntry); err != nil {
		fmt.Println("warning: document write failed:", err)
	}

	fmt.Println("completed")
	fmt.Println("selected entry:", selectedEntry)
	fmt.Println("pdfium.dll    :", *outPath)
	fmt.Println("doc           :", *docPath)
}

func downloadFile(url string, targetPath string) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("url is empty")
	}

	if err := ensureDir(filepath.Dir(targetPath)); err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status: %s", resp.Status)
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}

	return out.Sync()
}

func extractPDFiumDLLFromNuGet(packagePath string, targetPath string) (string, error) {
	r, err := zip.OpenReader(packagePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var selected *zip.File

	// 1순위: Windows x64 native DLL
	preferredSuffixes := []string{
		"runtimes/win-x64/native/pdfium.dll",
		"build/native/x64/pdfium.dll",
		"x64/pdfium.dll",
	}

	for _, suffix := range preferredSuffixes {
		for _, f := range r.File {
			name := normalizeZipName(f.Name)
			if strings.EqualFold(name, suffix) {
				selected = f
				break
			}
		}
		if selected != nil {
			break
		}
	}

	// fallback: 패키지 안의 pdfium.dll 중 win-x64가 포함된 항목 우선
	if selected == nil {
		for _, f := range r.File {
			name := normalizeZipName(f.Name)
			if strings.HasSuffix(strings.ToLower(name), "/pdfium.dll") &&
				strings.Contains(strings.ToLower(name), "win-x64") {
				selected = f
				break
			}
		}
	}

	// fallback: 아무 pdfium.dll이나 선택
	if selected == nil {
		for _, f := range r.File {
			name := normalizeZipName(f.Name)
			if strings.EqualFold(filepath.Base(name), defaultDLLName) {
				selected = f
				break
			}
		}
	}

	if selected == nil {
		return "", fmt.Errorf("pdfium.dll not found in package")
	}

	if err := ensureDir(filepath.Dir(targetPath)); err != nil {
		return "", err
	}

	rc, err := selected.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	out, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil {
		return "", err
	}

	if err := out.Sync(); err != nil {
		return "", err
	}

	return selected.Name, nil
}

func writeRuntimeDoc(docPath, packageURL, packagePath, dllPath, selectedEntry string) error {
	if err := ensureDir(filepath.Dir(docPath)); err != nil {
		return err
	}

	content := strings.TrimSpace(fmt.Sprintf(`
# PDFium Runtime Test

## Purpose

This document records a temporary PDFium DLL download test for S2QT.

## Installed File

- %s

## Downloaded Package

- %s

## Selected Entry

- %s

## Source Package

- Package: bblanchon.PDFium
- URL: %s

## Notes

This is a temporary test flow.

If PDFium-based PDF to PNG conversion is verified successfully, the final runtime installation flow should be moved into util_service.go.

Target final placement:

- bin/pdfium.dll
- bin/pdfium_to_png.exe

The existing HTML screenshot-based PNG generation should remain as fallback until PDFium conversion is stable.
`, dllPath, packagePath, selectedEntry, packageURL))

	return os.WriteFile(docPath, []byte(content+"\n"), 0o644)
}

func verifyFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("path is directory: %s", path)
	}
	if info.Size() <= 0 {
		return fmt.Errorf("file size is zero: %s", path)
	}
	return nil
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func normalizeZipName(name string) string {
	return strings.ReplaceAll(strings.TrimSpace(name), "\\", "/")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "ERROR:", err)
	os.Exit(1)
}

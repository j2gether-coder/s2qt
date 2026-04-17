package service

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func FileExists(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func EnsureParentDir(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("대상 경로가 비어 있습니다")
	}

	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("대상 폴더 생성 실패: %w", err)
	}
	return nil
}

func CopyFile(srcPath, dstPath string) error {
	srcPath = strings.TrimSpace(srcPath)
	dstPath = strings.TrimSpace(dstPath)

	if srcPath == "" {
		return fmt.Errorf("원본 파일 경로가 비어 있습니다")
	}
	if dstPath == "" {
		return fmt.Errorf("대상 파일 경로가 비어 있습니다")
	}

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("원본 파일을 찾을 수 없습니다: %w", err)
	}
	if srcInfo.IsDir() {
		return fmt.Errorf("원본 경로가 파일이 아닙니다")
	}

	if err := EnsureParentDir(dstPath); err != nil {
		return err
	}

	in, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("원본 파일 열기 실패: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("대상 파일 생성 실패: %w", err)
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("파일 복사 실패: %w", err)
	}

	if err := out.Sync(); err != nil {
		return fmt.Errorf("파일 저장 확정 실패: %w", err)
	}

	return nil
}

func CopyFileToFixedPath(srcPath, dstPath string) (string, error) {
	if err := CopyFile(srcPath, dstPath); err != nil {
		return "", err
	}
	return dstPath, nil
}

const createNoWindow = 0x08000000

func newHiddenCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)

	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: createNoWindow,
		}
	}

	return cmd
}

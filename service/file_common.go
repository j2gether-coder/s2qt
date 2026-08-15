package service

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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

// scanLinesCR는 \n 뿐 아니라 \r(yt-dlp 진행바 등)로도 줄을 분리한다.
// 외부 프로세스가 진행률을 \r로 같은 줄에 갱신하더라도 실시간으로 토큰을 얻기 위함이다.
func scanLinesCR(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// runHiddenCommandStreaming은 newHiddenCommand와 동일하게 창을 숨긴 채 외부 명령을
// 실행하되, stdout/stderr를 줄 단위로 스트리밍하여 onLine 콜백으로 전달한다.
// 기존 CombinedOutput() 사용처와 호환되도록 전체 출력 문자열과 종료 에러를 함께 반환한다.
func runHiddenCommandStreaming(onLine func(line string), name string, args ...string) (string, error) {
	cmd := newHiddenCommand(name, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	var mu sync.Mutex
	var buf strings.Builder

	scan := func(r io.Reader) {
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		sc.Split(scanLinesCR)
		for sc.Scan() {
			line := sc.Text()
			mu.Lock()
			buf.WriteString(line)
			buf.WriteByte('\n')
			mu.Unlock()
			if onLine != nil {
				onLine(line)
			}
		}

		// 스캔이 중단되면(예: 한 줄이 버퍼 한도를 초과) 이후 출력이 조용히 사라진다.
		// 이 출력은 실패 시 에러 메시지로 쓰이므로, 잘렸다는 사실을 남긴다.
		if err := sc.Err(); err != nil {
			mu.Lock()
			buf.WriteString("[출력 읽기 중단] ")
			buf.WriteString(err.Error())
			buf.WriteByte('\n')
			mu.Unlock()
		}
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); scan(stdout) }()
	go func() { defer wg.Done(); scan(stderr) }()
	wg.Wait()

	waitErr := cmd.Wait()
	return buf.String(), waitErr
}

// percentInLine은 문자열 안의 "NN%" 또는 "NN.N%" 토큰에서 정수 퍼센트를 추출한다.
// 0~100 범위를 벗어나면 무시한다. 진행률 토큰이 없으면 ok=false.
func percentInLine(line string) (int, bool) {
	idx := strings.LastIndexByte(line, '%')
	if idx <= 0 {
		return 0, false
	}

	start := idx
	for start > 0 {
		c := line[start-1]
		if (c >= '0' && c <= '9') || c == '.' {
			start--
		} else {
			break
		}
	}

	token := strings.TrimSuffix(line[start:idx], ".")
	if token == "" {
		return 0, false
	}

	dot := strings.IndexByte(token, '.')
	if dot >= 0 {
		token = token[:dot]
	}
	if token == "" {
		return 0, false
	}

	n := 0
	for i := 0; i < len(token); i++ {
		n = n*10 + int(token[i]-'0')
	}
	if n < 0 || n > 100 {
		return 0, false
	}
	return n, true
}

// parseWhisperProgress은 whisper-cli -pp 출력 줄에서 전사 진행률을 추출한다.
// 예: "whisper_print_progress_callback: progress =  45%"
func parseWhisperProgress(line string) (int, bool) {
	if !strings.Contains(line, "progress") {
		return 0, false
	}
	return percentInLine(line)
}

// parseYtDlpProgress은 yt-dlp --newline 출력 줄에서 다운로드 진행률을 추출한다.
// 예: "[download]  35.6% of  45.00MiB at  1.20MiB/s ETA 00:30"
func parseYtDlpProgress(line string) (int, bool) {
	if !strings.Contains(line, "[download]") {
		return 0, false
	}
	return percentInLine(line)
}

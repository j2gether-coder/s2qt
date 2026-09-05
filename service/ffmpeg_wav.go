package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"s2qt/util"
)

// ffmpegTailLines는 실패 메시지에 남길 ffmpeg 출력 줄 수다.
// 배너를 끈 상태이므로 마지막 몇 줄이면 원인 파악에 충분하다.
const ffmpegTailLines = 12

// ffmpegOutputTail은 ffmpeg 출력에서 의미 있는 마지막 줄만 추린다.
// 배너/설정 문자열이 섞여 실제 오류가 묻히는 것을 막기 위함이다.
func ffmpegOutputTail(out string) string {
	normalized := strings.ReplaceAll(out, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	var lines []string
	for _, line := range strings.Split(normalized, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// ffmpeg 배너(버전/빌드/configuration)는 원인과 무관하다.
		if strings.HasPrefix(trimmed, "ffmpeg version") ||
			strings.HasPrefix(trimmed, "built with") ||
			strings.HasPrefix(trimmed, "configuration:") ||
			strings.HasPrefix(trimmed, "lib") {
			continue
		}
		lines = append(lines, trimmed)
	}

	if len(lines) == 0 {
		return "(ffmpeg 출력 없음)"
	}
	if len(lines) > ffmpegTailLines {
		lines = lines[len(lines)-ffmpegTailLines:]
	}

	return strings.Join(lines, "\n")
}

// mediaHasAudioStream은 입력 파일에 오디오 트랙이 있는지 ffprobe로 확인한다.
// ffprobe가 없거나 확인에 실패하면 판단을 보류(true)하고 ffmpeg 쪽 오류에 맡긴다.
func mediaHasAudioStream(ffprobeExe, path string) bool {
	if !fileExists(ffprobeExe) {
		return true
	}

	out, err := newHiddenCommand(ffprobeExe,
		"-v", "error",
		"-select_streams", "a",
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		path,
	).Output()
	if err != nil {
		return true
	}

	return strings.Contains(string(out), "audio")
}

// prepareWavOutput은 이전 실행이 남긴 WAV를 지운다.
// 다른 프로세스가 파일을 잡고 있으면 ffmpeg의 -y 덮어쓰기도 실패하므로,
// 이 단계에서 원인을 분명한 메시지로 알린다.
func prepareWavOutput(outputPath string) error {
	if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
		if fileExists(outputPath) {
			return fmt.Errorf("이전 WAV 파일을 삭제할 수 없습니다(다른 프로그램이 사용 중일 수 있습니다): %s\n%v", outputPath, err)
		}
	}
	return nil
}

// convertMediaToWav는 오디오/동영상 입력에서 오디오 트랙만 뽑아
// whisper가 요구하는 16kHz mono PCM WAV로 변환한다.
// 반환하는 문자열은 로그에 남길 ffmpeg 출력이다.
func convertMediaToWav(paths *util.AppPaths, inputPath, outputPath string) (string, error) {
	info, err := os.Stat(inputPath)
	if err != nil {
		return "", fmt.Errorf("변환할 입력 파일을 찾을 수 없습니다: %s\n%v", inputPath, err)
	}
	if info.Size() == 0 {
		return "", fmt.Errorf("변환할 입력 파일이 비어 있습니다: %s", inputPath)
	}

	if !mediaHasAudioStream(paths.FfprobeExe, inputPath) {
		return "", fmt.Errorf("입력 파일에 오디오 트랙이 없습니다: %s\n동영상이 영상 전용 포맷으로 내려받아졌을 수 있습니다. 다시 시도해 주세요.", inputPath)
	}

	if err := prepareWavOutput(outputPath); err != nil {
		return "", err
	}

	args := []string{
		"-hide_banner",
		"-nostdin",
		"-loglevel", "error",
		"-y",
		"-i", inputPath,
		// 오디오 트랙만 취한다. wav 컨테이너는 영상/표지 이미지를 담을 수 없으므로
		// 자동 스트림 선택에 맡기면 커버아트가 있는 파일에서 변환이 실패한다.
		"-vn",
		"-map", "0:a:0",
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		outputPath,
	}

	out, err := newHiddenCommand(paths.FfmpegExe, args...).CombinedOutput()
	log := string(out)
	if err != nil {
		detail := ffmpegOutputTail(log)
		if exitErr, ok := err.(*exec.ExitError); ok {
			return log, fmt.Errorf("WAV 변환 실패 (ffmpeg exit=%d)\n%s", exitErr.ExitCode(), detail)
		}
		return log, fmt.Errorf("WAV 변환 실패 (%v)\n%s", err, detail)
	}

	wavInfo, statErr := os.Stat(outputPath)
	if statErr != nil {
		return log, fmt.Errorf("WAV 파일이 생성되지 않았습니다: %s\n%s", outputPath, ffmpegOutputTail(log))
	}
	if wavInfo.Size() == 0 {
		return log, fmt.Errorf("생성된 WAV 파일이 비어 있습니다: %s\n%s", outputPath, ffmpegOutputTail(log))
	}

	return log, nil
}

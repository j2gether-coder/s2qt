package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"syscall"
)

// VideoMeta
// Step1 동영상 입력에서 사용할 메타정보 모델
type VideoMeta struct {
	Title          string `json:"title"`
	Uploader       string `json:"uploader"`
	Channel        string `json:"channel"`
	Thumbnail      string `json:"thumbnail"`
	Description    string `json:"description"`
	WebpageURL     string `json:"webpageUrl"`
	UploadDate     string `json:"uploadDate"`     // 원본값 예: 20260331
	UploadDateText string `json:"uploadDateText"` // 표시값 예: 2026-03-31
	Duration       int    `json:"duration"`       // 초 단위
	DurationText   string `json:"durationText"`   // 표시값 예: 01:12:05
}

// FetchVideoMeta
// yt-dlp --dump-json 기반 메타정보 조회
func FetchVideoMeta(ytdlpPath, url string) (*VideoMeta, error) {
	if ytdlpPath == "" {
		return nil, errors.New("yt-dlp path not set")
	}
	if url == "" {
		return nil, errors.New("URL is empty")
	}

	cmd := exec.Command(
		ytdlpPath,
		"--no-playlist",
		"--dump-json",
		url,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp execute failed: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("JSON parsing failed: %w", err)
	}

	// 라이브 영상 차단
	if v, ok := raw["is_live"].(bool); ok && v {
		return nil, errors.New("Live videos are not supported.")
	}

	meta := &VideoMeta{
		Title:       getString(raw, "title"),
		Uploader:    getString(raw, "uploader"),
		Channel:     getString(raw, "channel"),
		Thumbnail:   getString(raw, "thumbnail"),
		Description: getString(raw, "description"),
		WebpageURL:  getString(raw, "webpage_url"),
		UploadDate:  getString(raw, "upload_date"),
	}

	if v, ok := raw["duration"].(float64); ok {
		meta.Duration = int(math.Round(v))
	}

	if meta.Duration <= 0 {
		return nil, errors.New("Cannot retrieve video duration.")
	}

	meta.DurationText = formatDuration(meta.Duration)
	meta.UploadDateText = formatUploadDate(meta.UploadDate)

	return meta, nil
}

func formatUploadDate(raw string) string {
	if len(raw) != 8 {
		return raw
	}
	return raw[:4] + "-" + raw[4:6] + "-" + raw[6:8]
}

func formatDuration(sec int) string {
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

package service

import (
	"net/url"
	"strings"
)

// NormalizeVideoURL은 재생목록 등 부가 파라미터가 붙은 유튜브 URL을
// 영상 하나만 가리키는 표준 형태로 정리한다.
//
//	https://www.youtube.com/watch?v=ID&list=PL...&index=3 -> https://www.youtube.com/watch?v=ID
//	https://youtu.be/ID?list=PL...                        -> https://www.youtube.com/watch?v=ID
//
// 유튜브가 아니거나 /watch 형태가 아니면(shorts, live, embed 등) 원본을 그대로 돌려준다.
func NormalizeVideoURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return trimmed
	}

	host := strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")

	switch host {
	case "youtu.be":
		// youtu.be/ID 형태는 경로 첫 구간이 영상 ID다.
		id := strings.TrimPrefix(parsed.Path, "/")
		if i := strings.Index(id, "/"); i >= 0 {
			id = id[:i]
		}
		if id == "" {
			return trimmed
		}
		return youTubeWatchURL(id)

	case "youtube.com", "m.youtube.com", "music.youtube.com":
		if parsed.Path != "/watch" {
			return trimmed
		}
		id := strings.TrimSpace(parsed.Query().Get("v"))
		if id == "" {
			return trimmed
		}
		return youTubeWatchURL(id)
	}

	return trimmed
}

func youTubeWatchURL(videoID string) string {
	return "https://www.youtube.com/watch?v=" + videoID
}

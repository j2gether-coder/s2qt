package service

import (
	"strings"
	"testing"
)

func TestLooksHallucinatedTranscriptDetectsRepeatedLines(t *testing.T) {
	text := strings.Repeat("[두 번째 '두'두]\n", 8) + "거룩하고 자비하신 하나님께 감사드립니다."

	ok, reason := looksHallucinatedTranscript(text)
	if !ok {
		t.Fatalf("expected repeated transcript to be flagged, reason=%q", reason)
	}
}

func TestLooksHallucinatedTranscriptAllowsNormalText(t *testing.T) {
	text := strings.Join([]string{
		"오늘 본문은 믿음으로 살아가는 성도의 길을 말합니다.",
		"우리는 말씀 앞에서 마음을 낮추고 하나님의 뜻을 구합니다.",
		"각자의 자리에서 사랑을 실천하며 공동체를 세워 갑니다.",
	}, "\n")

	ok, reason := looksHallucinatedTranscript(text)
	if ok {
		t.Fatalf("expected normal transcript to pass, reason=%q", reason)
	}
}

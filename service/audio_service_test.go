package service

import (
	"fmt"
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

// 반복 구간 뒤에 정상 문장이 이어지면 기존 로직이 놓쳤던 케이스.
func TestLooksHallucinatedTranscriptDetectsRepeatBeforeNormalTail(t *testing.T) {
	text := "오늘 본문은 믿음으로 살아가는 성도의 길을 말합니다.\n" +
		strings.Repeat("시청해주셔서 감사합니다.\n", 7) +
		"우리는 말씀 앞에서 마음을 낮추고 하나님의 뜻을 구합니다.\n" +
		"각자의 자리에서 사랑을 실천하며 공동체를 세워 갑니다."

	ok, reason := looksHallucinatedTranscript(text)
	if !ok {
		t.Fatalf("expected mid-transcript repetition to be flagged, reason=%q", reason)
	}
}

// 긴 설교에서는 반복 구간의 비율이 낮아도 잡아야 한다.
func TestLooksHallucinatedTranscriptDetectsShortLoopInLongTranscript(t *testing.T) {
	body := make([]string, 0, 220)
	for i := 0; i < 200; i++ {
		body = append(body, fmt.Sprintf("본문 %d절을 보면 하나님께서 그 백성에게 새로운 길을 열어 주십니다.", i+1))
	}
	// 전체의 6%밖에 되지 않는 짧은 반복 루프도 잡아야 한다.
	for i := 0; i < 12; i++ {
		body = append(body, "감사합니다")
	}

	ok, reason := looksHallucinatedTranscript(strings.Join(body, "\n"))
	if !ok {
		t.Fatalf("expected short trailing loop to be flagged, reason=%q", reason)
	}
}

// 한 줄 안에 반복이 몰려 나오는 케이스.
func TestLooksHallucinatedTranscriptDetectsSingleLineLoop(t *testing.T) {
	text := "오늘 본문을 함께 나누겠습니다. " + strings.Repeat("구독과 좋아요 부탁드립니다. ", 6)

	ok, reason := looksHallucinatedTranscript(text)
	if !ok {
		t.Fatalf("expected single-line repetition to be flagged, reason=%q", reason)
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

// 정상 전사문에 짧은 문구(아멘 등)가 흩어져 반복되는 것만으로는 걸리지 않아야 한다.
func TestLooksHallucinatedTranscriptAllowsScatteredShortPhrases(t *testing.T) {
	text := strings.Join([]string{
		"오늘 우리가 함께 읽은 본문은 요한복음 십오 장입니다.",
		"예수님은 자신을 참포도나무라고 소개하십니다.",
		"아멘",
		"가지가 나무에 붙어 있어야 열매를 맺을 수 있습니다.",
		"우리의 삶도 마찬가지라고 말씀하십니다.",
		"기도의 자리를 떠나면 우리는 금세 메마르게 됩니다.",
		"아멘",
		"그래서 매일 아침 말씀 앞에 앉는 훈련이 필요합니다.",
		"작은 순종이 쌓여서 삶의 방향을 바꾸어 놓습니다.",
		"이번 한 주간 각자의 자리에서 그 열매를 맺기를 바랍니다.",
		"아멘",
		"함께 기도하겠습니다.",
	}, "\n")

	ok, reason := looksHallucinatedTranscript(text)
	if ok {
		t.Fatalf("expected scattered short phrases to pass, reason=%q", reason)
	}
}

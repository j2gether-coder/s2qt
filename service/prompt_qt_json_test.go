package service

import (
	"strings"
	"testing"
)

func testQTMeta(audience string) QTMeta {
	return QTMeta{
		Title:     "테스트 제목",
		BibleText: "로마서 8:26-28",
		RawText:   "전사문 본문입니다.",
		Audience:  audience,
	}
}

// 인포그래픽 블록은 장년(adult) 실행에만 붙어야 한다.
func TestBuildQTPromptJSON_InfographicOnlyForAdult(t *testing.T) {
	const marker = "[Infographic Output Contract"

	adult := BuildQTPromptJSON(testQTMeta(AudienceAdult))
	if !strings.Contains(adult, marker) {
		t.Fatalf("adult 프롬프트에 인포그래픽 블록이 없습니다")
	}
	if !strings.Contains(adult, `"version" 값을 "1.0"이 아니라 "1.1"로 출력한다`) {
		t.Errorf("adult 프롬프트에 version 1.1 오버라이드 지시가 없습니다")
	}

	for _, audience := range []string{"young_adult", "teen", "child"} {
		prompt := BuildQTPromptJSON(testQTMeta(audience))
		if strings.Contains(prompt, marker) {
			t.Errorf("%s 프롬프트에 인포그래픽 블록이 붙었습니다", audience)
		}
		if strings.Contains(prompt, `"1.1"`) {
			t.Errorf("%s 프롬프트에 version 1.1 언급이 있습니다", audience)
		}
	}
}

// 메타데이터 날조 방지 규칙은 모든 연령대에 공통으로 적용한다.
func TestBuildQTPromptJSON_MetadataFidelityForAllAudiences(t *testing.T) {
	for _, audience := range []string{AudienceAdult, "young_adult", "teen", "child"} {
		prompt := BuildQTPromptJSON(testQTMeta(audience))
		if !strings.Contains(prompt, "METADATA FIDELITY") {
			t.Errorf("%s 프롬프트에 메타데이터 규칙이 없습니다", audience)
		}
	}
}

// 인포그래픽 블록은 전사문 바로 앞에 와야 한다.
// 응답이 잘려도 QT 본체가 온전하게 남도록 하기 위함이다.
func TestBuildQTPromptJSON_InfographicBeforeTranscript(t *testing.T) {
	prompt := BuildQTPromptJSON(testQTMeta(AudienceAdult))

	infoIdx := strings.Index(prompt, "[Infographic Output Contract")
	transcriptIdx := strings.Index(prompt, "[Sermon Transcript]")

	if infoIdx < 0 || transcriptIdx < 0 {
		t.Fatalf("블록을 찾을 수 없습니다: infographic=%d transcript=%d", infoIdx, transcriptIdx)
	}
	if infoIdx > transcriptIdx {
		t.Errorf("인포그래픽 블록이 전사문 뒤에 있습니다")
	}
}

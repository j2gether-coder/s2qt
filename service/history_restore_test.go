package service

import (
	"encoding/json"
	"testing"
)

// 구버전 이력(Step2 평면 페이로드)의 모양을 재현한다.
func flatHistoryJSON(audience string) string {
	payload := map[string]any{
		"audience":      audience,
		"title":         "[QT] 평면 제목",
		"bibleText":     "로마서 8:26-28",
		"summaryTitle":  "말씀의 길잡이",
		"summaryBody":   "평면 요약",
		"messageTitle1": "요점 1", "messageBody1": "본문 1",
		"messageTitle2": "요점 2", "messageBody2": "본문 2",
		"messageTitle3": "요점 3", "messageBody3": "본문 3",
		"reflectionItem1": "묵상 1",
		"reflectionItem2": "묵상 2",
		"reflectionItem3": "묵상 3",
		"prayerTitle":     "오늘의 기도",
		"prayerBody":      "평면 기도",
	}

	b, _ := json.Marshal(payload)
	return string(b)
}

// 신규 이력은 LLM 원본 경로로 복원되고, 인포그래픽 데이터가 함께 나와야 한다.
func TestRestoreQTSectionDoc_DetectsLLMOriginal(t *testing.T) {
	doc, llmDoc, err := restoreQTSectionDoc(llmJSON(AudienceAdult, true), AudienceAdult)
	if err != nil {
		t.Fatalf("복원 실패: %v", err)
	}

	if doc.Version != "1.0" {
		t.Errorf("version = %q, want 1.0", doc.Version)
	}
	if len(doc.Sections) == 0 {
		t.Errorf("sections가 복원되지 않았습니다")
	}
	if llmDoc == nil || llmDoc.Infographic == nil {
		t.Fatalf("인포그래픽 데이터가 복원되지 않았습니다")
	}
	if reasons := ValidateInfographic(llmDoc.Infographic); len(reasons) > 0 {
		t.Errorf("복원된 인포그래픽이 검증을 통과하지 못했습니다: %v", reasons)
	}

	// blog 전용 필드도 원본에서 그대로 살아나야 한다.
	// (평면 경로에서는 유실되던 값이다)
	if doc.Metadata["support_scriptures_full"] == nil {
		t.Errorf("support_scriptures_full이 복원되지 않았습니다")
	}
}

// 구버전 이력은 평면 경로로 복원되며, 인포그래픽 데이터는 없다.
func TestRestoreQTSectionDoc_FallsBackToFlat(t *testing.T) {
	doc, llmDoc, err := restoreQTSectionDoc(flatHistoryJSON(AudienceAdult), AudienceAdult)
	if err != nil {
		t.Fatalf("구버전 이력 복원 실패: %v", err)
	}

	if llmDoc != nil {
		t.Errorf("구버전 이력인데 LLM 문서가 반환되었습니다")
	}
	if doc.Version != "1.0" {
		t.Errorf("version = %q, want 1.0", doc.Version)
	}

	if got := getStringFromMap(doc.Metadata, "title"); got != "[QT] 평면 제목" {
		t.Errorf("title = %q", got)
	}

	// 평면 경로는 4개 섹션을 고정으로 만든다.
	if len(doc.Sections) != 4 {
		t.Fatalf("섹션 수 = %d, want 4", len(doc.Sections))
	}
	if doc.Sections[0].Blocks[0].Text != "평면 요약" {
		t.Errorf("요약 본문이 복원되지 않았습니다: %q", doc.Sections[0].Blocks[0].Text)
	}
}

// 인포그래픽이 없는 신규 이력(비장년)도 원본 경로로 복원되어야 한다.
func TestRestoreQTSectionDoc_LLMOriginalWithoutInfographic(t *testing.T) {
	doc, llmDoc, err := restoreQTSectionDoc(llmJSON("teen", false), "teen")
	if err != nil {
		t.Fatalf("복원 실패: %v", err)
	}

	if doc.Audience != "teen" {
		t.Errorf("audience = %q, want teen", doc.Audience)
	}
	if llmDoc == nil {
		t.Fatalf("LLM 문서가 반환되지 않았습니다")
	}
	if llmDoc.Infographic != nil {
		t.Errorf("비장년인데 인포그래픽 데이터가 있습니다")
	}
}

func TestRestoreQTSectionDoc_RejectsEmptyAndBroken(t *testing.T) {
	tests := []struct {
		name     string
		jsonText string
	}{
		{"빈 문자열", "   "},
		{"깨진 JSON", "{ not json"},
		{"내용 없는 평면 페이로드", `{"audience":"adult"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := restoreQTSectionDoc(tt.jsonText, AudienceAdult); err == nil {
				t.Errorf("에러가 반환되지 않았습니다")
			}
		})
	}
}

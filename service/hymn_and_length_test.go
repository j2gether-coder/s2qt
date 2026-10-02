package service

import (
	"strings"
	"testing"
)

// ── 길이 규칙 ─────────────────────────────────────────────

// 설교요약문이 QT summary/prayer를 재활용하므로, 인포그래픽에 들어갈 만한
// 길이로 줄였다. 문장 수와 글자 수를 함께 지시해야 한다 —
// 문장 수만 적으면 모델이 한 문장을 길게 늘여 결과가 그대로다.
func TestBuildQTPromptJSON_AdultLengthRules(t *testing.T) {
	prompt := BuildQTPromptJSON(testQTMeta(AudienceAdult))

	for _, want := range []string{
		"Summary: 3-4 sentences, about 150-180 Korean characters",
		"Prayer: 3-4 sentences, about 120-150 Korean characters",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("장년 길이 규칙이 없습니다: %q", want)
		}
	}

	if strings.Contains(prompt, "Summary: 5-6 sentences") {
		t.Errorf("옛 Summary 길이 규칙이 남아 있습니다")
	}
	if strings.Contains(prompt, "Prayer: 5-6 sentences") {
		t.Errorf("옛 Prayer 길이 규칙이 남아 있습니다")
	}
}

// 문장 수만 줄이라고 하면 모델이 회개나 감사를 통째로 버린다.
// "압축하되 빼지 말 것"이 4단계 구조와 함께 있어야 한다.
func TestBuildQTPromptJSON_PrayerKeepsStructure(t *testing.T) {
	prompt := BuildQTPromptJSON(testQTMeta(AudienceAdult))

	if !strings.Contains(prompt, "경배 → 회개 → 감사 → 간구") {
		t.Fatalf("기도 4단계 구조 지시가 없습니다")
	}
	if !strings.Contains(prompt, "네 단계를 모두 담되 압축한다") {
		t.Errorf("단계를 빼지 말라는 지시가 없습니다")
	}
}

// md는 장년 전용이므로 다른 연령대의 길이는 각자 이유대로 둔다.
func TestBuildQTPromptJSON_NonAdultLengthUnchanged(t *testing.T) {
	tests := []struct {
		audience string
		want     string
	}{
		{"young_adult", "Summary: 4-5 sentences"},
		{"teen", "Summary: 3-4 short sentences"},
		{"child", "Summary: 2-3 very simple sentences"},
	}

	for _, tt := range tests {
		prompt := BuildQTPromptJSON(testQTMeta(tt.audience))
		if !strings.Contains(prompt, tt.want) {
			t.Errorf("%s의 길이 규칙이 바뀌었습니다: %q 없음", tt.audience, tt.want)
		}
	}
}

// ── 찬송가 ────────────────────────────────────────────────

// 모델에게 가장 강한 신호는 설명문이 아니라 스키마 예시다.
// 거기에 구체적인 곡명이 박혀 있으면 입력값을 두고도 그것을 베낀다.
func TestBuildQTPromptJSON_HymnPlaceholderInSchema(t *testing.T) {
	meta := testQTMeta(AudienceAdult)
	meta.Hymn = "새찬송가 488장"

	schema := schemaTemplateBlock(t, BuildQTPromptJSON(meta))

	if !strings.Contains(schema, `"hymn": "새찬송가 488장"`) {
		t.Errorf("스키마 템플릿에 입력값이 치환되지 않았습니다:\n%s", schema)
	}
	// 반례로 쓰는 456장은 METADATA FIDELITY에 남아 있어도 된다.
	// 스키마 템플릿 안에 있으면 모델이 그대로 베낀다.
	if strings.Contains(schema, "새찬송가 456장") {
		t.Errorf("스키마 템플릿에 하드코딩된 곡명이 남아 있습니다")
	}
}

// schemaTemplateBlock은 [JSON Schema Template] 이후 QUALITY CHECKS 전까지를 돌려준다.
func schemaTemplateBlock(t *testing.T, prompt string) string {
	t.Helper()

	start := strings.Index(prompt, "[JSON Schema Template")
	if start < 0 {
		t.Fatalf("스키마 템플릿 블록을 찾을 수 없습니다")
	}

	rest := prompt[start:]
	if end := strings.Index(rest, "QUALITY CHECKS"); end > 0 {
		rest = rest[:end]
	}
	return rest
}

// hymn은 빈 값을 채워도 되는 유일한 필드다. 조건부 규칙이 없으면
// 스키마가 "hymn": "" 로 치환될 때 모델이 빈 값을 그대로 복사해 추천이 죽는다.
func TestBuildQTPromptJSON_HymnFidelityRule(t *testing.T) {
	prompt := BuildQTPromptJSON(testQTMeta(AudienceAdult))

	if !strings.Contains(prompt, "hymn 은 조건부다") {
		t.Fatalf("METADATA FIDELITY에 hymn 조건부 규칙이 없습니다")
	}
	for _, want := range []string{
		"글자 그대로 복사한다",
		"한국 찬송가 1개를 추천한다",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("hymn 규칙에 %q 가 없습니다", want)
		}
	}
}

func TestBuildQTPromptJSON_EmptyHymnLeavesPlaceholderBlank(t *testing.T) {
	prompt := BuildQTPromptJSON(testQTMeta(AudienceAdult))

	if !strings.Contains(schemaTemplateBlock(t, prompt), `"hymn": ""`) {
		t.Errorf("입력이 빈 경우 스키마가 빈 값으로 치환되지 않았습니다")
	}
	// 빈 값이어도 추천 지시가 살아 있어야 한다.
	if !strings.Contains(prompt, "한국 찬송가 1개를 추천한다") {
		t.Errorf("빈 입력에서 추천 지시가 사라졌습니다")
	}
}

// ── Step1 저장 ────────────────────────────────────────────

func hymnRequest(screenHymn string) *QTStep1SaveRequest {
	req := baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true))
	req.Hymn = screenHymn
	return req
}

func tempJSONHymn(t *testing.T, path string) string {
	t.Helper()

	meta, _ := readTempJSONMap(t, path)["metadata"].(map[string]any)
	if meta == nil {
		t.Fatalf("metadata가 없습니다")
	}
	s, _ := meta["hymn"].(string)
	return s
}

// 화면에 입력한 찬송가가 LLM이 지어낸 값을 이긴다.
func TestStep1Save_HymnFromScreenWins(t *testing.T) {
	svc := newTestStep1Service(t)

	if _, err := svc.Save(hymnRequest("새찬송가 488장")); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	if got := tempJSONHymn(t, svc.Paths.TempJson); got != "새찬송가 488장" {
		t.Errorf("화면 입력 찬송가가 쓰이지 않았습니다: %q", got)
	}
}

// 화면이 비면 LLM의 추천을 살린다. 찬송가는 조건부다.
func TestStep1Save_EmptyHymnKeepsLLMRecommendation(t *testing.T) {
	svc := newTestStep1Service(t)

	for _, screen := range []string{"", "   "} {
		if _, err := svc.Save(hymnRequest(screen)); err != nil {
			t.Fatalf("screen=%q 저장 실패: %v", screen, err)
		}

		if got := tempJSONHymn(t, svc.Paths.TempJson); got != llmHymn {
			t.Errorf("screen=%q: LLM 추천이 유지되지 않았습니다: %q", screen, got)
		}
	}
}

// 접두어를 떼는 정규화가 temp.json과 작업내역 양쪽에 같게 적용되어야 한다.
// 한쪽만 정규화하면 목록의 찬송과 산출물의 찬송이 갈린다.
func TestStep1Save_HymnNormalizedConsistently(t *testing.T) {
	svc := newTestStep1Service(t)

	if _, err := svc.Save(hymnRequest("찬송가: 새찬송가 488장")); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	const want = "새찬송가 488장"
	if got := tempJSONHymn(t, svc.Paths.TempJson); got != want {
		t.Errorf("temp.json의 찬송가가 정규화되지 않았습니다: %q", got)
	}
}

// Save()가 정규화한 값을 saveHistory에도 그대로 넘기는지는 DB가 필요해
// 단위 테스트로 확인하지 않는다. 앱 확인 항목 H3로 남긴다.
// 여기서는 두 곳이 공유하는 정규화 계약만 고정한다.
func TestNormalizeHymnText_StripsPrefixes(t *testing.T) {
	tests := map[string]string{
		"찬송가: 새찬송가 488장": "새찬송가 488장",
		"찬송: 488장":       "488장",
		"  새찬송가 488장  ":  "새찬송가 488장",
		"":               "",
	}

	for in, want := range tests {
		if got := normalizeHymnText(in); got != want {
			t.Errorf("normalizeHymnText(%q) = %q, want %q", in, got, want)
		}
	}
}

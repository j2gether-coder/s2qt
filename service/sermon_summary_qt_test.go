package service

import (
	"strings"
	"testing"
)

// qtSections는 설교요약문이 재활용하는 3개 섹션을 갖춘 문서를 만든다.
func qtSections() []QTSectionData {
	return []QTSectionData{
		{Type: "summary", Title: "말씀의 길잡이", Blocks: []QTBlockData{
			{Type: "paragraph", Text: "섹션에서 온 길잡이입니다."},
		}},
		{Type: "message", Title: "오늘의 메시지", Blocks: []QTBlockData{
			{Type: "message_title", Text: "첫째 제목"},
			{Type: "paragraph", Text: "첫째 본문입니다."},
		}},
		{Type: "reflection", Title: "깊은 묵상과 적용", Blocks: []QTBlockData{
			{Type: "list", Items: []string{"섹션 적용 하나.", "섹션 적용 둘."}},
		}},
		{Type: "prayer", Title: "오늘의 기도", Blocks: []QTBlockData{
			{Type: "paragraph", Text: "섹션에서 온 기도입니다."},
		}},
	}
}

// ── 조립 우선순위 ─────────────────────────────────────────

// 2026-10-01 이전 이력에는 저장된 guide/apply/prayer가 있다. 그것이 이긴다.
func TestResolveSermonSummaryQT_StoredWins(t *testing.T) {
	qt := resolveSermonSummaryQT(legacyInfographic(), qtSections())

	if qt.Guide != "저장된 길잡이입니다." {
		t.Errorf("저장된 guide가 쓰이지 않았습니다: %q", qt.Guide)
	}
	if qt.Prayer != "저장된 기도입니다." {
		t.Errorf("저장된 prayer가 쓰이지 않았습니다: %q", qt.Prayer)
	}
	if len(qt.Apply) != 2 || qt.Apply[0] != "저장된 적용 하나." {
		t.Errorf("저장된 apply가 쓰이지 않았습니다: %v", qt.Apply)
	}
}

// 새 3필드 JSON에는 없으므로 QT 섹션에서 가져온다.
func TestResolveSermonSummaryQT_FallsBackToSections(t *testing.T) {
	qt := resolveSermonSummaryQT(validInfographic(), qtSections())

	if qt.Guide != "섹션에서 온 길잡이입니다." {
		t.Errorf("섹션의 guide가 쓰이지 않았습니다: %q", qt.Guide)
	}
	if qt.Prayer != "섹션에서 온 기도입니다." {
		t.Errorf("섹션의 prayer가 쓰이지 않았습니다: %q", qt.Prayer)
	}
	if len(qt.Apply) != 2 || qt.Apply[0] != "섹션 적용 하나." {
		t.Errorf("섹션의 apply가 쓰이지 않았습니다: %v", qt.Apply)
	}
}

// 일부만 저장된 경우 — 저장된 것만 이기고 나머지는 섹션에서 온다.
func TestResolveSermonSummaryQT_PartialStored(t *testing.T) {
	data := validInfographic()
	data.Guide = "저장된 길잡이만 있습니다."

	qt := resolveSermonSummaryQT(data, qtSections())

	if qt.Guide != "저장된 길잡이만 있습니다." {
		t.Errorf("저장된 guide가 쓰이지 않았습니다: %q", qt.Guide)
	}
	if qt.Prayer != "섹션에서 온 기도입니다." {
		t.Errorf("prayer는 섹션에서 와야 합니다: %q", qt.Prayer)
	}
}

func TestResolveSermonSummaryQT_NilData(t *testing.T) {
	qt := resolveSermonSummaryQT(nil, qtSections())

	if qt.Guide != "섹션에서 온 길잡이입니다." {
		t.Errorf("nil 입력에서 섹션 추출이 동작하지 않았습니다: %q", qt.Guide)
	}
}

func TestExtractSermonSummaryQT_MissingSections(t *testing.T) {
	qt := extractSermonSummaryQT([]QTSectionData{
		{Type: "summary", Blocks: []QTBlockData{{Type: "paragraph", Text: "길잡이만 있습니다."}}},
	})

	if qt.Guide != "길잡이만 있습니다." {
		t.Errorf("guide 추출 실패: %q", qt.Guide)
	}
	if qt.Prayer != "" || len(qt.Apply) != 0 {
		t.Errorf("없는 섹션이 빈 값이 아닙니다: prayer=%q apply=%v", qt.Prayer, qt.Apply)
	}
}

// paragraph가 아닌 블록이 앞에 있어도 본문을 찾아야 한다.
func TestExtractSermonSummaryQT_SkipsNonParagraphBlocks(t *testing.T) {
	qt := extractSermonSummaryQT([]QTSectionData{
		{Type: "summary", Blocks: []QTBlockData{
			{Type: "message_title", Text: "제목 블록"},
			{Type: "paragraph", Text: "진짜 본문입니다."},
		}},
	})

	if qt.Guide != "진짜 본문입니다." {
		t.Errorf("paragraph 블록을 건너뛰지 못했습니다: %q", qt.Guide)
	}
}

// ── 렌더 ──────────────────────────────────────────────────

func TestRenderInfographicMD_ReusesQTSections(t *testing.T) {
	md := RenderInfographicMD(validInfographic(), extractSermonSummaryQT(qtSections()), SermonSummaryMeta{
		Title:     "제목",
		BibleText: "로마서 8:26-28",
	})

	for _, want := range []string{
		"## 말씀의 길잡이\n섹션에서 온 길잡이입니다.",
		"- 섹션 적용 하나.",
		"## 오늘의 기도\n섹션에서 온 기도입니다.",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("%q 가 렌더되지 않았습니다:\n%s", want, md)
		}
	}
}

// 내용이 QT reflection이 되었으므로 제목도 그에 맞춘다.
func TestRenderInfographicMD_ApplyHeadingRenamed(t *testing.T) {
	md := RenderInfographicMD(validInfographic(), validSermonSummaryQT(), SermonSummaryMeta{
		Title:     "제목",
		BibleText: "로마서 8:26-28",
	})

	if !strings.Contains(md, "## 깊은 묵상과 적용") {
		t.Errorf("'깊은 묵상과 적용' 제목이 없습니다")
	}
	if strings.Contains(md, "## 오늘의 적용") {
		t.Errorf("옛 제목 '오늘의 적용'이 남아 있습니다")
	}
}

// 길잡이·기도는 더 이상 ValidateInfographic이 비어 있지 않음을 보장하지 않는다.
// 비면 제목만 남지 않도록 섹션째 생략한다.
func TestRenderInfographicMD_OmitsEmptyQTSections(t *testing.T) {
	md := RenderInfographicMD(validInfographic(), SermonSummaryQT{}, SermonSummaryMeta{
		Title:     "제목",
		BibleText: "로마서 8:26-28",
	})

	for _, unwanted := range []string{"## 말씀의 길잡이", "## 깊은 묵상과 적용", "## 오늘의 기도"} {
		if strings.Contains(md, unwanted) {
			t.Errorf("%q 가 비었는데 섹션이 남아 있습니다:\n%s", unwanted, md)
		}
	}

	// LLM이 쓴 부분은 그대로 남아야 한다.
	if !strings.Contains(md, "## 말씀의 핵심") {
		t.Errorf("다른 섹션까지 사라졌습니다:\n%s", md)
	}
}

// ── 검증 ──────────────────────────────────────────────────

// guide/apply/prayer가 없는 것이 정상이다.
// 검증에 남아 있으면 장년 md 생성이 전부 막힌다.
func TestValidateInfographic_OnlyThreeFields(t *testing.T) {
	if reasons := ValidateInfographic(validInfographic()); len(reasons) != 0 {
		t.Errorf("3필드 데이터가 거부되었습니다: %v", reasons)
	}
}

// ── Step1 ─────────────────────────────────────────────────

func TestStep1Save_SermonSummaryUsesQTSections(t *testing.T) {
	svc := newTestStep1Service(t)

	if _, err := svc.Save(baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true))); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	md := readFileString(t, svc.Paths.TempSermonSummary)

	if !strings.Contains(md, "## 말씀의 길잡이\n요약 본문입니다.") {
		t.Errorf("길잡이가 QT summary에서 오지 않았습니다:\n%s", md)
	}
	if !strings.Contains(md, "- 첫째 묵상입니다.") {
		t.Errorf("적용이 QT reflection에서 오지 않았습니다:\n%s", md)
	}
	if !strings.Contains(md, "## 오늘의 기도\n기도문입니다.") {
		t.Errorf("기도가 QT prayer에서 오지 않았습니다:\n%s", md)
	}
}

// 새 temp.json에는 쓰지 않는 키가 남지 않아야 한다(omitempty).
func TestStep1Save_InfographicOmitsLegacyKeys(t *testing.T) {
	svc := newTestStep1Service(t)

	if _, err := svc.Save(baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true))); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	meta, _ := readTempJSONMap(t, svc.Paths.TempJson)["metadata"].(map[string]any)
	info, _ := meta["infographic"].(map[string]any)
	if info == nil {
		t.Fatalf("metadata.infographic이 없습니다")
	}

	for _, key := range []string{"guide", "apply", "prayer"} {
		if _, exists := info[key]; exists {
			t.Errorf("쓰지 않는 키 %q가 남아 있습니다", key)
		}
	}
}

// ── Step2 ─────────────────────────────────────────────────

// 이번 작업의 핵심 — Step2에서 고친 QT 본문이 md에 반영된다.
func TestStep2Save_QTEditsReachSermonSummary(t *testing.T) {
	step1, step2 := newTestStep12Services(t)

	if _, err := step1.Save(baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true))); err != nil {
		t.Fatalf("step1 저장 실패: %v", err)
	}

	data := loadStep2(t, step2)
	data.PrayerBody = "Step2에서 고친 기도입니다."
	data.SummaryBody = "Step2에서 고친 길잡이입니다."
	data.ReflectionItem1 = "Step2에서 고친 묵상입니다."

	if err := step2.Save(data); err != nil {
		t.Fatalf("step2 저장 실패: %v", err)
	}

	md := readFileString(t, step2.Paths.TempSermonSummary)

	for _, want := range []string{
		"## 말씀의 길잡이\nStep2에서 고친 길잡이입니다.",
		"- Step2에서 고친 묵상입니다.",
		"## 오늘의 기도\nStep2에서 고친 기도입니다.",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("%q 가 md에 반영되지 않았습니다:\n%s", want, md)
		}
	}
	if strings.Contains(md, "기도문입니다.") {
		t.Errorf("Step1 시점의 기도가 남아 있습니다:\n%s", md)
	}
}

// ── 재작업 ────────────────────────────────────────────────

// 확정 4 — 옛 6필드 이력은 저장된 값으로 복원된다.
func TestRestoreQTSectionDoc_LegacyInfographicSurvives(t *testing.T) {
	doc, llmDoc, err := restoreQTSectionDoc(legacyLLMJSON(), AudienceAdult)
	if err != nil {
		t.Fatalf("복원 실패: %v", err)
	}
	if llmDoc == nil || llmDoc.Infographic == nil {
		t.Fatalf("infographic이 복원되지 않았습니다")
	}

	if llmDoc.Infographic.Guide == "" {
		t.Errorf("옛 guide가 유실되었습니다 — InfographicData에서 필드를 지웠는지 확인하십시오")
	}

	qt := resolveSermonSummaryQT(llmDoc.Infographic, doc.Sections)
	if qt.Guide != "저장된 길잡이입니다." {
		t.Errorf("재작업에서 저장된 guide가 쓰이지 않았습니다: %q", qt.Guide)
	}
}

// legacyLLMJSON은 2026-10-01 이전에 저장된 6필드 이력을 흉내 낸다.
func legacyLLMJSON() string {
	raw := llmJSON(AudienceAdult, true)

	// infographic 객체만 옛 형태로 바꾼다.
	old := `"infographic":{"follow"`
	if !strings.Contains(raw, old) {
		panic("llmJSON 구조가 바뀌었습니다 — legacyLLMJSON을 고치십시오")
	}

	return strings.Replace(raw, old,
		`"infographic":{"guide":"저장된 길잡이입니다.",`+
			`"apply":["저장된 적용 하나.","저장된 적용 둘."],`+
			`"prayer":"저장된 기도입니다.","follow"`, 1)
}

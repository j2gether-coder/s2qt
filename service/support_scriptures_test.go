package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"s2qt/util"
)

// ── 렌더러 ────────────────────────────────────────────────

func TestRenderInfographicMD_SupportScriptures(t *testing.T) {
	md := RenderInfographicMD(validInfographic(), validSermonSummaryQT(), SermonSummaryMeta{
		Title:             "제목",
		BibleText:         "로마서 8:26-28",
		SupportScriptures: []string{"시편 27:8", "요한복음 14:9", "고린도후서 3:18"},
	})

	if !strings.Contains(md, "## 관련 성구\n시편 27:8, 요한복음 14:9, 고린도후서 3:18\n") {
		t.Errorf("관련 성구가 쉼표로 이어진 한 줄로 렌더되지 않았습니다:\n%s", md)
	}

	// 오늘의 기도 "아래"여야 한다.
	prayerIdx := strings.Index(md, "## 오늘의 기도")
	refsIdx := strings.Index(md, "## 관련 성구")
	if prayerIdx < 0 || refsIdx < 0 || refsIdx < prayerIdx {
		t.Errorf("관련 성구가 오늘의 기도 아래에 있지 않습니다:\n%s", md)
	}

	if !strings.HasSuffix(md, "고린도후서 3:18\n") {
		t.Errorf("문서 끝의 개행이 하나가 아닙니다: %q", md[len(md)-20:])
	}
}

// 관련 성구는 0개가 정상이다(프롬프트 규정: 0-3개).
// 제목만 남지 않도록 섹션째 생략하고, 기도가 마지막 섹션으로 돌아가야 한다.
func TestRenderInfographicMD_OmitsEmptySupportScriptures(t *testing.T) {
	for _, refs := range [][]string{nil, {}, {"  ", ""}} {
		md := RenderInfographicMD(validInfographic(), validSermonSummaryQT(), SermonSummaryMeta{
			Title:             "제목",
			BibleText:         "로마서 8:26-28",
			SupportScriptures: refs,
		})

		if strings.Contains(md, "## 관련 성구") {
			t.Errorf("refs=%q: 관련 성구가 비었는데 섹션이 남아 있습니다", refs)
		}
		if !strings.Contains(md, "## 오늘의 기도") {
			t.Errorf("refs=%q: 앞 섹션까지 사라졌습니다", refs)
		}

		// 기도가 마지막이면 예전과 똑같이 개행 하나로 끝나야 한다.
		if !strings.HasSuffix(md, validSermonSummaryQT().Prayer+"\n") {
			t.Errorf("refs=%q: 기도로 끝나지 않거나 개행이 하나가 아닙니다:\n%q", refs, md)
		}
	}
}

// 표기는 다른 산출물과 같아야 한다. 정규화를 거치지 않으면
// md만 "살전 1:1"이고 나머지는 "데살로니가전서 1:1"이 된다.
func TestRenderInfographicMD_NormalizesSupportScriptures(t *testing.T) {
	md := RenderInfographicMD(validInfographic(), validSermonSummaryQT(), SermonSummaryMeta{
		Title:     "제목",
		BibleText: "로마서 8:26-28",
		// 약어 + 중복
		SupportScriptures: []string{"살전 1:1", "데살로니가전서 1:1", "시 27:8"},
	})

	if !strings.Contains(md, "## 관련 성구\n데살로니가전서 1:1, 시편 27:8\n") {
		t.Errorf("관련 성구가 정규화·중복 제거되지 않았습니다:\n%s", md)
	}
}

// 관련 성구는 InfographicData가 아니므로 검증 대상이 아니다.
// 검증에 넣으면 관련 성구 없는 설교의 md 생성이 통째로 막힌다.
func TestValidateInfographic_IgnoresSupportScriptures(t *testing.T) {
	if reasons := ValidateInfographic(validInfographic()); len(reasons) > 0 {
		t.Errorf("관련 성구와 무관하게 통과해야 합니다: %v", reasons)
	}
}

// ── Step1 ─────────────────────────────────────────────────

// Step2가 md를 다시 렌더하려면 인포그래픽이 temp.json에 실려 있어야 한다.
// 단, version은 1.0 그대로여야 Step2 동작이 달라지지 않는다.
func TestStep1Save_TempJSONCarriesInfographic(t *testing.T) {
	svc := newTestStep1Service(t)

	if _, err := svc.Save(baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true))); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	raw := readTempJSONMap(t, svc.Paths.TempJson)

	meta, _ := raw["metadata"].(map[string]any)
	if meta == nil || meta["infographic"] == nil {
		t.Fatalf("metadata.infographic이 실리지 않았습니다")
	}
	if raw["version"] != "1.0" {
		t.Errorf("version = %v, want 1.0", raw["version"])
	}

	if data := getInfographicFromMap(meta, "infographic"); data == nil || data.Core == "" {
		t.Errorf("metadata.infographic을 다시 읽을 수 없습니다")
	}
}

// 검증에 실패한 인포그래픽은 싣지 않는다.
// 그래야 Step2의 규칙이 "있으면 쓴다"만으로 끝난다.
func TestStep1Save_InvalidInfographicNotStored(t *testing.T) {
	svc := newTestStep1Service(t)

	// infographic이 아예 없는 LLM 출력 = 검증 실패
	if _, err := svc.Save(baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, false))); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	meta, _ := readTempJSONMap(t, svc.Paths.TempJson)["metadata"].(map[string]any)
	if meta != nil && meta["infographic"] != nil {
		t.Errorf("검증에 실패한 인포그래픽이 저장되었습니다")
	}
}

// 비장년은 md를 만들지 않으므로 인포그래픽도 싣지 않는다.
func TestStep1Save_NonAdultDoesNotCarryInfographic(t *testing.T) {
	svc := newTestStep1Service(t)

	if _, err := svc.Save(baseStep1Request("teen", llmJSON("teen", true))); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	meta, _ := readTempJSONMap(t, svc.Paths.TempJson)["metadata"].(map[string]any)
	if meta != nil && meta["infographic"] != nil {
		t.Errorf("비장년인데 인포그래픽이 저장되었습니다")
	}
}

// ── Step2 ─────────────────────────────────────────────────

func TestStep2Save_RewritesSermonSummary(t *testing.T) {
	step1, step2 := newTestStep12Services(t)

	if _, err := step1.Save(baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true))); err != nil {
		t.Fatalf("step1 저장 실패: %v", err)
	}

	data := loadStep2(t, step2)
	data.SupportScriptures = []string{"시편 27:8", "요한복음 14:9"}

	if err := step2.Save(data); err != nil {
		t.Fatalf("step2 저장 실패: %v", err)
	}

	md := readFileString(t, step2.Paths.TempSermonSummary)

	if !strings.Contains(md, "## 관련 성구\n시편 27:8, 요한복음 14:9\n") {
		t.Errorf("Step2에서 고친 관련 성구가 md에 반영되지 않았습니다:\n%s", md)
	}
	if strings.Contains(md, "마태복음 11:28") {
		t.Errorf("Step1 시점의 관련 성구가 md에 남아 있습니다:\n%s", md)
	}
}

// 보존을 빠뜨리면 첫 저장은 정상이고 두 번째 저장부터 조용히 망가진다.
// support_scriptures_full이 정확히 이 방식으로 한 번 유실됐다.
func TestStep2Save_PreservesInfographic(t *testing.T) {
	step1, step2 := newTestStep12Services(t)

	if _, err := step1.Save(baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true))); err != nil {
		t.Fatalf("step1 저장 실패: %v", err)
	}

	// 1회차
	first := loadStep2(t, step2)
	first.SupportScriptures = []string{"시편 27:8"}
	if err := step2.Save(first); err != nil {
		t.Fatalf("1회차 저장 실패: %v", err)
	}

	// 2회차 — 여기서 metadata.infographic이 살아 있어야 md가 또 갱신된다.
	second := loadStep2(t, step2)
	second.SupportScriptures = []string{"요한복음 14:9"}
	if err := step2.Save(second); err != nil {
		t.Fatalf("2회차 저장 실패: %v", err)
	}

	meta, _ := readTempJSONMap(t, step2.Paths.TempJson)["metadata"].(map[string]any)
	if meta == nil || meta["infographic"] == nil {
		t.Fatalf("저장 2회 만에 metadata.infographic이 유실되었습니다")
	}
	if meta["support_scriptures_full"] == nil {
		t.Errorf("저장 2회 만에 support_scriptures_full이 유실되었습니다")
	}

	md := readFileString(t, step2.Paths.TempSermonSummary)
	if !strings.Contains(md, "## 관련 성구\n요한복음 14:9\n") {
		t.Errorf("두 번째 저장이 md에 반영되지 않았습니다:\n%s", md)
	}
}

// 인포그래픽이 없으면(비장년·검증 실패) md 파일을 건드리지 않는다.
func TestStep2Save_NoInfographicLeavesFileAlone(t *testing.T) {
	step1, step2 := newTestStep12Services(t)

	if _, err := step1.Save(baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, false))); err != nil {
		t.Fatalf("step1 저장 실패: %v", err)
	}

	const sentinel = "사람이 손으로 고친 내용"
	if err := os.WriteFile(step2.Paths.TempSermonSummary, []byte(sentinel), 0o644); err != nil {
		t.Fatalf("사전 파일 쓰기 실패: %v", err)
	}

	data := loadStep2(t, step2)
	if err := step2.Save(data); err != nil {
		t.Fatalf("step2 저장 실패: %v", err)
	}

	if got := readFileString(t, step2.Paths.TempSermonSummary); got != sentinel {
		t.Errorf("인포그래픽이 없는데 md를 덮어썼습니다: %q", got)
	}
}

// ── 재작업 ────────────────────────────────────────────────

// 재작업은 writeTempJSON을 거치지 않으므로 별도로 인포그래픽을 실어야 한다.
// 빠뜨리면 재작업 직후 Step2 저장에서 md가 갱신되지 않는다.
func TestRestoreQTSectionDoc_CarriesInfographic(t *testing.T) {
	doc, _, err := restoreQTSectionDoc(llmJSON(AudienceAdult, true), AudienceAdult)
	if err != nil {
		t.Fatalf("복원 실패: %v", err)
	}

	if doc.Metadata["infographic"] == nil {
		t.Errorf("재작업 temp.json에 metadata.infographic이 없습니다")
	}
}

func TestRestoreQTSectionDoc_NonAdultCarriesNoInfographic(t *testing.T) {
	doc, _, err := restoreQTSectionDoc(llmJSON("teen", true), "teen")
	if err != nil {
		t.Fatalf("복원 실패: %v", err)
	}

	if doc.Metadata["infographic"] != nil {
		t.Errorf("비장년인데 metadata.infographic이 실렸습니다")
	}
}

// ── 헬퍼 ──────────────────────────────────────────────────

// newTestStep12Services는 같은 임시 디렉터리를 공유하는 Step1/Step2 서비스를 만든다.
// Step1 저장 → Step2 저장으로 이어지는 실제 흐름을 그대로 태우기 위함이다.
func newTestStep12Services(t *testing.T) (*QTStep1Service, *QTStep2Service) {
	t.Helper()

	dir := t.TempDir()
	paths := &util.AppPaths{
		TempJson:          filepath.Join(dir, "temp.json"),
		TempHtml:          filepath.Join(dir, "temp.html"),
		TempSermonSummary: filepath.Join(dir, "sermon_summary.md"),
	}

	return &QTStep1Service{Paths: paths}, &QTStep2Service{Paths: paths}
}

func loadStep2(t *testing.T, svc *QTStep2Service) *QTStep2Data {
	t.Helper()

	data, err := svc.Load()
	if err != nil {
		t.Fatalf("step2 load 실패: %v", err)
	}
	return data
}

func readTempJSONMap(t *testing.T, path string) map[string]any {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("temp.json 읽기 실패: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("temp.json 파싱 실패: %v", err)
	}
	return raw
}

func readFileString(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s 읽기 실패: %v", filepath.Base(path), err)
	}
	return string(b)
}

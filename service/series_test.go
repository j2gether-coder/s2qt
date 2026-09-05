package service

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const testSeries = "본받고 싶은 교회(1)"

// ── 렌더러 ────────────────────────────────────────────────

func TestRenderInfographicMD_SeriesAboveTitle(t *testing.T) {
	md := RenderInfographicMD(validInfographic(), testSeries, "[QT] 데살로니가서를 시작하며", "살전 1:1-10")

	want := testSeries + "\n\n# 데살로니가서를 시작하며\n"
	if !strings.HasPrefix(md, want) {
		t.Errorf("시리즈가 제목 위 한 줄로 렌더되지 않았습니다:\n%s", firstLines(md, 4))
	}
}

// 시리즈가 없으면 줄도 빈 줄도 남기지 않는다.
// 문서가 빈 줄로 시작하면 제목을 못 잡는 도구가 있다.
func TestRenderInfographicMD_OmitsEmptySeries(t *testing.T) {
	for _, series := range []string{"", "   "} {
		md := RenderInfographicMD(validInfographic(), series, "데살로니가서를 시작하며", "살전 1:1-10")

		if !strings.HasPrefix(md, "# 데살로니가서를 시작하며\n") {
			t.Errorf("series=%q: 문서가 제목으로 시작하지 않습니다:\n%s", series, firstLines(md, 3))
		}
	}
}

// ── temp.html ────────────────────────────────────────────

func TestBuildQTStep2HTML_SeriesBlock(t *testing.T) {
	req := &QTStep2Data{
		Series:    testSeries,
		Title:     "데살로니가서를 시작하며",
		BibleText: "살전 1:1-10",
	}

	html := buildQTStep2HTML(req)

	if !strings.Contains(html, `<div class="qt-series">`+testSeries+`</div>`) {
		t.Fatalf("qt-series 블록이 없습니다")
	}

	seriesIdx := strings.Index(html, "qt-series")
	titleIdx := strings.Index(html, "qt-title")
	if seriesIdx > titleIdx {
		t.Errorf("시리즈가 제목 뒤에 있습니다")
	}
}

// 빈 div를 남기면 margin만큼 여백이 생겨 레이아웃이 달라진다.
func TestBuildQTStep2HTML_OmitsEmptySeries(t *testing.T) {
	req := &QTStep2Data{
		Title:     "데살로니가서를 시작하며",
		BibleText: "살전 1:1-10",
	}

	if html := buildQTStep2HTML(req); strings.Contains(html, "qt-series") {
		t.Errorf("시리즈가 비었는데 블록이 남아 있습니다")
	}
}

// ── 작업내역 라벨 ──────────────────────────────────────────

func TestBuildHistoryTitle(t *testing.T) {
	tests := []struct {
		name   string
		series string
		title  string
		want   string
	}{
		{"시리즈 있음", testSeries, "데살로니가서를 시작하며", testSeries + "|||데살로니가서를 시작하며"},
		{"시리즈 없음", "", "데살로니가서를 시작하며", "데살로니가서를 시작하며"},
		{"시리즈 공백", "   ", "데살로니가서를 시작하며", "데살로니가서를 시작하며"},
		{"양쪽 공백 제거", "  " + testSeries + "  ", "  제목  ", testSeries + "|||제목"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildHistoryTitle(tt.series, tt.title); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// 라벨은 반드시 원래 값으로 되나뉘어야 한다.
// 제목에 하이픈·em dash가 들어가도 안전한지 함께 확인한다.
func TestHistoryTitle_RoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		series string
		title  string
	}{
		{"일반", testSeries, "데살로니가서를 시작하며"},
		{"제목에 em dash", testSeries, "사랑 — 가장 큰 계명"},
		{"제목에 하이픈", testSeries, "은혜 - 값없이 주신 선물"},
		{"시리즈에 하이픈", "본받고 싶은 교회 - 1부", "데살로니가서를 시작하며"},
		{"시리즈 없음", "", "사랑 — 가장 큰 계명"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSeries, gotTitle := splitHistoryTitle(buildHistoryTitle(tt.series, tt.title))

			if gotSeries != tt.series {
				t.Errorf("series = %q, want %q", gotSeries, tt.series)
			}
			if gotTitle != tt.title {
				t.Errorf("title = %q, want %q", gotTitle, tt.title)
			}
		})
	}
}

// 구분자가 없는 라벨(시리즈 도입 이전 이력)은 전체가 제목이다.
func TestSplitHistoryTitle_LegacyLabel(t *testing.T) {
	series, title := splitHistoryTitle("사랑 — 가장 큰 계명")

	if series != "" {
		t.Errorf("series = %q, want 빈 문자열", series)
	}
	if title != "사랑 — 가장 큰 계명" {
		t.Errorf("title = %q", title)
	}
}

// 사용자가 시리즈명에 구분자를 넣어도 되나누기가 깨지지 않아야 한다.
func TestBuildHistoryTitle_SanitizesSeparatorInInput(t *testing.T) {
	label := buildHistoryTitle("시리즈|||이상한", "제목")

	if strings.Count(label, "|||") != 1 {
		t.Fatalf("구분자가 %d개입니다: %q", strings.Count(label, "|||"), label)
	}

	series, title := splitHistoryTitle(label)
	if title != "제목" {
		t.Errorf("title = %q, want 제목", title)
	}
	if strings.Contains(series, "|||") {
		t.Errorf("series에 구분자가 남았습니다: %q", series)
	}
}

// ── Step1 저장 ────────────────────────────────────────────

func TestStep1Save_WritesSeriesToTempJSON(t *testing.T) {
	svc := newTestStep1Service(t)

	req := baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true))
	req.Series = testSeries

	if _, err := svc.Save(req); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	if got := readTempJSONMetadataString(t, svc, "series"); got != testSeries {
		t.Errorf("metadata.series = %q, want %q", got, testSeries)
	}
}

// series는 순수 사용자 입력이므로 LLM 출력을 신뢰하지 않는다.
func TestStep1Save_ScreenSeriesOverridesLLM(t *testing.T) {
	svc := newTestStep1Service(t)

	var doc map[string]any
	if err := json.Unmarshal([]byte(llmJSON(AudienceAdult, true)), &doc); err != nil {
		t.Fatalf("사전 JSON 파싱 실패: %v", err)
	}
	meta, _ := doc["metadata"].(map[string]any)
	meta["series"] = "LLM이 지어낸 시리즈"
	b, _ := json.Marshal(doc)

	req := baseStep1Request(AudienceAdult, string(b))
	req.Series = testSeries

	if _, err := svc.Save(req); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	if got := readTempJSONMetadataString(t, svc, "series"); got != testSeries {
		t.Errorf("LLM 값이 남았습니다: %q", got)
	}
}

// 시리즈가 비면 md에도 흔적이 없어야 한다.
func TestStep1Save_EmptySeriesLeavesNoTrace(t *testing.T) {
	svc := newTestStep1Service(t)

	if _, err := svc.Save(baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true))); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	if got := readTempJSONMetadataString(t, svc, "series"); got != "" {
		t.Errorf("series = %q, want 빈 문자열", got)
	}

	md, err := os.ReadFile(svc.Paths.TempSermonSummary)
	if err != nil {
		t.Fatalf("md 읽기 실패: %v", err)
	}
	if !strings.HasPrefix(string(md), "# ") {
		t.Errorf("md가 제목으로 시작하지 않습니다:\n%s", firstLines(string(md), 3))
	}
}

func TestStep1Save_SeriesRendersIntoInfographic(t *testing.T) {
	svc := newTestStep1Service(t)

	req := baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true))
	req.Series = testSeries

	if _, err := svc.Save(req); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	md, err := os.ReadFile(svc.Paths.TempSermonSummary)
	if err != nil {
		t.Fatalf("md 읽기 실패: %v", err)
	}
	if !strings.HasPrefix(string(md), testSeries+"\n\n# ") {
		t.Errorf("md 첫 줄이 시리즈가 아닙니다:\n%s", firstLines(string(md), 3))
	}
}

// ── 사람 입력 확정 (DB 저장분 포함) ─────────────────────────

// LLM이 다른 값을 내놔도 DB에 저장되는 JSON에는 화면 입력 시리즈가 들어가야 한다.
// 여기가 어긋나면 재작업에서 LLM이 지어낸 시리즈로 복원된다.
func TestStep1Save_HistoryJSONCarriesScreenSeries(t *testing.T) {
	svc := newTestStep1Service(t)

	saved := captureSavedHistoryJSON(t, svc, func(req *QTStep1SaveRequest) {
		req.Series = testSeries
	}, "LLM이 지어낸 시리즈")

	if got := getStringFromMap(saved.Metadata, "series"); got != testSeries {
		t.Errorf("DB 저장 JSON의 series = %q, want %q", got, testSeries)
	}
}

// 장년은 화면에 입력한 제목으로 확정한다.
func TestStep1Save_AdultTitleFromScreen(t *testing.T) {
	svc := newTestStep1Service(t)

	saved := captureSavedHistoryJSON(t, svc, func(req *QTStep1SaveRequest) {
		req.Title = "사람이 입력한 제목"
	}, "")

	if got := getStringFromMap(saved.Metadata, "title"); got != "[QT] 사람이 입력한 제목" {
		t.Errorf("장년 제목 = %q", got)
	}
}

// 비장년은 LLM이 연령대에 맞게 지은 제목을 살린다.
func TestStep1Save_NonAdultKeepsLLMTitle(t *testing.T) {
	svc := newTestStep1Service(t)

	req := baseStep1Request("teen", llmJSON("teen", false))
	req.Title = "사람이 입력한 제목"

	if _, err := svc.Save(req); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	if got := readTempJSONMetadataString(t, svc, "title"); got != "[QT] 테스트 제목" {
		t.Errorf("비장년 제목이 화면 입력값으로 덮어써졌습니다: %q", got)
	}
}

// captureSavedHistoryJSON은 DB에 저장될 JSON을 파싱해 돌려준다.
// History가 nil이면 저장을 건너뛰므로, 동일 경로를 타는 temp.json으로 확인한다.
func captureSavedHistoryJSON(
	t *testing.T,
	svc *QTStep1Service,
	mutate func(*QTStep1SaveRequest),
	llmSeries string,
) QTLLMDoc {
	t.Helper()

	var raw map[string]any
	if err := json.Unmarshal([]byte(llmJSON(AudienceAdult, true)), &raw); err != nil {
		t.Fatalf("사전 JSON 파싱 실패: %v", err)
	}
	if llmSeries != "" {
		meta, _ := raw["metadata"].(map[string]any)
		meta["series"] = llmSeries
	}
	b, _ := json.Marshal(raw)

	req := baseStep1Request(AudienceAdult, string(b))
	mutate(req)

	if _, err := svc.Save(req); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	// temp.json과 DB 저장분은 같은 doc에서 나오므로 metadata 확정 결과가 동일하다.
	tempBytes, err := os.ReadFile(svc.Paths.TempJson)
	if err != nil {
		t.Fatalf("temp.json 읽기 실패: %v", err)
	}

	var doc QTLLMDoc
	if err := json.Unmarshal(tempBytes, &doc); err != nil {
		t.Fatalf("temp.json 파싱 실패: %v", err)
	}
	return doc
}

// ── Step2 왕복 ────────────────────────────────────────────

// Save()가 QTSectionDoc을 새로 조립하므로, metadata에 넣지 않으면 유실된다.
func TestStep2SaveLoad_PreservesSeries(t *testing.T) {
	svc := newTestStep1Service(t)

	req := baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true))
	req.Series = testSeries
	if _, err := svc.Save(req); err != nil {
		t.Fatalf("Step1 저장 실패: %v", err)
	}

	step2 := &QTStep2Service{Paths: svc.Paths}

	loaded, err := step2.Load()
	if err != nil {
		t.Fatalf("Step2 로드 실패: %v", err)
	}
	if loaded.Series != testSeries {
		t.Fatalf("로드된 series = %q, want %q", loaded.Series, testSeries)
	}

	if err := step2.Save(loaded); err != nil {
		t.Fatalf("Step2 저장 실패: %v", err)
	}

	if got := readTempJSONMetadataString(t, svc, "series"); got != testSeries {
		t.Errorf("Step2 저장 후 series가 유실되었습니다: %q", got)
	}
}

// ── 프롬프트 ──────────────────────────────────────────────

func TestBuildQTPromptJSON_SeriesSubstituted(t *testing.T) {
	meta := testQTMeta(AudienceAdult)
	meta.Series = testSeries

	prompt := BuildQTPromptJSON(meta)

	if !strings.Contains(prompt, testSeries) {
		t.Errorf("프롬프트에 시리즈가 치환되지 않았습니다")
	}
	if strings.Contains(prompt, "{{series}}") {
		t.Errorf("{{series}} 치환자가 남아 있습니다")
	}
	if !strings.Contains(prompt, "시리즈명을 title에 반복하지 않는다") {
		t.Errorf("제목 중복 금지 규칙이 없습니다")
	}
}

func TestBuildQTPromptJSON_EmptySeriesLeavesNoPlaceholder(t *testing.T) {
	prompt := BuildQTPromptJSON(testQTMeta("teen"))

	if strings.Contains(prompt, "{{series}}") {
		t.Errorf("{{series}} 치환자가 남아 있습니다")
	}
}

// ── 재작업 복원 ────────────────────────────────────────────

func TestRestoreQTSectionDoc_KeepsSeries(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(llmJSON(AudienceAdult, true)), &doc); err != nil {
		t.Fatalf("사전 JSON 파싱 실패: %v", err)
	}
	meta, _ := doc["metadata"].(map[string]any)
	meta["series"] = testSeries
	b, _ := json.Marshal(doc)

	restored, _, err := restoreQTSectionDoc(string(b), AudienceAdult)
	if err != nil {
		t.Fatalf("복원 실패: %v", err)
	}

	if got := getStringFromMap(restored.Metadata, "series"); got != testSeries {
		t.Errorf("재작업 복원에서 series가 유실되었습니다: %q", got)
	}
}

// 구버전 평면 이력에는 series가 없다. 오류 없이 빈 값으로 복원되어야 한다.
func TestRestoreQTSectionDoc_FlatHasNoSeries(t *testing.T) {
	restored, _, err := restoreQTSectionDoc(flatHistoryJSON(AudienceAdult), AudienceAdult)
	if err != nil {
		t.Fatalf("구버전 이력 복원 실패: %v", err)
	}

	if got := getStringFromMap(restored.Metadata, "series"); got != "" {
		t.Errorf("구버전 이력에 series가 생겼습니다: %q", got)
	}
}

// ── 헬퍼 ─────────────────────────────────────────────────

func readTempJSONMetadataString(t *testing.T, svc *QTStep1Service, key string) string {
	t.Helper()

	b, err := os.ReadFile(svc.Paths.TempJson)
	if err != nil {
		t.Fatalf("temp.json 읽기 실패: %v", err)
	}

	var raw struct {
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("temp.json 파싱 실패: %v", err)
	}

	return getStringFromMap(raw.Metadata, key)
}

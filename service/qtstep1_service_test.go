package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"s2qt/util"
)

// newTestStep1Service는 임시 디렉터리에 파일을 쓰는 서비스를 만든다.
// DB는 붙이지 않으므로 이력 저장은 건너뛴다.
func newTestStep1Service(t *testing.T) *QTStep1Service {
	t.Helper()

	dir := t.TempDir()
	return &QTStep1Service{
		Paths: &util.AppPaths{
			TempJson: filepath.Join(dir, "temp.json"),
			// Step2 Save()가 temp.html도 함께 쓰므로 경로가 필요하다.
			TempHtml:        filepath.Join(dir, "temp.html"),
			TempSermonSummary: filepath.Join(dir, "sermon_summary.md"),
		},
	}
}

func llmJSON(audience string, withInfographic bool) string {
	doc := map[string]any{
		"version":     "1.0",
		"doc_type":    "qt",
		"audience":    audience,
		"template_id": "qt_classic",
		"metadata": map[string]any{
			"title":              "[QT] 테스트 제목",
			"bible_text":         "로마서 8:26-28",
			"support_scriptures": []string{"마태복음 11:28"},
			"support_scriptures_full": []map[string]string{
				{"reference": "마태복음 11:28", "text": "수고하고 무거운 짐진 자들아"},
			},
		},
		"sections": []map[string]any{
			{"type": "summary", "title": "말씀의 길잡이", "blocks": []map[string]any{
				{"type": "paragraph", "text": "요약 본문입니다."},
			}},
			{"type": "prayer", "title": "오늘의 기도", "blocks": []map[string]any{
				{"type": "paragraph", "text": "기도문입니다."},
			}},
		},
	}

	if withInfographic {
		doc["version"] = "1.1"
		doc["infographic"] = validInfographic()
	}

	b, _ := json.Marshal(doc)
	return string(b)
}

func baseStep1Request(audience string, jsonText string) *QTStep1SaveRequest {
	return &QTStep1SaveRequest{
		Audience:  audience,
		Title:     "화면에서 입력한 제목",
		BibleText: "로마서 8:26-28",
		JSONText:  jsonText,
	}
}

// temp.json에는 infographic이 실리지 않고 version은 1.0이어야 한다.
func TestStep1Save_SplitsInfographicOutOfTempJSON(t *testing.T) {
	svc := newTestStep1Service(t)

	if _, err := svc.Save(baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true))); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	b, err := os.ReadFile(svc.Paths.TempJson)
	if err != nil {
		t.Fatalf("temp.json 읽기 실패: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("temp.json 파싱 실패: %v", err)
	}

	if _, exists := raw["infographic"]; exists {
		t.Errorf("temp.json에 infographic 키가 남아 있습니다")
	}
	if raw["version"] != "1.0" {
		t.Errorf("temp.json version = %v, want 1.0", raw["version"])
	}

	// blog 전용 내부 필드는 보존되어야 한다.
	meta, _ := raw["metadata"].(map[string]any)
	if meta == nil || meta["support_scriptures_full"] == nil {
		t.Errorf("support_scriptures_full이 유실되었습니다")
	}
}

func TestStep1Save_WritesInfographicForAdult(t *testing.T) {
	svc := newTestStep1Service(t)

	result, err := svc.Save(baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true)))
	if err != nil {
		t.Fatalf("저장 실패: %v", err)
	}
	if result.InfographicPath == "" {
		t.Fatalf("infographic 경로가 비어 있습니다: warnings=%v", result.Warnings)
	}

	md, err := os.ReadFile(svc.Paths.TempSermonSummary)
	if err != nil {
		t.Fatalf("sermon_summary.md 읽기 실패: %v", err)
	}

	// 제목은 LLM 출력이 아니라 화면 기본정보에서 와야 한다.
	if !strings.HasPrefix(string(md), "# 화면에서 입력한 제목\n") {
		t.Errorf("제목이 화면 기본정보에서 오지 않았습니다:\n%s", firstLines(string(md), 2))
	}
	if strings.Contains(string(md), "테스트 제목") {
		t.Errorf("LLM metadata의 제목이 사용되었습니다")
	}
}

// 비장년은 인포그래픽을 만들지 않고 파일을 0바이트로 둔다.
func TestStep1Save_NoInfographicForNonAdult(t *testing.T) {
	svc := newTestStep1Service(t)

	// 이전 설교의 내용이 남아 있는 상황을 재현한다.
	if err := os.WriteFile(svc.Paths.TempSermonSummary, []byte("이전 설교 인포그래픽"), 0o644); err != nil {
		t.Fatalf("사전 파일 생성 실패: %v", err)
	}

	result, err := svc.Save(baseStep1Request("teen", llmJSON("teen", false)))
	if err != nil {
		t.Fatalf("저장 실패: %v", err)
	}
	if result.InfographicPath != "" {
		t.Errorf("비장년인데 infographic이 생성되었습니다")
	}

	info, err := os.Stat(svc.Paths.TempSermonSummary)
	if err != nil {
		t.Fatalf("sermon_summary.md stat 실패: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("이전 내용이 남아 있습니다: size=%d", info.Size())
	}
}

// 인포그래픽이 규칙에 어긋나도 QT 저장은 성공해야 한다(결정 5).
func TestStep1Save_InvalidInfographicDoesNotBlockSave(t *testing.T) {
	svc := newTestStep1Service(t)

	var doc map[string]any
	_ = json.Unmarshal([]byte(llmJSON(AudienceAdult, true)), &doc)
	doc["infographic"] = map[string]any{
		"guide":  "안내",
		"follow": []string{"하나뿐"},
		"core":   "",
		"apply":  []string{"하나뿐"},
		"prayer": "기도",
	}
	b, _ := json.Marshal(doc)

	result, err := svc.Save(baseStep1Request(AudienceAdult, string(b)))
	if err != nil {
		t.Fatalf("인포그래픽 문제로 저장이 실패했습니다: %v", err)
	}
	if result.TempJSONPath == "" {
		t.Errorf("temp.json이 저장되지 않았습니다")
	}
	if result.InfographicPath != "" {
		t.Errorf("잘못된 데이터로 infographic이 생성되었습니다")
	}
	if len(result.Warnings) == 0 {
		t.Errorf("경고 사유가 기록되지 않았습니다")
	}
}

// 필수값이 없으면 파일을 하나도 건드리지 않아야 한다(부분 저장 방지).
func TestStep1Save_RejectsMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*QTStep1SaveRequest)
	}{
		{"제목 없음", func(r *QTStep1SaveRequest) { r.Title = "  " }},
		{"성구 없음", func(r *QTStep1SaveRequest) { r.BibleText = "" }},
		{"연령대 없음", func(r *QTStep1SaveRequest) { r.Audience = "" }},
		{"JSON 없음", func(r *QTStep1SaveRequest) { r.JSONText = "" }},
		{"JSON 깨짐", func(r *QTStep1SaveRequest) { r.JSONText = "{ not json" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestStep1Service(t)

			req := baseStep1Request(AudienceAdult, llmJSON(AudienceAdult, true))
			tt.mutate(req)

			if _, err := svc.Save(req); err == nil {
				t.Fatalf("에러가 반환되지 않았습니다")
			}
			if _, err := os.Stat(svc.Paths.TempJson); !os.IsNotExist(err) {
				t.Errorf("검증 실패인데 temp.json이 생성되었습니다")
			}
		})
	}
}

// 코드펜스가 붙은 응답도 저장되어야 한다.
func TestStep1Save_AcceptsFencedJSON(t *testing.T) {
	svc := newTestStep1Service(t)

	req := baseStep1Request(AudienceAdult, "```json\n"+llmJSON(AudienceAdult, true)+"\n```")
	if _, err := svc.Save(req); err != nil {
		t.Fatalf("코드펜스가 붙은 JSON 저장 실패: %v", err)
	}
}

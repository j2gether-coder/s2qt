package service

import (
	"strings"
	"testing"
)

func validInfographic() *InfographicData {
	return &InfographicData{
		Guide:  "우리는 연약함을 절감합니다. 성령께서 도우십니다.",
		Follow: []string{"성령께서 친히 간구하신다.", "성령께서 우리 마음을 아신다.", "모든 것이 합력하여 선을 이룬다."},
		Extra:  []string{"한 목회자는 수술 중에 성령의 탄식을 경험했다.", "한 선교사는 아이들의 이름만 듣고도 필요를 알았다."},
		Core:   "내가 무너져도 성령께서 나를 위해 기도하신다.",
		Apply:  []string{"기도할 바를 모를 때 주의 이름을 부른다.", "일어나는 일을 합력하여 선을 이루시는 과정으로 믿는다."},
		Prayer: "아버지 하나님, 성령을 보내 주셔서 감사합니다. 오늘도 성령을 따라 순종하게 하옵소서.",
	}
}

func TestRenderInfographicMD_Structure(t *testing.T) {
	md := RenderInfographicMD(validInfographic(), "", "[QT] 내가 무너져도 성령은 일하신다", "로마서 8:26-28")

	// 제목에서 [QT] 접두어가 제거되어야 한다.
	if !strings.HasPrefix(md, "# 내가 무너져도 성령은 일하신다\n") {
		t.Errorf("제목 렌더가 올바르지 않습니다:\n%s", firstLines(md, 3))
	}

	// 8개 섹션이 지정된 순서대로 나와야 한다.
	wantOrder := []string{
		"# 내가 무너져도",
		"## 성경본문",
		"## 말씀의 길잡이",
		"## 말씀을 따라",
		"## 더하는 말씀",
		"## 말씀의 핵심",
		"## 오늘의 적용",
		"## 오늘의 기도",
	}

	prev := -1
	for _, heading := range wantOrder {
		idx := strings.Index(md, heading)
		if idx < 0 {
			t.Fatalf("%q 섹션이 없습니다", heading)
		}
		if idx <= prev {
			t.Errorf("%q 섹션의 순서가 잘못되었습니다", heading)
		}
		prev = idx
	}
}

func TestRenderInfographicMD_ListsUseBullets(t *testing.T) {
	md := RenderInfographicMD(validInfographic(), "", "제목", "로마서 8:26-28")

	if !strings.Contains(md, "- 성령께서 친히 간구하신다.") {
		t.Errorf("follow가 불릿으로 렌더되지 않았습니다")
	}
	if !strings.Contains(md, "- 기도할 바를 모를 때 주의 이름을 부른다.") {
		t.Errorf("apply가 불릿으로 렌더되지 않았습니다")
	}

	// extra는 불릿이 아니라 문단이며, 빈 줄로 구분한다.
	if strings.Contains(md, "- 한 목회자는") {
		t.Errorf("extra가 불릿으로 렌더되었습니다 (문단이어야 함)")
	}
	if !strings.Contains(md, "성령의 탄식을 경험했다.\n\n한 선교사는") {
		t.Errorf("extra 문단이 빈 줄로 구분되지 않았습니다")
	}
}

// 예화가 없으면 "더하는 말씀" 제목만 남지 않도록 섹션 자체를 생략한다.
func TestRenderInfographicMD_OmitsEmptyExtra(t *testing.T) {
	data := validInfographic()
	data.Extra = []string{}

	md := RenderInfographicMD(data, "", "제목", "로마서 8:26-28")

	if strings.Contains(md, "## 더하는 말씀") {
		t.Errorf("extra가 비었는데 섹션이 남아 있습니다")
	}
	if !strings.Contains(md, "## 말씀의 핵심") {
		t.Errorf("다음 섹션까지 사라졌습니다")
	}
}

func TestRenderInfographicMD_NilReturnsEmpty(t *testing.T) {
	if got := RenderInfographicMD(nil, "", "제목", "본문"); got != "" {
		t.Errorf("nil 입력에 빈 문자열이 아닌 값이 반환되었습니다: %q", got)
	}
}

func TestValidateInfographic_AcceptsValid(t *testing.T) {
	if reasons := ValidateInfographic(validInfographic()); len(reasons) != 0 {
		t.Errorf("정상 데이터가 거부되었습니다: %v", reasons)
	}
}

func TestValidateInfographic_ReportsEachViolation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*InfographicData)
		wantSub string
	}{
		{"follow 부족", func(d *InfographicData) { d.Follow = []string{"하나만"} }, "follow가 1개입니다"},
		{"follow 초과", func(d *InfographicData) {
			d.Follow = []string{"문장1", "문장2", "문장3", "문장4", "문장5", "문장6"}
		}, "follow가 6개입니다"},
		{"apply 부족", func(d *InfographicData) { d.Apply = []string{"하나만"} }, "apply가 1개입니다"},
		{"extra 초과", func(d *InfographicData) { d.Extra = []string{"1", "2", "3", "4"} }, "extra가 4개입니다"},
		{"core 공백", func(d *InfographicData) { d.Core = "   " }, "core가 비어 있습니다"},
		{"guide 공백", func(d *InfographicData) { d.Guide = "" }, "guide가 비어 있습니다"},
		{"prayer 공백", func(d *InfographicData) { d.Prayer = "" }, "prayer가 비어 있습니다"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := validInfographic()
			tt.mutate(data)

			reasons := ValidateInfographic(data)
			if !containsSubstring(reasons, tt.wantSub) {
				t.Errorf("사유 %q를 찾을 수 없습니다: %v", tt.wantSub, reasons)
			}
		})
	}
}

func TestValidateInfographic_NilIsRejected(t *testing.T) {
	if reasons := ValidateInfographic(nil); len(reasons) == 0 {
		t.Errorf("nil이 통과되었습니다")
	}
}

// 빈 문자열만 든 배열은 개수로 세지 않는다(LLM이 자리만 채운 경우).
func TestValidateInfographic_IgnoresBlankItems(t *testing.T) {
	data := validInfographic()
	data.Follow = []string{"문장1", "  ", "", "문장2"}

	if !containsSubstring(ValidateInfographic(data), "follow가 2개입니다") {
		t.Errorf("공백 항목이 개수에 포함되었습니다")
	}
}

// cleanStringSlice가 중복을 제거하므로, 같은 문장을 반복해 개수를 채울 수 없다.
func TestValidateInfographic_IgnoresDuplicateItems(t *testing.T) {
	data := validInfographic()
	data.Follow = []string{"같은 문장", "같은 문장", "같은 문장"}

	if !containsSubstring(ValidateInfographic(data), "follow가 1개입니다") {
		t.Errorf("중복 항목이 개수에 포함되었습니다")
	}
}

func containsSubstring(items []string, sub string) bool {
	for _, item := range items {
		if strings.Contains(item, sub) {
			return true
		}
	}
	return false
}

func firstLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

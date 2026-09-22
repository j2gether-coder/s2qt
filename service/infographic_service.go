package service

import (
	"fmt"
	"strings"
)

// sermon_summary.md는 기존 HTML/PDF/PNG/blog와 완전히 분리된 산출물이다.
// Step1에서 LLM 원본 JSON의 infographic 객체를 마크다운으로 렌더해 만든다.
// (이전에는 프롬프트+전사문을 합친 "입력 문서"였으나, 통합 프롬프트 도입으로
//  완성된 산출물로 바뀌었다. var/conf/prompt_sermon_summary.md는 규칙 참고용으로만 남는다.)

// 인포그래픽 필드 개수 규칙. 프롬프트의 QUALITY CHECKS와 1:1로 대응한다.
const (
	infographicFollowMin = 3
	infographicFollowMax = 5
	infographicApplyMin  = 2
	infographicApplyMax  = 3
	infographicExtraMax  = 3
)

// ValidateInfographic는 인포그래픽 데이터의 위반 사유를 모두 모아 반환한다.
// 빈 슬라이스면 통과다. 반환값은 화면에 표시하지 않고 로그로만 사용한다(결정 5).
func ValidateInfographic(data *InfographicData) []string {
	if data == nil {
		return []string{"infographic 객체가 없습니다"}
	}

	var reasons []string

	if strings.TrimSpace(data.Guide) == "" {
		reasons = append(reasons, "guide가 비어 있습니다")
	}
	if strings.TrimSpace(data.Core) == "" {
		reasons = append(reasons, "core가 비어 있습니다")
	}
	if strings.TrimSpace(data.Prayer) == "" {
		reasons = append(reasons, "prayer가 비어 있습니다")
	}

	follow := cleanStringSlice(data.Follow)
	if len(follow) < infographicFollowMin || len(follow) > infographicFollowMax {
		reasons = append(reasons, fmt.Sprintf(
			"follow가 %d개입니다 (%d~%d개 필요)",
			len(follow), infographicFollowMin, infographicFollowMax))
	}

	apply := cleanStringSlice(data.Apply)
	if len(apply) < infographicApplyMin || len(apply) > infographicApplyMax {
		reasons = append(reasons, fmt.Sprintf(
			"apply가 %d개입니다 (%d~%d개 필요)",
			len(apply), infographicApplyMin, infographicApplyMax))
	}

	// extra는 0개도 정상이다(예화가 없으면 빈 배열). 상한만 확인한다.
	if extra := cleanStringSlice(data.Extra); len(extra) > infographicExtraMax {
		reasons = append(reasons, fmt.Sprintf(
			"extra가 %d개입니다 (최대 %d개)", len(extra), infographicExtraMax))
	}

	return reasons
}

// RenderInfographicMD는 인포그래픽 데이터를 마크다운 문서로 렌더한다.
// meta의 series/title/bibleText는 LLM 출력이 아니라 화면 기본정보에서 받는다.
// LLM이 메타정보를 날조하는 사례가 있어, 사용자가 입력한 값을 신뢰한다.
//
// series가 있으면 제목 위 한 줄로 넣는다. 없으면 그 줄과 뒤따르는 빈 줄을 모두 생략한다.
// (빈 줄이 남으면 문서가 빈 줄로 시작해 제목을 못 잡는 도구가 있다)
func RenderInfographicMD(data *InfographicData, meta SermonSummaryMeta) string {
	if data == nil {
		return ""
	}

	var b strings.Builder

	if s := strings.TrimSpace(meta.Series); s != "" {
		b.WriteString(s + "\n\n")
	}

	b.WriteString("# " + stripQTTitlePrefix(meta.Title) + "\n\n")

	b.WriteString("## 성경본문\n")
	b.WriteString(normalizeBibleReference(meta.BibleText) + "\n\n")

	b.WriteString("## 말씀의 길잡이\n")
	b.WriteString(strings.TrimSpace(data.Guide) + "\n\n")

	writeInfographicList(&b, "말씀을 따라", data.Follow)

	// 예화가 없으면 섹션 자체를 넣지 않는다(빈 제목만 남는 것을 막는다).
	writeInfographicParagraphs(&b, "더하는 말씀", data.Extra)

	b.WriteString("## 말씀의 핵심\n")
	b.WriteString(strings.TrimSpace(data.Core) + "\n\n")

	writeInfographicList(&b, "오늘의 적용", data.Apply)

	b.WriteString("## 오늘의 기도\n")
	b.WriteString(strings.TrimSpace(data.Prayer) + "\n\n")

	// 관련 성구는 0개가 정상이다(프롬프트 규정: 0-3개). 비면 섹션째 생략한다.
	// 표기는 다른 산출물과 같은 normalizeBibleRefSlice를 거친다 — 여기서 한 번만
	// 호출해 두 호출부가 각자 기억하지 않게 한다.
	writeInfographicInline(&b, "관련 성구", normalizeBibleRefSlice(meta.SupportScriptures))

	// 섹션은 모두 "\n\n"로 끝난다. 어느 것이 마지막이 되든 결과가 같도록
	// 반환 직전에 한 번만 정규화한다(분기를 만들지 않는다).
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// writeInfographicList는 불릿 목록 섹션을 쓴다(말씀을 따라 / 오늘의 적용).
func writeInfographicList(b *strings.Builder, heading string, items []string) {
	cleaned := cleanStringSlice(items)
	if len(cleaned) == 0 {
		return
	}

	b.WriteString("## " + heading + "\n")
	for _, item := range cleaned {
		b.WriteString("- " + item + "\n")
	}
	b.WriteString("\n")
}

// writeInfographicInline은 쉼표로 이어 붙인 한 줄 섹션을 쓴다(관련 성구).
// 다른 두 헬퍼와 같은 계약이다 — 비면 아무것도 쓰지 않는다.
func writeInfographicInline(b *strings.Builder, heading string, items []string) {
	if len(items) == 0 {
		return
	}

	b.WriteString("## " + heading + "\n")
	b.WriteString(strings.Join(items, ", ") + "\n\n")
}

// writeInfographicParagraphs는 문단 목록 섹션을 쓴다(더하는 말씀).
func writeInfographicParagraphs(b *strings.Builder, heading string, items []string) {
	cleaned := cleanStringSlice(items)
	if len(cleaned) == 0 {
		return
	}

	b.WriteString("## " + heading + "\n")
	b.WriteString(strings.Join(cleaned, "\n\n"))
	b.WriteString("\n\n")
}

// stripQTTitlePrefix는 인포그래픽 제목에 불필요한 "[QT]" 접두어를 제거한다.
func stripQTTitlePrefix(title string) string {
	t := strings.TrimSpace(title)
	if strings.HasPrefix(t, "[QT]") {
		t = strings.TrimSpace(strings.TrimPrefix(t, "[QT]"))
	}
	return t
}

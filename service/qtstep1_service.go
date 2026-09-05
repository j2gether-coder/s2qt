package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"s2qt/util"
)

// Step1 결과저장은 LLM 원본 JSON 하나를 받아 세 곳으로 나눠 보낸다.
//
//	temp.json         : QT 부분만 (version 1.0) — Step2 편집 대상
//	sermon_summary.md : infographic 부분을 렌더한 설교요약문 — 외부 에디터로 편집
//	작업내역 DB        : 사람 입력값을 확정한 JSON 전문 — 재작업의 유일한 원천
//
// DB에는 LLM이 준 문자열을 그대로 넣지 않는다. 사람이 입력한 시리즈·제목을
// 확정한 뒤 직렬화해야 재작업에서 같은 값이 되살아난다.
//
// 수동 붙여넣기(SaveManualLLMResult)와 API 자동 생성(RunLLMPrepare)이
// 모두 이 서비스를 거치므로, 경로에 따라 동작이 갈리지 않는다.

type QTStep1Service struct {
	Paths   *util.AppPaths
	History *HistoryService
}

// NewQTStep1Service는 이력 저장 없이 파일만 쓰는 서비스를 만든다.
// 이력까지 남기려면 NewQTStep1ServiceWithHistory를 사용한다.
func NewQTStep1Service() (*QTStep1Service, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}
	return &QTStep1Service{Paths: paths}, nil
}

func NewQTStep1ServiceWithHistory(history *HistoryService) (*QTStep1Service, error) {
	svc, err := NewQTStep1Service()
	if err != nil {
		return nil, err
	}
	svc.History = history
	return svc, nil
}

// Save는 LLM 결과를 검증하고 분리 저장한다.
//
// 순서를 지키는 것이 중요하다. 검증과 파싱을 모두 끝낸 뒤에 파일을 건드리므로,
// 잘못된 입력으로 기존 temp.json이 깨지지 않는다.
//
//  1. 필수값 검증
//  2. JSON 정리·파싱
//  3. temp.json 쓰기
//  4. sermon_summary.md 쓰기 (조건 불충족이면 0바이트로 비움)
//  5. 작업내역 DB 저장
func (s *QTStep1Service) Save(req *QTStep1SaveRequest) (*QTStep1SaveResult, error) {
	if s == nil || s.Paths == nil {
		return nil, fmt.Errorf("step1 service가 준비되지 않았습니다")
	}
	if req == nil {
		return nil, fmt.Errorf("step1 저장 요청이 비어 있습니다")
	}

	// 1) 필수값 검증 — 화면에서 제목과 본문 성구만 필수이며, DB의 NOT NULL 컬럼과 일치한다.
	audience := strings.TrimSpace(req.Audience)
	title := strings.TrimSpace(req.Title)
	bibleText := strings.TrimSpace(req.BibleText)

	if audience == "" {
		return nil, fmt.Errorf("대상 연령층이 비어 있습니다")
	}
	if title == "" {
		return nil, fmt.Errorf("제목이 비어 있습니다")
	}
	if bibleText == "" {
		return nil, fmt.Errorf("본문 성구가 비어 있습니다")
	}

	// 2) JSON 정리·파싱 — 여기까지 실패하면 파일을 하나도 건드리지 않는다.
	jsonText := strings.TrimSpace(req.JSONText)
	if jsonText == "" {
		return nil, fmt.Errorf("저장할 JSON 결과가 비어 있습니다")
	}

	jsonText = CleanLLMJSONOutput(jsonText)

	var doc QTLLMDoc
	if err := json.Unmarshal([]byte(jsonText), &doc); err != nil {
		return nil, fmt.Errorf("유효한 JSON 형식이 아닙니다")
	}

	result := &QTStep1SaveResult{}

	// 사람이 입력한 값을 doc에 확정해 넣는다.
	// 이 doc이 temp.json·md·작업내역 DB의 공통 원본이 되므로,
	// 여기서 확정해야 재작업에서도 같은 값이 되살아난다.
	series := sanitizeSeriesTitlePart(req.Series)
	if doc.Metadata == nil {
		doc.Metadata = map[string]any{}
	}
	doc.Metadata["series"] = series

	// 장년은 화면에 입력한 제목을 그대로 쓴다.
	// 비장년은 LLM이 연령대에 맞게 새로 지은 제목을 살린다
	// (Step2의 resolveQtTitleByAudience가 쓰던 규칙을 여기서 확정한다).
	if audience == AudienceAdult {
		doc.Metadata["title"] = ensureQTTitlePrefix(title)
	}

	// 3) temp.json — QTSectionDoc만 마샬하므로 infographic이 자동으로 빠진다.
	if err := s.writeTempJSON(&doc, audience); err != nil {
		return nil, err
	}
	result.TempJSONPath = s.Paths.TempJson

	// 4) sermon_summary.md
	infoPath, warnings := s.writeSermonSummary(&doc, audience, series, title, bibleText)
	result.InfographicPath = infoPath
	result.Warnings = warnings

	// 5) 작업내역 DB
	//    실패해도 파일 산출물은 이미 정상이므로 저장 자체를 되돌리지 않는다.
	historyID, err := s.saveHistory(req, &doc)
	if err != nil {
		LogError("step1: 작업내역 저장 실패: " + err.Error())
		return result, err
	}
	result.HistoryID = historyID

	return result, nil
}

// writeTempJSON은 QT 부분만 기존 v1.0 스키마로 저장한다.
// version을 1.0으로 고정해 Step2가 지금까지와 똑같이 읽고 쓰도록 한다.
//
// metadata는 호출자가 이미 확정한 상태로 넘어온다(series 덮어쓰기 포함).
func (s *QTStep1Service) writeTempJSON(doc *QTLLMDoc, audience string) error {
	qt := doc.QTSectionDoc

	qt.Version = "1.0"
	if strings.TrimSpace(qt.DocType) == "" {
		qt.DocType = "qt"
	}
	if strings.TrimSpace(qt.Audience) == "" {
		qt.Audience = audience
	}
	if strings.TrimSpace(qt.Template) == "" {
		qt.Template = "qt_classic"
	}

	b, err := json.MarshalIndent(qt, "", "  ")
	if err != nil {
		return fmt.Errorf("temp.json 직렬화 실패: %w", err)
	}

	if err := os.WriteFile(s.Paths.TempJson, b, 0o644); err != nil {
		return fmt.Errorf("temp.json 저장 실패: %w", err)
	}

	return nil
}

// writeSermonSummary는 조건을 만족할 때만 md를 쓰고, 아니면 파일을 0바이트로 비운다.
// 비우기는 항상 먼저 수행해 이전 설교의 내용이 남지 않게 한다.
//
// 생성 조건은 두 가지뿐이다(장년 / 데이터가 규칙에 맞음).
// Step3는 더 이상 md를 만들지 않으므로 "편집본 보호" 게이트가 필요 없다.
//
// 반환값의 warnings는 화면에 표시하지 않고 로그로만 사용한다(결정 5).
func (s *QTStep1Service) writeSermonSummary(doc *QTLLMDoc, audience, series, title, bibleText string) (string, []string) {
	path := s.Paths.TempSermonSummary

	truncateFile(path)

	if audience != AudienceAdult {
		return "", nil
	}

	if reasons := ValidateInfographic(doc.Infographic); len(reasons) > 0 {
		LogWarn(fmt.Sprintf(
			"step1: infographic 생성 건너뜀 — %s (md는 빈 상태로 둡니다)",
			strings.Join(reasons, ", ")))
		return "", reasons
	}

	// 시리즈·제목·성경본문은 LLM 출력이 아니라 화면 기본정보를 쓴다.
	// LLM이 메타정보를 날조하는 사례가 있어 사용자 입력을 신뢰한다.
	content := RenderInfographicMD(doc.Infographic, series, title, bibleText)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		msg := "sermon_summary.md 저장 실패: " + err.Error()
		LogError("step1: " + msg)
		return "", []string{msg}
	}

	LogInfo("step1: sermon_summary.md 생성 완료 path=" + path)
	return path, nil
}

// saveHistory는 LLM 결과 전문을 작업내역에 남긴다.
// 재작업은 이 기록에서 temp.json과 sermon_summary.md를 다시 만든다.
//
// LLM이 준 문자열을 그대로 넣지 않고 doc을 다시 직렬화한다.
// series는 화면 입력값으로 확정된 상태여야 재작업에서 되살아나기 때문이다.
// sections·infographic·metadata의 나머지 필드는 파싱→직렬화를 거쳐도 그대로 보존된다.
func (s *QTStep1Service) saveHistory(req *QTStep1SaveRequest, doc *QTLLMDoc) (int64, error) {
	if s.History == nil {
		// 이력 저장 없이 파일만 쓰는 경로(테스트 등)에서는 건너뛴다.
		return 0, nil
	}

	jsonText, err := json.Marshal(doc)
	if err != nil {
		return 0, fmt.Errorf("작업내역 JSON 직렬화 실패: %w", err)
	}

	// 목록 라벨의 제목은 doc에 확정된 값을 쓴다.
	// 장년은 화면 입력 제목, 비장년은 LLM이 지은 제목이 들어 있어
	// 목록에 보이는 제목과 실제 산출물 제목이 어긋나지 않는다.
	labelTitle := strings.TrimSpace(getStringFromMap(doc.Metadata, "title"))
	if labelTitle == "" {
		labelTitle = strings.TrimSpace(req.Title)
	}

	historyID, err := s.History.SaveHistory(SaveHistoryRequest{
		Title:        buildHistoryTitle(req.Series, labelTitle),
		BibleText:    strings.TrimSpace(req.BibleText),
		Hymn:         strings.TrimSpace(req.Hymn),
		Preacher:     strings.TrimSpace(req.Preacher),
		ChurchName:   strings.TrimSpace(req.ChurchName),
		SermonDate:   strings.TrimSpace(req.SermonDate),
		Audience:     strings.TrimSpace(req.Audience),
		QTResultJSON: string(jsonText),
	})
	if err != nil {
		return 0, fmt.Errorf("작업내역 저장 실패: %w", err)
	}

	LogInfo(fmt.Sprintf("step1: 작업내역 저장 완료 history_id=%d audience=%s", historyID, req.Audience))
	return historyID, nil
}

// seriesTitleSeparator는 history_master.title에서 시리즈와 제목을 잇는 구분자다.
//
// 사람이 쓰는 글자를 쓰면 안 된다. "—"나 "-"는 제목 안에 실제로 등장할 수 있어
// ("사랑 — 가장 큰 계명") 나중에 되나눌 때 조용히 잘못 잘린다.
// 설교 제목에 나올 일이 없는 "|||"를 쓴다.
const seriesTitleSeparator = "|||"

// buildHistoryTitle은 작업내역 목록에 저장할 라벨을 만든다.
//
// history_master에 series 컬럼을 두지 않고 title에 합쳐 넣는다.
// 이렇게 하면 기존 키워드 검색(title LIKE)과 제목순 정렬이 그대로 동작하며,
// DB 스키마를 바꾸지 않아도 된다.
func buildHistoryTitle(series, title string) string {
	t := sanitizeSeriesTitlePart(title)
	s := sanitizeSeriesTitlePart(series)
	if s == "" {
		return t
	}
	return s + seriesTitleSeparator + t
}

// splitHistoryTitle은 라벨을 시리즈와 제목으로 되나눈다.
// 구분자가 없으면 전체가 제목이다(시리즈 없는 이력, 시리즈 도입 이전 이력).
func splitHistoryTitle(label string) (series string, title string) {
	l := strings.TrimSpace(label)

	idx := strings.Index(l, seriesTitleSeparator)
	if idx < 0 {
		return "", l
	}

	series = strings.TrimSpace(l[:idx])
	title = strings.TrimSpace(l[idx+len(seriesTitleSeparator):])
	return series, title
}

// sanitizeSeriesTitlePart는 구분자가 값 안에 섞여 들어가는 것을 막는다.
// 사용자가 시리즈명에 "|||"를 넣을 일은 없지만, 넣으면 되나누기가 깨진다.
func sanitizeSeriesTitlePart(v string) string {
	return strings.TrimSpace(strings.ReplaceAll(v, seriesTitleSeparator, " "))
}

// truncateFile은 파일을 지우지 않고 내용만 비운다.
// 경로와 열려 있는 편집기 핸들이 유지되고, 파일 크기 0이
// "이 작업에는 설교요약문이 없다"는 표시가 된다.
func truncateFile(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	if err := os.Truncate(path, 0); err != nil && !os.IsNotExist(err) {
		LogError("step1: 파일 비우기 실패: " + err.Error())
	}
}

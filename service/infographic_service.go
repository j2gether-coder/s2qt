package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"s2qt/util"
)

// infographic.md는 기존 HTML/PDF/PNG/blog와 완전히 분리된 산출물이다.
// var/conf/prompt_infographic.md(프롬프트)와 var/temp/temp.txt(전사문)를 하나로 합쳐
// var/temp/infographic.md를 만든다. LLM 호출은 하지 않고, 외부 인포그래픽 생성에
// 그대로 붙여 넣을 수 있는 입력 문서를 만드는 것이 목적이다.

type infographicMetadata struct {
	Title     string `json:"title"`
	BibleText string `json:"bible_text"`
}

type infographicDoc struct {
	Metadata infographicMetadata `json:"metadata"`
}

type InfographicService struct {
	Paths *util.AppPaths
}

func NewInfographicService() (*InfographicService, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}
	return &InfographicService{Paths: paths}, nil
}

// BuildInfographicMD는 프롬프트와 전사문을 합쳐 infographic.md를 생성하고 경로를 반환한다.
func (s *InfographicService) BuildInfographicMD() (string, error) {
	rawText, err := s.loadTranscript()
	if err != nil {
		return "", err
	}

	title, bibleText := s.loadMetadata()

	content := BuildInfographicPrompt(title, bibleText, rawText)

	if err := os.WriteFile(s.Paths.TempInfographic, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("infographic.md 저장 실패: %w", err)
	}

	return s.Paths.TempInfographic, nil
}

// loadTranscript는 Step1 산출물인 temp.txt(전사문)를 읽는다.
func (s *InfographicService) loadTranscript() (string, error) {
	b, err := os.ReadFile(s.Paths.TempTxt)
	if err != nil {
		return "", fmt.Errorf("temp.txt 읽기 실패: %w", err)
	}

	text := strings.TrimSpace(string(b))
	if text == "" {
		return "", fmt.Errorf("temp.txt가 비어 있습니다. Step1을 먼저 실행해 주세요")
	}

	return text, nil
}

// loadMetadata는 temp.json에서 제목/본문 성구를 읽는다.
// temp.json이 없거나 파싱에 실패해도 전사문만으로 생성할 수 있도록 빈 값을 반환한다.
func (s *InfographicService) loadMetadata() (string, string) {
	b, err := os.ReadFile(s.Paths.TempJson)
	if err != nil {
		LogError("infographic: temp.json 읽기 실패(메타정보 생략): " + err.Error())
		return "", ""
	}

	var doc infographicDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		LogError("infographic: temp.json 파싱 실패(메타정보 생략): " + err.Error())
		return "", ""
	}

	return stripQTTitlePrefix(doc.Metadata.Title), normalizeBibleReference(doc.Metadata.BibleText)
}

// stripQTTitlePrefix는 인포그래픽 제목에 불필요한 "[QT]" 접두어를 제거한다.
func stripQTTitlePrefix(title string) string {
	t := strings.TrimSpace(title)
	if strings.HasPrefix(t, "[QT]") {
		t = strings.TrimSpace(strings.TrimPrefix(t, "[QT]"))
	}
	return t
}

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type LLMService struct {
	APIKey string
	Model  string
	Client *http.Client
}

func NewLLMService() (*LLMService, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY가 비어 있습니다")
	}

	return &LLMService{
		APIKey: apiKey,
		Model:  "gpt-5.4-mini",
		Client: &http.Client{
			Timeout: 180 * time.Second,
		},
	}, nil
}

type ResponsesAPIRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type ResponsesAPIResponse struct {
	OutputText string `json:"output_text"`
}

func CleanLLMJSONOutput(s string) string {
	s = strings.TrimSpace(s)

	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSpace(s)
	}
	if strings.HasPrefix(s, "```JSON") {
		s = strings.TrimPrefix(s, "```JSON")
		s = strings.TrimSpace(s)
	}
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSpace(s)
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}

	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		s = strings.TrimSpace(s[start : end+1])
	}

	// LLM이 구조적 콤마를 빠뜨려 JSON이 깨지는 경우가 있어, 유효하지 않을 때만 보정한다.
	if !json.Valid([]byte(s)) {
		if repaired := repairJSONCommas(s); json.Valid([]byte(repaired)) {
			s = repaired
		}
	}

	return s
}

// repairJSONCommas는 LLM이 자주 만드는 "구조적 콤마 누락"을 보정한다.
// 문자열 밖에서 값 종료 토큰(} ] ")과 값/문자열 시작 토큰({ [ ")이
// 콤마 없이 인접하면 콤마를 삽입한다. 유효한 JSON에는 이 패턴이 존재하지 않으므로
// 정상 입력에는 영향을 주지 않는다. (키 뒤에는 항상 ':'가 오므로 오삽입되지 않는다.)
func repairJSONCommas(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 32)

	inString := false
	escaped := false
	var lastSig byte // 문자열 밖에서 마지막으로 본 유의미 문자(닫는 따옴표 포함)

	for i := 0; i < len(s); i++ {
		c := s[i]

		if inString {
			b.WriteByte(c)
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inString = false
				lastSig = '"'
			}
			continue
		}

		switch c {
		case ' ', '\t', '\r', '\n':
			b.WriteByte(c)
		case '"', '{', '[':
			if lastSig == '}' || lastSig == ']' || lastSig == '"' {
				b.WriteByte(',')
			}
			if c == '"' {
				inString = true
			}
			b.WriteByte(c)
			lastSig = c
		default:
			b.WriteByte(c)
			lastSig = c
		}
	}

	return b.String()
}

func (s *LLMService) BuildPrompt(meta QTMeta) string {
	return BuildQTPromptJSON(meta)
}

func (s *LLMService) GenerateQTJSON(meta QTMeta) (string, error) {
	if strings.TrimSpace(meta.Title) == "" {
		return "", fmt.Errorf("제목이 비어 있습니다")
	}
	if strings.TrimSpace(meta.BibleText) == "" {
		return "", fmt.Errorf("본문 성구가 비어 있습니다")
	}
	if strings.TrimSpace(meta.RawText) == "" {
		return "", fmt.Errorf("원문 텍스트가 비어 있습니다")
	}

	prompt := BuildQTPromptJSON(meta)

	reqBody := ResponsesAPIRequest{
		Model: s.Model,
		Input: prompt,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("요청 JSON 생성 실패: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/responses", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("요청 생성 실패: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("응답 읽기 실패: %w", err)
	}

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("LLM 호출 실패: status=%s, body=%s", resp.Status, string(respBytes))
	}

	var result ResponsesAPIResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("응답 JSON 파싱 실패: %w, body=%s", err, string(respBytes))
	}

	if result.OutputText == "" {
		return "", fmt.Errorf("LLM 응답에 output_text가 없습니다: %s", string(respBytes))
	}

	cleaned := CleanLLMJSONOutput(result.OutputText)
	if cleaned == "" {
		return "", fmt.Errorf("정리 후 JSON 결과가 비어 있습니다")
	}

	return cleaned, nil
}

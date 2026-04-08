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

	return s
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

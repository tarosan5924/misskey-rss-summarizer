package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"misskey-rss-summarizer/internal/domain/repository"
)

// geminiSummarizer はGoogle Gemini APIを使用した要約実装
type geminiSummarizer struct {
	apiKey         string
	model          string
	maxTokens      int
	systemPrompt   string
	maxInputLength int
	client         *http.Client
}

func newGeminiSummarizer(cfg Config) (repository.SummarizerRepository, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}

	model := cfg.Model
	if model == "" {
		model = "gemini-1.5-flash" // コスト効率の良いデフォルト
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 500
	}

	prompt := cfg.Prompt
	if prompt == "" {
		prompt = DefaultSystemPrompt
	}

	maxInputLength := cfg.MaxInputLength
	if maxInputLength == 0 {
		maxInputLength = 4000
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &geminiSummarizer{
		apiKey:         cfg.APIKey,
		model:          model,
		maxTokens:      maxTokens,
		systemPrompt:   prompt,
		maxInputLength: maxInputLength,
		client:         &http.Client{Timeout: timeout},
	}, nil
}

func (s *geminiSummarizer) Summarize(ctx context.Context, content, title string) (string, error) {
	// 入力テキストの長さを制限
	if len(content) > s.maxInputLength {
		content = content[:s.maxInputLength] + "..."
	}

	userPrompt := fmt.Sprintf("記事タイトル: %s\n\n記事内容:\n%s", title, content)

	// Gemini APIリクエストペイロードの構築
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": s.systemPrompt},
				},
			},
			{
				"role": "model",
				"parts": []map[string]string{
					{"text": "承知しました。記事を要約します。"},
				},
			},
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": userPrompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": s.maxTokens,
			"temperature":     0.3,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Gemini API呼び出し
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		s.model, s.apiKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API returned status %d", resp.StatusCode)
	}

	// レスポンスのパース
	var apiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no summary returned from Gemini API")
	}

	summary := apiResp.Candidates[0].Content.Parts[0].Text

	return summary, nil
}

func (s *geminiSummarizer) IsEnabled() bool {
	return true
}

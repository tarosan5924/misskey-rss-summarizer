package llm

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/genai"

	"misskey-rss-summarizer/internal/domain/repository"
)

// geminiSummarizer はGoogle Gemini APIを使用した要約実装
type geminiSummarizer struct {
	client       *genai.Client
	model        string
	maxTokens    *int32
	systemPrompt string
	timeout      time.Duration
}

func newGeminiSummarizer(cfg Config) (repository.SummarizerRepository, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}

	if cfg.Model == "" {
		return nil, fmt.Errorf("Gemini model name is required")
	}

	// maxTokensが0の場合はnil（指定なし）にする
	var maxTokens *int32
	if cfg.MaxTokens > 0 {
		tokens := int32(cfg.MaxTokens)
		maxTokens = &tokens
	}

	systemInstruction := cfg.SystemInstruction
	if systemInstruction == "" {
		systemInstruction = DefaultSystemPrompt
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Gemini クライアントを作成
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: cfg.APIKey,
		// Backend はデフォルトで BackendGeminiAPI が使用される
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &geminiSummarizer{
		client:       client,
		model:        cfg.Model,
		maxTokens:    maxTokens,
		systemPrompt: systemInstruction,
		timeout:      timeout,
	}, nil
}

func (s *geminiSummarizer) Summarize(ctx context.Context, url, title string) (string, error) {
	// タイムアウト付きコンテキストを作成
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// URLと記事タイトルをプロンプトに含める
	userPrompt := fmt.Sprintf("以下のURLの記事を要約してください。\n\n記事タイトル: %s\n記事URL: %s", title, url)

	// システムインストラクションとユーザープロンプトを設定
	systemInstruction := genai.NewContentFromText(s.systemPrompt, genai.RoleUser)
	userContent := genai.NewContentFromText(userPrompt, genai.RoleUser)

	// GenerateContent設定
	temperature := float32(0.3)
	config := &genai.GenerateContentConfig{
		Temperature:       &temperature,
		SystemInstruction: systemInstruction,
	}
	// maxTokensが指定されている場合のみ設定
	if s.maxTokens != nil {
		config.MaxOutputTokens = *s.maxTokens
	}

	// Gemini API呼び出し
	resp, err := s.client.Models.GenerateContent(ctx, s.model, []*genai.Content{userContent}, config)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	// レスポンスから要約を抽出
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates returned from Gemini API")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("no content in candidate response")
	}

	summary := candidate.Content.Parts[0].Text

	return summary, nil
}

func (s *geminiSummarizer) IsEnabled() bool {
	return true
}

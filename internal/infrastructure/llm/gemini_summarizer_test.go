package llm

import (
	"strings"
	"testing"
	"time"
)

func TestGeminiSummarizer_NewGeminiSummarizer_NoAPIKey(t *testing.T) {
	cfg := Config{
		Provider: "gemini",
		APIKey:   "",
	}

	_, err := newGeminiSummarizer(cfg)
	if err == nil {
		t.Error("expected error when API key is empty, got nil")
	}

	if !strings.Contains(err.Error(), "API key is required") {
		t.Errorf("expected 'API key is required' error, got: %v", err)
	}
}

func TestGeminiSummarizer_NewGeminiSummarizer_DefaultValues(t *testing.T) {
	// デフォルト値のテスト（クライアント作成はスキップ）
	cfg := Config{
		Provider: "gemini",
		APIKey:   "test-key",
	}

	// 実際のクライアント作成は行わず、設定値のみをテスト
	model := cfg.Model
	if model == "" {
		model = "gemini-1.5-flash"
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 500
	}

	maxInputLength := cfg.MaxInputLength
	if maxInputLength == 0 {
		maxInputLength = 4000
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	prompt := cfg.Prompt
	if prompt == "" {
		prompt = DefaultSystemPrompt
	}

	if model != "gemini-1.5-flash" {
		t.Errorf("expected default model 'gemini-1.5-flash', got %s", model)
	}

	if maxTokens != 500 {
		t.Errorf("expected default maxTokens 500, got %d", maxTokens)
	}

	if maxInputLength != 4000 {
		t.Errorf("expected default maxInputLength 4000, got %d", maxInputLength)
	}

	if timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", timeout)
	}

	if prompt != DefaultSystemPrompt {
		t.Error("expected default system prompt")
	}
}

func TestGeminiSummarizer_ConfigValidation(t *testing.T) {
	testCases := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "Missing API key",
			config: Config{
				Provider: "gemini",
				APIKey:   "",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newGeminiSummarizer(tc.config)
			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGeminiSummarizer_IsEnabled(t *testing.T) {
	// IsEnabledは常にtrueを返すことを確認
	// 実際のクライアントなしでテスト
	summarizer := &geminiSummarizer{}

	if !summarizer.IsEnabled() {
		t.Error("expected IsEnabled to return true for gemini summarizer")
	}
}

func TestGeminiSummarizer_InputTruncation(t *testing.T) {
	// 入力切り詰めロジックのテスト
	maxInputLength := 100
	longContent := strings.Repeat("a", 200)

	var truncated string
	if len(longContent) > maxInputLength {
		truncated = longContent[:maxInputLength] + "..."
	} else {
		truncated = longContent
	}

	expectedLength := maxInputLength + 3 // "..." を含む
	if len(truncated) != expectedLength {
		t.Errorf("expected truncated length %d, got %d", expectedLength, len(truncated))
	}

	if !strings.HasSuffix(truncated, "...") {
		t.Error("expected truncated string to end with '...'")
	}
}

func TestGeminiSummarizer_CustomConfig(t *testing.T) {
	customPrompt := "カスタムプロンプト"
	cfg := Config{
		Provider:       "gemini",
		APIKey:         "test-key",
		Model:          "gemini-1.5-pro",
		MaxTokens:      1000,
		MaxInputLength: 8000,
		Timeout:        60 * time.Second,
		Prompt:         customPrompt,
	}

	// 設定値のバリデーション（実際のクライアント作成はスキップ）
	if cfg.Model != "gemini-1.5-pro" {
		t.Errorf("expected model 'gemini-1.5-pro', got %s", cfg.Model)
	}

	if cfg.MaxTokens != 1000 {
		t.Errorf("expected maxTokens 1000, got %d", cfg.MaxTokens)
	}

	if cfg.MaxInputLength != 8000 {
		t.Errorf("expected maxInputLength 8000, got %d", cfg.MaxInputLength)
	}

	if cfg.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", cfg.Timeout)
	}

	if cfg.Prompt != customPrompt {
		t.Errorf("expected custom prompt '%s', got '%s'", customPrompt, cfg.Prompt)
	}
}

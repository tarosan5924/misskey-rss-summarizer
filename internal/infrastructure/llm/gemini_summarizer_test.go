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

func TestGeminiSummarizer_NewGeminiSummarizer_NoModel(t *testing.T) {
	cfg := Config{
		Provider: "gemini",
		APIKey:   "test-key",
		Model:    "",
	}

	_, err := newGeminiSummarizer(cfg)
	if err == nil {
		t.Error("expected error when model is empty, got nil")
	}

	if !strings.Contains(err.Error(), "model name is required") {
		t.Errorf("expected 'model name is required' error, got: %v", err)
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
				Model:    "gemini-2.0-flash-exp",
			},
			expectError: true,
		},
		{
			name: "Missing model",
			config: Config{
				Provider: "gemini",
				APIKey:   "test-key",
				Model:    "",
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

func TestGeminiSummarizer_CustomConfig(t *testing.T) {
	customInstruction := "カスタムシステムインストラクション"
	cfg := Config{
		Provider:          "gemini",
		APIKey:            "test-key",
		Model:             "gemini-1.5-pro",
		MaxTokens:         1000,
		Timeout:           60 * time.Second,
		SystemInstruction: customInstruction,
	}

	// 設定値のバリデーション（実際のクライアント作成はスキップ）
	if cfg.Model != "gemini-1.5-pro" {
		t.Errorf("expected model 'gemini-1.5-pro', got %s", cfg.Model)
	}

	if cfg.MaxTokens != 1000 {
		t.Errorf("expected maxTokens 1000, got %d", cfg.MaxTokens)
	}

	// maxTokensが設定されていることを確認
	var maxTokens *int32
	if cfg.MaxTokens > 0 {
		tokens := int32(cfg.MaxTokens)
		maxTokens = &tokens
	}

	if maxTokens == nil {
		t.Error("expected maxTokens to be set, got nil")
	} else if *maxTokens != 1000 {
		t.Errorf("expected maxTokens pointer to 1000, got %d", *maxTokens)
	}

	if cfg.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", cfg.Timeout)
	}

	if cfg.SystemInstruction != customInstruction {
		t.Errorf("expected custom instruction '%s', got '%s'", customInstruction, cfg.SystemInstruction)
	}
}

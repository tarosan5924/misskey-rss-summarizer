package llm

import (
	"strings"
	"testing"
	"time"
)

func TestNewSummarizerRepository_Gemini(t *testing.T) {
	cfg := Config{
		Provider: "gemini",
		APIKey:   "test-api-key",
		Model:    "gemini-2.0-flash-exp",
	}

	repo, err := NewSummarizerRepository(cfg)
	if err != nil {
		t.Fatalf("failed to create gemini summarizer: %v", err)
	}

	if !repo.IsEnabled() {
		t.Error("expected gemini summarizer to be enabled")
	}

	if _, ok := repo.(*geminiSummarizer); !ok {
		t.Error("expected geminiSummarizer type")
	}
}

func TestNewSummarizerRepository_Noop(t *testing.T) {
	testCases := []struct {
		name     string
		provider string
	}{
		{"empty provider", ""},
		{"noop provider", "noop"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				Provider: tc.provider,
			}

			repo, err := NewSummarizerRepository(cfg)
			if err != nil {
				t.Fatalf("failed to create noop summarizer: %v", err)
			}

			if repo.IsEnabled() {
				t.Error("expected noop summarizer to be disabled")
			}

			if _, ok := repo.(*noopSummarizer); !ok {
				t.Error("expected noopSummarizer type")
			}
		})
	}
}

func TestNewSummarizerRepository_UnknownProvider(t *testing.T) {
	cfg := Config{
		Provider: "unknown-provider",
	}

	_, err := NewSummarizerRepository(cfg)
	if err == nil {
		t.Error("expected error for unknown provider, got nil")
	}

	if !strings.Contains(err.Error(), "unknown LLM provider") {
		t.Errorf("expected 'unknown LLM provider' error, got: %v", err)
	}
}

func TestNewSummarizerRepository_GeminiNoAPIKey(t *testing.T) {
	cfg := Config{
		Provider: "gemini",
		APIKey:   "",
		Model:    "gemini-2.0-flash-exp",
	}

	_, err := NewSummarizerRepository(cfg)
	if err == nil {
		t.Error("expected error when gemini API key is empty, got nil")
	}
}

func TestNewSummarizerRepository_GeminiNoModel(t *testing.T) {
	cfg := Config{
		Provider: "gemini",
		APIKey:   "test-key",
		Model:    "",
	}

	_, err := NewSummarizerRepository(cfg)
	if err == nil {
		t.Error("expected error when gemini model is empty, got nil")
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{
		Provider: "gemini",
		APIKey:   "test-key",
		Model:    "gemini-2.0-flash-exp",
	}

	// Gemini summarizerを作成してデフォルト値を確認
	summarizer, err := newGeminiSummarizer(cfg)
	if err != nil {
		t.Fatalf("failed to create summarizer: %v", err)
	}

	gs := summarizer.(*geminiSummarizer)

	// デフォルト値の確認
	if gs.model != "gemini-2.0-flash-exp" {
		t.Errorf("expected model 'gemini-2.0-flash-exp', got %s", gs.model)
	}

	if gs.maxTokens != nil {
		t.Errorf("expected default maxTokens to be nil (no limit), got %d", *gs.maxTokens)
	}

	expectedTimeout := 30 * time.Second
	if gs.timeout != expectedTimeout {
		t.Errorf("expected default timeout %v, got %v", expectedTimeout, gs.timeout)
	}

	if gs.systemPrompt != DefaultSystemPrompt {
		t.Error("expected default system prompt")
	}
}

func TestConfig_CustomValues(t *testing.T) {
	customInstruction := "カスタムシステムインストラクション"
	cfg := Config{
		Provider:          "gemini",
		APIKey:            "test-key",
		Model:             "gemini-1.5-pro",
		MaxTokens:         1000,
		Timeout:           60 * time.Second,
		SystemInstruction: customInstruction,
	}

	summarizer, err := newGeminiSummarizer(cfg)
	if err != nil {
		t.Fatalf("failed to create summarizer: %v", err)
	}

	gs := summarizer.(*geminiSummarizer)

	if gs.model != "gemini-1.5-pro" {
		t.Errorf("expected model 'gemini-1.5-pro', got %s", gs.model)
	}

	if gs.maxTokens == nil {
		t.Error("expected maxTokens to be set, got nil")
	} else if *gs.maxTokens != 1000 {
		t.Errorf("expected maxTokens 1000, got %d", *gs.maxTokens)
	}

	if gs.timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", gs.timeout)
	}

	if gs.systemPrompt != customInstruction {
		t.Errorf("expected custom instruction '%s', got '%s'", customInstruction, gs.systemPrompt)
	}
}

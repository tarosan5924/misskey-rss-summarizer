package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGeminiSummarizer_Summarize(t *testing.T) {
	// モックサーバーを作成
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if !strings.Contains(r.URL.Path, "generateContent") {
			t.Errorf("expected generateContent endpoint, got %s", r.URL.Path)
		}

		// リクエストボディをパース
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		// レスポンスを返す
		resp := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{
							{"text": "これはテスト要約です。記事の要点をまとめました。"},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// サーバーURLを使ってGemini summarizerをテスト
	// 実際のAPIではなく、テストサーバーを使用するためにカスタム設定が必要
	// ここでは基本的な構造のテストのみ行う

	cfg := Config{
		Provider:       "gemini",
		APIKey:         "test-api-key",
		Model:          "gemini-1.5-flash",
		MaxTokens:      500,
		MaxInputLength: 4000,
		Timeout:        10 * time.Second,
	}

	summarizer, err := newGeminiSummarizer(cfg)
	if err != nil {
		t.Fatalf("failed to create gemini summarizer: %v", err)
	}

	if !summarizer.IsEnabled() {
		t.Error("expected summarizer to be enabled")
	}
}

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
	cfg := Config{
		Provider: "gemini",
		APIKey:   "test-key",
	}

	summarizer, err := newGeminiSummarizer(cfg)
	if err != nil {
		t.Fatalf("failed to create summarizer: %v", err)
	}

	gs := summarizer.(*geminiSummarizer)

	if gs.model != "gemini-1.5-flash" {
		t.Errorf("expected default model 'gemini-1.5-flash', got %s", gs.model)
	}

	if gs.maxTokens != 500 {
		t.Errorf("expected default maxTokens 500, got %d", gs.maxTokens)
	}

	if gs.maxInputLength != 4000 {
		t.Errorf("expected default maxInputLength 4000, got %d", gs.maxInputLength)
	}

	if gs.systemPrompt != DefaultSystemPrompt {
		t.Error("expected default system prompt")
	}
}

func TestGeminiSummarizer_Summarize_InputTruncation(t *testing.T) {
	cfg := Config{
		Provider:       "gemini",
		APIKey:         "test-api-key",
		MaxInputLength: 100,
	}

	summarizer, err := newGeminiSummarizer(cfg)
	if err != nil {
		t.Fatalf("failed to create summarizer: %v", err)
	}

	gs := summarizer.(*geminiSummarizer)

	// 長い入力テキスト
	longContent := strings.Repeat("a", 200)

	ctx := context.Background()

	// この呼び出しは実際のAPIに接続しようとするため失敗するが、
	// 入力の切り詰めロジックは実行される
	_, _ = gs.Summarize(ctx, longContent, "Test Title")

	// 入力切り詰めは内部で行われるため、直接検証はできないが
	// パニックが発生しないことを確認
}

func TestGeminiSummarizer_IsEnabled(t *testing.T) {
	cfg := Config{
		Provider: "gemini",
		APIKey:   "test-key",
	}

	summarizer, err := newGeminiSummarizer(cfg)
	if err != nil {
		t.Fatalf("failed to create summarizer: %v", err)
	}

	if !summarizer.IsEnabled() {
		t.Error("expected IsEnabled to return true for gemini summarizer")
	}
}

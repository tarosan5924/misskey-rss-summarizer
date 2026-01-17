package llm

import (
	"context"
	"testing"
)

func TestNoopSummarizer_Summarize(t *testing.T) {
	summarizer := newNoopSummarizer()

	ctx := context.Background()
	summary, err := summarizer.Summarize(ctx, "http://example.com/article", "test title")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if summary != "" {
		t.Errorf("expected empty summary, got %s", summary)
	}
}

func TestNoopSummarizer_IsEnabled(t *testing.T) {
	summarizer := newNoopSummarizer()

	if summarizer.IsEnabled() {
		t.Error("expected IsEnabled to return false for noop summarizer")
	}
}

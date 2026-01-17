package llm

import (
	"context"

	"misskeyRSSbot/internal/domain/repository"
)

// noopSummarizer は要約機能が無効の場合に使用される何もしない実装
type noopSummarizer struct{}

func newNoopSummarizer() repository.SummarizerRepository {
	return &noopSummarizer{}
}

func (s *noopSummarizer) Summarize(ctx context.Context, url, title string) (string, error) {
	return "", nil
}

func (s *noopSummarizer) IsEnabled() bool {
	return false
}

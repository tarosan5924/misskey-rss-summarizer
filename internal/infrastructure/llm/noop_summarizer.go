package llm

import (
	"context"

	"misskeyRSSbot/internal/domain/repository"
)

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

package llm

import (
	"context"
	"fmt"
	"time"

	"misskeyRSSbot/internal/domain/repository"
)

type Config struct {
	Provider          string
	APIKey            string
	Model             string
	MaxTokens         int
	SystemInstruction string
	Timeout           time.Duration
}

const DefaultSystemPrompt = `あなたは記事要約の専門家です。
以下の記事を簡潔に要約してください。
- 3〜5文程度で要点をまとめる
- 重要な情報を優先する
- 日本語で出力する`

func NewSummarizerRepository(ctx context.Context, cfg Config) (repository.SummarizerRepository, error) {
	switch cfg.Provider {
	case "gemini":
		return newGeminiSummarizer(ctx, cfg)
	case "noop", "":
		return newNoopSummarizer(), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.Provider)
	}
}

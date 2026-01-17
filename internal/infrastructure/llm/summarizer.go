package llm

import (
	"fmt"
	"time"

	"misskey-rss-summarizer/internal/domain/repository"
)

// Config はLLM要約機能の設定
type Config struct {
	Provider          string        // "gemini" or "noop" (empty defaults to "noop")
	APIKey            string        // LLM APIキー
	Model             string        // モデル名
	MaxTokens         int           // 最大出力トークン数
	SystemInstruction string        // カスタムシステムインストラクション
	Timeout           time.Duration // APIタイムアウト
}

// DefaultSystemPrompt はデフォルトの要約プロンプト
const DefaultSystemPrompt = `あなたは記事要約の専門家です。
以下の記事を簡潔に要約してください。
- 3〜5文程度で要点をまとめる
- 重要な情報を優先する
- 日本語で出力する`

// NewSummarizerRepository はConfigに基づいてSummarizerRepositoryを生成します
func NewSummarizerRepository(cfg Config) (repository.SummarizerRepository, error) {
	switch cfg.Provider {
	case "gemini":
		return newGeminiSummarizer(cfg)
	case "noop", "":
		return newNoopSummarizer(), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.Provider)
	}
}

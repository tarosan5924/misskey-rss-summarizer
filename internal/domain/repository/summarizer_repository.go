package repository

import "context"

// SummarizerRepository はコンテンツの要約機能を提供するインターフェース
type SummarizerRepository interface {
	// Summarize はURLの記事を要約します
	// url: 記事のURL（LLMがアクセスして内容を取得）
	// title: 記事タイトル（コンテキスト情報として使用）
	// 戻り値: 要約文字列, エラー
	Summarize(ctx context.Context, url, title string) (string, error)

	// IsEnabled は要約機能が有効かどうかを返します
	IsEnabled() bool
}

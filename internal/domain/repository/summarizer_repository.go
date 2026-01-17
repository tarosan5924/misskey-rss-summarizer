package repository

import "context"

// SummarizerRepository はコンテンツの要約機能を提供するインターフェース
type SummarizerRepository interface {
	// Summarize はコンテンツを要約します
	// content: 要約対象のテキスト
	// title: 記事タイトル（コンテキスト情報として使用）
	// 戻り値: 要約文字列, エラー
	Summarize(ctx context.Context, content, title string) (string, error)

	// IsEnabled は要約機能が有効かどうかを返します
	IsEnabled() bool
}

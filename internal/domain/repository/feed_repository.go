package repository

import (
	"context"

	"misskey-rss-summarizer/internal/domain/entity"
)

type FeedRepository interface {
	Fetch(ctx context.Context, url string) ([]*entity.FeedEntry, error)
}

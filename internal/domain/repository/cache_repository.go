package repository

import (
	"context"
	"time"
)

type CacheRepository interface {
	GetLatestPublishedTime(ctx context.Context, rssURL string) (time.Time, error)
	SaveLatestPublishedTime(ctx context.Context, rssURL string, published time.Time) error
	IsProcessed(ctx context.Context, guid string) (bool, error)
	MarkAsProcessed(ctx context.Context, guid string) error
}

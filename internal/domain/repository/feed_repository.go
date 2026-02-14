package repository

import (
	"context"

	"misskeyRSSbot/internal/domain/entity"
)

type FeedRepository interface {
	Fetch(ctx context.Context, url string, useFilter bool) ([]*entity.FeedEntry, error)
}

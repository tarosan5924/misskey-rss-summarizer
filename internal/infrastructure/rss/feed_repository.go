package rss

import (
	"context"
	"fmt"
	"strings"

	"misskeyRSSbot/internal/domain/entity"
	"misskeyRSSbot/internal/domain/repository"

	"github.com/mmcdole/gofeed"
)

type feedRepository struct {
	parser *gofeed.Parser
}

func NewFeedRepository() repository.FeedRepository {
	return &feedRepository{
		parser: gofeed.NewParser(),
	}
}

func (r *feedRepository) Fetch(ctx context.Context, url string, keywords []string) ([]*entity.FeedEntry, error) {
	feed, err := r.parser.ParseURLWithContext(url, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	entries := make([]*entity.FeedEntry, 0, len(feed.Items))

	for _, item := range feed.Items {
		if item.PublishedParsed == nil {
			continue
		}

		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}

		entry := entity.NewFeedEntry(
			item.Title,
			item.Link,
			item.Description,
			*item.PublishedParsed,
			guid,
		)

		// Keywords が空の場合はフィルターなし（全件返す）
		if len(keywords) > 0 {
			found := false
			for _, k := range keywords {
				if strings.Contains(entry.Title, k) || strings.Contains(entry.Description, k) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

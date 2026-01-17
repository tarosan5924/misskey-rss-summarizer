package storage

import (
	"context"
	"sync"
	"time"

	"misskeyRSSbot/internal/domain/repository"
)

type memoryCache struct {
	mu              sync.RWMutex
	latestPublished map[string]time.Time
	processedGUIDs  map[string]bool
}

func NewMemoryCacheRepository() repository.CacheRepository {
	return &memoryCache{
		latestPublished: make(map[string]time.Time),
		processedGUIDs:  make(map[string]bool),
	}
}

func (c *memoryCache) GetLatestPublishedTime(ctx context.Context, rssURL string) (time.Time, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	t, ok := c.latestPublished[rssURL]
	if !ok {
		return time.Time{}, nil
	}
	return t, nil
}

func (c *memoryCache) SaveLatestPublishedTime(ctx context.Context, rssURL string, published time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.latestPublished[rssURL] = published
	return nil
}

func (c *memoryCache) IsProcessed(ctx context.Context, guid string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	processed, ok := c.processedGUIDs[guid]
	return ok && processed, nil
}

func (c *memoryCache) MarkAsProcessed(ctx context.Context, guid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.processedGUIDs[guid] = true
	return nil
}

package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"misskeyRSSbot/internal/domain/entity"
)

type mockFeedRepository struct {
	entries []*entity.FeedEntry
	err     error
}

func (m *mockFeedRepository) Fetch(ctx context.Context, url string) ([]*entity.FeedEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.entries, nil
}

type mockNoteRepository struct {
	posted []*entity.Note
	err    error
}

func (m *mockNoteRepository) Post(ctx context.Context, note *entity.Note) error {
	if m.err != nil {
		return m.err
	}
	m.posted = append(m.posted, note)
	return nil
}

type mockCacheRepository struct {
	latestTime     time.Time
	processedGUIDs map[string]bool
}

func newMockCacheRepository() *mockCacheRepository {
	return &mockCacheRepository{
		processedGUIDs: make(map[string]bool),
	}
}

func (m *mockCacheRepository) GetLatestPublishedTime(ctx context.Context, rssURL string) (time.Time, error) {
	return m.latestTime, nil
}

func (m *mockCacheRepository) SaveLatestPublishedTime(ctx context.Context, rssURL string, published time.Time) error {
	m.latestTime = published
	return nil
}

func (m *mockCacheRepository) IsProcessed(ctx context.Context, guid string) (bool, error) {
	return m.processedGUIDs[guid], nil
}

func (m *mockCacheRepository) MarkAsProcessed(ctx context.Context, guid string) error {
	m.processedGUIDs[guid] = true
	return nil
}

func TestRSSFeedService_ProcessFeed_NewEntries(t *testing.T) {
	ctx := context.Background()

	now := time.Now()
	entries := []*entity.FeedEntry{
		entity.NewFeedEntry("Article 1", "https://example.tld/1", "Desc 1", now.Add(-2*time.Hour), "guid-1"),
		entity.NewFeedEntry("Article 2", "https://example.tld/2", "Desc 2", now.Add(-1*time.Hour), "guid-2"),
		entity.NewFeedEntry("Article 3", "https://example.tld/3", "Desc 3", now, "guid-3"),
	}

	feedRepo := &mockFeedRepository{entries: entries}
	noteRepo := &mockNoteRepository{}
	cacheRepo := newMockCacheRepository()

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo)

	err := service.ProcessFeed(ctx, "https://example.tld/rss")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(noteRepo.posted) != 1 {
		t.Errorf("expected 1 note posted on first run (most recent only), got %d", len(noteRepo.posted))
	}

	if len(noteRepo.posted) > 0 && noteRepo.posted[0].Text != "📰 Article 3\nhttps://example.tld/3" {
		t.Errorf("expected most recent article (Article 3) to be posted first")
	}

	if !cacheRepo.processedGUIDs["guid-3"] {
		t.Errorf("GUID guid-3 was not marked as processed")
	}
}

func TestRSSFeedService_ProcessFeed_SkipProcessedEntries(t *testing.T) {
	ctx := context.Background()

	now := time.Now()
	entries := []*entity.FeedEntry{
		entity.NewFeedEntry("Article 1", "https://example.tld/1", "Desc 1", now.Add(-1*time.Hour), "guid-1"),
		entity.NewFeedEntry("Article 2", "https://example.tld/2", "Desc 2", now, "guid-2"),
	}

	feedRepo := &mockFeedRepository{entries: entries}
	noteRepo := &mockNoteRepository{}
	cacheRepo := newMockCacheRepository()
	cacheRepo.processedGUIDs["guid-1"] = true
	cacheRepo.latestTime = now.Add(-2 * time.Hour)

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo)

	err := service.ProcessFeed(ctx, "https://example.tld/rss")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(noteRepo.posted) != 1 {
		t.Errorf("expected 1 note posted, got %d", len(noteRepo.posted))
	}
}

func TestRSSFeedService_ProcessFeed_FetchError(t *testing.T) {
	ctx := context.Background()

	feedRepo := &mockFeedRepository{err: errors.New("fetch error")}
	noteRepo := &mockNoteRepository{}
	cacheRepo := newMockCacheRepository()

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo)

	err := service.ProcessFeed(ctx, "https://example.tld/rss")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestRSSFeedService_ProcessAllFeeds(t *testing.T) {
	ctx := context.Background()

	now := time.Now()

	feedRepo := &mockFeedRepository{
		entries: []*entity.FeedEntry{
			entity.NewFeedEntry("Article 1", "https://example.tld/1", "Desc 1", now, "guid-1"),
			entity.NewFeedEntry("Article 2", "https://example.tld/2", "Desc 2", now.Add(1*time.Hour), "guid-2"),
		},
	}
	noteRepo := &mockNoteRepository{}
	cacheRepo := newMockCacheRepository()

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo)

	urls := []string{
		"https://example.tld/rss",
	}

	err := service.ProcessAllFeeds(ctx, urls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(noteRepo.posted) != 1 {
		t.Errorf("expected 1 note posted on first run (most recent only), got %d", len(noteRepo.posted))
	}
}

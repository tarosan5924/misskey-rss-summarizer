package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"misskey-rss-summarizer/internal/domain/entity"
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

type mockSummarizerRepository struct {
	summary string
	err     error
	enabled bool
	called  int
}

func (m *mockSummarizerRepository) Summarize(ctx context.Context, content, title string) (string, error) {
	m.called++
	if m.err != nil {
		return "", m.err
	}
	return m.summary, nil
}

func (m *mockSummarizerRepository) IsEnabled() bool {
	return m.enabled
}

type mockContentFetcher struct {
	content string
	err     error
	called  int
}

func (m *mockContentFetcher) FetchContent(ctx context.Context, url string) (string, error) {
	m.called++
	if m.err != nil {
		return "", m.err
	}
	return m.content, nil
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

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, nil, nil)

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

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, nil, nil)

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

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, nil, nil)

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

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, nil, nil)

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

func TestRSSFeedService_ProcessFeed_WithSummarizer(t *testing.T) {
	ctx := context.Background()

	now := time.Now()
	entries := []*entity.FeedEntry{
		entity.NewFeedEntry("Article 1", "https://example.tld/1", "This is a long description with more than 100 characters to test summarization feature. It contains enough content to trigger the summarization process.", now, "guid-1"),
	}

	feedRepo := &mockFeedRepository{entries: entries}
	noteRepo := &mockNoteRepository{}
	cacheRepo := newMockCacheRepository()
	summarizerRepo := &mockSummarizerRepository{
		summary: "これは要約されたテキストです。",
		enabled: true,
	}

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, summarizerRepo, nil)

	err := service.ProcessFeed(ctx, "https://example.tld/rss")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(noteRepo.posted) != 1 {
		t.Fatalf("expected 1 note posted, got %d", len(noteRepo.posted))
	}

	postedText := noteRepo.posted[0].Text
	if !strings.Contains(postedText, "【要約】") {
		t.Error("expected note to contain summary section")
	}

	if !strings.Contains(postedText, "これは要約されたテキストです。") {
		t.Error("expected note to contain summary text")
	}

	if summarizerRepo.called != 1 {
		t.Errorf("expected summarizer to be called once, got %d", summarizerRepo.called)
	}
}

func TestRSSFeedService_ProcessFeed_SummarizerDisabled(t *testing.T) {
	ctx := context.Background()

	now := time.Now()
	entries := []*entity.FeedEntry{
		entity.NewFeedEntry("Article 1", "https://example.tld/1", "Description", now, "guid-1"),
	}

	feedRepo := &mockFeedRepository{entries: entries}
	noteRepo := &mockNoteRepository{}
	cacheRepo := newMockCacheRepository()
	summarizerRepo := &mockSummarizerRepository{
		summary: "要約",
		enabled: false, // 無効
	}

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, summarizerRepo, nil)

	err := service.ProcessFeed(ctx, "https://example.tld/rss")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(noteRepo.posted) != 1 {
		t.Fatalf("expected 1 note posted, got %d", len(noteRepo.posted))
	}

	postedText := noteRepo.posted[0].Text
	if strings.Contains(postedText, "【要約】") {
		t.Error("expected note not to contain summary when summarizer is disabled")
	}

	if summarizerRepo.called != 0 {
		t.Errorf("expected summarizer not to be called, got %d calls", summarizerRepo.called)
	}
}

func TestRSSFeedService_ProcessFeed_WithContentFetcher(t *testing.T) {
	ctx := context.Background()

	now := time.Now()
	entries := []*entity.FeedEntry{
		entity.NewFeedEntry("Article 1", "https://example.tld/1", "Short", now, "guid-1"),
	}

	feedRepo := &mockFeedRepository{entries: entries}
	noteRepo := &mockNoteRepository{}
	cacheRepo := newMockCacheRepository()
	summarizerRepo := &mockSummarizerRepository{
		summary: "要約テキスト",
		enabled: true,
	}
	contentFetcher := &mockContentFetcher{
		content: "This is the full article content fetched from the web page. It contains much more detail than the short RSS description.",
	}

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, summarizerRepo, contentFetcher)

	err := service.ProcessFeed(ctx, "https://example.tld/rss")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if contentFetcher.called != 1 {
		t.Errorf("expected content fetcher to be called once (description < 100 chars), got %d", contentFetcher.called)
	}

	if summarizerRepo.called != 1 {
		t.Errorf("expected summarizer to be called once, got %d", summarizerRepo.called)
	}
}

func TestRSSFeedService_ProcessFeed_ContentFetcherError(t *testing.T) {
	ctx := context.Background()

	now := time.Now()
	entries := []*entity.FeedEntry{
		entity.NewFeedEntry("Article 1", "https://example.tld/1", "Short description", now, "guid-1"),
	}

	feedRepo := &mockFeedRepository{entries: entries}
	noteRepo := &mockNoteRepository{}
	cacheRepo := newMockCacheRepository()
	summarizerRepo := &mockSummarizerRepository{
		summary: "要約",
		enabled: true,
	}
	contentFetcher := &mockContentFetcher{
		err: errors.New("fetch error"),
	}

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, summarizerRepo, contentFetcher)

	err := service.ProcessFeed(ctx, "https://example.tld/rss")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// エラーが発生してもポストは続行される
	if len(noteRepo.posted) != 1 {
		t.Errorf("expected 1 note posted despite content fetcher error, got %d", len(noteRepo.posted))
	}
}

func TestRSSFeedService_ProcessFeed_SummarizerError(t *testing.T) {
	ctx := context.Background()

	now := time.Now()
	entries := []*entity.FeedEntry{
		entity.NewFeedEntry("Article 1", "https://example.tld/1", "This is a long description with more than 100 characters to test error handling.", now, "guid-1"),
	}

	feedRepo := &mockFeedRepository{entries: entries}
	noteRepo := &mockNoteRepository{}
	cacheRepo := newMockCacheRepository()
	summarizerRepo := &mockSummarizerRepository{
		err:     errors.New("summarization error"),
		enabled: true,
	}

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, summarizerRepo, nil)

	err := service.ProcessFeed(ctx, "https://example.tld/rss")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// エラーが発生しても要約なしでポストは続行される
	if len(noteRepo.posted) != 1 {
		t.Errorf("expected 1 note posted despite summarizer error, got %d", len(noteRepo.posted))
	}

	postedText := noteRepo.posted[0].Text
	if strings.Contains(postedText, "【要約】") {
		t.Error("expected note not to contain summary when summarization fails")
	}
}

func TestRSSFeedService_ProcessFeed_LongDescription_NoContentFetcher(t *testing.T) {
	ctx := context.Background()

	now := time.Now()
	longDesc := "This is a very long description from the RSS feed itself. " + strings.Repeat("More content. ", 20)
	entries := []*entity.FeedEntry{
		entity.NewFeedEntry("Article 1", "https://example.tld/1", longDesc, now, "guid-1"),
	}

	feedRepo := &mockFeedRepository{entries: entries}
	noteRepo := &mockNoteRepository{}
	cacheRepo := newMockCacheRepository()
	summarizerRepo := &mockSummarizerRepository{
		summary: "RSS要約",
		enabled: true,
	}
	contentFetcher := &mockContentFetcher{
		content: "Fetched content",
	}

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, summarizerRepo, contentFetcher)

	err := service.ProcessFeed(ctx, "https://example.tld/rss")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Description が100文字以上なのでコンテンツフェッチャーは呼ばれない
	if contentFetcher.called != 0 {
		t.Errorf("expected content fetcher not to be called (description >= 100 chars), got %d calls", contentFetcher.called)
	}

	// 要約は呼ばれる
	if summarizerRepo.called != 1 {
		t.Errorf("expected summarizer to be called once, got %d", summarizerRepo.called)
	}
}

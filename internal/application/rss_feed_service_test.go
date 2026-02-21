package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"misskeyRSSbot/internal/domain/entity"
	"misskeyRSSbot/internal/interfaces/config"
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

func (m *mockSummarizerRepository) Summarize(ctx context.Context, url, title string) (string, error) {
	m.called++
	if m.err != nil {
		return "", m.err
	}
	return m.summary, nil
}

func (m *mockSummarizerRepository) IsEnabled() bool {
	return m.enabled
}

func (m *mockSummarizerRepository) Close() error {
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

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, nil)

	err := service.ProcessFeed(ctx, config.RSSSettings{URL: "https://example.tld/rss"})
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

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, nil)

	err := service.ProcessFeed(ctx, config.RSSSettings{URL: "https://example.tld/rss"})
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

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, nil)

	err := service.ProcessFeed(ctx, config.RSSSettings{URL: "https://example.tld/rss"})
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

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, nil)

	settings := []config.RSSSettings{
		{URL: "https://example.tld/rss"},
	}

	err := service.ProcessAllFeeds(ctx, settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(noteRepo.posted) != 1 {
		t.Errorf("expected 1 note posted on first run (most recent only), got %d", len(noteRepo.posted))
	}
}

func TestRSSFeedService_ProcessAllFeeds_WithFilter(t *testing.T) {
	ctx := context.Background()

	now := time.Now()

	feedRepo := &mockFeedRepository{
		entries: []*entity.FeedEntry{
			entity.NewFeedEntry("Article 1", "https://example.tld/1", "Desc 1", now, "guid-1"),
		},
	}
	noteRepo := &mockNoteRepository{}
	cacheRepo := newMockCacheRepository()

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, nil)

	settings := []config.RSSSettings{
		{URL: "https://example.tld/rss1", Keywords: []string{"テスト"}},
		{URL: "https://example.tld/rss2"},
	}

	err := service.ProcessAllFeeds(ctx, settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, summarizerRepo)

	err := service.ProcessFeed(ctx, config.RSSSettings{URL: "https://example.tld/rss"})
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

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, summarizerRepo)

	err := service.ProcessFeed(ctx, config.RSSSettings{URL: "https://example.tld/rss"})
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

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, summarizerRepo)

	err := service.ProcessFeed(ctx, config.RSSSettings{URL: "https://example.tld/rss"})
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

func TestRSSFeedService_ProcessFeed_FirstRunLatestOnlyEnabled(t *testing.T) {
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

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, nil, WithFirstRunLatestOnly(true))

	err := service.ProcessFeed(ctx, config.RSSSettings{URL: "https://example.tld/rss"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(noteRepo.posted) != 1 {
		t.Errorf("expected 1 note posted (most recent only), got %d", len(noteRepo.posted))
	}

	if len(noteRepo.posted) > 0 && !strings.Contains(noteRepo.posted[0].Text, "Article 3") {
		t.Errorf("expected most recent article (Article 3) to be posted")
	}
}

func TestRSSFeedService_ProcessFeed_FirstRunLatestOnlyDisabled(t *testing.T) {
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

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, nil, WithFirstRunLatestOnly(false))

	err := service.ProcessFeed(ctx, config.RSSSettings{URL: "https://example.tld/rss"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(noteRepo.posted) != 3 {
		t.Errorf("expected 3 notes posted (all entries), got %d", len(noteRepo.posted))
	}
}

func TestRSSFeedService_ProcessFeed_FirstRunLatestOnlyDisabled_SkipProcessed(t *testing.T) {
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
	cacheRepo.processedGUIDs["guid-1"] = true

	service := NewRSSFeedService(feedRepo, noteRepo, cacheRepo, nil, WithFirstRunLatestOnly(false))

	err := service.ProcessFeed(ctx, config.RSSSettings{URL: "https://example.tld/rss"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(noteRepo.posted) != 2 {
		t.Errorf("expected 2 notes posted (skipping processed guid-1), got %d", len(noteRepo.posted))
	}
}

func TestFilterByKeywords_MatchesTitle(t *testing.T) {
	now := time.Now()
	entries := []*entity.FeedEntry{
		entity.NewFeedEntry("マユリカの新番組", "https://example.tld/1", "お笑いの話題", now, "guid-1"),
		entity.NewFeedEntry("関係ない記事", "https://example.tld/2", "関係ない内容", now, "guid-2"),
	}

	filtered := filterByKeywords(entries, []string{"マユリカ", "エバース"})

	if len(filtered) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(filtered))
	}
	if filtered[0].Title != "マユリカの新番組" {
		t.Errorf("expected 'マユリカの新番組', got '%s'", filtered[0].Title)
	}
}

func TestFilterByKeywords_MatchesDescription(t *testing.T) {
	now := time.Now()
	entries := []*entity.FeedEntry{
		entity.NewFeedEntry("お笑い番組まとめ", "https://example.tld/1", "エバースが出演する番組", now, "guid-1"),
		entity.NewFeedEntry("別の記事", "https://example.tld/2", "全く関係ない内容", now, "guid-2"),
	}

	filtered := filterByKeywords(entries, []string{"マユリカ", "エバース"})

	if len(filtered) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(filtered))
	}
	if filtered[0].GUID != "guid-1" {
		t.Errorf("expected guid-1, got '%s'", filtered[0].GUID)
	}
}

func TestFilterByKeywords_NoKeywordsReturnsAll(t *testing.T) {
	now := time.Now()
	entries := []*entity.FeedEntry{
		entity.NewFeedEntry("記事1", "https://example.tld/1", "内容1", now, "guid-1"),
		entity.NewFeedEntry("記事2", "https://example.tld/2", "内容2", now, "guid-2"),
	}

	filtered := filterByKeywords(entries, nil)

	if len(filtered) != 2 {
		t.Errorf("expected 2 entries when keywords is nil, got %d", len(filtered))
	}
}

func TestFilterByKeywords_EmptyKeywordsReturnsAll(t *testing.T) {
	now := time.Now()
	entries := []*entity.FeedEntry{
		entity.NewFeedEntry("記事1", "https://example.tld/1", "内容1", now, "guid-1"),
	}

	filtered := filterByKeywords(entries, []string{})

	if len(filtered) != 1 {
		t.Errorf("expected 1 entry when keywords is empty, got %d", len(filtered))
	}
}

func TestFilterByKeywords_NoMatch(t *testing.T) {
	now := time.Now()
	entries := []*entity.FeedEntry{
		entity.NewFeedEntry("関係ない記事", "https://example.tld/1", "関係ない内容", now, "guid-1"),
	}

	filtered := filterByKeywords(entries, []string{"マユリカ", "エバース"})

	if len(filtered) != 0 {
		t.Errorf("expected 0 entries, got %d", len(filtered))
	}
}

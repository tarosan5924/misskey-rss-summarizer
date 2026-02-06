package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func closeSQLiteCache(t *testing.T, cache interface{}) {
	t.Helper()
	if closer, ok := cache.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			t.Errorf("failed to close cache: %v", err)
		}
	}
}

func TestSQLiteCache_LatestPublishedTime(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	cache, err := NewSQLiteCacheRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer closeSQLiteCache(t, cache)

	ctx := context.Background()
	rssURL := "https://example.tld/rss"

	latest, err := cache.GetLatestPublishedTime(ctx, rssURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !latest.IsZero() {
		t.Errorf("expected zero time, got %v", latest)
	}

	now := time.Now().Truncate(time.Second)
	err = cache.SaveLatestPublishedTime(ctx, rssURL, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	latest, err = cache.GetLatestPublishedTime(ctx, rssURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !latest.Equal(now) {
		t.Errorf("expected %v, got %v", now, latest)
	}
}

func TestSQLiteCache_ProcessedGUIDs(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	cache, err := NewSQLiteCacheRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer closeSQLiteCache(t, cache)

	ctx := context.Background()
	guid := "test-guid-123"

	processed, err := cache.IsProcessed(ctx, guid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Error("expected not processed, but was processed")
	}

	err = cache.MarkAsProcessed(ctx, guid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	processed, err = cache.IsProcessed(ctx, guid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed {
		t.Error("expected processed, but was not processed")
	}
}

func TestSQLiteCache_MultipleRSSURLs(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	cache, err := NewSQLiteCacheRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer closeSQLiteCache(t, cache)

	ctx := context.Background()
	url1 := "https://example.tld/rss1"
	url2 := "https://example.tld/rss2"

	time1 := time.Now().Truncate(time.Second)
	time2 := time.Now().Add(1 * time.Hour).Truncate(time.Second)

	if err := cache.SaveLatestPublishedTime(ctx, url1, time1); err != nil {
		t.Fatalf("failed to save time1: %v", err)
	}
	if err := cache.SaveLatestPublishedTime(ctx, url2, time2); err != nil {
		t.Fatalf("failed to save time2: %v", err)
	}

	got1, _ := cache.GetLatestPublishedTime(ctx, url1)
	got2, _ := cache.GetLatestPublishedTime(ctx, url2)

	if !got1.Equal(time1) {
		t.Errorf("url1: expected %v, got %v", time1, got1)
	}
	if !got2.Equal(time2) {
		t.Errorf("url2: expected %v, got %v", time2, got2)
	}
}

func TestSQLiteCache_Persistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	ctx := context.Background()

	rssURL := "https://example.tld/rss"
	guid := "persistent-guid-456"
	publishedTime := time.Now().Truncate(time.Second)

	cache1, err := NewSQLiteCacheRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	err = cache1.SaveLatestPublishedTime(ctx, rssURL, publishedTime)
	if err != nil {
		t.Fatalf("failed to save published time: %v", err)
	}

	err = cache1.MarkAsProcessed(ctx, guid)
	if err != nil {
		t.Fatalf("failed to mark as processed: %v", err)
	}

	cache1.(*sqliteCache).Close()

	cache2, err := NewSQLiteCacheRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen cache: %v", err)
	}
	defer cache2.(*sqliteCache).Close()

	latest, err := cache2.GetLatestPublishedTime(ctx, rssURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !latest.Equal(publishedTime) {
		t.Errorf("expected %v, got %v", publishedTime, latest)
	}

	processed, err := cache2.IsProcessed(ctx, guid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed {
		t.Error("expected processed after reopen, but was not processed")
	}
}

func TestSQLiteCache_UpdateLatestPublishedTime(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	cache, err := NewSQLiteCacheRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer closeSQLiteCache(t, cache)

	ctx := context.Background()
	rssURL := "https://example.tld/rss"

	time1 := time.Now().Truncate(time.Second)
	time2 := time1.Add(1 * time.Hour)

	_ = cache.SaveLatestPublishedTime(ctx, rssURL, time1)
	_ = cache.SaveLatestPublishedTime(ctx, rssURL, time2)

	latest, _ := cache.GetLatestPublishedTime(ctx, rssURL)
	if !latest.Equal(time2) {
		t.Errorf("expected updated time %v, got %v", time2, latest)
	}
}

func TestSQLiteCache_DuplicateMarkAsProcessed(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	cache, err := NewSQLiteCacheRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer closeSQLiteCache(t, cache)

	ctx := context.Background()
	guid := "duplicate-guid"

	err = cache.MarkAsProcessed(ctx, guid)
	if err != nil {
		t.Fatalf("first mark failed: %v", err)
	}

	err = cache.MarkAsProcessed(ctx, guid)
	if err != nil {
		t.Fatalf("duplicate mark should not fail: %v", err)
	}
}

func TestSQLiteCache_InvalidPath(t *testing.T) {
	_, err := NewSQLiteCacheRepository("/nonexistent/path/test.db")
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestSQLiteCache_FileCreation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "subdir", "test.db")

	err := os.MkdirAll(filepath.Dir(dbPath), 0755)
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	cache, err := NewSQLiteCacheRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer closeSQLiteCache(t, cache)

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestSQLiteCache_CleanupOldGUIDs(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	cache, err := NewSQLiteCacheRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer closeSQLiteCache(t, cache)

	ctx := context.Background()
	sqlCache := cache.(*sqliteCache)

	_, execErr := sqlCache.db.ExecContext(ctx,
		"INSERT INTO processed_guids (guid, processed_at) VALUES (?, ?)",
		"old-guid-1", time.Now().Add(-48*time.Hour).Unix())
	if execErr != nil {
		t.Fatalf("failed to insert old-guid-1: %v", execErr)
	}
	_, execErr = sqlCache.db.ExecContext(ctx,
		"INSERT INTO processed_guids (guid, processed_at) VALUES (?, ?)",
		"old-guid-2", time.Now().Add(-25*time.Hour).Unix())
	if execErr != nil {
		t.Fatalf("failed to insert old-guid-2: %v", execErr)
	}
	_, execErr = sqlCache.db.ExecContext(ctx,
		"INSERT INTO processed_guids (guid, processed_at) VALUES (?, ?)",
		"new-guid-1", time.Now().Add(-1*time.Hour).Unix())
	if execErr != nil {
		t.Fatalf("failed to insert new-guid-1: %v", execErr)
	}

	deleted, err := sqlCache.CleanupOldGUIDs(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	if deleted != 2 {
		t.Errorf("expected 2 deleted, got %d", deleted)
	}

	processed, _ := cache.IsProcessed(ctx, "old-guid-1")
	if processed {
		t.Error("old-guid-1 should have been deleted")
	}

	processed, _ = cache.IsProcessed(ctx, "old-guid-2")
	if processed {
		t.Error("old-guid-2 should have been deleted")
	}

	processed, _ = cache.IsProcessed(ctx, "new-guid-1")
	if !processed {
		t.Error("new-guid-1 should still exist")
	}
}

func TestSQLiteCache_CleanupOldGUIDs_NoOldRecords(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	cache, err := NewSQLiteCacheRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer closeSQLiteCache(t, cache)

	ctx := context.Background()
	sqlCache := cache.(*sqliteCache)

	markErr := cache.MarkAsProcessed(ctx, "recent-guid")
	if markErr != nil {
		t.Fatalf("failed to mark as processed: %v", markErr)
	}

	deleted, err := sqlCache.CleanupOldGUIDs(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}
}

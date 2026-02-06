package storage

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache_LatestPublishedTime(t *testing.T) {
	cache := NewMemoryCacheRepository()
	ctx := context.Background()

	rssURL := "https://example.tld/rss"

	latest, err := cache.GetLatestPublishedTime(ctx, rssURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !latest.IsZero() {
		t.Errorf("expected zero time, got %v", latest)
	}

	now := time.Now()
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

func TestMemoryCache_ProcessedGUIDs(t *testing.T) {
	cache := NewMemoryCacheRepository()
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

func TestMemoryCache_MultipleRSSURLs(t *testing.T) {
	cache := NewMemoryCacheRepository()
	ctx := context.Background()

	url1 := "https://example.tld/rss1"
	url2 := "https://example.tld/rss2"

	time1 := time.Now()
	time2 := time.Now().Add(1 * time.Hour)

	_ = cache.SaveLatestPublishedTime(ctx, url1, time1)
	_ = cache.SaveLatestPublishedTime(ctx, url2, time2)

	got1, _ := cache.GetLatestPublishedTime(ctx, url1)
	got2, _ := cache.GetLatestPublishedTime(ctx, url2)

	if !got1.Equal(time1) {
		t.Errorf("url1: expected %v, got %v", time1, got1)
	}
	if !got2.Equal(time2) {
		t.Errorf("url2: expected %v, got %v", time2, got2)
	}
}

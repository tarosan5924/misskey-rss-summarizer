package rss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFeedRepository_Fetch_Success(t *testing.T) {
	rssXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Test Feed</title>
		<item>
			<title>Article 1</title>
			<link>https://example.com/article1</link>
			<description>Description 1</description>
			<guid>guid-1</guid>
			<pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
		</item>
		<item>
			<title>Article 2</title>
			<link>https://example.com/article2</link>
			<description>Description 2</description>
			<guid>guid-2</guid>
			<pubDate>Tue, 03 Jan 2006 15:04:05 MST</pubDate>
		</item>
	</channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rssXML))
	}))
	defer server.Close()

	repo := NewFeedRepository()
	ctx := context.Background()

	entries, err := repo.Fetch(ctx, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Title != "Article 1" {
		t.Errorf("expected title 'Article 1', got '%s'", entries[0].Title)
	}
	if entries[0].Link != "https://example.com/article1" {
		t.Errorf("expected link 'https://example.com/article1', got '%s'", entries[0].Link)
	}
	if entries[0].GUID != "guid-1" {
		t.Errorf("expected GUID 'guid-1', got '%s'", entries[0].GUID)
	}
}

func TestFeedRepository_Fetch_EmptyGUID(t *testing.T) {
	rssXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Test Feed</title>
		<item>
			<title>Article Without GUID</title>
			<link>https://example.com/article</link>
			<description>Description</description>
			<pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
		</item>
	</channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rssXML))
	}))
	defer server.Close()

	repo := NewFeedRepository()
	ctx := context.Background()

	entries, err := repo.Fetch(ctx, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].GUID != "https://example.com/article" {
		t.Errorf("expected GUID to fallback to link 'https://example.com/article', got '%s'", entries[0].GUID)
	}
}

func TestFeedRepository_Fetch_SkipNoPubDate(t *testing.T) {
	rssXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Test Feed</title>
		<item>
			<title>Article With Date</title>
			<link>https://example.com/article1</link>
			<guid>guid-1</guid>
			<pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
		</item>
		<item>
			<title>Article Without Date</title>
			<link>https://example.com/article2</link>
			<guid>guid-2</guid>
		</item>
	</channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rssXML))
	}))
	defer server.Close()

	repo := NewFeedRepository()
	ctx := context.Background()

	entries, err := repo.Fetch(ctx, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (items without pubDate should be skipped), got %d", len(entries))
	}

	if entries[0].Title != "Article With Date" {
		t.Errorf("expected 'Article With Date', got '%s'", entries[0].Title)
	}
}

func TestFeedRepository_Fetch_InvalidURL(t *testing.T) {
	repo := NewFeedRepository()
	ctx := context.Background()

	_, err := repo.Fetch(ctx, "http://invalid-url-that-does-not-exist-12345.com/feed")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestFeedRepository_Fetch_InvalidXML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid xml content"))
	}))
	defer server.Close()

	repo := NewFeedRepository()
	ctx := context.Background()

	_, err := repo.Fetch(ctx, server.URL)
	if err == nil {
		t.Error("expected error for invalid XML, got nil")
	}
}

func TestFeedRepository_Fetch_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<rss></rss>"))
	}))
	defer server.Close()

	repo := NewFeedRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.Fetch(ctx, server.URL)
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}


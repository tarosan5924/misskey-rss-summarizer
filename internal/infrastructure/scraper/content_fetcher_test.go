package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestContentFetcher_FetchContent_Success(t *testing.T) {
	// モックHTMLサーバーを作成
	html := `
	<!DOCTYPE html>
	<html>
	<head><title>Test Page</title></head>
	<body>
		<header>Header content</header>
		<nav>Navigation</nav>
		<article>
			<h1>Article Title</h1>
			<p>This is the main content of the article.</p>
			<p>It contains important information.</p>
		</article>
		<footer>Footer content</footer>
		<script>console.log('test');</script>
	</body>
	</html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	fetcher := NewContentFetcher(5 * time.Second)
	ctx := context.Background()

	content, err := fetcher.FetchContent(ctx, server.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if content == "" {
		t.Error("expected non-empty content")
	}

	// articleタグの内容が含まれているか確認
	if !strings.Contains(content, "main content") {
		t.Error("expected content to contain article text")
	}

	// script、nav、headerなどが除外されているか確認
	if strings.Contains(content, "console.log") {
		t.Error("expected script content to be removed")
	}
}

func TestContentFetcher_FetchContent_MainTag(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<body>
		<main>
			<h1>Main Content</h1>
			<p>This is in the main tag.</p>
		</main>
	</body>
	</html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	fetcher := NewContentFetcher(5 * time.Second)
	ctx := context.Background()

	content, err := fetcher.FetchContent(ctx, server.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(content, "main tag") {
		t.Error("expected content to contain main tag text")
	}
}

func TestContentFetcher_FetchContent_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := NewContentFetcher(5 * time.Second)
	ctx := context.Background()

	_, err := fetcher.FetchContent(ctx, server.URL)
	if err == nil {
		t.Error("expected error for 404 status, got nil")
	}

	if !strings.Contains(err.Error(), "HTTP status 404") {
		t.Errorf("expected HTTP status error, got: %v", err)
	}
}

func TestContentFetcher_FetchContent_InvalidURL(t *testing.T) {
	fetcher := NewContentFetcher(5 * time.Second)
	ctx := context.Background()

	_, err := fetcher.FetchContent(ctx, "http://invalid-url-that-does-not-exist-12345.com")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestContentFetcher_FetchContent_Timeout(t *testing.T) {
	// タイムアウトするサーバー
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := NewContentFetcher(100 * time.Millisecond)
	ctx := context.Background()

	_, err := fetcher.FetchContent(ctx, server.URL)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestContentFetcher_FetchContent_NoContent(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<body>
		<script>console.log('only script');</script>
	</body>
	</html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	fetcher := NewContentFetcher(5 * time.Second)
	ctx := context.Background()

	_, err := fetcher.FetchContent(ctx, server.URL)
	if err == nil {
		t.Error("expected 'no content found' error, got nil")
	}

	if !strings.Contains(err.Error(), "no content found") {
		t.Errorf("expected 'no content found' error, got: %v", err)
	}
}

func TestContentFetcher_FetchContent_WithContentClass(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<body>
		<div class="content">
			<h1>Content in .content class</h1>
			<p>This is the article content in a div with class 'content'.</p>
		</div>
	</body>
	</html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	fetcher := NewContentFetcher(5 * time.Second)
	ctx := context.Background()

	content, err := fetcher.FetchContent(ctx, server.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(content, "article content") {
		t.Error("expected content from .content class")
	}
}

func TestNewContentFetcher_DefaultTimeout(t *testing.T) {
	fetcher := NewContentFetcher(0)

	ws := fetcher.(*webScraper)
	expectedTimeout := 15 * time.Second

	if ws.client.Timeout != expectedTimeout {
		t.Errorf("expected default timeout %v, got %v", expectedTimeout, ws.client.Timeout)
	}
}

func TestNewContentFetcher_CustomTimeout(t *testing.T) {
	customTimeout := 30 * time.Second
	fetcher := NewContentFetcher(customTimeout)

	ws := fetcher.(*webScraper)

	if ws.client.Timeout != customTimeout {
		t.Errorf("expected timeout %v, got %v", customTimeout, ws.client.Timeout)
	}
}

func TestContentFetcher_UserAgent(t *testing.T) {
	var capturedUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><article>Content</article></body></html>"))
	}))
	defer server.Close()

	fetcher := NewContentFetcher(5 * time.Second)
	ctx := context.Background()

	_, _ = fetcher.FetchContent(ctx, server.URL)

	expectedUserAgent := "MisskeyRSSBot/1.0"
	if capturedUserAgent != expectedUserAgent {
		t.Errorf("expected User-Agent '%s', got '%s'", expectedUserAgent, capturedUserAgent)
	}
}

func TestExtractMainContent_MinimumLength(t *testing.T) {
	// 100文字未満のコンテンツでも取得される（フォールバックでbodyから取得）
	shortHTML := `
	<!DOCTYPE html>
	<html>
	<body>
		<article>Short</article>
	</body>
	</html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(shortHTML))
	}))
	defer server.Close()

	fetcher := NewContentFetcher(5 * time.Second)
	ctx := context.Background()

	// 短いコンテンツでもフォールバックで取得される
	content, err := fetcher.FetchContent(ctx, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(content, "Short") {
		t.Error("expected content to contain 'Short'")
	}
}

package html

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	maxHTMLBytes = int64(2 * 1024 * 1024)
	maxTextChars = 8000
)

func FetchArticleText(ctx context.Context, url string, timeout time.Duration) (string, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTMLBytes))
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to parse html: %w", err)
	}

	text := strings.TrimSpace(doc.Find("article").Text())
	if text == "" {
		text = strings.TrimSpace(doc.Find("main").Text())
	}
	if text == "" {
		text = strings.TrimSpace(doc.Text())
	}

	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return "", fmt.Errorf("empty article content")
	}

	if len([]rune(text)) > maxTextChars {
		text = string([]rune(text)[:maxTextChars])
	}

	return text, nil
}

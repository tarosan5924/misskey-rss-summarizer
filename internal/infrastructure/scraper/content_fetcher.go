package scraper

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ContentFetcher はWebページから本文を取得するインターフェース
type ContentFetcher interface {
	FetchContent(ctx context.Context, url string) (string, error)
}

type webScraper struct {
	client    *http.Client
	userAgent string
}

// NewContentFetcher は新しいContentFetcherを生成します
func NewContentFetcher(timeout time.Duration) ContentFetcher {
	if timeout == 0 {
		timeout = 15 * time.Second
	}

	return &webScraper{
		client:    &http.Client{Timeout: timeout},
		userAgent: "MisskeyRSSBot/1.0",
	}
}

// FetchContent はURLから記事本文を取得します
func (s *webScraper) FetchContent(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	content := extractMainContent(doc)

	if content == "" {
		return "", fmt.Errorf("no content found")
	}

	return content, nil
}

// extractMainContent はHTMLドキュメントから本文を抽出します
func extractMainContent(doc *goquery.Document) string {
	// 一般的な本文セレクタを試行
	selectors := []string{
		"article",
		"main",
		".post-content",
		".entry-content",
		".article-body",
		".article-content",
		"#content",
		".content",
	}

	for _, selector := range selectors {
		selection := doc.Find(selector)
		if selection.Length() > 0 {
			// 不要な要素を除去
			selection.Find("script, style, nav, header, footer, aside, .ad, .advertisement").Remove()
			text := selection.Text()
			cleaned := strings.TrimSpace(text)
			if cleaned != "" && len(cleaned) > 100 { // 最小文字数チェック
				return cleaned
			}
		}
	}

	// フォールバック: body全体から抽出
	doc.Find("script, style, nav, header, footer, aside").Remove()
	bodyText := doc.Find("body").Text()
	return strings.TrimSpace(bodyText)
}

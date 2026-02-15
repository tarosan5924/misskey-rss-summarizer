package html

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchArticleText(t *testing.T) {
	testCases := []struct {
		name      string
		body      string
		want      string
		wantError bool
	}{
		{
			name: "article tag",
			body: "<html><article>Hello <b>World</b></article></html>",
			want: "Hello World",
		},
		{
			name: "main tag fallback",
			body: "<html><main>Main <i>Content</i></main></html>",
			want: "Main Content",
		},
		{
			name: "document text fallback",
			body: "<html><body>Doc <span>Text</span></body></html>",
			want: "Doc Text",
		},
		{
			name: "normalize spaces",
			body: "<html><article>  Hello\n\t  World   </article></html>",
			want: "Hello World",
		},
		{
			name: "too long text is trimmed",
			// 8000文字目にマルチバイト文字が入るケース
			body: "<html><article>" + strings.Repeat("a", 7999) + "あ" + "</article></html>",
			want: strings.Repeat("a", 7999) + "あ",
		},
		{
			name:      "http status error",
			body:      "<html><article>ignored</article></html>",
			wantError: true,
		},
		{
			name:      "empty content",
			body:      "<html><body></body></html>",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.name == "http status error" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			ctx := context.Background()
			got, err := FetchArticleText(ctx, server.URL, 5*time.Second)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.TrimSpace(got) != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

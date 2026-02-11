package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

func TestBedrockSummarizerBuildConverseInput(t *testing.T) {
	customTimeout := 12 * time.Second
	s := &bedrockSummarizer{
		modelID:      "test-model",
		maxTokens:    256,
		systemPrompt: "system prompt",
		timeout:      customTimeout,
	}

	input := s.buildConverseInput("hello")
	if input.ModelId == nil || *input.ModelId != "test-model" {
		t.Fatalf("expected model ID to be set, got %v", input.ModelId)
	}

	if len(input.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(input.Messages))
	}

	msg := input.Messages[0]
	if msg.Role != types.ConversationRoleUser {
		t.Fatalf("expected user role, got %s", msg.Role)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(msg.Content))
	}

	textBlock, ok := msg.Content[0].(*types.ContentBlockMemberText)
	if !ok {
		t.Fatalf("expected text content block, got %T", msg.Content[0])
	}
	if textBlock.Value != "hello" {
		t.Fatalf("expected prompt text, got %q", textBlock.Value)
	}

	if len(input.System) != 1 {
		t.Fatalf("expected 1 system block, got %d", len(input.System))
	}
	systemBlock, ok := input.System[0].(*types.SystemContentBlockMemberText)
	if !ok {
		t.Fatalf("expected system text block, got %T", input.System[0])
	}
	if systemBlock.Value != "system prompt" {
		t.Fatalf("expected system prompt, got %q", systemBlock.Value)
	}

	if input.InferenceConfig == nil || input.InferenceConfig.MaxTokens == nil {
		t.Fatalf("expected inference config max tokens to be set")
	}
	if *input.InferenceConfig.MaxTokens != 256 {
		t.Fatalf("expected max tokens 256, got %d", *input.InferenceConfig.MaxTokens)
	}
	if input.InferenceConfig.Temperature == nil || *input.InferenceConfig.Temperature != float32(0.3) {
		t.Fatalf("expected temperature 0.3, got %v", input.InferenceConfig.Temperature)
	}
	if input.InferenceConfig.TopP == nil || *input.InferenceConfig.TopP != float32(0.9) {
		t.Fatalf("expected topP 0.9, got %v", input.InferenceConfig.TopP)
	}

	if s.timeout != customTimeout {
		t.Fatalf("expected timeout %v, got %v", customTimeout, s.timeout)
	}
}

func TestBedrockSummarizerParseResponse(t *testing.T) {
	testCases := []struct {
		name      string
		resp      *bedrockruntime.ConverseOutput
		want      string
		wantError bool
	}{
		{
			name: "success",
			resp: &bedrockruntime.ConverseOutput{
				Output: &types.ConverseOutputMemberMessage{Value: types.Message{
					Role: types.ConversationRoleAssistant,
					Content: []types.ContentBlock{
						&types.ContentBlockMemberText{Value: " first "},
						&types.ContentBlockMemberText{Value: "second"},
					},
				}},
			},
			want: "first second",
		},
		{
			name:      "no output",
			resp:      &bedrockruntime.ConverseOutput{},
			wantError: true,
		},
		{
			name: "empty content",
			resp: &bedrockruntime.ConverseOutput{
				Output: &types.ConverseOutputMemberMessage{Value: types.Message{
					Role:    types.ConversationRoleAssistant,
					Content: []types.ContentBlock{},
				}},
			},
			wantError: true,
		},
		{
			name: "empty summary",
			resp: &bedrockruntime.ConverseOutput{
				Output: &types.ConverseOutputMemberMessage{Value: types.Message{
					Role: types.ConversationRoleAssistant,
					Content: []types.ContentBlock{
						&types.ContentBlockMemberText{Value: "   "},
					},
				}},
			},
			wantError: true,
		},
	}

	s := &bedrockSummarizer{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.parseResponse(tc.resp)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

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
			name:      "empty content",
			body:      "<html><body></body></html>",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			ctx := context.Background()
			got, err := fetchArticleText(ctx, server.URL, 5*time.Second)
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

func TestNewBedrockSummarizerRequiresRegion(t *testing.T) {
	cfg := Config{
		Provider: "bedrock",
		APIKey:   "test-token",
		Model:    "test-model",
	}

	_, err := newBedrockSummarizer(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected error when region is empty, got nil")
	}
}

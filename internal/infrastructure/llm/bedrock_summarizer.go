package llm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/smithy-go/auth/bearer"

	"misskeyRSSbot/internal/domain/repository"
)

type bedrockSummarizer struct {
	client       *bedrockruntime.Client
	modelID      string
	maxTokens    int32
	systemPrompt string
	timeout      time.Duration
}

const (
	bedrockDefaultMaxTokens = int32(512)
	bedrockMaxHTMLBytes     = int64(2 * 1024 * 1024)
	bedrockMaxTextChars     = 8000
)

func newBedrockSummarizer(ctx context.Context, cfg Config) (repository.SummarizerRepository, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("bedrock model ID is required")
	}
	bearerToken := cfg.APIKey
	if bearerToken == "" {
		return nil, fmt.Errorf("bedrock bearer token is required (set LLM_API_KEY)")
	}

	region := cfg.Region
	if region == "" {
		return nil, fmt.Errorf("bedrock region is required (set LLM_REGION)")
	}

	systemInstruction := cfg.SystemInstruction
	if systemInstruction == "" {
		systemInstruction = DefaultSystemInstruction
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	maxTokens := bedrockDefaultMaxTokens
	if cfg.MaxTokens > 0 {
		maxTokens = int32(cfg.MaxTokens)
	}

	sdkConfig, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	sdkConfig.BearerAuthTokenProvider = bearer.NewTokenCache(bearer.StaticTokenProvider{
		Token: bearer.Token{Value: bearerToken},
	})
	sdkConfig.AuthSchemePreference = []string{"httpBearerAuth"}

	client := bedrockruntime.NewFromConfig(sdkConfig)

	return &bedrockSummarizer{
		client:       client,
		modelID:      cfg.Model,
		maxTokens:    maxTokens,
		systemPrompt: systemInstruction,
		timeout:      timeout,
	}, nil
}

func (s *bedrockSummarizer) Summarize(ctx context.Context, url, title string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	articleText, err := fetchArticleText(ctx, url, s.timeout)
	if err != nil {
		return "", fmt.Errorf("failed to fetch article text: %w", err)
	}

	prompt := fmt.Sprintf("記事タイトル: %s\n記事URL: %s\n\n記事本文:\n%s", title, url, articleText)
	input := s.buildConverseInput(prompt)
	resp, err := s.client.Converse(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to invoke bedrock model: %w", err)
	}

	summary, err := s.parseResponse(resp)
	if err != nil {
		return "", err
	}

	return summary, nil
}

func (s *bedrockSummarizer) IsEnabled() bool {
	return true
}

func (s *bedrockSummarizer) buildConverseInput(prompt string) *bedrockruntime.ConverseInput {
	temperature := float32(0.3)
	topP := float32(0.9)

	return &bedrockruntime.ConverseInput{
		ModelId: aws.String(s.modelID),
		Messages: []types.Message{
			{
				Role: types.ConversationRoleUser,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: prompt},
				},
			},
		},
		System: []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{Value: s.systemPrompt},
		},
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens:   aws.Int32(s.maxTokens),
			Temperature: aws.Float32(temperature),
			TopP:        aws.Float32(topP),
		},
	}
}

func (s *bedrockSummarizer) parseResponse(resp *bedrockruntime.ConverseOutput) (string, error) {
	messageOutput, ok := resp.Output.(*types.ConverseOutputMemberMessage)
	if !ok {
		return "", fmt.Errorf("unexpected bedrock response output type: %T", resp.Output)
	}

	if len(messageOutput.Value.Content) == 0 {
		return "", fmt.Errorf("no content in bedrock response")
	}

	var builder strings.Builder
	for _, block := range messageOutput.Value.Content {
		textBlock, ok := block.(*types.ContentBlockMemberText)
		if !ok {
			continue
		}
		text := strings.TrimSpace(textBlock.Value)
		if text == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(text)
	}

	summary := strings.TrimSpace(builder.String())
	if summary == "" {
		return "", fmt.Errorf("empty summary in bedrock response")
	}
	return summary, nil
}

func fetchArticleText(ctx context.Context, url string, timeout time.Duration) (string, error) {
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, bedrockMaxHTMLBytes))
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

	if len(text) > bedrockMaxTextChars {
		text = text[:bedrockMaxTextChars]
	}

	return text, nil
}

package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

const (
	defaultBaseURL   = "https://api.anthropic.com"
	messagesEndpoint = "/v1/messages"
	apiVersion       = "2023-06-01"
	defaultModel     = "claude-sonnet-4-20250514"
	maxRetries       = 3
	initialBackoff   = 1 * time.Second
	maxTokens        = 1024
)

// StoryResult holds the output of a story generation request.
type StoryResult struct {
	Text       string                `json:"text"`
	LayerType  domain.StoryLayerType `json:"layer_type"`
	Confidence int                   `json:"confidence"`
	TokensIn   int                   `json:"tokens_in"`
	TokensOut  int                   `json:"tokens_out"`
	Duration   time.Duration         `json:"duration"`
}

// Client communicates with the Anthropic Claude API to generate stories.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// Config holds settings for the Claude API client.
type Config struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// NewClient creates a new Claude API client.
func NewClient(cfg *Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}

	httpClient := configuredHTTPClient(cfg.HTTPClient, cfg.Timeout, 60*time.Second)

	return &Client{
		apiKey:     cfg.APIKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		httpClient: httpClient,
	}
}

func configuredHTTPClient(base *http.Client, timeout, defaultTimeout time.Duration) *http.Client {
	overrideTimeout := timeout > 0
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if base == nil {
		return &http.Client{Timeout: timeout}
	}
	clientCopy := *base
	if clientCopy.Timeout == 0 || overrideTimeout {
		clientCopy.Timeout = timeout
	}
	return &clientCopy
}

// messagesRequest is the request body for the Claude Messages API.
type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// messagesResponse is the response from the Claude Messages API.
type messagesResponse struct {
	Content []contentBlock `json:"content"`
	Usage   usage          `json:"usage"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// storyJSON is the expected JSON output from Claude.
type storyJSON struct {
	Text       string `json:"text"`
	LayerType  string `json:"layer_type"`
	Confidence int    `json:"confidence"`
}

// GenerateStory generates a story for the given POI in the specified language.
func (c *Client) GenerateStory(ctx context.Context, poi *domain.POI, language string) (*StoryResult, error) {
	start := time.Now()

	systemPrompt := buildSystemPrompt(language)
	userPrompt := buildUserPrompt(poi, language)

	req := messagesRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages: []message{
			{Role: "user", Content: userPrompt},
		},
	}

	resp, err := c.sendWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("claude: generate story for POI %d: %w", poi.ID, err)
	}

	text := extractText(resp)
	if text == "" {
		return nil, fmt.Errorf("claude: empty response for POI %d", poi.ID)
	}

	parsed, err := parseStoryResponse(text)
	if err != nil {
		return nil, fmt.Errorf("claude: parse response for POI %d: %w", poi.ID, err)
	}

	return &StoryResult{
		Text:       parsed.Text,
		LayerType:  domain.StoryLayerType(parsed.LayerType),
		Confidence: parsed.Confidence,
		TokensIn:   resp.Usage.InputTokens,
		TokensOut:  resp.Usage.OutputTokens,
		Duration:   time.Since(start),
	}, nil
}

// sendWithRetry sends a request with exponential backoff retry on 429/5xx errors.
func (c *Client) sendWithRetry(ctx context.Context, req messagesRequest) (*messagesResponse, error) {
	var lastErr error
	backoff := initialBackoff

	for attempt := range maxRetries {
		resp, err := c.send(ctx, req)
		if err == nil {
			return resp, nil
		}

		var apiErr *APIError
		isRetryable := false
		if errors.As(err, &apiErr) {
			isRetryable = apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500
		}

		if !isRetryable || attempt == maxRetries-1 {
			return nil, err
		}

		lastErr = err
		log.Printf("Claude API attempt %d/%d failed (status %d), retrying in %s...",
			attempt+1, maxRetries, apiErr.StatusCode, backoff)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
	}

	return nil, lastErr
}

// send performs a single API request.
func (c *Client) send(ctx context.Context, reqBody messagesRequest) (*messagesResponse, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+messagesEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // Close error on read-only response body is safe to ignore

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	var result messagesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func extractText(resp *messagesResponse) string {
	for _, block := range resp.Content {
		if block.Type == "text" {
			return block.Text
		}
	}
	return ""
}

func parseStoryResponse(text string) (*storyJSON, error) {
	// Try to extract JSON from markdown code blocks first
	cleaned := text
	if idx := strings.Index(text, "```json"); idx != -1 {
		start := idx + len("```json")
		end := strings.Index(text[start:], "```")
		if end != -1 {
			cleaned = strings.TrimSpace(text[start : start+end])
		}
	} else if idx := strings.Index(text, "```"); idx != -1 {
		start := idx + len("```")
		end := strings.Index(text[start:], "```")
		if end != -1 {
			cleaned = strings.TrimSpace(text[start : start+end])
		}
	}

	// Try direct JSON parse
	var result storyJSON
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		// Try finding JSON object in text
		start := strings.Index(text, "{")
		end := strings.LastIndex(text, "}")
		if start == -1 || end == -1 || end <= start {
			return nil, fmt.Errorf("no JSON object found in response")
		}
		if err2 := json.Unmarshal([]byte(text[start:end+1]), &result); err2 != nil {
			return nil, fmt.Errorf("parse JSON: %w (raw: %s)", err2, truncate(text, 200))
		}
	}

	if result.Text == "" {
		return nil, fmt.Errorf("empty text in parsed response")
	}

	// Validate layer_type
	if !isValidLayerType(result.LayerType) {
		result.LayerType = string(domain.StoryLayerGeneral)
	}

	// Clamp confidence to [0, 100]
	if result.Confidence < 0 {
		result.Confidence = 0
	}
	if result.Confidence > 100 {
		result.Confidence = 100
	}

	return &result, nil
}

func isValidLayerType(lt string) bool {
	switch domain.StoryLayerType(lt) {
	case domain.StoryLayerAtmosphere,
		domain.StoryLayerHumanStory,
		domain.StoryLayerHiddenDetail,
		domain.StoryLayerTimeShift,
		domain.StoryLayerGeneral:
		return true
	}
	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// APIError represents an error response from the Claude API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("claude API error (status %d): %s", e.StatusCode, truncate(e.Body, 200))
}

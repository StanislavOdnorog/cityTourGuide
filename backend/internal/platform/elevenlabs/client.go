package elevenlabs

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
)

const (
	defaultBaseURL = "https://api.elevenlabs.io"
	ttsEndpoint    = "/v1/text-to-speech/"
	defaultModelID = "eleven_multilingual_v2"
	// Default voice IDs — can be overridden via Config.
	defaultVoiceEN = "21m00Tcm4TlvDq8ikWAM" // Rachel (clear, narrative)
	defaultVoiceRU = "21m00Tcm4TlvDq8ikWAM" // Same voice, multilingual model handles RU
	maxRetries     = 3
	initialBackoff = 1 * time.Second
)

// AudioResult holds the output of a TTS generation request.
type AudioResult struct {
	Audio    io.Reader     `json:"-"`
	Duration time.Duration `json:"duration"`
}

// Client communicates with the ElevenLabs TTS API.
type Client struct {
	apiKey     string
	baseURL    string
	modelID    string
	voiceEN    string
	voiceRU    string
	stability  float64
	similarity float64
	style      float64
	httpClient *http.Client
}

// Config holds settings for the ElevenLabs API client.
type Config struct {
	APIKey     string
	BaseURL    string
	ModelID    string
	VoiceEN    string
	VoiceRU    string
	Stability  float64
	Similarity float64
	Style      float64
}

// NewClient creates a new ElevenLabs TTS client.
func NewClient(cfg *Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	modelID := cfg.ModelID
	if modelID == "" {
		modelID = defaultModelID
	}
	voiceEN := cfg.VoiceEN
	if voiceEN == "" {
		voiceEN = defaultVoiceEN
	}
	voiceRU := cfg.VoiceRU
	if voiceRU == "" {
		voiceRU = defaultVoiceRU
	}
	stability := cfg.Stability
	if stability == 0 {
		stability = 0.5
	}
	similarity := cfg.Similarity
	if similarity == 0 {
		similarity = 0.75
	}
	style := cfg.Style
	if style == 0 {
		style = 0.3
	}

	return &Client{
		apiKey:     cfg.APIKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		modelID:    modelID,
		voiceEN:    voiceEN,
		voiceRU:    voiceRU,
		stability:  stability,
		similarity: similarity,
		style:      style,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// ttsRequest is the request body for the ElevenLabs TTS API.
type ttsRequest struct {
	Text          string        `json:"text"`
	ModelID       string        `json:"model_id"`
	VoiceSettings voiceSettings `json:"voice_settings"`
}

type voiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
	Style           float64 `json:"style"`
}

// GenerateAudio converts text to speech and returns the MP3 audio as an io.Reader.
// Language determines voice selection: "ru" uses the Russian voice, everything else uses English.
func (c *Client) GenerateAudio(ctx context.Context, text, language string) (*AudioResult, error) {
	if text == "" {
		return nil, fmt.Errorf("elevenlabs: text is required")
	}

	start := time.Now()

	voiceID := c.voiceEN
	if language == "ru" {
		voiceID = c.voiceRU
	}

	reqBody := ttsRequest{
		Text:    text,
		ModelID: c.modelID,
		VoiceSettings: voiceSettings{
			Stability:       c.stability,
			SimilarityBoost: c.similarity,
			Style:           c.style,
		},
	}

	audio, err := c.sendWithRetry(ctx, voiceID, reqBody)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs: generate audio: %w", err)
	}

	return &AudioResult{
		Audio:    audio,
		Duration: time.Since(start),
	}, nil
}

// sendWithRetry sends a request with exponential backoff retry on 429/5xx errors.
func (c *Client) sendWithRetry(ctx context.Context, voiceID string, req ttsRequest) (io.Reader, error) {
	var lastErr error
	backoff := initialBackoff

	for attempt := range maxRetries {
		audio, err := c.send(ctx, voiceID, req)
		if err == nil {
			return audio, nil
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
		log.Printf("ElevenLabs API attempt %d/%d failed (status %d), retrying in %s...",
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

// send performs a single TTS API request and returns the MP3 audio data.
func (c *Client) send(ctx context.Context, voiceID string, reqBody ttsRequest) (io.Reader, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + ttsEndpoint + voiceID
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close() //nolint:errcheck // Close error on read-only response body is safe to ignore
		respBody, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	// Read the full MP3 body into memory so the caller does not need to manage the response.
	audioData, err := io.ReadAll(resp.Body)
	resp.Body.Close() //nolint:errcheck // Close error on read-only response body is safe to ignore
	if err != nil {
		return nil, fmt.Errorf("read audio response: %w", err)
	}

	if len(audioData) == 0 {
		return nil, fmt.Errorf("empty audio response")
	}

	return bytes.NewReader(audioData), nil
}

// APIError represents an error response from the ElevenLabs API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("elevenlabs API error (status %d): %s", e.StatusCode, truncate(e.Body, 200))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

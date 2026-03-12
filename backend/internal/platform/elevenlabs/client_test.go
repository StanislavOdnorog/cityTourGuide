package elevenlabs

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func makeAudioHandler(audioData []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("xi-api-key") == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "bad content type", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Accept") != "audio/mpeg" {
			http.Error(w, "bad accept header", http.StatusBadRequest)
			return
		}

		var req ttsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(audioData) //nolint:errcheck // test helper
	}
}

func fakeMP3() []byte {
	// Minimal fake MP3 data (ID3 header + some bytes)
	return []byte{0xFF, 0xFB, 0x90, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
}

func TestGenerateAudio_Success_EN(t *testing.T) {
	audio := fakeMP3()
	server := httptest.NewServer(makeAudioHandler(audio))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	result, err := client.GenerateAudio(context.Background(), "Hello, this is a test story.", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := io.ReadAll(result.Audio)
	if err != nil {
		t.Fatalf("failed to read audio: %v", err)
	}

	if len(data) != len(audio) {
		t.Errorf("audio length = %d, want %d", len(data), len(audio))
	}
	if result.Duration <= 0 {
		t.Error("duration should be positive")
	}
}

func TestGenerateAudio_Success_RU(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeMP3()) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		VoiceRU: "ru-voice-id",
	})

	result, err := client.GenerateAudio(context.Background(), "Привет, это тестовая история.", "ru")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := io.ReadAll(result.Audio)
	if err != nil {
		t.Fatalf("failed to read audio: %v", err)
	}

	if len(data) == 0 {
		t.Error("audio should not be empty")
	}

	expectedPath := ttsEndpoint + "ru-voice-id"
	if receivedPath != expectedPath {
		t.Errorf("path = %q, want %q", receivedPath, expectedPath)
	}
}

func TestGenerateAudio_EmptyText(t *testing.T) {
	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: "http://localhost:1",
	})

	_, err := client.GenerateAudio(context.Background(), "", "en")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	if err.Error() != "elevenlabs: text is required" {
		t.Errorf("error = %q, want 'elevenlabs: text is required'", err.Error())
	}
}

func TestGenerateAudio_VoiceSelection_EN(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeMP3()) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		VoiceEN: "en-voice-id",
		VoiceRU: "ru-voice-id",
	})

	_, err := client.GenerateAudio(context.Background(), "English test.", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPath := ttsEndpoint + "en-voice-id"
	if receivedPath != expectedPath {
		t.Errorf("EN path = %q, want %q", receivedPath, expectedPath)
	}
}

func TestGenerateAudio_VoiceSelection_RU(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeMP3()) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		VoiceEN: "en-voice-id",
		VoiceRU: "ru-voice-id",
	})

	_, err := client.GenerateAudio(context.Background(), "Русский тест.", "ru")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPath := ttsEndpoint + "ru-voice-id"
	if receivedPath != expectedPath {
		t.Errorf("RU path = %q, want %q", receivedPath, expectedPath)
	}
}

func TestGenerateAudio_Headers(t *testing.T) {
	var (
		gotAPIKey      string
		gotContentType string
		gotAccept      string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("xi-api-key")
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeMP3()) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "my-secret-key",
		BaseURL: server.URL,
	})

	_, err := client.GenerateAudio(context.Background(), "Test text.", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAPIKey != "my-secret-key" {
		t.Errorf("xi-api-key = %q, want %q", gotAPIKey, "my-secret-key")
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", gotContentType, "application/json")
	}
	if gotAccept != "audio/mpeg" {
		t.Errorf("Accept = %q, want %q", gotAccept, "audio/mpeg")
	}
}

func TestGenerateAudio_RequestBody(t *testing.T) {
	var received ttsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received) //nolint:errcheck // test helper
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeMP3()) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:     "test-key",
		BaseURL:    server.URL,
		Stability:  0.6,
		Similarity: 0.8,
		Style:      0.4,
	})

	_, err := client.GenerateAudio(context.Background(), "Story text here.", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.Text != "Story text here." {
		t.Errorf("text = %q, want %q", received.Text, "Story text here.")
	}
	if received.ModelID != defaultModelID {
		t.Errorf("model_id = %q, want %q", received.ModelID, defaultModelID)
	}
	if received.VoiceSettings.Stability != 0.6 {
		t.Errorf("stability = %f, want %f", received.VoiceSettings.Stability, 0.6)
	}
	if received.VoiceSettings.SimilarityBoost != 0.8 {
		t.Errorf("similarity_boost = %f, want %f", received.VoiceSettings.SimilarityBoost, 0.8)
	}
	if received.VoiceSettings.Style != 0.4 {
		t.Errorf("style = %f, want %f", received.VoiceSettings.Style, 0.4)
	}
}

func TestGenerateAudio_DefaultConfig(t *testing.T) {
	client := NewClient(&Config{
		APIKey: "test-key",
	})

	if client.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", client.baseURL, defaultBaseURL)
	}
	if client.modelID != defaultModelID {
		t.Errorf("modelID = %q, want %q", client.modelID, defaultModelID)
	}
	if client.voiceEN != defaultVoiceEN {
		t.Errorf("voiceEN = %q, want %q", client.voiceEN, defaultVoiceEN)
	}
	if client.voiceRU != defaultVoiceRU {
		t.Errorf("voiceRU = %q, want %q", client.voiceRU, defaultVoiceRU)
	}
	if client.stability != 0.5 {
		t.Errorf("stability = %f, want %f", client.stability, 0.5)
	}
	if client.similarity != 0.75 {
		t.Errorf("similarity = %f, want %f", client.similarity, 0.75)
	}
	if client.style != 0.3 {
		t.Errorf("style = %f, want %f", client.style, 0.3)
	}
	if client.httpClient == nil {
		t.Fatal("expected http client to be initialized")
	}
	if client.httpClient.Timeout != 120*time.Second {
		t.Errorf("http client timeout = %s, want 120s", client.httpClient.Timeout)
	}
}

func TestGenerateAudio_CustomConfig(t *testing.T) {
	client := NewClient(&Config{
		APIKey:     "key",
		BaseURL:    "https://custom.api.com",
		ModelID:    "custom_model",
		VoiceEN:    "custom-en",
		VoiceRU:    "custom-ru",
		Stability:  0.9,
		Similarity: 0.1,
		Style:      0.7,
		Timeout:    5 * time.Second,
	})

	if client.baseURL != "https://custom.api.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://custom.api.com")
	}
	if client.modelID != "custom_model" {
		t.Errorf("modelID = %q, want %q", client.modelID, "custom_model")
	}
	if client.voiceEN != "custom-en" {
		t.Errorf("voiceEN = %q, want %q", client.voiceEN, "custom-en")
	}
	if client.voiceRU != "custom-ru" {
		t.Errorf("voiceRU = %q, want %q", client.voiceRU, "custom-ru")
	}
	if client.stability != 0.9 {
		t.Errorf("stability = %f, want %f", client.stability, 0.9)
	}
	if client.similarity != 0.1 {
		t.Errorf("similarity = %f, want %f", client.similarity, 0.1)
	}
	if client.style != 0.7 {
		t.Errorf("style = %f, want %f", client.style, 0.7)
	}
	if client.httpClient.Timeout != 5*time.Second {
		t.Errorf("http client timeout = %s, want 5s", client.httpClient.Timeout)
	}
}

func TestGenerateAudio_CustomHTTPClient(t *testing.T) {
	baseClient := &http.Client{Timeout: 3 * time.Second}
	client := NewClient(&Config{
		APIKey:     "key",
		HTTPClient: baseClient,
	})

	if client.httpClient == baseClient {
		t.Fatal("expected client to copy injected http client")
	}
	if client.httpClient.Timeout != 3*time.Second {
		t.Errorf("http client timeout = %s, want 3s", client.httpClient.Timeout)
	}

	override := NewClient(&Config{
		APIKey:     "key",
		HTTPClient: baseClient,
		Timeout:    7 * time.Second,
	})
	if override.httpClient.Timeout != 7*time.Second {
		t.Errorf("http client timeout = %s, want 7s", override.httpClient.Timeout)
	}
}

func TestGenerateAudio_Retry429_ThenSuccess(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limited"}`)) //nolint:errcheck // test helper
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeMP3()) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	result, err := client.GenerateAudio(context.Background(), "Retry test.", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := io.ReadAll(result.Audio)
	if err != nil {
		t.Fatalf("failed to read audio: %v", err)
	}
	if len(data) == 0 {
		t.Error("audio should not be empty after retry success")
	}
	if attempts.Load() != 3 {
		t.Errorf("attempts = %d, want 3", attempts.Load())
	}
}

func TestGenerateAudio_Retry500_ThenSuccess(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal error"}`)) //nolint:errcheck // test helper
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeMP3()) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	result, err := client.GenerateAudio(context.Background(), "Server error retry.", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := io.ReadAll(result.Audio)
	if err != nil {
		t.Fatalf("failed to read audio: %v", err)
	}
	if len(data) == 0 {
		t.Error("audio should not be empty")
	}
	if attempts.Load() != 2 {
		t.Errorf("attempts = %d, want 2", attempts.Load())
	}
}

func TestGenerateAudio_NoRetryOn400(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`)) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	_, err := client.GenerateAudio(context.Background(), "Bad request test.", "en")
	if err == nil {
		t.Fatal("expected error for 400")
	}
	if attempts.Load() != 1 {
		t.Errorf("attempts = %d, want 1 (no retry on 400)", attempts.Load())
	}
}

func TestGenerateAudio_MaxRetriesExhausted(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`)) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	_, err := client.GenerateAudio(context.Background(), "Max retries test.", "en")
	if err == nil {
		t.Fatal("expected error when max retries exhausted")
	}
	if attempts.Load() != int32(maxRetries) {
		t.Errorf("attempts = %d, want %d", attempts.Load(), maxRetries)
	}
}

func TestGenerateAudio_ContextCanceled(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		// Simulate a slow response; the client's context cancellation will abort the request.
		time.Sleep(5 * time.Second)
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeMP3()) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.GenerateAudio(ctx, "Canceled context test.", "en")
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
	<-started // ensure server handler was called
}

func TestGenerateAudio_ClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeMP3()) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Timeout: 50 * time.Millisecond,
	})

	_, err := client.GenerateAudio(context.Background(), "Slow response.", "en")
	if err == nil {
		t.Fatal("expected timeout error")
	}

	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestGenerateAudio_InjectedHTTPClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeMP3()) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:     "test-key",
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 50 * time.Millisecond},
	})

	_, err := client.GenerateAudio(context.Background(), "Slow response.", "en")
	if err == nil {
		t.Fatal("expected timeout error")
	}

	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestGenerateAudio_RetryStopsOnContextCancellation(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`)) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.GenerateAudio(ctx, "Retry cancel.", "en")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
	if attempts.Load() != 1 {
		t.Errorf("attempts = %d, want 1 after context cancellation", attempts.Load())
	}
}

func TestGenerateAudio_EmptyAudioResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		// Write nothing — empty body
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	_, err := client.GenerateAudio(context.Background(), "Empty response test.", "en")
	if err == nil {
		t.Fatal("expected error for empty audio response")
	}
}

func TestAPIError_Format(t *testing.T) {
	err := &APIError{
		StatusCode: 429,
		Body:       `{"error":"rate limited"}`,
	}

	want := `elevenlabs API error (status 429): {"error":"rate limited"}`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAPIError_Truncation(t *testing.T) {
	longBody := ""
	for range 300 {
		longBody += "x"
	}

	err := &APIError{
		StatusCode: 500,
		Body:       longBody,
	}

	msg := err.Error()
	if len(msg) > 300 {
		t.Errorf("error message too long: %d chars", len(msg))
	}
}

func TestGenerateAudio_ModelInRequest(t *testing.T) {
	var received ttsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received) //nolint:errcheck // test helper
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeMP3()) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		ModelID: "custom_model_v3",
	})

	_, err := client.GenerateAudio(context.Background(), "Model test.", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.ModelID != "custom_model_v3" {
		t.Errorf("model_id = %q, want %q", received.ModelID, "custom_model_v3")
	}
}

func TestGenerateAudio_BaseURL_TrailingSlash(t *testing.T) {
	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: "https://api.example.com/",
	})

	if client.baseURL != "https://api.example.com" {
		t.Errorf("baseURL = %q, want trailing slash removed", client.baseURL)
	}
}

func TestGenerateAudio_NonENDefaultsToENVoice(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeMP3()) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		VoiceEN: "en-voice",
		VoiceRU: "ru-voice",
	})

	// "fr" is not "ru", so it should use EN voice
	_, err := client.GenerateAudio(context.Background(), "Bonjour.", "fr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPath := ttsEndpoint + "en-voice"
	if receivedPath != expectedPath {
		t.Errorf("path = %q, want %q (non-RU should use EN voice)", receivedPath, expectedPath)
	}
}

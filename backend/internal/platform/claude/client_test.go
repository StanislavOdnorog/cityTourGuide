package claude

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

func testPOI() *domain.POI {
	nameRu := "Нарикала"
	addr := "Narikala Fortress Road"
	return &domain.POI{
		ID:            1,
		CityID:        1,
		Name:          "Narikala Fortress",
		NameRu:        &nameRu,
		Lat:           41.6875,
		Lng:           44.8089,
		Type:          domain.POITypeMonument,
		Tags:          json.RawMessage(`{"wikidata":"Q474028","wikipedia":"en:Narikala"}`),
		Address:       &addr,
		InterestScore: 80,
		Status:        domain.POIStatusActive,
	}
}

func makeSuccessHandler(storyText, layerType string, confidence int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("anthropic-version") != apiVersion {
			http.Error(w, "bad version", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "bad content type", http.StatusBadRequest)
			return
		}

		var req messagesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}

		storyObj := map[string]interface{}{
			"text":       storyText,
			"layer_type": layerType,
			"confidence": confidence,
		}
		storyBytes, _ := json.Marshal(storyObj) //nolint:errcheck // test helper

		resp := messagesResponse{
			Content: []contentBlock{
				{Type: "text", Text: string(storyBytes)},
			},
			Usage: usage{InputTokens: 500, OutputTokens: 150},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck // test helper
	}
}

func TestGenerateStory_Success(t *testing.T) {
	server := httptest.NewServer(makeSuccessHandler(
		"The ancient fortress of Narikala watches over Tbilisi like a weathered guardian.",
		"atmosphere",
		85,
	))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
	result, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Text == "" {
		t.Error("expected non-empty text")
	}
	if result.LayerType != domain.StoryLayerAtmosphere {
		t.Errorf("expected layer_type=atmosphere, got %s", result.LayerType)
	}
	if result.Confidence != 85 {
		t.Errorf("expected confidence=85, got %d", result.Confidence)
	}
	if result.TokensIn != 500 {
		t.Errorf("expected tokens_in=500, got %d", result.TokensIn)
	}
	if result.TokensOut != 150 {
		t.Errorf("expected tokens_out=150, got %d", result.TokensOut)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestGenerateStory_RussianLanguage(t *testing.T) {
	var receivedSystem string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req messagesRequest
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck // test helper
		receivedSystem = req.System

		text := `{"text":"Крепость Нарикала возвышается над Тбилиси.","layer_type":"time_shift","confidence":75}`
		resp := messagesResponse{
			Content: []contentBlock{{Type: "text", Text: text}},
			Usage:   usage{InputTokens: 600, OutputTokens: 200},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
	result, err := client.GenerateStory(context.Background(), testPOI(), "ru")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Text != "Крепость Нарикала возвышается над Тбилиси." {
		t.Errorf("unexpected text: %s", result.Text)
	}
	if result.LayerType != domain.StoryLayerTimeShift {
		t.Errorf("expected time_shift, got %s", result.LayerType)
	}
	if receivedSystem == "" {
		t.Error("system prompt not received")
	}
}

func TestGenerateStory_MarkdownCodeBlock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		text := "```json\n{\"text\":\"A hidden story.\",\"layer_type\":\"hidden_detail\",\"confidence\":70}\n```"
		resp := messagesResponse{
			Content: []contentBlock{{Type: "text", Text: text}},
			Usage:   usage{InputTokens: 400, OutputTokens: 100},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
	result, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Text != "A hidden story." {
		t.Errorf("expected parsed text, got: %s", result.Text)
	}
	if result.LayerType != domain.StoryLayerHiddenDetail {
		t.Errorf("expected hidden_detail, got %s", result.LayerType)
	}
}

func TestGenerateStory_InvalidLayerType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		text := `{"text":"Some story text.","layer_type":"unknown_type","confidence":60}`
		resp := messagesResponse{
			Content: []contentBlock{{Type: "text", Text: text}},
			Usage:   usage{InputTokens: 300, OutputTokens: 80},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
	result, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.LayerType != domain.StoryLayerGeneral {
		t.Errorf("expected general (fallback), got %s", result.LayerType)
	}
}

func TestGenerateStory_ConfidenceClamping(t *testing.T) {
	tests := []struct {
		name       string
		confidence int
		expected   int
	}{
		{"negative", -10, 0},
		{"over 100", 150, 100},
		{"normal", 75, 75},
		{"zero", 0, 0},
		{"max", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(makeSuccessHandler("Test story.", "atmosphere", tt.confidence))
			defer server.Close()

			client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
			result, err := client.GenerateStory(context.Background(), testPOI(), "en")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Confidence != tt.expected {
				t.Errorf("expected confidence=%d, got %d", tt.expected, result.Confidence)
			}
		})
	}
}

func TestGenerateStory_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := messagesResponse{Content: []contentBlock{}, Usage: usage{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
	_, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err == nil {
		t.Fatal("expected error for empty response")
	}
}

func TestGenerateStory_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := messagesResponse{
			Content: []contentBlock{{Type: "text", Text: "This is not JSON at all."}},
			Usage:   usage{InputTokens: 100, OutputTokens: 20},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
	_, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err == nil {
		t.Fatal("expected error for non-JSON response text")
	}
}

func TestGenerateStory_EmptyText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		text := `{"text":"","layer_type":"atmosphere","confidence":50}`
		resp := messagesResponse{
			Content: []contentBlock{{Type: "text", Text: text}},
			Usage:   usage{InputTokens: 300, OutputTokens: 50},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
	_, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestGenerateStory_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"type":"invalid_request_error","message":"bad request"}}`)) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
	_, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestGenerateStory_RetryOn429(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"rate limited"}}`)) //nolint:errcheck // test helper
			return
		}
		text := `{"text":"Success after retry.","layer_type":"human_story","confidence":80}`
		resp := messagesResponse{
			Content: []contentBlock{{Type: "text", Text: text}},
			Usage:   usage{InputTokens: 400, OutputTokens: 100},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
	result, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if result.Text != "Success after retry." {
		t.Errorf("unexpected text: %s", result.Text)
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestGenerateStory_RetryOn500(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":{"type":"api_error","message":"internal error"}}`)) //nolint:errcheck // test helper
			return
		}
		text := `{"text":"Recovered.","layer_type":"atmosphere","confidence":70}`
		resp := messagesResponse{
			Content: []contentBlock{{Type: "text", Text: text}},
			Usage:   usage{InputTokens: 300, OutputTokens: 80},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
	result, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if result.Text != "Recovered." {
		t.Errorf("unexpected text: %s", result.Text)
	}
}

func TestGenerateStory_NoRetryOn400(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"type":"invalid_request_error","message":"bad"}}`)) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
	_, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts.Load() != 1 {
		t.Errorf("expected 1 attempt (no retry on 400), got %d", attempts.Load())
	}
}

func TestGenerateStory_MaxRetriesExhausted(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"rate limited"}}`)) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
	_, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err == nil {
		t.Fatal("expected error after max retries")
	}
	if attempts.Load() != int32(maxRetries) {
		t.Errorf("expected %d attempts, got %d", maxRetries, attempts.Load())
	}
}

func TestGenerateStory_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.GenerateStory(ctx, testPOI(), "en")
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func TestGenerateStory_ClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Timeout: 50 * time.Millisecond,
	})

	_, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err == nil {
		t.Fatal("expected timeout error")
	}

	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestGenerateStory_InjectedHTTPClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(&Config{
		APIKey:     "test-key",
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 50 * time.Millisecond},
	})

	_, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err == nil {
		t.Fatal("expected timeout error")
	}

	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestGenerateStory_RetryStopsOnContextCancellation(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"rate limited"}}`)) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.GenerateStory(ctx, testPOI(), "en")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
	if attempts.Load() != 1 {
		t.Errorf("expected retry loop to stop after 1 attempt, got %d", attempts.Load())
	}
}

func TestGenerateStory_RequestHeaders(t *testing.T) {
	var gotAPIKey, gotVersion, gotContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		gotContentType = r.Header.Get("Content-Type")

		text := `{"text":"Header test.","layer_type":"general","confidence":50}`
		resp := messagesResponse{
			Content: []contentBlock{{Type: "text", Text: text}},
			Usage:   usage{InputTokens: 200, OutputTokens: 50},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := NewClient(&Config{APIKey: "my-secret-key", BaseURL: server.URL})
	_, err := client.GenerateStory(context.Background(), testPOI(), "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAPIKey != "my-secret-key" {
		t.Errorf("expected x-api-key=my-secret-key, got %s", gotAPIKey)
	}
	if gotVersion != apiVersion {
		t.Errorf("expected anthropic-version=%s, got %s", apiVersion, gotVersion)
	}
	if gotContentType != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %s", gotContentType)
	}
}

func TestGenerateStory_AllLayerTypes(t *testing.T) {
	layerTypes := []struct {
		input    string
		expected domain.StoryLayerType
	}{
		{"atmosphere", domain.StoryLayerAtmosphere},
		{"human_story", domain.StoryLayerHumanStory},
		{"hidden_detail", domain.StoryLayerHiddenDetail},
		{"time_shift", domain.StoryLayerTimeShift},
		{"general", domain.StoryLayerGeneral},
	}

	for _, tt := range layerTypes {
		t.Run(tt.input, func(t *testing.T) {
			server := httptest.NewServer(makeSuccessHandler("Story.", tt.input, 80))
			defer server.Close()

			client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
			result, err := client.GenerateStory(context.Background(), testPOI(), "en")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.LayerType != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.LayerType)
			}
		})
	}
}

func TestGenerateStory_POIWithoutOptionalFields(t *testing.T) {
	server := httptest.NewServer(makeSuccessHandler("Minimal POI story.", "general", 60))
	defer server.Close()

	poi := &domain.POI{
		ID:     2,
		CityID: 1,
		Name:   "Unknown Bridge",
		Lat:    41.70,
		Lng:    44.80,
		Type:   domain.POITypeBridge,
		Status: domain.POIStatusActive,
	}

	client := NewClient(&Config{APIKey: "test-key", BaseURL: server.URL})
	result, err := client.GenerateStory(context.Background(), poi, "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Minimal POI story." {
		t.Errorf("unexpected text: %s", result.Text)
	}
}

func TestNewClient_DefaultValues(t *testing.T) {
	client := NewClient(&Config{APIKey: "test"})
	if client.baseURL != defaultBaseURL {
		t.Errorf("expected default baseURL %s, got %s", defaultBaseURL, client.baseURL)
	}
	if client.model != defaultModel {
		t.Errorf("expected default model %s, got %s", defaultModel, client.model)
	}
	if client.httpClient == nil {
		t.Fatal("expected http client to be initialized")
	}
	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("expected default timeout 60s, got %s", client.httpClient.Timeout)
	}
}

func TestNewClient_CustomValues(t *testing.T) {
	client := NewClient(&Config{
		APIKey:  "custom-key",
		BaseURL: "https://custom.api.com",
		Model:   "claude-haiku-4-5-20251001",
		Timeout: 5 * time.Second,
	})
	if client.baseURL != "https://custom.api.com" {
		t.Errorf("unexpected baseURL: %s", client.baseURL)
	}
	if client.model != "claude-haiku-4-5-20251001" {
		t.Errorf("unexpected model: %s", client.model)
	}
	if client.httpClient.Timeout != 5*time.Second {
		t.Errorf("unexpected timeout: %s", client.httpClient.Timeout)
	}
}

func TestNewClient_CustomHTTPClient(t *testing.T) {
	baseClient := &http.Client{Timeout: 3 * time.Second}
	client := NewClient(&Config{
		APIKey:     "custom-key",
		HTTPClient: baseClient,
	})

	if client.httpClient == baseClient {
		t.Fatal("expected client to copy injected http client")
	}
	if client.httpClient.Timeout != 3*time.Second {
		t.Errorf("unexpected timeout: %s", client.httpClient.Timeout)
	}

	override := NewClient(&Config{
		APIKey:     "custom-key",
		HTTPClient: baseClient,
		Timeout:    7 * time.Second,
	})
	if override.httpClient.Timeout != 7*time.Second {
		t.Errorf("expected override timeout 7s, got %s", override.httpClient.Timeout)
	}
}

func TestParseStoryResponse_JSONInText(t *testing.T) {
	text := `Here is the story: {"text":"Embedded JSON.","layer_type":"atmosphere","confidence":80} That's all.`
	result, err := parseStoryResponse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Embedded JSON." {
		t.Errorf("unexpected text: %s", result.Text)
	}
}

func TestParseStoryResponse_PlainJSON(t *testing.T) {
	text := `{"text":"Plain JSON.","layer_type":"human_story","confidence":90}`
	result, err := parseStoryResponse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Plain JSON." {
		t.Errorf("unexpected text: %s", result.Text)
	}
	if result.LayerType != "human_story" {
		t.Errorf("unexpected layer_type: %s", result.LayerType)
	}
}

func TestBuildSystemPrompt_English(t *testing.T) {
	prompt := buildSystemPrompt("en")
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
	if !strings.Contains(prompt, "Write in English") {
		t.Error("expected English instruction in prompt")
	}
	if !strings.Contains(prompt, "Anchor") || !strings.Contains(prompt, "Hook") || !strings.Contains(prompt, "Facts") || !strings.Contains(prompt, "Meaning") {
		t.Error("expected story structure in prompt")
	}
}

func TestBuildSystemPrompt_Russian(t *testing.T) {
	prompt := buildSystemPrompt("ru")
	if !strings.Contains(prompt, "Russian") {
		t.Error("expected Russian instruction in prompt")
	}
}

func TestBuildUserPrompt_FullPOI(t *testing.T) {
	poi := testPOI()
	prompt := buildUserPrompt(poi, "en")

	if !strings.Contains(prompt, "Narikala Fortress") {
		t.Error("expected POI name in prompt")
	}
	if !strings.Contains(prompt, "Нарикала") {
		t.Error("expected Russian name in prompt")
	}
	if !strings.Contains(prompt, "monument") {
		t.Error("expected POI type in prompt")
	}
	if !strings.Contains(prompt, "41.687500") {
		t.Error("expected coordinates in prompt")
	}
	if !strings.Contains(prompt, "wikidata") {
		t.Error("expected tags info in prompt")
	}
}

func TestBuildUserPrompt_MinimalPOI(t *testing.T) {
	poi := &domain.POI{
		ID:   1,
		Name: "Simple Place",
		Lat:  41.0,
		Lng:  44.0,
		Type: domain.POITypePark,
	}
	prompt := buildUserPrompt(poi, "en")
	if !strings.Contains(prompt, "Simple Place") {
		t.Error("expected POI name in prompt")
	}
	if !strings.Contains(prompt, "park") {
		t.Error("expected POI type in prompt")
	}
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{StatusCode: 429, Body: "rate limited"}
	msg := err.Error()
	if !strings.Contains(msg, "429") || !strings.Contains(msg, "rate limited") {
		t.Errorf("unexpected error message: %s", msg)
	}
}

func TestAPIError_TruncatesLongBody(t *testing.T) {
	longBody := strings.Repeat("x", 500)
	err := &APIError{StatusCode: 500, Body: longBody}
	msg := err.Error()
	if len(msg) > 300 {
		t.Errorf("error message too long: %d chars", len(msg))
	}
}

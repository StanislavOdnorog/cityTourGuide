package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/service"
)

// mockNearbyService implements NearbyStoriesGetter for testing.
type mockNearbyService struct {
	candidates []service.StoryCandidate
	err        error
	// captured arguments for assertion
	calledLat      float64
	calledLng      float64
	calledRadius   float64
	calledHeading  float64
	calledSpeed    float64
	calledUserID   string
	calledLanguage string
}

func (m *mockNearbyService) GetNearbyStories(
	_ context.Context,
	lat, lng, radiusM, heading, speed float64,
	userID, language string,
) ([]service.StoryCandidate, error) {
	m.calledLat = lat
	m.calledLng = lng
	m.calledRadius = radiusM
	m.calledHeading = heading
	m.calledSpeed = speed
	m.calledUserID = userID
	m.calledLanguage = language
	return m.candidates, m.err
}

func setupRouter(h *NearbyHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/nearby-stories", h.GetNearbyStories)
	return r
}

func TestGetNearbyStories_Success(t *testing.T) {
	audioURL := "https://example.com/audio.mp3"
	dur := int16(30)
	mock := &mockNearbyService{
		candidates: []service.StoryCandidate{
			{
				POIID:       1,
				POIName:     "Narikala Fortress",
				StoryID:     10,
				StoryText:   "A great story",
				AudioURL:    &audioURL,
				DurationSec: &dur,
				DistanceM:   42.5,
				Score:       87.3,
			},
		},
	}
	h := NewNearbyHandler(mock)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nearby-stories?lat=41.7151&lng=44.8271&language=en", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []service.StoryCandidate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(resp.Data))
	}
	if resp.Data[0].POIID != 1 {
		t.Errorf("expected poi_id=1, got %d", resp.Data[0].POIID)
	}
	if resp.Data[0].POIName != "Narikala Fortress" {
		t.Errorf("expected poi_name='Narikala Fortress', got %q", resp.Data[0].POIName)
	}
	if resp.Data[0].AudioURL == nil || *resp.Data[0].AudioURL != audioURL {
		t.Errorf("expected audio_url=%q, got %v", audioURL, resp.Data[0].AudioURL)
	}
}

func TestGetNearbyStories_EmptyResult(t *testing.T) {
	mock := &mockNearbyService{candidates: nil}
	h := NewNearbyHandler(mock)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nearby-stories?lat=0&lng=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data []service.StoryCandidate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("expected empty data array, got %d items", len(resp.Data))
	}
}

func TestGetNearbyStories_MissingLat(t *testing.T) {
	mock := &mockNearbyService{}
	h := NewNearbyHandler(mock)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nearby-stories?lng=44.8271", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["error"] != "lat is required" {
		t.Errorf("expected 'lat is required' error, got %q", resp["error"])
	}
}

func TestGetNearbyStories_MissingLng(t *testing.T) {
	mock := &mockNearbyService{}
	h := NewNearbyHandler(mock)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nearby-stories?lat=41.7151", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["error"] != "lng is required" {
		t.Errorf("expected 'lng is required' error, got %q", resp["error"])
	}
}

func TestGetNearbyStories_InvalidLat(t *testing.T) {
	tests := []struct {
		name    string
		lat     string
		wantErr string
	}{
		{"too high", "999", "lat must be between -90 and 90"},
		{"too low", "-91", "lat must be between -90 and 90"},
		{"not a number", "abc", "lat must be a valid number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockNearbyService{}
			h := NewNearbyHandler(mock)
			r := setupRouter(h)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/nearby-stories?lat="+tt.lat+"&lng=44.8", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", w.Code)
			}

			var resp map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if resp["error"] != tt.wantErr {
				t.Errorf("expected %q, got %q", tt.wantErr, resp["error"])
			}
		})
	}
}

func TestGetNearbyStories_InvalidLng(t *testing.T) {
	tests := []struct {
		name    string
		lng     string
		wantErr string
	}{
		{"too high", "181", "lng must be between -180 and 180"},
		{"too low", "-181", "lng must be between -180 and 180"},
		{"not a number", "xyz", "lng must be a valid number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockNearbyService{}
			h := NewNearbyHandler(mock)
			r := setupRouter(h)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/nearby-stories?lat=41.7&lng="+tt.lng, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", w.Code)
			}

			var resp map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if resp["error"] != tt.wantErr {
				t.Errorf("expected %q, got %q", tt.wantErr, resp["error"])
			}
		})
	}
}

func TestGetNearbyStories_InvalidRadius(t *testing.T) {
	tests := []struct {
		name    string
		radius  string
		wantErr string
	}{
		{"too small", "5", "radius must be between 10 and 500"},
		{"too large", "501", "radius must be between 10 and 500"},
		{"not a number", "big", "radius must be a valid number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockNearbyService{}
			h := NewNearbyHandler(mock)
			r := setupRouter(h)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/nearby-stories?lat=41.7&lng=44.8&radius="+tt.radius, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", w.Code)
			}

			var resp map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if resp["error"] != tt.wantErr {
				t.Errorf("expected %q, got %q", tt.wantErr, resp["error"])
			}
		})
	}
}

func TestGetNearbyStories_DefaultValues(t *testing.T) {
	mock := &mockNearbyService{candidates: nil}
	h := NewNearbyHandler(mock)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nearby-stories?lat=41.7&lng=44.8", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Check defaults passed to service
	if mock.calledRadius != 150 {
		t.Errorf("expected default radius=150, got %f", mock.calledRadius)
	}
	if mock.calledHeading != -1 {
		t.Errorf("expected default heading=-1, got %f", mock.calledHeading)
	}
	if mock.calledSpeed != 0 {
		t.Errorf("expected default speed=0, got %f", mock.calledSpeed)
	}
	if mock.calledLanguage != "en" {
		t.Errorf("expected default language='en', got %q", mock.calledLanguage)
	}
	if mock.calledUserID != "" {
		t.Errorf("expected empty user_id, got %q", mock.calledUserID)
	}
}

func TestGetNearbyStories_AllParams(t *testing.T) {
	mock := &mockNearbyService{candidates: nil}
	h := NewNearbyHandler(mock)
	r := setupRouter(h)

	url := "/api/v1/nearby-stories?lat=41.7151&lng=44.8271&radius=200&heading=90&speed=1.5&language=ru&user_id=user-123"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if mock.calledLat != 41.7151 {
		t.Errorf("expected lat=41.7151, got %f", mock.calledLat)
	}
	if mock.calledLng != 44.8271 {
		t.Errorf("expected lng=44.8271, got %f", mock.calledLng)
	}
	if mock.calledRadius != 200 {
		t.Errorf("expected radius=200, got %f", mock.calledRadius)
	}
	if mock.calledHeading != 90 {
		t.Errorf("expected heading=90, got %f", mock.calledHeading)
	}
	if mock.calledSpeed != 1.5 {
		t.Errorf("expected speed=1.5, got %f", mock.calledSpeed)
	}
	if mock.calledLanguage != "ru" {
		t.Errorf("expected language='ru', got %q", mock.calledLanguage)
	}
	if mock.calledUserID != "user-123" {
		t.Errorf("expected user_id='user-123', got %q", mock.calledUserID)
	}
}

func TestGetNearbyStories_ServiceError(t *testing.T) {
	mock := &mockNearbyService{err: errors.New("database error")}
	h := NewNearbyHandler(mock)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nearby-stories?lat=41.7&lng=44.8", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["error"] != "failed to fetch nearby stories" {
		t.Errorf("expected generic error message, got %q", resp["error"])
	}
}

func TestGetNearbyStories_ResponseIncludesAudioURL(t *testing.T) {
	audioURL := "https://s3.example.com/audio/1/2/3.mp3"
	dur := int16(25)
	mock := &mockNearbyService{
		candidates: []service.StoryCandidate{
			{
				POIID:       5,
				POIName:     "Bridge of Peace",
				StoryID:     20,
				StoryText:   "A modern masterpiece",
				AudioURL:    &audioURL,
				DurationSec: &dur,
				DistanceM:   100.0,
				Score:       65.0,
			},
		},
	}
	h := NewNearbyHandler(mock)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nearby-stories?lat=41.7&lng=44.8&language=en", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data []struct {
			POIID       int     `json:"poi_id"`
			POIName     string  `json:"poi_name"`
			StoryID     int     `json:"story_id"`
			StoryText   string  `json:"story_text"`
			AudioURL    *string `json:"audio_url"`
			DurationSec *int16  `json:"duration_sec"`
			DistanceM   float64 `json:"distance_m"`
			Score       float64 `json:"score"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(resp.Data))
	}
	c := resp.Data[0]
	if c.AudioURL == nil || *c.AudioURL != audioURL {
		t.Errorf("expected audio_url=%q, got %v", audioURL, c.AudioURL)
	}
	if c.DurationSec == nil || *c.DurationSec != 25 {
		t.Errorf("expected duration_sec=25, got %v", c.DurationSec)
	}
	if c.DistanceM != 100.0 {
		t.Errorf("expected distance_m=100.0, got %f", c.DistanceM)
	}
	if c.Score != 65.0 {
		t.Errorf("expected score=65.0, got %f", c.Score)
	}
}

func TestGetNearbyStories_BoundaryLatLng(t *testing.T) {
	tests := []struct {
		name     string
		lat, lng string
		wantCode int
	}{
		{"lat=-90 valid", "-90", "0", http.StatusOK},
		{"lat=90 valid", "90", "0", http.StatusOK},
		{"lng=-180 valid", "0", "-180", http.StatusOK},
		{"lng=180 valid", "0", "180", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockNearbyService{candidates: nil}
			h := NewNearbyHandler(mock)
			r := setupRouter(h)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/nearby-stories?lat="+tt.lat+"&lng="+tt.lng, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("expected %d, got %d: %s", tt.wantCode, w.Code, w.Body.String())
			}
		})
	}
}

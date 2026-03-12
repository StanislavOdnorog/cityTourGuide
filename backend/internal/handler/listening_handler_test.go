package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// mockListeningRepo implements ListeningRepository for testing.
type mockListeningRepo struct {
	listening  *domain.UserListening
	listenings []domain.UserListening
	err        error
	listErr    error
	// Captured args for verification
	lastUserID    string
	lastStoryID   int
	lastCompleted bool
	lastLat       *float64
	lastLng       *float64
}

func (m *mockListeningRepo) CreateOrUpdate(_ context.Context, userID string, storyID int, completed bool, lat, lng *float64) (*domain.UserListening, error) {
	m.lastUserID = userID
	m.lastStoryID = storyID
	m.lastCompleted = completed
	m.lastLat = lat
	m.lastLng = lng

	if m.err != nil {
		return nil, m.err
	}
	if m.listening != nil {
		return m.listening, nil
	}

	return &domain.UserListening{
		ID:         1,
		UserID:     userID,
		StoryID:    storyID,
		ListenedAt: time.Now(),
		Completed:  completed,
		Lat:        lat,
		Lng:        lng,
	}, nil
}

func (m *mockListeningRepo) ListByUserID(_ context.Context, _ string, page domain.PageRequest) (*domain.PageResponse[domain.UserListening], error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	items := m.listenings
	hasMore := false
	if len(items) > page.Limit {
		items = items[:page.Limit]
		hasMore = true
	}
	return &domain.PageResponse[domain.UserListening]{
		Items:   items,
		HasMore: hasMore,
	}, nil
}

func setupListeningRouter(h *ListeningHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/listenings", h.ListListenings)
	r.POST("/api/v1/listenings", h.TrackListening)
	return r
}

func postJSON(router *gin.Engine, path string, body interface{}) *httptest.ResponseRecorder {
	jsonBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestTrackListening_Success(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	lat := 41.7151
	lng := 44.8271
	body := map[string]interface{}{
		"user_id":   "550e8400-e29b-41d4-a716-446655440000",
		"story_id":  42,
		"completed": true,
		"lat":       lat,
		"lng":       lng,
	}

	w := postJSON(router, "/api/v1/listenings", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("response missing data field")
	}

	if data["user_id"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("unexpected user_id: %v", data["user_id"])
	}
	if int(data["story_id"].(float64)) != 42 {
		t.Errorf("unexpected story_id: %v", data["story_id"])
	}
	if data["completed"] != true {
		t.Errorf("expected completed=true, got %v", data["completed"])
	}
}

func TestListListenings_Success(t *testing.T) {
	mock := &mockListeningRepo{
		listenings: []domain.UserListening{
			{ID: 1, UserID: "user-1", StoryID: 10, Completed: true, ListenedAt: time.Now()},
			{ID: 2, UserID: "user-1", StoryID: 11, Completed: false, ListenedAt: time.Now()},
		},
	}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/listenings?user_id=user-1&limit=1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items      []domain.UserListening `json:"items"`
		NextCursor string                 `json:"next_cursor"`
		HasMore    bool                   `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 listening, got %d", len(resp.Items))
	}
	if !resp.HasMore {
		t.Fatal("expected has_more=true")
	}
}

func TestListListenings_InvalidLimit(t *testing.T) {
	h := NewListeningHandler(&mockListeningRepo{})
	router := setupListeningRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/listenings?user_id=user-1&limit=101", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] != "limit must not exceed 100" {
		t.Fatalf("unexpected error: %q", resp["error"])
	}
}

func TestListListenings_InvalidCursor(t *testing.T) {
	h := NewListeningHandler(&mockListeningRepo{listErr: errors.New("invalid cursor: malformed encoding")})
	router := setupListeningRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/listenings?user_id=user-1&cursor=bad", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestTrackListening_SuccessWithoutCoordinates(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	body := map[string]interface{}{
		"user_id":   "550e8400-e29b-41d4-a716-446655440000",
		"story_id":  42,
		"completed": false,
	}

	w := postJSON(router, "/api/v1/listenings", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	if mock.lastLat != nil || mock.lastLng != nil {
		t.Error("expected nil lat/lng when not provided")
	}
}

func TestTrackListening_MissingUserID(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	body := map[string]interface{}{
		"story_id":  42,
		"completed": true,
	}

	w := postJSON(router, "/api/v1/listenings", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestTrackListening_MissingStoryID(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	body := map[string]interface{}{
		"user_id":   "550e8400-e29b-41d4-a716-446655440000",
		"completed": true,
	}

	w := postJSON(router, "/api/v1/listenings", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestTrackListening_InvalidStoryID(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	body := map[string]interface{}{
		"user_id":   "550e8400-e29b-41d4-a716-446655440000",
		"story_id":  -1,
		"completed": true,
	}

	w := postJSON(router, "/api/v1/listenings", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "story_id must be a positive integer" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestTrackListening_LatWithoutLng(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	lat := 41.7151
	body := map[string]interface{}{
		"user_id":   "550e8400-e29b-41d4-a716-446655440000",
		"story_id":  42,
		"completed": true,
		"lat":       lat,
	}

	w := postJSON(router, "/api/v1/listenings", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "lat and lng must both be provided or both omitted" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestTrackListening_LngWithoutLat(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	lng := 44.8271
	body := map[string]interface{}{
		"user_id":   "550e8400-e29b-41d4-a716-446655440000",
		"story_id":  42,
		"completed": true,
		"lng":       lng,
	}

	w := postJSON(router, "/api/v1/listenings", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestTrackListening_InvalidLatRange(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	tests := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"lat too high", 91.0, 44.0},
		{"lat too low", -91.0, 44.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]interface{}{
				"user_id":   "550e8400-e29b-41d4-a716-446655440000",
				"story_id":  42,
				"completed": true,
				"lat":       tt.lat,
				"lng":       tt.lng,
			}

			w := postJSON(router, "/api/v1/listenings", body)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			if resp["error"] != "lat must be between -90 and 90" {
				t.Errorf("unexpected error: %v", resp["error"])
			}
		})
	}
}

func TestTrackListening_InvalidLngRange(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	tests := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"lng too high", 41.0, 181.0},
		{"lng too low", 41.0, -181.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]interface{}{
				"user_id":   "550e8400-e29b-41d4-a716-446655440000",
				"story_id":  42,
				"completed": true,
				"lat":       tt.lat,
				"lng":       tt.lng,
			}

			w := postJSON(router, "/api/v1/listenings", body)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			if resp["error"] != "lng must be between -180 and 180" {
				t.Errorf("unexpected error: %v", resp["error"])
			}
		})
	}
}

func TestTrackListening_ServiceError(t *testing.T) {
	mock := &mockListeningRepo{
		err: errors.New("database error"),
	}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	body := map[string]interface{}{
		"user_id":   "550e8400-e29b-41d4-a716-446655440000",
		"story_id":  42,
		"completed": true,
	}

	w := postJSON(router, "/api/v1/listenings", body)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "failed to track listening" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestTrackListening_DefaultCompletedFalse(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	body := map[string]interface{}{
		"user_id":  "550e8400-e29b-41d4-a716-446655440000",
		"story_id": 42,
	}

	w := postJSON(router, "/api/v1/listenings", body)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	if mock.lastCompleted != false {
		t.Error("expected completed=false by default")
	}
}

func TestTrackListening_ParamsPassedToRepo(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	lat := 41.7151
	lng := 44.8271
	body := map[string]interface{}{
		"user_id":   "550e8400-e29b-41d4-a716-446655440000",
		"story_id":  42,
		"completed": true,
		"lat":       lat,
		"lng":       lng,
	}

	w := postJSON(router, "/api/v1/listenings", body)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	if mock.lastUserID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("unexpected user_id passed to repo: %s", mock.lastUserID)
	}
	if mock.lastStoryID != 42 {
		t.Errorf("unexpected story_id passed to repo: %d", mock.lastStoryID)
	}
	if mock.lastCompleted != true {
		t.Error("expected completed=true passed to repo")
	}
	if mock.lastLat == nil || *mock.lastLat != lat {
		t.Errorf("unexpected lat passed to repo: %v", mock.lastLat)
	}
	if mock.lastLng == nil || *mock.lastLng != lng {
		t.Errorf("unexpected lng passed to repo: %v", mock.lastLng)
	}
}

func TestTrackListening_BoundaryCoordinates(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	tests := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"max lat", 90.0, 0.0},
		{"min lat", -90.0, 0.0},
		{"max lng", 0.0, 180.0},
		{"min lng", 0.0, -180.0},
		{"all zeros", 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]interface{}{
				"user_id":   "550e8400-e29b-41d4-a716-446655440000",
				"story_id":  1,
				"completed": true,
				"lat":       tt.lat,
				"lng":       tt.lng,
			}

			w := postJSON(router, "/api/v1/listenings", body)

			if w.Code != http.StatusCreated {
				t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestTrackListening_ResponseContainsFields(t *testing.T) {
	lat := 41.7151
	lng := 44.8271
	mock := &mockListeningRepo{
		listening: &domain.UserListening{
			ID:         5,
			UserID:     "550e8400-e29b-41d4-a716-446655440000",
			StoryID:    42,
			ListenedAt: time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC),
			Completed:  true,
			Lat:        &lat,
			Lng:        &lng,
		},
	}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	body := map[string]interface{}{
		"user_id":   "550e8400-e29b-41d4-a716-446655440000",
		"story_id":  42,
		"completed": true,
		"lat":       lat,
		"lng":       lng,
	}

	w := postJSON(router, "/api/v1/listenings", body)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].(map[string]interface{})

	// Verify all expected fields present
	expectedFields := []string{"id", "user_id", "story_id", "listened_at", "completed", "lat", "lng"}
	for _, field := range expectedFields {
		if _, ok := data[field]; !ok {
			t.Errorf("response missing field: %s", field)
		}
	}

	if int(data["id"].(float64)) != 5 {
		t.Errorf("unexpected id: %v", data["id"])
	}
}

func TestTrackListening_EmptyBody(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/listenings", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTrackListening_InvalidJSON(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := setupListeningRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/listenings", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestListeningHandler_ValidationResponses(t *testing.T) {
	mock := &mockListeningRepo{}
	h := NewListeningHandler(mock)
	router := newRouterWithTrace("trace-listening-123", func(r *gin.Engine) {
		r.GET("/api/v1/listenings", h.ListListenings)
		r.POST("/api/v1/listenings", h.TrackListening)
	})

	tests := []struct {
		name          string
		method        string
		path          string
		body          string
		expectedCode  int
		expectedError string
		expectedField map[string]string
	}{
		{
			name:         "track listening missing user id",
			method:       http.MethodPost,
			path:         "/api/v1/listenings",
			body:         `{"story_id":42}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"userid": "this field is required",
			},
		},
		{
			name:         "track listening missing story id",
			method:       http.MethodPost,
			path:         "/api/v1/listenings",
			body:         `{"user_id":"550e8400-e29b-41d4-a716-446655440000"}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"storyid": "this field is required",
			},
		},
		{
			name:          "list listenings missing user id",
			method:        http.MethodGet,
			path:          "/api/v1/listenings",
			expectedCode:  http.StatusBadRequest,
			expectedError: "user_id is required",
		},
		{
			name:          "list listenings invalid limit",
			method:        http.MethodGet,
			path:          "/api/v1/listenings?user_id=user-1&limit=0",
			expectedCode:  http.StatusBadRequest,
			expectedError: "limit must be a positive integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body == "" {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			} else {
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Fatalf("expected status %d, got %d: %s", tt.expectedCode, w.Code, w.Body.String())
			}

			if tt.expectedField != nil {
				assertValidationErrorResponse(t, w.Body.Bytes(), tt.expectedField, "trace-listening-123")
				return
			}
			assertErrorResponse(t, w.Body.Bytes(), tt.expectedError, "trace-listening-123")
		})
	}
}

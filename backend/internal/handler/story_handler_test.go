package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// mockStoryRepo implements StoryRepository for testing.
type mockStoryRepo struct {
	stories   []domain.Story
	story     *domain.Story
	err       error
	createErr error
	deleteErr error
	// captured args
	calledPOIID    int
	calledLanguage string
	calledStatus   *domain.StoryStatus
}

func (m *mockStoryRepo) Create(_ context.Context, story *domain.Story) (*domain.Story, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	story.ID = 1
	story.CreatedAt = time.Now()
	story.UpdatedAt = time.Now()
	return story, nil
}

func (m *mockStoryRepo) GetByID(_ context.Context, _ int) (*domain.Story, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.story, nil
}

func (m *mockStoryRepo) GetByPOIID(_ context.Context, poiID int, language string, status *domain.StoryStatus) ([]domain.Story, error) {
	m.calledPOIID = poiID
	m.calledLanguage = language
	m.calledStatus = status
	if m.err != nil {
		return nil, m.err
	}
	return m.stories, nil
}

func (m *mockStoryRepo) Update(_ context.Context, story *domain.Story) (*domain.Story, error) {
	if m.err != nil {
		return nil, m.err
	}
	story.UpdatedAt = time.Now()
	return story, nil
}

func (m *mockStoryRepo) Delete(_ context.Context, _ int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func setupStoryRouter(h *StoryHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/stories", h.ListStories)
	r.GET("/api/v1/stories/:id", h.GetStory)
	r.POST("/api/v1/admin/stories", h.CreateStory)
	r.PUT("/api/v1/admin/stories/:id", h.UpdateStory)
	r.DELETE("/api/v1/admin/stories/:id", h.DeleteStory)
	return r
}

func TestListStories_Success(t *testing.T) {
	audioURL := "https://example.com/audio.mp3"
	dur := int16(30)
	mock := &mockStoryRepo{
		stories: []domain.Story{
			{ID: 1, POIID: 1, Language: "en", Text: "A great story", AudioURL: &audioURL, DurationSec: &dur, LayerType: domain.StoryLayerGeneral, Status: domain.StoryStatusActive},
			{ID: 2, POIID: 1, Language: "en", Text: "Another story", LayerType: domain.StoryLayerAtmosphere, Status: domain.StoryStatusActive},
		},
	}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stories?poi_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []domain.Story `json:"data"`
		Meta struct {
			Total   int `json:"total"`
			Page    int `json:"page"`
			PerPage int `json:"per_page"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 stories, got %d", len(resp.Data))
	}
	if resp.Meta.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Meta.Total)
	}
}

func TestListStories_MissingPOIID(t *testing.T) {
	mock := &mockStoryRepo{}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stories", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["error"] != "poi_id is required" {
		t.Errorf("expected 'poi_id is required', got %q", resp["error"])
	}
}

func TestListStories_DefaultLanguage(t *testing.T) {
	mock := &mockStoryRepo{stories: []domain.Story{}}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stories?poi_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if mock.calledLanguage != "en" {
		t.Errorf("expected default language=en, got %q", mock.calledLanguage)
	}
}

func TestListStories_WithFilters(t *testing.T) {
	mock := &mockStoryRepo{stories: []domain.Story{}}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stories?poi_id=1&language=ru&status=active", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if mock.calledPOIID != 1 {
		t.Errorf("expected poi_id=1, got %d", mock.calledPOIID)
	}
	if mock.calledLanguage != "ru" {
		t.Errorf("expected language=ru, got %q", mock.calledLanguage)
	}
	if mock.calledStatus == nil || *mock.calledStatus != domain.StoryStatusActive {
		t.Errorf("expected status=active, got %v", mock.calledStatus)
	}
}

func TestListStories_Pagination(t *testing.T) {
	stories := make([]domain.Story, 25)
	for i := range stories {
		stories[i] = domain.Story{ID: i + 1, POIID: 1, Language: "en", Text: "Story", LayerType: domain.StoryLayerGeneral, Status: domain.StoryStatusActive}
	}
	mock := &mockStoryRepo{stories: stories}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stories?poi_id=1&page=2&per_page=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data []domain.Story `json:"data"`
		Meta struct {
			Total   int `json:"total"`
			Page    int `json:"page"`
			PerPage int `json:"per_page"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Data) != 10 {
		t.Errorf("expected 10 stories on page 2, got %d", len(resp.Data))
	}
	if resp.Meta.Total != 25 {
		t.Errorf("expected total=25, got %d", resp.Meta.Total)
	}
}

func TestGetStory_Success(t *testing.T) {
	audioURL := "https://example.com/audio.mp3"
	dur := int16(30)
	mock := &mockStoryRepo{
		story: &domain.Story{ID: 1, POIID: 1, Language: "en", Text: "Great story", AudioURL: &audioURL, DurationSec: &dur, LayerType: domain.StoryLayerHumanStory, Status: domain.StoryStatusActive},
	}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stories/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data domain.Story `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.Text != "Great story" {
		t.Errorf("expected text='Great story', got %q", resp.Data.Text)
	}
}

func TestGetStory_NotFound(t *testing.T) {
	mock := &mockStoryRepo{err: repository.ErrNotFound}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stories/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCreateStory_Success(t *testing.T) {
	mock := &mockStoryRepo{}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	body := `{"poi_id":1,"language":"en","text":"A test story about history","layer_type":"general"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data domain.Story `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.Text != "A test story about history" {
		t.Errorf("expected text, got %q", resp.Data.Text)
	}
	if resp.Data.Confidence != 80 {
		t.Errorf("expected default confidence=80, got %d", resp.Data.Confidence)
	}
	if resp.Data.Status != domain.StoryStatusActive {
		t.Errorf("expected default status=active, got %q", resp.Data.Status)
	}
	if resp.Data.OrderIndex != 0 {
		t.Errorf("expected default order_index=0, got %d", resp.Data.OrderIndex)
	}
	if resp.Data.IsInflation {
		t.Error("expected default is_inflation=false")
	}
}

func TestCreateStory_MissingRequired(t *testing.T) {
	mock := &mockStoryRepo{}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	body := `{"poi_id":1,"language":"en"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStory_WithOptionalFields(t *testing.T) {
	mock := &mockStoryRepo{}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	audioURL := "https://example.com/audio.mp3"
	dur := int16(30)
	orderIdx := int16(2)
	isInflation := true
	confidence := int16(90)
	status := domain.StoryStatusDisabled
	sources := json.RawMessage(`{"generator":"claude"}`)

	reqBody, _ := json.Marshal(createStoryRequest{
		POIID:       1,
		Language:    "en",
		Text:        "Story text",
		AudioURL:    &audioURL,
		DurationSec: &dur,
		LayerType:   domain.StoryLayerHumanStory,
		OrderIndex:  &orderIdx,
		IsInflation: &isInflation,
		Confidence:  &confidence,
		Sources:     &sources,
		Status:      &status,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stories", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data domain.Story `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.OrderIndex != 2 {
		t.Errorf("expected order_index=2, got %d", resp.Data.OrderIndex)
	}
	if !resp.Data.IsInflation {
		t.Error("expected is_inflation=true")
	}
	if resp.Data.Confidence != 90 {
		t.Errorf("expected confidence=90, got %d", resp.Data.Confidence)
	}
	if resp.Data.Status != domain.StoryStatusDisabled {
		t.Errorf("expected status=disabled, got %q", resp.Data.Status)
	}
}

func TestUpdateStory_Success(t *testing.T) {
	mock := &mockStoryRepo{}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	body := `{"poi_id":1,"language":"en","text":"Updated story","layer_type":"atmosphere"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/stories/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data domain.Story `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.Text != "Updated story" {
		t.Errorf("expected text='Updated story', got %q", resp.Data.Text)
	}
}

func TestUpdateStory_NotFound(t *testing.T) {
	mock := &mockStoryRepo{err: repository.ErrNotFound}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	body := `{"poi_id":1,"language":"en","text":"Test","layer_type":"general"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/stories/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteStory_Success(t *testing.T) {
	mock := &mockStoryRepo{}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/stories/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["message"] != "story deleted" {
		t.Errorf("expected 'story deleted', got %q", resp["message"])
	}
}

func TestDeleteStory_NotFound(t *testing.T) {
	mock := &mockStoryRepo{deleteErr: repository.ErrNotFound}
	h := NewStoryHandler(mock)
	r := setupStoryRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/stories/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

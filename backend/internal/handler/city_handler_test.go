package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// mockManifestRepo implements DownloadManifestRepository for testing.
type mockManifestRepo struct {
	items []domain.DownloadManifestItem
	err   error
}

func (m *mockManifestRepo) GetDownloadManifest(_ context.Context, _ int, _ string) ([]domain.DownloadManifestItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.items, nil
}

// mockCityRepo implements CityRepository for testing.
type mockCityRepo struct {
	cities     []domain.City
	city       *domain.City
	err        error
	createErr  error
	deleteErr  error
	restoreErr error
	restoreOut *domain.City
	calledSort repository.ListSort
}

func (m *mockCityRepo) GetActiveByID(_ context.Context, id int) (*domain.City, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.city != nil && (!m.city.IsActive || m.city.DeletedAt != nil) {
		return nil, repository.ErrNotFound
	}
	return m.city, nil
}

func (m *mockCityRepo) ListActive(_ context.Context, page domain.PageRequest) (*domain.PageResponse[domain.City], error) {
	if m.err != nil {
		return nil, m.err
	}
	var active []domain.City
	for _, c := range m.cities {
		if c.IsActive {
			active = append(active, c)
		}
	}
	hasMore := false
	if len(active) > page.Limit {
		active = active[:page.Limit]
		hasMore = true
	}
	return &domain.PageResponse[domain.City]{
		Items:   active,
		HasMore: hasMore,
	}, nil
}

func (m *mockCityRepo) Create(_ context.Context, city *domain.City) (*domain.City, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	city.ID = 1
	city.CreatedAt = time.Now()
	city.UpdatedAt = time.Now()
	return city, nil
}

func (m *mockCityRepo) GetByID(_ context.Context, _ int, includeDeleted bool) (*domain.City, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.city != nil && m.city.DeletedAt != nil && !includeDeleted {
		return nil, repository.ErrNotFound
	}
	return m.city, nil
}

func (m *mockCityRepo) GetAll(_ context.Context) ([]domain.City, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.cities, nil
}

func (m *mockCityRepo) Update(_ context.Context, city *domain.City) (*domain.City, error) {
	if m.err != nil {
		return nil, m.err
	}
	city.UpdatedAt = time.Now()
	return city, nil
}

func (m *mockCityRepo) List(_ context.Context, page domain.PageRequest, includeDeleted bool, sort repository.ListSort) (*domain.PageResponse[domain.City], error) {
	m.calledSort = sort
	if m.err != nil {
		return nil, m.err
	}
	items := make([]domain.City, 0, len(m.cities))
	for _, city := range m.cities {
		if city.DeletedAt != nil && !includeDeleted {
			continue
		}
		items = append(items, city)
	}
	hasMore := false
	if len(items) > page.Limit {
		items = items[:page.Limit]
		hasMore = true
	}
	return &domain.PageResponse[domain.City]{
		Items:   items,
		HasMore: hasMore,
	}, nil
}

func TestListAdminCities_SortParams(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cities?sort_by=name&sort_dir=desc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if mock.calledSort.By != "name" || mock.calledSort.Dir != repository.SortDirDesc {
		t.Fatalf("expected sort {name desc}, got %+v", mock.calledSort)
	}
}

func TestListAdminCities_InvalidSortBy(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cities?sort_by=deleted_at", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func (m *mockCityRepo) Delete(_ context.Context, _ int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func (m *mockCityRepo) Restore(_ context.Context, _ int) (*domain.City, error) {
	if m.restoreErr != nil {
		return nil, m.restoreErr
	}
	if m.restoreOut != nil {
		return m.restoreOut, nil
	}
	return &domain.City{ID: 1, Name: "Restored", Country: "GE", IsActive: true}, nil
}

func setupCityRouter(h *CityHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/cities", h.ListCities)
	r.GET("/api/v1/cities/:id", h.GetCity)
	r.GET("/api/v1/cities/:id/download-manifest", h.GetDownloadManifest)
	r.GET("/api/v1/admin/cities", h.ListAdminCities)
	r.POST("/api/v1/admin/cities", h.CreateCity)
	r.PUT("/api/v1/admin/cities/:id", h.UpdateCity)
	r.DELETE("/api/v1/admin/cities/:id", h.DeleteCity)
	r.POST("/api/v1/admin/cities/:id/restore", h.RestoreCity)
	return r
}

func TestListCities_Success(t *testing.T) {
	nameRu := "Тбилиси"
	mock := &mockCityRepo{
		cities: []domain.City{
			{ID: 1, Name: "Tbilisi", NameRu: &nameRu, Country: "GE", CenterLat: 41.7, CenterLng: 44.8, RadiusKm: 15, IsActive: true},
			{ID: 2, Name: "Batumi", Country: "GE", CenterLat: 41.6, CenterLng: 41.6, RadiusKm: 10, IsActive: true},
		},
	}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	w := executeRequest(r, httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items      []domain.City `json:"items"`
		NextCursor string        `json:"next_cursor"`
		HasMore    bool          `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Errorf("expected 2 cities, got %d", len(resp.Items))
	}
	if resp.HasMore {
		t.Error("expected has_more=false")
	}
}

func TestListCities_EmptyResult(t *testing.T) {
	mock := &mockCityRepo{cities: nil}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	w := executeRequest(r, httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Items []domain.City `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp.Items))
	}
}

func TestListCities_Pagination(t *testing.T) {
	cities := make([]domain.City, 25)
	for i := range cities {
		cities[i] = domain.City{ID: i + 1, Name: "City", Country: "GE", IsActive: true}
	}
	mock := &mockCityRepo{cities: cities}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	w := executeRequest(r, httptest.NewRequest(http.MethodGet, "/api/v1/cities?limit=10", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Items      []domain.City `json:"items"`
		NextCursor string        `json:"next_cursor"`
		HasMore    bool          `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Items) != 10 {
		t.Errorf("expected 10 cities, got %d", len(resp.Items))
	}
	if !resp.HasMore {
		t.Error("expected has_more=true")
	}
}

func TestListCities_ServiceError(t *testing.T) {
	mock := &mockCityRepo{err: repository.ErrNotFound}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	w := executeRequest(r, httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGetCity_Success(t *testing.T) {
	mock := &mockCityRepo{
		city: &domain.City{ID: 1, Name: "Tbilisi", Country: "GE", CenterLat: 41.7, CenterLng: 44.8, RadiusKm: 15, IsActive: true},
	}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data domain.City `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.Name != "Tbilisi" {
		t.Errorf("expected name=Tbilisi, got %q", resp.Data.Name)
	}
}

func TestGetCity_NotFound(t *testing.T) {
	mock := &mockCityRepo{err: repository.ErrNotFound}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["error"] != "city not found" {
		t.Errorf("expected 'city not found', got %q", resp["error"])
	}
}

func TestGetCity_InvalidID(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateCity_Success(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/admin/cities", `{"name":"Tbilisi","country":"GE","center_lat":41.7151,"center_lng":44.8271,"radius_km":15}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data domain.City `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.Name != "Tbilisi" {
		t.Errorf("expected name=Tbilisi, got %q", resp.Data.Name)
	}
	if resp.Data.ID != 1 {
		t.Errorf("expected id=1, got %d", resp.Data.ID)
	}
	if !resp.Data.IsActive {
		t.Error("expected is_active=true by default")
	}
}

func TestCreateCity_MissingRequired(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/admin/cities", `{"name":"Tbilisi"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCity_WithOptionalFields(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	isActive := false
	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/admin/cities", createCityRequest{
		Name:      "Batumi",
		Country:   "GE",
		CenterLat: 41.6,
		CenterLng: 41.6,
		RadiusKm:  10,
		IsActive:  &isActive,
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data domain.City `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.IsActive {
		t.Error("expected is_active=false")
	}
}

func TestUpdateCity_Success(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	w := executeJSONRequest(t, r, http.MethodPut, "/api/v1/admin/cities/1", `{"name":"Tbilisi Updated","country":"GE","center_lat":41.7151,"center_lng":44.8271,"radius_km":20}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data domain.City `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.Name != "Tbilisi Updated" {
		t.Errorf("expected name='Tbilisi Updated', got %q", resp.Data.Name)
	}
}

func TestUpdateCity_NotFound(t *testing.T) {
	mock := &mockCityRepo{err: repository.ErrNotFound}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	w := executeJSONRequest(t, r, http.MethodPut, "/api/v1/admin/cities/999", `{"name":"Tbilisi","country":"GE","center_lat":41.7,"center_lng":44.8,"radius_km":15}`)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteCity_Success(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/cities/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["message"] != "city deleted" {
		t.Errorf("expected 'city deleted', got %q", resp["message"])
	}
}

func TestDeleteCity_NotFound(t *testing.T) {
	mock := &mockCityRepo{deleteErr: repository.ErrNotFound}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/cities/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetDownloadManifest_Success(t *testing.T) {
	audioURL := "https://s3.example.com/audio/story_1.mp3"
	dur := int16(120)
	manifestMock := &mockManifestRepo{
		items: []domain.DownloadManifestItem{
			{StoryID: 1, POIID: 10, POIName: "Narikala Fortress", AudioURL: &audioURL, DurationSec: &dur, FileSizeBytes: 2880000},
			{StoryID: 2, POIID: 10, POIName: "Narikala Fortress", AudioURL: &audioURL, DurationSec: &dur, FileSizeBytes: 1920000},
		},
	}
	cityMock := &mockCityRepo{
		city: &domain.City{ID: 1, Name: "Tbilisi", Country: "GE", CenterLat: 41.7, CenterLng: 44.8, RadiusKm: 15, IsActive: true},
	}
	h := NewCityHandler(cityMock, manifestMock, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/1/download-manifest?language=en", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data           []domain.DownloadManifestItem `json:"data"`
		TotalSizeBytes int64                         `json:"total_size_bytes"`
		TotalStories   int                           `json:"total_stories"`
		CityName       string                        `json:"city_name"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Data))
	}
	if resp.TotalSizeBytes != 4800000 {
		t.Errorf("expected total_size_bytes=4800000, got %d", resp.TotalSizeBytes)
	}
	if resp.TotalStories != 2 {
		t.Errorf("expected total_stories=2, got %d", resp.TotalStories)
	}
	if resp.CityName != "Tbilisi" {
		t.Errorf("expected city_name=Tbilisi, got %q", resp.CityName)
	}
}

func TestGetDownloadManifest_CityNotFound(t *testing.T) {
	cityMock := &mockCityRepo{err: repository.ErrNotFound}
	h := NewCityHandler(cityMock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/999/download-manifest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetDownloadManifest_EmptyManifest(t *testing.T) {
	cityMock := &mockCityRepo{
		city: &domain.City{ID: 1, Name: "Tbilisi", Country: "GE", IsActive: true},
	}
	manifestMock := &mockManifestRepo{items: nil}
	h := NewCityHandler(cityMock, manifestMock, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/1/download-manifest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data           []domain.DownloadManifestItem `json:"data"`
		TotalSizeBytes int64                         `json:"total_size_bytes"`
		TotalStories   int                           `json:"total_stories"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("expected empty data, got %d items", len(resp.Data))
	}
	if resp.TotalSizeBytes != 0 {
		t.Errorf("expected total_size_bytes=0, got %d", resp.TotalSizeBytes)
	}
}

func TestGetDownloadManifest_DefaultLanguage(t *testing.T) {
	cityMock := &mockCityRepo{
		city: &domain.City{ID: 1, Name: "Tbilisi", Country: "GE", IsActive: true},
	}
	manifestMock := &mockManifestRepo{items: []domain.DownloadManifestItem{}}
	h := NewCityHandler(cityMock, manifestMock, nil)
	r := setupCityRouter(h)

	// No language param — should default to "en"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/1/download-manifest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCityHandler_InvalidRequests(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)
	addTraceIDMiddleware(r, "trace-city-123")

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
			name:          "list cities invalid limit",
			method:        http.MethodGet,
			path:          "/api/v1/cities?limit=-1",
			expectedCode:  http.StatusBadRequest,
			expectedError: "limit must be a positive integer",
		},
		{
			name:          "get city invalid path id",
			method:        http.MethodGet,
			path:          "/api/v1/cities/abc",
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid id parameter",
		},
		{
			name:         "create city missing required fields",
			method:       http.MethodPost,
			path:         "/api/v1/admin/cities",
			body:         `{"name":"Tbilisi"}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"country":   "this field is required",
				"centerlat": "this field is required",
				"centerlng": "this field is required",
				"radiuskm":  "this field is required",
			},
		},
		{
			name:         "update city missing name",
			method:       http.MethodPut,
			path:         "/api/v1/admin/cities/1",
			body:         `{"country":"GE","center_lat":41.7,"center_lng":44.8,"radius_km":15}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"name": "this field is required",
			},
		},
		{
			name:          "delete city invalid path id",
			method:        http.MethodDelete,
			path:          "/api/v1/admin/cities/nope",
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid id parameter",
		},
		{
			name:          "download manifest invalid path id",
			method:        http.MethodGet,
			path:          "/api/v1/cities/nope/download-manifest",
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid id parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w *httptest.ResponseRecorder
			if tt.body == "" {
				w = executeRequest(r, httptest.NewRequest(tt.method, tt.path, nil))
			} else {
				w = executeJSONRequest(t, r, tt.method, tt.path, tt.body)
			}

			if tt.expectedField != nil {
				assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
					AllowExtraDetails: true,
					DetailsByField:    tt.expectedField,
				})
				return
			}
			if w.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d: %s", tt.expectedCode, w.Code, w.Body.String())
			}
			assertErrorResponse(t, w.Body.Bytes(), tt.expectedError, "")
		})
	}
}

func TestCreateCity_InvalidJSON(t *testing.T) {
	h := NewCityHandler(&mockCityRepo{}, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/admin/cities", "not json")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	assertErrorResponseContains(t, w.Body.Bytes(), "invalid character")
}

func TestCreateCity_UniqueViolation(t *testing.T) {
	mock := &mockCityRepo{createErr: repository.ErrConflict}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/admin/cities", `{"name":"Tbilisi","country":"GE","center_lat":41.7,"center_lng":44.8,"radius_km":15}`)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorResponse(t, w.Body.Bytes(), "city already exists", "")
}

func TestUpdateCity_Conflict(t *testing.T) {
	mock := &mockCityRepo{err: repository.ErrConflict}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	w := executeJSONRequest(t, r, http.MethodPut, "/api/v1/admin/cities/1", `{"name":"Tbilisi","country":"GE","center_lat":41.7,"center_lng":44.8,"radius_km":15}`)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorResponse(t, w.Body.Bytes(), "city already exists", "")
}

func TestDeleteCity_ForeignKeyViolation(t *testing.T) {
	mock := &mockCityRepo{deleteErr: repository.ErrInvalidReference}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/cities/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorResponse(t, w.Body.Bytes(), "referenced record does not exist", "")
}

func TestListCities_ExcludesInactiveCities(t *testing.T) {
	mock := &mockCityRepo{
		cities: []domain.City{
			{ID: 1, Name: "Tbilisi", Country: "GE", IsActive: true},
			{ID: 2, Name: "Batumi", Country: "GE", IsActive: false},
			{ID: 3, Name: "Kutaisi", Country: "GE", IsActive: true},
		},
	}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items []domain.City `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Errorf("expected 2 active cities, got %d", len(resp.Items))
	}
	for _, c := range resp.Items {
		if !c.IsActive {
			t.Errorf("expected only active cities, got inactive city %q", c.Name)
		}
	}
}

func TestListAdminCities_IncludesInactiveCities(t *testing.T) {
	mock := &mockCityRepo{
		cities: []domain.City{
			{ID: 1, Name: "Tbilisi", Country: "GE", IsActive: true},
			{ID: 2, Name: "Batumi", Country: "GE", IsActive: false},
			{ID: 3, Name: "Kutaisi", Country: "GE", IsActive: true},
		},
	}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items []domain.City `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Items) != 3 {
		t.Errorf("expected 3 cities (including inactive), got %d", len(resp.Items))
	}
}

func TestListAdminCities_IncludeDeletedOptIn(t *testing.T) {
	now := time.Now()
	mock := &mockCityRepo{
		cities: []domain.City{
			{ID: 1, Name: "Visible", Country: "GE", IsActive: true},
			{ID: 2, Name: "Deleted", Country: "GE", IsActive: false, DeletedAt: &now},
		},
	}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cities?include_deleted=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items []domain.City `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 cities with include_deleted=true, got %d", len(resp.Items))
	}
	if resp.Items[1].DeletedAt == nil {
		t.Fatal("expected deleted city to be present when include_deleted=true")
	}
}

func TestListAdminCities_InvalidIncludeDeleted(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cities?include_deleted=nope", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	assertErrorResponse(t, w.Body.Bytes(), "include_deleted must be a boolean", "")
}

func TestGetCity_InactiveCityReturnsNotFound(t *testing.T) {
	mock := &mockCityRepo{
		city: &domain.City{ID: 1, Name: "Batumi", Country: "GE", IsActive: false},
	}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for inactive city, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCity_ValidationErrors(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	validBase := func() map[string]interface{} {
		return map[string]interface{}{
			"name":       "Tbilisi",
			"country":    "GE",
			"center_lat": 41.7,
			"center_lng": 44.8,
			"radius_km":  15,
		}
	}

	tests := []struct {
		name          string
		modify        func(m map[string]interface{})
		expectedField string
		expectedMsg   string
	}{
		{
			name:          "latitude too high",
			modify:        func(m map[string]interface{}) { m["center_lat"] = 91.0 },
			expectedField: "latitude",
			expectedMsg:   "must be between -90 and 90",
		},
		{
			name:          "latitude too low",
			modify:        func(m map[string]interface{}) { m["center_lat"] = -91.0 },
			expectedField: "latitude",
			expectedMsg:   "must be between -90 and 90",
		},
		{
			name:          "longitude too high",
			modify:        func(m map[string]interface{}) { m["center_lng"] = 181.0 },
			expectedField: "longitude",
			expectedMsg:   "must be between -180 and 180",
		},
		{
			name:          "longitude too low",
			modify:        func(m map[string]interface{}) { m["center_lng"] = -181.0 },
			expectedField: "longitude",
			expectedMsg:   "must be between -180 and 180",
		},
		{
			name:          "radius zero",
			modify:        func(m map[string]interface{}) { m["radius_km"] = 0 },
			expectedField: "radiuskm",
			expectedMsg:   "this field is required",
		},
		{
			name:          "radius negative",
			modify:        func(m map[string]interface{}) { m["radius_km"] = -5 },
			expectedField: "radius_km",
			expectedMsg:   "must be at least 0.1",
		},
		{
			name:          "radius too small",
			modify:        func(m map[string]interface{}) { m["radius_km"] = 0.05 },
			expectedField: "radius_km",
			expectedMsg:   "must be at least 0.1",
		},
		{
			name:          "radius exceeds max",
			modify:        func(m map[string]interface{}) { m["radius_km"] = 1001 },
			expectedField: "radius_km",
			expectedMsg:   "must not exceed 1000",
		},
		{
			name: "name too long",
			modify: func(m map[string]interface{}) {
				m["name"] = strings.Repeat("a", 201)
			},
			expectedField: "name",
			expectedMsg:   "must not exceed 200 characters",
		},
		{
			name: "country too long",
			modify: func(m map[string]interface{}) {
				m["country"] = strings.Repeat("X", 101)
			},
			expectedField: "country",
			expectedMsg:   "must not exceed 100 characters",
		},
		{
			name:          "negative download size",
			modify:        func(m map[string]interface{}) { m["download_size_mb"] = -1 },
			expectedField: "download_size_mb",
			expectedMsg:   "must be non-negative",
		},
		{
			name: "name_ru too long",
			modify: func(m map[string]interface{}) {
				long := strings.Repeat("б", 201)
				m["name_ru"] = long
			},
			expectedField: "name_ru",
			expectedMsg:   "must not exceed 200 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := validBase()
			tt.modify(body)
			w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/admin/cities", body)
			assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
				AllowExtraDetails: true,
				DetailsByField: map[string]string{
					tt.expectedField: tt.expectedMsg,
				},
			})
		})
	}
}

func TestUpdateCity_ValidationErrors(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	tests := []struct {
		name          string
		body          string
		expectedField string
		expectedMsg   string
	}{
		{
			name:          "latitude out of range",
			body:          `{"name":"X","country":"GE","center_lat":95,"center_lng":44.8,"radius_km":15}`,
			expectedField: "latitude",
			expectedMsg:   "must be between -90 and 90",
		},
		{
			name:          "longitude out of range",
			body:          `{"name":"X","country":"GE","center_lat":41.7,"center_lng":200,"radius_km":15}`,
			expectedField: "longitude",
			expectedMsg:   "must be between -180 and 180",
		},
		{
			name:          "radius zero",
			body:          `{"name":"X","country":"GE","center_lat":41.7,"center_lng":44.8,"radius_km":0}`,
			expectedField: "radiuskm",
			expectedMsg:   "this field is required",
		},
		{
			name:          "radius exceeds max",
			body:          `{"name":"X","country":"GE","center_lat":41.7,"center_lng":44.8,"radius_km":1001}`,
			expectedField: "radius_km",
			expectedMsg:   "must not exceed 1000",
		},
		{
			name:          "negative download size",
			body:          `{"name":"X","country":"GE","center_lat":41.7,"center_lng":44.8,"radius_km":15,"download_size_mb":-5}`,
			expectedField: "download_size_mb",
			expectedMsg:   "must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := executeJSONRequest(t, r, http.MethodPut, "/api/v1/admin/cities/1", tt.body)
			assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
				AllowExtraDetails: true,
				DetailsByField: map[string]string{
					tt.expectedField: tt.expectedMsg,
				},
			})
		})
	}
}

func TestGetDownloadManifest_InvalidLanguage(t *testing.T) {
	cityMock := &mockCityRepo{
		city: &domain.City{ID: 1, Name: "Tbilisi", Country: "GE", CenterLat: 41.7, CenterLng: 44.8, RadiusKm: 15, IsActive: true},
	}
	h := NewCityHandler(cityMock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/1/download-manifest?language=fr", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported language, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorResponse(t, w.Body.Bytes(), "unsupported language; supported: en, ru", "")
}

func TestGetDownloadManifest_ValidLanguageRu(t *testing.T) {
	cityMock := &mockCityRepo{
		city: &domain.City{ID: 1, Name: "Tbilisi", Country: "GE", CenterLat: 41.7, CenterLng: 44.8, RadiusKm: 15, IsActive: true},
	}
	manifestMock := &mockManifestRepo{items: []domain.DownloadManifestItem{}}
	h := NewCityHandler(cityMock, manifestMock, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/1/download-manifest?language=ru", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for language=ru, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetDownloadManifest_RussianCityNameWhenLanguageRu(t *testing.T) {
	nameRu := "Тбилиси"
	audioURL := "https://s3.example.com/audio/story_1.mp3"
	dur := int16(120)
	cityMock := &mockCityRepo{
		city: &domain.City{ID: 1, Name: "Tbilisi", NameRu: &nameRu, Country: "GE", CenterLat: 41.7, CenterLng: 44.8, RadiusKm: 15, IsActive: true},
	}
	manifestMock := &mockManifestRepo{
		items: []domain.DownloadManifestItem{
			{StoryID: 1, POIID: 10, POIName: "Крепость Нарикала", AudioURL: &audioURL, DurationSec: &dur, FileSizeBytes: 2880000},
		},
	}
	h := NewCityHandler(cityMock, manifestMock, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/1/download-manifest?language=ru", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data     []domain.DownloadManifestItem `json:"data"`
		CityName string                        `json:"city_name"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.CityName != "Тбилиси" {
		t.Errorf("expected Russian city_name 'Тбилиси', got %q", resp.CityName)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Data))
	}
	if resp.Data[0].POIName != "Крепость Нарикала" {
		t.Errorf("expected Russian POI name, got %q", resp.Data[0].POIName)
	}
}

func TestGetDownloadManifest_EnglishCityNameWhenLanguageEn(t *testing.T) {
	nameRu := "Тбилиси"
	cityMock := &mockCityRepo{
		city: &domain.City{ID: 1, Name: "Tbilisi", NameRu: &nameRu, Country: "GE", CenterLat: 41.7, CenterLng: 44.8, RadiusKm: 15, IsActive: true},
	}
	manifestMock := &mockManifestRepo{items: []domain.DownloadManifestItem{}}
	h := NewCityHandler(cityMock, manifestMock, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/1/download-manifest?language=en", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		CityName string `json:"city_name"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.CityName != "Tbilisi" {
		t.Errorf("expected English city_name 'Tbilisi', got %q", resp.CityName)
	}
}

func TestGetDownloadManifest_FallbackCityNameWhenNameRuMissing(t *testing.T) {
	cityMock := &mockCityRepo{
		city: &domain.City{ID: 1, Name: "Tbilisi", Country: "GE", CenterLat: 41.7, CenterLng: 44.8, RadiusKm: 15, IsActive: true},
	}
	manifestMock := &mockManifestRepo{items: []domain.DownloadManifestItem{}}
	h := NewCityHandler(cityMock, manifestMock, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/1/download-manifest?language=ru", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		CityName string `json:"city_name"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.CityName != "Tbilisi" {
		t.Errorf("expected fallback to English city_name 'Tbilisi' when name_ru missing, got %q", resp.CityName)
	}
}

func TestGetDownloadManifest_InactiveCityReturnsNotFound(t *testing.T) {
	cityMock := &mockCityRepo{
		city: &domain.City{ID: 1, Name: "Batumi", Country: "GE", IsActive: false},
	}
	h := NewCityHandler(cityMock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/1/download-manifest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for inactive city manifest, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Visibility boundary regression tests ---

func TestVisibility_PublicListExcludesInactiveAcrossPages(t *testing.T) {
	// Seed a mix: 3 active, 2 inactive, interleaved by ID.
	cities := []domain.City{
		{ID: 1, Name: "Active1", Country: "GE", IsActive: true},
		{ID: 2, Name: "Inactive1", Country: "GE", IsActive: false},
		{ID: 3, Name: "Active2", Country: "GE", IsActive: true},
		{ID: 4, Name: "Inactive2", Country: "GE", IsActive: false},
		{ID: 5, Name: "Active3", Country: "GE", IsActive: true},
	}
	mock := &mockCityRepo{cities: cities}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	// Page 1: limit=2 — should return exactly 2 active cities, no inactive.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities?limit=2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("page 1: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var page1 struct {
		Items   []domain.City `json:"items"`
		HasMore bool          `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &page1); err != nil {
		t.Fatalf("page 1 unmarshal: %v", err)
	}
	if len(page1.Items) != 2 {
		t.Errorf("page 1: expected 2 active items, got %d", len(page1.Items))
	}
	for _, c := range page1.Items {
		if !c.IsActive {
			t.Errorf("page 1: inactive city %q leaked into public listing", c.Name)
		}
	}
	if !page1.HasMore {
		t.Error("page 1: expected has_more=true since 3 active cities exist")
	}
}

func TestVisibility_AdminListIncludesAllStatusesAcrossPages(t *testing.T) {
	cities := []domain.City{
		{ID: 1, Name: "Active1", Country: "GE", IsActive: true},
		{ID: 2, Name: "Inactive1", Country: "GE", IsActive: false},
		{ID: 3, Name: "Active2", Country: "GE", IsActive: true},
		{ID: 4, Name: "Inactive2", Country: "GE", IsActive: false},
		{ID: 5, Name: "Active3", Country: "GE", IsActive: true},
	}
	mock := &mockCityRepo{cities: cities}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	// Admin listing with limit=3 should include both active and inactive.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cities?limit=3", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items   []domain.City `json:"items"`
		HasMore bool          `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 3 {
		t.Errorf("expected 3 items on admin page, got %d", len(resp.Items))
	}
	if !resp.HasMore {
		t.Error("expected has_more=true for admin listing with 5 total cities")
	}

	// Verify the admin page includes at least one inactive city.
	hasInactive := false
	for _, c := range resp.Items {
		if !c.IsActive {
			hasInactive = true
		}
	}
	if !hasInactive {
		t.Error("admin listing should include inactive cities but none were returned")
	}
}

func TestVisibility_PublicAndAdminReturnDifferentCounts(t *testing.T) {
	cities := []domain.City{
		{ID: 1, Name: "Visible", Country: "GE", IsActive: true},
		{ID: 2, Name: "Hidden", Country: "GE", IsActive: false},
		{ID: 3, Name: "AlsoVisible", Country: "GE", IsActive: true},
		{ID: 4, Name: "AlsoHidden", Country: "GE", IsActive: false},
	}
	mock := &mockCityRepo{cities: cities}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	// Public endpoint
	pubReq := httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil)
	pubW := httptest.NewRecorder()
	r.ServeHTTP(pubW, pubReq)

	var pubResp struct {
		Items []domain.City `json:"items"`
	}
	if err := json.Unmarshal(pubW.Body.Bytes(), &pubResp); err != nil {
		t.Fatalf("public unmarshal: %v", err)
	}

	// Admin endpoint
	adminReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cities", nil)
	adminW := httptest.NewRecorder()
	r.ServeHTTP(adminW, adminReq)

	var adminResp struct {
		Items []domain.City `json:"items"`
	}
	if err := json.Unmarshal(adminW.Body.Bytes(), &adminResp); err != nil {
		t.Fatalf("admin unmarshal: %v", err)
	}

	if len(pubResp.Items) != 2 {
		t.Errorf("public: expected 2 active cities, got %d", len(pubResp.Items))
	}
	if len(adminResp.Items) != 4 {
		t.Errorf("admin: expected 4 total cities, got %d", len(adminResp.Items))
	}
	if len(pubResp.Items) >= len(adminResp.Items) {
		t.Error("public listing should return fewer items than admin listing when inactive cities exist")
	}
}

func TestVisibility_AllInactiveCities_PublicReturnsEmpty(t *testing.T) {
	cities := []domain.City{
		{ID: 1, Name: "Draft1", Country: "GE", IsActive: false},
		{ID: 2, Name: "Draft2", Country: "GE", IsActive: false},
	}
	mock := &mockCityRepo{cities: cities}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Items []domain.City `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("public listing should return 0 cities when all are inactive, got %d", len(resp.Items))
	}
}

func TestVisibility_DeactivatedCityBecomesNotFoundOnPublicGet(t *testing.T) {
	// Simulates a city that was deactivated — public GET should return 404.
	mock := &mockCityRepo{
		city: &domain.City{ID: 10, Name: "Deactivated", Country: "GE", IsActive: false},
	}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for deactivated city on public GET, got %d", w.Code)
	}
}

func TestVisibility_DeactivatedCityManifestReturnsNotFound(t *testing.T) {
	mock := &mockCityRepo{
		city: &domain.City{ID: 10, Name: "Deactivated", Country: "GE", IsActive: false},
	}
	manifestMock := &mockManifestRepo{
		items: []domain.DownloadManifestItem{
			{StoryID: 1, POIID: 1, POIName: "POI", FileSizeBytes: 1000},
		},
	}
	h := NewCityHandler(mock, manifestMock, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/10/download-manifest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for deactivated city manifest, got %d", w.Code)
	}
}

// --- Soft-delete and restore tests ---

func TestDeleteCity_SoftDelete_Idempotent(t *testing.T) {
	// Deleting an already-deleted city should succeed (idempotent).
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	// First delete
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/cities/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first delete: expected 200, got %d", w.Code)
	}

	// Second delete (idempotent)
	req2 := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/cities/1", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second delete (idempotent): expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestRestoreCity_Success(t *testing.T) {
	now := time.Now()
	mock := &mockCityRepo{
		restoreOut: &domain.City{ID: 1, Name: "Tbilisi", Country: "GE", IsActive: true, CreatedAt: now, UpdatedAt: now},
	}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities/1/restore", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data domain.City `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.Name != "Tbilisi" {
		t.Errorf("expected name=Tbilisi, got %q", resp.Data.Name)
	}
	if resp.Data.DeletedAt != nil {
		t.Error("expected deleted_at to be nil after restore")
	}
}

func TestRestoreCity_NotFound(t *testing.T) {
	mock := &mockCityRepo{restoreErr: repository.ErrNotFound}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities/999/restore", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	assertErrorResponse(t, w.Body.Bytes(), "city not found or not deleted", "")
}

func TestRestoreCity_InvalidID(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities/abc/restore", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRestoreCity_Conflict(t *testing.T) {
	mock := &mockCityRepo{restoreErr: repository.ErrConflict}
	h := NewCityHandler(mock, &mockManifestRepo{}, nil)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities/1/restore", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
	assertErrorResponse(t, w.Body.Bytes(), "city already exists", "")
}

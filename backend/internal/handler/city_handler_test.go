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

// mockCityRepo implements CityRepository for testing.
type mockCityRepo struct {
	cities    []domain.City
	city      *domain.City
	err       error
	createErr error
	deleteErr error
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

func (m *mockCityRepo) GetByID(_ context.Context, _ int) (*domain.City, error) {
	if m.err != nil {
		return nil, m.err
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

func (m *mockCityRepo) Delete(_ context.Context, _ int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func setupCityRouter(h *CityHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/cities", h.ListCities)
	r.GET("/api/v1/cities/:id", h.GetCity)
	r.POST("/api/v1/admin/cities", h.CreateCity)
	r.PUT("/api/v1/admin/cities/:id", h.UpdateCity)
	r.DELETE("/api/v1/admin/cities/:id", h.DeleteCity)
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
	h := NewCityHandler(mock)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []domain.City `json:"data"`
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
		t.Errorf("expected 2 cities, got %d", len(resp.Data))
	}
	if resp.Meta.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Meta.Total)
	}
	if resp.Meta.Page != 1 {
		t.Errorf("expected page=1, got %d", resp.Meta.Page)
	}
	if resp.Meta.PerPage != 20 {
		t.Errorf("expected per_page=20, got %d", resp.Meta.PerPage)
	}
}

func TestListCities_EmptyResult(t *testing.T) {
	mock := &mockCityRepo{cities: nil}
	h := NewCityHandler(mock)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data []domain.City `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp.Data))
	}
}

func TestListCities_Pagination(t *testing.T) {
	cities := make([]domain.City, 25)
	for i := range cities {
		cities[i] = domain.City{ID: i + 1, Name: "City", Country: "GE"}
	}
	mock := &mockCityRepo{cities: cities}
	h := NewCityHandler(mock)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities?page=2&per_page=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data []domain.City `json:"data"`
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
		t.Errorf("expected 10 cities on page 2, got %d", len(resp.Data))
	}
	if resp.Meta.Total != 25 {
		t.Errorf("expected total=25, got %d", resp.Meta.Total)
	}
	if resp.Meta.Page != 2 {
		t.Errorf("expected page=2, got %d", resp.Meta.Page)
	}
}

func TestListCities_ServiceError(t *testing.T) {
	mock := &mockCityRepo{err: repository.ErrNotFound}
	h := NewCityHandler(mock)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGetCity_Success(t *testing.T) {
	mock := &mockCityRepo{
		city: &domain.City{ID: 1, Name: "Tbilisi", Country: "GE", CenterLat: 41.7, CenterLng: 44.8, RadiusKm: 15, IsActive: true},
	}
	h := NewCityHandler(mock)
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
	h := NewCityHandler(mock)
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
	h := NewCityHandler(mock)
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
	h := NewCityHandler(mock)
	r := setupCityRouter(h)

	body := `{"name":"Tbilisi","country":"GE","center_lat":41.7151,"center_lng":44.8271,"radius_km":15}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

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
	h := NewCityHandler(mock)
	r := setupCityRouter(h)

	body := `{"name":"Tbilisi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCity_WithOptionalFields(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock)
	r := setupCityRouter(h)

	isActive := false
	body, _ := json.Marshal(createCityRequest{
		Name:      "Batumi",
		Country:   "GE",
		CenterLat: 41.6,
		CenterLng: 41.6,
		RadiusKm:  10,
		IsActive:  &isActive,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

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
	h := NewCityHandler(mock)
	r := setupCityRouter(h)

	body := `{"name":"Tbilisi Updated","country":"GE","center_lat":41.7151,"center_lng":44.8271,"radius_km":20}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/cities/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
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
	if resp.Data.Name != "Tbilisi Updated" {
		t.Errorf("expected name='Tbilisi Updated', got %q", resp.Data.Name)
	}
}

func TestUpdateCity_NotFound(t *testing.T) {
	mock := &mockCityRepo{err: repository.ErrNotFound}
	h := NewCityHandler(mock)
	r := setupCityRouter(h)

	body := `{"name":"Tbilisi","country":"GE","center_lat":41.7,"center_lng":44.8,"radius_km":15}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/cities/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteCity_Success(t *testing.T) {
	mock := &mockCityRepo{}
	h := NewCityHandler(mock)
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
	h := NewCityHandler(mock)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/cities/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

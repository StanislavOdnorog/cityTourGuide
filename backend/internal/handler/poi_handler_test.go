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

// mockPOIRepo implements POIRepository for testing.
type mockPOIRepo struct {
	pois      []domain.POI
	poi       *domain.POI
	err       error
	createErr error
	deleteErr error
	// captured args
	calledCityID int
	calledStatus *domain.POIStatus
	calledType   *domain.POIType
}

func (m *mockPOIRepo) Create(_ context.Context, poi *domain.POI) (*domain.POI, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	poi.ID = 1
	poi.CreatedAt = time.Now()
	poi.UpdatedAt = time.Now()
	return poi, nil
}

func (m *mockPOIRepo) GetByID(_ context.Context, _ int) (*domain.POI, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.poi, nil
}

func (m *mockPOIRepo) GetByCityID(_ context.Context, cityID int, status *domain.POIStatus, poiType *domain.POIType) ([]domain.POI, error) {
	m.calledCityID = cityID
	m.calledStatus = status
	m.calledType = poiType
	if m.err != nil {
		return nil, m.err
	}
	return m.pois, nil
}

func (m *mockPOIRepo) Update(_ context.Context, poi *domain.POI) (*domain.POI, error) {
	if m.err != nil {
		return nil, m.err
	}
	poi.UpdatedAt = time.Now()
	return poi, nil
}

func (m *mockPOIRepo) Delete(_ context.Context, _ int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func setupPOIRouter(h *POIHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/pois", h.ListPOIs)
	r.GET("/api/v1/pois/:id", h.GetPOI)
	r.POST("/api/v1/admin/pois", h.CreatePOI)
	r.PUT("/api/v1/admin/pois/:id", h.UpdatePOI)
	r.DELETE("/api/v1/admin/pois/:id", h.DeletePOI)
	return r
}

func TestListPOIs_Success(t *testing.T) {
	mock := &mockPOIRepo{
		pois: []domain.POI{
			{ID: 1, CityID: 1, Name: "Narikala", Type: domain.POITypeMonument, Status: domain.POIStatusActive},
			{ID: 2, CityID: 1, Name: "Bridge of Peace", Type: domain.POITypeBridge, Status: domain.POIStatusActive},
		},
	}
	h := NewPOIHandler(mock)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pois?city_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []domain.POI `json:"data"`
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
		t.Errorf("expected 2 POIs, got %d", len(resp.Data))
	}
	if resp.Meta.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Meta.Total)
	}
}

func TestListPOIs_MissingCityID(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pois", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["error"] != "city_id is required" {
		t.Errorf("expected 'city_id is required', got %q", resp["error"])
	}
}

func TestListPOIs_WithFilters(t *testing.T) {
	mock := &mockPOIRepo{pois: []domain.POI{}}
	h := NewPOIHandler(mock)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pois?city_id=1&status=active&type=monument", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if mock.calledCityID != 1 {
		t.Errorf("expected city_id=1, got %d", mock.calledCityID)
	}
	if mock.calledStatus == nil || *mock.calledStatus != domain.POIStatusActive {
		t.Errorf("expected status=active, got %v", mock.calledStatus)
	}
	if mock.calledType == nil || *mock.calledType != domain.POITypeMonument {
		t.Errorf("expected type=monument, got %v", mock.calledType)
	}
}

func TestListPOIs_Pagination(t *testing.T) {
	pois := make([]domain.POI, 30)
	for i := range pois {
		pois[i] = domain.POI{ID: i + 1, CityID: 1, Name: "POI", Type: domain.POITypeBuilding, Status: domain.POIStatusActive}
	}
	mock := &mockPOIRepo{pois: pois}
	h := NewPOIHandler(mock)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pois?city_id=1&page=2&per_page=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data []domain.POI `json:"data"`
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
		t.Errorf("expected 10 POIs on page 2, got %d", len(resp.Data))
	}
	if resp.Meta.Total != 30 {
		t.Errorf("expected total=30, got %d", resp.Meta.Total)
	}
}

func TestGetPOI_Success(t *testing.T) {
	mock := &mockPOIRepo{
		poi: &domain.POI{ID: 1, CityID: 1, Name: "Narikala", Lat: 41.68, Lng: 44.80, Type: domain.POITypeMonument, Status: domain.POIStatusActive},
	}
	h := NewPOIHandler(mock)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pois/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data domain.POI `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.Name != "Narikala" {
		t.Errorf("expected name=Narikala, got %q", resp.Data.Name)
	}
}

func TestGetPOI_NotFound(t *testing.T) {
	mock := &mockPOIRepo{err: repository.ErrNotFound}
	h := NewPOIHandler(mock)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pois/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCreatePOI_Success(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock)
	r := setupPOIRouter(h)

	body := `{"city_id":1,"name":"Narikala","lat":41.68,"lng":44.80,"type":"monument"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/pois", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data domain.POI `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.Name != "Narikala" {
		t.Errorf("expected name=Narikala, got %q", resp.Data.Name)
	}
	if resp.Data.InterestScore != 50 {
		t.Errorf("expected default interest_score=50, got %d", resp.Data.InterestScore)
	}
	if resp.Data.Status != domain.POIStatusActive {
		t.Errorf("expected default status=active, got %q", resp.Data.Status)
	}
}

func TestCreatePOI_MissingRequired(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock)
	r := setupPOIRouter(h)

	body := `{"name":"Narikala"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/pois", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePOI_WithOptionalFields(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock)
	r := setupPOIRouter(h)

	score := int16(80)
	status := domain.POIStatusDisabled
	body, _ := json.Marshal(createPOIRequest{
		CityID:        1,
		Name:          "Test",
		Lat:           41.7,
		Lng:           44.8,
		Type:          domain.POITypeMuseum,
		InterestScore: &score,
		Status:        &status,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/pois", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data domain.POI `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Data.InterestScore != 80 {
		t.Errorf("expected interest_score=80, got %d", resp.Data.InterestScore)
	}
	if resp.Data.Status != domain.POIStatusDisabled {
		t.Errorf("expected status=disabled, got %q", resp.Data.Status)
	}
}

func TestUpdatePOI_Success(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock)
	r := setupPOIRouter(h)

	body := `{"city_id":1,"name":"Updated","lat":41.68,"lng":44.80,"type":"monument"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/pois/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdatePOI_NotFound(t *testing.T) {
	mock := &mockPOIRepo{err: repository.ErrNotFound}
	h := NewPOIHandler(mock)
	r := setupPOIRouter(h)

	body := `{"city_id":1,"name":"Test","lat":41.7,"lng":44.8,"type":"monument"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/pois/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeletePOI_Success(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/pois/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestDeletePOI_NotFound(t *testing.T) {
	mock := &mockPOIRepo{deleteErr: repository.ErrNotFound}
	h := NewPOIHandler(mock)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/pois/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

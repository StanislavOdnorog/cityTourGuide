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
	calledSort   repository.ListSort
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

func (m *mockPOIRepo) ListByCityID(_ context.Context, cityID int, status *domain.POIStatus, poiType *domain.POIType, page domain.PageRequest, sort repository.ListSort) (*domain.PageResponse[domain.POI], error) {
	m.calledCityID = cityID
	m.calledStatus = status
	m.calledType = poiType
	m.calledSort = sort
	if m.err != nil {
		return nil, m.err
	}
	items := m.pois
	hasMore := false
	if len(items) > page.Limit {
		items = items[:page.Limit]
		hasMore = true
	}
	return &domain.PageResponse[domain.POI]{
		Items:   items,
		HasMore: hasMore,
	}, nil
}

func TestListPOIs_UsesDefaultSort(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pois?city_id=1&sort_by=interest_score&sort_dir=desc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if mock.calledSort.By != "id" || mock.calledSort.Dir != repository.SortDirAsc {
		t.Fatalf("expected default sort {id asc}, got %+v", mock.calledSort)
	}
}

func TestListAdminPOIs_SortParams(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/pois?city_id=1&sort_by=interest_score&sort_dir=desc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if mock.calledSort.By != "interest_score" || mock.calledSort.Dir != repository.SortDirDesc {
		t.Fatalf("expected sort {interest_score desc}, got %+v", mock.calledSort)
	}
}

func TestListAdminPOIs_InvalidSortBy(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/pois?city_id=1&sort_by=deleted_at", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListAdminPOIs_InvalidSortDir(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/pois?city_id=1&sort_dir=sideways", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
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
	r.GET("/api/v1/admin/pois", h.ListAdminPOIs)
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
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pois?city_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items      []domain.POI `json:"items"`
		NextCursor string       `json:"next_cursor"`
		HasMore    bool         `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Errorf("expected 2 POIs, got %d", len(resp.Items))
	}
	if resp.HasMore {
		t.Error("expected has_more=false")
	}
}

func TestListPOIs_EmptyCityReturnsEmptyArray(t *testing.T) {
	mock := &mockPOIRepo{
		pois: nil,
	}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pois?city_id=999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items   json.RawMessage `json:"items"`
		HasMore bool            `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if string(resp.Items) != "[]" {
		t.Fatalf("expected items to be [], got %s", string(resp.Items))
	}
	if resp.HasMore {
		t.Error("expected has_more=false")
	}
}

func TestListPOIs_MissingCityID(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock, nil)
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
	h := NewPOIHandler(mock, nil)
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

func TestListPOIs_InvalidStatus(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	w := executeRequest(r, httptest.NewRequest(http.MethodGet, "/api/v1/pois?city_id=1&status=archived", nil))
	assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
		OrderedDetails: []validationDetail{{
			Field:   "status",
			Message: "must be one of: active, disabled, pending_review",
		}},
	})
}

func TestListPOIs_InvalidType(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	w := executeRequest(r, httptest.NewRequest(http.MethodGet, "/api/v1/pois?city_id=1&type=castle", nil))
	assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
		OrderedDetails: []validationDetail{{
			Field:   "type",
			Message: "must be one of: building, street, park, monument, church, bridge, square, museum, district, other",
		}},
	})
}

func TestListPOIs_Pagination(t *testing.T) {
	pois := make([]domain.POI, 30)
	for i := range pois {
		pois[i] = domain.POI{ID: i + 1, CityID: 1, Name: "POI", Type: domain.POITypeBuilding, Status: domain.POIStatusActive}
	}
	mock := &mockPOIRepo{pois: pois}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pois?city_id=1&limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Items      []domain.POI `json:"items"`
		NextCursor string       `json:"next_cursor"`
		HasMore    bool         `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Items) != 10 {
		t.Errorf("expected 10 POIs, got %d", len(resp.Items))
	}
	if !resp.HasMore {
		t.Error("expected has_more=true")
	}
}

func TestGetPOI_Success(t *testing.T) {
	mock := &mockPOIRepo{
		poi: &domain.POI{ID: 1, CityID: 1, Name: "Narikala", Lat: 41.68, Lng: 44.80, Type: domain.POITypeMonument, Status: domain.POIStatusActive},
	}
	h := NewPOIHandler(mock, nil)
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
	h := NewPOIHandler(mock, nil)
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
	h := NewPOIHandler(mock, nil)
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
	h := NewPOIHandler(mock, nil)
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
	h := NewPOIHandler(mock, nil)
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

func TestCreatePOI_InvalidValidation(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	longName := bytes.Repeat([]byte("a"), 501)
	longAddress := bytes.Repeat([]byte("b"), 1001)
	body := map[string]any{
		"city_id":        1,
		"name":           string(longName),
		"lat":            91,
		"lng":            181,
		"type":           "castle",
		"interest_score": 101,
		"address":        string(longAddress),
		"status":         "archived",
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/admin/pois", payload)
	assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
		AllowExtraDetails: true,
		DetailsByField: map[string]string{
			"name":          "must not exceed 500 characters",
			"lat":           "must be at most 90",
			"lng":           "must be at most 180",
			"type":          "must be one of: building street park monument church bridge square museum district other",
			"interestscore": "must be at most 100",
			"address":       "must not exceed 1000 characters",
			"status":        "must be one of: active disabled pending_review",
		},
	})
}

func TestUpdatePOI_Success(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock, nil)
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
	h := NewPOIHandler(mock, nil)
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

func TestUpdatePOI_InvalidValidation(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	body := `{"city_id":1,"name":"Updated","lat":-91,"lng":44.80,"type":"invalid","interest_score":-1,"status":"bad"}`
	w := executeJSONRequest(t, r, http.MethodPut, "/api/v1/admin/pois/1", body)
	assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
		AllowExtraDetails: true,
		DetailsByField: map[string]string{
			"lat":           "must be at least -90",
			"type":          "must be one of: building street park monument church bridge square museum district other",
			"interestscore": "must be at least 0",
			"status":        "must be one of: active disabled pending_review",
		},
	})
}

func TestDeletePOI_Success(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock, nil)
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
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/pois/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestPOIHandler_InvalidRequests(t *testing.T) {
	mock := &mockPOIRepo{}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)
	addTraceIDMiddleware(r, "trace-poi-123")

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
			name:          "list pois invalid city id",
			method:        http.MethodGet,
			path:          "/api/v1/pois?city_id=bad",
			expectedCode:  http.StatusBadRequest,
			expectedError: "city_id must be a positive integer",
		},
		{
			name:          "list pois invalid limit",
			method:        http.MethodGet,
			path:          "/api/v1/pois?city_id=1&limit=101",
			expectedCode:  http.StatusBadRequest,
			expectedError: "limit must not exceed 100",
		},
		{
			name:          "get poi invalid path id",
			method:        http.MethodGet,
			path:          "/api/v1/pois/abc",
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid id parameter",
		},
		{
			name:         "create poi missing required fields",
			method:       http.MethodPost,
			path:         "/api/v1/admin/pois",
			body:         `{"name":"POI"}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"cityid": "this field is required",
				"type":   "this field is required",
			},
		},
		{
			name:         "update poi invalid coordinates",
			method:       http.MethodPut,
			path:         "/api/v1/admin/pois/1",
			body:         `{"city_id":1,"name":"POI","lat":91,"lng":44.8,"type":"monument"}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"lat": "must be at most 90",
			},
		},
		{
			name:          "delete poi invalid path id",
			method:        http.MethodDelete,
			path:          "/api/v1/admin/pois/zero",
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

func TestCreatePOI_InvalidJSON(t *testing.T) {
	h := NewPOIHandler(&mockPOIRepo{}, nil)
	r := setupPOIRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/admin/pois", "not json")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	assertErrorResponseContains(t, w.Body.Bytes(), "invalid character")
}

func TestCreatePOI_ForeignKeyViolation(t *testing.T) {
	mock := &mockPOIRepo{createErr: repository.ErrInvalidReference}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/admin/pois", `{"city_id":999,"name":"Narikala","lat":41.68,"lng":44.80,"type":"monument"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorResponse(t, w.Body.Bytes(), "referenced record does not exist", "")
}

func TestCreatePOI_UniqueViolation(t *testing.T) {
	mock := &mockPOIRepo{createErr: repository.ErrConflict}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/admin/pois", `{"city_id":1,"name":"Narikala","lat":41.68,"lng":44.80,"type":"monument"}`)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorResponse(t, w.Body.Bytes(), "POI already exists", "")
}

func TestUpdatePOI_ForeignKeyViolation(t *testing.T) {
	mock := &mockPOIRepo{err: repository.ErrInvalidReference}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	w := executeJSONRequest(t, r, http.MethodPut, "/api/v1/admin/pois/1", `{"city_id":999,"name":"Test","lat":41.7,"lng":44.8,"type":"monument"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorResponse(t, w.Body.Bytes(), "referenced record does not exist", "")
}

func TestDeletePOI_ForeignKeyViolation(t *testing.T) {
	mock := &mockPOIRepo{deleteErr: repository.ErrInvalidReference}
	h := NewPOIHandler(mock, nil)
	r := setupPOIRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/pois/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorResponse(t, w.Body.Bytes(), "referenced record does not exist", "")
}

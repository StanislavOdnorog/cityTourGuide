package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

type mockReportRepo struct {
	reports      []domain.Report
	adminReports []domain.AdminReportListItem
	report       *domain.Report
	err          error
	// captured args
	calledStatus string
	calledSort   repository.ListSort
}

func (m *mockReportRepo) Create(_ context.Context, _ int, _ string, _ domain.ReportType, _ *string, _, _ *float64) (*domain.Report, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.report != nil {
		return m.report, nil
	}
	now := time.Now()
	return &domain.Report{ID: 1, Status: domain.ReportStatusNew, CreatedAt: now}, nil
}

func (m *mockReportRepo) GetByID(_ context.Context, _ int) (*domain.Report, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.report, nil
}

func (m *mockReportRepo) List(_ context.Context, _ string, page domain.PageRequest) (*domain.PageResponse[domain.Report], error) {
	if m.err != nil {
		return nil, m.err
	}
	items := m.reports
	hasMore := false
	if len(items) > page.Limit {
		items = items[:page.Limit]
		hasMore = true
	}
	return &domain.PageResponse[domain.Report]{
		Items:   items,
		HasMore: hasMore,
	}, nil
}

func (m *mockReportRepo) ListAdmin(_ context.Context, status string, page domain.PageRequest, sort repository.ListSort) (*domain.PageResponse[domain.AdminReportListItem], error) {
	m.calledStatus = status
	m.calledSort = sort
	if m.err != nil {
		return nil, m.err
	}
	items := m.adminReports
	hasMore := false
	if len(items) > page.Limit {
		items = items[:page.Limit]
		hasMore = true
	}
	return &domain.PageResponse[domain.AdminReportListItem]{
		Items:   items,
		HasMore: hasMore,
	}, nil
}

func TestListReports_SortParams(t *testing.T) {
	mock := &mockReportRepo{}
	h := NewReportHandler(mock, &mockReportModerationService{}, nil)
	r := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports?sort_by=created_at&sort_dir=desc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if mock.calledSort.By != "created_at" || mock.calledSort.Dir != repository.SortDirDesc {
		t.Fatalf("expected sort {created_at desc}, got %+v", mock.calledSort)
	}
}

func TestListReports_InvalidSortBy(t *testing.T) {
	mock := &mockReportRepo{}
	h := NewReportHandler(mock, &mockReportModerationService{}, nil)
	r := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports?sort_by=poi_id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListReports_InvalidSortDir(t *testing.T) {
	mock := &mockReportRepo{}
	h := NewReportHandler(mock, &mockReportModerationService{}, nil)
	r := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports?sort_dir=sideways", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func (m *mockReportRepo) UpdateStatus(_ context.Context, _ int, _ domain.ReportStatus) (*domain.Report, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.report != nil {
		return m.report, nil
	}
	now := time.Now()
	return &domain.Report{ID: 1, Status: domain.ReportStatusReviewed, CreatedAt: now}, nil
}

func (m *mockReportRepo) GetByPOIID(_ context.Context, _ int) ([]domain.Report, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.reports, nil
}

type mockReportModerationService struct {
	result *domain.ModeratedReportResult
	err    error
}

func (m *mockReportModerationService) DisableStory(_ context.Context, _ int) (*domain.ModeratedReportResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &domain.ModeratedReportResult{
		Report: domain.Report{ID: 1, StoryID: 10, Status: domain.ReportStatusResolved, CreatedAt: time.Now()},
		Story:  domain.ModeratedStory{ID: 10, POIID: 5, Language: "en", Status: domain.StoryStatusDisabled},
	}, nil
}

func setupReportRouter(h *ReportHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/reports", h.CreateReport)
	r.GET("/api/v1/admin/reports", h.ListReports)
	r.PUT("/api/v1/admin/reports/:id", h.UpdateReportStatus)
	r.POST("/api/v1/admin/reports/:id/disable-story", h.DisableStory)
	r.GET("/api/v1/admin/pois/:id/reports", h.ListByPOI)
	return r
}

func TestCreateReport_Success(t *testing.T) {
	mock := &mockReportRepo{}
	h := NewReportHandler(mock, &mockReportModerationService{}, nil)
	router := setupReportRouter(h)

	w := executeJSONRequest(t, router, http.MethodPost, "/api/v1/reports", `{"story_id":10,"user_id":"550e8400-e29b-41d4-a716-446655440000","type":"wrong_fact"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListReports_Success(t *testing.T) {
	poiID := 5
	poiName := "Test POI"
	lang := "en"
	storyStatus := "active"
	mock := &mockReportRepo{
		adminReports: []domain.AdminReportListItem{
			{Report: domain.Report{ID: 1, StoryID: 10, UserID: "user-1", Type: domain.ReportTypeWrongFact, Status: domain.ReportStatusNew, CreatedAt: time.Now()}, POIID: &poiID, POIName: &poiName, StoryLanguage: &lang, StoryStatus: &storyStatus},
			{Report: domain.Report{ID: 2, StoryID: 11, UserID: "user-2", Type: domain.ReportTypeWrongLocation, Status: domain.ReportStatusReviewed, CreatedAt: time.Now()}, POIID: &poiID, POIName: &poiName, StoryLanguage: &lang, StoryStatus: &storyStatus},
		},
	}
	h := NewReportHandler(mock, &mockReportModerationService{}, nil)
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports?limit=1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items      []domain.AdminReportListItem `json:"items"`
		NextCursor string                       `json:"next_cursor"`
		HasMore    bool                         `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 report, got %d", len(resp.Items))
	}
	if !resp.HasMore {
		t.Fatal("expected has_more=true")
	}
}

func TestListReports_ValidStatusFilter(t *testing.T) {
	mock := &mockReportRepo{
		adminReports: []domain.AdminReportListItem{},
	}
	h := NewReportHandler(mock, &mockReportModerationService{}, nil)
	router := setupReportRouter(h)

	for _, status := range []string{"new", "reviewed", "resolved", "dismissed"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports?status="+status, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 for status=%s, got %d: %s", status, w.Code, w.Body.String())
		}
	}
}

func TestListReports_InvalidLimit(t *testing.T) {
	h := NewReportHandler(&mockReportRepo{}, &mockReportModerationService{}, nil)
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports?limit=101", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestListReports_InvalidCursor(t *testing.T) {
	h := NewReportHandler(&mockReportRepo{err: fmt.Errorf("malformed encoding: %w", domain.ErrInvalidCursor)}, &mockReportModerationService{}, nil)
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports?cursor=bad", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestReportHandler_InvalidRequests(t *testing.T) {
	h := NewReportHandler(&mockReportRepo{}, &mockReportModerationService{}, nil)
	router := newRouterWithTrace("trace-report-123", func(r *gin.Engine) {
		r.POST("/api/v1/reports", h.CreateReport)
		r.GET("/api/v1/admin/reports", h.ListReports)
		r.PUT("/api/v1/admin/reports/:id", h.UpdateReportStatus)
		r.POST("/api/v1/admin/reports/:id/disable-story", h.DisableStory)
		r.GET("/api/v1/admin/pois/:id/reports", h.ListByPOI)
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
			name:         "create report missing user id",
			method:       http.MethodPost,
			path:         "/api/v1/reports",
			body:         `{"story_id":10,"type":"wrong_fact"}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"userid": "this field is required",
			},
		},
		{
			name:          "create report invalid story id",
			method:        http.MethodPost,
			path:          "/api/v1/reports",
			body:          `{"story_id":-1,"user_id":"550e8400-e29b-41d4-a716-446655440000","type":"wrong_fact"}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "story_id must be a positive integer",
		},
		{
			name:         "create report invalid user_id format",
			method:       http.MethodPost,
			path:         "/api/v1/reports",
			body:         `{"story_id":10,"user_id":"not-a-uuid","type":"wrong_fact"}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"user_id": "must be a valid UUID",
			},
		},
		{
			name:         "create report comment too short",
			method:       http.MethodPost,
			path:         "/api/v1/reports",
			body:         `{"story_id":10,"user_id":"550e8400-e29b-41d4-a716-446655440000","type":"wrong_fact","comment":"short"}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"comment": "must be at least 10 characters",
			},
		},
		{
			name:         "create report comment too long",
			method:       http.MethodPost,
			path:         "/api/v1/reports",
			body:         `{"story_id":10,"user_id":"550e8400-e29b-41d4-a716-446655440000","type":"wrong_fact","comment":"` + strings.Repeat("a", 1001) + `"}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"comment": "must not exceed 1000 characters",
			},
		},
		{
			name:          "create report invalid type",
			method:        http.MethodPost,
			path:          "/api/v1/reports",
			body:          `{"story_id":10,"user_id":"550e8400-e29b-41d4-a716-446655440000","type":"other"}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "type must be one of: wrong_location, wrong_fact, inappropriate_content",
		},
		{
			name:          "create report invalid coordinate pair",
			method:        http.MethodPost,
			path:          "/api/v1/reports",
			body:          `{"story_id":10,"user_id":"550e8400-e29b-41d4-a716-446655440000","type":"wrong_fact","lat":41.7}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "lat and lng must both be provided or both omitted",
		},
		{
			name:         "list reports invalid status filter",
			method:       http.MethodGet,
			path:         "/api/v1/admin/reports?status=archived",
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"status": "must be one of: new, reviewed, resolved, dismissed",
			},
		},
		{
			name:         "list reports invalid status filter garbage",
			method:       http.MethodGet,
			path:         "/api/v1/admin/reports?status=foobar",
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"status": "must be one of: new, reviewed, resolved, dismissed",
			},
		},
		{
			name:          "list reports invalid limit",
			method:        http.MethodGet,
			path:          "/api/v1/admin/reports?limit=0",
			expectedCode:  http.StatusBadRequest,
			expectedError: "limit must be a positive integer",
		},
		{
			name:         "update report missing status",
			method:       http.MethodPut,
			path:         "/api/v1/admin/reports/1",
			body:         `{}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"status": "this field is required",
			},
		},
		{
			name:          "update report invalid path id",
			method:        http.MethodPut,
			path:          "/api/v1/admin/reports/not-int",
			body:          `{"status":"reviewed"}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid id parameter",
		},
		{
			name:          "update report invalid status enum",
			method:        http.MethodPut,
			path:          "/api/v1/admin/reports/1",
			body:          `{"status":"archived"}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "status must be one of: new, reviewed, resolved, dismissed",
		},
		{
			name:          "disable story invalid path id",
			method:        http.MethodPost,
			path:          "/api/v1/admin/reports/not-int/disable-story",
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid id parameter",
		},
		{
			name:          "list by poi invalid path id",
			method:        http.MethodGet,
			path:          "/api/v1/admin/pois/nope/reports",
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid id parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w *httptest.ResponseRecorder
			if tt.body == "" {
				w = executeRequest(router, httptest.NewRequest(tt.method, tt.path, nil))
			} else {
				w = executeJSONRequest(t, router, tt.method, tt.path, tt.body)
			}

			if tt.expectedField != nil {
				assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
					RequestID:         "trace-report-123",
					AllowExtraDetails: true,
					DetailsByField:    tt.expectedField,
				})
				return
			}
			if w.Code != tt.expectedCode {
				t.Fatalf("expected status %d, got %d: %s", tt.expectedCode, w.Code, w.Body.String())
			}
			assertErrorResponse(t, w.Body.Bytes(), tt.expectedError, "trace-report-123")
		})
	}
}

func TestCreateReport_InvalidJSON(t *testing.T) {
	h := NewReportHandler(&mockReportRepo{}, &mockReportModerationService{}, nil)
	router := setupReportRouter(h)

	w := executeJSONRequest(t, router, http.MethodPost, "/api/v1/reports", "{")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	assertErrorResponseContains(t, w.Body.Bytes(), "unexpected EOF")
}

func TestUpdateReportStatus_Success(t *testing.T) {
	mock := &mockReportRepo{}
	h := NewReportHandler(mock, &mockReportModerationService{}, nil)
	router := setupReportRouter(h)

	w := executeJSONRequest(t, router, http.MethodPut, "/api/v1/admin/reports/1", `{"status":"reviewed"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateReport_ForeignKeyViolation(t *testing.T) {
	mock := &mockReportRepo{err: repository.ErrInvalidReference}
	h := NewReportHandler(mock, &mockReportModerationService{}, nil)
	router := setupReportRouter(h)

	w := executeJSONRequest(t, router, http.MethodPost, "/api/v1/reports", `{"story_id":999,"user_id":"550e8400-e29b-41d4-a716-446655440000","type":"wrong_fact"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorResponse(t, w.Body.Bytes(), "referenced record does not exist", "")
}

func TestUpdateReportStatus_Conflict(t *testing.T) {
	mock := &mockReportRepo{err: repository.ErrConflict}
	h := NewReportHandler(mock, &mockReportModerationService{}, nil)
	router := setupReportRouter(h)

	w := executeJSONRequest(t, router, http.MethodPut, "/api/v1/admin/reports/1", `{"status":"reviewed"}`)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorResponse(t, w.Body.Bytes(), "report already exists", "")
}

func TestListReportsByPOI_Success(t *testing.T) {
	mock := &mockReportRepo{
		reports: []domain.Report{{ID: 1, StoryID: 10, UserID: "user-1", Type: domain.ReportTypeWrongFact, Status: domain.ReportStatusNew, CreatedAt: time.Now()}},
	}
	h := NewReportHandler(mock, &mockReportModerationService{}, nil)
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/pois/10/reports", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListReports_EmptyResult(t *testing.T) {
	mock := &mockReportRepo{
		adminReports: nil,
	}
	h := NewReportHandler(mock, &mockReportModerationService{}, nil)
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items   json.RawMessage `json:"items"`
		HasMore bool            `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if string(resp.Items) != "[]" {
		t.Fatalf("expected items to be [], got %s", string(resp.Items))
	}
	if resp.HasMore {
		t.Fatal("expected has_more=false")
	}
}

func TestListReports_StatusFilterPassedToRepo(t *testing.T) {
	mock := &mockReportRepo{
		adminReports: []domain.AdminReportListItem{},
	}
	h := NewReportHandler(mock, &mockReportModerationService{}, nil)
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports?status=reviewed", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if mock.calledStatus != "reviewed" {
		t.Fatalf("expected status filter 'reviewed' passed to repo, got %q", mock.calledStatus)
	}
}

func TestDisableStory_Success(t *testing.T) {
	h := NewReportHandler(&mockReportRepo{}, &mockReportModerationService{}, nil)
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/reports/1/disable-story", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

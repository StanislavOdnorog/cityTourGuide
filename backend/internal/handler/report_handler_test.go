package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

type mockReportRepo struct {
	reports []domain.Report
	report  *domain.Report
	err     error
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

func (m *mockReportRepo) GetAll(_ context.Context, _ string, _, _ int) ([]domain.Report, int, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.reports, len(m.reports), nil
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

func setupReportRouter(h *ReportHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/reports", h.CreateReport)
	r.GET("/api/v1/admin/reports", h.ListReports)
	r.PUT("/api/v1/admin/reports/:id", h.UpdateReportStatus)
	r.GET("/api/v1/admin/pois/:id/reports", h.ListByPOI)
	return r
}

func TestCreateReport_Success(t *testing.T) {
	mock := &mockReportRepo{}
	h := NewReportHandler(mock)
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", strings.NewReader(`{"story_id":10,"user_id":"550e8400-e29b-41d4-a716-446655440000","type":"wrong_fact"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListReports_Success(t *testing.T) {
	mock := &mockReportRepo{
		reports: []domain.Report{
			{ID: 1, StoryID: 10, UserID: "user-1", Type: domain.ReportTypeWrongFact, Status: domain.ReportStatusNew, CreatedAt: time.Now()},
			{ID: 2, StoryID: 11, UserID: "user-2", Type: domain.ReportTypeWrongLocation, Status: domain.ReportStatusReviewed, CreatedAt: time.Now()},
		},
	}
	h := NewReportHandler(mock)
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports?limit=1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items      []domain.Report `json:"items"`
		NextCursor string          `json:"next_cursor"`
		HasMore    bool            `json:"has_more"`
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

func TestListReports_InvalidLimit(t *testing.T) {
	h := NewReportHandler(&mockReportRepo{})
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports?limit=101", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestListReports_InvalidCursor(t *testing.T) {
	h := NewReportHandler(&mockReportRepo{err: errors.New("invalid cursor: malformed encoding")})
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports?cursor=bad", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestReportHandler_InvalidRequests(t *testing.T) {
	h := NewReportHandler(&mockReportRepo{})
	router := newRouterWithTrace("trace-report-123", func(r *gin.Engine) {
		r.POST("/api/v1/reports", h.CreateReport)
		r.GET("/api/v1/admin/reports", h.ListReports)
		r.PUT("/api/v1/admin/reports/:id", h.UpdateReportStatus)
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
			name:          "list by poi invalid path id",
			method:        http.MethodGet,
			path:          "/api/v1/admin/pois/nope/reports",
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid id parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body == "" {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			} else {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Fatalf("expected status %d, got %d: %s", tt.expectedCode, w.Code, w.Body.String())
			}

			if tt.expectedField != nil {
				assertValidationErrorResponse(t, w.Body.Bytes(), tt.expectedField, "trace-report-123")
				return
			}
			assertErrorResponse(t, w.Body.Bytes(), tt.expectedError, "trace-report-123")
		})
	}
}

func TestCreateReport_InvalidJSON(t *testing.T) {
	h := NewReportHandler(&mockReportRepo{})
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	assertErrorResponseContains(t, w.Body.Bytes(), "unexpected EOF")
}

func TestUpdateReportStatus_Success(t *testing.T) {
	mock := &mockReportRepo{}
	h := NewReportHandler(mock)
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/reports/1", strings.NewReader(`{"status":"reviewed"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListReportsByPOI_Success(t *testing.T) {
	mock := &mockReportRepo{
		reports: []domain.Report{{ID: 1, StoryID: 10, UserID: "user-1", Type: domain.ReportTypeWrongFact, Status: domain.ReportStatusNew, CreatedAt: time.Now()}},
	}
	h := NewReportHandler(mock)
	router := setupReportRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/pois/10/reports", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

package handler

import (
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
	r.GET("/api/v1/admin/reports", h.ListReports)
	return r
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

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/repository"
)

type mockAdminStatsRepo struct {
	stats *repository.AdminStats
	err   error
}

func (m *mockAdminStatsRepo) Get(_ context.Context) (*repository.AdminStats, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.stats != nil {
		return m.stats, nil
	}
	return &repository.AdminStats{}, nil
}

func setupAdminStatsRouter(h *AdminStatsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/admin/stats", h.Get)
	return r
}

func TestAdminStatsHandler_Get_Success(t *testing.T) {
	h := NewAdminStatsHandler(&mockAdminStatsRepo{
		stats: &repository.AdminStats{
			CitiesCount:     4,
			POIsCount:       12,
			StoriesCount:    28,
			ReportsCount:    3,
			NewReportsCount: 2,
		},
	})
	router := setupAdminStatsRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data repository.AdminStats `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Data.CitiesCount != 4 || resp.Data.POIsCount != 12 || resp.Data.StoriesCount != 28 || resp.Data.ReportsCount != 3 || resp.Data.NewReportsCount != 2 {
		t.Fatalf("unexpected stats response: %+v", resp.Data)
	}
}

func TestAdminStatsHandler_Get_DataEnvelope(t *testing.T) {
	h := NewAdminStatsHandler(&mockAdminStatsRepo{
		stats: &repository.AdminStats{
			CitiesCount:     1,
			POIsCount:       2,
			StoriesCount:    3,
			ReportsCount:    4,
			NewReportsCount: 5,
		},
	})
	router := setupAdminStatsRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the top-level key is "data"
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, ok := raw["data"]; !ok {
		t.Fatal("response missing top-level \"data\" key")
	}
	if len(raw) != 1 {
		t.Fatalf("expected exactly 1 top-level key, got %d", len(raw))
	}

	// Verify all expected fields are present and correct
	var data struct {
		CitiesCount     int `json:"cities_count"`
		POIsCount       int `json:"pois_count"`
		StoriesCount    int `json:"stories_count"`
		ReportsCount    int `json:"reports_count"`
		NewReportsCount int `json:"new_reports_count"`
	}
	if err := json.Unmarshal(raw["data"], &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if data.CitiesCount != 1 || data.POIsCount != 2 || data.StoriesCount != 3 || data.ReportsCount != 4 || data.NewReportsCount != 5 {
		t.Fatalf("unexpected data: %+v", data)
	}
}

func TestAdminStatsHandler_Get_ZeroCounts(t *testing.T) {
	h := NewAdminStatsHandler(&mockAdminStatsRepo{})
	router := setupAdminStatsRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data repository.AdminStats `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.CitiesCount != 0 || resp.Data.POIsCount != 0 || resp.Data.StoriesCount != 0 || resp.Data.ReportsCount != 0 || resp.Data.NewReportsCount != 0 {
		t.Fatalf("expected all zeros, got %+v", resp.Data)
	}
}

func TestAdminStatsHandler_Get_Error(t *testing.T) {
	h := NewAdminStatsHandler(&mockAdminStatsRepo{err: errors.New("boom")})
	router := newRouterWithTrace("trace-admin-stats-123", func(r *gin.Engine) {
		r.GET("/api/v1/admin/stats", h.Get)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	assertErrorResponse(t, w.Body.Bytes(), "failed to fetch admin stats", "trace-admin-stats-123")
}

func TestAdminStatsHandler_Get_ErrorWithoutTrace(t *testing.T) {
	h := NewAdminStatsHandler(&mockAdminStatsRepo{err: errors.New("boom")})
	router := setupAdminStatsRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["error"] != "failed to fetch admin stats" {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
	if _, ok := resp["trace_id"]; ok {
		t.Fatal("expected no trace_id when context has no trace")
	}
}

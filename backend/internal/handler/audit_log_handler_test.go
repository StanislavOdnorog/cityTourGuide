package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/middleware"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

type mockAuditLogRepo struct {
	result *domain.PageResponse[domain.AuditLog]
	filter repository.AuditLogFilter
	page   domain.PageRequest
	err    error
	sort   repository.ListSort
}

func (m *mockAuditLogRepo) List(_ context.Context, filter repository.AuditLogFilter, page domain.PageRequest, sort repository.ListSort) (*domain.PageResponse[domain.AuditLog], error) {
	m.filter = filter
	m.page = page
	m.sort = sort
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &domain.PageResponse[domain.AuditLog]{
		Items:   []domain.AuditLog{},
		HasMore: false,
	}, nil
}

func TestAuditLogHandler_List_SortParams(t *testing.T) {
	mock := &mockAuditLogRepo{}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?sort_by=action&sort_dir=asc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if mock.sort.By != "action" || mock.sort.Dir != repository.SortDirAsc {
		t.Fatalf("expected sort {action asc}, got %+v", mock.sort)
	}
}

func TestAuditLogHandler_List_InvalidSortBy(t *testing.T) {
	mock := &mockAuditLogRepo{}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?sort_by=actor_id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuditLogHandler_List_InvalidSortDir(t *testing.T) {
	mock := &mockAuditLogRepo{}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?sort_dir=sideways", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func setupAuditLogRouter(h *AuditLogHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/admin/audit-logs", h.List)
	return r
}

type stubAdminValidator struct {
	userID string
	err    error
}

func (s stubAdminValidator) ValidateAdminToken(string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.userID, nil
}

func setupProtectedAuditLogRouter(h *AuditLogHandler, validator middleware.AdminTokenValidator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AdminAuth(validator))
	admin.GET("/audit-logs", h.List)
	return r
}

func TestAuditLogHandler_List_Default(t *testing.T) {
	now := time.Now()
	mock := &mockAuditLogRepo{
		result: &domain.PageResponse[domain.AuditLog]{
			Items: []domain.AuditLog{
				{
					ID: 10, ActorID: "admin-1", Action: "create", ResourceType: "city",
					ResourceID: "5", HTTPMethod: "POST", RequestPath: "/api/v1/admin/cities",
					TraceID: "trace-1", Payload: []byte(`{"name":"Tbilisi"}`), Status: "success",
					CreatedAt: now,
				},
				{
					ID: 9, ActorID: "admin-2", Action: "update", ResourceType: "poi",
					ResourceID: "3", HTTPMethod: "PUT", RequestPath: "/api/v1/admin/pois/3",
					TraceID: "trace-2", Status: "success", CreatedAt: now,
				},
			},
			NextCursor: domain.EncodeCursor64(9),
			HasMore:    true,
		},
	}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items      []auditLogItem `json:"items"`
		NextCursor string         `json:"next_cursor"`
		HasMore    bool           `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if !resp.HasMore {
		t.Error("expected has_more=true")
	}
	if resp.NextCursor == "" {
		t.Error("expected non-empty next_cursor")
	}
	if resp.Items[0].Action != "create" {
		t.Errorf("expected first item action=create, got %q", resp.Items[0].Action)
	}
	if resp.Items[1].Payload != nil {
		t.Error("expected nil payload for second item")
	}
	if resp.Items[0].Payload == nil {
		t.Fatal("expected non-nil payload for first item")
	}
}

func TestAuditLogHandler_List_WithFilters(t *testing.T) {
	mock := &mockAuditLogRepo{}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	const createdFrom = "2026-01-01T00:00:00Z"
	const createdTo = "2026-01-31T23:59:59Z"

	const actorUUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/admin/audit-logs?actor_id="+actorUUID+"&resource_type=city&action=create&created_from="+createdFrom+"&created_to="+createdTo,
		nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if mock.filter.ActorID != actorUUID {
		t.Errorf("expected actor_id=%s, got %q", actorUUID, mock.filter.ActorID)
	}
	if mock.filter.ResourceType != "city" {
		t.Errorf("expected resource_type=city, got %q", mock.filter.ResourceType)
	}
	if mock.filter.Action != "create" {
		t.Errorf("expected action=create, got %q", mock.filter.Action)
	}
	if mock.filter.CreatedFrom == nil || mock.filter.CreatedFrom.Format(time.RFC3339) != createdFrom {
		t.Fatalf("expected created_from=%s, got %+v", createdFrom, mock.filter.CreatedFrom)
	}
	if mock.filter.CreatedTo == nil || mock.filter.CreatedTo.Format(time.RFC3339) != createdTo {
		t.Fatalf("expected created_to=%s, got %+v", createdTo, mock.filter.CreatedTo)
	}
}

func TestAuditLogHandler_List_InvalidCursor(t *testing.T) {
	mock := &mockAuditLogRepo{
		err: fmt.Errorf("audit_log_repo: list: %w", domain.ErrInvalidCursor),
	}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?cursor=bad-cursor", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuditLogHandler_List_EmptyResult(t *testing.T) {
	mock := &mockAuditLogRepo{
		result: &domain.PageResponse[domain.AuditLog]{
			Items:   []domain.AuditLog{},
			HasMore: false,
		},
	}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items      []auditLogItem `json:"items"`
		NextCursor string         `json:"next_cursor"`
		HasMore    bool           `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(resp.Items))
	}
	if resp.HasMore {
		t.Error("expected has_more=false")
	}
	if resp.NextCursor != "" {
		t.Errorf("expected empty next_cursor, got %q", resp.NextCursor)
	}
}

func TestAuditLogHandler_List_InvalidLimit(t *testing.T) {
	mock := &mockAuditLogRepo{}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?limit=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuditLogHandler_List_InvalidCreatedFrom(t *testing.T) {
	mock := &mockAuditLogRepo{}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?created_from=not-a-date", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuditLogHandler_List_InvalidCreatedWindow(t *testing.T) {
	mock := &mockAuditLogRepo{}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/audit-logs?created_from=2026-02-01T00:00:00Z&created_to=2026-01-01T00:00:00Z",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuditLogHandler_List_NullPayload(t *testing.T) {
	mock := &mockAuditLogRepo{
		result: &domain.PageResponse[domain.AuditLog]{
			Items: []domain.AuditLog{
				{
					ID: 1, ActorID: "", Action: "create", ResourceType: "city",
					ResourceID: "1", HTTPMethod: "POST", RequestPath: "/api/v1/admin/cities",
					TraceID: "", Payload: nil, Status: "success",
					CreatedAt: time.Now(),
				},
			},
			HasMore: false,
		},
	}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items []auditLogItem `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].Payload != nil {
		t.Error("expected null payload in response")
	}
	if resp.Items[0].ActorID != "" {
		t.Errorf("expected empty actor_id, got %q", resp.Items[0].ActorID)
	}
}

func TestAuditLogHandler_List_ServerError(t *testing.T) {
	mock := &mockAuditLogRepo{
		err: fmt.Errorf("audit_log_repo: list: connection refused"),
	}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuditLogHandler_List_CustomLimit(t *testing.T) {
	mock := &mockAuditLogRepo{}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?limit=5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if mock.page.Limit != 5 {
		t.Errorf("expected limit=5, got %d", mock.page.Limit)
	}
}

func TestAuditLogHandler_List_RequiresAdminAuth(t *testing.T) {
	h := NewAuditLogHandler(&mockAuditLogRepo{})
	r := setupProtectedAuditLogRouter(h, stubAdminValidator{userID: "admin-1"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuditLogHandler_List_RejectsNonAdminToken(t *testing.T) {
	h := NewAuditLogHandler(&mockAuditLogRepo{})
	r := setupProtectedAuditLogRouter(h, stubAdminValidator{err: fmt.Errorf("admin access required")})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs", nil)
	req.Header.Set("Authorization", "Bearer user-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuditLogHandler_List_InvalidActorID(t *testing.T) {
	h := NewAuditLogHandler(&mockAuditLogRepo{})
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?actor_id=not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuditLogHandler_List_InvalidAction(t *testing.T) {
	h := NewAuditLogHandler(&mockAuditLogRepo{})
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?action=bogus", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuditLogHandler_List_InvalidResourceType(t *testing.T) {
	h := NewAuditLogHandler(&mockAuditLogRepo{})
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?resource_type=widget", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuditLogHandler_List_InvalidStatus(t *testing.T) {
	h := NewAuditLogHandler(&mockAuditLogRepo{})
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?status=maybe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuditLogHandler_List_ValidEnumValues(t *testing.T) {
	mock := &mockAuditLogRepo{}
	h := NewAuditLogHandler(mock)
	r := setupAuditLogRouter(h)

	// All valid enum values should pass through
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?action=disable_story&resource_type=inflation_job&status=error", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if mock.filter.Action != "disable_story" {
		t.Errorf("expected action=disable_story, got %q", mock.filter.Action)
	}
	if mock.filter.ResourceType != "inflation_job" {
		t.Errorf("expected resource_type=inflation_job, got %q", mock.filter.ResourceType)
	}
	if mock.filter.Status != "error" {
		t.Errorf("expected status=error, got %q", mock.filter.Status)
	}
}

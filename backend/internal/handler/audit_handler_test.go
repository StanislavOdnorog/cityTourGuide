package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCreateCity_AuditLogWritten(t *testing.T) {
	audit := &mockAuditLogger{}
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, audit)
	r := setupCityRouter(h)

	body := `{"name":"Tbilisi","country":"GE","center_lat":41.7,"center_lng":44.8,"radius_km":15}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	if audit.count() != 1 {
		t.Fatalf("expected 1 audit entry, got %d", audit.count())
	}

	entry := audit.lastEntry()
	if entry.Action != "create" {
		t.Errorf("expected action=create, got %q", entry.Action)
	}
	if entry.ResourceType != "city" {
		t.Errorf("expected resource_type=city, got %q", entry.ResourceType)
	}
	if entry.ResourceID != "1" {
		t.Errorf("expected resource_id=1, got %q", entry.ResourceID)
	}
	if entry.HTTPMethod != "POST" {
		t.Errorf("expected method=POST, got %q", entry.HTTPMethod)
	}
	if entry.Status != "success" {
		t.Errorf("expected status=success, got %q", entry.Status)
	}
}

func TestUpdateCity_AuditLogWritten(t *testing.T) {
	audit := &mockAuditLogger{}
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, audit)
	r := setupCityRouter(h)

	body := `{"name":"Tbilisi Updated","country":"GE","center_lat":41.7,"center_lng":44.8,"radius_km":20}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/cities/5", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if audit.count() != 1 {
		t.Fatalf("expected 1 audit entry, got %d", audit.count())
	}

	entry := audit.lastEntry()
	if entry.Action != "update" {
		t.Errorf("expected action=update, got %q", entry.Action)
	}
	if entry.ResourceID != "5" {
		t.Errorf("expected resource_id=5, got %q", entry.ResourceID)
	}
}

func TestDeleteCity_AuditLogWritten(t *testing.T) {
	audit := &mockAuditLogger{}
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, audit)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/cities/3", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if audit.count() != 1 {
		t.Fatalf("expected 1 audit entry, got %d", audit.count())
	}

	entry := audit.lastEntry()
	if entry.Action != "delete" {
		t.Errorf("expected action=delete, got %q", entry.Action)
	}
	if entry.ResourceType != "city" {
		t.Errorf("expected resource_type=city, got %q", entry.ResourceType)
	}
	if entry.ResourceID != "3" {
		t.Errorf("expected resource_id=3, got %q", entry.ResourceID)
	}
	if entry.Payload != nil {
		t.Error("expected nil payload for delete")
	}
}

func TestRestoreCity_AuditLogWritten(t *testing.T) {
	audit := &mockAuditLogger{}
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, audit)
	r := setupCityRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities/3/restore", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if audit.count() != 1 {
		t.Fatalf("expected 1 audit entry, got %d", audit.count())
	}

	entry := audit.lastEntry()
	if entry.Action != "restore" {
		t.Errorf("expected action=restore, got %q", entry.Action)
	}
	if entry.ResourceType != "city" {
		t.Errorf("expected resource_type=city, got %q", entry.ResourceType)
	}
	if entry.ResourceID != "3" {
		t.Errorf("expected resource_id=3, got %q", entry.ResourceID)
	}
	if entry.Payload != nil {
		t.Error("expected nil payload for restore")
	}
}

func TestCreateCity_ValidationFailure_NoAuditLog(t *testing.T) {
	audit := &mockAuditLogger{}
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, audit)
	r := setupCityRouter(h)

	// Missing required fields
	body := `{"name":"Tbilisi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	if audit.count() != 0 {
		t.Fatalf("expected 0 audit entries for failed validation, got %d", audit.count())
	}
}

func TestUpdateReportStatus_AuditLogWritten(t *testing.T) {
	audit := &mockAuditLogger{}
	mock := &mockReportRepo{}
	h := NewReportHandler(mock, &mockReportModerationService{}, audit)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/api/v1/admin/reports/:id", h.UpdateReportStatus)

	body := `{"status":"resolved"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/reports/7", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if audit.count() != 1 {
		t.Fatalf("expected 1 audit entry, got %d", audit.count())
	}

	entry := audit.lastEntry()
	if entry.Action != "update_status" {
		t.Errorf("expected action=update_status, got %q", entry.Action)
	}
	if entry.ResourceType != "report" {
		t.Errorf("expected resource_type=report, got %q", entry.ResourceType)
	}
	if entry.ResourceID != "7" {
		t.Errorf("expected resource_id=7, got %q", entry.ResourceID)
	}
}

func TestUpdateReportStatus_InvalidStatus_NoAuditLog(t *testing.T) {
	audit := &mockAuditLogger{}
	mock := &mockReportRepo{}
	h := NewReportHandler(mock, &mockReportModerationService{}, audit)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/api/v1/admin/reports/:id", h.UpdateReportStatus)

	body := `{"status":"invalid_status"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/reports/7", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	if audit.count() != 0 {
		t.Fatalf("expected 0 audit entries for failed validation, got %d", audit.count())
	}
}

func TestCreateCity_AuditLogWithActorAndTrace(t *testing.T) {
	audit := &mockAuditLogger{}
	mock := &mockCityRepo{}
	h := NewCityHandler(mock, &mockManifestRepo{}, audit)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Add middleware to set user_id and trace_id
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "admin-uuid-789")
		c.Set("trace_id", "trace-xyz-000")
		c.Next()
	})
	r.POST("/api/v1/admin/cities", h.CreateCity)

	body := `{"name":"Batumi","country":"GE","center_lat":41.6,"center_lng":41.6,"radius_km":10}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	entry := audit.lastEntry()
	if entry.ActorID != "admin-uuid-789" {
		t.Errorf("expected actor_id=admin-uuid-789, got %q", entry.ActorID)
	}
	if entry.TraceID != "trace-xyz-000" {
		t.Errorf("expected trace_id=trace-xyz-000, got %q", entry.TraceID)
	}
}

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// mockAuditLogger captures audit log entries for testing.
type mockAuditLogger struct {
	mu      sync.Mutex
	entries []*domain.AuditLog
	err     error
}

func (m *mockAuditLogger) Insert(_ context.Context, log *domain.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.entries = append(m.entries, log)
	return nil
}

func (m *mockAuditLogger) lastEntry() *domain.AuditLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.entries) == 0 {
		return nil
	}
	return m.entries[len(m.entries)-1]
}

func (m *mockAuditLogger) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.entries)
}

func TestAuditEntry_RecordsSuccessfulAction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAuditLogger{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities", nil)
	c.Set("user_id", "admin-user-123")
	c.Set("trace_id", "trace-abc-456")

	payload := map[string]string{"name": "Tbilisi"}
	auditEntry(c, mock, "create", "city", "1", payload)

	if mock.count() != 1 {
		t.Fatalf("expected 1 audit entry, got %d", mock.count())
	}

	entry := mock.lastEntry()
	if entry.ActorID != "admin-user-123" {
		t.Errorf("expected actor_id=admin-user-123, got %q", entry.ActorID)
	}
	if entry.Action != "create" {
		t.Errorf("expected action=create, got %q", entry.Action)
	}
	if entry.ResourceType != "city" {
		t.Errorf("expected resource_type=city, got %q", entry.ResourceType)
	}
	if entry.ResourceID != "1" {
		t.Errorf("expected resource_id=1, got %q", entry.ResourceID)
	}
	if entry.HTTPMethod != http.MethodPost {
		t.Errorf("expected method=POST, got %q", entry.HTTPMethod)
	}
	if entry.RequestPath != "/api/v1/admin/cities" {
		t.Errorf("expected path=/api/v1/admin/cities, got %q", entry.RequestPath)
	}
	if entry.TraceID != "trace-abc-456" {
		t.Errorf("expected trace_id=trace-abc-456, got %q", entry.TraceID)
	}
	if entry.Status != "success" {
		t.Errorf("expected status=success, got %q", entry.Status)
	}

	var p map[string]string
	if err := json.Unmarshal(entry.Payload, &p); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if p["name"] != "Tbilisi" {
		t.Errorf("expected payload name=Tbilisi, got %q", p["name"])
	}
}

func TestAuditEntry_NilLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities", nil)

	// Should not panic with nil logger
	auditEntry(c, nil, "create", "city", "1", nil)
}

func TestAuditEntry_NoUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAuditLogger{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/cities/5", nil)

	auditEntry(c, mock, "delete", "city", "5", nil)

	if mock.count() != 1 {
		t.Fatalf("expected 1 audit entry, got %d", mock.count())
	}

	entry := mock.lastEntry()
	if entry.ActorID != "" {
		t.Errorf("expected empty actor_id, got %q", entry.ActorID)
	}
	if entry.Payload != nil {
		t.Errorf("expected nil payload, got %v", entry.Payload)
	}
}

func TestAuditEntry_OversizedPayloadDropped(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAuditLogger{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/stories/1", nil)

	// Create a payload that exceeds 4096 bytes when serialized
	bigPayload := map[string]string{"text": string(make([]byte, 5000))}
	auditEntry(c, mock, "update", "story", "1", bigPayload)

	entry := mock.lastEntry()
	if entry.Payload != nil {
		t.Error("expected nil payload for oversized data")
	}
}

func TestAuditEntry_RedactsSensitiveFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAuditLogger{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", nil)
	c.Set("user_id", "admin-1")

	payload := map[string]any{
		"name":     "Alice",
		"password": "hunter2",
		"token":    "tok-secret",
	}
	auditEntry(c, mock, "create", "user", "1", payload)

	entry := mock.lastEntry()
	if entry.Payload == nil {
		t.Fatal("expected non-nil payload")
	}

	var p map[string]any
	if err := json.Unmarshal(entry.Payload, &p); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if p["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", p["name"])
	}
	if p["password"] != "[REDACTED]" {
		t.Errorf("expected password redacted, got %v", p["password"])
	}
	if p["token"] != "[REDACTED]" {
		t.Errorf("expected token redacted, got %v", p["token"])
	}
}

func TestAuditEntry_RedactsNestedSensitiveFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAuditLogger{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/config", nil)

	payload := map[string]any{
		"settings": map[string]any{
			"api_key": "sk-123",
			"region":  "eu-west",
		},
		"name": "config-1",
	}
	auditEntry(c, mock, "update", "config", "1", payload)

	var p map[string]any
	json.Unmarshal(mock.lastEntry().Payload, &p)

	settings := p["settings"].(map[string]any)
	if settings["api_key"] != "[REDACTED]" {
		t.Errorf("expected nested api_key redacted, got %v", settings["api_key"])
	}
	if settings["region"] != "eu-west" {
		t.Errorf("expected region preserved, got %v", settings["region"])
	}
}

func TestAuditEntry_RedactsArrayPayloads(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAuditLogger{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/batch", nil)

	payload := []map[string]any{
		{"id": "1", "secret": "s1"},
		{"id": "2", "secret": "s2"},
	}
	auditEntry(c, mock, "batch", "tokens", "batch", payload)

	var p []map[string]any
	json.Unmarshal(mock.lastEntry().Payload, &p)
	for i, item := range p {
		if item["secret"] != "[REDACTED]" {
			t.Errorf("item %d: expected secret redacted, got %v", i, item["secret"])
		}
		if item["id"] != fmt.Sprintf("%d", i+1) {
			t.Errorf("item %d: expected id preserved", i)
		}
	}
}

func TestAuditEntry_NilPayloadStillWorks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAuditLogger{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/cities/1", nil)

	auditEntry(c, mock, "delete", "city", "1", nil)

	if mock.count() != 1 {
		t.Fatalf("expected 1 entry, got %d", mock.count())
	}
	if mock.lastEntry().Payload != nil {
		t.Error("expected nil payload for nil input")
	}
}

func TestAuditEntry_StoryPayloadPreserved(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAuditLogger{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/stories", nil)

	payload := storyAuditPayload(42, "en", "atmosphere")
	auditEntry(c, mock, "create", "story", "42", payload)

	var p map[string]any
	json.Unmarshal(mock.lastEntry().Payload, &p)
	if p["poi_id"] != float64(42) {
		t.Errorf("expected poi_id=42, got %v", p["poi_id"])
	}
	if p["language"] != "en" {
		t.Errorf("expected language=en, got %v", p["language"])
	}
}

func TestAuditEntry_OversizedAfterRedactionStillDropped(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAuditLogger{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/bulk", nil)

	// Create payload that's still over 4KB even after redaction
	bigPayload := map[string]string{"description": string(make([]byte, 5000))}
	auditEntry(c, mock, "update", "bulk", "1", bigPayload)

	if mock.lastEntry().Payload != nil {
		t.Error("expected nil payload for oversized data even after redaction")
	}
}

func TestAuditEntry_PayloadFitsAfterRedaction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockAuditLogger{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", nil)

	// Large secret value gets replaced by short [REDACTED], bringing size under 4KB
	payload := map[string]any{
		"name":     "test",
		"password": string(make([]byte, 4000)),
	}
	auditEntry(c, mock, "create", "user", "1", payload)

	entry := mock.lastEntry()
	if entry.Payload == nil {
		t.Fatal("expected payload to fit after redaction shrunk the secret")
	}

	var p map[string]any
	json.Unmarshal(entry.Payload, &p)
	if p["password"] != "[REDACTED]" {
		t.Errorf("expected password redacted, got %v", p["password"])
	}
}

func TestStoryAuditPayload(t *testing.T) {
	p := storyAuditPayload(42, "en", "atmosphere")
	if p["poi_id"] != 42 {
		t.Errorf("expected poi_id=42, got %v", p["poi_id"])
	}
	if p["language"] != "en" {
		t.Errorf("expected language=en, got %v", p["language"])
	}
	if p["layer_type"] != "atmosphere" {
		t.Errorf("expected layer_type=atmosphere, got %v", p["layer_type"])
	}
}

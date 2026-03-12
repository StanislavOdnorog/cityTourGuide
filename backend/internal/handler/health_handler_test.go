package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(_ context.Context) error {
	return m.err
}

type readinessTestResponse struct {
	Status string                 `json:"status"`
	Error  string                 `json:"error"`
	Checks []readinessCheckResult `json:"checks"`
}

func TestHealthz_ReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(&mockPinger{})

	r := gin.New()
	r.GET("/healthz", h.Healthz)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %s", body["status"])
	}
}

func TestReadyz_DBHealthy_ReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(&mockPinger{})

	r := gin.New()
	r.GET("/readyz", h.Readyz)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/readyz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body readinessTestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Status != readinessStatusOK {
		t.Errorf("expected status ok, got %s", body.Status)
	}
	if len(body.Checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(body.Checks))
	}
	if body.Checks[0] != (readinessCheckResult{Name: "server", Required: true, Status: readinessStatusOK}) {
		t.Errorf("unexpected server check: %+v", body.Checks[0])
	}
	if body.Checks[1] != (readinessCheckResult{Name: "database", Required: true, Status: readinessStatusOK}) {
		t.Errorf("unexpected database check: %+v", body.Checks[1])
	}
}

func TestReadyz_RequiredFailure_Returns503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(&mockPinger{err: errors.New("connection refused")})

	r := gin.New()
	r.GET("/readyz", h.Readyz)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/readyz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}

	var body readinessTestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Status != readinessStatusUnavailable {
		t.Errorf("expected status unavailable, got %s", body.Status)
	}
	if body.Error != "database unreachable" {
		t.Errorf("expected error 'database unreachable', got %s", body.Error)
	}
	if len(body.Checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(body.Checks))
	}
	if body.Checks[1] != (readinessCheckResult{Name: "database", Required: true, Status: readinessStatusUnavailable}) {
		t.Errorf("unexpected database check: %+v", body.Checks[1])
	}
}

func TestReadyz_OptionalFailure_ReturnsDegradedWith200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(&mockPinger{}, ReadinessCheck{
		Name:     "fcm",
		Required: false,
		Check: func(context.Context) error {
			return errors.New("misconfigured")
		},
	})

	r := gin.New()
	r.GET("/readyz", h.Readyz)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/readyz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body readinessTestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Status != readinessStatusDegraded {
		t.Errorf("expected status degraded, got %s", body.Status)
	}
	if body.Error != "" {
		t.Errorf("expected empty error, got %s", body.Error)
	}
	if len(body.Checks) != 3 {
		t.Fatalf("expected 3 checks, got %d", len(body.Checks))
	}
	if body.Checks[2] != (readinessCheckResult{Name: "fcm", Required: false, Status: readinessStatusDegraded}) {
		t.Errorf("unexpected optional check: %+v", body.Checks[2])
	}
}

func TestReadyz_ShutdownMode_Returns503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(&mockPinger{})
	h.SetShuttingDown(true)

	r := gin.New()
	r.GET("/readyz", h.Readyz)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/readyz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}

	var body readinessTestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Status != readinessStatusUnavailable {
		t.Errorf("expected status unavailable, got %s", body.Status)
	}
	if body.Error != "server shutting down" {
		t.Errorf("expected error 'server shutting down', got %s", body.Error)
	}
	if len(body.Checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(body.Checks))
	}
	if body.Checks[0] != (readinessCheckResult{Name: "server", Required: true, Status: readinessStatusUnavailable}) {
		t.Errorf("unexpected server check: %+v", body.Checks[0])
	}
	if body.Checks[1] != (readinessCheckResult{Name: "database", Required: true, Status: readinessStatusOK}) {
		t.Errorf("unexpected database check: %+v", body.Checks[1])
	}
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/api/v1/pois/123", "/api/v1/pois/:id"},
		{"/api/v1/pois/123/reports", "/api/v1/pois/:id/reports"},
		{"/api/v1/cities", "/api/v1/cities"},
		{"/api/v1/stories/456", "/api/v1/stories/:id"},
		{"/api/v1/users/550e8400-e29b-41d4-a716-446655440000", "/api/v1/users/:id"},
		{"/healthz", "/healthz"},
	}

	for _, tt := range tests {
		got := normalizePath(tt.input)
		if got != tt.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMetricsMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Metrics())
	r.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/api/v1/test/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
	})

	// Make a request to a simple endpoint.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Make a request with a numeric ID.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/test/42", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Verify counter incremented for the base path.
	counter, err := httpRequestsTotal.GetMetricWithLabelValues("GET", "/api/v1/test", "200")
	if err != nil {
		t.Fatalf("failed to get counter: %v", err)
	}
	var m dto.Metric
	if err := counter.Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected counter=1 for /api/v1/test, got %v", m.GetCounter().GetValue())
	}

	// Verify counter incremented for the :id path (normalized).
	counter2, err := httpRequestsTotal.GetMetricWithLabelValues("GET", "/api/v1/test/:id", "200")
	if err != nil {
		t.Fatalf("failed to get counter: %v", err)
	}
	var m2 dto.Metric
	if err := counter2.Write(&m2); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if m2.GetCounter().GetValue() != 1 {
		t.Errorf("expected counter=1 for /api/v1/test/:id, got %v", m2.GetCounter().GetValue())
	}

	// Verify histogram observed for the base path.
	observer, err := httpRequestDuration.GetMetricWithLabelValues("GET", "/api/v1/test", "200")
	if err != nil {
		t.Fatalf("failed to get histogram: %v", err)
	}
	var hm dto.Metric
	if err := observer.(prometheus.Metric).Write(&hm); err != nil {
		t.Fatalf("failed to write histogram metric: %v", err)
	}
	if hm.GetHistogram().GetSampleCount() != 1 {
		t.Errorf("expected histogram sample_count=1, got %d", hm.GetHistogram().GetSampleCount())
	}
}

func TestMetricsInFlightGauge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Verify in-flight gauge returns to 0 after request completes.
	r := gin.New()
	r.Use(Metrics())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	var gm dto.Metric
	if err := httpRequestsInFlight.Write(&gm); err != nil {
		t.Fatalf("failed to write gauge metric: %v", err)
	}
	if gm.GetGauge().GetValue() != 0 {
		t.Errorf("expected in-flight gauge=0 after request, got %v", gm.GetGauge().GetValue())
	}
}

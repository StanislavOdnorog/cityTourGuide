package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// captureSlog redirects slog output to a buffer for the duration of fn.
func captureSlog(fn func()) string {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	old := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(old)
	fn()
	return buf.String()
}

func TestRequestLogger_ContainsFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	output := captureSlog(func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("User-Agent", "TestAgent/1.0")
		r.ServeHTTP(w, req)
	})

	var entry map[string]any
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\nOutput: %s", err, output)
	}

	if entry["method"] != "GET" {
		t.Errorf("expected method GET, got %v", entry["method"])
	}
	if entry["path"] != "/test" {
		t.Errorf("expected path /test, got %v", entry["path"])
	}
	if sc, ok := entry["status_code"].(float64); !ok || int(sc) != 200 {
		t.Errorf("expected status_code 200, got %v", entry["status_code"])
	}
	if entry["client_ip"] == nil || entry["client_ip"] == "" {
		t.Error("expected client_ip to be set")
	}
	if entry["user_agent"] != "TestAgent/1.0" {
		t.Errorf("expected user_agent TestAgent/1.0, got %v", entry["user_agent"])
	}
	if entry["time"] == nil || entry["time"] == "" {
		t.Error("expected time to be set")
	}
}

func TestRequestLogger_InfoLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	output := captureSlog(func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/ok", nil)
		r.ServeHTTP(w, req)
	})

	var entry map[string]any
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("invalid JSON: %s", output)
	}
	if entry["level"] != "INFO" {
		t.Errorf("expected level INFO for 200, got %v", entry["level"])
	}
}

func TestRequestLogger_WarnLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/bad", func(c *gin.Context) {
		c.Status(http.StatusBadRequest)
	})

	output := captureSlog(func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/bad", nil)
		r.ServeHTTP(w, req)
	})

	var entry map[string]any
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("invalid JSON: %s", output)
	}
	if entry["level"] != "WARN" {
		t.Errorf("expected level WARN for 400, got %v", entry["level"])
	}
}

func TestRequestLogger_ErrorLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/err", func(c *gin.Context) {
		c.Status(http.StatusInternalServerError)
	})

	output := captureSlog(func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/err", nil)
		r.ServeHTTP(w, req)
	})

	var entry map[string]any
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("invalid JSON: %s", output)
	}
	if entry["level"] != "ERROR" {
		t.Errorf("expected level ERROR for 500, got %v", entry["level"])
	}
}

func TestRequestLogger_IncludesUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-abc")
		c.Next()
	})
	r.Use(RequestLogger())
	r.GET("/auth", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	output := captureSlog(func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/auth", nil)
		r.ServeHTTP(w, req)
	})

	var entry map[string]any
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("invalid JSON: %s", output)
	}
	if entry["user_id"] != "user-abc" {
		t.Errorf("expected user_id user-abc, got %q", entry["user_id"])
	}
}

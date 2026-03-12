package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/saas/city-stories-guide/backend/internal/logger"
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

func TestRequestLogger_IncludesTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TraceIDMiddleware())
	r.Use(RequestLogger())
	r.GET("/trace", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	output := captureSlog(func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/trace", nil)
		req.Header.Set(requestIDHeader, "trace-abc-123")
		r.ServeHTTP(w, req)
	})

	var entry map[string]any
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("invalid JSON: %s", output)
	}
	if entry["trace_id"] != "trace-abc-123" {
		t.Errorf("expected trace_id trace-abc-123, got %q", entry["trace_id"])
	}
}

// captureSlogRedacted is like captureSlog but wraps the JSON handler with
// logger.RedactHandler, simulating the production logging pipeline.
func captureSlogRedacted(fn func()) string {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	old := slog.Default()
	slog.SetDefault(slog.New(logger.NewRedactHandler(inner)))
	defer slog.SetDefault(old)
	fn()
	return buf.String()
}

func TestRedactHandler_SensitiveFieldsScrubbedInLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TraceIDMiddleware())
	r.Use(RequestLogger())
	r.POST("/auth/login", func(c *gin.Context) {
		// Simulate handler accidentally logging sensitive context.
		LoggerFromContext(c.Request.Context()).Info("auth attempt",
			"email", "user@example.com",
			"token", "secret-jwt-value",
			"user_id", "u-42",
		)
		c.Status(http.StatusOK)
	})

	output := captureSlogRedacted(func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", nil)
		req.Header.Set(requestIDHeader, "trace-redact-001")
		r.ServeHTTP(w, req)
	})

	// Parse each JSON line.
	lines := splitJSONLines(output)
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 log lines, got %d: %s", len(lines), output)
	}

	// Find the "auth attempt" log entry.
	var authEntry map[string]any
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry["msg"] == "auth attempt" {
			authEntry = entry
			break
		}
	}
	if authEntry == nil {
		t.Fatalf("could not find 'auth attempt' log entry in: %s", output)
	}

	// Sensitive fields must be redacted.
	if authEntry["email"] != logger.Placeholder {
		t.Errorf("email not redacted: %v", authEntry["email"])
	}
	if authEntry["token"] != logger.Placeholder {
		t.Errorf("token not redacted: %v", authEntry["token"])
	}

	// Non-sensitive fields must be preserved.
	if authEntry["user_id"] != "u-42" {
		t.Errorf("user_id should be preserved: %v", authEntry["user_id"])
	}
	if authEntry["trace_id"] != "trace-redact-001" {
		t.Errorf("trace_id should be preserved: %v", authEntry["trace_id"])
	}
}

func TestRedactHandler_CorrelationIDPreservedAfterRedaction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TraceIDMiddleware())
	r.Use(RequestLogger())
	r.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	output := captureSlogRedacted(func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set(requestIDHeader, "corr-id-999")
		r.ServeHTTP(w, req)
	})

	var entry map[string]any
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("invalid JSON: %s", output)
	}
	if entry["trace_id"] != "corr-id-999" {
		t.Errorf("expected trace_id corr-id-999, got %v", entry["trace_id"])
	}
	if entry["method"] != "GET" {
		t.Errorf("expected method GET, got %v", entry["method"])
	}
}

// splitJSONLines splits newline-delimited JSON into individual lines.
func splitJSONLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			line := s[start:i]
			if len(line) > 0 {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

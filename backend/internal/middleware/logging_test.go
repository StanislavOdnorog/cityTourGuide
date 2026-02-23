package middleware

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func captureLog(fn func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(nil)
		log.SetFlags(log.LstdFlags)
	}()
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

	output := captureLog(func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("User-Agent", "TestAgent/1.0")
		r.ServeHTTP(w, req)
	})

	var entry requestLog
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\nOutput: %s", err, output)
	}

	if entry.Method != "GET" {
		t.Errorf("expected method GET, got %s", entry.Method)
	}
	if entry.Path != "/test" {
		t.Errorf("expected path /test, got %s", entry.Path)
	}
	if entry.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", entry.StatusCode)
	}
	if entry.DurationMs < 0 {
		t.Errorf("expected non-negative duration, got %d", entry.DurationMs)
	}
	if entry.ClientIP == "" {
		t.Error("expected client_ip to be set")
	}
	if entry.UserAgent != "TestAgent/1.0" {
		t.Errorf("expected user_agent TestAgent/1.0, got %s", entry.UserAgent)
	}
	if entry.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}
}

func TestRequestLogger_InfoLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	output := captureLog(func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/ok", nil)
		r.ServeHTTP(w, req)
	})

	var entry requestLog
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("invalid JSON: %s", output)
	}
	if entry.Level != "info" {
		t.Errorf("expected level info for 200, got %s", entry.Level)
	}
}

func TestRequestLogger_WarnLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/bad", func(c *gin.Context) {
		c.Status(http.StatusBadRequest)
	})

	output := captureLog(func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/bad", nil)
		r.ServeHTTP(w, req)
	})

	var entry requestLog
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("invalid JSON: %s", output)
	}
	if entry.Level != "warn" {
		t.Errorf("expected level warn for 400, got %s", entry.Level)
	}
}

func TestRequestLogger_ErrorLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/err", func(c *gin.Context) {
		c.Status(http.StatusInternalServerError)
	})

	output := captureLog(func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/err", nil)
		r.ServeHTTP(w, req)
	})

	var entry requestLog
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("invalid JSON: %s", output)
	}
	if entry.Level != "error" {
		t.Errorf("expected level error for 500, got %s", entry.Level)
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

	output := captureLog(func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/auth", nil)
		r.ServeHTTP(w, req)
	})

	var entry requestLog
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("invalid JSON: %s", output)
	}
	if entry.UserID != "user-abc" {
		t.Errorf("expected user_id user-abc, got %q", entry.UserID)
	}
}

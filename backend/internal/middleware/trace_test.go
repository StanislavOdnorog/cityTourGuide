package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
)

var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestTraceIDMiddleware_GeneratesTraceIDWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TraceIDMiddleware())
	r.GET("/trace", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"trace_id": TraceID(c)})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/trace", nil)
	r.ServeHTTP(w, req)

	traceID := w.Header().Get(requestIDHeader)
	if !uuidV4Pattern.MatchString(traceID) {
		t.Fatalf("expected generated UUIDv4 trace ID, got %q", traceID)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if body["trace_id"] != traceID {
		t.Fatalf("expected body trace_id %q, got %q", traceID, body["trace_id"])
	}
}

func TestTraceIDMiddleware_ReusesValidRequestIDHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TraceIDMiddleware())
	r.GET("/trace", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"trace_id": TraceID(c)})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/trace", nil)
	req.Header.Set(requestIDHeader, "client-trace-123")
	r.ServeHTTP(w, req)

	if got := w.Header().Get(requestIDHeader); got != "client-trace-123" {
		t.Fatalf("expected response header to reuse request ID, got %q", got)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if body["trace_id"] != "client-trace-123" {
		t.Fatalf("expected body trace_id to reuse request ID, got %q", body["trace_id"])
	}
}

func TestTraceIDMiddleware_InvalidRequestIDGeneratesNewID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TraceIDMiddleware())
	r.GET("/trace", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"trace_id": TraceID(c)})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/trace", nil)
	req.Header.Set(requestIDHeader, "bad\ntrace")
	r.ServeHTTP(w, req)

	traceID := w.Header().Get(requestIDHeader)
	if traceID == "bad\ntrace" {
		t.Fatal("expected invalid request ID to be replaced")
	}
	if !uuidV4Pattern.MatchString(traceID) {
		t.Fatalf("expected generated UUIDv4 trace ID, got %q", traceID)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if body["trace_id"] != traceID {
		t.Fatalf("expected body trace_id %q, got %q", traceID, body["trace_id"])
	}
}

func TestTraceIDMiddleware_StoresTraceIDInGinContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TraceIDMiddleware())
	r.GET("/trace", func(c *gin.Context) {
		value, exists := c.Get(TraceIDKey)
		if !exists {
			t.Fatal("expected trace_id in Gin context")
		}
		traceID, ok := value.(string)
		if !ok || traceID == "" {
			t.Fatalf("expected string trace_id in Gin context, got %#v", value)
		}
		c.JSON(http.StatusOK, gin.H{"trace_id": traceID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/trace", nil)
	r.ServeHTTP(w, req)

	if got := w.Header().Get(requestIDHeader); got == "" {
		t.Fatal("expected X-Request-ID response header to be set")
	}
}

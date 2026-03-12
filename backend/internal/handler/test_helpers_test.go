package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func addTraceIDMiddleware(r *gin.Engine, traceID string) {
	r.Use(func(c *gin.Context) {
		c.Set("trace_id", traceID)
		c.Next()
	})
}

func newRouterWithTrace(traceID string, register func(*gin.Engine)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	addTraceIDMiddleware(r, traceID)
	register(r)
	return r
}

func assertValidationErrorResponse(t *testing.T, body []byte, expected map[string]string, requestID string) {
	t.Helper()

	var resp struct {
		Error     string             `json:"error"`
		Details   []validationDetail `json:"details"`
		RequestID string             `json:"request_id"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal validation response: %v", err)
	}

	if resp.Error != "validation_error" {
		t.Fatalf("expected validation_error, got %q", resp.Error)
	}
	if requestID != "" && resp.RequestID != requestID {
		t.Fatalf("expected request_id %q, got %q", requestID, resp.RequestID)
	}
	if len(resp.Details) < len(expected) {
		t.Fatalf("expected at least %d validation details, got %d", len(expected), len(resp.Details))
	}

	detailMap := make(map[string]string, len(resp.Details))
	for _, detail := range resp.Details {
		detailMap[detail.Field] = detail.Message
	}

	for field, message := range expected {
		if got, ok := detailMap[field]; !ok {
			t.Fatalf("expected validation detail for %q, got %+v", field, detailMap)
		} else if got != message {
			t.Fatalf("expected %q message %q, got %q", field, message, got)
		}
	}
}

func assertErrorResponse(t *testing.T, body []byte, expectedError, requestID string) {
	t.Helper()

	var resp struct {
		Error     string `json:"error"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}

	if resp.Error != expectedError {
		t.Fatalf("expected error %q, got %q", expectedError, resp.Error)
	}
	if requestID != "" && resp.RequestID != requestID {
		t.Fatalf("expected request_id %q, got %q", requestID, resp.RequestID)
	}
}

func assertErrorResponseContains(t *testing.T, body []byte, expectedSubstring string) {
	t.Helper()

	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if !strings.Contains(resp.Error, expectedSubstring) {
		t.Fatalf("expected error containing %q, got %q", expectedSubstring, resp.Error)
	}
}

package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

type errorResponseEnvelope struct {
	Error     string             `json:"error"`
	Details   []validationDetail `json:"details"`
	RequestID string             `json:"trace_id"`
}

type validationResponseExpectation struct {
	RequestID         string
	OrderedDetails    []validationDetail
	DetailsByField    map[string]string
	AllowExtraDetails bool
}

func newJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()

	var requestBody *bytes.Reader
	switch value := body.(type) {
	case nil:
		requestBody = bytes.NewReader(nil)
	case string:
		requestBody = bytes.NewReader([]byte(value))
	case []byte:
		requestBody = bytes.NewReader(value)
	default:
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		requestBody = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, path, requestBody)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func executeRequest(router http.Handler, req *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func executeJSONRequest(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return executeRequest(router, newJSONRequest(t, method, path, body))
}

func decodeErrorResponse(t *testing.T, body []byte) errorResponseEnvelope {
	t.Helper()

	var resp errorResponseEnvelope
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	return resp
}

func assertValidationErrorResponse(t *testing.T, body []byte, expected map[string]string, requestID string) {
	t.Helper()
	assertValidationResponse(t, http.StatusBadRequest, body, validationResponseExpectation{
		RequestID:      requestID,
		DetailsByField: expected,
	})
}

func assertValidationResponse(t *testing.T, statusCode int, body []byte, expected validationResponseExpectation) {
	t.Helper()

	if statusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", statusCode, string(body))
	}

	resp := decodeErrorResponse(t, body)
	if resp.Error != "validation_error" {
		t.Fatalf("expected validation_error, got %q", resp.Error)
	}
	if expected.RequestID != "" && resp.RequestID != expected.RequestID {
		t.Fatalf("expected trace_id %q, got %q", expected.RequestID, resp.RequestID)
	}

	if len(expected.OrderedDetails) > 0 {
		if !expected.AllowExtraDetails && len(resp.Details) != len(expected.OrderedDetails) {
			t.Fatalf("expected %d validation details, got %d", len(expected.OrderedDetails), len(resp.Details))
		}
		if len(resp.Details) < len(expected.OrderedDetails) {
			t.Fatalf("expected at least %d validation details, got %d", len(expected.OrderedDetails), len(resp.Details))
		}
		for i, detail := range expected.OrderedDetails {
			if resp.Details[i] != detail {
				t.Fatalf("expected detail[%d] = %+v, got %+v", i, detail, resp.Details[i])
			}
		}
		return
	}

	if !expected.AllowExtraDetails && len(resp.Details) != len(expected.DetailsByField) {
		t.Fatalf("expected %d validation details, got %d", len(expected.DetailsByField), len(resp.Details))
	}
	if len(resp.Details) < len(expected.DetailsByField) {
		t.Fatalf("expected at least %d validation details, got %d", len(expected.DetailsByField), len(resp.Details))
	}
	detailMap := make(map[string]string, len(resp.Details))
	for _, detail := range resp.Details {
		detailMap[detail.Field] = detail.Message
	}

	for field, message := range expected.DetailsByField {
		if got, ok := detailMap[field]; !ok {
			t.Fatalf("expected validation detail for %q, got %+v", field, detailMap)
		} else if got != message {
			t.Fatalf("expected %q message %q, got %q", field, message, got)
		}
	}
}

func assertErrorResponse(t *testing.T, body []byte, expectedError, requestID string) {
	t.Helper()

	resp := decodeErrorResponse(t, body)
	if resp.Error != expectedError {
		t.Fatalf("expected error %q, got %q", expectedError, resp.Error)
	}
	if requestID != "" && resp.RequestID != requestID {
		t.Fatalf("expected trace_id %q, got %q", requestID, resp.RequestID)
	}
}

func assertErrorResponseContains(t *testing.T, body []byte, expectedSubstring string) {
	t.Helper()

	resp := decodeErrorResponse(t, body)
	if !strings.Contains(resp.Error, expectedSubstring) {
		t.Fatalf("expected error containing %q, got %q", expectedSubstring, resp.Error)
	}
}

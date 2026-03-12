package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestErrorJSON_IncludesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/err", func(c *gin.Context) {
		c.Set("trace_id", "trace-123")
		errorJSON(c, http.StatusBadRequest, "bad request")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/err", nil)
	r.ServeHTTP(w, req)

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if body["error"] != "bad request" {
		t.Fatalf("expected error body, got %q", body["error"])
	}
	if body["request_id"] != "trace-123" {
		t.Fatalf("expected request_id trace-123, got %q", body["request_id"])
	}
}

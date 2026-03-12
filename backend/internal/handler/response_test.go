package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

func TestErrorJSON_IncludesTraceID(t *testing.T) {
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
	if body["trace_id"] != "trace-123" {
		t.Fatalf("expected trace_id trace-123, got %q", body["trace_id"])
	}
}

func TestErrorJSON_OmitsTraceIDWhenAbsent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/err", nil)

	errorJSON(c, http.StatusBadRequest, "bad request")

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if body["error"] != "bad request" {
		t.Fatalf("expected error body, got %v", body["error"])
	}
	if _, exists := body["trace_id"]; exists {
		t.Fatal("trace_id should not be present when not set in context")
	}
}

func TestWriteCursorPage_NilItemsBecomesEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	page := &domain.PageResponse[string]{
		Items:      nil,
		NextCursor: "",
		HasMore:    false,
	}
	writeCursorPage(c, page)

	body := w.Body.String()
	var resp map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if string(resp["items"]) != "[]" {
		t.Errorf("expected items=[], got %s", string(resp["items"]))
	}
	if string(resp["has_more"]) != "false" {
		t.Errorf("expected has_more=false, got %s", string(resp["has_more"]))
	}
	if string(resp["next_cursor"]) != `""` {
		t.Errorf("expected next_cursor empty string, got %s", string(resp["next_cursor"]))
	}
}

func TestWriteCursorPage_NonEmptyItems(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	page := &domain.PageResponse[string]{
		Items:      []string{"a", "b"},
		NextCursor: "cursor123",
		HasMore:    true,
	}
	writeCursorPage(c, page)

	var resp struct {
		Items      []string `json:"items"`
		NextCursor string   `json:"next_cursor"`
		HasMore    bool     `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Items) != 2 || resp.Items[0] != "a" || resp.Items[1] != "b" {
		t.Errorf("unexpected items: %v", resp.Items)
	}
	if resp.NextCursor != "cursor123" {
		t.Errorf("expected next_cursor=cursor123, got %q", resp.NextCursor)
	}
	if !resp.HasMore {
		t.Error("expected has_more=true")
	}
}

func TestWriteCursorPageItems_NilItemsBecomesEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	writeCursorPageItems[string](c, nil, "", false)

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if string(resp["items"]) != "[]" {
		t.Errorf("expected items=[], got %s", string(resp["items"]))
	}
}

func TestWriteCursorPage_StatusOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	writeCursorPage(c, &domain.PageResponse[int]{Items: []int{1}})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestErrorJSONWithFields_IncludesTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/err", nil)
	c.Set("trace_id", "trace-456")

	errorJSONWithFields(c, http.StatusConflict, "conflict", gin.H{"resource": "city"})

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if body["error"] != "conflict" {
		t.Fatalf("expected error 'conflict', got %v", body["error"])
	}
	if body["trace_id"] != "trace-456" {
		t.Fatalf("expected trace_id 'trace-456', got %v", body["trace_id"])
	}
	if body["resource"] != "city" {
		t.Fatalf("expected resource 'city', got %v", body["resource"])
	}
}

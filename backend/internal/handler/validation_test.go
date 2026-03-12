package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func TestValidationErrorResponse_ValidatorErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type testStruct struct {
		Email    string `validate:"required,email"`
		Password string `validate:"required,min=8"`
	}

	validate := validator.New()
	err := validate.Struct(&testStruct{})
	if err == nil {
		t.Fatal("expected validation error")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	validationErrorResponse(c, err)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp struct {
		Error   string             `json:"error"`
		Details []validationDetail `json:"details"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error != "validation_error" {
		t.Errorf("expected error 'validation_error', got %q", resp.Error)
	}

	if len(resp.Details) != 2 {
		t.Fatalf("expected 2 details, got %d", len(resp.Details))
	}

	detailMap := make(map[string]string)
	for _, d := range resp.Details {
		detailMap[d.Field] = d.Message
	}

	if msg, ok := detailMap["email"]; !ok {
		t.Error("expected 'email' field in details")
	} else if msg != "this field is required" {
		t.Errorf("expected 'this field is required' for email, got %q", msg)
	}

	if msg, ok := detailMap["password"]; !ok {
		t.Error("expected 'password' field in details")
	} else if msg != "this field is required" {
		t.Errorf("expected 'this field is required' for password, got %q", msg)
	}
}

func TestValidationErrorResponse_NonValidatorError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	validationErrorResponse(c, fmt.Errorf("some other error"))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error != "some other error" {
		t.Errorf("expected 'some other error', got %q", resp.Error)
	}
}

func TestValidationErrorResponse_WithTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type testStruct struct {
		Name string `validate:"required"`
	}

	validate := validator.New()
	err := validate.Struct(&testStruct{})
	if err == nil {
		t.Fatal("expected validation error")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set("trace_id", "abc-123")

	validationErrorResponse(c, err)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["request_id"] != "abc-123" {
		t.Errorf("expected request_id 'abc-123', got %v", resp["request_id"])
	}
}

func TestFieldMessage_Tags(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type testStruct struct {
		Email string `validate:"required,email"`
		Name  string `validate:"min=3,max=50"`
		URL   string `validate:"url"`
	}

	validate := validator.New()

	// Test email tag
	err := validate.Struct(&testStruct{Email: "notanemail", Name: "Jo", URL: "notaurl"})
	if err == nil {
		t.Fatal("expected validation error")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	validationErrorResponse(c, err)

	var resp struct {
		Details []validationDetail `json:"details"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	detailMap := make(map[string]string)
	for _, d := range resp.Details {
		detailMap[d.Field] = d.Message
	}

	if msg := detailMap["email"]; msg != "invalid email format" {
		t.Errorf("expected 'invalid email format' for email, got %q", msg)
	}
	if msg := detailMap["name"]; msg != "must be at least 3 characters" {
		t.Errorf("expected 'must be at least 3 characters' for name, got %q", msg)
	}
	if msg := detailMap["url"]; msg != "invalid URL format" {
		t.Errorf("expected 'invalid URL format' for url, got %q", msg)
	}
}

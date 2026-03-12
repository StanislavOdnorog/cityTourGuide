package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	if resp["trace_id"] != "abc-123" {
		t.Errorf("expected trace_id 'abc-123', got %v", resp["trace_id"])
	}
}

func TestValidationErrorResponse_OmitsTraceIDWhenAbsent(t *testing.T) {
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

	validationErrorResponse(c, err)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if _, exists := resp["trace_id"]; exists {
		t.Error("trace_id should not be present when not set in context")
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

// --- Query parsing helper tests ---

// newQueryContext creates a gin.Context with the given query parameters.
func newQueryContext(query url.Values) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+query.Encode(), nil)
	return w, c
}

func getErrorMessage(w *httptest.ResponseRecorder, t *testing.T) string {
	t.Helper()
	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	return resp.Error
}

func TestParseRequiredFloat(t *testing.T) {
	tests := []struct {
		name     string
		query    url.Values
		min, max float64
		wantVal  float64
		wantOK   bool
		wantErr  string
	}{
		{"valid", url.Values{"lat": {"41.7"}}, -90, 90, 41.7, true, ""},
		{"missing", url.Values{}, -90, 90, 0, false, "lat is required"},
		{"not a number", url.Values{"lat": {"abc"}}, -90, 90, 0, false, "lat must be a valid number"},
		{"too high", url.Values{"lat": {"91"}}, -90, 90, 0, false, "lat must be between -90 and 90"},
		{"too low", url.Values{"lat": {"-91"}}, -90, 90, 0, false, "lat must be between -90 and 90"},
		{"boundary low", url.Values{"lat": {"-90"}}, -90, 90, -90, true, ""},
		{"boundary high", url.Values{"lat": {"90"}}, -90, 90, 90, true, ""},
		{"lng valid", url.Values{"lat": {"180"}}, -180, 180, 180, true, ""},
		{"lng too high", url.Values{"lat": {"181"}}, -180, 180, 0, false, "lat must be between -180 and 180"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, c := newQueryContext(tt.query)
			val, ok := parseRequiredFloat(c, "lat", tt.min, tt.max)
			if ok != tt.wantOK {
				t.Fatalf("ok=%v, want %v", ok, tt.wantOK)
			}
			if ok {
				if val != tt.wantVal {
					t.Errorf("val=%v, want %v", val, tt.wantVal)
				}
			} else {
				if w.Code != http.StatusBadRequest {
					t.Errorf("status=%d, want 400", w.Code)
				}
				if msg := getErrorMessage(w, t); msg != tt.wantErr {
					t.Errorf("error=%q, want %q", msg, tt.wantErr)
				}
			}
		})
	}
}

func TestParseOptionalFloat(t *testing.T) {
	tests := []struct {
		name       string
		query      url.Values
		defaultVal float64
		min, max   float64
		wantVal    float64
		wantOK     bool
		wantErr    string
	}{
		{"missing uses default", url.Values{}, 150, 10, 500, 150, true, ""},
		{"valid", url.Values{"radius": {"200"}}, 150, 10, 500, 200, true, ""},
		{"not a number", url.Values{"radius": {"big"}}, 150, 10, 500, 0, false, "radius must be a valid number"},
		{"too small", url.Values{"radius": {"5"}}, 150, 10, 500, 0, false, "radius must be between 10 and 500"},
		{"too large", url.Values{"radius": {"501"}}, 150, 10, 500, 0, false, "radius must be between 10 and 500"},
		{"boundary low", url.Values{"radius": {"10"}}, 150, 10, 500, 10, true, ""},
		{"boundary high", url.Values{"radius": {"500"}}, 150, 10, 500, 500, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, c := newQueryContext(tt.query)
			val, ok := parseOptionalFloat(c, "radius", tt.defaultVal, tt.min, tt.max)
			if ok != tt.wantOK {
				t.Fatalf("ok=%v, want %v", ok, tt.wantOK)
			}
			if ok {
				if val != tt.wantVal {
					t.Errorf("val=%v, want %v", val, tt.wantVal)
				}
			} else {
				if w.Code != http.StatusBadRequest {
					t.Errorf("status=%d, want 400", w.Code)
				}
				if msg := getErrorMessage(w, t); msg != tt.wantErr {
					t.Errorf("error=%q, want %q", msg, tt.wantErr)
				}
			}
		})
	}
}

func TestParseRequiredQueryInt(t *testing.T) {
	tests := []struct {
		name    string
		query   url.Values
		wantVal int
		wantOK  bool
		wantErr string
	}{
		{"valid", url.Values{"city_id": {"5"}}, 5, true, ""},
		{"missing", url.Values{}, 0, false, "city_id is required"},
		{"not a number", url.Values{"city_id": {"abc"}}, 0, false, "city_id must be a positive integer"},
		{"zero", url.Values{"city_id": {"0"}}, 0, false, "city_id must be a positive integer"},
		{"negative", url.Values{"city_id": {"-1"}}, 0, false, "city_id must be a positive integer"},
		{"float", url.Values{"city_id": {"1.5"}}, 0, false, "city_id must be a positive integer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, c := newQueryContext(tt.query)
			val, ok := parseRequiredQueryInt(c, "city_id")
			if ok != tt.wantOK {
				t.Fatalf("ok=%v, want %v", ok, tt.wantOK)
			}
			if ok {
				if val != tt.wantVal {
					t.Errorf("val=%v, want %v", val, tt.wantVal)
				}
			} else {
				if w.Code != http.StatusBadRequest {
					t.Errorf("status=%d, want 400", w.Code)
				}
				if msg := getErrorMessage(w, t); msg != tt.wantErr {
					t.Errorf("error=%q, want %q", msg, tt.wantErr)
				}
			}
		})
	}
}

func TestParseRequiredUUIDQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   url.Values
		wantVal string
		wantOK  bool
		wantErr string
	}{
		{"valid", url.Values{"user_id": {"550e8400-e29b-41d4-a716-446655440000"}}, "550e8400-e29b-41d4-a716-446655440000", true, ""},
		{"missing", url.Values{}, "", false, "user_id is required"},
		{"empty", url.Values{"user_id": {""}}, "", false, "user_id is required"},
		{"not a uuid", url.Values{"user_id": {"not-a-uuid"}}, "", false, "user_id must be a valid UUID"},
		{"too short", url.Values{"user_id": {"550e8400"}}, "", false, "user_id must be a valid UUID"},
		{"no dashes", url.Values{"user_id": {"550e8400e29b41d4a716446655440000"}}, "", false, "user_id must be a valid UUID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, c := newQueryContext(tt.query)
			val, ok := parseRequiredUUIDQuery(c, "user_id")
			if ok != tt.wantOK {
				t.Fatalf("ok=%v, want %v", ok, tt.wantOK)
			}
			if ok {
				if val != tt.wantVal {
					t.Errorf("val=%q, want %q", val, tt.wantVal)
				}
			} else {
				if w.Code != http.StatusBadRequest {
					t.Errorf("status=%d, want 400", w.Code)
				}
				if msg := getErrorMessage(w, t); msg != tt.wantErr {
					t.Errorf("error=%q, want %q", msg, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateCoordPair(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	tests := []struct {
		name    string
		lat     *float64
		lng     *float64
		wantOK  bool
		wantErr string
	}{
		{"both nil", nil, nil, true, ""},
		{"both valid", f(41.7), f(44.8), true, ""},
		{"lat only", f(41.7), nil, false, "lat and lng must both be provided or both omitted"},
		{"lng only", nil, f(44.8), false, "lat and lng must both be provided or both omitted"},
		{"lat too high", f(91), f(0), false, "lat must be between -90 and 90"},
		{"lat too low", f(-91), f(0), false, "lat must be between -90 and 90"},
		{"lng too high", f(0), f(181), false, "lng must be between -180 and 180"},
		{"lng too low", f(0), f(-181), false, "lng must be between -180 and 180"},
		{"boundary lat", f(-90), f(180), true, ""},
		{"boundary lat high", f(90), f(-180), true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, c := newQueryContext(url.Values{})
			ok := validateCoordPair(c, tt.lat, tt.lng)
			if ok != tt.wantOK {
				t.Fatalf("ok=%v, want %v", ok, tt.wantOK)
			}
			if !ok {
				if w.Code != http.StatusBadRequest {
					t.Errorf("status=%d, want 400", w.Code)
				}
				if msg := getErrorMessage(w, t); msg != tt.wantErr {
					t.Errorf("error=%q, want %q", msg, tt.wantErr)
				}
			}
		})
	}
}

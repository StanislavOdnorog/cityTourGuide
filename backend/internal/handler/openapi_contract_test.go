package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
	"github.com/saas/city-stories-guide/backend/internal/service"
)

// ---------- helpers ----------

// jsonKeys decodes a JSON body and returns top-level keys.
func jsonKeys(t *testing.T, body []byte) map[string]json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("unmarshal response body: %v\nbody: %s", err, string(body))
	}
	return m
}

// requireKeys asserts that every key in want exists in the decoded JSON map.
func requireKeys(t *testing.T, body []byte, want ...string) map[string]json.RawMessage {
	t.Helper()
	m := jsonKeys(t, body)
	for _, k := range want {
		if _, ok := m[k]; !ok {
			t.Fatalf("expected key %q in response, got keys %v\nbody: %s", k, keysOf(m), string(body))
		}
	}
	return m
}

func keysOf(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// requirePaginatedKeys asserts items, has_more, next_cursor exist and items is an array.
func requirePaginatedKeys(t *testing.T, body []byte) {
	t.Helper()
	m := requireKeys(t, body, "items", "has_more", "next_cursor")

	// items must be a JSON array (not null)
	raw := m["items"]
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		t.Fatalf("expected items to be a JSON array, got: %s", string(raw))
	}
}

// requireErrorEnvelope asserts the standard error envelope with "error" and optionally "trace_id".
func requireErrorEnvelope(t *testing.T, body []byte) map[string]json.RawMessage {
	t.Helper()
	return requireKeys(t, body, "error")
}

// requireValidationEnvelope asserts the structured validation error body.
func requireValidationEnvelope(t *testing.T, body []byte) {
	t.Helper()
	m := requireKeys(t, body, "error", "details")

	var errVal string
	if err := json.Unmarshal(m["error"], &errVal); err != nil {
		t.Fatalf("expected error to be a string: %v", err)
	}
	if errVal != "validation_error" {
		t.Fatalf("expected error=%q, got %q", "validation_error", errVal)
	}

	// details must be an array
	trimmed := bytes.TrimSpace(m["details"])
	if len(trimmed) == 0 || trimmed[0] != '[' {
		t.Fatalf("expected details to be a JSON array, got: %s", string(trimmed))
	}

	// Each detail must have field + message
	var details []map[string]json.RawMessage
	if err := json.Unmarshal(m["details"], &details); err != nil {
		t.Fatalf("unmarshal details: %v", err)
	}
	for i, d := range details {
		if _, ok := d["field"]; !ok {
			t.Fatalf("details[%d] missing 'field'", i)
		}
		if _, ok := d["message"]; !ok {
			t.Fatalf("details[%d] missing 'message'", i)
		}
	}
}

// contractRouter wires up all the endpoints under test with the given mocks.
func contractRouter(
	cityRepo *mockCityRepo,
	manifestRepo *mockManifestRepo,
	reportRepo *mockReportRepo,
	modSvc *mockReportModerationService,
	authSvc *mockAuthService,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("trace_id", "contract-trace")
		c.Next()
	})

	cityH := NewCityHandler(cityRepo, manifestRepo, nil)
	reportH := NewReportHandler(reportRepo, modSvc, nil)
	authH := NewAuthHandler(authSvc)

	r.GET("/api/v1/cities", cityH.ListCities)
	r.GET("/api/v1/cities/:id", cityH.GetCity)
	r.POST("/api/v1/admin/cities", cityH.CreateCity)

	r.POST("/api/v1/reports", reportH.CreateReport)
	r.GET("/api/v1/admin/reports", reportH.ListReports)

	auth := r.Group("/api/v1/auth")
	auth.POST("/login", authH.Login)

	return r
}

// ---------- contract tests ----------

// TestContract_ListCities_Populated verifies the paginated response shape for a populated list.
func TestContract_ListCities_Populated(t *testing.T) {
	repo := &mockCityRepo{
		cities: []domain.City{
			{ID: 1, Name: "Tbilisi", Country: "GE", CenterLat: 41.7, CenterLng: 44.8, RadiusKm: 15, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		},
	}
	r := contractRouter(repo, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, &mockAuthService{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	requirePaginatedKeys(t, w.Body.Bytes())

	// Verify item shape has documented city keys
	var resp struct {
		Items []map[string]json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) == 0 {
		t.Fatal("expected at least 1 item")
	}
	cityKeys := []string{"id", "name", "country", "center_lat", "center_lng", "radius_km", "is_active"}
	for _, k := range cityKeys {
		if _, ok := resp.Items[0][k]; !ok {
			t.Errorf("city item missing documented key %q", k)
		}
	}
}

// TestContract_ListCities_Empty verifies empty paginated response still includes all cursor fields.
func TestContract_ListCities_Empty(t *testing.T) {
	repo := &mockCityRepo{cities: nil}
	r := contractRouter(repo, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, &mockAuthService{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	requirePaginatedKeys(t, w.Body.Bytes())

	// items must be an empty array, not null
	var resp struct {
		Items   []json.RawMessage `json:"items"`
		HasMore bool              `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Items == nil {
		t.Fatal("items must be an empty array, not null")
	}
	if len(resp.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(resp.Items))
	}
	if resp.HasMore {
		t.Fatal("expected has_more=false for empty result")
	}
}

// TestContract_GetCity_Success verifies single city response shape.
func TestContract_GetCity_Success(t *testing.T) {
	repo := &mockCityRepo{
		city: &domain.City{ID: 1, Name: "Tbilisi", Country: "GE", CenterLat: 41.7, CenterLng: 44.8, RadiusKm: 15, IsActive: true},
	}
	r := contractRouter(repo, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, &mockAuthService{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	m := requireKeys(t, w.Body.Bytes(), "data")

	// data must be an object with city keys
	var city map[string]json.RawMessage
	if err := json.Unmarshal(m["data"], &city); err != nil {
		t.Fatalf("data should be an object: %v", err)
	}
	for _, k := range []string{"id", "name", "country", "center_lat", "center_lng", "radius_km", "is_active"} {
		if _, ok := city[k]; !ok {
			t.Errorf("city data missing key %q", k)
		}
	}
}

// TestContract_GetCity_NotFound verifies error envelope for 404.
func TestContract_GetCity_NotFound(t *testing.T) {
	repo := &mockCityRepo{err: repository.ErrNotFound}
	r := contractRouter(repo, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, &mockAuthService{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/999", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	m := requireErrorEnvelope(t, w.Body.Bytes())

	var errMsg string
	if err := json.Unmarshal(m["error"], &errMsg); err != nil {
		t.Fatal(err)
	}
	if errMsg != "city not found" {
		t.Fatalf("expected error=%q, got %q", "city not found", errMsg)
	}

	// trace_id should be present when set
	requireKeys(t, w.Body.Bytes(), "trace_id")
}

// TestContract_GetCity_InvalidID verifies error envelope for bad id parameter.
func TestContract_GetCity_InvalidID(t *testing.T) {
	r := contractRouter(&mockCityRepo{}, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, &mockAuthService{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cities/abc", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	requireErrorEnvelope(t, w.Body.Bytes())
}

// TestContract_CreateCity_ValidationError verifies validation error envelope on missing fields.
func TestContract_CreateCity_ValidationError(t *testing.T) {
	r := contractRouter(&mockCityRepo{}, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, &mockAuthService{})

	body := `{"name":"Tbilisi"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	requireValidationEnvelope(t, w.Body.Bytes())

	// trace_id should be present
	requireKeys(t, w.Body.Bytes(), "trace_id")
}

// TestContract_CreateCity_Success verifies the admin mutation response shape.
func TestContract_CreateCity_Success(t *testing.T) {
	r := contractRouter(&mockCityRepo{}, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, &mockAuthService{})

	body := `{"name":"Tbilisi","country":"GE","center_lat":41.7,"center_lng":44.8,"radius_km":15}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cities", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	m := requireKeys(t, w.Body.Bytes(), "data")

	var city map[string]json.RawMessage
	if err := json.Unmarshal(m["data"], &city); err != nil {
		t.Fatalf("data should be an object: %v", err)
	}
	for _, k := range []string{"id", "name", "country", "is_active"} {
		if _, ok := city[k]; !ok {
			t.Errorf("created city missing key %q", k)
		}
	}
}

// TestContract_CreateReport_ValidationError verifies validation error envelope for reports.
func TestContract_CreateReport_ValidationError(t *testing.T) {
	r := contractRouter(&mockCityRepo{}, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, &mockAuthService{})

	// Missing required fields
	body := `{"story_id":10}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	requireValidationEnvelope(t, w.Body.Bytes())
}

// TestContract_CreateReport_BadRequest verifies the plain error envelope for non-validation errors.
func TestContract_CreateReport_BadRequest(t *testing.T) {
	r := contractRouter(&mockCityRepo{}, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, &mockAuthService{})

	// Invalid type (not a validation error, a plain error)
	body := `{"story_id":10,"user_id":"550e8400-e29b-41d4-a716-446655440000","type":"unknown_type"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	requireErrorEnvelope(t, w.Body.Bytes())
}

// TestContract_CreateReport_Success verifies the created report response shape.
func TestContract_CreateReport_Success(t *testing.T) {
	r := contractRouter(&mockCityRepo{}, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, &mockAuthService{})

	body := `{"story_id":10,"user_id":"550e8400-e29b-41d4-a716-446655440000","type":"wrong_fact"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	m := requireKeys(t, w.Body.Bytes(), "data")

	var report map[string]json.RawMessage
	if err := json.Unmarshal(m["data"], &report); err != nil {
		t.Fatalf("data should be an object: %v", err)
	}
	for _, k := range []string{"id", "status"} {
		if _, ok := report[k]; !ok {
			t.Errorf("report missing key %q", k)
		}
	}
}

// TestContract_AdminReports_Populated verifies admin reports paginated response shape.
func TestContract_AdminReports_Populated(t *testing.T) {
	poiID := 5
	poiName := "Test POI"
	lang := "en"
	storyStatus := "active"
	repo := &mockReportRepo{
		adminReports: []domain.AdminReportListItem{
			{
				Report:        domain.Report{ID: 1, StoryID: 10, UserID: "u1", Type: domain.ReportTypeWrongFact, Status: domain.ReportStatusNew, CreatedAt: time.Now()},
				POIID:         &poiID,
				POIName:       &poiName,
				StoryLanguage: &lang,
				StoryStatus:   &storyStatus,
			},
		},
	}
	r := contractRouter(&mockCityRepo{}, &mockManifestRepo{}, repo, &mockReportModerationService{}, &mockAuthService{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	requirePaginatedKeys(t, w.Body.Bytes())

	// Verify item shape includes report and joined fields
	var resp struct {
		Items []map[string]json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) == 0 {
		t.Fatal("expected at least 1 item")
	}
	for _, k := range []string{"id", "story_id", "type", "status", "poi_id", "poi_name", "story_language", "story_status"} {
		if _, ok := resp.Items[0][k]; !ok {
			t.Errorf("admin report item missing key %q", k)
		}
	}
}

// TestContract_AdminReports_Empty verifies empty admin reports still returns cursor fields.
func TestContract_AdminReports_Empty(t *testing.T) {
	r := contractRouter(&mockCityRepo{}, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, &mockAuthService{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	requirePaginatedKeys(t, w.Body.Bytes())

	var resp struct {
		Items   []json.RawMessage `json:"items"`
		HasMore bool              `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Items == nil {
		t.Fatal("items must be an empty array, not null")
	}
	if len(resp.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(resp.Items))
	}
}

// TestContract_AuthLogin_Success verifies login response shape with data + tokens.
func TestContract_AuthLogin_Success(t *testing.T) {
	mock := &mockAuthService{
		loginFn: func(_ context.Context, _, _ string) (*domain.User, *service.TokenPair, error) {
			email := "test@example.com"
			return &domain.User{
				ID:           "user-uuid-123",
				Email:        &email,
				AuthProvider: domain.AuthProviderEmail,
				LanguagePref: "en",
			}, &service.TokenPair{
				AccessToken:  "access",
				RefreshToken: "refresh",
				ExpiresIn:    900,
			}, nil
		},
	}
	r := contractRouter(&mockCityRepo{}, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, mock)

	body := `{"email":"test@example.com","password":"password123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	m := requireKeys(t, w.Body.Bytes(), "data", "tokens")

	// Verify user shape
	var user map[string]json.RawMessage
	if err := json.Unmarshal(m["data"], &user); err != nil {
		t.Fatalf("data should be an object: %v", err)
	}
	for _, k := range []string{"id", "auth_provider"} {
		if _, ok := user[k]; !ok {
			t.Errorf("user data missing key %q", k)
		}
	}

	// Verify tokens shape
	var tokens map[string]json.RawMessage
	if err := json.Unmarshal(m["tokens"], &tokens); err != nil {
		t.Fatalf("tokens should be an object: %v", err)
	}
	for _, k := range []string{"access_token", "refresh_token", "expires_in"} {
		if _, ok := tokens[k]; !ok {
			t.Errorf("tokens missing key %q", k)
		}
	}
}

// TestContract_AuthLogin_InvalidCredentials verifies 401 error envelope.
func TestContract_AuthLogin_InvalidCredentials(t *testing.T) {
	mock := &mockAuthService{
		loginFn: func(_ context.Context, _, _ string) (*domain.User, *service.TokenPair, error) {
			return nil, nil, service.ErrInvalidCredentials
		},
	}
	r := contractRouter(&mockCityRepo{}, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, mock)

	body := `{"email":"test@example.com","password":"wrongpass"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	m := requireErrorEnvelope(t, w.Body.Bytes())

	var errMsg string
	if err := json.Unmarshal(m["error"], &errMsg); err != nil {
		t.Fatal(err)
	}
	if errMsg != "invalid email or password" {
		t.Fatalf("expected error=%q, got %q", "invalid email or password", errMsg)
	}
}

// TestContract_AuthLogin_ValidationError verifies validation envelope on missing fields.
func TestContract_AuthLogin_ValidationError(t *testing.T) {
	r := contractRouter(&mockCityRepo{}, &mockManifestRepo{}, &mockReportRepo{}, &mockReportModerationService{}, &mockAuthService{})

	body := `{"email":"test@example.com"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	requireValidationEnvelope(t, w.Body.Bytes())
}

// TestContract_ErrorEnvelope_IncludesTraceID verifies trace_id in all error responses.
func TestContract_ErrorEnvelope_IncludesTraceID(t *testing.T) {
	r := contractRouter(
		&mockCityRepo{err: repository.ErrNotFound},
		&mockManifestRepo{},
		&mockReportRepo{},
		&mockReportModerationService{},
		&mockAuthService{},
	)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{"city not found", http.MethodGet, "/api/v1/cities/999", "", http.StatusNotFound},
		{"invalid city id", http.MethodGet, "/api/v1/cities/abc", "", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Fatalf("expected %d, got %d: %s", tt.status, w.Code, w.Body.String())
			}

			m := requireErrorEnvelope(t, w.Body.Bytes())

			// trace_id must be present
			if _, ok := m["trace_id"]; !ok {
				t.Fatalf("expected trace_id in error response, got keys %v", keysOf(m))
			}

			var traceID string
			if err := json.Unmarshal(m["trace_id"], &traceID); err != nil {
				t.Fatalf("trace_id should be a string: %v", err)
			}
			if traceID != "contract-trace" {
				t.Fatalf("expected trace_id=%q, got %q", "contract-trace", traceID)
			}
		})
	}
}

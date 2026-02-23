package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupCORSRouter(origins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(CORSConfig{AllowedOrigins: origins}))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestCORS_AllowedOrigin(t *testing.T) {
	r := setupCORSRouter([]string{"http://localhost:5173", "https://admin.example.com"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("expected Allow-Origin http://localhost:5173, got %q", got)
	}
}

func TestCORS_BlockedOrigin(t *testing.T) {
	r := setupCORSRouter([]string{"http://localhost:5173"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	r := setupCORSRouter([]string{"http://localhost:5173"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 without Origin header, got %d", w.Code)
	}

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("should not set Allow-Origin without Origin header, got %q", got)
	}
}

func TestCORS_PreflightOptions(t *testing.T) {
	r := setupCORSRouter([]string{"http://localhost:5173"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for preflight, got %d", w.Code)
	}

	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("expected Allow-Methods header for preflight")
	}

	if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("expected Allow-Headers header for preflight")
	}
}

func TestCORS_TrailingSlashNormalization(t *testing.T) {
	r := setupCORSRouter([]string{"http://localhost:5173"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:5173/")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with trailing slash origin, got %d", w.Code)
	}
}

func TestCORS_CredentialsHeader(t *testing.T) {
	r := setupCORSRouter([]string{"http://localhost:5173"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected Allow-Credentials true, got %q", got)
	}
}

func TestCORS_MultipleAllowedOrigins(t *testing.T) {
	r := setupCORSRouter([]string{"http://localhost:5173", "https://admin.example.com"})

	// First origin
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req1.Header.Set("Origin", "https://admin.example.com")
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("expected 200 for second allowed origin, got %d", w1.Code)
	}

	if got := w1.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.example.com" {
		t.Errorf("expected Allow-Origin https://admin.example.com, got %q", got)
	}
}

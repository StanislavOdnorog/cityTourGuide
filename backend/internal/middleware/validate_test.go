package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupValidateRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ValidateGPSParams())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestValidateGPS_ValidCoordinates(t *testing.T) {
	r := setupValidateRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test?lat=41.7151&lng=44.8271", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestValidateGPS_BoundaryValues(t *testing.T) {
	r := setupValidateRouter()

	tests := []struct {
		name  string
		query string
	}{
		{"lat=90", "/test?lat=90&lng=0"},
		{"lat=-90", "/test?lat=-90&lng=0"},
		{"lng=180", "/test?lat=0&lng=180"},
		{"lng=-180", "/test?lat=0&lng=-180"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, tt.query, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200 for %s, got %d", tt.name, w.Code)
			}
		})
	}
}

func TestValidateGPS_LatTooHigh(t *testing.T) {
	r := setupValidateRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test?lat=91&lng=44", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestValidateGPS_LatTooLow(t *testing.T) {
	r := setupValidateRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test?lat=-91&lng=44", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestValidateGPS_LngTooHigh(t *testing.T) {
	r := setupValidateRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test?lat=41&lng=181", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestValidateGPS_LngTooLow(t *testing.T) {
	r := setupValidateRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test?lat=41&lng=-181", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestValidateGPS_LatNotANumber(t *testing.T) {
	r := setupValidateRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test?lat=abc&lng=44", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestValidateGPS_LngNotANumber(t *testing.T) {
	r := setupValidateRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test?lat=41&lng=xyz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestValidateGPS_NoGPSParams(t *testing.T) {
	r := setupValidateRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test?page=1&per_page=20", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 without GPS params, got %d", w.Code)
	}
}

func TestValidateGPS_LatWithout999_RejectBeforeDB(t *testing.T) {
	r := setupValidateRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test?lat=999", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("lat=999 should be rejected with 400, got %d", w.Code)
	}
}

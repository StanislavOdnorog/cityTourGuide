package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockTokenValidator struct {
	validateFn func(token string) (string, error)
}

func (m *mockTokenValidator) ValidateAccessToken(token string) (string, error) {
	return m.validateFn(token)
}

func setupTestRouter(mw gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", mw, func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})
	return r
}

func TestJWTAuth_ValidToken(t *testing.T) {
	validator := &mockTokenValidator{
		validateFn: func(token string) (string, error) {
			if token == "valid-token" {
				return "user-123", nil
			}
			return "", errors.New("invalid")
		},
	}

	r := setupTestRouter(JWTAuth(validator))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	validator := &mockTokenValidator{}
	r := setupTestRouter(JWTAuth(validator))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	validator := &mockTokenValidator{}
	r := setupTestRouter(JWTAuth(validator))

	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "just-a-token"},
		{"wrong prefix", "Basic token123"},
		{"empty after bearer", "Bearer "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", tt.header)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", w.Code)
			}
		})
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	validator := &mockTokenValidator{
		validateFn: func(_ string) (string, error) {
			return "", errors.New("token expired")
		},
	}
	r := setupTestRouter(JWTAuth(validator))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_CaseInsensitiveBearer(t *testing.T) {
	validator := &mockTokenValidator{
		validateFn: func(token string) (string, error) {
			return "user-123", nil
		},
	}
	r := setupTestRouter(JWTAuth(validator))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "bearer valid-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestJWTAuth_UserIDInContext(t *testing.T) {
	validator := &mockTokenValidator{
		validateFn: func(_ string) (string, error) {
			return "user-456", nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	var contextUserID string
	r.GET("/test", JWTAuth(validator), func(c *gin.Context) {
		id, exists := c.Get("user_id")
		if !exists {
			t.Error("user_id should exist in context")
		}
		contextUserID = id.(string)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	r.ServeHTTP(w, req)

	if contextUserID != "user-456" {
		t.Errorf("expected user-456, got %s", contextUserID)
	}
}

func TestOptionalJWTAuth_WithToken(t *testing.T) {
	validator := &mockTokenValidator{
		validateFn: func(_ string) (string, error) {
			return "user-123", nil
		},
	}
	r := setupTestRouter(OptionalJWTAuth(validator))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestOptionalJWTAuth_WithoutToken(t *testing.T) {
	validator := &mockTokenValidator{}
	r := setupTestRouter(OptionalJWTAuth(validator))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestOptionalJWTAuth_InvalidToken(t *testing.T) {
	validator := &mockTokenValidator{
		validateFn: func(_ string) (string, error) {
			return "", errors.New("invalid")
		},
	}
	r := setupTestRouter(OptionalJWTAuth(validator))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(w, req)

	// Should still return 200 — optional auth doesn't block
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

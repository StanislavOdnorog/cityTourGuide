package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
	"github.com/saas/city-stories-guide/backend/internal/service"
)

type mockUserService struct {
	deleteFn  func(ctx context.Context, userID string) error
	restoreFn func(ctx context.Context, userID string) error
	getFn     func(ctx context.Context, userID string) (*domain.User, error)
}

func (m *mockUserService) ScheduleDeletion(ctx context.Context, userID string) error {
	return m.deleteFn(ctx, userID)
}

func (m *mockUserService) RestoreAccount(ctx context.Context, userID string) error {
	return m.restoreFn(ctx, userID)
}

func (m *mockUserService) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	return m.getFn(ctx, userID)
}

func setupUserRouter(h *UserHandler, withUser bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if withUser {
		r.Use(func(c *gin.Context) {
			c.Set("user_id", "550e8400-e29b-41d4-a716-446655440000")
			c.Next()
		})
	}
	r.DELETE("/api/v1/users/me", h.DeleteAccount)
	r.POST("/api/v1/users/me/restore", h.RestoreAccount)
	r.GET("/api/v1/users/me", h.GetMe)
	return r
}

func TestUserHandler_SuccessPaths(t *testing.T) {
	email := "test@example.com"
	h := NewUserHandler(&mockUserService{
		deleteFn:  func(context.Context, string) error { return nil },
		restoreFn: func(context.Context, string) error { return nil },
		getFn: func(_ context.Context, userID string) (*domain.User, error) {
			return &domain.User{ID: userID, Email: &email}, nil
		},
	})
	r := setupUserRouter(h, true)

	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{name: "delete account", method: http.MethodDelete, path: "/api/v1/users/me", status: http.StatusOK},
		{name: "restore account", method: http.MethodPost, path: "/api/v1/users/me/restore", status: http.StatusOK},
		{name: "get me", method: http.MethodGet, path: "/api/v1/users/me", status: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Fatalf("expected %d, got %d: %s", tt.status, w.Code, w.Body.String())
			}
		})
	}
}

func TestUserHandler_ErrorContracts(t *testing.T) {
	h := NewUserHandler(&mockUserService{
		deleteFn:  func(context.Context, string) error { return repository.ErrNotFound },
		restoreFn: func(context.Context, string) error { return service.ErrAccountNotScheduled },
		getFn:     func(context.Context, string) (*domain.User, error) { return nil, repository.ErrNotFound },
	})

	t.Run("unauthorized", func(t *testing.T) {
		r := setupUserRouter(h, false)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
		assertErrorResponse(t, w.Body.Bytes(), "unauthorized", "")
	})

	t.Run("delete not found", func(t *testing.T) {
		r := setupUserRouter(h, true)
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
		assertErrorResponse(t, w.Body.Bytes(), "user not found", "")
	})

	t.Run("restore not scheduled", func(t *testing.T) {
		r := setupUserRouter(h, true)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/restore", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
		assertErrorResponse(t, w.Body.Bytes(), "account is not scheduled for deletion", "")
	})

	t.Run("get me not found", func(t *testing.T) {
		r := setupUserRouter(h, true)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
		assertErrorResponse(t, w.Body.Bytes(), "user not found", "")
	})
}

func TestUserHandler_InternalError(t *testing.T) {
	h := NewUserHandler(&mockUserService{
		deleteFn:  func(context.Context, string) error { return errors.New("boom") },
		restoreFn: func(context.Context, string) error { return nil },
		getFn:     func(context.Context, string) (*domain.User, error) { return nil, nil },
	})
	r := setupUserRouter(h, true)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["error"] != "internal server error" {
		t.Fatalf("expected internal server error, got %q", resp["error"])
	}
}

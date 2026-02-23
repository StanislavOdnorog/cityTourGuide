package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/service"
)

// mockAuthService mocks the AuthService interface for handler tests.
type mockAuthService struct {
	registerFn   func(ctx context.Context, email, password, name string) (*domain.User, *service.TokenPair, error)
	loginFn      func(ctx context.Context, email, password string) (*domain.User, *service.TokenPair, error)
	deviceAuthFn func(ctx context.Context, deviceID string, language string) (*domain.User, *service.TokenPair, error)
	refreshFn    func(ctx context.Context, refreshToken string) (*service.TokenPair, error)
}

func (m *mockAuthService) Register(ctx context.Context, email, password, name string) (*domain.User, *service.TokenPair, error) {
	return m.registerFn(ctx, email, password, name)
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (*domain.User, *service.TokenPair, error) {
	return m.loginFn(ctx, email, password)
}

func (m *mockAuthService) DeviceAuth(ctx context.Context, deviceID string, language string) (*domain.User, *service.TokenPair, error) {
	return m.deviceAuthFn(ctx, deviceID, language)
}

func (m *mockAuthService) RefreshTokens(ctx context.Context, refreshToken string) (*service.TokenPair, error) {
	return m.refreshFn(ctx, refreshToken)
}

func setupAuthRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := r.Group("/api/v1/auth")
	auth.POST("/register", h.Register)
	auth.POST("/login", h.Login)
	auth.POST("/device", h.DeviceAuth)
	auth.POST("/refresh", h.Refresh)
	return r
}

func testUser() *domain.User {
	email := "test@example.com"
	name := "Test User"
	return &domain.User{
		ID:           "user-uuid-123",
		Email:        &email,
		Name:         &name,
		AuthProvider: domain.AuthProviderEmail,
		LanguagePref: "en",
		IsAnonymous:  false,
	}
}

func testTokenPair() *service.TokenPair {
	return &service.TokenPair{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresIn:    900,
	}
}

func TestAuthHandler_Register_Success(t *testing.T) {
	mock := &mockAuthService{
		registerFn: func(_ context.Context, email, password, name string) (*domain.User, *service.TokenPair, error) {
			if email != "test@example.com" || name != "Test User" {
				t.Errorf("unexpected params: email=%s, name=%s", email, name)
			}
			return testUser(), testTokenPair(), nil
		},
	}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["tokens"] == nil {
		t.Error("response should contain tokens")
	}
	if resp["data"] == nil {
		t.Error("response should contain data")
	}
}

func TestAuthHandler_Register_MissingFields(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing email", map[string]string{"password": "password123", "name": "Test"}},
		{"missing password", map[string]string{"email": "test@example.com", "name": "Test"}},
		{"missing name", map[string]string{"email": "test@example.com", "password": "password123"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAuthHandler_Register_InvalidEmail(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"email":    "not-an-email",
		"password": "password123",
		"name":     "Test User",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_ShortPassword(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "short",
		"name":     "Test User",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	mock := &mockAuthService{
		registerFn: func(_ context.Context, _, _, _ string) (*domain.User, *service.TokenPair, error) {
			return nil, nil, service.ErrEmailAlreadyExists
		},
	}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestAuthHandler_Register_InternalError(t *testing.T) {
	mock := &mockAuthService{
		registerFn: func(_ context.Context, _, _, _ string) (*domain.User, *service.TokenPair, error) {
			return nil, nil, errors.New("database error")
		},
	}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	mock := &mockAuthService{
		loginFn: func(_ context.Context, email, password string) (*domain.User, *service.TokenPair, error) {
			if email != "test@example.com" || password != "password123" {
				t.Errorf("unexpected params: email=%s", email)
			}
			return testUser(), testTokenPair(), nil
		},
	}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	mock := &mockAuthService{
		loginFn: func(_ context.Context, _, _ string) (*domain.User, *service.TokenPair, error) {
			return nil, nil, service.ErrInvalidCredentials
		},
	}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "wrongpass",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_Login_MissingFields(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"email": "test@example.com",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_DeviceAuth_Success(t *testing.T) {
	mock := &mockAuthService{
		deviceAuthFn: func(_ context.Context, deviceID string, language string) (*domain.User, *service.TokenPair, error) {
			if deviceID != "device-uuid-123" {
				t.Errorf("unexpected device ID: %s", deviceID)
			}
			return &domain.User{
				ID:          deviceID,
				IsAnonymous: true,
			}, testTokenPair(), nil
		},
	}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"device_id": "device-uuid-123",
		"language":  "ru",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/device", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_DeviceAuth_MissingDeviceID(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/device", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	mock := &mockAuthService{
		refreshFn: func(_ context.Context, refreshToken string) (*service.TokenPair, error) {
			return testTokenPair(), nil
		},
	}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"refresh_token": "valid-refresh-token",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["tokens"] == nil {
		t.Error("response should contain tokens")
	}
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	mock := &mockAuthService{
		refreshFn: func(_ context.Context, _ string) (*service.TokenPair, error) {
			return nil, service.ErrInvalidToken
		},
	}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"refresh_token": "invalid-token",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_Refresh_MissingToken(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_EmailNormalization(t *testing.T) {
	var receivedEmail string
	mock := &mockAuthService{
		registerFn: func(_ context.Context, email, _, _ string) (*domain.User, *service.TokenPair, error) {
			receivedEmail = email
			return testUser(), testTokenPair(), nil
		},
	}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	body, _ := json.Marshal(map[string]string{
		"email":    "Test@Example.COM",
		"password": "password123",
		"name":     "Test User",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if receivedEmail != "test@example.com" {
		t.Errorf("expected normalized email, got %s", receivedEmail)
	}
}

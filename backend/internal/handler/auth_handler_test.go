package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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
	oauthLoginFn func(ctx context.Context, provider domain.AuthProvider, claims *service.OAuthClaims) (*domain.User, *service.TokenPair, error)
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

func (m *mockAuthService) OAuthLogin(ctx context.Context, provider domain.AuthProvider, claims *service.OAuthClaims) (*domain.User, *service.TokenPair, error) {
	if m.oauthLoginFn != nil {
		return m.oauthLoginFn(ctx, provider, claims)
	}
	return nil, nil, errors.New("oauthLoginFn not set")
}

func setupAuthRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := r.Group("/api/v1/auth")
	auth.POST("/register", h.Register)
	auth.POST("/login", h.Login)
	auth.POST("/device", h.DeviceAuth)
	auth.POST("/refresh", h.Refresh)
	auth.POST("/google", h.GoogleAuth)
	auth.POST("/apple", h.AppleAuth)
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

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
	})

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
			w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/register", tt.body)

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

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "not-an-email",
		"password": "password123",
		"name":     "Test User",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_ShortPassword(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "test@example.com",
		"password": "short",
		"name":     "Test User",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_EmailTooLong(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	localPart := strings.Repeat("a", 243)
	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    localPart + "@example.com",
		"password": "password123",
		"name":     "Test User",
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
		DetailsByField: map[string]string{
			"email": "must not exceed 254 characters",
		},
	})
}

func TestAuthHandler_Register_NameTooLong(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     strings.Repeat("n", 201),
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
		DetailsByField: map[string]string{
			"name": "must not exceed 200 characters",
		},
	})
}

func TestAuthHandler_Register_PasswordTooLongForBcrypt(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "test@example.com",
		"password": strings.Repeat("a", 73),
		"name":     "Test User",
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
		DetailsByField: map[string]string{
			"password": "must not exceed 72 bytes; bcrypt silently truncates longer passwords",
		},
	})
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	mock := &mockAuthService{
		registerFn: func(_ context.Context, _, _, _ string) (*domain.User, *service.TokenPair, error) {
			return nil, nil, service.ErrEmailAlreadyExists
		},
	}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
	})

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

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
	})

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

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})

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

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "wrongpass",
	})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_Login_MissingFields(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "test@example.com",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Login_EmailTooLong(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	localPart := strings.Repeat("a", 243)
	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    localPart + "@example.com",
		"password": "password123",
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
		DetailsByField: map[string]string{
			"email": "must not exceed 254 characters",
		},
	})
}

func TestAuthHandler_Login_PasswordTooLongForBcrypt(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": strings.Repeat("a", 73),
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
		DetailsByField: map[string]string{
			"password": "must not exceed 72 bytes; bcrypt silently truncates longer passwords",
		},
	})
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

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/device", map[string]string{
		"device_id": "device-uuid-123",
		"language":  "ru",
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_DeviceAuth_MissingDeviceID(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/device", map[string]string{})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_DeviceAuth_DeviceIDTooLong(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/device", map[string]string{
		"device_id": strings.Repeat("d", 501),
		"language":  "en",
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
		DetailsByField: map[string]string{
			"device_id": "must not exceed 500 characters",
		},
	})
}

func TestAuthHandler_DeviceAuth_InvalidLanguage(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/device", map[string]string{
		"device_id": "device-uuid-123",
		"language":  "eng",
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
		DetailsByField: map[string]string{
			"language": "must be a 2-letter ISO 639-1 language code",
		},
	})
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	mock := &mockAuthService{
		refreshFn: func(_ context.Context, refreshToken string) (*service.TokenPair, error) {
			return testTokenPair(), nil
		},
	}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": "valid-refresh-token",
	})

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

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": "invalid-token",
	})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_Refresh_MissingToken(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/refresh", map[string]string{})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// mockGoogleVerifier is a mock for GoogleVerifier.
type mockGoogleVerifier struct {
	result *OAuthResult
	err    error
}

func (m *mockGoogleVerifier) Verify(_ string) (*OAuthResult, error) {
	return m.result, m.err
}

// mockAppleVerifier is a mock for AppleVerifier.
type mockAppleVerifier struct {
	result *OAuthResult
	err    error
}

func (m *mockAppleVerifier) Verify(_ string) (*OAuthResult, error) {
	return m.result, m.err
}

func (m *mockAppleVerifier) VerifyIDToken(_ string) (*OAuthResult, error) {
	return m.result, m.err
}

func TestAuthHandler_GoogleAuth_Success(t *testing.T) {
	email := "user@gmail.com"
	name := "Google User"
	mock := &mockAuthService{
		oauthLoginFn: func(_ context.Context, provider domain.AuthProvider, claims *service.OAuthClaims) (*domain.User, *service.TokenPair, error) {
			if provider != domain.AuthProviderGoogle {
				t.Errorf("expected google provider, got %s", provider)
			}
			if claims.Sub != "google-sub-123" {
				t.Errorf("expected sub google-sub-123, got %s", claims.Sub)
			}
			return &domain.User{
				ID:           "user-uuid-456",
				Email:        &email,
				Name:         &name,
				AuthProvider: domain.AuthProviderGoogle,
			}, testTokenPair(), nil
		},
	}
	h := NewAuthHandler(mock)
	h.SetGoogleVerifier(&mockGoogleVerifier{
		result: &OAuthResult{Sub: "google-sub-123", Email: "user@gmail.com", Name: "Google User"},
	})
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/google", map[string]string{"id_token": "valid-google-id-token"})

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
	if resp["data"] == nil {
		t.Error("response should contain data")
	}
}

func TestAuthHandler_GoogleAuth_InvalidToken(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	h.SetGoogleVerifier(&mockGoogleVerifier{
		err: errors.New("invalid token"),
	})
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/google", map[string]string{"id_token": "invalid-token"})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_GoogleAuth_MissingToken(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	h.SetGoogleVerifier(&mockGoogleVerifier{})
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/google", map[string]string{})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_GoogleAuth_NotConfigured(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	// No google verifier set
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/google", map[string]string{"id_token": "some-token"})

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_AppleAuth_WithCode_Success(t *testing.T) {
	email := "user@icloud.com"
	mock := &mockAuthService{
		oauthLoginFn: func(_ context.Context, provider domain.AuthProvider, claims *service.OAuthClaims) (*domain.User, *service.TokenPair, error) {
			if provider != domain.AuthProviderApple {
				t.Errorf("expected apple provider, got %s", provider)
			}
			return &domain.User{
				ID:           "user-uuid-789",
				Email:        &email,
				AuthProvider: domain.AuthProviderApple,
			}, testTokenPair(), nil
		},
	}
	h := NewAuthHandler(mock)
	h.SetAppleVerifier(&mockAppleVerifier{
		result: &OAuthResult{Sub: "apple-sub-123", Email: "user@icloud.com"},
	})
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/apple", map[string]string{"code": "valid-apple-code"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_AppleAuth_WithIDToken_Success(t *testing.T) {
	email := "user@icloud.com"
	mock := &mockAuthService{
		oauthLoginFn: func(_ context.Context, provider domain.AuthProvider, _ *service.OAuthClaims) (*domain.User, *service.TokenPair, error) {
			return &domain.User{
				ID:           "user-uuid-789",
				Email:        &email,
				AuthProvider: domain.AuthProviderApple,
			}, testTokenPair(), nil
		},
	}
	h := NewAuthHandler(mock)
	h.SetAppleVerifier(&mockAppleVerifier{
		result: &OAuthResult{Sub: "apple-sub-123", Email: "user@icloud.com"},
	})
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/apple", map[string]string{"id_token": "valid-apple-id-token"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_AppleAuth_InvalidToken(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	h.SetAppleVerifier(&mockAppleVerifier{
		err: errors.New("invalid token"),
	})
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/apple", map[string]string{"code": "invalid-code"})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_AppleAuth_MissingBothCodeAndToken(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	h.SetAppleVerifier(&mockAppleVerifier{})
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/apple", map[string]string{})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_AppleAuth_NotConfigured(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	// No apple verifier set
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/apple", map[string]string{"code": "some-code"})

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
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

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "Test@Example.COM",
		"password": "password123",
		"name":     "Test User",
	})

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if receivedEmail != "test@example.com" {
		t.Errorf("expected normalized email, got %s", receivedEmail)
	}
}

func TestAuthHandler_ValidationResponses_WithTraceID(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock)
	r := newRouterWithTrace("trace-auth-123", func(r *gin.Engine) {
		auth := r.Group("/api/v1/auth")
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/device", h.DeviceAuth)
		auth.POST("/refresh", h.Refresh)
		auth.POST("/google", h.GoogleAuth)
		auth.POST("/apple", h.AppleAuth)
	})

	tests := []struct {
		name     string
		path     string
		body     map[string]string
		expected map[string]string
	}{
		{
			name:     "register missing email",
			path:     "/api/v1/auth/register",
			body:     map[string]string{"password": "password123", "name": "Test User"},
			expected: map[string]string{"email": "this field is required"},
		},
		{
			name:     "login missing password",
			path:     "/api/v1/auth/login",
			body:     map[string]string{"email": "test@example.com"},
			expected: map[string]string{"password": "this field is required"},
		},
		{
			name:     "device invalid language",
			path:     "/api/v1/auth/device",
			body:     map[string]string{"device_id": "device-123", "language": "ENG"},
			expected: map[string]string{"language": "must be a 2-letter ISO 639-1 language code"},
		},
		{
			name:     "refresh missing token",
			path:     "/api/v1/auth/refresh",
			body:     map[string]string{},
			expected: map[string]string{"refreshtoken": "this field is required"},
		},
		{
			name:     "google missing token",
			path:     "/api/v1/auth/google",
			body:     map[string]string{},
			expected: map[string]string{"idtoken": "this field is required"},
		},
	}

	h.SetGoogleVerifier(&mockGoogleVerifier{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := executeJSONRequest(t, r, http.MethodPost, tt.path, tt.body)
			assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
				RequestID:      "trace-auth-123",
				DetailsByField: tt.expected,
			})
		})
	}
}

func TestAuthHandler_Register_InvalidJSON(t *testing.T) {
	h := NewAuthHandler(&mockAuthService{})
	r := setupAuthRouter(h)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/register", "not json")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	assertErrorResponseContains(t, w.Body.Bytes(), "invalid character")
}

func TestAuthHandler_Register_ValidationIncludesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewAuthHandler(&mockAuthService{})
	r := gin.New()
	addTraceIDMiddleware(r, "trace-auth-123")
	auth := r.Group("/api/v1/auth")
	auth.POST("/register", h.Register)

	w := executeJSONRequest(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"password": "password123",
		"name":     "Test User",
	})
	assertValidationResponse(t, w.Code, w.Body.Bytes(), validationResponseExpectation{
		RequestID: "trace-auth-123",
		DetailsByField: map[string]string{
			"email": "this field is required",
		},
	})
}

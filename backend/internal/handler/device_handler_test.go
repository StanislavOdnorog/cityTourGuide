package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

type mockPushNotificationService struct {
	registerFn   func(ctx context.Context, userID, token string, platform domain.DevicePlatform) (*domain.DeviceToken, error)
	unregisterFn func(ctx context.Context, token string) error
	listFn       func(ctx context.Context, userID string) ([]domain.DeviceToken, error)
}

func (m *mockPushNotificationService) RegisterDeviceToken(ctx context.Context, userID, token string, platform domain.DevicePlatform) (*domain.DeviceToken, error) {
	return m.registerFn(ctx, userID, token, platform)
}

func (m *mockPushNotificationService) UnregisterDeviceToken(ctx context.Context, token string) error {
	return m.unregisterFn(ctx, token)
}

func (m *mockPushNotificationService) GetUserDeviceTokens(ctx context.Context, userID string) ([]domain.DeviceToken, error) {
	return m.listFn(ctx, userID)
}

func setupDeviceRouter(h *DeviceHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/device-tokens", h.RegisterDeviceToken)
	r.DELETE("/api/v1/device-tokens", h.UnregisterDeviceToken)
	r.GET("/api/v1/device-tokens", h.ListDeviceTokens)
	return r
}

func TestRegisterDeviceToken_Success(t *testing.T) {
	h := NewDeviceHandler(&mockPushNotificationService{
		registerFn: func(_ context.Context, userID, token string, platform domain.DevicePlatform) (*domain.DeviceToken, error) {
			now := time.Now()
			return &domain.DeviceToken{ID: 1, UserID: userID, Token: token, Platform: platform, IsActive: true, CreatedAt: now, UpdatedAt: now}, nil
		},
		unregisterFn: func(context.Context, string) error { return nil },
		listFn:       func(context.Context, string) ([]domain.DeviceToken, error) { return nil, nil },
	})
	r := setupDeviceRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/device-tokens", bytes.NewBufferString(`{"user_id":"550e8400-e29b-41d4-a716-446655440000","token":"abc","platform":"ios"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeviceHandler_InvalidRequests(t *testing.T) {
	h := NewDeviceHandler(&mockPushNotificationService{
		registerFn: func(context.Context, string, string, domain.DevicePlatform) (*domain.DeviceToken, error) {
			return nil, errors.New("unexpected")
		},
		unregisterFn: func(context.Context, string) error { return nil },
		listFn:       func(context.Context, string) ([]domain.DeviceToken, error) { return []domain.DeviceToken{}, nil },
	})
	r := newRouterWithTrace("trace-device-123", func(r *gin.Engine) {
		r.POST("/api/v1/device-tokens", h.RegisterDeviceToken)
		r.DELETE("/api/v1/device-tokens", h.UnregisterDeviceToken)
		r.GET("/api/v1/device-tokens", h.ListDeviceTokens)
	})

	tests := []struct {
		name          string
		method        string
		path          string
		body          string
		expectedCode  int
		expectedError string
		expectedField map[string]string
	}{
		{
			name:         "register missing fields",
			method:       http.MethodPost,
			path:         "/api/v1/device-tokens",
			body:         `{"user_id":"550e8400-e29b-41d4-a716-446655440000"}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"token":    "this field is required",
				"platform": "this field is required",
			},
		},
		{
			name:         "register invalid platform enum",
			method:       http.MethodPost,
			path:         "/api/v1/device-tokens",
			body:         `{"user_id":"550e8400-e29b-41d4-a716-446655440000","token":"abc","platform":"web"}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"platform": "must be one of: ios android",
			},
		},
		{
			name:         "unregister missing token",
			method:       http.MethodDelete,
			path:         "/api/v1/device-tokens",
			body:         `{}`,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"token": "this field is required",
			},
		},
		{
			name:          "list missing user id",
			method:        http.MethodGet,
			path:          "/api/v1/device-tokens",
			expectedCode:  http.StatusBadRequest,
			expectedError: "user_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body == "" {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			} else {
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d: %s", tt.expectedCode, w.Code, w.Body.String())
			}

			if tt.expectedField != nil {
				assertValidationErrorResponse(t, w.Body.Bytes(), tt.expectedField, "trace-device-123")
				return
			}
			assertErrorResponse(t, w.Body.Bytes(), tt.expectedError, "trace-device-123")
		})
	}
}

func TestRegisterDeviceToken_InvalidJSON(t *testing.T) {
	h := NewDeviceHandler(&mockPushNotificationService{
		registerFn: func(context.Context, string, string, domain.DevicePlatform) (*domain.DeviceToken, error) {
			return nil, nil
		},
		unregisterFn: func(context.Context, string) error { return nil },
		listFn:       func(context.Context, string) ([]domain.DeviceToken, error) { return nil, nil },
	})
	r := setupDeviceRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/device-tokens", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	assertErrorResponseContains(t, w.Body.Bytes(), "unexpected EOF")
}

func TestListDeviceTokens_Success(t *testing.T) {
	h := NewDeviceHandler(&mockPushNotificationService{
		registerFn: func(context.Context, string, string, domain.DevicePlatform) (*domain.DeviceToken, error) {
			return nil, nil
		},
		unregisterFn: func(context.Context, string) error { return nil },
		listFn: func(_ context.Context, userID string) ([]domain.DeviceToken, error) {
			now := time.Now()
			return []domain.DeviceToken{{ID: 1, UserID: userID, Token: "abc", Platform: domain.DevicePlatformIOS, IsActive: true, CreatedAt: now, UpdatedAt: now}}, nil
		},
	})
	r := setupDeviceRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-tokens?user_id=user-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []domain.DeviceToken `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 token, got %d", len(resp.Data))
	}
}

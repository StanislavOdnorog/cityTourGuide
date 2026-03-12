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
	"github.com/saas/city-stories-guide/backend/internal/repository"
	"github.com/saas/city-stories-guide/backend/internal/service"
)

type mockPurchaseService struct {
	verifyFn func(ctx context.Context, req *service.VerifyPurchaseRequest) (*domain.Purchase, error)
	statusFn func(ctx context.Context, userID string) (*service.PurchaseStatus, error)
}

func (m *mockPurchaseService) VerifyAndCreate(ctx context.Context, req *service.VerifyPurchaseRequest) (*domain.Purchase, error) {
	return m.verifyFn(ctx, req)
}

func (m *mockPurchaseService) GetStatus(ctx context.Context, userID string) (*service.PurchaseStatus, error) {
	return m.statusFn(ctx, userID)
}

func setupPurchaseRouter(h *PurchaseHandler, withUser bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if withUser {
		r.Use(func(c *gin.Context) {
			c.Set("user_id", "550e8400-e29b-41d4-a716-446655440000")
			c.Next()
		})
	}
	r.POST("/api/v1/purchases/verify", h.VerifyPurchase)
	r.GET("/api/v1/purchases/status", h.GetStatus)
	return r
}

func TestVerifyPurchase_Success(t *testing.T) {
	h := NewPurchaseHandler(&mockPurchaseService{
		verifyFn: func(_ context.Context, req *service.VerifyPurchaseRequest) (*domain.Purchase, error) {
			now := time.Now()
			return &domain.Purchase{ID: 1, UserID: req.UserID, Type: req.Type, Platform: req.Platform, Price: req.Price, CreatedAt: now}, nil
		},
		statusFn: func(context.Context, string) (*service.PurchaseStatus, error) { return nil, nil },
	})
	r := setupPurchaseRouter(h, true)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases/verify", bytes.NewBufferString(`{"platform":"ios","transaction_id":"tx-1","receipt":"receipt","type":"subscription","price":9.99}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPurchaseHandler_InvalidRequests(t *testing.T) {
	h := NewPurchaseHandler(&mockPurchaseService{
		verifyFn: func(context.Context, *service.VerifyPurchaseRequest) (*domain.Purchase, error) {
			return nil, errors.New("unexpected")
		},
		statusFn: func(_ context.Context, userID string) (*service.PurchaseStatus, error) {
			return &service.PurchaseStatus{FreeStoriesLimit: 5, FreeStoriesLeft: 5, CityPacks: []domain.Purchase{}}, nil
		},
	})
	r := newRouterWithTrace("trace-purchase-123", func(r *gin.Engine) {
		r.Use(func(c *gin.Context) {
			c.Set("user_id", "550e8400-e29b-41d4-a716-446655440000")
			c.Next()
		})
		r.POST("/api/v1/purchases/verify", h.VerifyPurchase)
		r.GET("/api/v1/purchases/status", h.GetStatus)
	})

	tests := []struct {
		name          string
		method        string
		path          string
		body          string
		router        *gin.Engine
		expectedCode  int
		expectedError string
		expectedField map[string]string
	}{
		{
			name:         "verify purchase missing required fields",
			method:       http.MethodPost,
			path:         "/api/v1/purchases/verify",
			body:         `{"platform":"ios"}`,
			router:       r,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"transactionid": "this field is required",
				"receipt":       "this field is required",
				"type":          "this field is required",
				"price":         "this field is required",
			},
		},
		{
			name:         "verify purchase invalid price",
			method:       http.MethodPost,
			path:         "/api/v1/purchases/verify",
			body:         `{"platform":"ios","transaction_id":"tx-1","receipt":"receipt","type":"subscription","price":-1}`,
			router:       r,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"price": "must be greater than 0",
			},
		},
		{
			name:         "verify purchase invalid platform",
			method:       http.MethodPost,
			path:         "/api/v1/purchases/verify",
			body:         `{"platform":"web","transaction_id":"tx-1","receipt":"receipt","type":"subscription","price":9.99}`,
			router:       r,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"platform": "must be one of: ios android",
			},
		},
		{
			name:         "verify purchase invalid type",
			method:       http.MethodPost,
			path:         "/api/v1/purchases/verify",
			body:         `{"platform":"ios","transaction_id":"tx-1","receipt":"receipt","type":"invalid_type","price":9.99}`,
			router:       r,
			expectedCode: http.StatusBadRequest,
			expectedField: map[string]string{
				"type": "must be one of: city_pack subscription lifetime",
			},
		},
		{
			name:          "verify purchase unauthorized",
			method:        http.MethodPost,
			path:          "/api/v1/purchases/verify",
			body:          `{"platform":"ios","transaction_id":"tx-1","receipt":"receipt","type":"subscription","price":9.99}`,
			router:        setupPurchaseRouter(h, false),
			expectedCode:  http.StatusUnauthorized,
			expectedError: "user_id not found in context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			tt.router.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Fatalf("expected %d, got %d: %s", tt.expectedCode, w.Code, w.Body.String())
			}

			if tt.expectedField != nil {
				assertValidationErrorResponse(t, w.Body.Bytes(), tt.expectedField, "trace-purchase-123")
				return
			}
			assertErrorResponse(t, w.Body.Bytes(), tt.expectedError, "")
		})
	}
}

func TestVerifyPurchase_InvalidJSON(t *testing.T) {
	h := NewPurchaseHandler(&mockPurchaseService{
		verifyFn: func(context.Context, *service.VerifyPurchaseRequest) (*domain.Purchase, error) { return nil, nil },
		statusFn: func(context.Context, string) (*service.PurchaseStatus, error) { return nil, nil },
	})
	r := setupPurchaseRouter(h, true)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases/verify", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	assertErrorResponseContains(t, w.Body.Bytes(), "unexpected EOF")
}

func TestVerifyPurchase_ForeignKeyViolation(t *testing.T) {
	h := NewPurchaseHandler(&mockPurchaseService{
		verifyFn: func(context.Context, *service.VerifyPurchaseRequest) (*domain.Purchase, error) {
			return nil, repository.ErrInvalidReference
		},
		statusFn: func(context.Context, string) (*service.PurchaseStatus, error) { return nil, nil },
	})
	r := setupPurchaseRouter(h, true)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases/verify", bytes.NewBufferString(`{"platform":"ios","transaction_id":"tx-1","receipt":"receipt","type":"subscription","city_id":999,"price":9.99}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorResponse(t, w.Body.Bytes(), "referenced record does not exist", "")
}

func TestVerifyPurchase_DuplicateTransaction(t *testing.T) {
	h := NewPurchaseHandler(&mockPurchaseService{
		verifyFn: func(context.Context, *service.VerifyPurchaseRequest) (*domain.Purchase, error) {
			return nil, service.ErrDuplicateTransaction
		},
		statusFn: func(context.Context, string) (*service.PurchaseStatus, error) { return nil, nil },
	})
	r := setupPurchaseRouter(h, true)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases/verify", bytes.NewBufferString(`{"platform":"ios","transaction_id":"tx-dup","receipt":"receipt","type":"subscription","price":9.99}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorResponse(t, w.Body.Bytes(), "transaction already processed", "")
}

func TestVerifyPurchase_DBConflict(t *testing.T) {
	// When the service returns ErrConflict (DB constraint), handler should map it to 409
	h := NewPurchaseHandler(&mockPurchaseService{
		verifyFn: func(context.Context, *service.VerifyPurchaseRequest) (*domain.Purchase, error) {
			return nil, repository.ErrConflict
		},
		statusFn: func(context.Context, string) (*service.PurchaseStatus, error) { return nil, nil },
	})
	r := setupPurchaseRouter(h, true)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases/verify", bytes.NewBufferString(`{"platform":"ios","transaction_id":"tx-conflict","receipt":"receipt","type":"subscription","price":9.99}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPurchaseStatus_Success(t *testing.T) {
	h := NewPurchaseHandler(&mockPurchaseService{
		verifyFn: func(context.Context, *service.VerifyPurchaseRequest) (*domain.Purchase, error) { return nil, nil },
		statusFn: func(_ context.Context, userID string) (*service.PurchaseStatus, error) {
			return &service.PurchaseStatus{FreeStoriesLimit: 5, FreeStoriesLeft: 4, FreeStoriesUsed: 1, CityPacks: []domain.Purchase{}}, nil
		},
	})
	r := setupPurchaseRouter(h, true)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchases/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data service.PurchaseStatus `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.FreeStoriesLeft != 4 {
		t.Fatalf("expected FreeStoriesLeft=4, got %d", resp.Data.FreeStoriesLeft)
	}
}

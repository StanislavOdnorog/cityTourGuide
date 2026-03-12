package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/metrics"
	"github.com/saas/city-stories-guide/backend/internal/service"
)

// AuthService defines the auth operations needed by the handler.
type AuthService interface {
	Register(ctx context.Context, email, password, name string) (*domain.User, *service.TokenPair, error)
	Login(ctx context.Context, email, password string) (*domain.User, *service.TokenPair, error)
	DeviceAuth(ctx context.Context, deviceID string, language string) (*domain.User, *service.TokenPair, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*service.TokenPair, error)
	OAuthLogin(ctx context.Context, provider domain.AuthProvider, claims *service.OAuthClaims) (*domain.User, *service.TokenPair, error)
}

// GoogleVerifier verifies Google ID tokens.
type GoogleVerifier interface {
	Verify(idToken string) (*OAuthResult, error)
}

// AppleVerifier verifies Apple Sign-In authorization codes or ID tokens.
type AppleVerifier interface {
	Verify(authorizationCode string) (*OAuthResult, error)
	VerifyIDToken(idToken string) (*OAuthResult, error)
}

// OAuthResult holds the result of an OAuth token verification.
type OAuthResult struct {
	Sub   string
	Email string
	Name  string
}

// AuthHandler handles authentication HTTP endpoints.
type AuthHandler struct {
	auth   AuthService
	google GoogleVerifier
	apple  AppleVerifier
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(auth AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// SetGoogleVerifier sets the Google OAuth verifier.
func (h *AuthHandler) SetGoogleVerifier(v GoogleVerifier) {
	h.google = v
}

// SetAppleVerifier sets the Apple Sign-In verifier.
func (h *AuthHandler) SetAppleVerifier(v AppleVerifier) {
	h.apple = v
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type deviceAuthRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	Language string `json:"language"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrorResponse(c, err)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, tokens, err := h.auth.Register(c.Request.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			errorJSON(c, http.StatusConflict, "email already registered")
			return
		}
		errorJSON(c, http.StatusInternalServerError, "internal server error")
		return
	}

	metrics.AccountsCreatedTotal.Inc()
	c.JSON(http.StatusCreated, gin.H{
		"data":   user,
		"tokens": tokens,
	})
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrorResponse(c, err)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, tokens, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			errorJSON(c, http.StatusUnauthorized, "invalid email or password")
			return
		}
		if errors.Is(err, service.ErrAccountPendingDeletion) {
			errorJSON(c, http.StatusForbidden, "Account scheduled for deletion")
			return
		}
		errorJSON(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   user,
		"tokens": tokens,
	})
}

// DeviceAuth handles POST /api/v1/auth/device.
func (h *AuthHandler) DeviceAuth(c *gin.Context) {
	var req deviceAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrorResponse(c, err)
		return
	}

	user, tokens, err := h.auth.DeviceAuth(c.Request.Context(), req.DeviceID, req.Language)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   user,
		"tokens": tokens,
	})
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrorResponse(c, err)
		return
	}

	tokens, err := h.auth.RefreshTokens(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			errorJSON(c, http.StatusUnauthorized, "invalid or expired refresh token")
			return
		}
		errorJSON(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokens,
	})
}

type googleAuthRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

// GoogleAuth handles POST /api/v1/auth/google.
func (h *AuthHandler) GoogleAuth(c *gin.Context) {
	if h.google == nil {
		errorJSON(c, http.StatusServiceUnavailable, "google sign-in not configured")
		return
	}

	var req googleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrorResponse(c, err)
		return
	}

	result, err := h.google.Verify(req.IDToken)
	if err != nil {
		errorJSON(c, http.StatusUnauthorized, "invalid google token")
		return
	}

	user, tokens, err := h.auth.OAuthLogin(c.Request.Context(), domain.AuthProviderGoogle, &service.OAuthClaims{
		Sub:   result.Sub,
		Email: result.Email,
		Name:  result.Name,
	})
	if err != nil {
		if errors.Is(err, service.ErrAccountPendingDeletion) {
			errorJSON(c, http.StatusForbidden, "Account scheduled for deletion")
			return
		}
		errorJSON(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   user,
		"tokens": tokens,
	})
}

type appleAuthRequest struct {
	Code    string `json:"code"`
	IDToken string `json:"id_token"`
}

// AppleAuth handles POST /api/v1/auth/apple.
func (h *AuthHandler) AppleAuth(c *gin.Context) {
	if h.apple == nil {
		errorJSON(c, http.StatusServiceUnavailable, "apple sign-in not configured")
		return
	}

	var req appleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrorResponse(c, err)
		return
	}

	if req.Code == "" && req.IDToken == "" {
		errorJSON(c, http.StatusBadRequest, "either code or id_token is required")
		return
	}

	var result *OAuthResult
	var err error

	if req.IDToken != "" {
		result, err = h.apple.VerifyIDToken(req.IDToken)
	} else {
		result, err = h.apple.Verify(req.Code)
	}

	if err != nil {
		errorJSON(c, http.StatusUnauthorized, "invalid apple token")
		return
	}

	user, tokens, err := h.auth.OAuthLogin(c.Request.Context(), domain.AuthProviderApple, &service.OAuthClaims{
		Sub:   result.Sub,
		Email: result.Email,
		Name:  result.Name,
	})
	if err != nil {
		if errors.Is(err, service.ErrAccountPendingDeletion) {
			errorJSON(c, http.StatusForbidden, "Account scheduled for deletion")
			return
		}
		errorJSON(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   user,
		"tokens": tokens,
	})
}

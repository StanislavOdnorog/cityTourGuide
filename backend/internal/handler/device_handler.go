package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// PushNotificationService defines the interface for push notification operations.
type PushNotificationService interface {
	RegisterDeviceToken(ctx context.Context, userID, token string, platform domain.DevicePlatform) (*domain.DeviceToken, error)
	UnregisterDeviceToken(ctx context.Context, token string) error
	GetUserDeviceTokens(ctx context.Context, userID string) ([]domain.DeviceToken, error)
}

// DeviceHandler handles device token registration endpoints.
type DeviceHandler struct {
	pushService PushNotificationService
}

// NewDeviceHandler creates a new DeviceHandler.
func NewDeviceHandler(pushService PushNotificationService) *DeviceHandler {
	return &DeviceHandler{pushService: pushService}
}

type registerDeviceTokenRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Token    string `json:"token" binding:"required"`
	Platform string `json:"platform" binding:"required,oneof=ios android"`
}

// RegisterDeviceToken handles POST /api/v1/device-tokens.
func (h *DeviceHandler) RegisterDeviceToken(c *gin.Context) {
	var req registerDeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrorResponse(c, err)
		return
	}

	dt, err := h.pushService.RegisterDeviceToken(
		c.Request.Context(),
		req.UserID,
		req.Token,
		domain.DevicePlatform(req.Platform),
	)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to register device token")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": dt})
}

type unregisterDeviceTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// UnregisterDeviceToken handles DELETE /api/v1/device-tokens.
func (h *DeviceHandler) UnregisterDeviceToken(c *gin.Context) {
	var req unregisterDeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrorResponse(c, err)
		return
	}

	if err := h.pushService.UnregisterDeviceToken(c.Request.Context(), req.Token); err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to unregister device token")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device token unregistered"})
}

// ListDeviceTokens handles GET /api/v1/device-tokens?user_id=xxx.
func (h *DeviceHandler) ListDeviceTokens(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		errorJSON(c, http.StatusBadRequest, "user_id is required")
		return
	}

	tokens, err := h.pushService.GetUserDeviceTokens(c.Request.Context(), userID)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to get device tokens")
		return
	}

	if tokens == nil {
		tokens = []domain.DeviceToken{}
	}

	c.JSON(http.StatusOK, gin.H{"data": tokens})
}

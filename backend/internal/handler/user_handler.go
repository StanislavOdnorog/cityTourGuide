package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
	"github.com/saas/city-stories-guide/backend/internal/service"
)

// UserService defines the user operations needed by the handler.
type UserService interface {
	ScheduleDeletion(ctx context.Context, userID string) error
	RestoreAccount(ctx context.Context, userID string) error
	GetByID(ctx context.Context, userID string) (*domain.User, error)
}

// UserHandler handles user account HTTP endpoints.
type UserHandler struct {
	users UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(users UserService) *UserHandler {
	return &UserHandler{users: users}
}

// DeleteAccount handles DELETE /api/v1/users/me.
func (h *UserHandler) DeleteAccount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.users.ScheduleDeletion(c.Request.Context(), userID.(string)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Account scheduled for deletion",
		"grace_period": "30 days",
	})
}

// RestoreAccount handles POST /api/v1/users/me/restore.
func (h *UserHandler) RestoreAccount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.users.RestoreAccount(c.Request.Context(), userID.(string)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if errors.Is(err, service.ErrAccountNotScheduled) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "account is not scheduled for deletion"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account restored successfully"})
}

// GetMe handles GET /api/v1/users/me.
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID.(string))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user})
}

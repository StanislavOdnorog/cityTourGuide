package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// ListeningRepository defines the interface for listening database operations.
type ListeningRepository interface {
	CreateOrUpdate(ctx context.Context, userID string, storyID int, completed bool, lat, lng *float64) (*domain.UserListening, error)
}

// ListeningHandler handles listening tracking endpoints.
type ListeningHandler struct {
	repo ListeningRepository
}

// NewListeningHandler creates a new ListeningHandler.
func NewListeningHandler(repo ListeningRepository) *ListeningHandler {
	return &ListeningHandler{repo: repo}
}

type trackListeningRequest struct {
	UserID    string   `json:"user_id" binding:"required"`
	StoryID   int      `json:"story_id" binding:"required"`
	Completed bool     `json:"completed"`
	Lat       *float64 `json:"lat"`
	Lng       *float64 `json:"lng"`
}

// TrackListening handles POST /api/v1/listenings.
func (h *ListeningHandler) TrackListening(c *gin.Context) {
	var req trackListeningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.StoryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "story_id must be a positive integer"})
		return
	}

	// Validate that lat/lng come in pairs
	if (req.Lat == nil) != (req.Lng == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng must both be provided or both omitted"})
		return
	}

	// Validate coordinate ranges if provided
	if req.Lat != nil && req.Lng != nil {
		if *req.Lat < -90 || *req.Lat > 90 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lat must be between -90 and 90"})
			return
		}
		if *req.Lng < -180 || *req.Lng > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lng must be between -180 and 180"})
			return
		}
	}

	listening, err := h.repo.CreateOrUpdate(
		c.Request.Context(),
		req.UserID, req.StoryID, req.Completed,
		req.Lat, req.Lng,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to track listening"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": listening})
}

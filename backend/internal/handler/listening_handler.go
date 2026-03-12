package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/metrics"
)

// ListeningRepository defines the interface for listening database operations.
type ListeningRepository interface {
	CreateOrUpdate(ctx context.Context, userID string, storyID int, completed bool, lat, lng *float64) (*domain.UserListening, error)
	ListByUserID(ctx context.Context, userID string, page domain.PageRequest) (*domain.PageResponse[domain.UserListening], error)
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
		validationErrorResponse(c, err)
		return
	}

	if req.StoryID <= 0 {
		errorJSON(c, http.StatusBadRequest, "story_id must be a positive integer")
		return
	}

	// Validate that lat/lng come in pairs
	if (req.Lat == nil) != (req.Lng == nil) {
		errorJSON(c, http.StatusBadRequest, "lat and lng must both be provided or both omitted")
		return
	}

	// Validate coordinate ranges if provided
	if req.Lat != nil && req.Lng != nil {
		if *req.Lat < -90 || *req.Lat > 90 {
			errorJSON(c, http.StatusBadRequest, "lat must be between -90 and 90")
			return
		}
		if *req.Lng < -180 || *req.Lng > 180 {
			errorJSON(c, http.StatusBadRequest, "lng must be between -180 and 180")
			return
		}
	}

	listening, err := h.repo.CreateOrUpdate(
		c.Request.Context(),
		req.UserID, req.StoryID, req.Completed,
		req.Lat, req.Lng,
	)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to track listening")
		return
	}

	metrics.StoriesPlayedTotal.Inc()
	c.JSON(http.StatusCreated, gin.H{"data": listening})
}

// ListListenings handles GET /api/v1/listenings?user_id=.
func (h *ListeningHandler) ListListenings(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		errorJSON(c, http.StatusBadRequest, "user_id is required")
		return
	}

	pageReq, ok := parseCursorPagination(c)
	if !ok {
		return
	}

	result, err := h.repo.ListByUserID(c.Request.Context(), userID, pageReq)
	if err != nil {
		if isCursorError(err) {
			errorJSON(c, http.StatusBadRequest, err.Error())
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to fetch listenings")
		return
	}

	if result.Items == nil {
		result.Items = []domain.UserListening{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       result.Items,
		"next_cursor": result.NextCursor,
		"has_more":    result.HasMore,
	})
}

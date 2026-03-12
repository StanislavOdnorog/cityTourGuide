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
	UserID    string   `json:"user_id" binding:"required,uuid"`
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

	// Validate that lat/lng come in pairs and ranges
	if !validateCoordPair(c, req.Lat, req.Lng) {
		return
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
	userID, ok := parseUserIDQuery(c)
	if !ok {
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

	writeCursorPage(c, result)
}

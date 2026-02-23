package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// StoryRepository defines the interface for story database operations.
type StoryRepository interface {
	Create(ctx context.Context, story *domain.Story) (*domain.Story, error)
	GetByID(ctx context.Context, id int) (*domain.Story, error)
	GetByPOIID(ctx context.Context, poiID int, language string, status *domain.StoryStatus) ([]domain.Story, error)
	Update(ctx context.Context, story *domain.Story) (*domain.Story, error)
	Delete(ctx context.Context, id int) error
}

// StoryHandler handles CRUD operations for stories.
type StoryHandler struct {
	repo StoryRepository
}

// NewStoryHandler creates a new StoryHandler.
func NewStoryHandler(repo StoryRepository) *StoryHandler {
	return &StoryHandler{repo: repo}
}

// createStoryRequest represents the request body for creating a story.
type createStoryRequest struct {
	POIID       int                   `json:"poi_id" binding:"required"`
	Language    string                `json:"language" binding:"required"`
	Text        string                `json:"text" binding:"required"`
	AudioURL    *string               `json:"audio_url"`
	DurationSec *int16                `json:"duration_sec"`
	LayerType   domain.StoryLayerType `json:"layer_type" binding:"required"`
	OrderIndex  *int16                `json:"order_index"`
	IsInflation *bool                 `json:"is_inflation"`
	Confidence  *int16                `json:"confidence"`
	Sources     *json.RawMessage      `json:"sources"`
	Status      *domain.StoryStatus   `json:"status"`
}

// updateStoryRequest represents the request body for updating a story.
type updateStoryRequest struct {
	POIID       int                   `json:"poi_id" binding:"required"`
	Language    string                `json:"language" binding:"required"`
	Text        string                `json:"text" binding:"required"`
	AudioURL    *string               `json:"audio_url"`
	DurationSec *int16                `json:"duration_sec"`
	LayerType   domain.StoryLayerType `json:"layer_type" binding:"required"`
	OrderIndex  *int16                `json:"order_index"`
	IsInflation *bool                 `json:"is_inflation"`
	Confidence  *int16                `json:"confidence"`
	Sources     *json.RawMessage      `json:"sources"`
	Status      *domain.StoryStatus   `json:"status"`
}

// ListStories handles GET /api/v1/stories?poi_id=&language=&status=.
func (h *StoryHandler) ListStories(c *gin.Context) {
	poiIDStr := c.Query("poi_id")
	if poiIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "poi_id is required"})
		return
	}

	poiID, ok := parseQueryInt(c, "poi_id", poiIDStr)
	if !ok {
		return
	}

	language := c.DefaultQuery("language", "en")

	var statusFilter *domain.StoryStatus
	if s := c.Query("status"); s != "" {
		st := domain.StoryStatus(s)
		statusFilter = &st
	}

	stories, err := h.repo.GetByPOIID(c.Request.Context(), poiID, language, statusFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stories"})
		return
	}

	if stories == nil {
		stories = []domain.Story{}
	}

	page, perPage := parsePagination(c)
	total := len(stories)

	start := (page - 1) * perPage
	if start > total {
		start = total
	}
	end := start + perPage
	if end > total {
		end = total
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stories[start:end],
		"meta": gin.H{
			"total":    total,
			"page":     page,
			"per_page": perPage,
		},
	})
}

// GetStory handles GET /api/v1/stories/:id.
func (h *StoryHandler) GetStory(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	story, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "story not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch story"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": story})
}

// CreateStory handles POST /api/v1/admin/stories.
func (h *StoryHandler) CreateStory(c *gin.Context) {
	var req createStoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderIndex := int16(0)
	if req.OrderIndex != nil {
		orderIndex = *req.OrderIndex
	}

	isInflation := false
	if req.IsInflation != nil {
		isInflation = *req.IsInflation
	}

	confidence := int16(80)
	if req.Confidence != nil {
		confidence = *req.Confidence
	}

	status := domain.StoryStatusActive
	if req.Status != nil {
		status = *req.Status
	}

	var sources json.RawMessage
	if req.Sources != nil {
		sources = *req.Sources
	}

	story := &domain.Story{
		POIID:       req.POIID,
		Language:    req.Language,
		Text:        req.Text,
		AudioURL:    req.AudioURL,
		DurationSec: req.DurationSec,
		LayerType:   req.LayerType,
		OrderIndex:  orderIndex,
		IsInflation: isInflation,
		Confidence:  confidence,
		Sources:     sources,
		Status:      status,
	}

	created, err := h.repo.Create(c.Request.Context(), story)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create story"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

// UpdateStory handles PUT /api/v1/admin/stories/:id.
func (h *StoryHandler) UpdateStory(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	var req updateStoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderIndex := int16(0)
	if req.OrderIndex != nil {
		orderIndex = *req.OrderIndex
	}

	isInflation := false
	if req.IsInflation != nil {
		isInflation = *req.IsInflation
	}

	confidence := int16(80)
	if req.Confidence != nil {
		confidence = *req.Confidence
	}

	status := domain.StoryStatusActive
	if req.Status != nil {
		status = *req.Status
	}

	var sources json.RawMessage
	if req.Sources != nil {
		sources = *req.Sources
	}

	story := &domain.Story{
		ID:          id,
		POIID:       req.POIID,
		Language:    req.Language,
		Text:        req.Text,
		AudioURL:    req.AudioURL,
		DurationSec: req.DurationSec,
		LayerType:   req.LayerType,
		OrderIndex:  orderIndex,
		IsInflation: isInflation,
		Confidence:  confidence,
		Sources:     sources,
		Status:      status,
	}

	updated, err := h.repo.Update(c.Request.Context(), story)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "story not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update story"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

// DeleteStory handles DELETE /api/v1/admin/stories/:id.
func (h *StoryHandler) DeleteStory(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	err := h.repo.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "story not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete story"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "story deleted"})
}

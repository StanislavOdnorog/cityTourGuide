package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// POIRepository defines the interface for POI database operations.
type POIRepository interface {
	Create(ctx context.Context, poi *domain.POI) (*domain.POI, error)
	GetByID(ctx context.Context, id int) (*domain.POI, error)
	GetByCityID(ctx context.Context, cityID int, status *domain.POIStatus, poiType *domain.POIType) ([]domain.POI, error)
	ListByCityID(ctx context.Context, cityID int, status *domain.POIStatus, poiType *domain.POIType, page domain.PageRequest) (*domain.PageResponse[domain.POI], error)
	Update(ctx context.Context, poi *domain.POI) (*domain.POI, error)
	Delete(ctx context.Context, id int) error
}

// POIHandler handles CRUD operations for Points of Interest.
type POIHandler struct {
	repo POIRepository
}

// NewPOIHandler creates a new POIHandler.
func NewPOIHandler(repo POIRepository) *POIHandler {
	return &POIHandler{repo: repo}
}

// createPOIRequest represents the request body for creating a POI.
type createPOIRequest struct {
	CityID        int               `json:"city_id" binding:"required"`
	Name          string            `json:"name" binding:"required"`
	NameRu        *string           `json:"name_ru"`
	Lat           float64           `json:"lat" binding:"required"`
	Lng           float64           `json:"lng" binding:"required"`
	Type          domain.POIType    `json:"type" binding:"required"`
	Tags          *json.RawMessage  `json:"tags"`
	Address       *string           `json:"address"`
	InterestScore *int16            `json:"interest_score"`
	Status        *domain.POIStatus `json:"status"`
}

// updatePOIRequest represents the request body for updating a POI.
type updatePOIRequest struct {
	CityID        int               `json:"city_id" binding:"required"`
	Name          string            `json:"name" binding:"required"`
	NameRu        *string           `json:"name_ru"`
	Lat           float64           `json:"lat" binding:"required"`
	Lng           float64           `json:"lng" binding:"required"`
	Type          domain.POIType    `json:"type" binding:"required"`
	Tags          *json.RawMessage  `json:"tags"`
	Address       *string           `json:"address"`
	InterestScore *int16            `json:"interest_score"`
	Status        *domain.POIStatus `json:"status"`
}

// ListPOIs handles GET /api/v1/pois?city_id=&status=&type=.
func (h *POIHandler) ListPOIs(c *gin.Context) {
	cityIDStr := c.Query("city_id")
	if cityIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "city_id is required"})
		return
	}

	cityID, ok := parseQueryInt(c, "city_id", cityIDStr)
	if !ok {
		return
	}

	var statusFilter *domain.POIStatus
	if s := c.Query("status"); s != "" {
		st := domain.POIStatus(s)
		statusFilter = &st
	}

	var typeFilter *domain.POIType
	if t := c.Query("type"); t != "" {
		pt := domain.POIType(t)
		typeFilter = &pt
	}

	pageReq, ok := parseCursorPagination(c)
	if !ok {
		return
	}

	result, err := h.repo.ListByCityID(c.Request.Context(), cityID, statusFilter, typeFilter, pageReq)
	if err != nil {
		if isCursorError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch POIs"})
		return
	}

	if result.Items == nil {
		result.Items = []domain.POI{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       result.Items,
		"next_cursor": result.NextCursor,
		"has_more":    result.HasMore,
	})
}

// GetPOI handles GET /api/v1/pois/:id.
func (h *POIHandler) GetPOI(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	poi, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "POI not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch POI"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": poi})
}

// CreatePOI handles POST /api/v1/admin/pois.
func (h *POIHandler) CreatePOI(c *gin.Context) {
	var req createPOIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	interestScore := int16(50)
	if req.InterestScore != nil {
		interestScore = *req.InterestScore
	}

	status := domain.POIStatusActive
	if req.Status != nil {
		status = *req.Status
	}

	var tags json.RawMessage
	if req.Tags != nil {
		tags = *req.Tags
	}

	poi := &domain.POI{
		CityID:        req.CityID,
		Name:          req.Name,
		NameRu:        req.NameRu,
		Lat:           req.Lat,
		Lng:           req.Lng,
		Type:          req.Type,
		Tags:          tags,
		Address:       req.Address,
		InterestScore: interestScore,
		Status:        status,
	}

	created, err := h.repo.Create(c.Request.Context(), poi)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create POI"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

// UpdatePOI handles PUT /api/v1/admin/pois/:id.
func (h *POIHandler) UpdatePOI(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	var req updatePOIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	interestScore := int16(50)
	if req.InterestScore != nil {
		interestScore = *req.InterestScore
	}

	status := domain.POIStatusActive
	if req.Status != nil {
		status = *req.Status
	}

	var tags json.RawMessage
	if req.Tags != nil {
		tags = *req.Tags
	}

	poi := &domain.POI{
		ID:            id,
		CityID:        req.CityID,
		Name:          req.Name,
		NameRu:        req.NameRu,
		Lat:           req.Lat,
		Lng:           req.Lng,
		Type:          req.Type,
		Tags:          tags,
		Address:       req.Address,
		InterestScore: interestScore,
		Status:        status,
	}

	updated, err := h.repo.Update(c.Request.Context(), poi)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "POI not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update POI"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

// DeletePOI handles DELETE /api/v1/admin/pois/:id.
func (h *POIHandler) DeletePOI(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	err := h.repo.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "POI not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete POI"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "POI deleted"})
}

// parseQueryInt parses a query parameter as a positive integer.
func parseQueryInt(c *gin.Context, name, value string) (int, bool) {
	v, err := strconv.Atoi(value)
	if err != nil || v <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": name + " must be a positive integer"})
		return 0, false
	}
	return v, true
}

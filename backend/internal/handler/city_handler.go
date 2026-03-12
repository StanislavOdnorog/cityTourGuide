package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/metrics"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// CityRepository defines the interface for city database operations.
type CityRepository interface {
	Create(ctx context.Context, city *domain.City) (*domain.City, error)
	GetByID(ctx context.Context, id int) (*domain.City, error)
	GetAll(ctx context.Context) ([]domain.City, error)
	List(ctx context.Context, page domain.PageRequest) (*domain.PageResponse[domain.City], error)
	Update(ctx context.Context, city *domain.City) (*domain.City, error)
	Delete(ctx context.Context, id int) error
}

// DownloadManifestRepository defines the interface for fetching download manifest data.
type DownloadManifestRepository interface {
	GetDownloadManifest(ctx context.Context, cityID int, language string) ([]domain.DownloadManifestItem, error)
}

// CityHandler handles CRUD operations for cities.
type CityHandler struct {
	repo         CityRepository
	manifestRepo DownloadManifestRepository
}

// NewCityHandler creates a new CityHandler.
func NewCityHandler(repo CityRepository, manifestRepo DownloadManifestRepository) *CityHandler {
	return &CityHandler{repo: repo, manifestRepo: manifestRepo}
}

// createCityRequest represents the request body for creating a city.
type createCityRequest struct {
	Name           string  `json:"name" binding:"required"`
	NameRu         *string `json:"name_ru"`
	Country        string  `json:"country" binding:"required"`
	CenterLat      float64 `json:"center_lat" binding:"required"`
	CenterLng      float64 `json:"center_lng" binding:"required"`
	RadiusKm       float64 `json:"radius_km" binding:"required"`
	IsActive       *bool   `json:"is_active"`
	DownloadSizeMB float64 `json:"download_size_mb"`
}

// updateCityRequest represents the request body for updating a city.
type updateCityRequest struct {
	Name           string  `json:"name" binding:"required"`
	NameRu         *string `json:"name_ru"`
	Country        string  `json:"country" binding:"required"`
	CenterLat      float64 `json:"center_lat" binding:"required"`
	CenterLng      float64 `json:"center_lng" binding:"required"`
	RadiusKm       float64 `json:"radius_km" binding:"required"`
	IsActive       *bool   `json:"is_active"`
	DownloadSizeMB float64 `json:"download_size_mb"`
}

// ListCities handles GET /api/v1/cities.
func (h *CityHandler) ListCities(c *gin.Context) {
	pageReq, ok := parseCursorPagination(c)
	if !ok {
		return
	}

	result, err := h.repo.List(c.Request.Context(), pageReq)
	if err != nil {
		if isCursorError(err) {
			errorJSON(c, http.StatusBadRequest, err.Error())
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to fetch cities")
		return
	}

	if result.Items == nil {
		result.Items = []domain.City{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       result.Items,
		"next_cursor": result.NextCursor,
		"has_more":    result.HasMore,
	})
}

// GetCity handles GET /api/v1/cities/:id.
func (h *CityHandler) GetCity(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	city, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			errorJSON(c, http.StatusNotFound, "city not found")
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to fetch city")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": city})
}

// CreateCity handles POST /api/v1/admin/cities.
func (h *CityHandler) CreateCity(c *gin.Context) {
	var req createCityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrorResponse(c, err)
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	city := &domain.City{
		Name:           req.Name,
		NameRu:         req.NameRu,
		Country:        req.Country,
		CenterLat:      req.CenterLat,
		CenterLng:      req.CenterLng,
		RadiusKm:       req.RadiusKm,
		IsActive:       isActive,
		DownloadSizeMB: req.DownloadSizeMB,
	}

	created, err := h.repo.Create(c.Request.Context(), city)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to create city")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

// UpdateCity handles PUT /api/v1/admin/cities/:id.
func (h *CityHandler) UpdateCity(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	var req updateCityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrorResponse(c, err)
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	city := &domain.City{
		ID:             id,
		Name:           req.Name,
		NameRu:         req.NameRu,
		Country:        req.Country,
		CenterLat:      req.CenterLat,
		CenterLng:      req.CenterLng,
		RadiusKm:       req.RadiusKm,
		IsActive:       isActive,
		DownloadSizeMB: req.DownloadSizeMB,
	}

	updated, err := h.repo.Update(c.Request.Context(), city)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			errorJSON(c, http.StatusNotFound, "city not found")
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to update city")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

// DeleteCity handles DELETE /api/v1/admin/cities/:id.
func (h *CityHandler) DeleteCity(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	err := h.repo.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			errorJSON(c, http.StatusNotFound, "city not found")
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to delete city")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "city deleted"})
}

// GetDownloadManifest handles GET /api/v1/cities/:id/download-manifest.
func (h *CityHandler) GetDownloadManifest(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	// Verify city exists
	city, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			errorJSON(c, http.StatusNotFound, "city not found")
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to fetch city")
		return
	}

	language := c.DefaultQuery("language", "en")

	items, err := h.manifestRepo.GetDownloadManifest(c.Request.Context(), id, language)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to fetch download manifest")
		return
	}

	if items == nil {
		items = []domain.DownloadManifestItem{}
	}

	var totalSizeBytes int64
	for i := range items {
		totalSizeBytes += items[i].FileSizeBytes
	}

	metrics.CitiesDownloadedTotal.Inc()
	c.JSON(http.StatusOK, gin.H{
		"data":             items,
		"total_size_bytes": totalSizeBytes,
		"total_stories":    len(items),
		"city_name":        city.Name,
	})
}

// parseIDParam extracts and validates the :id URL parameter.
func parseIDParam(c *gin.Context) (int, bool) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		errorJSON(c, http.StatusBadRequest, "invalid id parameter")
		return 0, false
	}
	return id, true
}

// parseCursorPagination extracts cursor and limit query parameters for cursor-based pagination.
// Returns false if validation failed (error already written to response).
func parseCursorPagination(c *gin.Context) (domain.PageRequest, bool) {
	pr := domain.PageRequest{
		Cursor: c.Query("cursor"),
		Limit:  domain.DefaultPageLimit,
	}

	if l := c.Query("limit"); l != "" {
		v, err := strconv.Atoi(l)
		if err != nil || v <= 0 {
			errorJSON(c, http.StatusBadRequest, "limit must be a positive integer")
			return pr, false
		}
		if v > domain.MaxPageLimit {
			errorJSON(c, http.StatusBadRequest, "limit must not exceed 100")
			return pr, false
		}
		pr.Limit = v
	}

	return pr, true
}

// isCursorError checks if an error is related to an invalid cursor.
func isCursorError(err error) bool {
	return strings.Contains(err.Error(), "invalid cursor")
}

// parsePagination extracts page and per_page query parameters with defaults (legacy).
func parsePagination(c *gin.Context) (page, perPage int) {
	page = 1
	perPage = 20

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if pp := c.Query("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 && v <= 100 {
			perPage = v
		}
	}
	return page, perPage
}

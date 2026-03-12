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
	GetByID(ctx context.Context, id int, includeDeleted bool) (*domain.City, error)
	GetActiveByID(ctx context.Context, id int) (*domain.City, error)
	GetAll(ctx context.Context) ([]domain.City, error)
	List(ctx context.Context, page domain.PageRequest, includeDeleted bool, sort repository.ListSort) (*domain.PageResponse[domain.City], error)
	ListActive(ctx context.Context, page domain.PageRequest) (*domain.PageResponse[domain.City], error)
	Update(ctx context.Context, city *domain.City) (*domain.City, error)
	Delete(ctx context.Context, id int) error
	Restore(ctx context.Context, id int) (*domain.City, error)
}

// DownloadManifestRepository defines the interface for fetching download manifest data.
type DownloadManifestRepository interface {
	GetDownloadManifest(ctx context.Context, cityID int, language string) ([]domain.DownloadManifestItem, error)
}

// CityCleanupService schedules orphaned audio cleanup on city deletion.
type CityCleanupService interface {
	DeleteCity(ctx context.Context, id int) error
}

// CityHandler handles CRUD operations for cities.
type CityHandler struct {
	repo         CityRepository
	manifestRepo DownloadManifestRepository
	audit        AuditLogger
	cleanup      CityCleanupService
}

// NewCityHandler creates a new CityHandler.
func NewCityHandler(repo CityRepository, manifestRepo DownloadManifestRepository, audit AuditLogger, cleanup ...CityCleanupService) *CityHandler {
	h := &CityHandler{repo: repo, manifestRepo: manifestRepo, audit: audit}
	if len(cleanup) > 0 {
		h.cleanup = cleanup[0]
	}
	return h
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

func (r *createCityRequest) validate() error {
	if err := domain.ValidateStringLength(r.Name, "name", 1, 200); err != nil {
		return err
	}
	if r.NameRu != nil {
		if err := domain.ValidateStringLength(*r.NameRu, "name_ru", 0, 200); err != nil {
			return err
		}
	}
	if err := domain.ValidateStringLength(r.Country, "country", 1, 100); err != nil {
		return err
	}
	if err := domain.ValidateCoordinate(r.CenterLat, r.CenterLng); err != nil {
		return err
	}
	if r.RadiusKm < 0.1 {
		return &domain.ValidationError{Field: "radius_km", Message: "must be at least 0.1"}
	}
	if r.RadiusKm > 1000 {
		return &domain.ValidationError{Field: "radius_km", Message: "must not exceed 1000"}
	}
	if r.DownloadSizeMB < 0 {
		return &domain.ValidationError{Field: "download_size_mb", Message: "must be non-negative"}
	}
	return nil
}

// supportedLanguages is the set of valid language codes for download manifests.
var supportedLanguages = map[string]bool{
	"en": true,
	"ru": true,
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

var adminCitySortColumns = map[string]repository.SortColumn{
	"id":         {Key: "id", Column: "id", Type: repository.SortValueInt},
	"name":       {Key: "name", Column: "name", Type: repository.SortValueString},
	"country":    {Key: "country", Column: "country", Type: repository.SortValueString},
	"is_active":  {Key: "is_active", Column: "is_active", Type: repository.SortValueBool},
	"created_at": {Key: "created_at", Column: "created_at", Type: repository.SortValueTime},
	"updated_at": {Key: "updated_at", Column: "updated_at", Type: repository.SortValueTime},
}

func (r *updateCityRequest) validate() error {
	if err := domain.ValidateStringLength(r.Name, "name", 1, 200); err != nil {
		return err
	}
	if r.NameRu != nil {
		if err := domain.ValidateStringLength(*r.NameRu, "name_ru", 0, 200); err != nil {
			return err
		}
	}
	if err := domain.ValidateStringLength(r.Country, "country", 1, 100); err != nil {
		return err
	}
	if err := domain.ValidateCoordinate(r.CenterLat, r.CenterLng); err != nil {
		return err
	}
	if r.RadiusKm < 0.1 {
		return &domain.ValidationError{Field: "radius_km", Message: "must be at least 0.1"}
	}
	if r.RadiusKm > 1000 {
		return &domain.ValidationError{Field: "radius_km", Message: "must not exceed 1000"}
	}
	if r.DownloadSizeMB < 0 {
		return &domain.ValidationError{Field: "download_size_mb", Message: "must be non-negative"}
	}
	return nil
}

// ListCities handles GET /api/v1/cities (public, active cities only).
func (h *CityHandler) ListCities(c *gin.Context) {
	pageReq, ok := parseCursorPagination(c)
	if !ok {
		return
	}

	result, err := h.repo.ListActive(c.Request.Context(), pageReq)
	if err != nil {
		if isCursorError(err) {
			errorJSON(c, http.StatusBadRequest, err.Error())
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to fetch cities")
		return
	}

	writeCursorPage(c, result)
}

// ListAdminCities handles GET /api/v1/admin/cities.
func (h *CityHandler) ListAdminCities(c *gin.Context) {
	pageReq, ok := parseCursorPagination(c)
	if !ok {
		return
	}

	includeDeleted, ok := parseOptionalBoolQuery(c, "include_deleted")
	if !ok {
		return
	}

	sortReq, ok := parseListSort(c, adminCitySortColumns, "id", repository.SortDirAsc)
	if !ok {
		return
	}

	result, err := h.repo.List(c.Request.Context(), pageReq, includeDeleted, sortReq)
	if err != nil {
		if isCursorError(err) {
			errorJSON(c, http.StatusBadRequest, err.Error())
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to fetch cities")
		return
	}

	writeCursorPage(c, result)
}

// GetCity handles GET /api/v1/cities/:id (public, active cities only).
func (h *CityHandler) GetCity(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	city, err := h.repo.GetActiveByID(c.Request.Context(), id)
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
	if err := req.validate(); err != nil {
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
		if handleDBError(c, repository.ClassifyError(err), "city") {
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to create city")
		return
	}

	auditEntry(c, h.audit, "create", "city", resourceID(created.ID), req)
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
	if err := req.validate(); err != nil {
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
		if handleDBError(c, repository.ClassifyError(err), "city") {
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to update city")
		return
	}

	auditEntry(c, h.audit, "update", "city", resourceID(id), req)
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

// DeleteCity handles DELETE /api/v1/admin/cities/:id.
// Soft-deletes the city (sets deleted_at). Audio cleanup is skipped because
// the city can be restored; audio files remain valid while POIs/stories exist.
func (h *CityHandler) DeleteCity(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	err := h.repo.Delete(c.Request.Context(), id)
	if err != nil {
		if handleDBError(c, repository.ClassifyError(err), "city") {
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to delete city")
		return
	}

	auditEntry(c, h.audit, "delete", "city", resourceID(id), nil)
	c.JSON(http.StatusOK, gin.H{"message": "city deleted"})
}

// RestoreCity handles POST /api/v1/admin/cities/:id/restore.
func (h *CityHandler) RestoreCity(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	restored, err := h.repo.Restore(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			errorJSON(c, http.StatusNotFound, "city not found or not deleted")
			return
		}
		if handleDBError(c, repository.ClassifyError(err), "city") {
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to restore city")
		return
	}

	auditEntry(c, h.audit, "restore", "city", resourceID(id), nil)
	c.JSON(http.StatusOK, gin.H{"data": restored})
}

// GetDownloadManifest handles GET /api/v1/cities/:id/download-manifest.
func (h *CityHandler) GetDownloadManifest(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	// Verify city exists and is active (public endpoint)
	city, err := h.repo.GetActiveByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			errorJSON(c, http.StatusNotFound, "city not found")
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to fetch city")
		return
	}

	language := c.DefaultQuery("language", "en")
	if !supportedLanguages[language] {
		errorJSON(c, http.StatusBadRequest, "unsupported language; supported: en, ru")
		return
	}

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
		"city_name":        city.DisplayName(language),
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

func parseOptionalBoolQuery(c *gin.Context, key string) (bool, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return false, true
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		errorJSON(c, http.StatusBadRequest, key+" must be a boolean")
		return false, false
	}

	return value, true
}

// isCursorError checks if an error is related to an invalid cursor.
func isCursorError(err error) bool {
	return errors.Is(err, domain.ErrInvalidCursor)
}

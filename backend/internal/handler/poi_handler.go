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

// POIRepository defines the interface for POI database operations.
type POIRepository interface {
	Create(ctx context.Context, poi *domain.POI) (*domain.POI, error)
	GetByID(ctx context.Context, id int) (*domain.POI, error)
	GetByCityID(ctx context.Context, cityID int, status *domain.POIStatus, poiType *domain.POIType) ([]domain.POI, error)
	ListByCityID(ctx context.Context, cityID int, status *domain.POIStatus, poiType *domain.POIType, page domain.PageRequest, sort repository.ListSort) (*domain.PageResponse[domain.POI], error)
	Update(ctx context.Context, poi *domain.POI) (*domain.POI, error)
	Delete(ctx context.Context, id int) error
}

// POICleanupService schedules orphaned audio cleanup on POI deletion.
type POICleanupService interface {
	DeletePOI(ctx context.Context, id int) error
}

// POIHandler handles CRUD operations for Points of Interest.
type POIHandler struct {
	repo    POIRepository
	audit   AuditLogger
	cleanup POICleanupService
}

// NewPOIHandler creates a new POIHandler.
func NewPOIHandler(repo POIRepository, audit AuditLogger, cleanup ...POICleanupService) *POIHandler {
	h := &POIHandler{repo: repo, audit: audit}
	if len(cleanup) > 0 {
		h.cleanup = cleanup[0]
	}
	return h
}

// createPOIRequest represents the request body for creating a POI.
type createPOIRequest struct {
	CityID        int               `json:"city_id" binding:"required"`
	Name          string            `json:"name" binding:"required,max=500"`
	NameRu        *string           `json:"name_ru"`
	Lat           float64           `json:"lat" binding:"required,gte=-90,lte=90"`
	Lng           float64           `json:"lng" binding:"required,gte=-180,lte=180"`
	Type          domain.POIType    `json:"type" binding:"required,oneof=building street park monument church bridge square museum district other"`
	Tags          *json.RawMessage  `json:"tags"`
	Address       *string           `json:"address" binding:"omitempty,max=1000"`
	InterestScore *int16            `json:"interest_score" binding:"omitempty,gte=0,lte=100"`
	Status        *domain.POIStatus `json:"status" binding:"omitempty,oneof=active disabled pending_review"`
}

// updatePOIRequest represents the request body for updating a POI.
type updatePOIRequest struct {
	CityID        int               `json:"city_id" binding:"required"`
	Name          string            `json:"name" binding:"required,max=500"`
	NameRu        *string           `json:"name_ru"`
	Lat           float64           `json:"lat" binding:"required,gte=-90,lte=90"`
	Lng           float64           `json:"lng" binding:"required,gte=-180,lte=180"`
	Type          domain.POIType    `json:"type" binding:"required,oneof=building street park monument church bridge square museum district other"`
	Tags          *json.RawMessage  `json:"tags"`
	Address       *string           `json:"address" binding:"omitempty,max=1000"`
	InterestScore *int16            `json:"interest_score" binding:"omitempty,gte=0,lte=100"`
	Status        *domain.POIStatus `json:"status" binding:"omitempty,oneof=active disabled pending_review"`
}

var adminPOISortColumns = map[string]repository.SortColumn{
	"id":             {Key: "id", Column: "id", Type: repository.SortValueInt},
	"name":           {Key: "name", Column: "name", Type: repository.SortValueString},
	"type":           {Key: "type", Column: "type", Type: repository.SortValueString},
	"status":         {Key: "status", Column: "status", Type: repository.SortValueString},
	"interest_score": {Key: "interest_score", Column: "interest_score", Type: repository.SortValueInt16},
	"created_at":     {Key: "created_at", Column: "created_at", Type: repository.SortValueTime},
	"updated_at":     {Key: "updated_at", Column: "updated_at", Type: repository.SortValueTime},
}

// ListPOIs handles GET /api/v1/pois?city_id=&status=&type=.
func (h *POIHandler) ListPOIs(c *gin.Context) {
	h.listPOIs(c, repository.ListSort{By: "id", Dir: repository.SortDirAsc})
}

// ListAdminPOIs handles GET /api/v1/admin/pois?city_id=&status=&type=.
func (h *POIHandler) ListAdminPOIs(c *gin.Context) {
	sortReq, ok := parseListSort(c, adminPOISortColumns, "id", repository.SortDirAsc)
	if !ok {
		return
	}

	h.listPOIs(c, sortReq)
}

func (h *POIHandler) listPOIs(c *gin.Context, sortReq repository.ListSort) {
	cityID, ok := parseRequiredQueryInt(c, "city_id")
	if !ok {
		return
	}

	var statusFilter *domain.POIStatus
	if s := c.Query("status"); s != "" {
		if err := domain.ValidateEnum(s, "status", []string{
			string(domain.POIStatusActive),
			string(domain.POIStatusDisabled),
			string(domain.POIStatusPendingReview),
		}); err != nil {
			validationErrorResponse(c, err)
			return
		}
		st := domain.POIStatus(s)
		statusFilter = &st
	}

	var typeFilter *domain.POIType
	if t := c.Query("type"); t != "" {
		if err := domain.ValidateEnum(t, "type", []string{
			string(domain.POITypeBuilding),
			string(domain.POITypeStreet),
			string(domain.POITypePark),
			string(domain.POITypeMonument),
			string(domain.POITypeChurch),
			string(domain.POITypeBridge),
			string(domain.POITypeSquare),
			string(domain.POITypeMuseum),
			string(domain.POITypeDistrict),
			string(domain.POITypeOther),
		}); err != nil {
			validationErrorResponse(c, err)
			return
		}
		pt := domain.POIType(t)
		typeFilter = &pt
	}

	pageReq, ok := parseCursorPagination(c)
	if !ok {
		return
	}

	result, err := h.repo.ListByCityID(c.Request.Context(), cityID, statusFilter, typeFilter, pageReq, sortReq)
	if err != nil {
		if isCursorError(err) {
			errorJSON(c, http.StatusBadRequest, err.Error())
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to fetch POIs")
		return
	}

	writeCursorPage(c, result)
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
			errorJSON(c, http.StatusNotFound, "POI not found")
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to fetch POI")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": poi})
}

// CreatePOI handles POST /api/v1/admin/pois.
func (h *POIHandler) CreatePOI(c *gin.Context) {
	var req createPOIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrorResponse(c, err)
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
		if handleDBError(c, repository.ClassifyError(err), "POI") {
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to create POI")
		return
	}

	auditEntry(c, h.audit, "create", "poi", resourceID(created.ID), req)
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
		validationErrorResponse(c, err)
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
		if handleDBError(c, repository.ClassifyError(err), "POI") {
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to update POI")
		return
	}

	auditEntry(c, h.audit, "update", "poi", resourceID(id), req)
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

// DeletePOI handles DELETE /api/v1/admin/pois/:id.
func (h *POIHandler) DeletePOI(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	var err error
	if h.cleanup != nil {
		err = h.cleanup.DeletePOI(c.Request.Context(), id)
	} else {
		err = h.repo.Delete(c.Request.Context(), id)
	}
	if err != nil {
		if handleDBError(c, repository.ClassifyError(err), "POI") {
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to delete POI")
		return
	}

	auditEntry(c, h.audit, "delete", "poi", resourceID(id), nil)
	c.JSON(http.StatusOK, gin.H{"message": "POI deleted"})
}

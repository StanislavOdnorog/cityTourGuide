package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// ReportRepository defines the interface for report database operations.
type ReportRepository interface {
	Create(ctx context.Context, storyID int, userID string, reportType domain.ReportType, comment *string, lat, lng *float64) (*domain.Report, error)
	GetByID(ctx context.Context, id int) (*domain.Report, error)
	GetAll(ctx context.Context, status string, page, perPage int) ([]domain.Report, int, error)
	List(ctx context.Context, status string, page domain.PageRequest) (*domain.PageResponse[domain.Report], error)
	UpdateStatus(ctx context.Context, id int, status domain.ReportStatus) (*domain.Report, error)
	GetByPOIID(ctx context.Context, poiID int) ([]domain.Report, error)
}

// ReportHandler handles report operations.
type ReportHandler struct {
	repo ReportRepository
}

// NewReportHandler creates a new ReportHandler.
func NewReportHandler(repo ReportRepository) *ReportHandler {
	return &ReportHandler{repo: repo}
}

type createReportRequest struct {
	StoryID int               `json:"story_id" binding:"required"`
	UserID  string            `json:"user_id" binding:"required"`
	Type    domain.ReportType `json:"type" binding:"required"`
	Comment *string           `json:"comment"`
	Lat     *float64          `json:"lat"`
	Lng     *float64          `json:"lng"`
}

// CreateReport handles POST /api/v1/reports.
func (h *ReportHandler) CreateReport(c *gin.Context) {
	var req createReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.StoryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "story_id must be a positive integer"})
		return
	}

	// Validate report type
	switch req.Type {
	case domain.ReportTypeWrongLocation, domain.ReportTypeWrongFact, domain.ReportTypeInappropriateContent:
		// valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be one of: wrong_location, wrong_fact, inappropriate_content"})
		return
	}

	// Validate lat/lng come in pairs
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

	report, err := h.repo.Create(
		c.Request.Context(),
		req.StoryID, req.UserID, req.Type, req.Comment,
		req.Lat, req.Lng,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create report"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": report})
}

// ListReports handles GET /api/v1/admin/reports.
func (h *ReportHandler) ListReports(c *gin.Context) {
	status := c.Query("status")

	pageReq, ok := parseCursorPagination(c)
	if !ok {
		return
	}

	result, err := h.repo.List(c.Request.Context(), status, pageReq)
	if err != nil {
		if isCursorError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch reports"})
		return
	}

	if result.Items == nil {
		result.Items = []domain.Report{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       result.Items,
		"next_cursor": result.NextCursor,
		"has_more":    result.HasMore,
	})
}

type updateReportStatusRequest struct {
	Status domain.ReportStatus `json:"status" binding:"required"`
}

// UpdateReportStatus handles PUT /api/v1/admin/reports/:id.
func (h *ReportHandler) UpdateReportStatus(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	var req updateReportStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status
	switch req.Status {
	case domain.ReportStatusNew, domain.ReportStatusReviewed, domain.ReportStatusResolved, domain.ReportStatusDismissed:
		// valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be one of: new, reviewed, resolved, dismissed"})
		return
	}

	report, err := h.repo.UpdateStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}

// ListByPOI handles GET /api/v1/admin/pois/:id/reports.
func (h *ReportHandler) ListByPOI(c *gin.Context) {
	poiID, ok := parseIDParam(c)
	if !ok {
		return
	}

	reports, err := h.repo.GetByPOIID(c.Request.Context(), poiID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch reports"})
		return
	}

	if reports == nil {
		reports = []domain.Report{}
	}

	c.JSON(http.StatusOK, gin.H{"data": reports})
}

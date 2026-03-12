package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// ReportRepository defines the interface for report database operations.
type ReportRepository interface {
	Create(ctx context.Context, storyID int, userID string, reportType domain.ReportType, comment *string, lat, lng *float64) (*domain.Report, error)
	GetByID(ctx context.Context, id int) (*domain.Report, error)
	List(ctx context.Context, status string, page domain.PageRequest) (*domain.PageResponse[domain.Report], error)
	ListAdmin(ctx context.Context, status string, page domain.PageRequest, sort repository.ListSort) (*domain.PageResponse[domain.AdminReportListItem], error)
	UpdateStatus(ctx context.Context, id int, status domain.ReportStatus) (*domain.Report, error)
	GetByPOIID(ctx context.Context, poiID int) ([]domain.Report, error)
}

// ReportModerationService defines the atomic admin moderation operation.
type ReportModerationService interface {
	DisableStory(ctx context.Context, reportID int) (*domain.ModeratedReportResult, error)
}

// ReportHandler handles report operations.
type ReportHandler struct {
	repo       ReportRepository
	moderation ReportModerationService
	audit      AuditLogger
}

// NewReportHandler creates a new ReportHandler.
func NewReportHandler(repo ReportRepository, moderation ReportModerationService, audit AuditLogger) *ReportHandler {
	return &ReportHandler{repo: repo, moderation: moderation, audit: audit}
}

type createReportRequest struct {
	StoryID int               `json:"story_id" binding:"required"`
	UserID  string            `json:"user_id" binding:"required"`
	Type    domain.ReportType `json:"type" binding:"required"`
	Comment *string           `json:"comment"`
	Lat     *float64          `json:"lat"`
	Lng     *float64          `json:"lng"`
}

var adminReportSortColumns = map[string]repository.SortColumn{
	"id":         {Key: "id", Column: "r.id", Type: repository.SortValueInt},
	"story_id":   {Key: "story_id", Column: "r.story_id", Type: repository.SortValueInt},
	"type":       {Key: "type", Column: "r.type", Type: repository.SortValueString},
	"status":     {Key: "status", Column: "r.status", Type: repository.SortValueString},
	"created_at": {Key: "created_at", Column: "r.created_at", Type: repository.SortValueTime},
}

// CreateReport handles POST /api/v1/reports.
func (h *ReportHandler) CreateReport(c *gin.Context) {
	var req createReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrorResponse(c, err)
		return
	}

	if req.StoryID <= 0 {
		errorJSON(c, http.StatusBadRequest, "story_id must be a positive integer")
		return
	}

	// Validate user_id is a valid UUID
	if err := domain.ValidateUUID(req.UserID); err != nil {
		if ve, ok := err.(*domain.ValidationError); ok {
			ve.Field = "user_id"
		}
		validationErrorResponse(c, err)
		return
	}

	// Validate comment length when provided
	if req.Comment != nil {
		if err := domain.ValidateStringLength(*req.Comment, "comment", 10, 1000); err != nil {
			validationErrorResponse(c, err)
			return
		}
	}

	// Validate report type
	switch req.Type {
	case domain.ReportTypeWrongLocation, domain.ReportTypeWrongFact, domain.ReportTypeInappropriateContent:
		// valid
	default:
		errorJSON(c, http.StatusBadRequest, "type must be one of: wrong_location, wrong_fact, inappropriate_content")
		return
	}

	// Validate lat/lng come in pairs and ranges
	if !validateCoordPair(c, req.Lat, req.Lng) {
		return
	}

	report, err := h.repo.Create(
		c.Request.Context(),
		req.StoryID, req.UserID, req.Type, req.Comment,
		req.Lat, req.Lng,
	)
	if err != nil {
		if handleDBError(c, repository.ClassifyError(err), "report") {
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to create report")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": report})
}

// ListReports handles GET /api/v1/admin/reports.
func (h *ReportHandler) ListReports(c *gin.Context) {
	status := c.Query("status")

	if status != "" {
		if err := domain.ValidateEnum(status, "status", []string{
			string(domain.ReportStatusNew),
			string(domain.ReportStatusReviewed),
			string(domain.ReportStatusResolved),
			string(domain.ReportStatusDismissed),
		}); err != nil {
			validationErrorResponse(c, err)
			return
		}
	}

	pageReq, ok := parseCursorPagination(c)
	if !ok {
		return
	}

	sortReq, ok := parseListSort(c, adminReportSortColumns, "id", repository.SortDirAsc)
	if !ok {
		return
	}

	result, err := h.repo.ListAdmin(c.Request.Context(), status, pageReq, sortReq)
	if err != nil {
		if isCursorError(err) {
			errorJSON(c, http.StatusBadRequest, err.Error())
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to fetch reports")
		return
	}

	writeCursorPage(c, result)
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
		validationErrorResponse(c, err)
		return
	}

	// Validate status
	switch req.Status {
	case domain.ReportStatusNew, domain.ReportStatusReviewed, domain.ReportStatusResolved, domain.ReportStatusDismissed:
		// valid
	default:
		errorJSON(c, http.StatusBadRequest, "status must be one of: new, reviewed, resolved, dismissed")
		return
	}

	report, err := h.repo.UpdateStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		if handleDBError(c, repository.ClassifyError(err), "report") {
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to update report")
		return
	}

	auditEntry(c, h.audit, "update_status", "report", resourceID(id), req)
	c.JSON(http.StatusOK, gin.H{"data": report})
}

// DisableStory handles POST /api/v1/admin/reports/:id/disable-story.
func (h *ReportHandler) DisableStory(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	result, err := h.moderation.DisableStory(c.Request.Context(), id)
	if err != nil {
		switch {
		case err == repository.ErrNotFound:
			errorJSON(c, http.StatusNotFound, "report not found")
			return
		default:
			if handleDBError(c, repository.ClassifyError(err), "report") {
				return
			}
			errorJSON(c, http.StatusInternalServerError, "failed to moderate report")
			return
		}
	}

	auditEntry(c, h.audit, "disable_story", "report", resourceID(id), nil)
	c.JSON(http.StatusOK, gin.H{"data": result})
}

// ListByPOI handles GET /api/v1/admin/pois/:id/reports.
func (h *ReportHandler) ListByPOI(c *gin.Context) {
	poiID, ok := parseIDParam(c)
	if !ok {
		return
	}

	reports, err := h.repo.GetByPOIID(c.Request.Context(), poiID)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to fetch reports")
		return
	}

	if reports == nil {
		reports = []domain.Report{}
	}

	c.JSON(http.StatusOK, gin.H{"data": reports})
}

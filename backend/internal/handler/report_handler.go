package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// ReportRepository defines the interface for report database operations.
type ReportRepository interface {
	GetByPOIID(ctx context.Context, poiID int) ([]domain.Report, error)
}

// ReportHandler handles report operations for admin panel.
type ReportHandler struct {
	repo ReportRepository
}

// NewReportHandler creates a new ReportHandler.
func NewReportHandler(repo ReportRepository) *ReportHandler {
	return &ReportHandler{repo: repo}
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

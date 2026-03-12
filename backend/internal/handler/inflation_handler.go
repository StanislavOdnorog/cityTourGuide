package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// InflationRepository defines the interface for inflation job database operations.
type InflationRepository interface {
	Create(ctx context.Context, job *domain.InflationJob) (*domain.InflationJob, error)
	GetByPOIID(ctx context.Context, poiID int) ([]domain.InflationJob, error)
	CountActiveByPOIID(ctx context.Context, poiID int) (int, error)
}

// InflationHandler handles inflation job operations for admin panel.
type InflationHandler struct {
	repo InflationRepository
}

// NewInflationHandler creates a new InflationHandler.
func NewInflationHandler(repo InflationRepository) *InflationHandler {
	return &InflationHandler{repo: repo}
}

// TriggerInflation handles POST /api/v1/admin/pois/:id/inflate.
func (h *InflationHandler) TriggerInflation(c *gin.Context) {
	poiID, ok := parseIDParam(c)
	if !ok {
		return
	}

	// Check if POI already has too many active jobs
	count, err := h.repo.CountActiveByPOIID(c.Request.Context(), poiID)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to check existing jobs")
		return
	}

	if count >= 3 {
		errorJSON(c, http.StatusConflict, "POI already has maximum inflation segments (3)")
		return
	}

	job := &domain.InflationJob{
		POIID:         poiID,
		Status:        domain.InflationJobStatusPending,
		TriggerType:   domain.InflationTriggerAdminManual,
		SegmentsCount: 0,
		MaxSegments:   3,
	}

	created, err := h.repo.Create(c.Request.Context(), job)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to create inflation job")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

// ListByPOI handles GET /api/v1/admin/pois/:id/inflation-jobs.
func (h *InflationHandler) ListByPOI(c *gin.Context) {
	poiID, ok := parseIDParam(c)
	if !ok {
		return
	}

	jobs, err := h.repo.GetByPOIID(c.Request.Context(), poiID)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to fetch inflation jobs")
		return
	}

	if jobs == nil {
		jobs = []domain.InflationJob{}
	}

	c.JSON(http.StatusOK, gin.H{"data": jobs})
}

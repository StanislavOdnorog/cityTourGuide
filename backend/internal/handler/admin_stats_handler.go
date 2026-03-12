package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// AdminStatsRepository defines the interface for aggregate admin stats queries.
type AdminStatsRepository interface {
	Get(ctx context.Context) (*repository.AdminStats, error)
}

// AdminStatsHandler handles admin dashboard summary requests.
type AdminStatsHandler struct {
	repo AdminStatsRepository
}

// NewAdminStatsHandler creates a new AdminStatsHandler.
func NewAdminStatsHandler(repo AdminStatsRepository) *AdminStatsHandler {
	return &AdminStatsHandler{repo: repo}
}

// Get handles GET /api/v1/admin/stats.
func (h *AdminStatsHandler) Get(c *gin.Context) {
	stats, err := h.repo.Get(c.Request.Context())
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to fetch admin stats")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

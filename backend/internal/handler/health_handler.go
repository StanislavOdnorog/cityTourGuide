package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DBPinger is implemented by *pgxpool.Pool.
type DBPinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler provides health and readiness check endpoints.
type HealthHandler struct {
	db DBPinger
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db DBPinger) *HealthHandler {
	return &HealthHandler{db: db}
}

// Healthz returns 200 if the server is running.
func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz returns 200 if the database is reachable, 503 otherwise.
func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		errorJSONWithFields(c, http.StatusServiceUnavailable, "database unreachable", gin.H{
			"status": "unavailable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

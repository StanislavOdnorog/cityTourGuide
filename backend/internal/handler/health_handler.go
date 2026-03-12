package handler

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// DBPinger is implemented by *pgxpool.Pool.
type DBPinger interface {
	Ping(ctx context.Context) error
}

const (
	readinessStatusOK          = "ok"
	readinessStatusDegraded    = "degraded"
	readinessStatusUnavailable = "unavailable"
)

// ReadinessCheck evaluates one readiness dependency.
type ReadinessCheck struct {
	Name     string
	Required bool
	Check    func(ctx context.Context) error
}

type readinessCheckResult struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Status   string `json:"status"`
}

type readinessResponse struct {
	Status string                 `json:"status"`
	Checks []readinessCheckResult `json:"checks"`
}

// HealthHandler provides health and readiness check endpoints.
type HealthHandler struct {
	db           DBPinger
	checks       []ReadinessCheck
	shuttingDown atomic.Bool
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db DBPinger, checks ...ReadinessCheck) *HealthHandler {
	return &HealthHandler{db: db, checks: checks}
}

// SetShuttingDown toggles readiness for connection draining.
func (h *HealthHandler) SetShuttingDown(shuttingDown bool) {
	h.shuttingDown.Store(shuttingDown)
}

// Healthz returns 200 if the server is running.
func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz returns component-level readiness details and 503 when required checks fail.
func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	resp := readinessResponse{
		Status: readinessStatusOK,
		Checks: make([]readinessCheckResult, 0, len(h.checks)+2),
	}

	var httpStatus = http.StatusOK
	var errMessage string

	shutdownStatus := readinessStatusOK
	if h.shuttingDown.Load() {
		shutdownStatus = readinessStatusUnavailable
		resp.Status = readinessStatusUnavailable
		httpStatus = http.StatusServiceUnavailable
		errMessage = "server shutting down"
	}

	resp.Checks = append(resp.Checks, readinessCheckResult{
		Name:     "server",
		Required: true,
		Status:   shutdownStatus,
	})

	databaseStatus := readinessStatusOK
	if err := h.db.Ping(ctx); err != nil {
		databaseStatus = readinessStatusUnavailable
		resp.Status = readinessStatusUnavailable
		httpStatus = http.StatusServiceUnavailable
		if errMessage == "" {
			errMessage = "database unreachable"
		}
	}

	resp.Checks = append(resp.Checks, readinessCheckResult{
		Name:     "database",
		Required: true,
		Status:   databaseStatus,
	})

	for _, check := range h.checks {
		status := readinessStatusOK
		if check.Check != nil {
			if err := check.Check(ctx); err != nil {
				if check.Required {
					status = readinessStatusUnavailable
					resp.Status = readinessStatusUnavailable
					httpStatus = http.StatusServiceUnavailable
					if errMessage == "" {
						errMessage = check.Name + " unavailable"
					}
				} else {
					status = readinessStatusDegraded
					if resp.Status == readinessStatusOK {
						resp.Status = readinessStatusDegraded
					}
				}
			}
		}

		resp.Checks = append(resp.Checks, readinessCheckResult{
			Name:     check.Name,
			Required: check.Required,
			Status:   status,
		})
	}

	if errMessage != "" {
		errorJSONWithFields(c, httpStatus, errMessage, gin.H{
			"status": resp.Status,
			"checks": resp.Checks,
		})
		return
	}

	c.JSON(httpStatus, resp)
}

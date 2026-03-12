package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	logutil "github.com/saas/city-stories-guide/backend/internal/logger"
)

// AuditLogger defines the interface for persisting audit log entries.
type AuditLogger interface {
	Insert(ctx context.Context, log *domain.AuditLog) error
}

// auditEntry records an admin action in the audit log.
// Errors are logged but never returned — audit failures must not break the request.
func auditEntry(c *gin.Context, logger AuditLogger, action, resourceType, resourceID string, payload any) {
	if logger == nil {
		return
	}

	actorID, _ := c.Get("user_id")
	actorStr, _ := actorID.(string)

	traceID, _ := c.Get("trace_id")
	traceStr, _ := traceID.(string)

	var payloadBytes []byte
	if payload != nil {
		sanitized := logutil.RedactAny(payload)
		if sanitized != nil {
			b, err := json.Marshal(sanitized)
			if err == nil && len(b) <= 4096 {
				payloadBytes = b
			}
		}
	}

	entry := &domain.AuditLog{
		ActorID:      actorStr,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		HTTPMethod:   c.Request.Method,
		RequestPath:  c.Request.URL.Path,
		TraceID:      traceStr,
		Payload:      payloadBytes,
		Status:       "success",
	}

	if err := logger.Insert(c.Request.Context(), entry); err != nil {
		slog.Error("failed to write audit log",
			"error", err,
			"action", action,
			"resource_type", resourceType,
			"resource_id", resourceID,
		)
	}
}

// resourceID returns the string representation of an int ID for audit logging.
func resourceID(id int) string {
	return fmt.Sprintf("%d", id)
}

// storyAuditPayload returns a trimmed payload for story audit logs,
// excluding the full text blob which can be up to 50k characters.
func storyAuditPayload(poiID int, language, layerType string) map[string]any {
	return map[string]any{
		"poi_id":     poiID,
		"language":   language,
		"layer_type": layerType,
	}
}

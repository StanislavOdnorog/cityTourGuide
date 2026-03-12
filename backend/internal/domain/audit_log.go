package domain

import (
	"encoding/json"
	"time"
)

// AuditLog represents a single admin audit trail entry.
type AuditLog struct {
	ID           int64     `json:"id"`
	ActorID      string    `json:"actor_id"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	HTTPMethod   string    `json:"http_method"`
	RequestPath  string    `json:"request_path"`
	TraceID      string    `json:"trace_id"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

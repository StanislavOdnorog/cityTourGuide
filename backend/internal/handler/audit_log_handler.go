package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// AuditLogRepository defines the interface for listing audit log entries.
type AuditLogRepository interface {
	List(ctx context.Context, filter repository.AuditLogFilter, page domain.PageRequest, sort repository.ListSort) (*domain.PageResponse[domain.AuditLog], error)
}

// AuditLogHandler handles admin audit log listing requests.
type AuditLogHandler struct {
	repo AuditLogRepository
}

// NewAuditLogHandler creates a new AuditLogHandler.
func NewAuditLogHandler(repo AuditLogRepository) *AuditLogHandler {
	return &AuditLogHandler{repo: repo}
}

// auditLogItem is the JSON response representation of an audit log entry.
type auditLogItem struct {
	ID           int64            `json:"id"`
	ActorID      string           `json:"actor_id"`
	Action       string           `json:"action"`
	ResourceType string           `json:"resource_type"`
	ResourceID   string           `json:"resource_id"`
	HTTPMethod   string           `json:"http_method"`
	RequestPath  string           `json:"request_path"`
	TraceID      string           `json:"trace_id"`
	Payload      *json.RawMessage `json:"payload"`
	Status       string           `json:"status"`
	CreatedAt    string           `json:"created_at"`
}

var adminAuditLogSortColumns = map[string]repository.SortColumn{
	"id":            {Key: "id", Column: "id", Type: repository.SortValueInt64},
	"action":        {Key: "action", Column: "action", Type: repository.SortValueString},
	"resource_type": {Key: "resource_type", Column: "resource_type", Type: repository.SortValueString},
	"status":        {Key: "status", Column: "status", Type: repository.SortValueString},
	"http_method":   {Key: "http_method", Column: "http_method", Type: repository.SortValueString},
	"request_path":  {Key: "request_path", Column: "request_path", Type: repository.SortValueString},
	"created_at":    {Key: "created_at", Column: "created_at", Type: repository.SortValueTime},
}

func toAuditLogItem(a domain.AuditLog) auditLogItem {
	item := auditLogItem{
		ID:           a.ID,
		ActorID:      a.ActorID,
		Action:       a.Action,
		ResourceType: a.ResourceType,
		ResourceID:   a.ResourceID,
		HTTPMethod:   a.HTTPMethod,
		RequestPath:  a.RequestPath,
		TraceID:      a.TraceID,
		Status:       a.Status,
		CreatedAt:    a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if a.Payload != nil {
		raw := json.RawMessage(a.Payload)
		item.Payload = &raw
	}
	return item
}

// List handles GET /api/v1/admin/audit-logs.
func (h *AuditLogHandler) List(c *gin.Context) {
	pageReq, ok := parseCursorPagination(c)
	if !ok {
		return
	}

	filter, ok := parseAuditLogFilter(c)
	if !ok {
		return
	}

	sortReq, ok := parseListSort(c, adminAuditLogSortColumns, "created_at", repository.SortDirDesc)
	if !ok {
		return
	}

	result, err := h.repo.List(c.Request.Context(), filter, pageReq, sortReq)
	if err != nil {
		if isCursorError(err) {
			errorJSON(c, http.StatusBadRequest, err.Error())
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to fetch audit logs")
		return
	}

	items := make([]auditLogItem, 0, len(result.Items))
	for _, a := range result.Items {
		items = append(items, toAuditLogItem(a))
	}

	writeCursorPageItems(c, items, result.NextCursor, result.HasMore)
}

// validAuditActions is the set of known audit log actions.
var validAuditActions = map[string]bool{
	"create":         true,
	"update":         true,
	"delete":         true,
	"restore":        true,
	"trigger":        true,
	"update_status":  true,
	"disable_story":  true,
}

// validAuditResourceTypes is the set of known audit log resource types.
var validAuditResourceTypes = map[string]bool{
	"city":           true,
	"poi":            true,
	"story":          true,
	"report":         true,
	"inflation_job":  true,
}

// validAuditStatuses is the set of known audit log statuses.
var validAuditStatuses = map[string]bool{
	"success": true,
	"error":   true,
}

func parseAuditLogFilter(c *gin.Context) (repository.AuditLogFilter, bool) {
	filter := repository.AuditLogFilter{
		ActorID:      c.Query("actor_id"),
		ResourceType: c.Query("resource_type"),
		Action:       c.Query("action"),
		Status:       c.Query("status"),
	}

	if filter.ActorID != "" && !isValidUUID(filter.ActorID) {
		errorJSON(c, http.StatusBadRequest, "actor_id must be a valid UUID")
		return filter, false
	}
	if filter.Action != "" && !validAuditActions[filter.Action] {
		errorJSON(c, http.StatusBadRequest, "action must be one of: create, update, delete, restore, trigger, update_status, disable_story")
		return filter, false
	}
	if filter.ResourceType != "" && !validAuditResourceTypes[filter.ResourceType] {
		errorJSON(c, http.StatusBadRequest, "resource_type must be one of: city, poi, story, report, inflation_job")
		return filter, false
	}
	if filter.Status != "" && !validAuditStatuses[filter.Status] {
		errorJSON(c, http.StatusBadRequest, "status must be one of: success, error")
		return filter, false
	}

	if value := c.Query("created_from"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			errorJSON(c, http.StatusBadRequest, "created_from must be a valid RFC3339 timestamp")
			return filter, false
		}
		filter.CreatedFrom = &parsed
	}

	if value := c.Query("created_to"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			errorJSON(c, http.StatusBadRequest, "created_to must be a valid RFC3339 timestamp")
			return filter, false
		}
		filter.CreatedTo = &parsed
	}

	if filter.CreatedFrom != nil && filter.CreatedTo != nil && filter.CreatedFrom.After(*filter.CreatedTo) {
		errorJSON(c, http.StatusBadRequest, "created_from must be before or equal to created_to")
		return filter, false
	}

	return filter, true
}

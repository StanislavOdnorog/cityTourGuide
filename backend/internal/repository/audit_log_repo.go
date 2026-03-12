package repository

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// AuditLogRepo handles persistence of admin audit log entries.
type AuditLogRepo struct {
	pool *pgxpool.Pool
}

// NewAuditLogRepo creates a new AuditLogRepo.
func NewAuditLogRepo(pool *pgxpool.Pool) *AuditLogRepo {
	return &AuditLogRepo{pool: pool}
}

// Insert persists an audit log entry.
func (r *AuditLogRepo) Insert(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO admin_audit_logs (actor_id, action, resource_type, resource_id, http_method, request_path, trace_id, payload, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, query,
		log.ActorID,
		log.Action,
		log.ResourceType,
		log.ResourceID,
		log.HTTPMethod,
		log.RequestPath,
		log.TraceID,
		log.Payload,
		log.Status,
	)
	if err != nil {
		return fmt.Errorf("audit_log_repo: insert: %w", err)
	}
	return nil
}

// AuditLogFilter holds optional filter parameters for listing audit logs.
type AuditLogFilter struct {
	ActorID      string
	ResourceType string
	Action       string
	Status       string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
}

func encodeAuditLogCursor(createdAt time.Time, id int64) string {
	return base64.URLEncoding.EncodeToString(
		[]byte(fmt.Sprintf("created_at:%d:id:%d", createdAt.UTC().UnixNano(), id)),
	)
}

func decodeAuditLogCursor(cursor string) (time.Time, int64, error) {
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor: malformed encoding")
	}

	parts := strings.Split(string(data), ":")
	if len(parts) != 4 || parts[0] != "created_at" || parts[2] != "id" {
		return time.Time{}, 0, fmt.Errorf("invalid cursor: unexpected format")
	}

	unixNano, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || unixNano <= 0 {
		return time.Time{}, 0, fmt.Errorf("invalid cursor: bad timestamp value")
	}

	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil || id <= 0 {
		return time.Time{}, 0, fmt.Errorf("invalid cursor: bad id value")
	}

	return time.Unix(0, unixNano).UTC(), id, nil
}

// List returns audit log entries with cursor-based pagination and optional filters.
func (r *AuditLogRepo) List(ctx context.Context, filter AuditLogFilter, page domain.PageRequest, sort ListSort) (*domain.PageResponse[domain.AuditLog], error) {
	if err := page.NormalizeLimit(); err != nil {
		return nil, fmt.Errorf("audit_log_repo: list: %w", err)
	}

	resolvedSort, err := ResolveSort(sort, map[string]SortColumn{
		"id":            {Key: "id", Column: "id", Type: SortValueInt64},
		"action":        {Key: "action", Column: "action", Type: SortValueString},
		"resource_type": {Key: "resource_type", Column: "resource_type", Type: SortValueString},
		"status":        {Key: "status", Column: "status", Type: SortValueString},
		"http_method":   {Key: "http_method", Column: "http_method", Type: SortValueString},
		"request_path":  {Key: "request_path", Column: "request_path", Type: SortValueString},
		"created_at":    {Key: "created_at", Column: "created_at", Type: SortValueTime},
	}, "created_at", SortDirDesc)
	if err != nil {
		return nil, fmt.Errorf("audit_log_repo: list: %w", err)
	}

	query := `
		SELECT id, actor_id, action, resource_type, resource_id,
		       http_method, request_path, trace_id, payload, status, created_at
		FROM admin_audit_logs`

	args := []interface{}{}
	argIdx := 1
	conditions := []string{}

	if filter.ActorID != "" {
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", argIdx))
		args = append(args, filter.ActorID)
		argIdx++
	}
	if filter.ResourceType != "" {
		conditions = append(conditions, fmt.Sprintf("resource_type = $%d", argIdx))
		args = append(args, filter.ResourceType)
		argIdx++
	}
	if filter.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, filter.Action)
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.CreatedFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, filter.CreatedFrom.UTC())
		argIdx++
	}
	if filter.CreatedTo != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, filter.CreatedTo.UTC())
		argIdx++
	}

	cursorCondition, cursorArgs, err := resolvedSort.CursorCondition(page.Cursor, argIdx)
	if err != nil {
		return nil, fmt.Errorf("audit_log_repo: list: %w", err)
	}
	if cursorCondition != "" {
		conditions = append(conditions, cursorCondition)
		args = append(args, cursorArgs...)
		argIdx += len(cursorArgs)
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, cond := range conditions[1:] {
			query += " AND " + cond
		}
	}

	query += fmt.Sprintf(" ORDER BY %s LIMIT $%d", resolvedSort.OrderBy(), argIdx)
	args = append(args, page.Limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("audit_log_repo: list: %w", err)
	}
	defer rows.Close()

	var items []domain.AuditLog
	for rows.Next() {
		var (
			item         domain.AuditLog
			actorID      sql.NullString
			resourceID   sql.NullString
			traceID      sql.NullString
			httpMethod   sql.NullString
			requestPath  sql.NullString
			status       sql.NullString
			action       sql.NullString
			resourceType sql.NullString
		)
		if err := rows.Scan(
			&item.ID, &actorID, &action, &resourceType, &resourceID,
			&httpMethod, &requestPath, &traceID, &item.Payload, &status, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("audit_log_repo: list scan: %w", err)
		}
		item.ActorID = actorID.String
		item.Action = action.String
		item.ResourceType = resourceType.String
		item.ResourceID = resourceID.String
		item.HTTPMethod = httpMethod.String
		item.RequestPath = requestPath.String
		item.TraceID = traceID.String
		item.Status = status.String
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit_log_repo: list rows: %w", err)
	}

	hasMore := len(items) > page.Limit
	if hasMore {
		items = items[:page.Limit]
	}

	var nextCursor string
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor, err = EncodeOrderedCursor64(resolvedSort, auditLogSortValue(last, resolvedSort.Key), last.ID)
		if err != nil {
			return nil, fmt.Errorf("audit_log_repo: list: %w", err)
		}
	}

	return &domain.PageResponse[domain.AuditLog]{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func auditLogSortValue(item domain.AuditLog, key string) interface{} {
	switch key {
	case "action":
		return item.Action
	case "resource_type":
		return item.ResourceType
	case "status":
		return item.Status
	case "http_method":
		return item.HTTPMethod
	case "request_path":
		return item.RequestPath
	case "created_at":
		return item.CreatedAt
	default:
		return item.ID
	}
}

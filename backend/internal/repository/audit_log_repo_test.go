//go:build integration

package repository_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

func TestAuditLogRepo_Insert(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewAuditLogRepo(tp.Pool)

	t.Run("with payload", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]string{"city": "Tbilisi"})
		entry := &domain.AuditLog{
			ActorID:      "admin-user-001",
			Action:       "created",
			ResourceType: "city",
			ResourceID:   "42",
			HTTPMethod:   "POST",
			RequestPath:  "/api/admin/cities",
			TraceID:      "trace-abc-123",
			Payload:      payload,
			Status:       "success",
		}

		if err := repo.Insert(ctx, entry); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		// Query directly to verify persisted columns
		var (
			id           int64
			actorID      string
			action       string
			resourceType string
			resourceID   string
			httpMethod   string
			requestPath  string
			traceID      string
			rawPayload   []byte
			status       string
			createdAt    time.Time
		)
		err := tp.Pool.QueryRow(ctx,
			`SELECT id, actor_id, action, resource_type, resource_id, http_method, request_path, trace_id, payload, status, created_at
			 FROM admin_audit_logs
			 WHERE trace_id = $1`, "trace-abc-123",
		).Scan(&id, &actorID, &action, &resourceType, &resourceID, &httpMethod, &requestPath, &traceID, &rawPayload, &status, &createdAt)
		if err != nil {
			t.Fatalf("query persisted row: %v", err)
		}
		defer func() {
			_, _ = tp.Pool.Exec(ctx, `DELETE FROM admin_audit_logs WHERE id = $1`, id)
		}()

		if id == 0 {
			t.Error("expected non-zero ID")
		}
		if actorID != "admin-user-001" {
			t.Errorf("actorID = %q, want %q", actorID, "admin-user-001")
		}
		if action != "created" {
			t.Errorf("action = %q, want %q", action, "created")
		}
		if resourceType != "city" {
			t.Errorf("resourceType = %q, want %q", resourceType, "city")
		}
		if resourceID != "42" {
			t.Errorf("resourceID = %q, want %q", resourceID, "42")
		}
		if httpMethod != "POST" {
			t.Errorf("httpMethod = %q, want %q", httpMethod, "POST")
		}
		if requestPath != "/api/admin/cities" {
			t.Errorf("requestPath = %q, want %q", requestPath, "/api/admin/cities")
		}
		if traceID != "trace-abc-123" {
			t.Errorf("traceID = %q, want %q", traceID, "trace-abc-123")
		}
		if status != "success" {
			t.Errorf("status = %q, want %q", status, "success")
		}
		if createdAt.IsZero() {
			t.Error("expected non-zero created_at")
		}

		// Verify JSON payload fidelity
		var parsed map[string]string
		if err := json.Unmarshal(rawPayload, &parsed); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if parsed["city"] != "Tbilisi" {
			t.Errorf("payload city = %q, want %q", parsed["city"], "Tbilisi")
		}
	})

	t.Run("without payload", func(t *testing.T) {
		entry := &domain.AuditLog{
			ActorID:      "admin-user-002",
			Action:       "deleted",
			ResourceType: "story",
			ResourceID:   "99",
			HTTPMethod:   "DELETE",
			RequestPath:  "/api/admin/stories/99",
			TraceID:      "trace-def-456",
			Payload:      nil,
			Status:       "error",
		}

		if err := repo.Insert(ctx, entry); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		var (
			id         int64
			rawPayload []byte
			status     string
			traceID    string
		)
		err := tp.Pool.QueryRow(ctx,
			`SELECT id, payload, status, trace_id FROM admin_audit_logs WHERE trace_id = $1`,
			"trace-def-456",
		).Scan(&id, &rawPayload, &status, &traceID)
		if err != nil {
			t.Fatalf("query persisted row: %v", err)
		}
		defer func() {
			_, _ = tp.Pool.Exec(ctx, `DELETE FROM admin_audit_logs WHERE id = $1`, id)
		}()

		if rawPayload != nil {
			t.Errorf("expected nil payload, got %s", rawPayload)
		}
		if status != "error" {
			t.Errorf("status = %q, want %q", status, "error")
		}
		if traceID != "trace-def-456" {
			t.Errorf("traceID = %q, want %q", traceID, "trace-def-456")
		}
	})
}

func TestAuditLogRepo_List_PaginationTraversal(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewAuditLogRepo(tp.Pool)
	actorID := "admin-pagination-" + time.Now().Format("150405.000000")

	createdIDs := make([]int64, 0, 3)
	for i := 0; i < 3; i++ {
		traceID := "trace-pagination-" + time.Now().Add(time.Duration(i)*time.Millisecond).Format("150405.000000") + "-" + string(rune('a'+i))
		entry := &domain.AuditLog{
			ActorID:      actorID,
			Action:       "update",
			ResourceType: "city",
			ResourceID:   "city-pagination",
			HTTPMethod:   "PUT",
			RequestPath:  "/api/v1/admin/cities/1",
			TraceID:      traceID,
			Status:       "success",
		}
		if err := repo.Insert(ctx, entry); err != nil {
			t.Fatalf("insert entry %d: %v", i, err)
		}

		var id int64
		if err := tp.Pool.QueryRow(ctx, `SELECT id FROM admin_audit_logs WHERE trace_id = $1`, traceID).Scan(&id); err != nil {
			t.Fatalf("fetch inserted id %d: %v", i, err)
		}
		createdIDs = append(createdIDs, id)
		defer func(id int64) {
			_, _ = tp.Pool.Exec(ctx, `DELETE FROM admin_audit_logs WHERE id = $1`, id)
		}(id)
	}

	filter := repository.AuditLogFilter{ActorID: actorID}

	firstPage, err := repo.List(ctx, filter, domain.PageRequest{
		Limit: 2,
	}, repository.ListSort{})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}

	if len(firstPage.Items) != 2 {
		t.Fatalf("expected 2 items on first page, got %d", len(firstPage.Items))
	}
	if !firstPage.HasMore {
		t.Fatal("expected first page to have more results")
	}
	if firstPage.NextCursor == "" {
		t.Fatal("expected next cursor on first page")
	}
	if firstPage.Items[0].CreatedAt.Before(firstPage.Items[1].CreatedAt) {
		t.Fatalf("expected created_at DESC order, got %s then %s", firstPage.Items[0].CreatedAt, firstPage.Items[1].CreatedAt)
	}

	secondPage, err := repo.List(ctx, filter, domain.PageRequest{
		Cursor: firstPage.NextCursor,
		Limit:  2,
	}, repository.ListSort{})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}

	if len(secondPage.Items) != 1 {
		t.Fatalf("expected 1 item on second page, got %d", len(secondPage.Items))
	}
	if secondPage.HasMore {
		t.Fatal("expected second page to be terminal")
	}
	if secondPage.NextCursor != "" {
		t.Fatal("expected empty next cursor on terminal page")
	}

	seen := make(map[int64]int)
	for _, item := range firstPage.Items {
		seen[item.ID]++
	}
	for _, item := range secondPage.Items {
		seen[item.ID]++
	}

	for _, id := range createdIDs {
		if seen[id] != 1 {
			t.Fatalf("expected audit log %d to appear exactly once, got %d", id, seen[id])
		}
	}
}

func TestAuditLogRepo_List_DeterministicWhenTimestampsMatch(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewAuditLogRepo(tp.Pool)

	sharedTime := time.Now().UTC().Truncate(time.Microsecond)
	tracePrefix := "trace-same-ts-" + sharedTime.Format("150405.000000")

	insertRow := func(suffix string) int64 {
		t.Helper()

		var id int64
		err := tp.Pool.QueryRow(ctx, `
			INSERT INTO admin_audit_logs (
				actor_id, action, resource_type, resource_id,
				http_method, request_path, trace_id, payload, status, created_at
			)
			VALUES ($1, 'update', 'city', 'same-ts', 'PUT', '/api/v1/admin/cities/1', $2, NULL, 'success', $3)
			RETURNING id
		`, "admin-same-ts", tracePrefix+"-"+suffix, sharedTime).Scan(&id)
		if err != nil {
			t.Fatalf("insert row %s: %v", suffix, err)
		}

		t.Cleanup(func() {
			_, _ = tp.Pool.Exec(ctx, `DELETE FROM admin_audit_logs WHERE id = $1`, id)
		})

		return id
	}

	firstID := insertRow("a")
	secondID := insertRow("b")
	thirdID := insertRow("c")

	pageOne, err := repo.List(ctx, repository.AuditLogFilter{
		ActorID: "admin-same-ts",
	}, domain.PageRequest{Limit: 2}, repository.ListSort{})
	if err != nil {
		t.Fatalf("list page one: %v", err)
	}

	if len(pageOne.Items) != 2 {
		t.Fatalf("expected 2 items on page one, got %d", len(pageOne.Items))
	}
	if !pageOne.HasMore {
		t.Fatal("expected page one to have more results")
	}
	if pageOne.Items[0].CreatedAt != pageOne.Items[1].CreatedAt {
		t.Fatalf("expected equal timestamps on first page, got %s and %s", pageOne.Items[0].CreatedAt, pageOne.Items[1].CreatedAt)
	}
	if pageOne.Items[0].ID <= pageOne.Items[1].ID {
		t.Fatalf("expected id DESC tiebreaker, got %d then %d", pageOne.Items[0].ID, pageOne.Items[1].ID)
	}

	pageTwo, err := repo.List(ctx, repository.AuditLogFilter{
		ActorID: "admin-same-ts",
	}, domain.PageRequest{
		Cursor: pageOne.NextCursor,
		Limit:  2,
	}, repository.ListSort{})
	if err != nil {
		t.Fatalf("list page two: %v", err)
	}

	if len(pageTwo.Items) != 1 {
		t.Fatalf("expected 1 item on page two, got %d", len(pageTwo.Items))
	}
	if pageTwo.Items[0].CreatedAt != sharedTime {
		t.Fatalf("expected shared timestamp %s, got %s", sharedTime, pageTwo.Items[0].CreatedAt)
	}

	seen := map[int64]bool{}
	for _, item := range pageOne.Items {
		seen[item.ID] = true
	}
	for _, item := range pageTwo.Items {
		seen[item.ID] = true
	}

	for _, id := range []int64{firstID, secondID, thirdID} {
		if !seen[id] {
			t.Fatalf("expected id %d to appear across pages", id)
		}
	}
}

func TestAuditLogRepo_List_WithCombinedFilters(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewAuditLogRepo(tp.Pool)

	matchingTraceID := "trace-filter-match-" + time.Now().Format("150405.000000")
	nonMatchingTraceID := "trace-filter-other-" + time.Now().Add(time.Millisecond).Format("150405.000000")

	insertRow := func(
		traceID, actorID, action, resourceType, resourceID, status string,
		createdAt time.Time,
	) int64 {
		t.Helper()
		payload := []byte(`{"kind":"audit"}`)
		var id int64
		err := tp.Pool.QueryRow(ctx, `
			INSERT INTO admin_audit_logs (
				actor_id, action, resource_type, resource_id,
				http_method, request_path, trace_id, payload, status, created_at
			)
			VALUES ($1, $2, $3, $4, 'POST', '/api/v1/admin/test', $5, $6, $7, $8)
			RETURNING id
		`, actorID, action, resourceType, resourceID, traceID, payload, status, createdAt).Scan(&id)
		if err != nil {
			t.Fatalf("insert audit row %s: %v", traceID, err)
		}
		return id
	}

	windowStart := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
	windowEnd := windowStart.Add(2 * time.Hour)

	matchID := insertRow(
		matchingTraceID, "admin-filter", "delete", "story", "story-7", "error", windowStart.Add(time.Hour),
	)
	defer func() {
		_, _ = tp.Pool.Exec(ctx, `DELETE FROM admin_audit_logs WHERE id = $1`, matchID)
	}()

	otherID := insertRow(
		nonMatchingTraceID, "admin-filter", "create", "story", "story-8", "success", windowStart.Add(-time.Hour),
	)
	defer func() {
		_, _ = tp.Pool.Exec(ctx, `DELETE FROM admin_audit_logs WHERE id = $1`, otherID)
	}()

	result, err := repo.List(ctx, repository.AuditLogFilter{
		ActorID:      "admin-filter",
		ResourceType: "story",
		Action:       "delete",
		Status:       "error",
		CreatedFrom:  &windowStart,
		CreatedTo:    &windowEnd,
	}, domain.PageRequest{Limit: 10}, repository.ListSort{})
	if err != nil {
		t.Fatalf("list with filters: %v", err)
	}

	if len(result.Items) != 1 {
		t.Fatalf("expected exactly 1 matching item, got %d", len(result.Items))
	}
	if result.Items[0].ID != matchID {
		t.Fatalf("expected item %d, got %d", matchID, result.Items[0].ID)
	}
	if result.Items[0].TraceID != matchingTraceID {
		t.Fatalf("expected trace id %q, got %q", matchingTraceID, result.Items[0].TraceID)
	}
}

func TestAuditLogRepo_List_InvalidCursor(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewAuditLogRepo(tp.Pool)

	_, err := repo.List(ctx, repository.AuditLogFilter{}, domain.PageRequest{
		Cursor: "not-base64",
		Limit:  20,
	}, repository.ListSort{})
	if err == nil {
		t.Fatal("expected error for invalid cursor")
	}
	if got := err.Error(); !strings.Contains(got, "invalid cursor") {
		t.Fatalf("expected wrapped descriptive error, got %q", got)
	}
}

func TestAuditLogRepo_List_EmptyPage(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewAuditLogRepo(tp.Pool)

	result, err := repo.List(ctx, repository.AuditLogFilter{
		ActorID: "actor-id-that-does-not-exist",
	}, domain.PageRequest{Limit: 5}, repository.ListSort{})
	if err != nil {
		t.Fatalf("list empty page: %v", err)
	}

	if len(result.Items) != 0 {
		t.Fatalf("expected no items, got %d", len(result.Items))
	}
	if result.HasMore {
		t.Fatal("expected has_more=false")
	}
	if result.NextCursor != "" {
		t.Fatalf("expected empty next cursor, got %q", result.NextCursor)
	}
}

func TestAuditLogRepo_List_SortsByActionAsc(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewAuditLogRepo(tp.Pool)

	insert := func(action string) int64 {
		t.Helper()
		traceID := "trace-action-" + action + "-" + time.Now().Format("150405.000000")
		entry := &domain.AuditLog{
			ActorID:      "admin-sort-action",
			Action:       action,
			ResourceType: "story",
			ResourceID:   action,
			HTTPMethod:   "POST",
			RequestPath:  "/api/v1/admin/test",
			TraceID:      traceID,
			Status:       "success",
		}
		if err := repo.Insert(ctx, entry); err != nil {
			t.Fatalf("insert %s: %v", action, err)
		}

		var id int64
		if err := tp.Pool.QueryRow(ctx, `SELECT id FROM admin_audit_logs WHERE trace_id = $1`, traceID).Scan(&id); err != nil {
			t.Fatalf("fetch %s id: %v", action, err)
		}
		t.Cleanup(func() {
			_, _ = tp.Pool.Exec(ctx, `DELETE FROM admin_audit_logs WHERE id = $1`, id)
		})
		return id
	}

	insert("update")
	insert("create")

	result, err := repo.List(ctx, repository.AuditLogFilter{
		ActorID: "admin-sort-action",
	}, domain.PageRequest{Limit: 10}, repository.ListSort{
		By:  "action",
		Dir: repository.SortDirAsc,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].Action != "create" || result.Items[1].Action != "update" {
		t.Fatalf("unexpected order: %q then %q", result.Items[0].Action, result.Items[1].Action)
	}
}

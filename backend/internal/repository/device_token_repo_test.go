//go:build integration

package repository_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

const deviceTokenTestUserID = "550e8400-e29b-41d4-a716-446655440088"
const deviceTokenTestUserID2 = "550e8400-e29b-41d4-a716-446655440089"

func seedDeviceTokenUser(t *testing.T, tp *repository.TestPool, userID string) func() {
	t.Helper()
	ctx := context.Background()
	if _, err := tp.Pool.Exec(ctx,
		`INSERT INTO users (id, auth_provider, is_anonymous) VALUES ($1, 'email', true) ON CONFLICT (id) DO NOTHING`,
		userID,
	); err != nil {
		t.Fatalf("seed user %s: %v", userID, err)
	}
	return func() {
		_, _ = tp.Pool.Exec(ctx, `DELETE FROM device_tokens WHERE user_id = $1`, userID)
		_, _ = tp.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	}
}

func setDeviceTokenUpdatedAt(t *testing.T, tp *repository.TestPool, token string, updatedAt time.Time) {
	t.Helper()
	ctx := context.Background()
	if _, err := tp.Pool.Exec(ctx, `UPDATE device_tokens SET updated_at = $1 WHERE token = $2`, updatedAt, token); err != nil {
		t.Fatalf("set updated_at for %s: %v", token, err)
	}
}

func TestDeviceTokenRepo_Upsert(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)
	cleanup := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup()

	dt, err := repo.Upsert(ctx, deviceTokenTestUserID, "token-upsert-001", domain.DevicePlatformIOS)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if dt.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if dt.UserID != deviceTokenTestUserID {
		t.Errorf("expected user_id %s, got %s", deviceTokenTestUserID, dt.UserID)
	}
	if dt.Token != "token-upsert-001" {
		t.Errorf("expected token token-upsert-001, got %s", dt.Token)
	}
	if dt.Platform != domain.DevicePlatformIOS {
		t.Errorf("expected platform ios, got %s", dt.Platform)
	}
	if !dt.IsActive {
		t.Error("expected is_active true")
	}
	if dt.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if dt.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestDeviceTokenRepo_Upsert_ReRegistration(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)
	cleanup := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup()

	// First registration
	first, err := repo.Upsert(ctx, deviceTokenTestUserID, "token-rereg-001", domain.DevicePlatformIOS)
	if err != nil {
		t.Fatalf("first Upsert failed: %v", err)
	}

	// Deactivate
	if err := repo.Deactivate(ctx, "token-rereg-001"); err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}

	// Re-register the same token — should reactivate
	second, err := repo.Upsert(ctx, deviceTokenTestUserID, "token-rereg-001", domain.DevicePlatformIOS)
	if err != nil {
		t.Fatalf("second Upsert failed: %v", err)
	}

	if second.ID != first.ID {
		t.Errorf("expected same ID after re-registration, got %d vs %d", first.ID, second.ID)
	}
	if !second.IsActive {
		t.Error("expected is_active true after re-registration")
	}
}

func TestDeviceTokenRepo_Upsert_DifferentUser_ReassignsRow(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)
	cleanup1 := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup1()
	cleanup2 := seedDeviceTokenUser(t, tp, deviceTokenTestUserID2)
	defer cleanup2()

	// User 1 registers the token
	first, err := repo.Upsert(ctx, deviceTokenTestUserID, "token-transfer-001", domain.DevicePlatformAndroid)
	if err != nil {
		t.Fatalf("first Upsert failed: %v", err)
	}

	// User 2 re-registers the same token (device changed hands)
	second, err := repo.Upsert(ctx, deviceTokenTestUserID2, "token-transfer-001", domain.DevicePlatformAndroid)
	if err != nil {
		t.Fatalf("second Upsert failed: %v", err)
	}

	// Row should be reassigned, not duplicated
	if second.ID != first.ID {
		t.Errorf("expected same row ID after reassignment (no duplicate), got %d vs %d", first.ID, second.ID)
	}
	if second.UserID != deviceTokenTestUserID2 {
		t.Errorf("expected user_id to be updated to %s, got %s", deviceTokenTestUserID2, second.UserID)
	}
	if !second.IsActive {
		t.Error("expected token to remain active after reassignment")
	}
	if second.UpdatedAt.Before(first.UpdatedAt) {
		t.Error("expected updated_at to advance after reassignment")
	}

	// User 1 should no longer have this token in active list
	tokens, err := repo.GetByUserID(ctx, deviceTokenTestUserID)
	if err != nil {
		t.Fatalf("GetByUserID failed: %v", err)
	}
	for _, tok := range tokens {
		if tok.Token == "token-transfer-001" {
			t.Error("token-transfer-001 should no longer belong to user 1")
		}
	}

	// User 2 should have it
	tokens2, err := repo.GetByUserID(ctx, deviceTokenTestUserID2)
	if err != nil {
		t.Fatalf("GetByUserID user2 failed: %v", err)
	}
	found := false
	for _, tok := range tokens2 {
		if tok.Token == "token-transfer-001" {
			found = true
		}
	}
	if !found {
		t.Error("expected token-transfer-001 in user 2's active tokens")
	}
}

func TestDeviceTokenRepo_Deactivate(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)
	cleanup := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup()

	_, err := repo.Upsert(ctx, deviceTokenTestUserID, "token-deact-001", domain.DevicePlatformIOS)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if err := repo.Deactivate(ctx, "token-deact-001"); err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}

	// GetByUserID should NOT return deactivated tokens
	tokens, err := repo.GetByUserID(ctx, deviceTokenTestUserID)
	if err != nil {
		t.Fatalf("GetByUserID failed: %v", err)
	}
	for _, tok := range tokens {
		if tok.Token == "token-deact-001" {
			t.Error("deactivated token should not appear in active list")
		}
	}

	// Verify via GetAllActive that deactivated tokens are excluded
	allActive, err := repo.GetAllActive(ctx)
	if err != nil {
		t.Fatalf("GetAllActive failed: %v", err)
	}
	for _, tok := range allActive {
		if tok.Token == "token-deact-001" {
			t.Error("deactivated token should not appear in GetAllActive")
		}
	}
}

// TestDeviceTokenRepo_Deactivate_Idempotent verifies that deactivating a
// nonexistent or already-deactivated token does not return an error.
func TestDeviceTokenRepo_Deactivate_Idempotent(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)
	cleanup := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup()

	// Deactivating a completely nonexistent token should succeed.
	if err := repo.Deactivate(ctx, "nonexistent-token"); err != nil {
		t.Fatalf("Deactivate nonexistent token: expected nil, got %v", err)
	}

	// Register, deactivate, then deactivate again — second call should also succeed.
	if _, err := repo.Upsert(ctx, deviceTokenTestUserID, "token-idem-001", domain.DevicePlatformIOS); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	if err := repo.Deactivate(ctx, "token-idem-001"); err != nil {
		t.Fatalf("first Deactivate failed: %v", err)
	}
	if err := repo.Deactivate(ctx, "token-idem-001"); err != nil {
		t.Fatalf("second Deactivate (idempotent) failed: %v", err)
	}
}

func TestDeviceTokenRepo_GetByUserID(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)
	cleanup := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup()

	// Register 3 tokens
	for i := 0; i < 3; i++ {
		token := "token-byuser-" + string(rune('a'+i))
		if _, err := repo.Upsert(ctx, deviceTokenTestUserID, token, domain.DevicePlatformIOS); err != nil {
			t.Fatalf("Upsert %d failed: %v", i, err)
		}
	}

	// Deactivate one
	if err := repo.Deactivate(ctx, "token-byuser-b"); err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}

	tokens, err := repo.GetByUserID(ctx, deviceTokenTestUserID)
	if err != nil {
		t.Fatalf("GetByUserID failed: %v", err)
	}

	if len(tokens) != 2 {
		t.Fatalf("expected 2 active tokens, got %d", len(tokens))
	}

	// All must belong to user and be active
	for _, tok := range tokens {
		if tok.UserID != deviceTokenTestUserID {
			t.Errorf("expected user_id %s, got %s", deviceTokenTestUserID, tok.UserID)
		}
		if !tok.IsActive {
			t.Error("expected all returned tokens to be active")
		}
		if tok.Token == "token-byuser-b" {
			t.Error("deactivated token should not be returned")
		}
	}
}

func TestDeviceTokenRepo_GetByUserID_Empty(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)
	cleanup := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup()

	tokens, err := repo.GetByUserID(ctx, deviceTokenTestUserID)
	if err != nil {
		t.Fatalf("GetByUserID failed: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(tokens))
	}
}

func TestDeviceTokenRepo_GetByID(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)
	cleanup := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup()

	created, err := repo.Upsert(ctx, deviceTokenTestUserID, "token-getbyid-001", domain.DevicePlatformAndroid)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, got.ID)
	}
	if got.Token != "token-getbyid-001" {
		t.Errorf("expected token token-getbyid-001, got %s", got.Token)
	}
	if got.Platform != domain.DevicePlatformAndroid {
		t.Errorf("expected platform android, got %s", got.Platform)
	}
}

func TestDeviceTokenRepo_GetByID_AfterReassignment(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)
	cleanup1 := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup1()
	cleanup2 := seedDeviceTokenUser(t, tp, deviceTokenTestUserID2)
	defer cleanup2()

	// User 1 registers
	created, err := repo.Upsert(ctx, deviceTokenTestUserID, "token-getbyid-reassign", domain.DevicePlatformIOS)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Reassign to user 2
	if _, err := repo.Upsert(ctx, deviceTokenTestUserID2, "token-getbyid-reassign", domain.DevicePlatformIOS); err != nil {
		t.Fatalf("reassign Upsert failed: %v", err)
	}

	// GetByID should reflect the new owner
	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.UserID != deviceTokenTestUserID2 {
		t.Errorf("expected user_id %s after reassignment, got %s", deviceTokenTestUserID2, got.UserID)
	}
	if !got.IsActive {
		t.Error("expected token to be active after reassignment")
	}
}

func TestDeviceTokenRepo_GetByID_AfterDeactivation(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)
	cleanup := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup()

	created, err := repo.Upsert(ctx, deviceTokenTestUserID, "token-getbyid-deact", domain.DevicePlatformAndroid)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if err := repo.Deactivate(ctx, "token-getbyid-deact"); err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}

	// GetByID should still return the row, but with is_active=false
	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.IsActive {
		t.Error("expected is_active=false after deactivation")
	}
	if got.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, got.ID)
	}
}

func TestDeviceTokenRepo_GetByID_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewDeviceTokenRepo(tp.Pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 999999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeviceTokenRepo_GetAllActive(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)
	cleanup1 := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup1()
	cleanup2 := seedDeviceTokenUser(t, tp, deviceTokenTestUserID2)
	defer cleanup2()

	// User 1: 2 tokens, one deactivated
	if _, err := repo.Upsert(ctx, deviceTokenTestUserID, "token-allactive-1a", domain.DevicePlatformIOS); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	if _, err := repo.Upsert(ctx, deviceTokenTestUserID, "token-allactive-1b", domain.DevicePlatformIOS); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	if err := repo.Deactivate(ctx, "token-allactive-1b"); err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}

	// User 2: 1 active token
	if _, err := repo.Upsert(ctx, deviceTokenTestUserID2, "token-allactive-2a", domain.DevicePlatformAndroid); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	allActive, err := repo.GetAllActive(ctx)
	if err != nil {
		t.Fatalf("GetAllActive failed: %v", err)
	}

	// Count our test tokens among the results (there may be other tokens from other tests)
	testTokens := map[string]bool{
		"token-allactive-1a": false,
		"token-allactive-1b": false,
		"token-allactive-2a": false,
	}
	for _, tok := range allActive {
		if _, ok := testTokens[tok.Token]; ok {
			testTokens[tok.Token] = true
		}
		if !tok.IsActive {
			t.Error("GetAllActive returned an inactive token")
		}
	}

	if !testTokens["token-allactive-1a"] {
		t.Error("expected token-allactive-1a in results")
	}
	if testTokens["token-allactive-1b"] {
		t.Error("deactivated token-allactive-1b should not be in results")
	}
	if !testTokens["token-allactive-2a"] {
		t.Error("expected token-allactive-2a in results")
	}
}

func TestDeviceTokenRepo_GetAllActive_Order(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)
	cleanup1 := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup1()
	cleanup2 := seedDeviceTokenUser(t, tp, deviceTokenTestUserID2)
	defer cleanup2()

	tokensToCreate := []struct {
		userID    string
		token     string
		updatedAt time.Time
	}{
		{deviceTokenTestUserID, "token-order-1a", time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)},
		{deviceTokenTestUserID, "token-order-1b", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)},
		{deviceTokenTestUserID2, "token-order-2a", time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tokensToCreate {
		if _, err := repo.Upsert(ctx, tc.userID, tc.token, domain.DevicePlatformIOS); err != nil {
			t.Fatalf("Upsert %s failed: %v", tc.token, err)
		}
		setDeviceTokenUpdatedAt(t, tp, tc.token, tc.updatedAt)
	}

	allActive, err := repo.GetAllActive(ctx)
	if err != nil {
		t.Fatalf("GetAllActive failed: %v", err)
	}

	var got []string
	for _, tok := range allActive {
		if strings.HasPrefix(tok.Token, "token-order-") {
			got = append(got, tok.Token)
		}
	}

	want := []string{"token-order-1a", "token-order-1b", "token-order-2a"}
	if len(got) != len(want) {
		t.Fatalf("filtered ordered tokens = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ordered tokens = %v, want %v", got, want)
		}
	}
}

func TestDeviceTokenRepo_GetAllActivePage(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)
	cleanup1 := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup1()
	cleanup2 := seedDeviceTokenUser(t, tp, deviceTokenTestUserID2)
	defer cleanup2()

	baseTime := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	tokensToCreate := []struct {
		userID    string
		token     string
		updatedAt time.Time
	}{
		{deviceTokenTestUserID, "token-page-1a", baseTime.Add(3 * time.Hour)},
		{deviceTokenTestUserID, "token-page-1b", baseTime.Add(2 * time.Hour)},
		{deviceTokenTestUserID, "token-page-1c", baseTime.Add(2 * time.Hour)},
		{deviceTokenTestUserID2, "token-page-2a", baseTime.Add(5 * time.Hour)},
	}

	for _, tc := range tokensToCreate {
		if _, err := repo.Upsert(ctx, tc.userID, tc.token, domain.DevicePlatformAndroid); err != nil {
			t.Fatalf("Upsert %s failed: %v", tc.token, err)
		}
		setDeviceTokenUpdatedAt(t, tp, tc.token, tc.updatedAt)
	}

	firstPage, err := repo.GetAllActivePage(ctx, domain.PageRequest{Limit: 2})
	if err != nil {
		t.Fatalf("GetAllActivePage first page failed: %v", err)
	}
	if len(firstPage.Items) != 2 {
		t.Fatalf("expected 2 items on first page, got %d", len(firstPage.Items))
	}
	if !firstPage.HasMore {
		t.Fatal("expected first page to have more results")
	}
	if firstPage.NextCursor == "" {
		t.Fatal("expected non-empty next cursor on first page")
	}

	firstWant := []string{"token-page-1a", "token-page-1b"}
	for i, want := range firstWant {
		if firstPage.Items[i].Token != want {
			t.Fatalf("first page tokens = [%s %s], want %v", firstPage.Items[0].Token, firstPage.Items[1].Token, firstWant)
		}
	}

	secondPage, err := repo.GetAllActivePage(ctx, domain.PageRequest{Cursor: firstPage.NextCursor, Limit: 2})
	if err != nil {
		t.Fatalf("GetAllActivePage second page failed: %v", err)
	}
	if len(secondPage.Items) != 2 {
		t.Fatalf("expected 2 items on second page, got %d", len(secondPage.Items))
	}
	if secondPage.HasMore {
		t.Fatal("expected second page to be terminal")
	}
	if secondPage.NextCursor != "" {
		t.Fatalf("expected empty next cursor on terminal page, got %q", secondPage.NextCursor)
	}

	secondWant := []string{"token-page-1c", "token-page-2a"}
	for i, want := range secondWant {
		if secondPage.Items[i].Token != want {
			t.Fatalf("second page tokens = [%s %s], want %v", secondPage.Items[0].Token, secondPage.Items[1].Token, secondWant)
		}
	}
}

func TestDeviceTokenRepo_GetAllActivePage_InvalidCursor(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)

	_, err := repo.GetAllActivePage(ctx, domain.PageRequest{Cursor: "bad", Limit: 10})
	if err == nil {
		t.Fatal("expected error for invalid cursor")
	}
	if got := err.Error(); !strings.Contains(got, "invalid cursor") {
		t.Fatalf("expected invalid cursor error, got %v", err)
	}
}

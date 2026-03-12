//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// seedPurchaseDeps creates a user and city needed for purchase tests.
func seedPurchaseDeps(t *testing.T, tp *repository.TestPool) (userID string, cityID int, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	cityRepo := repository.NewCityRepo(tp.Pool)
	city := createTestCity(t, cityRepo, ctx)

	userID = "550e8400-e29b-41d4-a716-446655440077"
	if _, err := tp.Pool.Exec(ctx,
		`INSERT INTO users (id, auth_provider, is_anonymous) VALUES ($1, 'email', true) ON CONFLICT (id) DO NOTHING`,
		userID,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	return userID, city.ID, func() {
		_, _ = tp.Pool.Exec(ctx, `DELETE FROM purchase WHERE user_id = $1`, userID)
		_, _ = tp.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
		_ = cityRepo.Delete(ctx, city.ID)
	}
}

func makePurchase(userID string, cityID *int, pType domain.PurchaseType, txnID *string, isLTD bool, expiresAt *time.Time) *domain.Purchase {
	return &domain.Purchase{
		UserID:        userID,
		Type:          pType,
		CityID:        cityID,
		Platform:      "ios",
		TransactionID: txnID,
		Price:         9.99,
		IsLTD:         isLTD,
		ExpiresAt:     expiresAt,
	}
}

func TestPurchaseRepo_Create(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewPurchaseRepo(tp.Pool)
	userID, cityID, cleanup := seedPurchaseDeps(t, tp)
	defer cleanup()

	txnID := "txn_create_test_001"
	p := makePurchase(userID, &cityID, domain.PurchaseTypeCityPack, &txnID, false, nil)

	created, err := repo.Create(ctx, p)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if created.UserID != userID {
		t.Errorf("expected user_id %s, got %s", userID, created.UserID)
	}
	if created.Type != domain.PurchaseTypeCityPack {
		t.Errorf("expected type city_pack, got %s", created.Type)
	}
	if created.CityID == nil || *created.CityID != cityID {
		t.Error("expected city_id to match")
	}
	if created.Platform != "ios" {
		t.Errorf("expected platform ios, got %s", created.Platform)
	}
	if created.TransactionID == nil || *created.TransactionID != txnID {
		t.Error("expected transaction_id to match")
	}
	if created.Price != 9.99 {
		t.Errorf("expected price 9.99, got %f", created.Price)
	}
	if created.IsLTD {
		t.Error("expected is_ltd false")
	}
	if created.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestPurchaseRepo_Create_DuplicateTransactionID(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewPurchaseRepo(tp.Pool)
	userID, cityID, cleanup := seedPurchaseDeps(t, tp)
	defer cleanup()

	txnID := "txn_dup_test_001"
	p := makePurchase(userID, &cityID, domain.PurchaseTypeCityPack, &txnID, false, nil)

	if _, err := repo.Create(ctx, p); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	_, err := repo.Create(ctx, p)
	if err == nil {
		t.Fatal("expected error for duplicate transaction_id, got nil")
	}
	if err != repository.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestPurchaseRepo_GetByID(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewPurchaseRepo(tp.Pool)
	userID, cityID, cleanup := seedPurchaseDeps(t, tp)
	defer cleanup()

	txnID := "txn_getbyid_001"
	created, err := repo.Create(ctx, makePurchase(userID, &cityID, domain.PurchaseTypeCityPack, &txnID, false, nil))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, got.ID)
	}
	if got.UserID != userID {
		t.Errorf("expected user_id %s, got %s", userID, got.UserID)
	}
}

func TestPurchaseRepo_GetByID_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewPurchaseRepo(tp.Pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 999999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPurchaseRepo_GetByTransactionID(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewPurchaseRepo(tp.Pool)
	userID, cityID, cleanup := seedPurchaseDeps(t, tp)
	defer cleanup()

	txnID := "txn_lookup_001"
	created, err := repo.Create(ctx, makePurchase(userID, &cityID, domain.PurchaseTypeCityPack, &txnID, false, nil))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByTransactionID(ctx, txnID)
	if err != nil {
		t.Fatalf("GetByTransactionID failed: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, got.ID)
	}
	if got.TransactionID == nil || *got.TransactionID != txnID {
		t.Error("expected transaction_id to match")
	}
}

func TestPurchaseRepo_GetByTransactionID_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewPurchaseRepo(tp.Pool)
	ctx := context.Background()

	_, err := repo.GetByTransactionID(ctx, "nonexistent_txn_id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPurchaseRepo_GetByUserID(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewPurchaseRepo(tp.Pool)
	userID, cityID, cleanup := seedPurchaseDeps(t, tp)
	defer cleanup()

	// Create 3 purchases for user
	for i := 0; i < 3; i++ {
		txnID := fmt.Sprintf("txn_user_list_%d", i)
		if _, err := repo.Create(ctx, makePurchase(userID, &cityID, domain.PurchaseTypeCityPack, &txnID, false, nil)); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}

	purchases, err := repo.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByUserID failed: %v", err)
	}

	if len(purchases) != 3 {
		t.Fatalf("expected 3 purchases, got %d", len(purchases))
	}

	// Verify ordering: created_at DESC (most recent first)
	for i := 1; i < len(purchases); i++ {
		if purchases[i].CreatedAt.After(purchases[i-1].CreatedAt) {
			t.Errorf("expected descending created_at order at index %d", i)
		}
	}

	// All must belong to user
	for _, p := range purchases {
		if p.UserID != userID {
			t.Errorf("expected user_id %s, got %s", userID, p.UserID)
		}
	}
}

func TestPurchaseRepo_GetByUserID_Empty(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewPurchaseRepo(tp.Pool)

	otherUserID := "550e8400-e29b-41d4-a716-446655440066"
	if _, err := tp.Pool.Exec(ctx,
		`INSERT INTO users (id, auth_provider, is_anonymous) VALUES ($1, 'email', true) ON CONFLICT (id) DO NOTHING`,
		otherUserID,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	defer func() { _, _ = tp.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, otherUserID) }()

	purchases, err := repo.GetByUserID(ctx, otherUserID)
	if err != nil {
		t.Fatalf("GetByUserID failed: %v", err)
	}
	if len(purchases) != 0 {
		t.Errorf("expected 0 purchases, got %d", len(purchases))
	}
}

func TestPurchaseRepo_GetActivePurchases(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewPurchaseRepo(tp.Pool)
	userID, cityID, cleanup := seedPurchaseDeps(t, tp)
	defer cleanup()

	// 1. Lifetime purchase (is_ltd=true) — always active
	txn1 := "txn_active_ltd"
	if _, err := repo.Create(ctx, makePurchase(userID, &cityID, domain.PurchaseTypeLifetime, &txn1, true, nil)); err != nil {
		t.Fatalf("Create lifetime: %v", err)
	}

	// 2. Active subscription (expires in the future)
	txn2 := "txn_active_sub"
	future := time.Now().Add(30 * 24 * time.Hour)
	if _, err := repo.Create(ctx, makePurchase(userID, &cityID, domain.PurchaseTypeSubscription, &txn2, false, &future)); err != nil {
		t.Fatalf("Create active subscription: %v", err)
	}

	// 3. Expired subscription (expired yesterday)
	txn3 := "txn_expired_sub"
	past := time.Now().Add(-24 * time.Hour)
	if _, err := repo.Create(ctx, makePurchase(userID, &cityID, domain.PurchaseTypeSubscription, &txn3, false, &past)); err != nil {
		t.Fatalf("Create expired subscription: %v", err)
	}

	// 4. City pack with no expiry (NULL expires_at) — always active
	txn4 := "txn_active_city"
	if _, err := repo.Create(ctx, makePurchase(userID, &cityID, domain.PurchaseTypeCityPack, &txn4, false, nil)); err != nil {
		t.Fatalf("Create city pack: %v", err)
	}

	active, err := repo.GetActivePurchases(ctx, userID)
	if err != nil {
		t.Fatalf("GetActivePurchases failed: %v", err)
	}

	// Should have 3 active: lifetime, future sub, city pack (no expiry)
	// The expired subscription should be excluded
	if len(active) != 3 {
		t.Fatalf("expected 3 active purchases, got %d", len(active))
	}

	activeIDs := map[string]bool{}
	for _, p := range active {
		if p.TransactionID != nil {
			activeIDs[*p.TransactionID] = true
		}
	}

	if !activeIDs[txn1] {
		t.Error("lifetime purchase should be active")
	}
	if !activeIDs[txn2] {
		t.Error("future subscription should be active")
	}
	if activeIDs[txn3] {
		t.Error("expired subscription should NOT be active")
	}
	if !activeIDs[txn4] {
		t.Error("city pack with no expiry should be active")
	}
}

func TestPurchaseRepo_CountListeningsSince(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewPurchaseRepo(tp.Pool)

	userID := "550e8400-e29b-41d4-a716-446655440055"
	if _, err := tp.Pool.Exec(ctx,
		`INSERT INTO users (id, auth_provider, is_anonymous) VALUES ($1, 'email', true) ON CONFLICT (id) DO NOTHING`,
		userID,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	defer func() {
		_, _ = tp.Pool.Exec(ctx, `DELETE FROM user_listening WHERE user_id = $1`, userID)
		_, _ = tp.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	}()

	// Need a story for the FK
	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "Listening POI", 41.69, 44.80)
	story1 := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Listening story 1")
	story2 := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Listening story 2")

	// Use start of today (UTC) as the since boundary, matching service behavior.
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Initially 0
	count, err := repo.CountListeningsSince(ctx, userID, startOfDay)
	if err != nil {
		t.Fatalf("CountListeningsSince failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	// Insert 2 listenings today
	for _, sID := range []int{story1.ID, story2.ID} {
		if _, err := tp.Pool.Exec(ctx,
			`INSERT INTO user_listening (user_id, story_id, listened_at) VALUES ($1, $2, NOW())
			 ON CONFLICT (user_id, story_id) DO UPDATE SET listened_at = NOW()`,
			userID, sID,
		); err != nil {
			t.Fatalf("insert listening for story %d: %v", sID, err)
		}
	}

	count, err = repo.CountListeningsSince(ctx, userID, startOfDay)
	if err != nil {
		t.Fatalf("CountListeningsSince failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}

	// Insert a listening from yesterday — should not count
	story3 := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Listening story 3")
	if _, err := tp.Pool.Exec(ctx,
		`INSERT INTO user_listening (user_id, story_id, listened_at) VALUES ($1, $2, NOW() - INTERVAL '1 day')
		 ON CONFLICT (user_id, story_id) DO UPDATE SET listened_at = NOW() - INTERVAL '1 day'`,
		userID, story3.ID,
	); err != nil {
		t.Fatalf("insert yesterday listening: %v", err)
	}

	count, err = repo.CountListeningsSince(ctx, userID, startOfDay)
	if err != nil {
		t.Fatalf("CountListeningsSince failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected still 2 (yesterday excluded), got %d", count)
	}
}

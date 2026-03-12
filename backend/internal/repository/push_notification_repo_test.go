//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

const pushNotifTestUserID = "550e8400-e29b-41d4-a716-446655440090"

// seedPushNotifDeps creates a user and an active device token for push notification tests.
func seedPushNotifDeps(t *testing.T, tp *repository.TestPool) (userID string, deviceTokenID int, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	userID = pushNotifTestUserID
	if _, err := tp.Pool.Exec(ctx,
		`INSERT INTO users (id, auth_provider, is_anonymous) VALUES ($1, 'email', true) ON CONFLICT (id) DO NOTHING`,
		userID,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	dtRepo := repository.NewDeviceTokenRepo(tp.Pool)
	dt, err := dtRepo.Upsert(ctx, userID, "push-notif-test-token-001", domain.DevicePlatformIOS)
	if err != nil {
		t.Fatalf("seed device token: %v", err)
	}
	deviceTokenID = dt.ID

	return userID, deviceTokenID, func() {
		_, _ = tp.Pool.Exec(ctx, `DELETE FROM push_notifications WHERE user_id = $1`, userID)
		_, _ = tp.Pool.Exec(ctx, `DELETE FROM device_tokens WHERE user_id = $1`, userID)
		_, _ = tp.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	}
}

func TestPushNotificationRepo_Create(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewPushNotificationRepo(tp.Pool)
	userID, dtID, cleanup := seedPushNotifDeps(t, tp)
	defer cleanup()

	now := time.Now()
	pn := &domain.PushNotification{
		UserID:        userID,
		DeviceTokenID: dtID,
		Type:          domain.NotificationTypeGeo,
		Title:         "Nearby story",
		Body:          "A new story is available near you.",
		Data:          map[string]any{"poi_id": 42},
		SentAt:        &now,
	}

	created, err := repo.Create(ctx, pn)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if created.UserID != userID {
		t.Errorf("expected user_id %s, got %s", userID, created.UserID)
	}
	if created.DeviceTokenID != dtID {
		t.Errorf("expected device_token_id %d, got %d", dtID, created.DeviceTokenID)
	}
	if created.Type != domain.NotificationTypeGeo {
		t.Errorf("expected type geo, got %s", created.Type)
	}
	if created.Title != "Nearby story" {
		t.Errorf("expected title 'Nearby story', got %s", created.Title)
	}
	if created.Body != "A new story is available near you." {
		t.Errorf("expected body mismatch, got %s", created.Body)
	}
	if created.SentAt == nil {
		t.Error("expected non-nil sent_at")
	}
	if created.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestPushNotificationRepo_Create_NilSentAt(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewPushNotificationRepo(tp.Pool)
	userID, dtID, cleanup := seedPushNotifDeps(t, tp)
	defer cleanup()

	pn := &domain.PushNotification{
		UserID:        userID,
		DeviceTokenID: dtID,
		Type:          domain.NotificationTypeContent,
		Title:         "New content",
		Body:          "Check out our latest stories.",
		SentAt:        nil,
	}

	created, err := repo.Create(ctx, pn)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.SentAt != nil {
		t.Errorf("expected nil sent_at, got %v", created.SentAt)
	}
}

func TestPushNotificationRepo_CountByUserAndTypeSince(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewPushNotificationRepo(tp.Pool)
	userID, dtID, cleanup := seedPushNotifDeps(t, tp)
	defer cleanup()

	now := time.Now()

	// Insert 3 geo notifications
	for i := 0; i < 3; i++ {
		pn := &domain.PushNotification{
			UserID:        userID,
			DeviceTokenID: dtID,
			Type:          domain.NotificationTypeGeo,
			Title:         "Geo notif",
			Body:          "Body",
			SentAt:        &now,
		}
		if _, err := repo.Create(ctx, pn); err != nil {
			t.Fatalf("Create geo %d failed: %v", i, err)
		}
	}

	// Insert 1 content notification
	contentNotif := &domain.PushNotification{
		UserID:        userID,
		DeviceTokenID: dtID,
		Type:          domain.NotificationTypeContent,
		Title:         "Content notif",
		Body:          "Body",
		SentAt:        &now,
	}
	if _, err := repo.Create(ctx, contentNotif); err != nil {
		t.Fatalf("Create content failed: %v", err)
	}

	// Count geo since 1 minute ago — should get 3
	since := time.Now().Add(-1 * time.Minute)
	count, err := repo.CountByUserAndTypeSince(ctx, userID, domain.NotificationTypeGeo, since)
	if err != nil {
		t.Fatalf("CountByUserAndTypeSince failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 geo notifications, got %d", count)
	}

	// Count content since 1 minute ago — should get 1
	count, err = repo.CountByUserAndTypeSince(ctx, userID, domain.NotificationTypeContent, since)
	if err != nil {
		t.Fatalf("CountByUserAndTypeSince content failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 content notification, got %d", count)
	}
}

func TestPushNotificationRepo_CountByUserAndTypeSince_ExcludesOldRecords(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewPushNotificationRepo(tp.Pool)
	userID, dtID, cleanup := seedPushNotifDeps(t, tp)
	defer cleanup()

	now := time.Now()

	// Create a recent notification
	recentNotif := &domain.PushNotification{
		UserID:        userID,
		DeviceTokenID: dtID,
		Type:          domain.NotificationTypeGeo,
		Title:         "Recent",
		Body:          "Body",
		SentAt:        &now,
	}
	if _, err := repo.Create(ctx, recentNotif); err != nil {
		t.Fatalf("Create recent failed: %v", err)
	}

	// Insert an old notification directly with a past created_at
	oldTime := time.Now().Add(-48 * time.Hour)
	if _, err := tp.Pool.Exec(ctx,
		`INSERT INTO push_notifications (user_id, device_token_id, type, title, body, sent_at, created_at)
		 VALUES ($1, $2, 'geo', 'Old', 'Body', $3, $3)`,
		userID, dtID, oldTime,
	); err != nil {
		t.Fatalf("insert old notification: %v", err)
	}

	// Count since 1 hour ago — should only get 1 (the recent one)
	since := time.Now().Add(-1 * time.Hour)
	count, err := repo.CountByUserAndTypeSince(ctx, userID, domain.NotificationTypeGeo, since)
	if err != nil {
		t.Fatalf("CountByUserAndTypeSince failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 (old record excluded), got %d", count)
	}

	// Count since 72 hours ago — should get both
	sinceFarBack := time.Now().Add(-72 * time.Hour)
	count, err = repo.CountByUserAndTypeSince(ctx, userID, domain.NotificationTypeGeo, sinceFarBack)
	if err != nil {
		t.Fatalf("CountByUserAndTypeSince far back failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 (both included), got %d", count)
	}
}

func TestPushNotificationRepo_CountByUserAndTypeSince_ZeroWhenNone(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewPushNotificationRepo(tp.Pool)
	userID, _, cleanup := seedPushNotifDeps(t, tp)
	defer cleanup()

	since := time.Now().Add(-1 * time.Hour)
	count, err := repo.CountByUserAndTypeSince(ctx, userID, domain.NotificationTypeGeo, since)
	if err != nil {
		t.Fatalf("CountByUserAndTypeSince failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

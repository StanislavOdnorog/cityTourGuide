package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/platform/fcm"
)

// --- Fakes ---

type fakeFCMSender struct {
	errByToken map[string]error
	sent       []string
}

func (f *fakeFCMSender) Send(_ context.Context, msg *fcm.Message) error {
	if err, ok := f.errByToken[msg.Token]; ok {
		return err
	}
	f.sent = append(f.sent, msg.Token)
	return nil
}

type fakeDeviceTokenRepo struct {
	tokens      map[string][]domain.DeviceToken
	deactivated []string
}

func (f *fakeDeviceTokenRepo) Upsert(_ context.Context, _, _ string, _ domain.DevicePlatform) (*domain.DeviceToken, error) {
	return nil, nil
}

func (f *fakeDeviceTokenRepo) Deactivate(_ context.Context, token string) error {
	f.deactivated = append(f.deactivated, token)
	return nil
}

func (f *fakeDeviceTokenRepo) GetByUserID(_ context.Context, userID string) ([]domain.DeviceToken, error) {
	return f.tokens[userID], nil
}

func (f *fakeDeviceTokenRepo) GetAllActive(_ context.Context) ([]domain.DeviceToken, error) {
	return nil, nil
}

func (f *fakeDeviceTokenRepo) GetAllActivePage(_ context.Context, _ domain.PageRequest) (*domain.PageResponse[domain.DeviceToken], error) {
	return &domain.PageResponse[domain.DeviceToken]{}, nil
}

func (f *fakeDeviceTokenRepo) GetByID(_ context.Context, _ int) (*domain.DeviceToken, error) {
	return nil, nil
}

type fakePushNotifRepo struct {
	created []*domain.PushNotification
	count   int
}

func (f *fakePushNotifRepo) Create(_ context.Context, n *domain.PushNotification) (*domain.PushNotification, error) {
	f.created = append(f.created, n)
	n.ID = len(f.created)
	return n, nil
}

func (f *fakePushNotifRepo) CountByUserAndTypeSince(_ context.Context, _ string, _ domain.NotificationType, _ time.Time) (int, error) {
	return f.count, nil
}

// --- Tests ---

func TestSendToUser_PermanentError_DeactivatesToken(t *testing.T) {
	sender := &fakeFCMSender{
		errByToken: map[string]error{
			"dead-token": fcm.NewSendError(404, "UNREGISTERED", "not found"),
		},
	}
	dtRepo := &fakeDeviceTokenRepo{
		tokens: map[string][]domain.DeviceToken{
			"user-1": {
				{ID: 1, UserID: "user-1", Token: "dead-token", Platform: domain.DevicePlatformIOS, IsActive: true},
				{ID: 2, UserID: "user-1", Token: "good-token", Platform: domain.DevicePlatformAndroid, IsActive: true},
			},
		},
	}
	notifRepo := &fakePushNotifRepo{}

	svc := NewPushNotificationService(dtRepo, notifRepo, sender, PushNotificationConfig{})

	sent, err := svc.SendGeoNotification(context.Background(), "user-1", "Hello", "World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sent {
		t.Error("expected sent=true because good-token should succeed")
	}

	if len(dtRepo.deactivated) != 1 || dtRepo.deactivated[0] != "dead-token" {
		t.Errorf("deactivated = %v, want [dead-token]", dtRepo.deactivated)
	}

	if len(notifRepo.created) != 1 {
		t.Fatalf("created = %d notifications, want 1", len(notifRepo.created))
	}
	if notifRepo.created[0].DeviceTokenID != 2 {
		t.Errorf("notification device_token_id = %d, want 2", notifRepo.created[0].DeviceTokenID)
	}

	if len(sender.sent) != 1 || sender.sent[0] != "good-token" {
		t.Errorf("sent tokens = %v, want [good-token]", sender.sent)
	}
}

func TestSendToUser_TransientError_DoesNotDeactivateToken(t *testing.T) {
	sender := &fakeFCMSender{
		errByToken: map[string]error{
			"flaky-token": fmt.Errorf("fcm: send request: connection reset"),
		},
	}
	dtRepo := &fakeDeviceTokenRepo{
		tokens: map[string][]domain.DeviceToken{
			"user-1": {
				{ID: 1, UserID: "user-1", Token: "flaky-token", Platform: domain.DevicePlatformIOS, IsActive: true},
			},
		},
	}
	notifRepo := &fakePushNotifRepo{}

	svc := NewPushNotificationService(dtRepo, notifRepo, sender, PushNotificationConfig{})

	sent, err := svc.SendGeoNotification(context.Background(), "user-1", "Hello", "World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sent {
		t.Error("expected sent=false because all tokens had transient errors")
	}

	if len(dtRepo.deactivated) != 0 {
		t.Errorf("deactivated = %v, want empty (transient error)", dtRepo.deactivated)
	}
	if len(notifRepo.created) != 0 {
		t.Errorf("created = %d notifications, want 0", len(notifRepo.created))
	}
}

func TestSendToUser_AllTokensPermanent_ReturnsFalse(t *testing.T) {
	sender := &fakeFCMSender{
		errByToken: map[string]error{
			"dead-1": fcm.NewSendError(404, "UNREGISTERED", "not found"),
			"dead-2": fcm.NewSendError(400, "INVALID_ARGUMENT", "bad token"),
		},
	}
	dtRepo := &fakeDeviceTokenRepo{
		tokens: map[string][]domain.DeviceToken{
			"user-1": {
				{ID: 1, UserID: "user-1", Token: "dead-1", Platform: domain.DevicePlatformIOS, IsActive: true},
				{ID: 2, UserID: "user-1", Token: "dead-2", Platform: domain.DevicePlatformAndroid, IsActive: true},
			},
		},
	}
	notifRepo := &fakePushNotifRepo{}

	svc := NewPushNotificationService(dtRepo, notifRepo, sender, PushNotificationConfig{})

	sent, err := svc.SendGeoNotification(context.Background(), "user-1", "Hello", "World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sent {
		t.Error("expected sent=false when all tokens are dead")
	}
	if len(dtRepo.deactivated) != 2 {
		t.Errorf("deactivated = %v, want both tokens", dtRepo.deactivated)
	}
	if len(notifRepo.created) != 0 {
		t.Errorf("created = %d notifications, want 0", len(notifRepo.created))
	}
}

func TestSendToUser_NilFCMClient_Skips(t *testing.T) {
	dtRepo := &fakeDeviceTokenRepo{}
	notifRepo := &fakePushNotifRepo{}

	svc := NewPushNotificationService(dtRepo, notifRepo, nil, PushNotificationConfig{})

	sent, err := svc.SendGeoNotification(context.Background(), "user-1", "Hello", "World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sent {
		t.Error("expected sent=false when FCM is nil")
	}
}

func TestSendToUser_TransientSendError_DoesNotDeactivate(t *testing.T) {
	// FCM returns a structured SendError but for a transient condition.
	sender := &fakeFCMSender{
		errByToken: map[string]error{
			"token-1": fcm.NewSendError(503, "UNAVAILABLE", "service unavailable"),
		},
	}
	dtRepo := &fakeDeviceTokenRepo{
		tokens: map[string][]domain.DeviceToken{
			"user-1": {
				{ID: 1, UserID: "user-1", Token: "token-1", Platform: domain.DevicePlatformIOS, IsActive: true},
			},
		},
	}
	notifRepo := &fakePushNotifRepo{}

	svc := NewPushNotificationService(dtRepo, notifRepo, sender, PushNotificationConfig{})

	sent, err := svc.SendGeoNotification(context.Background(), "user-1", "Hello", "World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sent {
		t.Error("expected sent=false")
	}
	if len(dtRepo.deactivated) != 0 {
		t.Errorf("deactivated = %v, want empty for transient SendError", dtRepo.deactivated)
	}
}

func TestSendContentNotification_PermanentError_DeactivatesToken(t *testing.T) {
	sender := &fakeFCMSender{
		errByToken: map[string]error{
			"dead-token": fcm.NewSendError(404, "UNREGISTERED", "not found"),
		},
	}
	dtRepo := &fakeDeviceTokenRepo{
		tokens: map[string][]domain.DeviceToken{
			"user-1": {
				{ID: 1, UserID: "user-1", Token: "dead-token", Platform: domain.DevicePlatformIOS, IsActive: true},
				{ID: 2, UserID: "user-1", Token: "good-token", Platform: domain.DevicePlatformAndroid, IsActive: true},
			},
		},
	}
	notifRepo := &fakePushNotifRepo{}

	svc := NewPushNotificationService(dtRepo, notifRepo, sender, PushNotificationConfig{})

	sent, err := svc.SendContentNotification(context.Background(), "user-1", "New Story", "Check it out")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sent {
		t.Error("expected sent=true because good-token should succeed")
	}
	if len(dtRepo.deactivated) != 1 || dtRepo.deactivated[0] != "dead-token" {
		t.Errorf("deactivated = %v, want [dead-token]", dtRepo.deactivated)
	}
	if len(notifRepo.created) != 1 {
		t.Errorf("created = %d notifications, want 1", len(notifRepo.created))
	}
}

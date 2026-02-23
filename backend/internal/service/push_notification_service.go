package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/platform/fcm"
)

// PushNotificationConfig holds configuration for the push notification service.
type PushNotificationConfig struct {
	GeoMaxPerDay      int // Max geo-push notifications per user per day (default 2)
	ContentMaxPerWeek int // Max content-push notifications per user per week (default 1)
}

// DeviceTokenRepository defines the interface for device token database operations.
type DeviceTokenRepository interface {
	Upsert(ctx context.Context, userID, token string, platform domain.DevicePlatform) (*domain.DeviceToken, error)
	Deactivate(ctx context.Context, token string) error
	GetByUserID(ctx context.Context, userID string) ([]domain.DeviceToken, error)
	GetAllActive(ctx context.Context) ([]domain.DeviceToken, error)
	GetByID(ctx context.Context, id int) (*domain.DeviceToken, error)
}

// PushNotificationRepository defines the interface for push notification database operations.
type PushNotificationRepository interface {
	Create(ctx context.Context, n *domain.PushNotification) (*domain.PushNotification, error)
	CountByUserAndTypeSince(ctx context.Context, userID string, notifType domain.NotificationType, since time.Time) (int, error)
}

// PushNotificationService handles push notification business logic.
type PushNotificationService struct {
	deviceTokenRepo DeviceTokenRepository
	pushNotifRepo   PushNotificationRepository
	fcmClient       *fcm.Client // nil if FCM is not configured
	config          PushNotificationConfig
}

// NewPushNotificationService creates a new PushNotificationService.
func NewPushNotificationService(
	deviceTokenRepo DeviceTokenRepository,
	pushNotifRepo PushNotificationRepository,
	fcmClient *fcm.Client,
	cfg PushNotificationConfig,
) *PushNotificationService {
	if cfg.GeoMaxPerDay <= 0 {
		cfg.GeoMaxPerDay = 2
	}
	if cfg.ContentMaxPerWeek <= 0 {
		cfg.ContentMaxPerWeek = 1
	}
	return &PushNotificationService{
		deviceTokenRepo: deviceTokenRepo,
		pushNotifRepo:   pushNotifRepo,
		fcmClient:       fcmClient,
		config:          cfg,
	}
}

// RegisterDeviceToken registers or updates a device push token for a user.
func (s *PushNotificationService) RegisterDeviceToken(ctx context.Context, userID, token string, platform domain.DevicePlatform) (*domain.DeviceToken, error) {
	if token == "" {
		return nil, fmt.Errorf("push: device token is required")
	}
	if platform != domain.DevicePlatformIOS && platform != domain.DevicePlatformAndroid {
		return nil, fmt.Errorf("push: invalid platform: %s", platform)
	}

	dt, err := s.deviceTokenRepo.Upsert(ctx, userID, token, platform)
	if err != nil {
		return nil, fmt.Errorf("push: register device token: %w", err)
	}

	slog.Info("device token registered", "user_id", userID, "platform", platform)
	return dt, nil
}

// UnregisterDeviceToken deactivates a device push token.
func (s *PushNotificationService) UnregisterDeviceToken(ctx context.Context, token string) error {
	if err := s.deviceTokenRepo.Deactivate(ctx, token); err != nil {
		return fmt.Errorf("push: unregister device token: %w", err)
	}
	return nil
}

// SendGeoNotification sends a geo-push notification to a user if rate limits allow.
// Returns true if the notification was sent, false if rate-limited or skipped.
func (s *PushNotificationService) SendGeoNotification(ctx context.Context, userID, title, body string) (bool, error) {
	if s.fcmClient == nil {
		slog.Debug("fcm not configured, skipping geo notification", "user_id", userID)
		return false, nil
	}

	// Check rate limit: max 2 geo pushes per day
	dayAgo := time.Now().Add(-24 * time.Hour)
	count, err := s.pushNotifRepo.CountByUserAndTypeSince(ctx, userID, domain.NotificationTypeGeo, dayAgo)
	if err != nil {
		return false, fmt.Errorf("push: check geo rate limit: %w", err)
	}
	if count >= s.config.GeoMaxPerDay {
		slog.Debug("geo notification rate limited", "user_id", userID, "count", count)
		return false, nil
	}

	return s.sendToUser(ctx, userID, domain.NotificationTypeGeo, title, body, map[string]string{"type": "geo"})
}

// SendContentNotification sends a content-push notification to a user if rate limits allow.
func (s *PushNotificationService) SendContentNotification(ctx context.Context, userID, title, body string) (bool, error) {
	if s.fcmClient == nil {
		slog.Debug("fcm not configured, skipping content notification", "user_id", userID)
		return false, nil
	}

	// Check rate limit: max 1 content push per week
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	count, err := s.pushNotifRepo.CountByUserAndTypeSince(ctx, userID, domain.NotificationTypeContent, weekAgo)
	if err != nil {
		return false, fmt.Errorf("push: check content rate limit: %w", err)
	}
	if count >= s.config.ContentMaxPerWeek {
		slog.Debug("content notification rate limited", "user_id", userID, "count", count)
		return false, nil
	}

	return s.sendToUser(ctx, userID, domain.NotificationTypeContent, title, body, map[string]string{"type": "content"})
}

// sendToUser sends a notification to all active device tokens for a user.
func (s *PushNotificationService) sendToUser(ctx context.Context, userID string, notifType domain.NotificationType, title, body string, data map[string]string) (bool, error) {
	tokens, err := s.deviceTokenRepo.GetByUserID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("push: get device tokens: %w", err)
	}

	if len(tokens) == 0 {
		return false, nil
	}

	sent := false
	now := time.Now()

	for _, dt := range tokens {
		err := s.fcmClient.Send(ctx, &fcm.Message{
			Token: dt.Token,
			Title: title,
			Body:  body,
			Data:  data,
		})
		if err != nil {
			slog.Error("push: failed to send notification", "user_id", userID, "token_id", dt.ID, "error", err)
			continue
		}

		// Record the sent notification
		_, recordErr := s.pushNotifRepo.Create(ctx, &domain.PushNotification{
			UserID:        userID,
			DeviceTokenID: dt.ID,
			Type:          notifType,
			Title:         title,
			Body:          body,
			Data:          convertData(data),
			SentAt:        &now,
		})
		if recordErr != nil {
			slog.Error("push: failed to record notification", "error", recordErr)
		}

		sent = true
	}

	return sent, nil
}

// GetUserDeviceTokens returns all active device tokens for a user.
func (s *PushNotificationService) GetUserDeviceTokens(ctx context.Context, userID string) ([]domain.DeviceToken, error) {
	return s.deviceTokenRepo.GetByUserID(ctx, userID)
}

func convertData(data map[string]string) map[string]any {
	if data == nil {
		return nil
	}
	result := make(map[string]any, len(data))
	for k, v := range data {
		result[k] = v
	}
	return result
}

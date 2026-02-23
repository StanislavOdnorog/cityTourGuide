package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// PushNotificationRepo handles database operations for push notification records.
type PushNotificationRepo struct {
	pool *pgxpool.Pool
}

// NewPushNotificationRepo creates a new PushNotificationRepo.
func NewPushNotificationRepo(pool *pgxpool.Pool) *PushNotificationRepo {
	return &PushNotificationRepo{pool: pool}
}

// Create inserts a new push notification record.
func (r *PushNotificationRepo) Create(ctx context.Context, n *domain.PushNotification) (*domain.PushNotification, error) {
	query := `
		INSERT INTO push_notifications (user_id, device_token_id, type, title, body, data, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, device_token_id, type, title, body, data, sent_at, created_at`

	var pn domain.PushNotification
	err := r.pool.QueryRow(ctx, query,
		n.UserID, n.DeviceTokenID, n.Type, n.Title, n.Body, n.Data, n.SentAt,
	).Scan(
		&pn.ID, &pn.UserID, &pn.DeviceTokenID, &pn.Type,
		&pn.Title, &pn.Body, &pn.Data, &pn.SentAt, &pn.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("push_notification_repo: create: %w", err)
	}

	return &pn, nil
}

// CountByUserAndTypeSince counts how many notifications of a given type were sent
// to a user since a given time. Used for rate limiting.
func (r *PushNotificationRepo) CountByUserAndTypeSince(ctx context.Context, userID string, notifType domain.NotificationType, since time.Time) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM push_notifications
		WHERE user_id = $1 AND type = $2 AND created_at >= $3`

	var count int
	err := r.pool.QueryRow(ctx, query, userID, notifType, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("push_notification_repo: count by user and type since: %w", err)
	}

	return count, nil
}

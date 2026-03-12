package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// DeviceTokenRepo handles database operations for device push tokens.
type DeviceTokenRepo struct {
	pool *pgxpool.Pool
}

// NewDeviceTokenRepo creates a new DeviceTokenRepo.
func NewDeviceTokenRepo(pool *pgxpool.Pool) *DeviceTokenRepo {
	return &DeviceTokenRepo{pool: pool}
}

// Upsert inserts a device token or reactivates it if it already exists.
func (r *DeviceTokenRepo) Upsert(ctx context.Context, userID, token string, platform domain.DevicePlatform) (*domain.DeviceToken, error) {
	query := `
		INSERT INTO device_tokens (user_id, token, platform, is_active)
		VALUES ($1, $2, $3, true)
		ON CONFLICT (token) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			platform = EXCLUDED.platform,
			is_active = true,
			updated_at = NOW()
		RETURNING id, user_id, token, platform, is_active, created_at, updated_at`

	var dt domain.DeviceToken
	err := r.pool.QueryRow(ctx, query, userID, token, platform).Scan(
		&dt.ID, &dt.UserID, &dt.Token, &dt.Platform, &dt.IsActive,
		&dt.CreatedAt, &dt.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("device_token_repo: upsert: %w", err)
	}

	return &dt, nil
}

// Deactivate marks a device token as inactive.
// The operation is idempotent: deactivating an already-inactive or
// nonexistent token is not an error.
func (r *DeviceTokenRepo) Deactivate(ctx context.Context, token string) error {
	query := `UPDATE device_tokens SET is_active = false, updated_at = NOW() WHERE token = $1`
	_, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("device_token_repo: deactivate: %w", err)
	}
	return nil
}

// GetByUserID returns all active device tokens for a user.
func (r *DeviceTokenRepo) GetByUserID(ctx context.Context, userID string) ([]domain.DeviceToken, error) {
	query := `
		SELECT id, user_id, token, platform, is_active, created_at, updated_at
		FROM device_tokens
		WHERE user_id = $1 AND is_active = true
		ORDER BY updated_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("device_token_repo: get by user id: %w", err)
	}
	defer rows.Close()

	var tokens []domain.DeviceToken
	for rows.Next() {
		var dt domain.DeviceToken
		if err := rows.Scan(
			&dt.ID, &dt.UserID, &dt.Token, &dt.Platform, &dt.IsActive,
			&dt.CreatedAt, &dt.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("device_token_repo: get by user id scan: %w", err)
		}
		tokens = append(tokens, dt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("device_token_repo: get by user id rows: %w", err)
	}

	return tokens, nil
}

// GetByID returns a device token by its ID.
func (r *DeviceTokenRepo) GetByID(ctx context.Context, id int) (*domain.DeviceToken, error) {
	query := `
		SELECT id, user_id, token, platform, is_active, created_at, updated_at
		FROM device_tokens
		WHERE id = $1`

	var dt domain.DeviceToken
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&dt.ID, &dt.UserID, &dt.Token, &dt.Platform, &dt.IsActive,
		&dt.CreatedAt, &dt.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("device_token_repo: get by id: %w", err)
	}

	return &dt, nil
}

// GetAllActive returns all active device tokens across all users.
func (r *DeviceTokenRepo) GetAllActive(ctx context.Context) ([]domain.DeviceToken, error) {
	query := `
		SELECT id, user_id, token, platform, is_active, created_at, updated_at
		FROM device_tokens
		WHERE is_active = true
		ORDER BY user_id ASC, updated_at DESC, id ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("device_token_repo: get all active: %w", err)
	}
	defer rows.Close()

	var tokens []domain.DeviceToken
	for rows.Next() {
		var dt domain.DeviceToken
		if err := rows.Scan(
			&dt.ID, &dt.UserID, &dt.Token, &dt.Platform, &dt.IsActive,
			&dt.CreatedAt, &dt.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("device_token_repo: get all active scan: %w", err)
		}
		tokens = append(tokens, dt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("device_token_repo: get all active rows: %w", err)
	}

	return tokens, nil
}

// GetAllActivePage returns active device tokens across all users with cursor pagination.
func (r *DeviceTokenRepo) GetAllActivePage(ctx context.Context, page domain.PageRequest) (*domain.PageResponse[domain.DeviceToken], error) {
	if err := page.NormalizeLimit(); err != nil {
		return nil, fmt.Errorf("device_token_repo: get all active page: %w", err)
	}

	query := `
		SELECT id, user_id, token, platform, is_active, created_at, updated_at
		FROM device_tokens
		WHERE is_active = true`

	args := []interface{}{}
	argIdx := 1

	if page.Cursor != "" {
		userID, updatedAt, id, err := decodeDeviceTokenCursor(page.Cursor)
		if err != nil {
			return nil, fmt.Errorf("device_token_repo: get all active page: %w", err)
		}
		query += fmt.Sprintf(`
			AND (
				user_id > $%d
				OR (user_id = $%d AND updated_at < $%d)
				OR (user_id = $%d AND updated_at = $%d AND id > $%d)
			)`, argIdx, argIdx, argIdx+1, argIdx, argIdx+1, argIdx+2)
		args = append(args, userID, updatedAt, id)
		argIdx += 3
	}

	query += fmt.Sprintf(`
		ORDER BY user_id ASC, updated_at DESC, id ASC
		LIMIT $%d`, argIdx)
	args = append(args, page.Limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("device_token_repo: get all active page: %w", err)
	}
	defer rows.Close()

	var tokens []domain.DeviceToken
	for rows.Next() {
		var dt domain.DeviceToken
		if err := rows.Scan(
			&dt.ID, &dt.UserID, &dt.Token, &dt.Platform, &dt.IsActive,
			&dt.CreatedAt, &dt.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("device_token_repo: get all active page scan: %w", err)
		}
		tokens = append(tokens, dt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("device_token_repo: get all active page rows: %w", err)
	}

	hasMore := len(tokens) > page.Limit
	if hasMore {
		tokens = tokens[:page.Limit]
	}

	var nextCursor string
	if hasMore && len(tokens) > 0 {
		last := tokens[len(tokens)-1]
		nextCursor = encodeDeviceTokenCursor(last.UserID, last.UpdatedAt, last.ID)
	}

	return &domain.PageResponse[domain.DeviceToken]{
		Items:      tokens,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func encodeDeviceTokenCursor(userID string, updatedAt time.Time, id int) string {
	payload := fmt.Sprintf("device_token:%s|%s|%d", userID, updatedAt.UTC().Format(time.RFC3339Nano), id)
	return base64.URLEncoding.EncodeToString([]byte(payload))
}

func decodeDeviceTokenCursor(cursor string) (string, time.Time, int, error) {
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return "", time.Time{}, 0, fmt.Errorf("invalid cursor: malformed encoding")
	}

	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 || !strings.HasPrefix(parts[0], "device_token:") {
		return "", time.Time{}, 0, fmt.Errorf("invalid cursor: unexpected format")
	}

	userID := strings.TrimPrefix(parts[0], "device_token:")
	if userID == "" {
		return "", time.Time{}, 0, fmt.Errorf("invalid cursor: unexpected format")
	}

	updatedAt, err := time.Parse(time.RFC3339Nano, parts[1])
	if err != nil {
		return "", time.Time{}, 0, fmt.Errorf("invalid cursor: bad updated_at value")
	}

	id, err := strconv.Atoi(parts[2])
	if err != nil || id <= 0 {
		return "", time.Time{}, 0, fmt.Errorf("invalid cursor: bad id value")
	}

	return userID, updatedAt, id, nil
}

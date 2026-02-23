package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// userColumns is the canonical column list for the users table.
const userColumns = `id, email, name, password_hash, auth_provider, provider_id,
	language_pref, is_anonymous, is_admin, deleted_at, deletion_scheduled_at,
	created_at, updated_at`

// UserRepo handles database operations for users.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// scanUser scans a row into a domain.User using the canonical column order.
func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.PasswordHash,
		&u.AuthProvider,
		&u.ProviderID,
		&u.LanguagePref,
		&u.IsAnonymous,
		&u.IsAdmin,
		&u.DeletedAt,
		&u.DeletionScheduledAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	return &u, err
}

// Create inserts a new user and returns it with generated fields.
func (r *UserRepo) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	query := `
		INSERT INTO users (email, name, password_hash, auth_provider, provider_id, language_pref, is_anonymous)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + userColumns

	u, err := scanUser(r.pool.QueryRow(ctx, query,
		user.Email,
		user.Name,
		user.PasswordHash,
		user.AuthProvider,
		user.ProviderID,
		user.LanguagePref,
		user.IsAnonymous,
	))
	if err != nil {
		return nil, fmt.Errorf("user_repo: create: %w", err)
	}

	return u, nil
}

// GetByID returns a user by their UUID.
func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`

	u, err := scanUser(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user_repo: get by id: %w", err)
	}

	return u, nil
}

// GetByEmail returns a user by their email address.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE email = $1`

	u, err := scanUser(r.pool.QueryRow(ctx, query, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user_repo: get by email: %w", err)
	}

	return u, nil
}

// CreateAnonymous inserts a new anonymous user with a given device UUID as ID.
func (r *UserRepo) CreateAnonymous(ctx context.Context, deviceID, languagePref string) (*domain.User, error) {
	query := `
		INSERT INTO users (id, auth_provider, language_pref, is_anonymous)
		VALUES ($1, $2, $3, true)
		ON CONFLICT (id) DO UPDATE SET updated_at = NOW()
		RETURNING ` + userColumns

	u, err := scanUser(r.pool.QueryRow(ctx, query,
		deviceID,
		domain.AuthProviderEmail,
		languagePref,
	))
	if err != nil {
		return nil, fmt.Errorf("user_repo: create anonymous: %w", err)
	}

	return u, nil
}

// GetByProviderID returns a user by their OAuth provider and provider-specific ID.
func (r *UserRepo) GetByProviderID(ctx context.Context, provider domain.AuthProvider, providerID string) (*domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE auth_provider = $1 AND provider_id = $2`

	u, err := scanUser(r.pool.QueryRow(ctx, query, provider, providerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user_repo: get by provider id: %w", err)
	}

	return u, nil
}

// SoftDelete marks a user account for deletion with a 30-day grace period.
func (r *UserRepo) SoftDelete(ctx context.Context, id string) error {
	query := `
		UPDATE users
		SET deletion_scheduled_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deletion_scheduled_at IS NULL`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("user_repo: soft delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// RestoreAccount cancels a pending deletion within the grace period.
func (r *UserRepo) RestoreAccount(ctx context.Context, id string) error {
	query := `
		UPDATE users
		SET deletion_scheduled_at = NULL, updated_at = NOW()
		WHERE id = $1 AND deletion_scheduled_at IS NOT NULL AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("user_repo: restore account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// HardDeleteExpired permanently deletes users whose grace period has expired.
// It deletes all users scheduled for deletion more than gracePeriod ago.
// Related data (listenings, reports, purchases, etc.) is cascade-deleted by FK constraints.
func (r *UserRepo) HardDeleteExpired(ctx context.Context, gracePeriod time.Duration) (int64, error) {
	query := `
		DELETE FROM users
		WHERE deletion_scheduled_at IS NOT NULL
			AND deletion_scheduled_at < NOW() - $1::interval
			AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, query, fmt.Sprintf("%d seconds", int(gracePeriod.Seconds())))
	if err != nil {
		return 0, fmt.Errorf("user_repo: hard delete expired: %w", err)
	}

	return tag.RowsAffected(), nil
}

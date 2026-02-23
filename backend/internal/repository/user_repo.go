package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// UserRepo handles database operations for users.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// Create inserts a new user and returns it with generated fields.
func (r *UserRepo) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	query := `
		INSERT INTO users (email, name, password_hash, auth_provider, language_pref, is_anonymous)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, email, name, password_hash, auth_provider, language_pref, is_anonymous, created_at, updated_at`

	var u domain.User
	err := r.pool.QueryRow(ctx, query,
		user.Email,
		user.Name,
		user.PasswordHash,
		user.AuthProvider,
		user.LanguagePref,
		user.IsAnonymous,
	).Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.PasswordHash,
		&u.AuthProvider,
		&u.LanguagePref,
		&u.IsAnonymous,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("user_repo: create: %w", err)
	}

	return &u, nil
}

// GetByID returns a user by their UUID.
func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, email, name, password_hash, auth_provider, language_pref, is_anonymous, created_at, updated_at
		FROM users
		WHERE id = $1`

	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.PasswordHash,
		&u.AuthProvider,
		&u.LanguagePref,
		&u.IsAnonymous,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user_repo: get by id: %w", err)
	}

	return &u, nil
}

// GetByEmail returns a user by their email address.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, name, password_hash, auth_provider, language_pref, is_anonymous, created_at, updated_at
		FROM users
		WHERE email = $1`

	var u domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.PasswordHash,
		&u.AuthProvider,
		&u.LanguagePref,
		&u.IsAnonymous,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user_repo: get by email: %w", err)
	}

	return &u, nil
}

// CreateAnonymous inserts a new anonymous user with a given device UUID as ID.
func (r *UserRepo) CreateAnonymous(ctx context.Context, deviceID, languagePref string) (*domain.User, error) {
	query := `
		INSERT INTO users (id, auth_provider, language_pref, is_anonymous)
		VALUES ($1, $2, $3, true)
		ON CONFLICT (id) DO UPDATE SET updated_at = NOW()
		RETURNING id, email, name, password_hash, auth_provider, language_pref, is_anonymous, created_at, updated_at`

	var u domain.User
	err := r.pool.QueryRow(ctx, query,
		deviceID,
		domain.AuthProviderEmail,
		languagePref,
	).Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.PasswordHash,
		&u.AuthProvider,
		&u.LanguagePref,
		&u.IsAnonymous,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("user_repo: create anonymous: %w", err)
	}

	return &u, nil
}

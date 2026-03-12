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

// PurchaseRepo handles database operations for purchase records.
type PurchaseRepo struct {
	pool *pgxpool.Pool
}

// NewPurchaseRepo creates a new PurchaseRepo.
func NewPurchaseRepo(pool *pgxpool.Pool) *PurchaseRepo {
	return &PurchaseRepo{pool: pool}
}

// Create inserts a new purchase record.
func (r *PurchaseRepo) Create(ctx context.Context, p *domain.Purchase) (*domain.Purchase, error) {
	query := `
		INSERT INTO purchase (user_id, type, city_id, platform, transaction_id, price, is_ltd, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, type, city_id, platform, transaction_id, price, is_ltd, expires_at, created_at`

	var created domain.Purchase
	err := r.pool.QueryRow(ctx, query,
		p.UserID, p.Type, p.CityID, p.Platform,
		p.TransactionID, p.Price, p.IsLTD, p.ExpiresAt,
	).Scan(
		&created.ID, &created.UserID, &created.Type, &created.CityID,
		&created.Platform, &created.TransactionID, &created.Price,
		&created.IsLTD, &created.ExpiresAt, &created.CreatedAt,
	)
	if err != nil {
		return nil, ClassifyError(err)
	}

	return &created, nil
}

// GetByID retrieves a purchase by its ID.
func (r *PurchaseRepo) GetByID(ctx context.Context, id int) (*domain.Purchase, error) {
	query := `
		SELECT id, user_id, type, city_id, platform, transaction_id, price, is_ltd, expires_at, created_at
		FROM purchase
		WHERE id = $1`

	var p domain.Purchase
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.UserID, &p.Type, &p.CityID,
		&p.Platform, &p.TransactionID, &p.Price,
		&p.IsLTD, &p.ExpiresAt, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("purchase_repo: get by id: %w", err)
	}

	return &p, nil
}

// GetByTransactionID retrieves a purchase by its transaction ID.
func (r *PurchaseRepo) GetByTransactionID(ctx context.Context, transactionID string) (*domain.Purchase, error) {
	query := `
		SELECT id, user_id, type, city_id, platform, transaction_id, price, is_ltd, expires_at, created_at
		FROM purchase
		WHERE transaction_id = $1`

	var p domain.Purchase
	err := r.pool.QueryRow(ctx, query, transactionID).Scan(
		&p.ID, &p.UserID, &p.Type, &p.CityID,
		&p.Platform, &p.TransactionID, &p.Price,
		&p.IsLTD, &p.ExpiresAt, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("purchase_repo: get by transaction id: %w", err)
	}

	return &p, nil
}

// GetByUserID retrieves all purchases for a user.
func (r *PurchaseRepo) GetByUserID(ctx context.Context, userID string) ([]domain.Purchase, error) {
	query := `
		SELECT id, user_id, type, city_id, platform, transaction_id, price, is_ltd, expires_at, created_at
		FROM purchase
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("purchase_repo: get by user id: %w", err)
	}
	defer rows.Close()

	var purchases []domain.Purchase
	for rows.Next() {
		var p domain.Purchase
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.Type, &p.CityID,
			&p.Platform, &p.TransactionID, &p.Price,
			&p.IsLTD, &p.ExpiresAt, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("purchase_repo: get by user id scan: %w", err)
		}
		purchases = append(purchases, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("purchase_repo: get by user id rows: %w", err)
	}

	return purchases, nil
}

// GetActivePurchases retrieves all currently valid purchases for a user
// (lifetime, non-expired subscriptions, and city packs).
func (r *PurchaseRepo) GetActivePurchases(ctx context.Context, userID string) ([]domain.Purchase, error) {
	query := `
		SELECT id, user_id, type, city_id, platform, transaction_id, price, is_ltd, expires_at, created_at
		FROM purchase
		WHERE user_id = $1
		  AND (
		    is_ltd = true
		    OR expires_at IS NULL
		    OR expires_at > $2
		  )
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("purchase_repo: get active purchases: %w", err)
	}
	defer rows.Close()

	var purchases []domain.Purchase
	for rows.Next() {
		var p domain.Purchase
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.Type, &p.CityID,
			&p.Platform, &p.TransactionID, &p.Price,
			&p.IsLTD, &p.ExpiresAt, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("purchase_repo: get active purchases scan: %w", err)
		}
		purchases = append(purchases, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("purchase_repo: get active purchases rows: %w", err)
	}

	return purchases, nil
}

// CountListeningsSince returns how many stories a user has listened to since
// the given timestamp (inclusive). Callers are expected to pass a UTC day
// boundary so that freemium limits are deterministic regardless of database
// timezone settings.
func (r *PurchaseRepo) CountListeningsSince(ctx context.Context, userID string, since time.Time) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM user_listening
		WHERE user_id = $1
		  AND listened_at >= $2`

	var count int
	err := r.pool.QueryRow(ctx, query, userID, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("purchase_repo: count listenings since: %w", err)
	}

	return count, nil
}

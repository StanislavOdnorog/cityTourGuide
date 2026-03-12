package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Transient PostgreSQL error codes that are safe to retry.
const (
	pgSerializationFailure = "40001"
	pgDeadlockDetected     = "40P01"
)

// isTransientPgError returns true if err is a PostgreSQL error with a code that
// indicates a transient failure (serialization conflict or deadlock).
func isTransientPgError(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	switch pgErr.Code {
	case pgSerializationFailure, pgDeadlockDetected:
		return true
	default:
		return false
	}
}

// RetryableTx executes fn inside a transaction, retrying exactly once if the
// first attempt fails with a transient PostgreSQL error (serialization failure
// or deadlock). Context cancellation and deadline errors are never retried.
//
// The caller's fn receives a pgx.Tx that is committed automatically on success.
// If fn returns an error the transaction is rolled back before a potential retry.
func RetryableTx(ctx context.Context, pool *pgxpool.Pool, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
	err := runTx(ctx, pool, opts, fn)
	if err == nil {
		return nil
	}

	// Never retry if the context is already done.
	if ctx.Err() != nil {
		return err
	}

	if !isTransientPgError(err) {
		return err
	}

	// One retry for transient failures.
	return runTx(ctx, pool, opts, fn)
}

// runTx executes fn inside a single transaction attempt.
func runTx(ctx context.Context, pool *pgxpool.Pool, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
	tx, err := pool.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("repository: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("repository: commit tx: %w", err)
	}
	return nil
}

// PoolConfig holds optional pool tuning parameters.
type PoolConfig struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// NewPool creates a new pgx connection pool from the given database URL.
// If opts is provided, its values are applied to the pool configuration.
func NewPool(ctx context.Context, databaseURL string, opts ...PoolConfig) (*pgxpool.Pool, error) {
	pgxCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("repository: parse database url: %w", err)
	}

	if len(opts) > 0 {
		applyPoolConfig(pgxCfg, opts[0])
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return nil, fmt.Errorf("repository: connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("repository: ping database: %w", err)
	}

	return pool, nil
}

// applyPoolConfig sets pool tuning values on the pgxpool config.
func applyPoolConfig(cfg *pgxpool.Config, opts PoolConfig) {
	if opts.MaxConns > 0 {
		cfg.MaxConns = opts.MaxConns
	}
	if opts.MinConns >= 0 {
		cfg.MinConns = opts.MinConns
	}
	if opts.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = opts.MaxConnLifetime
	}
	if opts.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = opts.MaxConnIdleTime
	}
	if opts.HealthCheckPeriod > 0 {
		cfg.HealthCheckPeriod = opts.HealthCheckPeriod
	}
}

package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPool wraps a pgxpool.Pool for testing purposes.
type TestPool struct {
	Pool    *pgxpool.Pool
	Shared  bool // when true, Close is a no-op (pool is owned by a shared harness)
}

// NewTestPool creates a connection pool for integration tests.
func NewTestPool(ctx context.Context, databaseURL string) (*TestPool, error) {
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("test pool: %w", err)
	}
	return &TestPool{Pool: pool}, nil
}

// Close releases the connection pool unless it is shared.
func (tp *TestPool) Close() {
	if !tp.Shared {
		tp.Pool.Close()
	}
}

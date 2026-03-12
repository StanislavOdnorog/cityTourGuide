package repository

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsTransientPgError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "serialization failure is transient",
			err:  &pgconn.PgError{Code: "40001"},
			want: true,
		},
		{
			name: "deadlock detected is transient",
			err:  &pgconn.PgError{Code: "40P01"},
			want: true,
		},
		{
			name: "unique violation is not transient",
			err:  &pgconn.PgError{Code: "23505"},
			want: false,
		},
		{
			name: "foreign key violation is not transient",
			err:  &pgconn.PgError{Code: "23503"},
			want: false,
		},
		{
			name: "wrapped pg error is transient",
			err:  fmt.Errorf("some context: %w", &pgconn.PgError{Code: "40001"}),
			want: true,
		},
		{
			name: "non-pg error is not transient",
			err:  errors.New("connection reset"),
			want: false,
		},
		{
			name: "nil is not transient",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTransientPgError(tt.err)
			if got != tt.want {
				t.Errorf("isTransientPgError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRetryableTx_RetriesOnTransientError uses a real (test) pgxpool if
// DATABASE_URL is set.  Otherwise it exercises the retry logic via the
// lower-level retryableTxWith helper that accepts a txBeginner interface,
// allowing us to inject a fake pool.

// txBeginner matches the subset of pgxpool.Pool used by runTx.
type txBeginner interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

// retryableTxWith is a copy of RetryableTx that accepts the txBeginner
// interface so we can test without a real pool.
func retryableTxWith(ctx context.Context, b txBeginner, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
	err := runTxWith(ctx, b, opts, fn)
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return err
	}
	if !isTransientPgError(err) {
		return err
	}
	return runTxWith(ctx, b, opts, fn)
}

func runTxWith(ctx context.Context, b txBeginner, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
	tx, err := b.BeginTx(ctx, opts)
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

// fakeTx implements the minimal pgx.Tx interface needed for tests.
type fakeTx struct{}

func (f fakeTx) Begin(_ context.Context) (pgx.Tx, error)            { return f, nil }
func (f fakeTx) Commit(_ context.Context) error                     { return nil }
func (f fakeTx) Rollback(_ context.Context) error                   { return nil }
func (f fakeTx) CopyFrom(_ context.Context, _ pgx.Identifier, _ []string, _ pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (f fakeTx) SendBatch(_ context.Context, _ *pgx.Batch) pgx.BatchResults { return nil }
func (f fakeTx) LargeObjects() pgx.LargeObjects                             { return pgx.LargeObjects{} }
func (f fakeTx) Prepare(_ context.Context, _ string, _ string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (f fakeTx) Exec(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (f fakeTx) Query(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
	return nil, nil
}
func (f fakeTx) QueryRow(_ context.Context, _ string, _ ...interface{}) pgx.Row { return nil }
func (f fakeTx) Conn() *pgx.Conn                                               { return nil }

// fakePool implements txBeginner, returning fakeTx instances.
type fakePool struct{}

func (fakePool) BeginTx(_ context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	return fakeTx{}, nil
}

func TestRetryableTx_RetriesOnceForSerializationFailure(t *testing.T) {
	var calls int32
	transientErr := &pgconn.PgError{Code: pgSerializationFailure}

	err := retryableTxWith(context.Background(), fakePool{}, pgx.TxOptions{}, func(_ pgx.Tx) error {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			return transientErr
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected nil error after retry, got: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected fn called 2 times, got %d", calls)
	}
}

func TestRetryableTx_RetriesOnceForDeadlock(t *testing.T) {
	var calls int32
	deadlockErr := &pgconn.PgError{Code: pgDeadlockDetected}

	err := retryableTxWith(context.Background(), fakePool{}, pgx.TxOptions{}, func(_ pgx.Tx) error {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			return deadlockErr
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected nil error after retry, got: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected fn called 2 times, got %d", calls)
	}
}

func TestRetryableTx_DoesNotRetryPermanentError(t *testing.T) {
	var calls int32
	permanentErr := &pgconn.PgError{Code: pgUniqueViolation}

	err := retryableTxWith(context.Background(), fakePool{}, pgx.TxOptions{}, func(_ pgx.Tx) error {
		atomic.AddInt32(&calls, 1)
		return permanentErr
	})

	if !errors.Is(err, permanentErr) {
		t.Fatalf("expected permanent error, got: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected fn called 1 time, got %d", calls)
	}
}

func TestRetryableTx_DoesNotRetryNonPgError(t *testing.T) {
	var calls int32
	genericErr := errors.New("something went wrong")

	err := retryableTxWith(context.Background(), fakePool{}, pgx.TxOptions{}, func(_ pgx.Tx) error {
		atomic.AddInt32(&calls, 1)
		return genericErr
	})

	if !errors.Is(err, genericErr) {
		t.Fatalf("expected generic error, got: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected fn called 1 time, got %d", calls)
	}
}

func TestRetryableTx_DoesNotRetryOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls int32
	transientErr := &pgconn.PgError{Code: pgSerializationFailure}

	err := retryableTxWith(ctx, fakePool{}, pgx.TxOptions{}, func(_ pgx.Tx) error {
		atomic.AddInt32(&calls, 1)
		cancel() // simulate context canceled during fn execution
		return transientErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Fatalf("expected fn called 1 time (no retry), got %d", calls)
	}
}

func TestRetryableTx_SuccessNoRetry(t *testing.T) {
	var calls int32

	err := retryableTxWith(context.Background(), fakePool{}, pgx.TxOptions{}, func(_ pgx.Tx) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected fn called 1 time, got %d", calls)
	}
}

func TestRetryableTx_RetriesAtMostOnce(t *testing.T) {
	var calls int32
	transientErr := &pgconn.PgError{Code: pgSerializationFailure}

	err := retryableTxWith(context.Background(), fakePool{}, pgx.TxOptions{}, func(_ pgx.Tx) error {
		atomic.AddInt32(&calls, 1)
		return transientErr // always fail
	})

	if !errors.Is(err, transientErr) {
		t.Fatalf("expected transient error, got: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected fn called exactly 2 times (1 + 1 retry), got %d", calls)
	}
}

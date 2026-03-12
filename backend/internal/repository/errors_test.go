package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestClassifyError_Nil(t *testing.T) {
	if err := ClassifyError(nil); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestClassifyError_NonPgError(t *testing.T) {
	orig := errors.New("some random error")
	if got := ClassifyError(orig); got != orig {
		t.Fatalf("expected original error, got %v", got)
	}
}

func TestClassifyError_UniqueViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505", Message: "duplicate key"}
	if got := ClassifyError(pgErr); !errors.Is(got, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", got)
	}
}

func TestClassifyError_ForeignKeyViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23503", Message: "foreign key violation"}
	if got := ClassifyError(pgErr); !errors.Is(got, ErrInvalidReference) {
		t.Fatalf("expected ErrInvalidReference, got %v", got)
	}
}

func TestClassifyError_CheckViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23514", Message: "check constraint"}
	if got := ClassifyError(pgErr); !errors.Is(got, ErrCheckViolation) {
		t.Fatalf("expected ErrCheckViolation, got %v", got)
	}
}

func TestClassifyError_InvalidTextRep(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "22P02", Message: "invalid input syntax"}
	if got := ClassifyError(pgErr); !errors.Is(got, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", got)
	}
}

func TestClassifyError_UnknownPgCode(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "42P01", Message: "undefined table"}
	if got := ClassifyError(pgErr); got != pgErr {
		t.Fatalf("expected original PgError, got %v", got)
	}
}

func TestClassifyError_WrappedPgError(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505", Message: "duplicate key"}
	wrapped := fmt.Errorf("city_repo: create: %w", pgErr)
	if got := ClassifyError(wrapped); !errors.Is(got, ErrConflict) {
		t.Fatalf("expected ErrConflict from wrapped error, got %v", got)
	}
}

func TestClassifyError_ErrNotFoundPassesThrough(t *testing.T) {
	if got := ClassifyError(ErrNotFound); !errors.Is(got, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", got)
	}
}

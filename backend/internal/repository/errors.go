package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("record not found")

// ErrConflict is returned when a unique constraint is violated.
var ErrConflict = errors.New("record already exists")

// ErrInvalidReference is returned when a foreign key constraint is violated.
var ErrInvalidReference = errors.New("referenced record does not exist")

// ErrCheckViolation is returned when a check constraint is violated.
var ErrCheckViolation = errors.New("value violates check constraint")

// ErrInvalidInput is returned when the database rejects input (e.g. invalid enum text).
var ErrInvalidInput = errors.New("invalid input value")

// PostgreSQL error codes
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
	pgInvalidTextRep      = "22P02"
)

// ClassifyError translates a raw database error into a domain-level sentinel
// error when the underlying cause is a recognisable PostgreSQL constraint or
// input violation.  Unknown errors are returned unchanged.
func ClassifyError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case pgUniqueViolation:
		return ErrConflict
	case pgForeignKeyViolation:
		return ErrInvalidReference
	case pgCheckViolation:
		return ErrCheckViolation
	case pgInvalidTextRep:
		return ErrInvalidInput
	default:
		return err
	}
}

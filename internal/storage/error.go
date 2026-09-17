package storage

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound            = errors.New("record not found")
	ErrUniqueViolation     = errors.New("unique violation")
	ErrForeignKeyViolation = errors.New("foreign key violation")
	ErrDeadlock            = errors.New("deadlock")
	ErrDB                  = errors.New("database error")
)

func ErrRepo(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		if pgErr.Code == "23505" {
			return fmt.Errorf("storage.ErrRepo: %w: %w", ErrUniqueViolation, err)
		}
		if pgErr.Code == "23503" {
			return fmt.Errorf("storage.ErrRepo: %w: %w", ErrForeignKeyViolation, err)
		}
		if pgErr.Code == "40P01" {
			return fmt.Errorf("storage.ErrRepo: %w: %w", ErrDeadlock, err)
		}
	}
	return fmt.Errorf("storage.ErrRepo: %w: %w", ErrDB, err)
}

package storage

import (
	"errors"
	"fmt"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
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
		return fmt.Errorf("storage.ErrRepo(%w): %w, %s", apperr.ErrDebug, ErrNotFound, err)
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		if pgErr.Code == "23505" {
			return fmt.Errorf("storage.ErrRepo(%w): %w: %s", apperr.ErrDebug, ErrUniqueViolation, err)
		}
		if pgErr.Code == "23503" {
			return fmt.Errorf("storage.ErrRepo(%w): %w: %s", apperr.ErrDebug, ErrForeignKeyViolation, err)
		}
		if pgErr.Code == "40P01" {
			return fmt.Errorf("storage.ErrRepo(%w): %w: %s", apperr.ErrError, ErrDeadlock, err)
		}
	}
	return fmt.Errorf("storage.ErrRepo(%w): %w: %s", apperr.ErrWarn, ErrDB, err)
}

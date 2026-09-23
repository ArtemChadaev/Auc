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
	ErrCheckViolation      = errors.New("check violation")
)

func ErrRepo(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("storage.ErrRepo(%w): %w, %v", apperr.ErrDebug, ErrNotFound, err)
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		if pgErr.Code == "23505" {
			return fmt.Errorf("storage.ErrRepo(%w): %w: %v", apperr.ErrDebug, ErrUniqueViolation, err)
		}
		if pgErr.Code == "23503" {
			return fmt.Errorf("storage.ErrRepo(%w): %w: %v", apperr.ErrDebug, ErrForeignKeyViolation, err)
		}
		if pgErr.Code == "40P01" {
			return fmt.Errorf("storage.ErrRepo(%w): %w: %v", apperr.ErrError, ErrDeadlock, err)
		}
		if pgErr.Code == "23514" {
			return fmt.Errorf("storage.ErrRepo(%w): %w: %v", apperr.ErrDebug, ErrCheckViolation, err)
		}
	}
	return fmt.Errorf("storage.ErrRepo(%w): %w: %v", apperr.ErrWarn, ErrDB, err)
}

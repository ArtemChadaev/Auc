package storage

import (
	"errors"
	"fmt"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound            = apperr.NewAppErrorString("record not found", -4)
	ErrUniqueViolation     = apperr.NewAppErrorString("unique violation", -4)
	ErrForeignKeyViolation = apperr.NewAppErrorString("foreign key violation", -4)
	ErrDeadlock            = apperr.NewAppErrorString("deadlock", 4)
	ErrDB                  = apperr.NewAppErrorString("database error", 4)
	ErrCheckViolation      = apperr.NewAppErrorString("check violation", -4)
)

func ErrRepo(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("storage.ErrRepo: %w, %v", ErrNotFound, err)
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("storage.ErrRepo: %w: %v", ErrUniqueViolation, err)
		case "23503":
			return fmt.Errorf("storage.ErrRepo: %w: %v", ErrForeignKeyViolation, err)
		case "40P01":
			return fmt.Errorf("storage.ErrRepo: %w: %v", ErrDeadlock, err)
		case "23514":
			return fmt.Errorf("storage.ErrRepo: %w: %v", ErrCheckViolation, err)
		}
	}
	return fmt.Errorf("storage.ErrRepo: %w: %v", ErrDB, err)
}

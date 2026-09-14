package storage

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBTX interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func QueryOne[T any](ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (T, error) {
	var zero T
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return zero, err
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[T])
}

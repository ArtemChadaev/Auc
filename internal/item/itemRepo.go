package item

import (
	"context"
	"encoding/json"
	"fmt"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/internal/storage"
	"github.com/jackc/pgx/v5"
)

type Repo struct {
	db storage.DBTX
}

func NewRepo(db storage.DBTX) *Repo {
	return &Repo{db: db}
}

func (r *Repo) withTx(tx storage.DBTX) *Repo {
	return &Repo{db: tx}
}

func (r *Repo) create(ctx context.Context, uid uuid.UUID, key string, iType Type, mimeType string, size int64, metadata json.RawMessage) error {
	_, err := r.db.Exec(ctx, "INSERT INTO items VALUES (default, $1, $1, $2, $3, $4, $5, $6, default)", uid, key, iType, mimeType, size, metadata)
	if err != nil {
		return fmt.Errorf("itemRepo.Create: %w", storage.ErrRepo(err))
	}
	return nil
}

func (r *Repo) getOwner(ctx context.Context, id int64) (uuid.UUID, error) {
	var uid uuid.UUID
	row := r.db.QueryRow(ctx, "SELECT owner_id FROM items WHERE id = $1", id)
	err := row.Scan(&uid)
	if err != nil {
		return uid, fmt.Errorf("itemRepo.GetOwner: %w", storage.ErrRepo(err))
	}
	return uid, nil
}

func (r *Repo) getOwnerForUpdate(ctx context.Context, id int64) (uuid.UUID, error) {
	var uid uuid.UUID
	row := r.db.QueryRow(ctx, "SELECT owner_id FROM items WHERE id = $1 FOR UPDATE", id)
	err := row.Scan(&uid)
	if err != nil {
		return uid, fmt.Errorf("itemRepo.GetOwner: %w", storage.ErrRepo(err))
	}
	return uid, nil
}

func (r *Repo) newOwner(ctx context.Context, uid uuid.UUID, id int64) error {
	res, err := r.db.Exec(ctx, "UPDATE items SET owner_id = $1 WHERE id = $2", uid, id)
	if err != nil {
		return fmt.Errorf("itemRepo.NewOwner: %w", storage.ErrRepo(err))
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("itemRepo.NewOwner(%w): %w", apperr.ErrDebug, storage.ErrNotFound)
	}
	return nil
}

func (r *Repo) getItem(ctx context.Context, id int64) (Item, error) {
	var item Item
	row, err := r.db.Query(ctx, "SELECT * FROM items WHERE id = $1", id)
	if err != nil {
		return item, fmt.Errorf("itemRepo.GetItem: %w", storage.ErrRepo(err))
	}
	item, err = pgx.CollectExactlyOneRow(row, pgx.RowToStructByName[Item])
	if err != nil {
		return item, fmt.Errorf("itemRepo.GetItem: %w", storage.ErrRepo(err))
	}
	return item, nil
}

func (r *Repo) delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, "DELETE FROM items WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("itemRepo.Delete: %w", storage.ErrRepo(err))
	}
	return nil
}

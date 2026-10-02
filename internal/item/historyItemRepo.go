package item

import (
	"context"
	"fmt"
	"uuid"

	"github.com/ArtemChadaev/Auction/internal/storage"
	"github.com/jackc/pgx/v5"
)

func (r *Repo) newHistory(ctx context.Context, itemID int64, oldUser, newUser uuid.UUID, lotID *int64, description *string) error {
	if lotID == nil && description == nil || oldUser == newUser {
		return fmt.Errorf("historyItemRepo.newHistory: %w", storage.ErrCheckViolation)
	}
	_, err := r.db.Exec(ctx, "Insert into history_item values(default, $1, $2, $3, default, $4, $5)", itemID, oldUser, newUser, lotID, description)
	if err != nil {
		return fmt.Errorf("historyItemRepo.newHistory: %w", storage.ErrRepo(err))
	}
	return nil
}

func (r *Repo) listHistoryByItemID(ctx context.Context, itemID int64) ([]HistoryItem, error) {
	var items []HistoryItem
	rows, err := r.db.Query(ctx, "Select * from history_item where id = $1", itemID)
	if err != nil {
		return nil, fmt.Errorf("historyItemRepo.listHistoryByItemID: %w", storage.ErrRepo(err))
	}
	items, err = pgx.CollectRows(rows, pgx.RowToStructByName[HistoryItem])
	if err != nil {
		return nil, fmt.Errorf("historyItemRepo.listHistoryByItemID: %w", storage.ErrRepo(err))
	}
	return items, nil
}

func (r *Repo) listHistoryByUserID(ctx context.Context, uid uuid.UUID) ([]HistoryItem, error) {
	var items []HistoryItem
	rows, err := r.db.Query(ctx, "Select * from history_item where old_user = $1 OR new_user = $1", uid)
	if err != nil {
		return nil, fmt.Errorf("historyItemRepo.listHistoryByUserID: %w", storage.ErrRepo(err))
	}
	items, err = pgx.CollectRows(rows, pgx.RowToStructByName[HistoryItem])
	if err != nil {
		return nil, fmt.Errorf("historyItemRepo.listHistoryByUserID: %w", storage.ErrRepo(err))
	}
	return items, nil
}

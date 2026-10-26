package lot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/storage"
	"github.com/ArtemChadaev/Auction/internal/user"
	"github.com/jackc/pgx/v5"
)

// FinishLots запускает фоновый процесс периодического завершения лотов, у которых истек срок ends_at + 10s.
func (s *Service) FinishLots(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	s.autoFinish(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.autoFinish(ctx)
		}
	}
}

// AutoFinish запускает один цикл проверки и завершения истекших лотов.
func (s *Service) AutoFinish(ctx context.Context) {
	s.autoFinish(ctx)
}

func (s *Service) autoFinish(ctx context.Context) {
	ids, err := s.repo.allFinish(ctx)
	if err != nil {
		apperr.Log(ctx, "autoFinish.allFinish", err)
		return
	}
	for _, id := range ids {
		if err = s.finishLot(ctx, id); err != nil {
			apperr.Log(ctx, "autoFinish.finishLot", err, "lot_id", id)
		}
	}
}

func (s *Service) finishLot(ctx context.Context, idLot int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("finishLot.begin: %w", apperr.NewAppError(err, 12))
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	lot, err := txRepo.getLotForFinish(ctx, idLot)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, storage.ErrNotFound) {
			// Лот уже обработан или заблокирован другим процессом
			return nil
		}
		return fmt.Errorf("finishLot.getLotForFinish: %w", err)
	}

	winnerID, amount, err := txRepo.winner(ctx, idLot)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, pgx.ErrNoRows) {
			// Ставок не было — закрываем лот без победителя
			if err = txRepo.finish(ctx, idLot, uuid.Nil(), 0); err != nil {
				return fmt.Errorf("finishLot.finishNoWinner: %w", err)
			}
			err = tx.Commit(ctx)
			if err != nil {
				return fmt.Errorf("finishLot.commit: %w", err)
			}
			return nil
		}
		return fmt.Errorf("finishLot.winner: %w", err)
	}

	// Бизнес-логика завершения лота с победителем:
	// 1. Блокируем пользователей по порядку UUID для предотвращения deadlock
	firstUID, secondUID := winnerID, lot.SellerID
	if bytes.Compare(firstUID[:], secondUID[:]) > 0 {
		firstUID, secondUID = secondUID, firstUID
	}
	u1, err := txRepo.getUserForUpdate(ctx, firstUID)
	if err != nil {
		return fmt.Errorf("finishLot.getUserForUpdate u1: %w", err)
	}
	u2, err := txRepo.getUserForUpdate(ctx, secondUID)
	if err != nil {
		return fmt.Errorf("finishLot.getUserForUpdate u2: %w", err)
	}
	var winnerUser, sellerUser user.User
	if u1.ID == winnerID {
		winnerUser, sellerUser = u1, u2
	} else {
		winnerUser, sellerUser = u2, u1
	}

	// 2. Списываем сумму и уменьшаем hold у победителя
	if err = txRepo.changeBalance(ctx, winnerID, -amount); err != nil {
		return fmt.Errorf("finishLot.changeBalance winner: %w", err)
	}
	if err = txRepo.changeHold(ctx, winnerID, -amount); err != nil {
		return fmt.Errorf("finishLot.changeHold winner: %w", err)
	}

	// 3. Зачисляем сумму продавцу
	if err = txRepo.changeBalance(ctx, lot.SellerID, amount); err != nil {
		return fmt.Errorf("finishLot.changeBalance seller: %w", err)
	}

	// 4. Передаем товар победителю
	if err = txRepo.transferItem(ctx, lot.ItemID, winnerID); err != nil {
		return fmt.Errorf("finishLot.transferItem: %w", err)
	}

	// 5. Записываем историю перемещения товара
	if err = txRepo.createHistoryItem(ctx, lot.ItemID, lot.SellerID, winnerID, lot.ID); err != nil {
		return fmt.Errorf("finishLot.createHistoryItem: %w", err)
	}

	// 6. Записываем проводки в ledger_entries
	winnerBalanceAfter := winnerUser.Balance - amount
	sellerBalanceAfter := sellerUser.Balance + amount

	if err = txRepo.createLedgerEntry(ctx, winnerID, -amount, winnerBalanceAfter, "transfer_to_seller", lot.ID); err != nil {
		return fmt.Errorf("finishLot.createLedgerEntry winner: %w", err)
	}
	if err = txRepo.createLedgerEntry(ctx, lot.SellerID, amount, sellerBalanceAfter, "payout_of_winnings", lot.ID); err != nil {
		return fmt.Errorf("finishLot.createLedgerEntry seller: %w", err)
	}

	// 7. Обновляем статус лота на Finished
	if err = txRepo.finish(ctx, lot.ID, winnerID, amount); err != nil {
		return fmt.Errorf("finishLot.finish: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("finishLot.commit: %w", err)
	}
	return nil
}

// allFinish возвращает ID активных лотов, у которых истек срок ends_at с запасом 10 секунд на плохую сеть
func (r *Repo) allFinish(ctx context.Context) ([]int64, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id 
		FROM lots 
		WHERE status = $1 AND ends_at <= now() - INTERVAL '10 seconds' 
		ORDER BY ends_at ASC`, Active)
	if err != nil {
		return nil, fmt.Errorf("finish.allFinish: %w", storage.ErrRepo(err))
	}
	ids, err := pgx.CollectRows(rows, pgx.RowToStructByName[int64])
	if err != nil {
		return nil, fmt.Errorf("finish.allFinish: %w", storage.ErrRepo(err))
	}
	return ids, nil
}

func (r *Repo) getLotForFinish(ctx context.Context, id int64) (Lot, error) {
	row, err := r.db.Query(ctx, `
		SELECT id, seller_id, item_id, status, winner_id, start_amount, min_step, finish_amount, title, description, created_at, started_at, ends_at, finished_at 
		FROM lots 
		WHERE id = $1 AND status = $2 AND ends_at <= now() - INTERVAL '10 seconds' 
		FOR UPDATE SKIP LOCKED`, id, Active)
	if err != nil {
		return Lot{}, fmt.Errorf("finish.getLotForFinish: %w", storage.ErrRepo(err))
	}
	lot, err := pgx.CollectExactlyOneRow(row, pgx.RowToStructByName[Lot])
	if err != nil {
		return Lot{}, fmt.Errorf("finish.getLotForFinish: %w", storage.ErrRepo(err))
	}
	return lot, nil
}

func (r *Repo) winner(ctx context.Context, idLot int64) (uuid.UUID, int, error) {
	var uID uuid.UUID
	var amount int
	err := r.db.QueryRow(ctx, `
		SELECT user_id, amount 
		FROM bids 
		WHERE lot_id = $1 
		ORDER BY id DESC 
		LIMIT 1`, idLot).Scan(&uID, &amount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil(), 0, storage.ErrNotFound
		}
		return uuid.Nil(), 0, storage.ErrRepo(err)
	}
	return uID, amount, nil
}

// changeBalance изменяет баланс пользователя на changeBalance (+увеличить, -уменьшить)
func (r *Repo) changeBalance(ctx context.Context, uid uuid.UUID, changeBalance int) error {
	res, err := r.db.Exec(ctx, "UPDATE users SET balance = balance + $1 WHERE id = $2 AND deleted_at IS NULL", changeBalance, uid)
	if err != nil {
		return fmt.Errorf("lotRepo.changeBalance: %w", storage.ErrRepo(err))
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("lotRepo.changeBalance: %w", storage.ErrNotFound)
	}
	return nil
}

func (r *Repo) transferItem(ctx context.Context, itemID int64, newOwnerID uuid.UUID) error {
	res, err := r.db.Exec(ctx, "UPDATE items SET owner_id = $1 WHERE id = $2", newOwnerID, itemID)
	if err != nil {
		return fmt.Errorf("lotRepo.transferItem: %w", storage.ErrRepo(err))
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("lotRepo.transferItem: %w", storage.ErrNotFound)
	}
	return nil
}

func (r *Repo) createHistoryItem(ctx context.Context, itemID int64, oldUser, newUser uuid.UUID, lotID int64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO history_item (item_id, old_user, new_user, lot_id, description) 
		VALUES ($1, $2, $3, $4, $5)`,
		itemID, oldUser, newUser, lotID, "Победа в аукционе")
	if err != nil {
		return fmt.Errorf("lotRepo.createHistoryItem: %w", storage.ErrRepo(err))
	}
	return nil
}

func (r *Repo) createLedgerEntry(ctx context.Context, uid uuid.UUID, amount, balanceAfter int, entryType string, lotID int64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ledger_entries (user_id, amount, balance_after, type, lot_id) 
		VALUES ($1, $2, $3, $4, $5)`,
		uid, amount, balanceAfter, entryType, lotID)
	if err != nil {
		return fmt.Errorf("lotRepo.createLedgerEntry: %w", storage.ErrRepo(err))
	}
	return nil
}

func (r *Repo) finish(ctx context.Context, id int64, winnerID uuid.UUID, finishAmount int) error {
	var query string
	var args []any
	if winnerID == uuid.Nil() {
		query = "UPDATE lots SET status = $1, winner_id = NULL, finish_amount = NULL, finished_at = now() WHERE id = $2 AND status = $3"
		args = []any{Finished, id, Active}
	} else {
		query = "UPDATE lots SET status = $1, winner_id = $2, finish_amount = $3, finished_at = now() WHERE id = $4 AND status = $5"
		args = []any{Finished, winnerID, finishAmount, id, Active}
	}
	res, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("lotRepo.finish: %w", storage.ErrRepo(err))
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("lotRepo.finish: %w", storage.ErrNotFound)
	}
	return nil
}

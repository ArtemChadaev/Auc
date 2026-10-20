package lot

import (
	"context"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/storage"
	"github.com/jackc/pgx/v5"
)

//TODO: доработать надо передавать item победителю, а так же убирать hold победителя на -.

func (s *Service) FinishLots(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			go s.autoFinish(ctx)
		}
	}
}

func (s *Service) autoFinish(ctx context.Context) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		apperr.Log(ctx, "begin tx", apperr.NewAppError(err, 12))
		return
	}
	defer tx.Rollback(ctx)
	txRepo := s.repo.WithTx(tx)

	ids, err := txRepo.allFinish(ctx)
	if err != nil {
		apperr.Log(ctx, "autoFinish.allFinish", err)
		return
	}
	for _, id := range ids {
		uID, amount, err := txRepo.winner(ctx, id)
		if err != nil {
			apperr.Log(ctx, "autoFinish.winner", err)
			continue
		}
		if err = txRepo.finish(ctx, id, uID, amount); err != nil {
			apperr.Log(ctx, "autoFinish.finish", err)
		}
	}
	_ = tx.Commit(ctx)
}

func (r *Repo) allFinish(ctx context.Context) ([]int64, error) {
	rows, err := r.db.Query(ctx, "SELECT id FROM lots WHERE status = $1 AND ends_at > now() + INTERVAL '30 seconds'", Active)
	if err != nil {
		return nil, storage.ErrRepo(err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowToStructByName[int64])
	if err != nil {
		return nil, storage.ErrRepo(err)
	}
	return ids, nil
}

func (r *Repo) winner(ctx context.Context, idLot int64) (uuid.UUID, int, error) {
	var uID uuid.UUID
	var amount int
	err := r.db.QueryRow(ctx, "SELECT user_id, amount FROM bids WHERE id = $1", idLot).Scan(&uID, &amount)
	if err != nil {
		return [16]byte{}, 0, apperr.NewAppError(err, 8)
	}
	return uID, amount, nil
}

func (r *Repo) finish(ctx context.Context, id int64, winnerID uuid.UUID, finishAmount int) error {
	_, err := r.db.Exec(ctx, "UPDATE lots SET status = $1, winner_id = $2, finish_amount = $3 WHERE id = $4 AND status = $5", Finished, winnerID, finishAmount, id, Active)
	if err != nil {
		return apperr.NewAppError(err, 8)
	}
	return nil
}

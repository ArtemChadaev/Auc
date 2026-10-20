package lot

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo *Repo
	pool *pgxpool.Pool
}

func NewService(repo *Repo, pool *pgxpool.Pool) *Service {
	return &Service{repo: repo, pool: pool}
}

func (s *Service) list(ctx context.Context) ([]Lot, error) {
	lots, err := s.repo.list(ctx)
	if err != nil {
		return nil, fmt.Errorf("lotService.list: %w", err)
	}
	return lots, nil
}

func (s *Service) getByID(ctx context.Context, id int64) (Lot, error) {
	if id <= 0 {
		return Lot{}, fmt.Errorf("lotService.getByID: %w", ErrInvalidID)
	}
	lot, err := s.repo.getById(ctx, id)
	if err != nil {
		return Lot{}, fmt.Errorf("lotService.getByID: %w", err)
	}
	return lot, nil
}

func (s *Service) getMyLots(ctx context.Context, sellerID uuid.UUID) ([]Lot, error) {
	lots, err := s.repo.getMy(ctx, sellerID)
	if err != nil {
		return nil, fmt.Errorf("lotService.getMyLots: %w", err)
	}
	return lots, nil
}

func (s *Service) createLot(ctx context.Context, sellerID uuid.UUID, req reqNewLot) (Lot, error) {
	lot, err := s.repo.create(
		ctx,
		sellerID,
		req.ItemID,
		req.Status,
		req.StartAmount,
		req.MinStep,
		req.Title,
		req.Description,
		req.StartedAt,
		req.EndedAt,
	)
	if err != nil {
		return Lot{}, fmt.Errorf("lotService.createLot: %w", err)
	}
	return lot, nil
}

func (s *Service) updateLot(ctx context.Context, id int64, sellerID uuid.UUID, req reqNewLot) (Lot, error) {
	if id <= 0 {
		return Lot{}, fmt.Errorf("lotService.updateLot: %w", ErrInvalidID)
	}
	lot, err := s.repo.updateFromSeller(
		ctx,
		id,
		sellerID,
		req.ItemID,
		req.Status,
		req.StartAmount,
		req.MinStep,
		req.Title,
		req.Description,
		req.StartedAt,
		req.EndedAt,
	)
	if err != nil {
		return Lot{}, fmt.Errorf("lotService.updateLot: %w", err)
	}
	return lot, nil
}

func (s *Service) cancelLot(ctx context.Context, id int64, sellerID uuid.UUID) error {
	if id <= 0 {
		return fmt.Errorf("lotService.cancelLot: %w", ErrInvalidID)
	}
	if err := s.repo.cancel(ctx, id, sellerID); err != nil {
		return fmt.Errorf("lotService.cancelLot: %w", err)
	}
	return nil
}

func (s *Service) activateLot(ctx context.Context, id int64, sellerID uuid.UUID, startedAt, endedAt *time.Time) (Lot, error) {
	if id <= 0 {
		return Lot{}, fmt.Errorf("lotService.activateLot: %w", ErrInvalidID)
	}
	if startedAt != nil && endedAt != nil && !endedAt.After(*startedAt) {
		return Lot{}, fmt.Errorf("lotService.activateLot: %w", ErrInvalidReqBody)
	}
	lot, err := s.repo.activate(ctx, id, sellerID, startedAt, endedAt)
	if err != nil {
		return Lot{}, fmt.Errorf("lotService.activateLot: %w", err)
	}
	return lot, nil
}

func (s *Service) bid(ctx context.Context, idLot int64, uid, idempotencyKey uuid.UUID, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("lotService.bid: %w", ErrInvalidReqBody)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("lotService.bid: %w", err)
	}
	defer tx.Rollback(ctx)
	txRepo := s.repo.WithTx(tx)

	lot, err := txRepo.getByIdForUpdate(ctx, idLot)
	if err != nil {
		return fmt.Errorf("lotService.bid: %w", err)
	}
	bid, err := txRepo.getLastBid(ctx, idLot)
	if err != nil {
		return fmt.Errorf("lotService.bid: %w", err)
	}
	if bid.IdempotencyKey == idempotencyKey {
		return nil
	}
	if amount < bid.Amount+lot.MinStep {
		//TODO: Сделать ошибки
		return errors.New("lotService.bid: amount is too small")
	}
	user, err := txRepo.getUserForUpdate(ctx, uid)
	if err != nil {
		return fmt.Errorf("lotService.bid: %w", err)
	}

	if uid == bid.UserId {
		if amount < (user.Balance - user.Hold + bid.Amount) {
			return errors.New("lotService.bid: amount is too small")
		}
		if err = txRepo.createBid(ctx, idLot, uid, idempotencyKey, amount); err != nil {
			return fmt.Errorf("lotService.bid: %w", err)
		}
		if err = txRepo.changeHold(ctx, uid, amount-bid.Amount); err != nil {
			return fmt.Errorf("lotService.bid: %w", err)
		}
		return nil
	}
	if amount < user.Balance-user.Hold {
		return errors.New("lotService.bid: amount is too small")
	}
	if err = txRepo.createBid(ctx, idLot, uid, idempotencyKey, amount); err != nil {
		return fmt.Errorf("lotService.bid: %w", err)
	}
	if err = txRepo.changeHold(ctx, uid, amount); err != nil {
		return fmt.Errorf("lotService.bid: %w", err)
	}
	if err = txRepo.changeHold(ctx, bid.UserId, -bid.Amount); err != nil {
		return fmt.Errorf("lotService.bid: %w", err)
	}

	_ = tx.Commit(ctx)
	return nil
}

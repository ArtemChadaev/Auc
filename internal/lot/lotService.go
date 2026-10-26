package lot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/storage"
	"github.com/ArtemChadaev/Auction/internal/user"
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

func (s *Service) approve(ctx context.Context, id int64) error {
	if err := s.repo.approve(ctx, id); err != nil {
		return fmt.Errorf("lotService.approve: %w", err)
	}
	return nil
}

/*
АНАЛИЗ SQL-КОДА И 3 ПРИМЕРА ВЫПОЛНЕНИЯ:

================================================================================
Пример 1. Первая ставка на лот (ставок еще нет, isFirst == true, storage.ErrNotFound)
Лот id=1, start_amount=1000, min_step=100.
Пользователь uid='0192e4a1-0001-7000-8000-000000000001' ставит amount=1000.

 1. Начало транзакции:
    BEGIN;

 2. Блокировка лота по ID:
    SELECT * FROM lots WHERE id = 1 AND status = 'active' FOR UPDATE;

 3. Запрос последней ставки (возвращает 0 строк -> storage.ErrNotFound):
    SELECT * FROM bids WHERE lot_id = 1 ORDER BY id desc LIMIT 1;

 4. Блокировка и получение данных пользователя:
    SELECT id, name, avatar_id, email, deleted_at, balance, hold, created_at
    FROM users WHERE id = '0192e4a1-0001-7000-8000-000000000001' AND deleted_at IS NULL FOR UPDATE;

 5. Вставка новой ставки:
    INSERT INTO bids (lot_id, user_id, amount) VALUES (1, '0192e4a1-0001-7000-8000-000000000001', 1000);

 6. Заморозка средств (hold += 1000):
    UPDATE users SET hold = hold + 1000 WHERE id = '0192e4a1-0001-7000-8000-000000000001' AND deleted_at IS NULL;

 7. Фиксация транзакции:
    COMMIT;

================================================================================
Пример 2. Перебитие ставки другим пользователем (isFirst == false, защита от deadlock)
Текущая ставка: 1000 от '0192e4a1-0001-7000-8000-000000000001'.
Новый участник uid='0192e4a1-0002-7000-8000-000000000002' ставит amount=1200.

 1. Начало транзакции:
    BEGIN;

 2. Блокировка лота:
    SELECT * FROM lots WHERE id = 1 AND status = 'active' FOR UPDATE;

 3. Получение последней ставки (находит ставку на 1000 от пользователя 1):
    SELECT * FROM bids WHERE lot_id = 1 ORDER BY id desc LIMIT 1;

 4. Упорядочивание firstUID и secondUID (по bytes.Compare) для предотвращения deadlock:
    firstUID = '0192e4a1-0001-7000-8000-000000000001', secondUID = '0192e4a1-0002-7000-8000-000000000002'

 5. Блокировка первого пользователя по порядку:
    SELECT id, name, avatar_id, email, deleted_at, balance, hold, created_at
    FROM users WHERE id = '0192e4a1-0001-7000-8000-000000000001' AND deleted_at IS NULL FOR UPDATE;

 6. Блокировка второго пользователя по порядку:
    SELECT id, name, avatar_id, email, deleted_at, balance, hold, created_at
    FROM users WHERE id = '0192e4a1-0002-7000-8000-000000000002' AND deleted_at IS NULL FOR UPDATE;

 7. Создание новой ставки:
    INSERT INTO bids (lot_id, user_id, amount) VALUES (1, '0192e4a1-0002-7000-8000-000000000002', 1200);

 8. Увеличение hold у нового лидера (+1200):
    UPDATE users SET hold = hold + 1200 WHERE id = '0192e4a1-0002-7000-8000-000000000002' AND deleted_at IS NULL;

 9. Разморозка hold у предыдущего лидера (-1000):
    UPDATE users SET hold = hold + -1000 WHERE id = '0192e4a1-0001-7000-8000-000000000001' AND deleted_at IS NULL;

 10. Фиксация транзакции:
    COMMIT;
*/
func (s *Service) bid(ctx context.Context, idLot int64, uid uuid.UUID, amount int) error {
	// Первоначальная проверка и начало транзакции
	if amount <= 0 {
		return fmt.Errorf("lotService.bid: %w", ErrInvalidReqBody)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("lotService.bid: %w", err)
	}
	defer tx.Rollback(ctx)
	txRepo := s.repo.WithTx(tx)

	// Получение лота с for update (Другие ставки на этот лот ждут завершения)
	lot, err := txRepo.getByIdForUpdate(ctx, idLot)
	if err != nil {
		return fmt.Errorf("lotService.bid: %w", err)
	}
	if lot.SellerID == uid {
		return fmt.Errorf("lotService.bid: %w", ErrForbidden)
	}

	bid, err := txRepo.getLastBid(ctx, idLot)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("lotService.bid: %w", err)
	}
	if err == nil && uid == bid.UserId {
		return fmt.Errorf("lotService.bid: %w", ErrLider)
	}

	isFirst := false
	if errors.Is(err, storage.ErrNotFound) {
		isFirst = true
	}
	var currentUser user.User
	if isFirst {
		if amount < lot.StartAmount {
			return fmt.Errorf("lotService.bid: %w", ErrAmountSmall)
		}
		currentUser, err = txRepo.getUserForUpdate(ctx, uid)
		if err != nil {
			return fmt.Errorf("lotService.bid: %w", err)
		}
	} else {
		// У нас 2 пользователя чтобы не было deadlock всегда по порядку делаем select for update
		if amount < bid.Amount+lot.MinStep {
			return fmt.Errorf("lotService.bid: %w", ErrAmountSmall)
		}
		firstUID, secondUID := uid, bid.UserId
		if bytes.Compare(firstUID[:], secondUID[:]) > 0 {
			firstUID, secondUID = secondUID, firstUID
		}
		var u1, u2 user.User
		// 1. Блокируем первого по порядку
		u1, err = txRepo.getUserForUpdate(ctx, firstUID)
		if err != nil {
			return fmt.Errorf("lotService.bid: %w", err)
		}

		// 2. Блокируем второго по порядку
		u2, err = txRepo.getUserForUpdate(ctx, secondUID)
		if err != nil {
			return fmt.Errorf("lotService.bid: %w", err)
		}

		// Определяем, кто из них текущий пользователь, чтобы проверить баланс
		if u1.ID == uid {
			currentUser = u1
		} else {
			currentUser = u2
		}
	}

	if amount > currentUser.Balance-currentUser.Hold {
		return fmt.Errorf("lotService.bid: %w", ErrInsufficientBalance)
	}
	if err = txRepo.createBid(ctx, idLot, currentUser.ID, amount); err != nil {
		return fmt.Errorf("lotService.bid: %w", err)
	}
	if err = txRepo.changeHold(ctx, currentUser.ID, amount); err != nil {
		return fmt.Errorf("lotService.bid: %w", err)
	}

	if !isFirst {
		if err = txRepo.changeHold(ctx, bid.UserId, -bid.Amount); err != nil {
			return fmt.Errorf("lotService.bid: %w", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("lotService.bid: %w", err)
	}
	return nil
}

func (s *Service) getLotBids(ctx context.Context, idLot int64) ([]Bid, error) {
	bids, err := s.repo.getLotBids(ctx, idLot)
	if err != nil {
		return nil, fmt.Errorf("lotService.getLotBids: %w", err)
	}
	return bids, nil
}

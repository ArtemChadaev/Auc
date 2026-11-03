package item

import (
	"context"
	"fmt"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/storageS3"
	"github.com/ArtemChadaev/Auction/cmd/valkey"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool      *pgxpool.Pool
	s3        *storageS3.Client
	repo      *Repo
	valkey    *valkey.Client
	uploadSem chan struct{}
}

func NewService(pool *pgxpool.Pool, s3 *storageS3.Client, repo *Repo, valkeyClient *valkey.Client) *Service {
	return &Service{
		pool:      pool,
		s3:        s3,
		repo:      repo,
		valkey:    valkeyClient,
		uploadSem: make(chan struct{}, 5), // N = 5 параллельных задач
	}
}

// getItem получение предмета по ID
func (s *Service) getItem(ctx context.Context, id int64) (Item, error) {
	item, err := s.repo.getItem(ctx, id)
	if err != nil {
		return Item{}, fmt.Errorf("itemService.getItem: %w", err)
	}
	return item, nil
}

// getMyItems получение списка предметов текущего пользователя
func (s *Service) getMyItems(ctx context.Context, uid uuid.UUID) ([]Item, error) {
	items, err := s.repo.getItemsByOwner(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("itemService.getMyItems: %w", err)
	}
	return items, nil
}

// newOwner смена владельца
func (s *Service) newOwner(ctx context.Context, uid, newOwner uuid.UUID, itemID int64, lotID *int64, description *string) error {
	if itemID <= 0 || newOwner == uuid.Nil() {
		return fmt.Errorf("itemService.newOwner: invalid itemID or newOwner")
	}
	if uid == newOwner {
		return fmt.Errorf("itemService.newOwner: new owner is same as current owner")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("itemService.newOwner: %w", err)
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.withTx(tx)

	oldOwner, err := txRepo.getOwnerForUpdate(ctx, itemID)
	if err != nil {
		return fmt.Errorf("itemService.newOwner: %w", err)
	}
	if oldOwner != uid {
		return fmt.Errorf("itemService.newOwner: %w", apperr.ErrForbidden)
	}

	// Проверяем, что предмет не находится в активном или находящемся на модерации лоте
	var activeOrPendingExists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM lots WHERE item_id = $1 AND status IN ('active', 'pending_review'))", itemID).Scan(&activeOrPendingExists)
	if err != nil {
		return fmt.Errorf("itemService.newOwner.checkActiveLot: %w", err)
	}
	if activeOrPendingExists {
		return fmt.Errorf("itemService.newOwner: cannot transfer item that is in an active or pending review lot")
	}

	err = txRepo.newOwner(ctx, newOwner, itemID)
	if err != nil {
		return fmt.Errorf("itemService.newOwner: %w", err)
	}

	err = txRepo.newHistory(ctx, itemID, oldOwner, newOwner, lotID, description)
	if err != nil {
		return fmt.Errorf("itemService.newOwner: %w", err)
	}

	// После передачи удаляем все лоты старого владельца с этим предметом со статусом draft и approved
	if err = txRepo.deleteDraftAndApprovedLots(ctx, oldOwner, itemID); err != nil {
		return fmt.Errorf("itemService.newOwner.deleteLots: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("itemService.newOwner: %w", apperr.NewAppError(err, 8))
	}
	return nil
}

// giftItem передача предмета в подарок
func (s *Service) giftItem(ctx context.Context, uid, newOwner uuid.UUID, itemID int64, description string) error {
	return s.newOwner(ctx, uid, newOwner, itemID, nil, &description)
}

// NewOwnerWhoWinLot для экспорта в пакете lot
func (s *Service) NewOwnerWhoWinLot(ctx context.Context, uid, newOwner uuid.UUID, itemID, lotID int64) error {
	return s.newOwner(ctx, uid, newOwner, itemID, &lotID, nil)
}

// hasHash проверка наличия хеша в БД
func (s *Service) hasHash(ctx context.Context, hash string) (bool, error) {
	return s.repo.hasHash(ctx, hash)
}

// AdminTransferItem передает предмет новому владельцу администратором с добавлением истории перемещения
func (s *Service) AdminTransferItem(ctx context.Context, itemID int64, newOwner uuid.UUID, description string) error {
	if itemID <= 0 || newOwner == uuid.Nil() {
		return fmt.Errorf("itemService.AdminTransferItem: invalid itemID or newOwner")
	}
	if description == "" {
		description = "Передача администратором"
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("itemService.AdminTransferItem.begin: %w", err)
	}
	defer tx.Rollback(ctx)
	txRepo := s.repo.withTx(tx)

	// Проверяем, что предмет не находится в активном, одобренном или находящемся на модерации лоте
	var lotExists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM lots WHERE item_id = $1 AND status IN ('active', 'approved', 'pending_review'))", itemID).Scan(&lotExists)
	if err != nil {
		return fmt.Errorf("itemService.AdminTransferItem.checkLots: %w", err)
	}
	if lotExists {
		return fmt.Errorf("itemService.AdminTransferItem: cannot transfer item that is in an active, approved, or pending review lot")
	}

	oldOwner, err := txRepo.getOwnerForUpdate(ctx, itemID)
	if err != nil {
		return fmt.Errorf("itemService.AdminTransferItem.getOwnerForUpdate: %w", err)
	}
	if oldOwner == newOwner {
		return fmt.Errorf("itemService.AdminTransferItem: new owner is same as current owner")
	}

	if err = txRepo.newOwner(ctx, newOwner, itemID); err != nil {
		return fmt.Errorf("itemService.AdminTransferItem.newOwner: %w", err)
	}

	if err = txRepo.newHistory(ctx, itemID, oldOwner, newOwner, nil, &description); err != nil {
		return fmt.Errorf("itemService.AdminTransferItem.newHistory: %w", err)
	}

	// После передачи удаляем все лоты старого владельца с этим предметом со статусом draft и approved
	if err = txRepo.deleteDraftAndApprovedLots(ctx, oldOwner, itemID); err != nil {
		return fmt.Errorf("itemService.AdminTransferItem.deleteLots: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("itemService.AdminTransferItem.commit: %w", apperr.NewAppError(err, 8))
	}
	return nil
}

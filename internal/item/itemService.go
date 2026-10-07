package item

import (
	"context"
	"encoding/json"
	"fmt"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/storageS3"
	"github.com/ArtemChadaev/Auction/cmd/valkey"
	"github.com/ArtemChadaev/Auction/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repo interface {
	withTx(tx storage.DBTX) *Repo
	createS3(ctx context.Context, id uuid.UUID, hash string) error
	createItem(ctx context.Context, uid uuid.UUID, s3ID uuid.UUID, iType itemType, metadata json.RawMessage) (Item, error)
	getOwner(ctx context.Context, id int64) (uuid.UUID, error)
	getOwnerForUpdate(ctx context.Context, id int64) (uuid.UUID, error)
	newOwner(ctx context.Context, uid uuid.UUID, id int64) error
	getItem(ctx context.Context, id int64) (Item, error)
	delete(ctx context.Context, id int64) error
	hasHash(ctx context.Context, hash string) (bool, error)

	newHistory(ctx context.Context, itemID int64, oldUser, newUser uuid.UUID, lotID *int64, description *string) error
	listHistoryByItemID(ctx context.Context, itemID int64) ([]HistoryItem, error)
	listHistoryByUserID(ctx context.Context, uid uuid.UUID) ([]HistoryItem, error)
}

type Service struct {
	pool      *pgxpool.Pool
	s3        *storageS3.Client
	repo      repo
	valkey    *valkey.Client
	uploadSem chan struct{}
}

func NewService(pool *pgxpool.Pool, s3 *storageS3.Client, repo repo, valkeyClient *valkey.Client) *Service {
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

// newOwner смена владельца
func (s *Service) newOwner(ctx context.Context, uid, newOwner uuid.UUID, itemID int64, lotID *int64, description *string) error {
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

	err = txRepo.newOwner(ctx, newOwner, itemID)
	if err != nil {
		return fmt.Errorf("itemService.newOwner: %w", err)
	}

	err = txRepo.newHistory(ctx, itemID, oldOwner, newOwner, lotID, description)
	if err != nil {
		return fmt.Errorf("itemService.newOwner: %w", err)
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

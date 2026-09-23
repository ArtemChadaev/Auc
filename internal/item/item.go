package item

import (
	"context"
	"encoding/json"
	"fmt"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/storageS3"
	"github.com/ArtemChadaev/Auction/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repo interface {
	withTx(tx storage.DBTX) *Repo
	create(ctx context.Context, uid uuid.UUID, key string, iType Type, mimeType string, size int64, metadata json.RawMessage) error
	getOwner(ctx context.Context, id int64) (uuid.UUID, error)
	getOwnerForUpdate(ctx context.Context, id int64) (uuid.UUID, error)
	newOwner(ctx context.Context, uid uuid.UUID, id int64) error
	getItem(ctx context.Context, id int64) (Item, error)
	delete(ctx context.Context, id int64) error

	newHistory(ctx context.Context, itemID int64, oldUser, newUser uuid.UUID, lotID *int64, description *string) error
	listHistoryByItemID(ctx context.Context, itemID int64) ([]HistoryItem, error)
	listHistoryByUserID(ctx context.Context, uid uuid.UUID) ([]HistoryItem, error)
}

type Service struct {
	pool *pgxpool.Pool
	s3   *storageS3.Client
	repo repo
}

func NewService(pool *pgxpool.Pool, s3 *storageS3.Client, repo repo) *Service {
	return &Service{
		pool: pool,
		s3:   s3,
		repo: repo,
	}
}

func (s *Service) create(ctx context.Context, uid uuid.UUID, iType Type, mimeType string, size int64, nameFile string, data []byte) {

}
func (s *Service) newOwner(ctx context.Context, uid, newOwner uuid.UUID, itemID int64, description string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("item.newOwner(%w): %w", apperr.ErrError, err)
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.withTx(tx)

	oldOwner, err := txRepo.getOwnerForUpdate(ctx, itemID)
	if err != nil {
		return fmt.Errorf("item.newOwner: %v", err)
	}
	if oldOwner != uid {
		return fmt.Errorf("item.newOwner(%w): %v", apperr.ErrDebug, apperr.ErrForbidden)
	}

	err = txRepo.newOwner(ctx, newOwner, itemID)
	if err != nil {
		return fmt.Errorf("item.newOwner: %v", err)
	}

	err = txRepo.newHistory(ctx, itemID, oldOwner, newOwner, nil, &description)
	if err != nil {
		return fmt.Errorf("item.newOwner: %v", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("item.newOwner(%w): %v", apperr.ErrError, err)
	}
	return nil
}

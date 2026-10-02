package item

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"time"
	"uuid"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	"golang.org/x/sync/errgroup"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/storageS3"
	"github.com/ArtemChadaev/Auction/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repo interface {
	withTx(tx storage.DBTX) *Repo
	create(ctx context.Context, uid uuid.UUID, key, hash string, iType itemType, metadata json.RawMessage) error
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

func (s *Service) create(ctx context.Context, iType itemType, mType string, hmData headerMetadata, file io.Reader, uid uuid.UUID) error {
	// Создаём контекст для отмене при ошибке
	ctxWithCause, cancel := context.WithTimeoutCause(ctx, time.Minute*10, apperr.ErrTimeout)
	defer cancel()
	g, gCtx := errgroup.WithContext(ctxWithCause)

	// Делаем 2 дополнительных чтеца для хеша и metadata, главный s3
	prHash, pwHash := io.Pipe()
	prMData, pwMData := io.Pipe()
	tee := io.TeeReader(file, io.MultiWriter(pwMData, pwHash))

	// Пишется err error для defer (Правильно закрытие при любом выходе из функции, с или без ошибки)
	// Функция создания хеша файла (ля проверки копий, если был -> ошибка только 1 уникальный файл)
	hasher := sha256.New()
	g.Go(func() (err error) {
		defer func() {
			_ = prMData.CloseWithError(err)
		}()

		if _, err = io.Copy(hasher, prHash); err != nil {
			slog.DebugContext(gCtx, "Failed to copy file to hasher", slog.Any("error", err))
			if err != nil {
				return fmt.Errorf("item.create: %w", apperr.NewAppError(err, -4))
			}
		}
		return
	})

	var metadata json.RawMessage
	// Функция создания metadata
	g.Go(func() (err error) {
		defer func() {
			_ = prMData.CloseWithError(err)
		}()
		switch iType {
		case Image:
			image, err := getImageMetadata(gCtx, hmData, mType, prMData)
			if err != nil {
				return fmt.Errorf("item.create: %w", err)
			}
			metadata, err = json.Marshal(image)
			if err != nil {
				return fmt.Errorf("item.create: %w", apperr.NewAppError(err, 4))
			}
			break
		case Model3D:
			model3D, err := getModel3DMetadata(gCtx, hmData, prMData)
			if err != nil {
				return fmt.Errorf("item.create: %w", err)
			}
			metadata, err = json.Marshal(model3D)
			if err != nil {
				return fmt.Errorf("item.create: %w", apperr.NewAppError(err, 4))
			}
			break
		case Audio:
			audio, err := getAudioMetadata(gCtx, hmData, prMData)
			if err != nil {
				return fmt.Errorf("item.create: %w", err)
			}
			metadata, err = json.Marshal(audio)
			if err != nil {
				return fmt.Errorf("item.create: %w", apperr.NewAppError(err, 4))
			}
			break
		case Video:
			video, err := getVideoMetadata(gCtx, hmData, prMData)
			if err != nil {
				return fmt.Errorf("item.create: %w", err)
			}
			metadata, err = json.Marshal(video)
			if err != nil {
				return fmt.Errorf("item.create: %w", apperr.NewAppError(err, 4))
			}
			break
		case Document:
			doc, err := getDocumentMetadata(gCtx, hmData, mType, prMData)
			if err != nil {
				return fmt.Errorf("item.create: %w", apperr.NewAppError(err, 4))
			}
			metadata, err = json.Marshal(doc)
			if err != nil {
				return fmt.Errorf("item.create: %w", apperr.NewAppError(err, 4))
			}
			break
		case Archive:
			archive, err := getArchiveMetadata(gCtx, hmData, prMData)
			if err != nil {
				return fmt.Errorf("item.create: %w", err)
			}
			metadata, err = json.Marshal(archive)
			if err != nil {
				return fmt.Errorf("item.create: %w", apperr.NewAppError(err, 4))
			}
			break
		default:
			return errors.New("not an image")
		}
		_, _ = io.Copy(io.Discard, prMData)
		return
	})

	// S3
	key := uuid.NewV7().String()
	g.Go(func() error {
		defer func() {
			_ = pwHash.CloseWithError(context.Cause(gCtx))
			_ = pwMData.CloseWithError(context.Cause(gCtx))
		}()
		if err := s.s3.Upload(gCtx, key, tee, mType); err != nil {
			cancel()
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	hashString := hex.EncodeToString(hasher.Sum(nil))
	if has, err := s.repo.hasHash(ctx, hashString); has {
		if err != nil {
			return fmt.Errorf("item.create: %w", err)
		}
		//TODO: Сделать функцию удаление из s3 по key и отдельно функцию вывод тогда данных для нахождения предмета у другого пользователя (id хз)
	}

	if err := s.repo.create(ctx, uid, key, hashString, iType, metadata); err != nil {
		//TODO: функцию удаления из s3 тогда
		return fmt.Errorf("item.create: %w", err)
	}

	return nil
}

func (s *Service) newOwner(ctx context.Context, uid, newOwner uuid.UUID, itemID int64, description string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("item.newOwner: %w", err)
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.withTx(tx)

	oldOwner, err := txRepo.getOwnerForUpdate(ctx, itemID)
	if err != nil {
		return fmt.Errorf("item.newOwner: %w", err)
	}
	if oldOwner != uid {
		return fmt.Errorf("item.newOwner: %w", apperr.ErrForbidden)
	}

	err = txRepo.newOwner(ctx, newOwner, itemID)
	if err != nil {
		return fmt.Errorf("item.newOwner: %w", err)
	}

	err = txRepo.newHistory(ctx, itemID, oldOwner, newOwner, nil, &description)
	if err != nil {
		return fmt.Errorf("item.newOwner: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("item.newOwner: %w", apperr.NewAppError(err, 8))
	}
	return nil
}

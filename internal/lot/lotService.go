package lot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/mimeType"
	"github.com/ArtemChadaev/Auction/cmd/storage"
	"github.com/ArtemChadaev/Auction/cmd/storageS3"
	"github.com/ArtemChadaev/Auction/cmd/valkey"
	"github.com/ArtemChadaev/Auction/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	repo     *Repo
	pool     *pgxpool.Pool
	photosS3 *storageS3.Client
	valkey   *valkey.Client
}

func NewService(repo *Repo, pool *pgxpool.Pool, photosS3 *storageS3.Client, valkey *valkey.Client) *Service {
	return &Service{repo: repo, pool: pool, photosS3: photosS3, valkey: valkey}
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

func sanitizePhotoUUIDs(desc *Description) {
	if desc == nil {
		return
	}
	raw := desc.GetImages()
	if len(raw) == 0 {
		desc.Images = nil
		desc.PhotoURL = nil
		return
	}
	var arrImages [10]string
	var arrPhotos [10]string
	count := 0
	for _, p := range raw {
		if u, ok := ExtractUUID(p); ok {
			if count < 10 {
				arrImages[count] = u
				arrPhotos[count] = fmt.Sprintf("%s/%s", PublicPhotosBucket, u)
				count++
			}
		}
	}
	if count > 0 {
		desc.Images = &arrImages
		desc.PhotoURL = &arrPhotos
	} else {
		desc.Images = nil
		desc.PhotoURL = nil
	}
}

func (s *Service) markPhotosAttached(ctx context.Context, desc Description) {
	if s.valkey == nil || s.valkey.Raw() == nil {
		return
	}
	photos := desc.GetImages()
	if len(photos) == 0 {
		return
	}
	var members []any
	for _, p := range photos {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if idx := strings.LastIndex(p, "/"); idx != -1 {
			p = p[idx+1:]
		}
		p = strings.TrimSpace(p)
		if p != "" {
			members = append(members, p)
		}
	}
	if len(members) > 0 {
		_ = s.valkey.Raw().ZRem(ctx, "photos:unattached", members...).Err()
	}
}

func (s *Service) createLot(ctx context.Context, sellerID uuid.UUID, req reqNewLot) (Lot, error) {
	sanitizePhotoUUIDs(&req.Description)
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
	s.markPhotosAttached(ctx, req.Description)
	return lot, nil
}

func (s *Service) updateLot(ctx context.Context, id int64, sellerID uuid.UUID, req reqNewLot) (Lot, error) {
	if id <= 0 {
		return Lot{}, fmt.Errorf("lotService.updateLot: %w", ErrInvalidID)
	}
	sanitizePhotoUUIDs(&req.Description)
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
	s.markPhotosAttached(ctx, req.Description)
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

func (s *Service) uploadPhoto(ctx context.Context, uid uuid.UUID, r io.Reader, filename string, size int64) (string, string, string, error) {
	if size > mimeType.MaxPhotoSize {
		return "", "", "", mimeType.ErrFileSizeExceeded
	}

	// Защита от спама и перегрузки: лимит на загрузку фотографий (не более 30 фото в час на пользователя)
	if s.valkey != nil && s.valkey.Raw() != nil && uid != uuid.Nil() {
		limitKey := fmt.Sprintf("ratelimit:photos:upload:%s", uid.String())
		count, err := s.valkey.Raw().Incr(ctx, limitKey).Result()
		if err == nil {
			if count == 1 {
				_ = s.valkey.Raw().Expire(ctx, limitKey, time.Hour).Err()
			}
			if count > 30 {
				return "", "", "", ErrPhotoUploadLimitReached
			}
		}
	}

	fileBytes, err := io.ReadAll(io.LimitReader(r, mimeType.MaxPhotoSize+1))
	if err != nil {
		return "", "", "", fmt.Errorf("lotService.uploadPhoto: read file: %w", err)
	}
	if int64(len(fileBytes)) > mimeType.MaxPhotoSize {
		return "", "", "", mimeType.ErrFileSizeExceeded
	}

	mtype, err := mimeType.ValidatePhoto(bytes.NewReader(fileBytes), filename, int64(len(fileBytes)))
	if err != nil {
		return "", "", "", fmt.Errorf("lotService.uploadPhoto.ValidatePhoto: %w", err)
	}

	// Хеш SHA-256 для дедупликации в таблице s3
	sum := sha256.Sum256(fileBytes)
	hash := hex.EncodeToString(sum[:])

	// Регистрируем в таблице s3
	s3ID, isInserted, err := s.repo.insertS3Hash(ctx, hash)
	if err != nil {
		return "", "", "", fmt.Errorf("lotService.uploadPhoto.insertS3Hash: %w", err)
	}

	// Если файла с таким хешем еще нет, загружаем в photosS3 чисто по UUID без префиксов
	if isInserted && s.photosS3 != nil {
		contentType := mtype.String()
		if contentType == "" {
			contentType = "image/jpeg"
		}
		if err = s.photosS3.Upload(ctx, s3ID.String(), bytes.NewReader(fileBytes), contentType); err != nil {
			return "", "", "", fmt.Errorf("lotService.uploadPhoto.s3Upload: %w", err)
		}
	}

	// Отслеживаем загруженную фотографию в Redis ZSet, пока она не прикреплена к лоту
	if s.valkey != nil && s.valkey.Raw() != nil {
		_ = s.valkey.Raw().ZAdd(ctx, "photos:unattached", redis.Z{
			Score:  float64(time.Now().UTC().Unix()),
			Member: s3ID.String(),
		}).Err()
	}

	directURL := s.getDirectPhotoURL(s3ID.String())
	photoURL := directURL
	if photoURL == "" {
		photoURL = "/api/lot/photo/" + s3ID.String()
	}

	return photoURL, s3ID.String(), directURL, nil
}

// GetPhotoPrefix возвращает базовый публичный префикс (URL) для получения фотографий по UUID.
func (s *Service) GetPhotoPrefix() string {
	return PublicPhotosBucket
}

// getDirectPhotoURL формирует прямой публичный URL к объекту в S3 хранилище photosS3.
func (s *Service) getDirectPhotoURL(key string) string {
	if key == "" {
		return ""
	}
	if u, ok := ExtractUUID(key); ok {
		key = u
	}
	return fmt.Sprintf("%s/%s", PublicPhotosBucket, key)
}

func (s *Service) getPhoto(ctx context.Context, idStr string) (io.ReadCloser, string, string, error) {
	key := strings.TrimPrefix(idStr, "/api/lot/photo/")
	if idx := strings.LastIndex(key, "/"); idx != -1 {
		key = key[idx+1:]
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, "", "", storageS3.ErrNotFound
	}

	directURL := s.getDirectPhotoURL(key)

	if s.photosS3 == nil {
		return nil, "", directURL, storageS3.ErrNotFound
	}

	rc, err := s.photosS3.Download(ctx, key)
	if err != nil {
		return nil, "", directURL, err
	}
	return rc, "", directURL, nil
}

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
		return fmt.Errorf("lotService.changeHold: %w", err)
	}

	if !isFirst {
		if err = txRepo.changeHold(ctx, bid.UserId, -bid.Amount); err != nil {
			return fmt.Errorf("lotService.changeHold: %w", err)
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

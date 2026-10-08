package item

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/storageS3"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
)

var (
	errUserAlreadyUploading = apperr.NewAppErrorString("user already has an active upload", -4)
	errUploadNotFound       = apperr.NewAppErrorString("upload not found or expired", -4)
	errForbidden            = apperr.NewAppErrorString("forbidden", -4)
	errFileSizeMismatch     = apperr.NewAppErrorString("file size mismatch", -4)
	errHashMismatch         = apperr.NewAppErrorString("file hash mismatch", -4)
	errDownloadLimitReached = apperr.NewAppErrorString("download limit reached: only 1 download per day allowed", -4)
	errHashAlreadyExists    = apperr.NewAppErrorString("hash already exists", -4)
	errCancelLimitReached   = apperr.NewAppErrorString("cancel limit reached: maximum 3 cancellations per hour allowed", -4)
)

const (
	processingSetKey = "upload:processing:keys"
)

type UploadPending struct {
	UserID    uuid.UUID `json:"user_id"`
	Key       uuid.UUID `json:"key"`
	FileSize  int64     `json:"file_size"`
	Sha256Hex string    `json:"sha_256_hex"`
	Filename  string    `json:"filename"`
	CreatedAt time.Time `json:"created_at"`
}

type UploadStatus struct {
	Status    string          `json:"status"` // "pending", "processing", "completed", "failed", "cancelled"
	Key       uuid.UUID       `json:"key"`
	Error     string          `json:"error,omitempty"`
	Item      *Item           `json:"item,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (s *Service) setStatus(ctx context.Context, key uuid.UUID, status string, errMsg string, item *Item, meta json.RawMessage) error {
	st := UploadStatus{
		Status:    status,
		Key:       key,
		Error:     errMsg,
		Item:      item,
		Metadata:  meta,
		UpdatedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("uploadManager.setStatus: %w", apperr.NewAppError(err, 4))
	}
	if err = s.valkey.Raw().Set(ctx, "upload:status:"+key.String(), data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("uploadManager.setStatus: %w", apperr.NewAppError(err, 4))
	}
	return nil
}

func (s *Service) getUploadStatus(ctx context.Context, uid uuid.UUID, key uuid.UUID) (UploadStatus, error) {
	val, err := s.valkey.Raw().Get(ctx, "upload:status:"+key.String()).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return UploadStatus{}, errUploadNotFound
		}
		return UploadStatus{}, fmt.Errorf("uploadManager.getUploadStatus: %w", apperr.NewAppError(err, 4))
	}

	var status UploadStatus
	if err = json.Unmarshal([]byte(val), &status); err != nil {
		return UploadStatus{}, fmt.Errorf("uploadManager.getUploadStatus: %w", apperr.NewAppError(err, 4))
	}

	return status, nil
}

func (s *Service) createUploadURL(ctx context.Context, uid uuid.UUID, fileSize int64, sha256Hex, filename string) (string, uuid.UUID, error) {
	// 1. Проверяем блокировку: у пользователя не более 1 активной загрузки
	activeKey, err := s.valkey.Raw().Get(ctx, "upload:user:"+uid.String()+":active").Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", uuid.Nil(), fmt.Errorf("uploadManager.createUploadURL: %w", apperr.NewAppError(err, 4))
	}
	if activeKey != "" {
		activeUUID, err := uuid.Parse(activeKey)
		if err != nil {
			return "", uuid.Nil(), fmt.Errorf("uploadManager.createUploadURL: invalid active key: %w", apperr.NewAppError(err, 4))
		}

		pendingVal, err := s.valkey.Raw().Get(ctx, "upload:pending:"+activeKey).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return "", activeUUID, errUploadNotFound
			}
			return "", activeUUID, fmt.Errorf("uploadManager.createUploadURL: %w", apperr.NewAppError(err, 4))
		}

		var pending UploadPending
		if err = json.Unmarshal([]byte(pendingVal), &pending); err != nil {
			return "", activeUUID, fmt.Errorf("uploadManager.createUploadURL: %w", apperr.NewAppError(err, 4))
		}

		url, err := s.s3.GetURLForUpload(ctx, activeUUID, pending.FileSize)
		if err != nil {
			return "", activeUUID, fmt.Errorf("uploadManager.createUploadURL: %w", err)
		}

		return url, activeUUID, errUserAlreadyUploading
	}

	// 2. Проверяем, существует ли уже данный хеш в базе
	exists, err := s.hasHash(ctx, sha256Hex)
	if err != nil {
		return "", uuid.Nil(), fmt.Errorf("uploadManager.createUploadURL: %w", err)
	}
	if exists {
		return "", uuid.Nil(), errHashAlreadyExists
	}

	key := uuid.New()

	// 3. Генерируем presigned URL для S3
	url, err := s.s3.GetURLForUpload(ctx, key, fileSize)
	if err != nil {
		return "", uuid.Nil(), fmt.Errorf("uploadManager.createUploadURL: %w", err)
	}

	pending := UploadPending{
		UserID:    uid,
		Key:       key,
		FileSize:  fileSize,
		Sha256Hex: sha256Hex,
		Filename:  filename,
		CreatedAt: time.Now().UTC(),
	}
	pendingBytes, err := json.Marshal(pending)
	if err != nil {
		return "", uuid.Nil(), fmt.Errorf("uploadManager.createUploadURL: %w", apperr.NewAppError(err, 4))
	}

	// 4. Сохраняем в Valkey
	pipe := s.valkey.Raw().Pipeline()
	pipe.Set(ctx, "upload:user:"+uid.String()+":active", key.String(), 2*time.Hour)
	pipe.Set(ctx, "upload:pending:"+key.String(), pendingBytes, 24*time.Hour)
	if _, err = pipe.Exec(ctx); err != nil {
		return "", uuid.Nil(), fmt.Errorf("uploadManager.createUploadURL: %w", apperr.NewAppError(err, 4))
	}

	_ = s.setStatus(ctx, key, "pending", "", nil, nil)

	return url, key, nil
}

// cancelUpload отменяет активную загрузку пользователя, удаляет временный файл из S3 и очищает Valkey.
// Разрешено не более 3 отмен в час на пользователя.
func (s *Service) cancelUpload(ctx context.Context, uid uuid.UUID) error {
	// Лимит: не чаще 3 отмен в час на пользователя (проверяется до основной логики)
	cancelLimitKey := "upload:cancel:limit:" + uid.String()
	count, err := s.valkey.Raw().Incr(ctx, cancelLimitKey).Result()
	if err != nil {
		return fmt.Errorf("uploadManager.cancelUpload: %w", apperr.NewAppError(err, 4))
	}
	if count == 1 {
		_ = s.valkey.Raw().Expire(ctx, cancelLimitKey, time.Hour).Err()
	}
	if count > 3 {
		return errCancelLimitReached
	}

	activeKey, err := s.valkey.Raw().Get(ctx, "upload:user:"+uid.String()+":active").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			slog.DebugContext(ctx, "uploadManager.cancelUpload: no active upload found", slog.String("uid", uid.String()))
			return nil
		}
		return fmt.Errorf("uploadManager.cancelUpload: %w", apperr.NewAppError(err, 4))
	}

	key, err := uuid.Parse(activeKey)
	if err != nil {
		_ = s.valkey.Raw().Del(ctx, "upload:user:"+uid.String()+":active").Err()
		return fmt.Errorf("uploadManager.cancelUpload: invalid active key: %w", apperr.NewAppError(err, 4))
	}

	// 1. Немедленно удаляем временный файл из S3, чтобы освободить место
	_ = s.s3.Delete(ctx, "tmp/"+key.String())

	// 2. Очищаем ключи и очередь в Valkey
	pipe := s.valkey.Raw().Pipeline()
	pipe.Del(ctx, "upload:user:"+uid.String()+":active")
	pipe.Del(ctx, "upload:pending:"+key.String())
	pipe.SRem(ctx, processingSetKey, key.String())
	if _, err = pipe.Exec(ctx); err != nil {
		return fmt.Errorf("uploadManager.cancelUpload: %w", apperr.NewAppError(err, 4))
	}

	_ = s.setStatus(ctx, key, "cancelled", "upload cancelled by user", nil, nil)
	return nil
}

// confirmUpload проверяет первые 512 байт (каплю), имя файла и размер, а затем запускает фоновую обработку
func (s *Service) confirmUpload(ctx context.Context, uid uuid.UUID, key uuid.UUID) error {
	pendingVal, err := s.valkey.Raw().Get(ctx, "upload:pending:"+key.String()).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return errUploadNotFound
		}
		return fmt.Errorf("uploadManager.confirmUpload: %w", apperr.NewAppError(err, 4))
	}

	var pending UploadPending
	if err = json.Unmarshal([]byte(pendingVal), &pending); err != nil {
		return fmt.Errorf("uploadManager.confirmUpload: %w", apperr.NewAppError(err, 4))
	}

	if pending.UserID != uid {
		return errForbidden
	}

	s3TmpKey := "tmp/" + key.String()

	// 1. Проверяем реальный размер файла в S3
	actualSize, err := s.s3.Head(ctx, s3TmpKey)
	if err != nil {
		if errors.Is(err, storageS3.ErrNotFound) {
			return errUploadNotFound
		}
		return fmt.Errorf("uploadManager.confirmUpload: %w", apperr.NewAppError(err, 4))
	}
	if actualSize != pending.FileSize {
		_ = s.s3.Delete(ctx, s3TmpKey)
		_ = s.valkey.Raw().Del(ctx, "upload:user:"+uid.String()+":active")
		_ = s.setStatus(ctx, key, "failed", "file size mismatch", nil, nil)
		return errFileSizeMismatch
	}

	// 2. Скачиваем буквально каплю (первые 512 байт) из S3
	capReader, err := s.s3.DownloadRange(ctx, s3TmpKey, 0, 511)
	if err != nil {
		if errors.Is(err, storageS3.ErrNotFound) {
			return errUploadNotFound
		}
		return fmt.Errorf("uploadManager.confirmUpload: %w", apperr.NewAppError(err, 4))
	}
	defer capReader.Close()

	// 3. Проверяем MIME-тип, соответствие имени файла и ограничения по типу (видео до 1 ГБ, архивы до 5 ГБ)
	iType, mtype, err := validateSample(capReader, pending.Filename, actualSize)
	if err != nil {
		_ = s.s3.Delete(ctx, s3TmpKey)
		_ = s.valkey.Raw().Del(ctx, "upload:user:"+uid.String()+":active")
		_ = s.setStatus(ctx, key, "failed", err.Error(), nil, nil)
		return err
	}

	// 4. Добавляем ключ в множество выполняемых задач Valkey для восстановления при падении
	if err = s.valkey.Raw().SAdd(ctx, processingSetKey, key.String()).Err(); err != nil {
		return fmt.Errorf("uploadManager.confirmUpload: %w", apperr.NewAppError(err, 4))
	}

	// 5. Переводим статус в processing и запускаем фоновую обработку
	_ = s.setStatus(ctx, key, "processing", "", nil, nil)

	go s.processFullUpload(context.Background(), pending, iType, mtype.String())

	return nil
}

// processFullUpload фоново скачивает весь файл, сверяет sha256 и строит metadata
func (s *Service) processFullUpload(ctx context.Context, pending UploadPending, iType itemType, mType string) {
	// Семафор: не более N параллельных обработок (остальные подождут)
	s.uploadSem <- struct{}{}
	defer func() { <-s.uploadSem }()

	// Очищаем блокировку пользователя и удаляем из очереди активных задач по завершении
	defer func() {
		_ = s.valkey.Raw().Del(context.Background(), "upload:user:"+pending.UserID.String()+":active").Err()
		_ = s.valkey.Raw().SRem(context.Background(), processingSetKey, pending.Key.String()).Err()
	}()

	s3TmpKey := "tmp/" + pending.Key.String()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	body, err := s.s3.Download(ctxWithTimeout, s3TmpKey)
	if err != nil {
		apperr.Log(ctx, "uploadManager.processFullUpload: download error", err)
		_ = s.s3.Delete(context.Background(), s3TmpKey)
		_ = s.setStatus(context.Background(), pending.Key, "failed", fmt.Sprintf("s3 download failed: %v", err), nil, nil)
		return
	}
	defer body.Close()

	prHash, pwHash := io.Pipe()
	prMData, pwMData := io.Pipe()
	tee := io.TeeReader(body, io.MultiWriter(pwHash, pwMData))

	g, gCtx := errgroup.WithContext(ctxWithTimeout)

	hasher := sha256.New()
	g.Go(func() (err error) {
		defer func() {
			_ = prHash.CloseWithError(err)
		}()
		_, err = io.Copy(hasher, prHash)
		return err
	})

	var metadata json.RawMessage
	hmData := headerMetadata{
		Filename: pending.Filename,
		Size:     pending.FileSize,
		MimeType: mType,
	}
	g.Go(func() (err error) {
		defer func() {
			_ = prMData.CloseWithError(err)
		}()
		metadata, err = createItemMetadata(gCtx, iType, mType, hmData, prMData)
		return err
	})

	g.Go(func() (err error) {
		defer func() {
			_ = pwHash.CloseWithError(context.Cause(gCtx))
			_ = pwMData.CloseWithError(context.Cause(gCtx))
		}()
		_, err = io.Copy(io.Discard, tee)
		return err
	})

	if err = g.Wait(); err != nil {
		apperr.Log(ctx, "uploadManager.processFullUpload: pipeline error", err)
		_ = s.s3.Delete(context.Background(), s3TmpKey)
		_ = s.setStatus(context.Background(), pending.Key, "failed", fmt.Sprintf("processing error: %v", err), nil, nil)
		return
	}

	// Сверяем реальный вычисленный хеш с хешем пользователя
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actualHash, pending.Sha256Hex) {
		apperr.Log(ctx, "uploadManager.processFullUpload: hash mismatch", errHashMismatch)
		_ = s.s3.Delete(context.Background(), s3TmpKey)
		_ = s.setStatus(context.Background(), pending.Key, "failed", "hash mismatch: actual file hash does not match expected", nil, nil)
		return
	}

	// Перемещаем файл из tmp/<key> в <key>
	if err = s.s3.Move(context.Background(), s3TmpKey, pending.Key.String()); err != nil {
		apperr.Log(ctx, "uploadManager.processFullUpload: move error", err)
		_ = s.s3.Delete(context.Background(), s3TmpKey)
		_ = s.setStatus(context.Background(), pending.Key, "failed", fmt.Sprintf("s3 move error: %v", err), nil, nil)
		return
	}

	// Сохраняем в таблицах s3 и items
	if err = s.repo.createS3(context.Background(), pending.Key, pending.Sha256Hex); err != nil {
		apperr.Log(ctx, "uploadManager.processFullUpload: createS3 error", err)
		_ = s.setStatus(context.Background(), pending.Key, "failed", fmt.Sprintf("database s3 error: %v", err), nil, nil)
		return
	}

	item, err := s.repo.createItem(context.Background(), pending.UserID, pending.Key, iType, metadata)
	if err != nil {
		apperr.Log(ctx, "uploadManager.processFullUpload: createItem error", err)
		_ = s.setStatus(context.Background(), pending.Key, "failed", fmt.Sprintf("database item error: %v", err), nil, nil)
		return
	}

	_ = s.setStatus(context.Background(), pending.Key, "completed", "", &item, metadata)
}

// RecoverPendingUploads восстанавливает незавершенные задачи при запуске сервера
func (s *Service) RecoverPendingUploads(ctx context.Context) {
	keys, err := s.valkey.Raw().SMembers(ctx, processingSetKey).Result()
	if err != nil {
		apperr.Log(ctx, "uploadManager.RecoverPendingUploads: failed to get processing keys", err)
		return
	}

	for _, kStr := range keys {
		k, err := uuid.Parse(kStr)
		if err != nil {
			_ = s.valkey.Raw().SRem(ctx, processingSetKey, kStr).Err()
			continue
		}

		pendingVal, err := s.valkey.Raw().Get(ctx, "upload:pending:"+k.String()).Result()
		if err != nil {
			_ = s.valkey.Raw().SRem(ctx, processingSetKey, kStr).Err()
			continue
		}

		var pending UploadPending
		if err = json.Unmarshal([]byte(pendingVal), &pending); err != nil {
			_ = s.valkey.Raw().SRem(ctx, processingSetKey, kStr).Err()
			continue
		}

		s3TmpKey := "tmp/" + k.String()
		size, err := s.s3.Head(ctx, s3TmpKey)
		if err != nil {
			_ = s.valkey.Raw().SRem(ctx, processingSetKey, kStr).Err()
			_ = s.setStatus(ctx, k, "failed", "interrupted upload: temporary file not found in s3", nil, nil)
			continue
		}

		capReader, err := s.s3.DownloadRange(ctx, s3TmpKey, 0, 511)
		if err != nil {
			_ = s.valkey.Raw().SRem(ctx, processingSetKey, kStr).Err()
			_ = s.setStatus(ctx, k, "failed", "interrupted upload: failed to read sample", nil, nil)
			continue
		}
		iType, mtype, err := validateSample(capReader, pending.Filename, size)
		capReader.Close()
		if err != nil {
			_ = s.s3.Delete(ctx, s3TmpKey)
			_ = s.valkey.Raw().SRem(ctx, processingSetKey, kStr).Err()
			_ = s.setStatus(ctx, k, "failed", err.Error(), nil, nil)
			continue
		}

		// Запускаем возобновление задачи
		go s.processFullUpload(context.Background(), pending, iType, mtype.String())
	}
}

// getDownloadURL проверяет владельца, суточный лимит и генерирует presigned URL
func (s *Service) getDownloadURL(ctx context.Context, uid uuid.UUID, itemID int64) (string, error) {
	item, err := s.repo.getItem(ctx, itemID)
	if err != nil {
		return "", fmt.Errorf("uploadManager.getDownloadURL: %w", err)
	}

	// 1. Только владелец может скачать
	if item.OwnerID != uid {
		return "", errForbidden
	}

	// TODO: Удаление файла из S3 делать при удалении аккаунта пользователя

	// 2. Скачать можно только 1 раз в день с 1 uuid
	now := time.Now().UTC()
	dateStr := now.Format("2006-01-02")
	limitKey := fmt.Sprintf("download:limit:%s:%s:%s", uid.String(), item.S3ID.String(), dateStr)

	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	ttl := time.Until(midnight)

	ok, err := s.valkey.Raw().SetNX(ctx, limitKey, "1", ttl).Result()
	if err != nil {
		return "", fmt.Errorf("uploadManager.getDownloadURL: %w", apperr.NewAppError(err, 4))
	}
	if !ok {
		return "", errDownloadLimitReached
	}

	// Извлекаем имя файла из metadata, если есть
	filename := fmt.Sprintf("%s.bin", item.S3ID.String())
	var metaMap map[string]any
	if err = json.Unmarshal(item.MetaData, &metaMap); err == nil {
		if fn, ok := metaMap["filename"].(string); ok && fn != "" {
			filename = fn
		}
	}

	url, err := s.s3.GetURLForDownload(ctx, item.S3ID.String(), 1*time.Hour, filename)
	if err != nil {
		return "", fmt.Errorf("uploadManager.getDownloadURL: %w", err)
	}

	return url, nil
}

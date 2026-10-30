package lot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/redis/go-redis/v9"
)

// CleanExpiredLotPhotos запускает периодический фоновый процесс очистки фотографий:
// - Сохраняются фотографии лотов в статусах: pending_review (проверка), approved (утвержден), active (активен), finished (завершен).
// - Если лот находится в статусе rejected, draft или cancelled и создан более 30 дней назад, его фотографии удаляются из S3 и базы данных.
// - Также удаляются неприкрепленные (осиротевшие) фотографии, загруженные более 30 дней назад.
func (s *Service) CleanExpiredLotPhotos(ctx context.Context) {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	s.cleanPhotosCycle(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanPhotosCycle(ctx)
		}
	}
}

func (s *Service) cleanPhotosCycle(ctx context.Context) {
	s.cleanExpiredLotsPhotos(ctx)
	s.cleanUnattachedPhotos(ctx)
}

// cleanExpiredLotsPhotos очищает фото из лотов со статусами draft/rejected/cancelled старше 30 дней
// в транзакции с FOR UPDATE SKIP LOCKED.
func (s *Service) cleanExpiredLotsPhotos(ctx context.Context) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		apperr.Log(ctx, "cleanPhotosCycle.beginTx", err)
		return
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	// Блокируем строки устаревших лотов через FOR UPDATE SKIP LOCKED
	lockedLotIDs, err := txRepo.lockCandidateExpiredLots(ctx)
	if err != nil {
		apperr.Log(ctx, "cleanPhotosCycle.lockCandidateExpiredLots", err)
		return
	}
	if len(lockedLotIDs) == 0 {
		return
	}

	// Находим ключи фото, которые нигде больше не используются
	expiredKeys, err := txRepo.getExpiredPhotoKeysForLockedLots(ctx, lockedLotIDs)
	if err != nil {
		apperr.Log(ctx, "cleanPhotosCycle.getExpiredPhotoKeysForLockedLots", err)
		return
	}

	// Очищаем массив фото в заблокированных лотах
	if err := txRepo.clearLockedLotsPhotos(ctx, lockedLotIDs); err != nil {
		apperr.Log(ctx, "cleanPhotosCycle.clearLockedLotsPhotos", err)
		return
	}

	// Удаляем записи из таблицы s3
	deletedKeys, err := txRepo.deleteS3RecordsIfUnused(ctx, expiredKeys)
	if err != nil {
		apperr.Log(ctx, "cleanPhotosCycle.deleteS3RecordsIfUnused", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		apperr.Log(ctx, "cleanPhotosCycle.commit", err)
		return
	}

	// Фиксация в БД успешна — удаляем объекты из S3 и Redis
	for _, key := range deletedKeys {
		if s.photosS3 != nil {
			if s3Err := s.photosS3.Delete(ctx, key); s3Err != nil {
				apperr.Log(ctx, "cleanPhotosCycle.s3Delete", s3Err, "key", key)
			}
		}
		if s.valkey != nil && s.valkey.Raw() != nil {
			_ = s.valkey.Raw().ZRem(ctx, "photos:unattached", key).Err()
		}
	}

	slog.Info("clean_expired_photos_lots_completed",
		"locked_lots", len(lockedLotIDs),
		"deleted_photos", len(deletedKeys),
	)
}

// cleanUnattachedPhotos очищает неприкрепленные фотографии старше 30 дней из Redis ZSet photos:unattached.
func (s *Service) cleanUnattachedPhotos(ctx context.Context) {
	if s.valkey == nil || s.valkey.Raw() == nil {
		return
	}

	cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour).Unix()
	unattachedKeys, err := s.valkey.Raw().ZRangeByScore(ctx, "photos:unattached", &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", cutoff),
	}).Result()
	if err != nil {
		apperr.Log(ctx, "cleanPhotosCycle.ZRangeByScore", err)
		return
	}

	unattachedDeleted := 0
	for _, key := range unattachedKeys {
		deleted, err := s.cleanSingleUnattachedPhoto(ctx, key)
		if err != nil {
			continue
		}
		if deleted {
			unattachedDeleted++
		}
	}

	if unattachedDeleted > 0 {
		slog.Info("clean_unattached_photos_completed",
			"deleted_unattached", unattachedDeleted,
		)
	}
}

func (s *Service) cleanSingleUnattachedPhoto(ctx context.Context, key string) (bool, error) {
	referenced, err := s.repo.isOrphanPhotoReferenced(ctx, key)
	if err != nil {
		apperr.Log(ctx, "cleanPhotosCycle.isOrphanPhotoReferenced", err, "key", key)
		return false, err
	}

	// Если фото используется в каком-то лоте, просто убираем его из очереди отслеживания
	if referenced {
		_ = s.valkey.Raw().ZRem(ctx, "photos:unattached", key).Err()
		return false, nil
	}

	deletedKeys, err := s.repo.deleteS3RecordsIfUnused(ctx, []string{key})
	if err != nil {
		apperr.Log(ctx, "cleanPhotosCycle.unattachedDeleteS3Record", err, "key", key)
		return false, err
	}

	if len(deletedKeys) > 0 && s.photosS3 != nil {
		if s3Err := s.photosS3.Delete(ctx, key); s3Err != nil {
			apperr.Log(ctx, "cleanPhotosCycle.unattachedS3Delete", s3Err, "key", key)
		}
	}

	_ = s.valkey.Raw().ZRem(ctx, "photos:unattached", key).Err()
	return true, nil
}

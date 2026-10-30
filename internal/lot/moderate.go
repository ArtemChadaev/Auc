package lot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/storage"
	"github.com/gabriel-vasile/mimetype"
	"github.com/jackc/pgx/v5"
)

// ModerateLots запускает фоновый процесс периодической модерации лотов в статусе pending_review.
func (s *Service) ModerateLots(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	s.autoModerate(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.autoModerate(ctx)
		}
	}
}

// AutoModerate запускает один цикл проверки лотов, ожидающих модерации.
func (s *Service) AutoModerate(ctx context.Context) {
	s.autoModerate(ctx)
}

func (s *Service) autoModerate(ctx context.Context) {
	ids, err := s.repo.allPendingReview(ctx)
	if err != nil {
		apperr.Log(ctx, "autoModerate.allPendingReview", err)
		return
	}
	for _, id := range ids {
		if err = s.moderateLot(ctx, id); err != nil {
			apperr.Log(ctx, "autoModerate.moderateLot", err, "lot_id", id)
		}
	}
}

func (s *Service) moderateLot(ctx context.Context, idLot int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("moderateLot.begin: %w", apperr.NewAppError(err, 12))
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	lot, err := txRepo.getLotForReview(ctx, idLot)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, storage.ErrNotFound) {
			// Лот уже заблокирован или обработан другим воркером
			return nil
		}
		return fmt.Errorf("moderateLot.getLotForReview: %w", err)
	}

	// 1. Проверка текстового контента лота (заголовок, описание, теги)
	if reason, ok := checkLotText(lot); !ok {
		apperr.Log(ctx, "moderateLot.textRejected", fmt.Errorf("lot #%d text rejected: %s", lot.ID, reason), "lot_id", lot.ID)
		if err = txRepo.reject(ctx, lot.ID, reason); err != nil {
			return fmt.Errorf("moderateLot.reject: %w", err)
		}
		return tx.Commit(ctx)
	}

	// 2. Проверка прикрепленных фотографий лота
	if reason, ok, isTransient := s.checkLotPhotos(ctx, lot); !ok {
		if isTransient {
			// Временная ошибка хранилища/сети S3 — не отклоняем лот, повторяем на следующем цикле
			apperr.Log(ctx, "moderateLot.photosTransientError", fmt.Errorf("lot #%d transient error: %s", lot.ID, reason), "lot_id", lot.ID)
			return nil
		}
		apperr.Log(ctx, "moderateLot.photosRejected", fmt.Errorf("lot #%d photos rejected: %s", lot.ID, reason), "lot_id", lot.ID)
		if err = txRepo.reject(ctx, lot.ID, reason); err != nil {
			return fmt.Errorf("moderateLot.reject: %w", err)
		}
		return tx.Commit(ctx)
	}

	// 3. Все проверки пройдены успешно: переводим статус лота в approved или active
	// с использованием имеющейся функции approve()
	if err = txRepo.approve(ctx, lot.ID); err != nil {
		return fmt.Errorf("moderateLot.approve: %w", err)
	}

	return tx.Commit(ctx)
}

// Список запрещенных слов и спам-паттернов согласно Content Policy
var prohibitedKeywords = []string{
	// Мошенничество и фишинг
	"t.me/", "bit.ly", "tinyurl", "free crypto", "free money", "раздача денег",
	"бесплатные деньги", "заработок без вложений", "100% доход", "гарантированный доход",
	"казино вулкан", "ставки на спорт", "взлом аккаунтов", "стиллер", "hack tool",
	// Запрещенные вещества и оружие
	"купить наркотики", "мефедрон", "героин", "кокаин", "купить оружие", "взрывчатка",
	// NSFW / контент для взрослых
	"порнография", "детское порно", "child abuse", "секс за деньги", "интим услуги",
}

func checkLotText(lot Lot) (string, bool) {
	trimmedTitle := strings.TrimSpace(lot.Title)
	if len([]rune(trimmedTitle)) < 3 {
		return "Название лота слишком короткое (минимум 3 символа)", false
	}
	if len([]rune(trimmedTitle)) > 250 {
		return "Название лота превышает допустимую длину (максимум 250 символов)", false
	}

	if match, found := containsProhibited(trimmedTitle); found {
		return fmt.Sprintf("Название лота содержит недопустимый контент: '%s'", match), false
	}

	if lot.Description.ShortText != nil {
		st := strings.TrimSpace(*lot.Description.ShortText)
		if len([]rune(st)) > 500 {
			return "Краткое описание превышает лимит 500 символов", false
		}
		if match, found := containsProhibited(st); found {
			return fmt.Sprintf("Краткое описание содержит недопустимый контент: '%s'", match), false
		}
	}

	if lot.Description.Text != nil {
		txt := strings.TrimSpace(*lot.Description.Text)
		if len([]rune(txt)) > 10000 {
			return "Подробное описание превышает лимит 10000 символов", false
		}
		if match, found := containsProhibited(txt); found {
			return fmt.Sprintf("Описание лота содержит недопустимый контент: '%s'", match), false
		}
	}

	if lot.Description.Tags != nil {
		for _, tag := range *lot.Description.Tags {
			t := strings.TrimSpace(tag)
			if t == "" {
				continue
			}
			if len([]rune(t)) > 50 {
				return "Тег превышает лимит 50 символов", false
			}
			if match, found := containsProhibited(t); found {
				return fmt.Sprintf("Тег содержит недопустимый контент: '%s'", match), false
			}
		}
	}

	return "", true
}

func containsProhibited(text string) (string, bool) {
	lower := strings.ToLower(text)
	for _, kw := range prohibitedKeywords {
		if strings.Contains(lower, kw) {
			return kw, true
		}
	}
	return "", false
}

func (s *Service) checkLotPhotos(ctx context.Context, lot Lot) (reason string, ok bool, isTransient bool) {
	images := lot.Description.GetImages()
	if len(images) == 0 {
		return "", true, false
	}

	for _, photoRef := range images {
		key := strings.TrimSpace(photoRef)
		if key == "" {
			continue
		}

		// Модерация ожидает строго UUID без путей и префиксов
		parsedUUID, err := uuid.Parse(key)
		if err != nil {
			return fmt.Sprintf("Идентификатор фотографии '%s' не является корректным UUID", key), false, false
		}
		cleanKey := parsedUUID.String()

		if s.photosS3 != nil {
			exists, err := s.photosS3.Exists(ctx, cleanKey)
			if err != nil {
				return fmt.Sprintf("Ошибка проверки наличия фото %s в S3: %v", cleanKey, err), false, true
			}

			if !exists {
				return fmt.Sprintf("Фотография %s не найдена в хранилище", cleanKey), false, false
			}

			// Скачиваем превью (первые 512 байт) для валидации типа и структуры изображения
			rc, err := s.photosS3.DownloadRange(ctx, cleanKey, 0, 511)
			if err != nil {
				rc, err = s.photosS3.Download(ctx, cleanKey)
			}
			if err != nil {
				return fmt.Sprintf("Ошибка чтения фото %s из S3: %v", cleanKey, err), false, true
			}

			sample := make([]byte, 512)
			n, _ := io.ReadFull(rc, sample)
			_ = rc.Close()

			if n == 0 {
				return fmt.Sprintf("Файл фотографии %s пуст", cleanKey), false, false
			}

			// Проверяем, что файл в S3 является валидным растровым изображением (JPEG, PNG, WebP)
			mtype := mimetype.Detect(sample[:n])
			if !strings.HasPrefix(mtype.String(), "image/") || mtype.String() == "image/svg+xml" {
				return fmt.Sprintf("Фотография %s не является допустимым растровым изображением", cleanKey), false, false
			}
			allowedMimes := []string{"image/jpeg", "image/png", "image/webp"}
			if !slices.Contains(allowedMimes, mtype.String()) {
				return fmt.Sprintf("Фотография %s имеет неподдерживаемый формат %s (разрешены JPEG, PNG, WebP)", cleanKey, mtype.String()), false, false
			}
		}
	}

	return "", true, false
}

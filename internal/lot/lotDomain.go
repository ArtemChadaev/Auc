package lot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/mimeType"
)

// PublicPhotosBucket публичный базовый URL бакета photosS3.
const PublicPhotosBucket = "https://s3.ru1.storage.beget.cloud/d7201f92dc71-auc-photos"

type Status string

var (
	Draft         Status = "draft"
	PendingReview Status = "pending_review"
	Approved      Status = "approved"
	Active        Status = "active"
	Rejected      Status = "rejected"
	Finished      Status = "finished"
	Cancelled     Status = "cancelled"
)

var (
	ErrInvalidID               = apperr.NewAppErrorString("invalid lot id", -4)
	ErrLotNotFound             = apperr.NewAppErrorString("lot not found", -4)
	ErrLotNotApproved          = apperr.NewAppErrorString("lot must be in approved status to be activated", -4)
	ErrForbidden               = apperr.NewAppErrorString("forbidden", -4)
	ErrInvalidReqBody          = apperr.NewAppErrorString("invalid request body", -4)
	ErrLider                   = apperr.NewAppErrorString("you are already the leading bidder", -4)
	ErrLeader                  = ErrLider
	ErrAmountSmall             = apperr.NewAppErrorString("bid amount is too small", -4)
	ErrInsufficientBalance     = apperr.NewAppErrorString("insufficient balance for bid", -4)
	ErrPhotoUploadLimitReached = apperr.NewAppErrorString("photo upload limit reached: max 30 photos per hour allowed", -4)
)

type Lot struct {
	ID           int64       `db:"id" json:"id"`
	SellerID     uuid.UUID   `db:"seller_id" json:"seller_id"`
	ItemID       int64       `db:"item_id" json:"item_id"`
	Status       Status      `db:"status" json:"status"`
	WinnerID     *uuid.UUID  `db:"winner_id" json:"winner_id,omitempty"`
	StartAmount  int         `db:"start_amount" json:"start_amount"`
	MinStep      int         `db:"min_step" json:"min_step"`
	FinishAmount *int        `db:"finish_amount" json:"finish_amount,omitempty"`
	Title        string      `db:"title" json:"title"`
	Description  Description `db:"description" json:"description"`
	CreatedAt    time.Time   `db:"created_at" json:"created_at"`
	StartedAt    *time.Time  `db:"started_at" json:"started_at,omitempty"`
	EndedAt      *time.Time  `db:"ends_at" json:"ends_at,omitempty"`
	FinishedAt   *time.Time  `db:"finished_at" json:"finished_at,omitempty"`
}

type Description struct {
	Type         mimeType.ItemType `json:"type,omitempty"`
	Tags         *[25]string       `json:"tags,omitempty"`
	PhotoURL     *[10]string       `json:"photo,omitempty"`
	Images       *[10]string       `json:"images,omitempty"`
	Text         *string           `json:"text,omitempty"`
	ShortText    *string           `json:"short_text,omitempty"`
	AutoActivate *bool             `json:"auto_activate,omitempty"`
	DraftUI      *string           `json:"draft_ui,omitempty"`
}

// ExtractUUID извлекает UUID из переданной строки:
// - если длина ровно 36 или 32 символа, сразу проверяет валидность UUID;
// - если больше 36 символов, ищет в первых 36 символах, затем в последних 36 символах;
// - если больше 32 символов, ищет в первых 32 символах, затем в последних 32 символах;
// - если передан путь/URL, проверяет сегмент после последнего слеша.
func ExtractUUID(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", false
	}

	// 1. Точное совпадение по длине UUID (36 с дефисами или 32 без дефисов)
	if len(s) == 36 || len(s) == 32 {
		if u, err := uuid.Parse(s); err == nil {
			return u.String(), true
		}
	}

	// 2. Длина больше 36: проверяем первые 36 символов, затем последние 36 символов
	if len(s) > 36 {
		if u, err := uuid.Parse(s[:36]); err == nil {
			return u.String(), true
		}
		if u, err := uuid.Parse(s[len(s)-36:]); err == nil {
			return u.String(), true
		}
	}

	// 3. Длина больше 32: проверяем первые 32 символа, затем последние 32 символа
	if len(s) > 32 {
		if u, err := uuid.Parse(s[:32]); err == nil {
			return u.String(), true
		}
		if u, err := uuid.Parse(s[len(s)-32:]); err == nil {
			return u.String(), true
		}
	}

	// 4. Если в строке есть слеши (URL или путь)
	if idx := strings.LastIndex(s, "/"); idx != -1 && idx < len(s)-1 {
		part := s[idx+1:]
		if u, ok := ExtractUUID(part); ok {
			return u, true
		}
	}

	return "", false
}

// GetImages возвращает список валидных идентификаторов (UUID) фотографий лота.
func (d Description) GetImages() []string {
	if d.Images != nil {
		var res []string
		for _, img := range *d.Images {
			if u, ok := ExtractUUID(img); ok {
				res = append(res, u)
			}
		}
		if len(res) > 0 {
			return res
		}
	}
	if d.PhotoURL != nil {
		var res []string
		for _, p := range *d.PhotoURL {
			if u, ok := ExtractUUID(p); ok {
				res = append(res, u)
			}
		}
		return res
	}
	return nil
}

// MarshalJSON сериализует Description, автоматически добавляя публичный URL в PhotoURL для каждого UUID из Images.
func (d Description) MarshalJSON() ([]byte, error) {
	type Alias Description

	images := d.GetImages()
	var photos []string
	if len(images) > 0 {
		photos = make([]string, len(images))
		for i, u := range images {
			photos[i] = fmt.Sprintf("%s/%s", PublicPhotosBucket, u)
		}
	}

	return json.Marshal(&struct {
		Alias
		Images   []string `json:"images,omitempty"`
		PhotoURL []string `json:"photo,omitempty"`
	}{
		Alias:    Alias(d),
		Images:   images,
		PhotoURL: photos,
	})
}

// UnmarshalJSON десериализует Description, принимая UUID только из images и игнорируя входящий photoURL.
func (d *Description) UnmarshalJSON(data []byte) error {
	type Alias Description
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(d),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Извлекаем только валидные UUID из images (или photo для обратной совместимости)
	rawList := d.Images
	if rawList == nil {
		rawList = d.PhotoURL
	}

	if rawList != nil {
		var cleanImages [10]string
		var cleanPhotos [10]string
		count := 0
		for _, raw := range *rawList {
			if strings.TrimSpace(raw) == "" {
				continue
			}
			if u, ok := ExtractUUID(raw); ok {
				if count < 10 {
					cleanImages[count] = u
					cleanPhotos[count] = fmt.Sprintf("%s/%s", PublicPhotosBucket, u)
					count++
				}
			}
		}
		if count > 0 {
			d.Images = &cleanImages
			d.PhotoURL = &cleanPhotos
		} else {
			d.Images = nil
			d.PhotoURL = nil
		}
	} else {
		d.Images = nil
		d.PhotoURL = nil
	}

	return nil
}

type Bid struct {
	Id        int64     `db:"id" json:"-"`
	LotId     int64     `db:"lot_id" json:"lot_id"`
	UserId    uuid.UUID `db:"user_id" json:"user_id"`
	Amount    int       `db:"amount" json:"amount"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

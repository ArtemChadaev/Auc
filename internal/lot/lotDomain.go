package lot

import (
	"encoding/json"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/mimeType"
)

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
	ErrInvalidID      = apperr.NewAppErrorString("invalid lot id", -4)
	ErrLotNotFound    = apperr.NewAppErrorString("lot not found", -4)
	ErrLotNotApproved = apperr.NewAppErrorString("lot must be in approved status to be activated", -4)
	ErrForbidden      = apperr.NewAppErrorString("forbidden", -4)
	ErrInvalidReqBody = apperr.NewAppErrorString("invalid request body", -4)
)

type Lot struct {
	ID           int64           `db:"id" json:"id"`
	SellerID     uuid.UUID       `db:"seller_id" json:"seller_id"`
	ItemID       int64           `db:"item_id" json:"item_id"`
	Status       Status          `db:"status" json:"status"`
	WinnerID     *uuid.UUID      `db:"winner_id" json:"winner_id,omitempty"`
	StartAmount  int             `db:"start_amount" json:"start_amount"`
	MinStep      int             `db:"min_step" json:"min_step"`
	FinishAmount *int            `db:"finish_amount" json:"finish_amount,omitempty"`
	Title        string          `db:"title" json:"title"`
	Description  json.RawMessage `db:"description" json:"description,omitempty"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	StartedAt    *time.Time      `db:"started_at" json:"started_at,omitempty"`
	EndedAt      *time.Time      `db:"ends_at" json:"ends_at,omitempty"`
	FinishedAt   *time.Time      `db:"finished_at" json:"finished_at,omitempty"`
}

type Description struct {
	Type         mimeType.ItemType `json:"type,omitempty"`
	Tags         *[25]string       `json:"tags,omitempty"`
	PhotoURL     *[10]string       `json:"photo,omitempty"`
	Text         *string           `json:"text,omitempty"`
	ShortText    *string           `json:"short_text,omitempty"`
	AutoActivate *bool             `json:"auto_activate,omitempty"`
	AutoActive   *bool             `json:"auto_active,omitempty"`
}

type Bid struct {
	Id             int64     `db:"id" json:"-"`
	LotId          int64     `db:"lot_id" json:"lot_id"`
	UserId         uuid.UUID `db:"user_id" json:"user_id"`
	Amount         int       `db:"amount" json:"amount"`
	IdempotencyKey uuid.UUID `db:"idempotency_key" json:"-"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

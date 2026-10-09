package documents

import (
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
)

type Document struct {
	ID           int64      `db:"id" json:"id"`
	Name         string     `db:"name" json:"name"`
	Version      string     `db:"version" json:"version"`
	IsMajor      bool       `db:"is_major" json:"is_major"`
	Text         string     `db:"text" json:"text"`
	PublishedAt  time.Time  `db:"published_at" json:"published_at"`
	SupersededAt *time.Time `db:"superseded_at" json:"superseded_at,omitempty"`
}

type UserConsent struct {
	DocumentID int64     `db:"document_id" json:"document_id"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

var (
	ErrMissingDocumentID = apperr.NewAppErrorString("document_id must be greater than 0", -4)
	ErrInvalidID         = apperr.NewAppErrorString("invalid id parameter", -4)
	ErrDocumentNotFound  = apperr.NewAppErrorString("document not found", -4)
	ErrConsentNotFound   = apperr.NewAppErrorString("consent not found", -4)
)

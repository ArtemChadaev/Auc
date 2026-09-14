package documents

import (
	"context"
	"net/netip"
	"uuid"

	"github.com/ArtemChadaev/Auction/internal/domain"
)

type docRepo interface {
	LastVersion(ctx context.Context, name string) (domain.Document, error)
	AllDocument(ctx context.Context) ([]domain.Document, error)
	UserAllow(ctx context.Context, docID int64, userID *int64, anonID *uuid.UUID, ip *netip.Addr, source string) error
	UserWithdrawn(ctx context.Context, docID int64, userID int64, source string) error
	UserConsents(ctx context.Context, userID int64) ([]domain.UserConsents, error)
}

type Document struct {
	docRepo docRepo
}

func NewDocument(docRepo docRepo) *Document {
	return &Document{docRepo: docRepo}
}

func (doc *Document) AccessUserDocument(ctx context.Context, userId int64, docID int64, ip *netip.Addr, source string) error {
	if err := doc.docRepo.UserAllow(ctx, docID, &userId, nil, ip, source); err != nil {
		return err
	}
	return nil
}

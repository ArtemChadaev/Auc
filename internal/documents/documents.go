package documents

import (
	"context"
	"net/netip"

	"github.com/ArtemChadaev/Auction/internal/repository"
)

type Document struct {
	docRepo repository.DocumentRepo
}

func NewDocument(docRepo repository.DocumentRepo) *Document {
	return &Document{docRepo: docRepo}
}

func (doc *Document) AccessUserDocument(ctx context.Context, userId int64, docID int64, ip *netip.Addr, source string) error {
	if err := doc.docRepo.UserAllow(ctx, docID, &userId, nil, ip, source); err != nil {
		return err
	}
	return nil
}

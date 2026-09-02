package repository

import (
	"context"
	"errors"

	"github.com/ArtemChadaev/Auction/internal/domain"
	"github.com/jackc/pgx/v5"
)

type DocumentRepo struct {
	db DBTX
}

func NewDocumentRepo(db DBTX) *DocumentRepo {
	return &DocumentRepo{db: db}
}

func (r *DocumentRepo) LastVersion(ctx context.Context, name string) (domain.Document, error) {
	row := r.db.QueryRow(ctx, "select id, name, version, text from documents where name = $1 and published_at <= now() order by published_at DESC limit 1", name)
	var doc domain.Document
	if err := row.Scan(&doc.ID, &doc.Name, &doc.Version, &doc.Text); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Document{}, domain.ErrDocumentNotFound
		}
		return domain.Document{}, err
	}
	return doc, nil
}

package documents

import (
	"context"
	"fmt"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/storage"
	"github.com/jackc/pgx/v5"
)

type Repo struct {
	db storage.DBTX
}

func NewRepo(db storage.DBTX) *Repo {
	return &Repo{db: db}
}

func (r *Repo) getActiveDocuments(ctx context.Context) ([]Document, error) {
	rows, err := r.db.Query(ctx, "SELECT * FROM documents WHERE superseded_at IS NULL ORDER BY id ASC")
	if err != nil {
		return nil, fmt.Errorf("documentRepo.getActiveDocuments: %w", storage.ErrRepo(err))
	}
	var docs []Document
	docs, err = pgx.CollectRows(rows, pgx.RowToStructByName[Document])
	if err != nil {
		return nil, fmt.Errorf("documentRepo.getActiveDocuments: %w", storage.ErrRepo(err))
	}
	return docs, nil
}

func (r *Repo) getDocumentByID(ctx context.Context, id int64) (Document, error) {
	var doc Document
	row, err := r.db.Query(ctx, "SELECT * FROM documents WHERE id = $1", id)
	if err != nil {
		return doc, fmt.Errorf("documentRepo.getDocumentByID(id=%d): %w", id, storage.ErrRepo(err))
	}
	doc, err = pgx.CollectExactlyOneRow(row, pgx.RowToStructByName[Document])
	if err != nil {
		return doc, fmt.Errorf("documentRepo.getDocumentByID(id=%d): %w", id, storage.ErrRepo(err))
	}
	return doc, nil
}

func (r *Repo) getActiveDocumentByName(ctx context.Context, name string) (Document, error) {
	var doc Document
	row, err := r.db.Query(ctx, "SELECT * FROM documents WHERE name = $1 AND superseded_at IS NULL", name)
	if err != nil {
		return doc, fmt.Errorf("documentRepo.getActiveDocumentByName(name=%s): %w", name, storage.ErrRepo(err))
	}
	doc, err = pgx.CollectExactlyOneRow(row, pgx.RowToStructByName[Document])
	if err != nil {
		return doc, fmt.Errorf("documentRepo.getActiveDocumentByName(name=%s): %w", name, storage.ErrRepo(err))
	}
	return doc, nil
}

func (r *Repo) createConsent(ctx context.Context, docID int64, uid uuid.UUID) (UserConsent, error) {
	var consent UserConsent
	row, err := r.db.Query(ctx, `
		INSERT INTO user_consents (document_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (document_id, user_id) DO UPDATE SET created_at = now()
		RETURNING *
	`, docID, uid)
	if err != nil {
		return consent, fmt.Errorf("documentRepo.createConsent(docID=%d, uid=%v): %w", docID, uid, storage.ErrRepo(err))
	}
	consent, err = pgx.CollectExactlyOneRow(row, pgx.RowToStructByName[UserConsent])
	if err != nil {
		return consent, fmt.Errorf("documentRepo.createConsent(docID=%d, uid=%v): %w", docID, uid, storage.ErrRepo(err))
	}
	return consent, nil
}

func (r *Repo) getUserConsents(ctx context.Context, uid uuid.UUID) ([]UserConsent, error) {
	rows, err := r.db.Query(ctx, "SELECT * FROM user_consents WHERE user_id = $1 ORDER BY created_at DESC", uid)
	if err != nil {
		return nil, fmt.Errorf("documentRepo.getUserConsents(uid=%v): %w", uid, storage.ErrRepo(err))
	}
	var consents []UserConsent
	consents, err = pgx.CollectRows(rows, pgx.RowToStructByName[UserConsent])
	if err != nil {
		return nil, fmt.Errorf("documentRepo.getUserConsents(uid=%v): %w", uid, storage.ErrRepo(err))
	}
	return consents, nil
}

func (r *Repo) getConsentByDocumentID(ctx context.Context, docID int64, uid uuid.UUID) (UserConsent, error) {
	var consent UserConsent
	row, err := r.db.Query(ctx, "SELECT * FROM user_consents WHERE document_id = $1 AND user_id = $2", docID, uid)
	if err != nil {
		return consent, fmt.Errorf("documentRepo.getConsentByDocumentID(docID=%d, uid=%v): %w", docID, uid, storage.ErrRepo(err))
	}
	consent, err = pgx.CollectExactlyOneRow(row, pgx.RowToStructByName[UserConsent])
	if err != nil {
		return consent, fmt.Errorf("documentRepo.getConsentByDocumentID(docID=%d, uid=%v): %w", docID, uid, storage.ErrRepo(err))
	}
	return consent, nil
}

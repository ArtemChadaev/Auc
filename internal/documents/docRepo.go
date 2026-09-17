package documents

//
//import (
//	"context"
//	"errors"
//	"net/netip"
//	"uuid"
//
//	"github.com/ArtemChadaev/Auction/internal/domain"
//	"github.com/jackc/pgx/v5"
//)
//
//type DocumentRepo struct {
//	db DBTX
//}
//
//func NewDocumentRepo(db DBTX) *DocumentRepo {
//	return &DocumentRepo{db: db}
//}
//
//func (r *DocumentRepo) LastVersion(ctx context.Context, name string) (domain.Document, error) {
//	row := r.db.QueryRow(ctx, "select id, name, version, text from documents where name = $1 and published_at <= now() order by published_at DESC limit 1", name)
//	var doc domain.Document
//	if err := row.Scan(&doc.ID, &doc.Name, &doc.Version, &doc.Text); err != nil {
//		if errors.Is(err, pgx.ErrNoRows) {
//			return domain.Document{}, domain.ErrDocumentNotFound
//		}
//		return domain.Document{}, err
//	}
//	return doc, nil
//}
//
//func (r *DocumentRepo) AllDocument(ctx context.Context) ([]domain.Document, error) {
//	rows, err := r.db.Query(ctx, "select id, name, version, text from documents")
//	if err != nil {
//		return nil, err
//	}
//	docs, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.Document])
//	if err != nil {
//		return nil, err
//	}
//	if docs == nil {
//		return nil, domain.ErrDocumentNotFound
//	}
//	return docs, nil
//}
//
//func (r *DocumentRepo) UserAllow(ctx context.Context, docID int64, userID *int64, anonID *uuid.UUID, ip *netip.Addr, source string) error {
//	//TODO: Узнать какие будут commandTeg и внести в обработку ошибок, перепроверить, работает только если есть или userID или anonID
//	if userID == nil && anonID == nil {
//		return domain.ErrDocumentNotUserAnonID
//	}
//	_, err := r.db.Exec(ctx, "Insert into user_consents values (default, $1, $2, $3, 'granted', default, $4, $5)", docID, &userID, &anonID, ip, source)
//	return err
//}
//
//func (r *DocumentRepo) UserWithdrawn(ctx context.Context, docID int64, userID int64, source string) error {
//	_, err := r.db.Exec(ctx, "Insert into user_consents values (default, $1, $2, null, 'withdrawn', default, null, $3)", docID, userID, source)
//	return err
//}
//
//func (r *DocumentRepo) UserConsents(ctx context.Context, userID int64) ([]domain.UserConsents, error) {
//	rows, err := r.db.Query(ctx, "select id, user_id, action, granted_at, ip, source from user_consents where user_id = $1", userID)
//	if err != nil {
//		return nil, err
//	}
//	var userConsents []domain.UserConsents
//	userConsents, err = pgx.CollectRows(rows, pgx.RowToStructByName[domain.UserConsents])
//	if err != nil {
//		return nil, err
//	}
//	if userConsents == nil {
//		return nil, domain.ErrDocumentNotFound
//	}
//	return userConsents, nil
//}

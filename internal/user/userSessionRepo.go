package user

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/internal/storage"
	"github.com/jackc/pgx/v5"
)

// TODO: везде писать uid а то непонятно уже что где, только uid->uuid
func (r *Repo) newToken(ctx context.Context, userID uuid.UUID, refreshToken []byte, device Device) error {
	_, err := r.db.Exec(ctx, "INSERT INTO user_sessions(user_id, refresh_token_hash, expires_at, device) values ($1, $2, $3, $4)", userID, refreshToken, time.Now().UTC().Add(time.Duration(cfg.Cfg.ExpiredRefreshToken)*time.Hour*24), device)
	if err != nil {
		return fmt.Errorf("userSessionRepo.newToken(uid=%s): %w", userID, storage.ErrRepo(err))
	}
	return nil
}

func (r *Repo) findCurrentToken(ctx context.Context, refreshToken []byte) (Session, error) {
	var session Session

	row, err := r.db.Query(ctx, "SELECT * FROM user_sessions WHERE refresh_token_hash = $1 AND revoked_at IS NULL ", refreshToken)
	if err != nil {
		return session, fmt.Errorf("userSessionRepo.findCurrentToken: %w", storage.ErrRepo(err))
	}
	session, err = pgx.CollectExactlyOneRow(row, pgx.RowToStructByName[Session])
	if err != nil {
		return session, fmt.Errorf("userSessionRepo.findCurrentToken: %w", storage.ErrRepo(err))
	}
	return session, nil
}

func (r *Repo) findAllTokens(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	rows, err := r.db.Query(ctx, "SELECT * FROM user_sessions WHERE user_id = $1", userID)
	if err != nil {
		return nil, fmt.Errorf("userSessionRepo.findAllTokens(uid=%s): %w", userID, storage.ErrRepo(err))
	}
	var sessions []Session
	sessions, err = pgx.CollectRows(rows, pgx.RowToStructByName[Session])
	if err != nil {
		return nil, fmt.Errorf("userSessionRepo.findAllTokens(uid=%s): %w", userID, storage.ErrRepo(err))
	}
	return sessions, nil
}

func (r *Repo) findAllCurrentTokens(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	rows, err := r.db.Query(ctx, "SELECT * FROM user_sessions WHERE user_id = $1  AND revoked_at IS NULL ", userID)
	if err != nil {
		return nil, fmt.Errorf("userSessionRepo.findAllCurrentTokens(uid=%s): %w", userID, storage.ErrRepo(err))
	}
	var sessions []Session
	sessions, err = pgx.CollectRows(rows, pgx.RowToStructByName[Session])
	if err != nil {
		return nil, fmt.Errorf("userSessionRepo.findAllCurrentTokens(uid=%s): %w", userID, storage.ErrRepo(err))
	}
	return sessions, nil
}
func (r *Repo) revokedToken(ctx context.Context, tokenID int64, uid uuid.UUID) error {
	_, err := r.db.Exec(ctx, "UPDATE user_sessions SET revoked_at = now() WHERE id = $1 AND user_id = $2", tokenID, uid)
	if err != nil {
		return fmt.Errorf("userSessionRepo.revokedToken(uid=%s): %w", uid, storage.ErrRepo(err))
	}
	return nil
}

func (r *Repo) updateToken(ctx context.Context, refreshToken []byte) error {
	_, err := r.db.Exec(ctx, "UPDATE user_sessions SET expires_at = $1 WHERE refresh_token_hash = $2 AND expires_at > now() AND expires_at - INTERVAL '7 days' < now()", time.Now().UTC().Add(time.Duration(cfg.Cfg.ExpiredRefreshToken)*time.Hour*24), refreshToken)
	if err != nil {
		return fmt.Errorf("userSessionRepo.updateToken: %w", storage.ErrRepo(err))
	}
	return nil
}

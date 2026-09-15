package user

import (
	"context"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/jackc/pgx/v5"
)

func (r *Repo) newToken(ctx context.Context, userID uuid.UUID, refreshToken []byte, device Device) error {
	_, err := r.db.Exec(ctx, "INSERT INTO user_sessions(user_id, refresh_token_hash, expires_at, device) values ($1, $2, $3, $4)", userID, refreshToken, time.Now().UTC().Add(time.Duration(cfg.Cfg.ExpiredRefreshToken)*time.Hour*24), device)
	return err
}

func (r *Repo) findToken(ctx context.Context, refreshToken []byte) (Session, error) {
	var session Session

	row, err := r.db.Query(ctx, "SELECT * FROM user_sessions WHERE refresh_token_hash = $1", refreshToken)
	if err != nil {
		return session, err
	}
	session, err = pgx.CollectExactlyOneRow(row, pgx.RowToStructByName[Session])
	if err != nil {
		return session, err
	}
	return session, nil
}

func (r *Repo) findAllTokens(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	rows, err := r.db.Query(ctx, "SELECT * FROM user_sessions WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	var sessions []Session
	sessions, err = pgx.CollectRows(rows, pgx.RowToStructByName[Session])
	return sessions, err
}

func (r *Repo) revokedToken(ctx context.Context, tokenID int64) error {
	_, err := r.db.Exec(ctx, "UPDATE user_sessions SET revoked_at = now() WHERE id = $1", tokenID)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) updateToken(ctx context.Context, tokenID int64) error {
	_, err := r.db.Exec(ctx, "UPDATE user_sessions SET expires_at = $1 WHERE id = $2 AND expires_at > now() AND expires_at - INTERVAL '7 days' < now()", time.Now().UTC().Add(time.Duration(cfg.Cfg.ExpiredRefreshToken)*time.Hour*24), tokenID)
	return err
}

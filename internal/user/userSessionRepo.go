package user

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repo) new(ctx context.Context, userID int64, refreshToken string, expiresAt time.Time, device Device) error {
	_, err := r.db.Exec(ctx, "INSERT INTO user_sessions(user_id, refresh_token_hash, expires_at, device) values ($1, $2, $4, $5)", userID, refreshToken, expiresAt, device)
	return err
}

func (r *Repo) findToken(ctx context.Context, refreshToken string) (userSession, error) {
	var session userSession

	row, err := r.db.Query(ctx, "SELECT (id, user_id, created_at, expires_at, revoked_at, device) FROM user_sessions WHERE revoked_at IS NULL AND refresh_token_hash = $1", refreshToken)
	if err != nil {
		return session, err
	}
	session, err = pgx.CollectExactlyOneRow(row, pgx.RowToStructByName[userSession])
	return session, err
}

func (r *Repo) findAllTokens(ctx context.Context, userID int64) ([]userSession, error) {
	rows, err := r.db.Query(ctx, "SELECT (id, user_id, created_at, expires_at, revoked_at, device) FROM user_sessions WHERE revoked_at IS NULL AND user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	var sessions []userSession
	sessions, err = pgx.CollectRows(rows, pgx.RowToStructByName[userSession])
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
	_, err := r.db.Exec(ctx, "UPDATE user_sessions SET expires_at = $1 WHERE id = $2 AND expires_at > now() AND expires_at - INTERVAL '7 days' < now()", time.Now().UTC().Add(time.Duration(expiredTokens)*time.Hour), tokenID)
	return err
}

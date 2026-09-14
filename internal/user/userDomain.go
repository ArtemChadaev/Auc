package user

import (
	"time"
)

type user struct {
	id        int64
	name      string
	email     string
	deletedAt *time.Time
	balance   int
	hold      int
	created   time.Time
}

type Device struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
}

type userSession struct {
	ID        int64      `db:"id"`
	UserID    int64      `db:"user_id"`
	TokenHash []byte     `db:"refresh_token_hash"`
	CreatedAt time.Time  `db:"created_at"`
	ExpiresAt time.Time  `db:"expires_at"`
	RevokedAt *time.Time `db:"revoked_at"`
	Device    Device     `db:"device"`
}

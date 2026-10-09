package user

import (
	"time"
	"uuid"
)

type User struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	Name      string     `db:"name" json:"name"`
	AvatarID  *uuid.UUID `db:"avatar_id" json:"avatar_id,omitempty"`
	Email     string     `db:"email" json:"email"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
	Balance   int        `db:"balance" json:"balance"`
	Hold      int        `db:"hold" json:"hold"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

type Device struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
}

type Session struct {
	ID        int64      `db:"id" json:"refresh_id"`
	UserID    uuid.UUID  `db:"user_id" json:"-"`
	TokenHash []byte     `db:"refresh_token_hash" json:"-"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	ExpiresAt time.Time  `db:"expires_at" json:"expires_at"`
	RevokedAt *time.Time `db:"revoked_at" json:"revoked_at"`
	Device    Device     `db:"device" json:"device"`
}

type tokenResponds struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"-"`
	Session      Session `json:"refresh_session"`
}

package user

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
	"uuid"
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

func (d Device) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d *Device) Scan(value interface{}) error {
	if value == nil {
		*d = Device{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("invalid type for Device")
	}

	return json.Unmarshal(bytes, d)
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

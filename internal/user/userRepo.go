package user

import (
	"context"
	"uuid"

	"github.com/ArtemChadaev/Auction/internal/storage"
)

type Repo struct {
	db storage.DBTX
}

func NewRepo(db storage.DBTX) *Repo {
	return &Repo{db: db}
}

// return pass or err
func (r *Repo) login(ctx context.Context, email string) (uuid.UUID, string, error) {
	row := r.db.QueryRow(ctx, "select id, password_hash from users where email = $1", email)
	var id uuid.UUID
	var password string
	if err := row.Scan(&id, &password); err != nil {
		return id, password, err
	}
	return id, password, nil
}

func (r *Repo) register(ctx context.Context, id uuid.UUID, name string, email string, password string) error {
	_, err := r.db.Exec(ctx, "insert into users(id, name, email, password_hash) values($1, $2, $3, $4)", id, name, email, password)
	return err
}

func (r *Repo) getEmailForID(ctx context.Context, uid uuid.UUID) (string, error) {
	row := r.db.QueryRow(ctx, "select email from users where id = $1", uid)
	var email string
	if err := row.Scan(&email); err != nil {
		return "", err
	}
	return email, nil
}

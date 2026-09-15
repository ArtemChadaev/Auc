package user

import (
	"context"

	"github.com/ArtemChadaev/Auction/internal/storage"
)

type Repo struct {
	db storage.DBTX
}

func NewRepo(db storage.DBTX) *Repo {
	return &Repo{db: db}
}

// return pass or err
func (r *Repo) login(ctx context.Context, email string) (string, error) {
	row := r.db.QueryRow(ctx, "select password_hash from users where email = $1", email)
	var password string
	if err := row.Scan(&password); err != nil {
		return "", err
	}
	return password, nil
}

// TODO: Херня переделать test register
func (r *Repo) miniRegister(ctx context.Context, name string, email string, password string) error {
	_, err := r.db.Exec(ctx, "insert into users(name, email, password_hash) values($1, $2, $3)", name, email, password)
	return err
}

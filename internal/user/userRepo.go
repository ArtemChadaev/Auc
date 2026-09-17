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
	row := r.db.QueryRow(ctx, "select id, password_hash from users where email = lower($1)", email)
	var id uuid.UUID
	var password string
	err := row.Scan(&id, &password)
	return id, password, err
}

func (r *Repo) register(ctx context.Context, id uuid.UUID, name string, email string, password string) error {
	_, err := r.db.Exec(ctx, "insert into users(id, name, email, password_hash) values($1, $2, lower($3), $4)", id, name, email, password)
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

func (r *Repo) getUser(ctx context.Context, uid uuid.UUID) (User, error) {
	var user User
	row := r.db.QueryRow(ctx, "select id, name, email, deleted_at, balance, hold, created_at from users where id = $1 and deleted_at is null", uid)
	err := row.Scan(&user.Id, &user.Name, &user.Email, &user.DeletedAt, &user.Balance, &user.Hold, &user.CreatedAt)
	return user, err
}

func (r *Repo) patchUserName(ctx context.Context, uid uuid.UUID, name string) error {
	_, err := r.db.Exec(ctx, "update users set name = $1 where id = $2", name, uid)
	return err
}

func (r *Repo) deletedUser(ctx context.Context, uid uuid.UUID) error {
	_, err := r.db.Exec(ctx, "update users set deleted_at = now() where id = $1", uid)
	return err
}

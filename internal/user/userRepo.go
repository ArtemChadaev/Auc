package user

import (
	"context"
	"fmt"
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
	var id uuid.UUID
	var password string
	row := r.db.QueryRow(ctx, "select id, password_hash from users where email = lower($1)", email)
	err := row.Scan(&id, &password)
	if err != nil {
		return id, password, fmt.Errorf("userRepo.login: %w", storage.ErrRepo(err))
	}
	return id, password, nil
}

func (r *Repo) register(ctx context.Context, id uuid.UUID, name string, email string, password string) error {
	_, err := r.db.Exec(ctx, "insert into users(id, name, email, password_hash) values($1, $2, lower($3), $4)", id, name, email, password)
	if err != nil {
		return fmt.Errorf("userRepo.register: %w", storage.ErrRepo(err))
	}
	return nil
}

func (r *Repo) getEmailForID(ctx context.Context, uid uuid.UUID) (string, error) {
	var email string
	row := r.db.QueryRow(ctx, "select email from users where id = $1", uid)
	err := row.Scan(&email)
	if err != nil {
		return "", fmt.Errorf("userRepo.getEmailForID: %w", storage.ErrRepo(err))
	}
	return email, nil
}

func (r *Repo) getUser(ctx context.Context, uid uuid.UUID) (User, error) {
	var user User
	row := r.db.QueryRow(ctx, "select id, name, email, deleted_at, balance, hold, created_at from users where id = $1 and deleted_at is null", uid)
	err := row.Scan(&user.Id, &user.Name, &user.Email, &user.DeletedAt, &user.Balance, &user.Hold, &user.CreatedAt)
	if err != nil {
		return user, fmt.Errorf("userRepo.getUser: %w", storage.ErrRepo(err))
	}
	return user, nil
}

func (r *Repo) patchUserName(ctx context.Context, uid uuid.UUID, name string) error {
	_, err := r.db.Exec(ctx, "update users set name = $1 where id = $2", name, uid)
	if err != nil {
		return fmt.Errorf("userRepo.patchUserName: %w", storage.ErrRepo(err))
	}
	return nil
}

func (r *Repo) deletedUser(ctx context.Context, uid uuid.UUID) error {
	_, err := r.db.Exec(ctx, "update users set deleted_at = now() where id = $1", uid)
	if err != nil {
		return fmt.Errorf("userRepo.deletedUser: %w", storage.ErrRepo(err))
	}
	return nil
}

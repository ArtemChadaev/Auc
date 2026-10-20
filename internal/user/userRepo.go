package user

import (
	"context"
	"fmt"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/storage"
)

type Repo struct {
	db storage.DBTX
}

func NewRepo(db storage.DBTX) *Repo {
	return &Repo{db: db}
}

func (r *Repo) login(ctx context.Context, email string) (uuid.UUID, string, string, error) {
	var id uuid.UUID
	var password, role string
	row := r.db.QueryRow(ctx, "SELECT id, password_hash, role FROM users WHERE email = lower($1) AND deleted_at IS NULL", email)
	err := row.Scan(&id, &password, &role)
	if err != nil {
		return id, password, role, fmt.Errorf("userRepo.login: %w", storage.ErrRepo(err))
	}
	return id, password, role, nil
}

func (r *Repo) register(ctx context.Context, id uuid.UUID, name, email, password string) error {
	_, err := r.db.Exec(ctx, "INSERT INTO users (id, name, email, password_hash) VALUES ($1, $2, lower($3), $4)", id, name, email, password)
	if err != nil {
		return fmt.Errorf("userRepo.register: %w", storage.ErrRepo(err))
	}
	return nil
}

func (r *Repo) getEmailRoleByID(ctx context.Context, uid uuid.UUID) (string, string, error) {
	var email, role string
	row := r.db.QueryRow(ctx, "SELECT email, role FROM users WHERE id = $1 AND deleted_at IS NULL", uid)
	err := row.Scan(&email, &role)
	if err != nil {
		return "", "", fmt.Errorf("userRepo.getEmailByID: %w", storage.ErrRepo(err))
	}
	return email, role, nil
}

func (r *Repo) getUser(ctx context.Context, uid uuid.UUID) (User, error) {
	var user User
	row := r.db.QueryRow(ctx, "SELECT id, name, avatar_id, email, deleted_at, balance, hold, created_at FROM users WHERE id = $1 AND deleted_at IS NULL", uid)
	err := row.Scan(&user.ID, &user.Name, &user.AvatarID, &user.Email, &user.DeletedAt, &user.Balance, &user.Hold, &user.CreatedAt)
	if err != nil {
		return user, fmt.Errorf("userRepo.getUser: %w", storage.ErrRepo(err))
	}
	return user, nil
}

func (r *Repo) patchUserName(ctx context.Context, uid uuid.UUID, name string) error {
	res, err := r.db.Exec(ctx, "UPDATE users SET name = $1 WHERE id = $2 AND deleted_at IS NULL", name, uid)
	if err != nil {
		return fmt.Errorf("userRepo.patchUserName: %w", storage.ErrRepo(err))
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("userRepo.patchUserName: %w", storage.ErrNotFound)
	}
	return nil
}

func (r *Repo) deleteUser(ctx context.Context, uid uuid.UUID) error {
	res, err := r.db.Exec(ctx, "UPDATE users SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL", uid)
	if err != nil {
		return fmt.Errorf("userRepo.deleteUser: %w", storage.ErrRepo(err))
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("userRepo.deleteUser: %w", storage.ErrNotFound)
	}
	return nil
}

func (r *Repo) insertS3Hash(ctx context.Context, hash string) (uuid.UUID, bool, error) {
	var id uuid.UUID
	var isInserted bool
	row := r.db.QueryRow(ctx, `
		INSERT INTO s3 (hash)
		VALUES ($1)
		ON CONFLICT (hash) DO UPDATE SET hash = EXCLUDED.hash
		RETURNING id, (xmax = 0) AS is_inserted
	`, hash)
	err := row.Scan(&id, &isInserted)
	if err != nil {
		return uuid.Nil(), false, fmt.Errorf("userRepo.insertS3Hash: %w", storage.ErrRepo(err))
	}
	return id, isInserted, nil
}

func (r *Repo) updateAvatar(ctx context.Context, uid uuid.UUID, avatarID uuid.UUID) error {
	res, err := r.db.Exec(ctx, "UPDATE users SET avatar_id = $1 WHERE id = $2 AND deleted_at IS NULL", avatarID, uid)
	if err != nil {
		return fmt.Errorf("userRepo.updateAvatar: %w", storage.ErrRepo(err))
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("userRepo.updateAvatar: %w", storage.ErrNotFound)
	}
	return nil
}

func (r *Repo) checkUserByEmail(ctx context.Context, email string) (bool, error) {
	res, err := r.db.Exec(ctx, "SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL", email)
	if err != nil {
		return false, fmt.Errorf("userRepo.getUserByEmail: %w", storage.ErrRepo(err))
	}
	if res.RowsAffected() == 0 {
		return false, fmt.Errorf("userRepo.getUserByEmail: %w", storage.ErrNotFound)
	}
	return true, nil
}

func (r *Repo) setAdmin(ctx context.Context, uid uuid.UUID) error {
	res, err := r.db.Exec(ctx, "UPDATE users SET role = 'admin' WHERE id = $1 AND deleted_at IS NULL", uid)
	if err != nil {
		return fmt.Errorf("userRepo.setAdmin: %w", storage.ErrRepo(err))
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("userRepo.setAdmin: %w", storage.ErrNotFound)
	}
	return nil
}

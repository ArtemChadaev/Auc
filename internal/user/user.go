package user

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

type repo interface {
	login(ctx context.Context, email string) (string, error)
	miniRegister(ctx context.Context, name string, email string, password string) error

	newToken(ctx context.Context, userID int64, refreshToken string, expiresAt time.Time, device Device) error
	findToken(ctx context.Context, refreshToken string) (userSession, error)
	findAllTokens(ctx context.Context, userID int64) ([]userSession, error)
	revokedToken(ctx context.Context, tokenID int64) error
	updateToken(ctx context.Context, tokenID int64) error
}

type Service struct {
	repo repo
}

func NewService(repo repo) *Service {
	return &Service{repo: repo}
}

const (
	SaltLength  = 16
	Time        = 1
	Memory      = 64 * 1024
	Parallelism = 4
	KeyLength   = 32
)

func createPassword(password string) (string, error) {
	salt := make([]byte, SaltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, Time, Memory, Parallelism, KeyLength)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64PassHash := base64.RawStdEncoding.EncodeToString(hash)
	pass := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%s$%s$%s", argon2.Version, Memory, Time, Parallelism, b64Salt, b64PassHash)
	return pass, nil
}

func checkPassword(password, passHash string) error {
	parts := strings.Split(passHash, "$")
	if len(parts) != 6 {
		return fmt.Errorf("invalid passwordHash")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return err
	}
	pass, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return err
	}
	passDB := argon2.IDKey([]byte(password), salt, Time, Memory, Parallelism, KeyLength)

	if subtle.ConstantTimeCompare(passDB, pass) != 1 {
		return errors.New("invalid password")
	}
	return nil
}

func (s *Service) Register(ctx context.Context, name string, email string, password string) error {
	pass, err := createPassword(password)
	if err != nil {
		return err
	}
	return s.repo.miniRegister(ctx, name, email, pass)
}

func (s *Service) Auth(ctx context.Context, email string, password string) error {
	passHash, err := s.repo.login(ctx, email)
	if err != nil {
		return err
	}
	if err = checkPassword(passHash, password); err != nil {
		return err
	}
	return nil
}

package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"golang.org/x/crypto/argon2"
)

type repo interface {
	login(ctx context.Context, email string) (uuid.UUID, string, error)
	register(ctx context.Context, id uuid.UUID, name string, email string, password string) error
	getEmailForID(ctx context.Context, uid uuid.UUID) (string, error)

	newToken(ctx context.Context, userID uuid.UUID, refreshToken []byte, device Device) error
	findCurrentToken(ctx context.Context, refreshToken []byte) (Session, error)
	findAllTokens(ctx context.Context, userID uuid.UUID) ([]Session, error)
	findAllCurrentTokens(ctx context.Context, userID uuid.UUID) ([]Session, error)
	revokedToken(ctx context.Context, tokenID int64, uid uuid.UUID) error
	updateToken(ctx context.Context, refreshToken []byte) error
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
	pass := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, Memory, Time, Parallelism, b64Salt, b64PassHash)
	return pass, nil
}

func checkPassword(passHash, password string) error {
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

func (s *Service) register(ctx context.Context, uid *uuid.UUID, name string, email string, password string, device Device) (Tokens, error) {
	var tokens Tokens

	pass, err := createPassword(password)
	if err != nil {
		return tokens, err
	}

	if uid == nil {
		id := uuid.NewV7()
		uid = &id
	}
	err = s.repo.register(ctx, *uid, name, email, pass)
	if err != nil {
		return tokens, err
	}

	tokens, err = createTokens(email, *uid)
	if err != nil {
		return tokens, err
	}

	sha := sha256.Sum256([]byte(tokens.RefreshToken))
	if err = s.repo.newToken(ctx, *uid, sha[:], device); err != nil {
		return tokens, err
	}
	return tokens, nil
}

func (s *Service) authPassword(ctx context.Context, email string, password string, device Device) (Tokens, error) {
	var tokens Tokens

	uid, passHash, err := s.repo.login(ctx, email)
	if err != nil {
		return tokens, err
	}
	if err = checkPassword(passHash, password); err != nil {
		return tokens, err
	}

	tokens, err = createTokens(email, uid)
	if err != nil {
		return tokens, err
	}

	sha := sha256.Sum256([]byte(tokens.RefreshToken))
	if err = s.repo.newToken(ctx, uid, sha[:], device); err != nil {
		return tokens, err
	}
	return tokens, nil
}

func (s *Service) authRefresh(ctx context.Context, refresh string) (Tokens, error) {
	sha := sha256.Sum256([]byte(refresh))
	userSession, err := s.repo.findCurrentToken(ctx, sha[:])
	if err != nil {
		return Tokens{}, err
	}
	if userSession.RevokedAt != nil {
		return Tokens{}, errors.New("refresh token is revoked")
	}
	// Токен закончится раньше чем через 7 дней
	if userSession.ExpiresAt.Before(time.Now().UTC().Add(7 * 24 * time.Hour)) {
		// Если произошла ошибка то пропускаем, если нет то еще раз его получаем, TODO: ПЕРЕПРОВЕРИТЬ ВЫГЛЯДИТ ОПАСНО
		if err = s.repo.updateToken(ctx, sha[:]); err != nil {
			return s.authRefresh(ctx, refresh)
		}
	}
	email, err := s.repo.getEmailForID(ctx, userSession.UserID)
	if err != nil {
		return Tokens{}, err
	}
	return createTokens(email, userSession.UserID)
}

func (s *Service) tokenResponds(ctx context.Context, refresh string) (Session, error) {
	sha := sha256.Sum256([]byte(refresh))
	return s.repo.findCurrentToken(ctx, sha[:])
}

func (s *Service) logout(ctx context.Context, refreshId []int64, uid uuid.UUID) error {
	var err error
	for _, id := range refreshId {
		err = s.repo.revokedToken(ctx, id, uid)
	}
	return err
}

func (s *Service) findTokens(ctx context.Context, uid uuid.UUID, current bool) ([]Session, error) {
	if current {
		return s.repo.findAllCurrentTokens(ctx, uid)
	}
	return s.repo.findAllTokens(ctx, uid)
}

package user

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/cmd/storageS3"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

var (
	ErrImageNotSquare = apperr.NewAppErrorString("image must be square", -4)
	ErrInvalidImage   = apperr.NewAppErrorString("invalid image data", -4)
)

type Service struct {
	repo *Repo
	s3   *storageS3.Client
}

func NewService(repo *Repo, s3 *storageS3.Client) *Service {
	return &Service{
		repo: repo,
		s3:   s3,
	}
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
		return "", fmt.Errorf(`userService.createPassword: %v`, apperr.NewAppError(err, 4))
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
		return apperr.NewAppErrorString("userService.checkPassword", 4)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return apperr.NewAppErrorString("userService.checkPassword", 4)
	}
	pass, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return apperr.NewAppErrorString("userService.checkPassword", 4)
	}
	passDB := argon2.IDKey([]byte(password), salt, Time, Memory, Parallelism, KeyLength)

	if subtle.ConstantTimeCompare(passDB, pass) != 1 {
		return apperr.NewAppErrorString("userService.checkPassword", -4)
	}
	return nil
}

func (s *Service) register(ctx context.Context, uid *uuid.UUID, name string, email string, password string, device Device) (Tokens, error) {
	var tokens Tokens

	pass, err := createPassword(password)
	if err != nil {
		return tokens, fmt.Errorf(`userService.register: %w`, err)
	}

	if uid == nil {
		id := uuid.NewV7()
		uid = &id
	}
	err = s.repo.register(ctx, *uid, name, email, pass)
	if err != nil {
		return tokens, fmt.Errorf("userService.register: %w", err)
	}

	tokens, err = createTokens(email, *uid, "user")
	if err != nil {
		return tokens, fmt.Errorf("userService.register: %w", err)
	}

	sha := sha256.Sum256([]byte(tokens.RefreshToken))
	if err = s.repo.newToken(ctx, *uid, sha[:], device); err != nil {
		return tokens, fmt.Errorf("userService.register: %w", err)
	}
	return tokens, nil
}

func (s *Service) authPassword(ctx context.Context, email string, password string, device Device) (Tokens, error) {
	var tokens Tokens

	uid, passHash, role, err := s.repo.login(ctx, email)
	if err != nil {
		return tokens, fmt.Errorf("userService.authPassword: %w", err)
	}
	if err = checkPassword(passHash, password); err != nil {
		return tokens, fmt.Errorf("userService.authPassword: %w", err)
	}

	tokens, err = createTokens(email, uid, role)
	if err != nil {
		return tokens, fmt.Errorf("userService.authPassword: %w", err)
	}

	sha := sha256.Sum256([]byte(tokens.RefreshToken))
	if err = s.repo.newToken(ctx, uid, sha[:], device); err != nil {
		return tokens, fmt.Errorf("userService.authPassword: %w", err)
	}
	return tokens, nil
}

func (s *Service) authRefresh(ctx context.Context, refresh string) (Tokens, error) {
	sha := sha256.Sum256([]byte(refresh))
	userSession, err := s.repo.findCurrentToken(ctx, sha[:])
	if err != nil {
		return Tokens{}, fmt.Errorf("userService.authRefresh: %w", err)
	}
	// Токен закончится раньше чем через 7 дней
	if userSession.ExpiresAt.Before(time.Now().UTC().Add(7 * 24 * time.Hour)) {
		if userSession, err = s.repo.updateToken(ctx, sha[:]); err != nil {
			return Tokens{}, fmt.Errorf("userService.authRefresh: %w", err)
		}
	}
	email, role, err := s.repo.getEmailRoleByID(ctx, userSession.UserID)
	if err != nil {
		return Tokens{}, fmt.Errorf("userService.authRefresh: %w", err)
	}
	accessToken, err := generateAccessToken(email, userSession.UserID, role)
	if err != nil {
		return Tokens{}, fmt.Errorf("userService.authRefresh: %w", err)
	}
	return Tokens{
		AccessToken:  accessToken,
		RefreshToken: refresh,
	}, nil
}

func (s *Service) tokenResponds(ctx context.Context, refresh string) (Session, error) {
	sha := sha256.Sum256([]byte(refresh))
	userSession, err := s.repo.findCurrentToken(ctx, sha[:])
	if err != nil {
		return userSession, fmt.Errorf("userService.tokenResponds: %w", err)
	}
	return userSession, nil
}

func (s *Service) logout(ctx context.Context, refreshId []int64, uid uuid.UUID) error {
	var err error
	for _, id := range refreshId {
		if err = s.repo.revokedToken(ctx, id, uid); err != nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("userService.logout: %w", err)
	}
	return nil
}

func (s *Service) findTokens(ctx context.Context, uid uuid.UUID, current bool) ([]Session, error) {
	if current {
		sessions, err := s.repo.findAllCurrentTokens(ctx, uid)
		if err != nil {
			return nil, fmt.Errorf("userService.findTokens: %w", err)
		}
		return sessions, nil
	}
	sessions, err := s.repo.findAllTokens(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("userService.findTokens: %w", err)
	}
	return sessions, nil
}

func (s *Service) getUser(ctx context.Context, uid uuid.UUID) (User, error) {
	user, err := s.repo.getUser(ctx, uid)
	if err != nil {
		return User{}, fmt.Errorf("userService.getUser: %w", err)
	}
	return user, nil
}

func (s *Service) patchUserName(ctx context.Context, uid uuid.UUID, name string) error {
	if err := s.repo.patchUserName(ctx, uid, name); err != nil {
		return fmt.Errorf("userService.patchUserName: %w", err)
	}
	return nil
}

func (s *Service) deleteUser(ctx context.Context, uid uuid.UUID) error {
	if err := s.repo.deleteUser(ctx, uid); err != nil {
		return fmt.Errorf("userService.deleteUser: %w", err)
	}
	return nil
}

func (s *Service) updateAvatar(ctx context.Context, uid uuid.UUID, r io.Reader) (uuid.UUID, error) {
	fileBytes, err := io.ReadAll(r)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("userService.updateAvatar: read file: %w", err)
	}

	// 1. Проверяем, что изображение квадратное
	cfg, _, err := image.DecodeConfig(bytes.NewReader(fileBytes))
	if err != nil {
		return uuid.Nil(), fmt.Errorf("userService.updateAvatar: %w", ErrInvalidImage)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width != cfg.Height {
		return uuid.Nil(), fmt.Errorf("userService.updateAvatar: %w", ErrImageNotSquare)
	}

	// 2. Используем ffmpeg для масштабирования в 256x256 (если был больше) и кодирования в WebP
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", "pipe:0",
		"-vf", "scale='min(256,iw)':'min(256,ih)'",
		"-c:v", "libwebp",
		"-f", "webp",
		"pipe:1",
	)
	cmd.Stdin = bytes.NewReader(fileBytes)
	var out bytes.Buffer
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		return uuid.Nil(), fmt.Errorf("userService.updateAvatar: ffmpeg transcode failed: %w, stderr: %s", apperr.NewAppError(err, 4), stderr.String())
	}
	webpBytes := out.Bytes()

	// 3. Генерируем SHA-256 хеш полученного WebP
	sum := sha256.Sum256(webpBytes)
	hash := hex.EncodeToString(sum[:])

	// 4. Запрос в userRepo: insertS3Hash с RETURNING id, (xmax = 0) AS is_inserted
	s3ID, isInserted, err := s.repo.insertS3Hash(ctx, hash)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("userService.updateAvatar: %w", err)
	}

	// 5. Если до этого хеша в S3 не было, загружаем с "image/webp" и key = id
	if isInserted {
		if s.s3 == nil {
			return uuid.Nil(), fmt.Errorf("userService.updateAvatar: s3 client is not configured")
		}
		if err = s.s3.Upload(ctx, s3ID.String(), bytes.NewReader(webpBytes), "image/webp"); err != nil {
			return uuid.Nil(), fmt.Errorf("userService.updateAvatar: %w", err)
		}
	}

	// 6. Как загрузится (или если уже было), обновляем в БД строку пользователя
	if err = s.repo.updateAvatar(ctx, uid, s3ID); err != nil {
		return uuid.Nil(), fmt.Errorf("userService.updateAvatar: %w", err)
	}

	return s3ID, nil
}

func (s *Service) setAdmin(ctx context.Context, uid uuid.UUID) error {
	err := s.repo.setAdmin(ctx, uid)
	if err != nil {
		return fmt.Errorf("userService.setAdmin: %w", err)
	}
	return nil
}

func (s *Service) checkUserByEmail(ctx context.Context, email string) (bool, error) {
	check, err := s.repo.checkUserByEmail(ctx, email)
	if err != nil {
		return check, fmt.Errorf("userService.getUserByEmail: %w", err)
	}
	return check, nil
}

func (s *Service) CreateAdmin() error {
	ctx := context.Background()
	check, err := s.checkUserByEmail(ctx, cfg.Cfg.AdminEmail)
	if check && err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("userService.CreateAdmin: %w", err)
	}

	uid := uuid.New()
	tokens, err := s.register(ctx, &uid, "Admin", cfg.Cfg.AdminEmail, cfg.Cfg.AdminPassword, Device{})
	if err != nil {
		return fmt.Errorf("userService.CreateAdmin: %w", err)
	}
	if err = s.setAdmin(ctx, uid); err != nil {
		return fmt.Errorf("userService.CreateAdmin: %w", err)
	}
	slog.Info("create new user", slog.String("uid", uid.String()), slog.Any("tokens", tokens))
	return nil
}

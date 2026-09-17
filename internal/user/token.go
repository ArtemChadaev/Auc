package user

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type Tokens struct {
	AccessToken  string
	RefreshToken string
}

func generateAccessToken(email string, uuid uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"iss":  cfg.Cfg.Domain,
		"sub":  email,
		"exp":  time.Now().Add(time.Minute * time.Duration(cfg.Cfg.ExpiredAccessToken)).Unix(),
		"iat":  time.Now().Unix(),
		"uuid": uuid.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	accessToken, err := token.SignedString([]byte(cfg.Cfg.JwtSecret))
	if err != nil {
		return "", fmt.Errorf("generate access token(%w): %s", apperr.ErrWarn, err)
	}
	return accessToken, nil
}

func keyFunc() jwt.Keyfunc {
	return func(_ *jwt.Token) (interface{}, error) { return []byte(cfg.Cfg.JwtSecret), nil }
}

// VerifyAccessToken uuid, error
func VerifyAccessToken(accessToken string) (uuid.UUID, error) {
	token, err := jwt.Parse(
		accessToken,
		keyFunc(),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return [16]byte{}, fmt.Errorf("user.VerfiAccessToken: %w", ErrInvalidToken)
	}
	if !token.Valid {
		return [16]byte{}, fmt.Errorf("user.VerfiAccessToken: %w", ErrInvalidToken)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return [16]byte{}, fmt.Errorf("user.VerfiAccessToken: %w", ErrInvalidToken)
	}

	uid, err := uuid.Parse(claims["uuid"].(string))
	if err != nil {
		return [16]byte{}, fmt.Errorf("user.VerfiAccessToken(%w): %s", apperr.ErrWarn, err)
	}
	return uid, nil
}

func createTokens(email string, uuid uuid.UUID) (Tokens, error) {
	var tokens Tokens
	var err error
	tokens.AccessToken, err = generateAccessToken(email, uuid)
	if err != nil {
		return tokens, fmt.Errorf("user.createTokens: %w", err)
	}
	rt := make([]byte, 32)
	if _, err = rand.Read(rt); err != nil {
		return tokens, fmt.Errorf("user.createTokens(%w): %s", apperr.ErrWarn, err)
	}
	tokens.RefreshToken = hex.EncodeToString(rt)
	return tokens, nil
}

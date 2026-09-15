package user

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/golang-jwt/jwt/v5"
)

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

	return token.SignedString([]byte(cfg.Cfg.JwtSecret))
}

func createTokens(email string, uuid uuid.UUID) (Tokens, error) {
	var tokens Tokens
	var err error
	tokens.AccessToken, err = generateAccessToken(email, uuid)
	if err != nil {
		return tokens, err
	}
	rt := make([]byte, 32)
	if _, err = rand.Read(rt); err != nil {
		return tokens, err
	}
	tokens.RefreshToken = hex.EncodeToString(rt)
	return tokens, nil
}

// email, error
func verifyAccessToken(accessToken string) (uuid.UUID, error) {
	token, err := jwt.Parse(
		accessToken,
		func(token *jwt.Token) (interface{}, error) { return []byte(cfg.Cfg.JwtSecret), nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return [16]byte{}, err
	}
	if !token.Valid {
		return [16]byte{}, errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return [16]byte{}, errors.New("invalid token")
	}

	return uuid.Parse(claims["uuid"].(string))
}

package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/cmd/httpx"
	"github.com/ArtemChadaev/Auction/cmd/storage"
	"github.com/ArtemChadaev/Auction/cmd/valkey"
	"github.com/ArtemChadaev/Auction/internal/user"
	"github.com/justinas/alice"
)

func authenticate(r *http.Request, vk *valkey.Client, db storage.DBTX) (uuid.UUID, string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return uuid.Nil(), "", user.ErrInvalidToken
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return uuid.Nil(), "", user.ErrInvalidToken
	}
	uid, role, err := user.VerifyAccessToken(parts[1])
	if err != nil {
		return uuid.Nil(), "", err
	}

	isDeleted, err := user.CheckUserDeleted(r.Context(), vk, db, uid)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return uuid.Nil(), "", user.ErrInvalidToken
		}
		return uuid.Nil(), "", err
	}
	if isDeleted {
		return uuid.Nil(), "", user.ErrUserDeleted
	}

	return uid, role, nil
}

func AuthAccessToken(vk *valkey.Client, db storage.DBTX) alice.Constructor {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid, role, err := authenticate(r, vk, db)
			if err != nil {
				if errors.Is(err, user.ErrInvalidToken) || errors.Is(err, user.ErrUserDeleted) {
					httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
				} else {
					apperr.Log(r.Context(), "AuthAccessToken", err)
					httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
				}
				return
			}

			ctx := context.WithValue(r.Context(), cfg.UID, uid)
			ctx = context.WithValue(ctx, cfg.Role, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AuthAdmin(vk *valkey.Client, db storage.DBTX) alice.Constructor {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid, role, err := authenticate(r, vk, db)
			if err != nil {
				if errors.Is(err, user.ErrInvalidToken) || errors.Is(err, user.ErrUserDeleted) {
					httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
				} else {
					apperr.Log(r.Context(), "AuthAdmin", err)
					httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
				}
				return
			}
			if role != "admin" {
				httpx.WriteResponse(w, httpx.ErrForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), cfg.UID, uid)
			ctx = context.WithValue(ctx, cfg.Role, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

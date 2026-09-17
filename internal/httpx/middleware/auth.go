package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/internal/httpx"
	"github.com/ArtemChadaev/Auction/internal/user"
)

func AuthAccessToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, httpx.InvalidToken, http.StatusUnauthorized)
			slog.InfoContext(r.Context(), httpx.InvalidToken)
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, httpx.InvalidToken, http.StatusUnauthorized)
			slog.InfoContext(r.Context(), httpx.InvalidToken)
			return
		}
		uid, err := user.VerifyAccessToken(parts[1])
		if err != nil {
			if errors.Is(err, user.ErrInvalidToken) {
				http.Error(w, httpx.InvalidToken, http.StatusUnauthorized)
				slog.InfoContext(r.Context(), httpx.InvalidToken, slog.Any("error", err))
			} else {
				http.Error(w, httpx.InternalServer, http.StatusInternalServerError)
				slog.WarnContext(r.Context(), httpx.InternalServer, slog.Any("error", err))
			}
			return
		}

		ctx := context.WithValue(r.Context(), cfg.UID, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

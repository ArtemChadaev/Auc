package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/internal/user"
)

func AuthAccessToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "err", http.StatusUnauthorized)
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Неверный формат заголовка (ожидается Bearer <token>)", http.StatusUnauthorized)
			return
		}
		id, err := user.VerifyAccessToken(parts[1])
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		//TODO: Переделать контекст
		ctx := context.WithValue(r.Context(), cfg.UID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

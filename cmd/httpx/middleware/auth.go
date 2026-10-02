package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/cmd/httpx"
	"github.com/ArtemChadaev/Auction/internal/user"
)

func AuthAccessToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
			return
		}
		uid, err := user.VerifyAccessToken(parts[1])
		if err != nil {
			if errors.Is(err, user.ErrInvalidToken) {
				httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
			} else {
				apperr.Log(r.Context(), "", err)
				httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
			}
			return
		}

		ctx := context.WithValue(r.Context(), cfg.UID, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

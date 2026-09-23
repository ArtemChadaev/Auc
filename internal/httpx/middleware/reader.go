package middleware

import (
	"log/slog"
	"net/http"

	"github.com/ArtemChadaev/Auction/internal/httpx"
	"github.com/justinas/alice"
)

func MaxBodySize(limit int64) alice.Constructor {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > limit {
				slog.DebugContext(r.Context(), "max body size exceeded")
				http.Error(w, httpx.ReqEntityTooLarge, http.StatusRequestEntityTooLarge)
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, limit)

			next.ServeHTTP(w, r)
		})
	}
}

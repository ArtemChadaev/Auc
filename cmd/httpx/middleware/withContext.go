package middleware

import (
	"context"
	"net"
	"net/http"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			ctx = context.WithValue(ctx, cfg.IP, ip)
		}
		ctx = context.WithValue(ctx, cfg.Path, r.Method+" "+r.URL.Path)

		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

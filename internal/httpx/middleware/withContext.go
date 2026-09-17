package middleware

import (
	"context"
	"net"
	"net/http"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			context.WithValue(r.Context(), cfg.IP, ip)
		}
		context.WithValue(r.Context(), cfg.Path, r.Method+" "+r.URL.Path)

		next.ServeHTTP(w, r)
	})
}

package middleware

import (
	"net/http"
)

func handleRequest(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<18)
}

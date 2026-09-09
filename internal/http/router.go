package handler

import "net/http"

func Router(doc DocHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("")

	return mux
}

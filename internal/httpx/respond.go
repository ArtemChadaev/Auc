package httpx

import (
	"encoding/json"
	"net/http"
)

// Переделать перепроверить
func WriteJSON(w http.ResponseWriter, status int, data any) {
	js, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Error response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(js)
}

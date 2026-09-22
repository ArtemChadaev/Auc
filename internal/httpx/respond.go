package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
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

// DecodeJSON декодирует и обрезает пробелы слева/справа
func DecodeJSON[T any](r *http.Request) (T, error) {
	var req T
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return req, fmt.Errorf("respond.DecodeJSON(%w): %w", apperr.ErrDebug, ErrReqBody)
	}
	val := reflect.ValueOf(&req).Elem()
	if val.Kind() != reflect.Struct {
		return req, fmt.Errorf("respond.DecodeJSON(%w): %w", apperr.ErrDebug, ErrReqBody)
	}
	for _, field := range val.Fields() {
		if !field.CanSet() {
			continue
		}

		if field.Kind() == reflect.String {
			strVal := field.String()
			trimmed := strings.TrimSpace(strVal)
			field.SetString(trimmed)
		}
	}
	return req, nil
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	err = fmt.Errorf("respond.WriteError: %w", err)
	if errors.Is(err, apperr.ErrDebug) {
		slog.DebugContext(r.Context(), "Error response", "error", err)
		http.Error(w, InvalidReqBody, http.StatusBadRequest)
		return
	}
	if errors.Is(err, apperr.ErrWarn) {
		slog.WarnContext(r.Context(), "Error app", "error", err)
		http.Error(w, InternalServer, http.StatusInternalServerError)
		return
	}
	if errors.Is(err, apperr.ErrError) {
		slog.ErrorContext(r.Context(), "Error app", "error", err)
		http.Error(w, InternalServer, http.StatusInternalServerError)
		return
	}
	slog.WarnContext(r.Context(), "Error app don't validate", "error", err)
	http.Error(w, InternalServer, http.StatusInternalServerError)
	return
}

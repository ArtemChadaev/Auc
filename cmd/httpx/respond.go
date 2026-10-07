package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
)

type Response struct {
	Code  int    `json:"-"`
	Error string `json:"error,omitempty"`
	Data  any    `json:"data,omitempty"`
}

var (
	ErrRespNotFound          = Response{Code: http.StatusNotFound, Error: "not found"}
	ErrRespInvalidAuth       = Response{Code: http.StatusUnauthorized, Error: "invalid authorization"}
	ErrRespInternalServer    = Response{Code: http.StatusInternalServerError, Error: "internal server error"}
	ErrRespFieldIsTaken      = Response{Code: http.StatusConflict, Error: "field is taken"}
	ErrRespMarshal           = Response{Code: http.StatusInternalServerError, Error: "failed to marshal response"}
	ErrRespInvalidReqBody    = Response{Code: http.StatusBadRequest, Error: "invalid request body"}
	ErrRespReqEntityTooLarge = Response{Code: http.StatusRequestEntityTooLarge, Error: "request entity too large"}

	// Upload & Items
	ErrRespHashAlreadyExists    = Response{Code: http.StatusConflict, Error: "hash already exists"}
	ErrRespFileAlreadyExists    = Response{Code: http.StatusConflict, Error: "file already exists"}
	ErrRespUserAlreadyUploading = Response{Code: http.StatusConflict, Error: "user already has an active upload"}
	ErrRespFileExpired          = Response{Code: http.StatusGone, Error: "file has expired"}
	ErrRespDownloadLimit        = Response{Code: http.StatusTooManyRequests, Error: "download limit reached: only 1 download per day allowed"}
)

// DecodeJSON декодирует и обрезает пробелы слева/справа. Сам отправляет ошибку клиенту
func DecodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var req T
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.Is(err, maxBytesErr) {
			WriteResponse(w, ErrRespReqEntityTooLarge)
		} else {
			WriteResponse(w, ErrRespInvalidReqBody)
		}
		return req, fmt.Errorf("respond.DecodeJSON: %v", apperr.NewAppError(err, 4))
	}
	val := reflect.ValueOf(&req).Elem()
	if val.Kind() != reflect.Struct {
		WriteResponse(w, ErrRespInvalidReqBody)
		return req, fmt.Errorf("respond.DecodeJSON: %w", apperr.NewAppErrorString("not struct", 4))
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

func WriteResponse(w http.ResponseWriter, resp Response) {
	js, err := json.Marshal(resp)
	if err != nil {
		WriteResponse(w, ErrRespMarshal)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Code)
	_, _ = w.Write(js)
}

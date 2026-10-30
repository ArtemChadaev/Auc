package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	ErrRespInvalidReqBody    = Response{Code: http.StatusBadRequest, Error: "invalid request body"}
	ErrRespReqEntityTooLarge = Response{Code: http.StatusRequestEntityTooLarge, Error: "request entity too large"}
	ErrForbidden             = Response{Code: http.StatusForbidden, Error: "forbidden"}

	// Upload & Items
	ErrRespHashAlreadyExists    = Response{Code: http.StatusConflict, Error: "hash already exists"}
	ErrRespFileAlreadyExists    = Response{Code: http.StatusConflict, Error: "file already exists"}
	ErrRespUserAlreadyUploading = Response{Code: http.StatusConflict, Error: "user already has an active upload"}
	ErrRespFileExpired          = Response{Code: http.StatusGone, Error: "file has expired"}
	ErrRespDownloadLimit        = Response{Code: http.StatusTooManyRequests, Error: "download limit reached: only 1 download per day allowed"}

	// User Limits
	ErrRespFreeCoinLimit    = Response{Code: http.StatusTooManyRequests, Error: "free coins limit reached: only once per day allowed"}
	ErrRespPhotoUploadLimit = Response{Code: http.StatusTooManyRequests, Error: "photo upload limit reached: maximum 30 photos per hour allowed"}
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
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		WriteResponse(w, ErrRespInvalidReqBody)
		return req, fmt.Errorf("respond.DecodeJSON: %w", apperr.NewAppErrorString("unexpected data after JSON", 4))
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

	type checker interface {
		Check() error
	}

	if reqCheck, ok := any(&req).(checker); ok {
		if err := reqCheck.Check(); err != nil {
			WriteResponse(w, ErrRespInvalidReqBody)
			return req, fmt.Errorf("respond.DecodeJSON: %w", apperr.NewAppError(err, -4))
		}
	}

	return req, nil
}

func WriteResponse(w http.ResponseWriter, resp Response) {
	js, err := json.Marshal(resp)
	if err != nil {
		WriteResponse(w, Response{Code: http.StatusInternalServerError, Error: "failed to marshal response"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Code)
	_, _ = w.Write(js)
}

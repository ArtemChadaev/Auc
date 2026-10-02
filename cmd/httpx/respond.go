package httpx

import (
	"encoding/json"
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

var ErrRespMarshal = Response{Code: http.StatusInternalServerError, Error: "failed to marshal response"}

// DecodeJSON декодирует и обрезает пробелы слева/справа
func DecodeJSON[T any](r *http.Request) (T, error) {
	var req T
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return req, fmt.Errorf("respond.DecodeJSON: %v", apperr.NewAppError(err, 4))
	}
	val := reflect.ValueOf(&req).Elem()
	if val.Kind() != reflect.Struct {
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

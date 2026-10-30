package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/cmd/httpx"
	"github.com/ArtemChadaev/Auction/cmd/valkey"
	"github.com/justinas/alice"
	"github.com/redis/go-redis/v9"
)

const (
	StatusProcessing = "PROCESSING"
	StatusCompleted  = "COMPLETED"

	idempotencyProcessingTTL = 10 * time.Second
	idempotencyCompletedTTL  = 10 * time.Second
)

type IdempotencyRecord struct {
	UserID     string              `json:"user_id"`
	Key        string              `json:"key"`
	Status     string              `json:"status"`
	StatusCode int                 `json:"status_code,omitempty"`
	Headers    map[string][]string `json:"headers,omitempty"`
	Body       []byte              `json:"body,omitempty"`
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	body        bytes.Buffer
	wroteHeader bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rec *responseRecorder) WriteHeader(code int) {
	if !rec.wroteHeader {
		rec.statusCode = code
		rec.wroteHeader = true
		rec.ResponseWriter.WriteHeader(code)
	}
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	if !rec.wroteHeader {
		rec.WriteHeader(http.StatusOK)
	}
	rec.body.Write(b)
	return rec.ResponseWriter.Write(b)
}

func Idempotency(vk *valkey.Client) alice.Constructor {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
			if idempotencyKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			if vk == nil || vk.Raw() == nil {
				next.ServeHTTP(w, r)
				return
			}
			rdb := vk.Raw()

			uid, err := cfg.GetUID(r.Context())
			if err != nil {
				apperr.Log(r.Context(), "", apperr.NewAppErrorString("no auth", -4))
				next.ServeHTTP(w, r)
				return
			}
			uidStr := uid.String()

			redisKey := fmt.Sprintf("idempotency:%s:%s", uidStr, idempotencyKey)

			val, err := rdb.Get(r.Context(), redisKey).Result()
			if err == nil {
				var record IdempotencyRecord
				if err := json.Unmarshal([]byte(val), &record); err != nil {
					apperr.Log(r.Context(), "idempotency.unmarshal", err)
					httpx.WriteResponse(w, httpx.ErrRespInternalServer)
					return
				}

				switch record.Status {
				case StatusProcessing:
					httpx.WriteResponse(w, httpx.Response{
						Code:  http.StatusConflict,
						Error: "request is already processing",
					})
					return
				case StatusCompleted:
					maps.Copy(w.Header(), record.Headers)
					statusCode := record.StatusCode
					if statusCode == 0 {
						statusCode = http.StatusOK
					}
					w.WriteHeader(statusCode)
					_, _ = w.Write(record.Body)
					slog.DebugContext(r.Context(), "apply idempotency")
					return
				default:
					httpx.WriteResponse(w, httpx.ErrRespInternalServer)
					return
				}
			}

			if !errors.Is(err, redis.Nil) {
				apperr.Log(r.Context(), "idempotency.get", err)
				httpx.WriteResponse(w, httpx.ErrRespInternalServer)
				return
			}

			initRecord := IdempotencyRecord{
				UserID: uidStr,
				Key:    idempotencyKey,
				Status: StatusProcessing,
			}
			initData, err := json.Marshal(initRecord)
			if err != nil {
				apperr.Log(r.Context(), "idempotency.marshal_init", err)
				httpx.WriteResponse(w, httpx.ErrRespInternalServer)
				return
			}

			ok, err := rdb.SetNX(r.Context(), redisKey, initData, idempotencyProcessingTTL).Result()
			if err != nil {
				apperr.Log(r.Context(), "idempotency.setnx", err)
				httpx.WriteResponse(w, httpx.ErrRespInternalServer)
				return
			}
			if !ok {
				httpx.WriteResponse(w, httpx.Response{
					Code:  http.StatusConflict,
					Error: "request is already processing",
				})
				return
			}

			rec := newResponseRecorder(w)
			var completed bool
			defer func() {
				if !completed {
					delCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					_ = rdb.Del(delCtx, redisKey).Err()
				}
			}()

			next.ServeHTTP(rec, r)

			headers := make(map[string][]string)
			maps.Copy(headers, rec.Header())

			completedRecord := IdempotencyRecord{
				UserID:     uidStr,
				Key:        idempotencyKey,
				Status:     StatusCompleted,
				StatusCode: rec.statusCode,
				Headers:    headers,
				Body:       rec.body.Bytes(),
			}

			completedData, err := json.Marshal(completedRecord)
			if err != nil {
				apperr.Log(r.Context(), "idempotency.marshal_completed", err)
				return
			}

			saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := rdb.Set(saveCtx, redisKey, completedData, idempotencyCompletedTTL).Err(); err != nil {
				apperr.Log(r.Context(), "idempotency.save_completed", err)
				return
			}

			completed = true
		})
	}
}

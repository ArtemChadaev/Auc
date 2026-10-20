package item

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/cmd/httpx"
	"github.com/ArtemChadaev/Auction/cmd/storage"
	"github.com/ArtemChadaev/Auction/cmd/storageS3"
	"github.com/justinas/alice"
)

type service interface {
	giftItem(ctx context.Context, uid, newOwner uuid.UUID, itemID int64, description string) error
	createUploadURL(ctx context.Context, uid uuid.UUID, fileSize int64, sha256Hex, filename string) (string, uuid.UUID, error)
	confirmUpload(ctx context.Context, uid uuid.UUID, key uuid.UUID) error
	cancelUpload(ctx context.Context, uid uuid.UUID) error
	getUploadStatus(ctx context.Context, uid uuid.UUID, key uuid.UUID) (UploadStatus, error)
	getDownloadURL(ctx context.Context, uid uuid.UUID, itemID int64) (string, error)
}

// В будущем на сервере будет домен claudflare + backblaze
type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{
		service: service,
	}
}

type uploadURLReq struct {
	FileSize  int64  `json:"file_size"`
	Sha256Hex string `json:"sha_256_hex"`
	Filename  string `json:"filename"`
}

type uploadURLResp struct {
	URL string    `json:"url"`
	Key uuid.UUID `json:"key"`
}

// getUploadURL генерирует Presigned PUT URL для прямой загрузки файла клиентом в S3.
//
// @Summary Получить URL для загрузки файла
// @Description Генерирует Presigned PUT URL для прямой загрузки файла в S3
// @Tags upload
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body uploadURLReq true "Параметры загружаемого файла"
// @Success 200 {object} httpx.Response{data=uploadURLResp}
// @Failure 400 {object} httpx.Response "Неверный размер, имя или SHA-256 хеш файла"
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 409 {object} httpx.Response{data=uploadURLResp} "Хеш уже существует или у пользователя уже есть активная загрузка"
// @Failure 413 {object} httpx.Response "Превышен максимальный размер файла"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/upload/url [post]
func (h *Handler) getUploadURL(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "getUploadURL.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	req, err := httpx.DecodeJSON[uploadURLReq](w, r)
	if err != nil {
		return
	}

	// 1. Валидация размера файла
	if req.FileSize <= 0 {
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusBadRequest,
			Error: "file size must be greater than 0",
		})
		return
	}
	if req.FileSize > storageS3.MaxUploadSize {
		httpx.WriteResponse(w, httpx.ErrRespReqEntityTooLarge)
		return
	}

	// 2. Валидация имени файла
	if req.Filename == "" {
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusBadRequest,
			Error: "filename is required",
		})
		return
	}

	// 3. Валидация хеша (SHA-256 в hex-формате должен состоять ровно из 64 hex-символов)
	if len(req.Sha256Hex) != 64 {
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusBadRequest,
			Error: "invalid sha256 hash",
		})
		return
	}

	if _, err = hex.DecodeString(req.Sha256Hex); err != nil {
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusBadRequest,
			Error: "invalid sha256 hex format",
		})
		return
	}

	url, key, err := h.service.createUploadURL(r.Context(), uid, req.FileSize, req.Sha256Hex, req.Filename)
	if err != nil {
		if errors.Is(err, errUserAlreadyUploading) {
			httpx.WriteResponse(w, httpx.Response{
				Code:  http.StatusConflict,
				Error: errUserAlreadyUploading.Error(),
				Data: map[string]any{
					"url": url,
					"key": key,
				},
			})
			return
		}
		if errors.Is(err, errHashAlreadyExists) {
			httpx.WriteResponse(w, httpx.ErrRespHashAlreadyExists)
			return
		}
		apperr.Log(r.Context(), "getUploadURL.createUploadURL", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{Code: http.StatusOK, Data: map[string]any{"url": url, "key": key}})
}

// cancelUpload отменяет активную загрузку пользователя, удаляет временный файл из S3 и освобождает слот.
//
// @Summary Отменить загрузку файла
// @Description Отменяет активную загрузку пользователя, удаляет временный файл из S3 и освобождает слот
// @Tags upload
// @Security BearerAuth
// @Produce json
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 429 {object} httpx.Response "Превышен лимит отмен"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/upload/cancel [post]
func (h *Handler) cancelUpload(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "cancelUpload.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	if err = h.service.cancelUpload(r.Context(), uid); err != nil {
		if errors.Is(err, errCancelLimitReached) {
			httpx.WriteResponse(w, httpx.Response{Code: http.StatusTooManyRequests, Error: err.Error()})
			return
		}
		apperr.Log(r.Context(), "cancelUpload", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{Code: http.StatusOK})
}

type confirmUploadReq struct {
	Key uuid.UUID `json:"key"`
}

type confirmUploadResp struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// confirmUpload вызывается после успешной заливки файла в S3 для мгновенной проверки «капли» и старта фонового парсинга.
//
// @Summary Подтвердить загрузку файла
// @Description Подтверждает успешную загрузку файла в S3 для проверки и старта фонового парсинга. Ключ можно передать в заголовке X-Upload-Key или в теле JSON
// @Tags upload
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param X-Upload-Key header string false "UUID ключ загрузки"
// @Param input body confirmUploadReq false "UUID ключ загрузки в теле"
// @Success 202 {object} httpx.Response{data=confirmUploadResp}
// @Failure 400 {object} httpx.Response "Неверный или отсутствующий ключ загрузки / ошибка валидации файла"
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 403 {object} httpx.Response "Доступ запрещен"
// @Failure 404 {object} httpx.Response "Загрузка не найдена"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/upload/confirm [post]
func (h *Handler) confirmUpload(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "confirmUpload.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	var key uuid.UUID
	keyHeader := r.Header.Get("X-Upload-Key")
	if keyHeader != "" {
		parsedKey, err := uuid.Parse(strings.TrimSpace(keyHeader))
		if err == nil {
			key = parsedKey
		}
	}

	if key == uuid.Nil() {
		req, err := httpx.DecodeJSON[confirmUploadReq](w, r)
		if err != nil {
			return
		}
		key = req.Key
	}

	if key == uuid.Nil() {
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusBadRequest,
			Error: "invalid or missing upload key",
		})
		return
	}

	err = h.service.confirmUpload(r.Context(), uid, key)
	if err != nil {
		if errors.Is(err, errUploadNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
			return
		}
		if errors.Is(err, errForbidden) {
			httpx.WriteResponse(w, httpx.Response{Code: http.StatusForbidden, Error: "forbidden"})
			return
		}
		if errors.Is(err, errFileSizeMismatch) || errors.Is(err, errUnsupportedType) ||
			errors.Is(err, errInvalidExtension) || errors.Is(err, errFileSizeExceeded) {
			httpx.WriteResponse(w, httpx.Response{
				Code:  http.StatusBadRequest,
				Error: err.Error(),
			})
			return
		}

		apperr.Log(r.Context(), "confirmUpload", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusAccepted,
		Data: map[string]string{
			"status":  "processing",
			"message": "file sample verified, background processing started",
		},
	})
}

// getUploadStatus проверяет текущий статус обработки файла.
//
// @Summary Получить статус обработки файла
// @Description Проверяет текущий статус обработки файла по ключу загрузки
// @Tags upload
// @Security BearerAuth
// @Produce json
// @Param key query string true "UUID ключ загрузки"
// @Success 200 {object} httpx.Response "Файл в процессе обработки (pending, processing)"
// @Success 201 {object} httpx.Response{data=Item} "Файл успешно обработан"
// @Failure 400 {object} httpx.Response "Неверный UUID ключа или несовпадение хеша"
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 404 {object} httpx.Response "Загрузка не найдена"
// @Failure 410 {object} httpx.Response "Загрузка отменена"
// @Failure 500 {object} httpx.Response "Ошибка обработки файла или внутренняя ошибка"
// @Router /api/upload/status [get]
func (h *Handler) getUploadStatus(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "getUploadStatus.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	keyStr := r.URL.Query().Get("key")
	if keyStr == "" {
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusBadRequest,
			Error: "query parameter 'key' is required",
		})
		return
	}

	key, err := uuid.Parse(keyStr)
	if err != nil {
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusBadRequest,
			Error: "invalid key uuid",
		})
		return
	}

	status, err := h.service.getUploadStatus(r.Context(), uid, key)
	if err != nil {
		if errors.Is(err, errUploadNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
			return
		}
		apperr.Log(r.Context(), "getUploadStatus", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	switch status.Status {
	case "completed":
		data := any(status.Item)
		if status.Item == nil {
			data = status.Metadata
		}
		httpx.WriteResponse(w, httpx.Response{
			Code: http.StatusCreated,
			Data: data,
		})
	case "failed":
		slog.ErrorContext(r.Context(), "getUploadStatus: upload failed", slog.String("key", key.String()), slog.String("error", status.Error))
		code := http.StatusInternalServerError
		clientErr := "file processing failed"
		if strings.Contains(status.Error, "hash mismatch") {
			code = http.StatusBadRequest
			clientErr = "file hash mismatch"
		}
		httpx.WriteResponse(w, httpx.Response{
			Code:  code,
			Error: clientErr,
		})
	case "cancelled":
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusGone,
			Error: "upload cancelled",
		})
	default:
		// "pending", "processing" -> 200 OK без data и error
		httpx.WriteResponse(w, httpx.Response{
			Code: http.StatusOK,
		})
	}
}

type downloadURLResp struct {
	URL string `json:"url"`
}

// getDownloadURL возвращает Presigned GET URL для скачивания файла владельцем (не более 1 раза в день).
//
// @Summary Получить URL для скачивания предмета
// @Description Возвращает Presigned GET URL для скачивания файла владельцем (не более 1 раза в день)
// @Tags item
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID предмета"
// @Success 200 {object} httpx.Response{data=downloadURLResp}
// @Failure 400 {object} httpx.Response "Неверный ID предмета"
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 403 {object} httpx.Response "Доступ запрещен"
// @Failure 404 {object} httpx.Response "Предмет не найден"
// @Failure 429 {object} httpx.Response "Превышен суточный лимит скачиваний"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/item/{id}/download [get]
func (h *Handler) getDownloadURL(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "getDownloadURL.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	idStr := r.PathValue("id")
	itemID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusBadRequest,
			Error: "invalid item id",
		})
		return
	}

	url, err := h.service.getDownloadURL(r.Context(), uid, itemID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
			return
		}
		if errors.Is(err, errForbidden) {
			httpx.WriteResponse(w, httpx.Response{Code: http.StatusForbidden, Error: "forbidden"})
			return
		}
		if errors.Is(err, errDownloadLimitReached) {
			httpx.WriteResponse(w, httpx.ErrRespDownloadLimit)
			return
		}
		apperr.Log(r.Context(), "getDownloadURL", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
		Data: downloadURLResp{
			URL: url,
		},
	})
}

func (h *Handler) RouterUpload(authChain alice.Chain) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /url", h.getUploadURL)
	mux.HandleFunc("POST /confirm", h.confirmUpload)
	mux.HandleFunc("POST /cancel", h.cancelUpload)
	mux.HandleFunc("GET /status", h.getUploadStatus)
	return authChain.Then(mux)
}

func (h *Handler) RouterItem(authChain alice.Chain) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{id}/download", h.getDownloadURL)
	mux.HandleFunc("POST /gift", h.giftItem)
	return authChain.Then(mux)
}

type giftItemReq struct {
	NewOwner    uuid.UUID `json:"new_owner"`
	ItemID      int64     `json:"item_id"`
	Description string    `json:"description"`
}

// giftItem передача предмета в подарок другому пользователю
//
// @Summary Передать предмет в подарок
// @Description Передает предмет новому владельцу с сохранением истории передачи
// @Tags item
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body giftItemReq true "Параметры подарка"
// @Success 202 {object} httpx.Response
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 404 {object} httpx.Response "Предмет не найден"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/item/gift [post]
func (h *Handler) giftItem(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[giftItemReq](w, r)
	if err != nil {
		apperr.Log(r.Context(), "", err)
		return
	}

	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	err = h.service.giftItem(r.Context(), uid, req.NewOwner, req.ItemID, req.Description)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
		} else {
			apperr.Log(r.Context(), "", err)
			httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		}
		return
	}
	httpx.WriteResponse(w, httpx.Response{Code: http.StatusAccepted})
}

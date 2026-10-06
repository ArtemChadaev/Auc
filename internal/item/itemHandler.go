package item

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/cmd/httpx"
	"github.com/ArtemChadaev/Auction/cmd/storageS3"
	"github.com/ArtemChadaev/Auction/internal/storage"
	"github.com/gabriel-vasile/mimetype"
)

type service interface {
	giftItem(ctx context.Context, uid, newOwner uuid.UUID, itemID int64, description string) error
	//create(ctx context.Context, iType itemType, mType string, hmData headerMetadata, file io.Reader, uid uuid.UUID)
	createURLForUpload(ctx context.Context, uuid4 uuid.UUID, fileSize int64) (string, error)
	hasHash(ctx context.Context, hash string) (bool, error)
}

type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{
		service: service,
	}
}

// validateFileExtension Если расширение файла не сходится с содержимым -> false (не сохранять)
func validateFileExtension(filename string, mtype *mimetype.MIME) bool {
	userExt := strings.ToLower(filepath.Ext(filename))
	if userExt == "" {
		return false
	}

	Ext := mtype.Extension()
	if Ext == userExt {
		return true
	}

	if aliases, ok := extensionAliases[Ext]; ok {
		if slices.Contains(aliases, userExt) {
			return true
		}

		//TODO:Разобраться
		//for _, alias := range aliases {
		//			if userExt == alias {
		//				return true
		//			}
		//		}
	}
	return false
}

// fileType Возвращает краткий тип файла
func fileType(filename string, mtype *mimetype.MIME) itemType {
	mimeStr := mtype.String()
	userExt := strings.ToLower(filepath.Ext(filename))

	if strings.HasPrefix(mimeStr, "image/") {
		return Image
	}

	if strings.HasPrefix(mimeStr, "audio/") {
		return Audio
	}

	if strings.HasPrefix(mimeStr, "video/") {
		return Video
	}

	// 4. 3D-МОДЕЛИ
	if strings.HasPrefix(mimeStr, "model/") ||
		mimeStr == "application/sla" || // устаревший MIME для STL
		mimeStr == "application/x-tgif" ||
		userExt == ".obj" || userExt == ".fbx" || userExt == ".step" || userExt == ".stp" {
		return Model3D
	}

	// 5. ДОКУМЕНТЫ
	if mimeStr == "application/pdf" ||
		mimeStr == "application/rtf" ||
		mimeStr == "application/epub+zip" ||
		mtype.Is("application/msword") || // покрывает бинарные .doc, .xls, .ppt
		strings.Contains(mimeStr, "openxmlformats-officedocument") || // покрывает .docx, .xlsx, .pptx
		strings.Contains(mimeStr, "opendocument") || // LibreOffice / OpenOffice (.odt, .ods, .odp)
		userExt == ".csv" || userExt == ".txt" || userExt == ".tsv" || userExt == ".md" {
		return Document
	}

	// 6. АРХИВЫ И ОБРАЗЫ ДИСКОВ
	if mtype.Is("application/zip") ||
		mtype.Is("application/x-tar") ||
		mimeStr == "application/x-7z-compressed" ||
		mimeStr == "application/vnd.rar" ||
		mimeStr == "application/x-rar-compressed" ||
		mimeStr == "application/gzip" ||
		mimeStr == "application/x-bzip2" ||
		mimeStr == "application/x-xz" ||
		mimeStr == "application/x-iso9660-image" ||
		mimeStr == "application/zstd" {
		return Archive
	}
	return ""
}

//var badFile = "bad file"
//
//func (h *Handler) uploadFile(w http.ResponseWriter, r *http.Request) {
//
//	// Проверка файла
//	if err := r.ParseMultipartForm(5 << 20); err != nil {
//		httpx.WriteResponse(w, httpx.ErrRespReqEntityTooLarge)
//		return
//	}
//
//	file, header, err := r.FormFile("file")
//	if err != nil {
//		apperr.Log(r.Context(), "", apperr.NewAppErrorString("form file error", -4))
//		httpx.WriteResponse(w, httpx.Response{
//			Code:  http.StatusBadRequest,
//			Error: badFile,
//		})
//		return
//	}
//	defer file.Close()
//
//	// Проверка типа файла
//	mtype, err := mimetype.DetectReader(file)
//	if err != nil {
//		apperr.Log(r.Context(), "", apperr.NewAppErrorString("don't mimetype", -4))
//		httpx.WriteResponse(w, httpx.Response{
//			Code:  http.StatusUnsupportedMediaType,
//			Error: badFile,
//		})
//		return
//	}
//	if !validateFileExtension(header.Filename, mtype) {
//		apperr.Log(r.Context(), "", apperr.NewAppErrorString("validateFileExtension", -4), slog.String("filename", header.Filename), slog.Any("mimetype", mtype.Extension()))
//		httpx.WriteResponse(w, httpx.Response{
//			Code:  http.StatusUnsupportedMediaType,
//			Error: badFile,
//		})
//		return
//	}
//
//	_, err = file.Seek(0, io.SeekStart)
//	if err != nil {
//		apperr.Log(r.Context(), "file.Seek error", apperr.NewAppError(err, -4))
//		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
//		return
//	}
//
//	fType := fileType(header.Filename, mtype)
//
//	var hmData = headerMetadata{
//		MimeType: mtype.String(),
//		Filename: header.Filename,
//		Size:     header.Size,
//		Header:   header.Header,
//	}
//	if err = h.service.create(r.Context(), fType, mtype.String(), hmData, file); err != nil {
//		apperr.Log(r.Context(), "error creating item", err)
//		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
//		return
//	}
//
//	httpx.WriteResponse(w, httpx.Response{Code: http.StatusCreated, Data: fType})
//}

// getUploadURL генерирует Presigned PUT URL для прямой загрузки файла клиентом в S3.
//
// Эндпоинт: POST /api/upload/url
// Content-Type: application/json
// Accept: application/json
//
// Принимает тело (JSON):
//   - file_size: int64 (обязательно) — размер файла в байтах. Должен быть: 0 < file_size <= 5 ГБ (storageS3.MaxUploadSize).
//   - sha_256_hex: string (обязательно) — криптографический SHA-256 хеш содержимого файла (ровно 64 hex-символа).
//
// Возвращает тело (JSON):
//   - url: string — подписанный Presigned URL для выполнения HTTP PUT в S3 (время жизни 24 часа).
//   - key: uuid.UUID — уникальный ключ/идентификатор файла в S3 (сохранен под префиксом "tmp/<key>").
//
// Коды ответов:
//   - 200 OK: Ссылка успешно сгенерирована.
//   - 400 Bad Request:
//     1) Невалидный JSON / неизвестные поля (error: "invalid request body")
//     2) Размер файла <= 0 (error: "file size must be greater than 0")
//     3) Длина хеша != 64 (error: "invalid sha256 hash")
//     4) Хеш не является валидной hex-строкой (error: "invalid sha256 hex format")
//   - 409 Conflict:
//     1) Хеш файла уже существует в базе данных (error: "hash already exists")
//   - 413 Payload Too Large:
//     1) Размер file_size превышает 5 ГБ (error: "request entity too large")
//     2) Тело самого JSON запроса превышает 5 МБ (error: "request entity too large")
//   - 500 Internal Server Error: внутренняя ошибка генерации подписи S3 / сервера (error: "internal server error")
func (h *Handler) getUploadURL(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[struct {
		FileSize  int64  `json:"file_size"`
		Sha256Hex string `json:"sha_256_hex"`
	}](w, r)
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

	// 2. Валидация хеша (SHA-256 в hex-формате должен состоять ровно из 64 hex-символов)
	if len(req.Sha256Hex) != 64 {
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusBadRequest,
			Error: "invalid sha256 hash",
		})
		return
	}

	req.Sha256Hex = strings.ToLower(req.Sha256Hex)
	if _, err := hex.DecodeString(req.Sha256Hex); err != nil {
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusBadRequest,
			Error: "invalid sha256 hex format",
		})
		return
	}

	// 3. Проверка существования хеша в БД (таблица s3)
	exists, err := h.service.hasHash(r.Context(), req.Sha256Hex)
	if err != nil {
		apperr.Log(r.Context(), "getUploadURL.hasHash", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}
	if exists {
		httpx.WriteResponse(w, httpx.ErrRespHashAlreadyExists)
		return
	}

	uuid4 := uuid.New()

	url, err := h.service.createURLForUpload(r.Context(), uuid4, req.FileSize)
	if err != nil {
		apperr.Log(r.Context(), "getUploadURL.createURLForUpload", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	resp := struct {
		URL string    `json:"url"`
		Key uuid.UUID `json:"key"`
	}{
		URL: url,
		Key: uuid4,
	}

	httpx.WriteResponse(w, httpx.Response{Code: http.StatusOK, Data: resp})
}

func (h *Handler) RouterUpload() http.Handler {
	mux := http.NewServeMux()
	//mux.HandleFunc("POST /", h.uploadFile)
	mux.HandleFunc("POST /url", h.getUploadURL)
	// TODO: mux.HandleFunc("POST /confirm", h.confirmUpload)

	return mux
}

// TODO: Реализовать паттерн Confirm (подтверждение и верификация загрузки):
// После того как клиент успешно выполнил PUT файла в S3, он вызывает эндпоинт подтверждения:
// POST /api/upload/confirm (или передает key при создании лота).
//
// Как работает правильная проверка (хендлер confirmUpload):
// 1. Принимает тело JSON:
//      - key: uuid.UUID        (идентификатор файла в S3)
//      - expected_hash: string (SHA-256 hex, 64 символа)
//      - expected_size: int64  (ожидаемый размер в байтах)
// 2. Делает запрос в S3 (s3.Download / HeadObject) по ключу "tmp/" + key:
//      - Проверяет реальный размер файла: если actualSize != expected_size ->
//        удалить s3.Delete("tmp/"+key) и вернуть 400 Bad Request.
//      - Потоком читает первые 512 байт и определяет настоящий MIME-тип через mimetype.DetectReader.
//      - Потоком через hasher := sha256.New() дочитывает файл и вычисляет actualSha256:
//        если actualSha256 != expected_hash -> удалить s3.Delete("tmp/"+key) и вернуть 400 Bad Request.
// 3. Если проверки пройдены:
//      - Переместить файл из "tmp/<key>" в постоянную директорию (или оставить, сняв статус временного).
//      - Привязать файл к товару/пользователю в базе данных.
//      - Вернуть клиенту статус 200 OK / 201 Created.

// В будущем на сервере будет домен claudflare + backblaze
func (h *Handler) giftItem(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[struct {
		NewOwner    uuid.UUID `json:"new_owner"`
		ItemID      int64     `json:"item_id"`
		Description string    `json:"description"`
	}](w, r)
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

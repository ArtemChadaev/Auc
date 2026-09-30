package item

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/internal/httpx"
	"github.com/ArtemChadaev/Auction/internal/storage"
	"github.com/gabriel-vasile/mimetype"
	"github.com/justinas/alice"
)

type service interface {
	newOwner(ctx context.Context, uid, newOwner uuid.UUID, itemID int64, description string) error
	create(ctx context.Context, iType itemType, mType string, hmData headerMetadata, file io.Reader) error
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

		//TODO:Разобратся
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

func (h *Handler) uploadFile(w http.ResponseWriter, r *http.Request) {
	// Проверка файла
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		slog.DebugContext(r.Context(), "max body size exceeded")
		http.Error(w, httpx.ReqEntityTooLarge, http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		slog.DebugContext(r.Context(), "form file error")
		http.Error(w, httpx.BadFile, http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Проверка типа файла
	mtype, err := mimetype.DetectReader(file)
	if err != nil {
		slog.DebugContext(r.Context(), "don't mimetype")
		http.Error(w, httpx.BadFile, http.StatusUnsupportedMediaType)
		return
	}
	if !validateFileExtension(header.Filename, mtype) {
		slog.DebugContext(r.Context(), "Ext error", slog.String("filename", header.Filename), slog.Any("mimetype", mtype.Extension()))
		http.Error(w, httpx.BadFile, http.StatusUnsupportedMediaType)
		return
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		slog.DebugContext(r.Context(), "offset 0")
		http.Error(w, httpx.InternalServer, http.StatusInternalServerError)
		return
	}

	fType := fileType(header.Filename, mtype)

	var hmData = headerMetadata{
		MimeType: mtype.String(),
		Filename: header.Filename,
		Size:     header.Size,
		Header:   header.Header,
	}
	if err = h.service.create(r.Context(), fType, mtype.String(), hmData, file); err != nil {
		slog.DebugContext(r.Context(), "error creating item", slog.Any("error", err))
		http.Error(w, httpx.InternalServer, http.StatusInternalServerError)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, fType)
}

func (h *Handler) RouterUpload() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", h.uploadFile)

	return mux
}

func (h *Handler) newOwner(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[struct {
		NewOwner    uuid.UUID `json:"new_owner"`
		ItemID      int64     `json:"item_id"`
		Description string    `json:"description"`
	}](r)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("itemHandler.newOwner: %w", err))
		return
	}

	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		slog.DebugContext(r.Context(), "", slog.String("error", err.Error()))
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	err = h.service.newOwner(r.Context(), uid, req.NewOwner, req.ItemID, req.Description)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			slog.DebugContext(r.Context(), "", slog.String("error", err.Error()))
			http.Error(w, httpx.NotFound, http.StatusNotFound)
		} else {
			httpx.WriteError(w, r, fmt.Errorf("itemHandler.newOwner: %w", err))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, nil)
}

// Router /api/item. chain - авторизация
func (h *Handler) Router(authChain alice.Chain) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("PATCH /gift", authChain.ThenFunc(h.newOwner))

	return mux
}

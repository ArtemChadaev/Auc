package item

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/internal/httpx"
	"github.com/ArtemChadaev/Auction/internal/storage"
	"github.com/justinas/alice"
)

type service interface {
	newOwner(ctx context.Context, uid, newOwner uuid.UUID, itemID int64, description string) error
}

type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) uploadFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		slog.DebugContext(r.Context(), "max body size exceeded")
		http.Error(w, httpx.ReqEntityTooLarge, http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		slog.DebugContext(r.Context(), "form file error")
		http.Error(w, httpx.BadFile, http.StatusRequestEntityTooLarge)
		return
	}

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

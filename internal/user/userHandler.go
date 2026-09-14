package user

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ArtemChadaev/Auction/internal/httpx"
)

type service interface {
	Auth(ctx context.Context, email string, password string) error
	Register(ctx context.Context, name string, email string, password string) error
}

type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{
		service: service,
	}
}

type userReq struct {
	Email string `json:"email"`
	Pass  string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req userReq

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		if strings.HasPrefix(err.Error(), "json: unknown field") {
			http.Error(w, "json don't have this field", http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
		return
	}

	if err := h.service.Auth(r.Context(), req.Email, req.Pass); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, "111")
}

type registerReq struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Pass  string `json:"password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerReq

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		if strings.HasPrefix(err.Error(), "json: unknown field") {
			http.Error(w, "json don't have this field", http.StatusBadRequest)
		} else {
			http.Error(w, "invalid request body", http.StatusBadRequest) // было StatusUnauthorized
		}
		return
	}
	if err := h.service.Register(r.Context(), req.Name, req.Email, req.Pass); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}
	httpx.WriteJSON(w, http.StatusOK, "user "+req.Name+" registered")
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/register", h.register)
}

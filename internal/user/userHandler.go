package user

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"uuid"

	"github.com/ArtemChadaev/Auction/internal/httpx"
)

type service interface {
	register(ctx context.Context, uid *uuid.UUID, name string, email string, password string, device Device) (Tokens, error)
	authPassword(ctx context.Context, email string, password string, device Device) (Tokens, error)
	authRefresh(ctx context.Context, refresh string) (Tokens, error)
	tokenResponds(ctx context.Context, refresh string) (Session, error)
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
	//TODO: На время заглушка потом переделать
	device := Device{}
	tokens, err := h.service.authPassword(r.Context(), req.Email, req.Pass, device)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	session, err := h.service.tokenResponds(r.Context(), tokens.RefreshToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	responds := tokenResponds{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Session:      session,
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    responds.RefreshToken,
		Path:     "/api/auth",
		Expires:  responds.Session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
	httpx.WriteJSON(w, http.StatusOK, responds)
}

type registerReq struct {
	ID    *uuid.UUID `json:"uid"`
	Name  string     `json:"name"`
	Email string     `json:"email"`
	Pass  string     `json:"password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerReq

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		if strings.HasPrefix(err.Error(), "json: unknown field") {
			http.Error(w, "json don't have this field", http.StatusBadRequest)
		} else {
			http.Error(w, "invalid request body", http.StatusBadRequest)
		}
		return
	}
	device := Device{}
	tokens, err := h.service.register(r.Context(), req.ID, req.Name, req.Email, req.Pass, device)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	session, err := h.service.tokenResponds(r.Context(), tokens.RefreshToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	responds := tokenResponds{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Session:      session,
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    responds.RefreshToken,
		Path:     "/api/auth",
		Expires:  responds.Session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
	httpx.WriteJSON(w, http.StatusOK, responds)
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/register", h.register)
}

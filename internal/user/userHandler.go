package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/internal/httpx"
	"github.com/ArtemChadaev/Auction/internal/storage"
	"github.com/justinas/alice"
)

type service interface {
	register(ctx context.Context, uid *uuid.UUID, name string, email string, password string, device Device) (Tokens, error)
	authPassword(ctx context.Context, email string, password string, device Device) (Tokens, error)
	authRefresh(ctx context.Context, refresh string) (Tokens, error)
	tokenResponds(ctx context.Context, refresh string) (Session, error)
	logout(ctx context.Context, refreshId []int64, uid uuid.UUID) error
	findTokens(ctx context.Context, uid uuid.UUID, current bool) ([]Session, error)
	getUser(ctx context.Context, uid uuid.UUID) (User, error)
	patchUserName(ctx context.Context, uid uuid.UUID, name string) error
	deleteUser(ctx context.Context, uid uuid.UUID) error
}

// TODO: ERROR: Всё сделать и перепроверить в trim!!!!! иначе ошибка будет
type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{
		service: service,
	}
}

func getDevice(r *http.Request) Device {

	return Device{}
}

type userReq struct {
	Email string `json:"email"`
	Pass  string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[userReq](r)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("userHandler.login: %w", err))
		return
	}

	tokens, err := h.service.authPassword(r.Context(), req.Email, req.Pass, getDevice(r))
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, apperr.ErrUnauthorized) {
			slog.DebugContext(r.Context(), "", slog.String("error", err.Error()))
			http.Error(w, httpx.InvalidAuth, http.StatusUnauthorized)
		} else {
			httpx.WriteError(w, r, fmt.Errorf("userHandler.login: %w", err))
		}
		return
	}

	session, err := h.service.tokenResponds(r.Context(), tokens.RefreshToken)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("userHandler.login: %w", err))
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
		Path:     "/api/auth/refresh",
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
	req, err := httpx.DecodeJSON[registerReq](r)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("userHandler.register: %w", err))
		return
	}

	tokens, err := h.service.register(r.Context(), req.ID, req.Name, req.Email, req.Pass, getDevice(r))
	if err != nil {
		if errors.Is(err, storage.ErrUniqueViolation) {
			slog.DebugContext(r.Context(), "", slog.String("error", err.Error()))
			http.Error(w, httpx.FieldIsTaken, http.StatusUnprocessableEntity)
		} else {
			httpx.WriteError(w, r, fmt.Errorf("userHandler.register: %w", err))
		}
		return
	}
	session, err := h.service.tokenResponds(r.Context(), tokens.RefreshToken)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("userHandler.register: %w", err))
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
		Path:     "/api/auth/refresh",
		Expires:  responds.Session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
	httpx.WriteJSON(w, http.StatusOK, responds)
}

type refreshIdReq struct {
	ID []int64 `json:"refresh_id"`
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[refreshIdReq](r)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("userHandler.logout: %w", err))
		return
	}

	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		slog.DebugContext(r.Context(), "", slog.String("error", err.Error()))
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if err = h.service.logout(r.Context(), req.ID, uid); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("userHandler.logout: %w", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, nil)
}

type findRefreshReq struct {
	Current *bool `json:"current,omitempty"`
}

func (h *Handler) findRefresh(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[findRefreshReq](r)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("userHandler.findRefresh: %w", err))
		return
	}

	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		slog.DebugContext(r.Context(), "", slog.String("error", err.Error()))
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	var current bool
	if req.Current == nil {
		current = true
	} else {
		current = *req.Current
	}
	sessions, err := h.service.findTokens(r.Context(), uid, current)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			slog.DebugContext(r.Context(), "", slog.String("error", err.Error()))
			http.Error(w, httpx.NotFound, http.StatusNotFound)
		} else {
			httpx.WriteError(w, r, fmt.Errorf("userHandler.findRefresh: %w", err))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, sessions)
}

func (h *Handler) loginToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		slog.DebugContext(r.Context(), "", slog.String("error", err.Error()))
		http.Error(w, httpx.InvalidAuth, http.StatusUnauthorized)
		return
	}
	tokens, err := h.service.authRefresh(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			slog.DebugContext(r.Context(), "", slog.String("error", err.Error()))
			http.Error(w, httpx.InvalidAuth, http.StatusUnauthorized)
		} else {
			httpx.WriteError(w, r, fmt.Errorf("userHandler.loginToken: %w", err))
		}
		return
	}
	session, err := h.service.tokenResponds(r.Context(), tokens.RefreshToken)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("userHandler.register: %w", err))
		return
	}
	responds := tokenResponds{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Session:      session,
	}
	cookie = &http.Cookie{
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

// RoutesAuth /api/auth
func (h *Handler) RoutesAuth(chain alice.Chain) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("POST /register", h.register)
	mux.HandleFunc("POST /refresh", h.loginToken)

	mux.Handle("POST /logout", chain.ThenFunc(h.logout))
	mux.Handle("POST /session", chain.ThenFunc(h.findRefresh))

	return mux
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		slog.DebugContext(r.Context(), "", slog.String("error", err.Error()))
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	user, err := h.service.getUser(r.Context(), uid)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("userHandler.register: %w", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

type newNameReq struct {
	Name string `json:"name"`
}

func (h *Handler) patchUserName(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[newNameReq](r)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("userHandler.register: %w", err))
		return
	}

	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		slog.DebugContext(r.Context(), "", slog.String("error", err.Error()))
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	err = h.service.patchUserName(r.Context(), uid, req.Name)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("userHandler.register: %w", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, nil)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		slog.DebugContext(r.Context(), "", slog.String("error", err.Error()))
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	err = h.service.deleteUser(r.Context(), uid)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("userHandler.register: %w", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, nil)
}

// RoutesUser /api/user
func (h *Handler) RoutesUser() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /me", h.getUser)
	mux.HandleFunc("PATCH /name", h.patchUserName)
	mux.HandleFunc("DELETE /me", h.deletedUser)

	return mux
}

package user

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/cmd/httpx"
	"github.com/ArtemChadaev/Auction/cmd/mimeType"
	"github.com/ArtemChadaev/Auction/cmd/storage"
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
	updateAvatar(ctx context.Context, uid uuid.UUID, r io.Reader) (uuid.UUID, error)
}

type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{
		service: service,
	}
}

func getDevice(r *http.Request) Device {
	//TODO:
	return Device{}
}

type userReq struct {
	Email string `json:"email"`
	Pass  string `json:"password"`
}

// @Summary Авторизация пользователя
// @Description Аутентификация пользователя по email и паролю, возвращает access и refresh токены
// @Tags auth
// @Accept json
// @Produce json
// @Param input body userReq true "Учетные данные пользователя"
// @Success 200 {object} httpx.Response{data=tokenResponds}
// @Failure 401 {object} httpx.Response "Неверный email или пароль"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/auth/login [post]
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[userReq](w, r)
	if err != nil {
		apperr.Log(r.Context(), "login", err)
		return
	}

	tokens, err := h.service.authPassword(r.Context(), req.Email, req.Pass, getDevice(r))
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, apperr.ErrUnauthorized) {
			slog.DebugContext(r.Context(), "login: invalid auth", slog.String("error", err.Error()))
			httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		} else {
			apperr.Log(r.Context(), "login.authPassword", err)
			httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		}
		return
	}

	session, err := h.service.tokenResponds(r.Context(), tokens.RefreshToken)
	if err != nil {
		apperr.Log(r.Context(), "login.tokenResponds", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}
	response := tokenResponds{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Session:      session,
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    response.RefreshToken,
		Path:     "/api/auth/refresh",
		Expires:  response.Session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
		Data: response,
	})
}

type registerReq struct {
	ID    *uuid.UUID `json:"uid"`
	Name  string     `json:"name"`
	Email string     `json:"email"`
	Pass  string     `json:"password"`
}

// @Summary Регистрация пользователя
// @Description Создает новый аккаунт пользователя и возвращает токены
// @Tags auth
// @Accept json
// @Produce json
// @Param input body registerReq true "Регистрационные данные"
// @Success 200 {object} httpx.Response{data=tokenResponds}
// @Failure 409 {object} httpx.Response "Email уже занят"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/auth/register [post]
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[registerReq](w, r)
	if err != nil {
		apperr.Log(r.Context(), "register", err)
		return
	}

	tokens, err := h.service.register(r.Context(), req.ID, req.Name, req.Email, req.Pass, getDevice(r))
	if err != nil {
		if errors.Is(err, storage.ErrUniqueViolation) {
			slog.DebugContext(r.Context(), "register: unique violation", slog.String("error", err.Error()))
			httpx.WriteResponse(w, httpx.ErrRespFieldIsTaken)
		} else {
			apperr.Log(r.Context(), "register.serviceRegister", err)
			httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		}
		return
	}
	session, err := h.service.tokenResponds(r.Context(), tokens.RefreshToken)
	if err != nil {
		apperr.Log(r.Context(), "register.tokenResponds", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}
	response := tokenResponds{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Session:      session,
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    response.RefreshToken,
		Path:     "/api/auth/refresh",
		Expires:  response.Session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
		Data: response,
	})
}

type refreshIdReq struct {
	ID []int64 `json:"refresh_id"`
}

// @Summary Выход из системы
// @Description Завершает указанные refresh-сессии пользователя. Требует авторизации
// @Tags auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body refreshIdReq true "ID сессий для завершения"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/auth/logout [post]
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[refreshIdReq](w, r)
	if err != nil {
		apperr.Log(r.Context(), "logout", err)
		return
	}

	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "logout.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}
	if err = h.service.logout(r.Context(), req.ID, uid); err != nil {
		apperr.Log(r.Context(), "logout.serviceLogout", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}
	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
	})
}

type findRefreshReq struct {
	Current *bool `json:"current,omitempty"`
}

// @Summary Получение сессий пользователя
// @Description Возвращает список сессий пользователя (текущих или всех). Требует авторизации
// @Tags auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body findRefreshReq true "Фильтр сессий"
// @Success 200 {object} httpx.Response{data=[]Session}
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 404 {object} httpx.Response "Сессии не найдены"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/auth/session [post]
func (h *Handler) findRefresh(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[findRefreshReq](w, r)
	if err != nil {
		apperr.Log(r.Context(), "findRefresh", err)
		return
	}

	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "findRefresh.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
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
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
		} else {
			apperr.Log(r.Context(), "findRefresh.findTokens", err)
			httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		}
		return
	}
	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
		Data: sessions,
	})
}

// @Summary Обновление токенов
// @Description Обновляет access токен с использованием refresh_token cookie
// @Tags auth
// @Produce json
// @Success 200 {object} httpx.Response{data=tokenResponds}
// @Failure 401 {object} httpx.Response "Отсутствует или невалидный cookie"
// @Failure 404 {object} httpx.Response "Сессия не найдена"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/auth/refresh [post]
func (h *Handler) loginToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		slog.DebugContext(r.Context(), "loginToken: missing cookie", slog.String("error", err.Error()))
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}
	tokens, err := h.service.authRefresh(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
		} else {
			apperr.Log(r.Context(), "loginToken.authRefresh", err)
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
		}
		return
	}
	session, err := h.service.tokenResponds(r.Context(), tokens.RefreshToken)
	if err != nil {
		apperr.Log(r.Context(), "loginToken.tokenResponds", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}
	response := tokenResponds{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Session:      session,
	}
	cookie = &http.Cookie{
		Name:     "refresh_token",
		Value:    response.RefreshToken,
		Path:     "/api/auth",
		Expires:  response.Session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
		Data: response,
	})
}

// RoutesAuth /api/auth
func (h *Handler) RoutesAuth(authChain alice.Chain) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("POST /register", h.register)
	mux.HandleFunc("POST /refresh", h.loginToken)

	mux.Handle("POST /logout", authChain.ThenFunc(h.logout))
	mux.Handle("POST /session", authChain.ThenFunc(h.findRefresh))

	return mux
}

// @Summary Получить профиль пользователя
// @Description Возвращает данные текущего авторизованного пользователя
// @Tags user
// @Security BearerAuth
// @Produce json
// @Success 200 {object} httpx.Response{data=User}
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/user/me [get]
func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "getUser.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}
	user, err := h.service.getUser(r.Context(), uid)
	if err != nil {
		apperr.Log(r.Context(), "getUser.serviceGetUser", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}
	httpx.WriteResponse(w, httpx.Response{Code: http.StatusOK, Data: user})
}

type newNameReq struct {
	Name string `json:"name"`
}

// @Summary Обновить имя пользователя
// @Description Изменяет отображаемое имя текущего пользователя
// @Tags user
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body newNameReq true "Новое имя пользователя"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/user/name [patch]
func (h *Handler) patchUserName(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[newNameReq](w, r)
	if err != nil {
		apperr.Log(r.Context(), "patchUserName", err)
		return
	}

	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "patchUserName.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}
	err = h.service.patchUserName(r.Context(), uid, req.Name)
	if err != nil {
		apperr.Log(r.Context(), "patchUserName.servicePatch", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}
	httpx.WriteResponse(w, httpx.Response{Code: http.StatusOK})
}

// @Summary Удалить профиль пользователя
// @Description Выполняет мягкое удаление профиля текущего пользователя
// @Tags user
// @Security BearerAuth
// @Produce json
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/user/me [delete]
func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "deleteUser.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}
	err = h.service.deleteUser(r.Context(), uid)
	if err != nil {
		apperr.Log(r.Context(), "deleteUser.serviceDelete", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}
	httpx.WriteResponse(w, httpx.Response{Code: http.StatusOK})
}

type avatarResp struct {
	Image uuid.UUID `json:"image"`
}

// updateAvatar обновляет или устанавливает аватар пользователя.
//
// @Summary Обновить аватар пользователя
// @Description Загружает и устанавливает квадратный аватар пользователя
// @Tags user
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Файл изображения аватара (квадратный)"
// @Success 200 {object} httpx.Response{data=avatarResp}
// @Failure 400 {object} httpx.Response "Неверный формат или пропорции изображения"
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 404 {object} httpx.Response "Пользователь не найден"
// @Failure 413 {object} httpx.Response "Превышен допустимый размер файла"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/user/me/avatar [put]
func (h *Handler) updateAvatar(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "updateAvatar.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "multipart/form-data") {
		httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			httpx.WriteResponse(w, httpx.ErrRespReqEntityTooLarge)
			return
		}
		httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
		return
	}
	defer file.Close()

	if _, err = mimeType.ValidatePhoto(file, header.Filename, header.Size); err != nil {
		apperr.Log(r.Context(), "", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
		return
	}

	if seeker, ok := file.(io.Seeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
	}

	s3ID, err := h.service.updateAvatar(r.Context(), uid, file)
	if err != nil {
		if errors.Is(err, ErrImageNotSquare) || errors.Is(err, ErrInvalidImage) {
			httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
			return
		}
		if errors.Is(err, storage.ErrNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
			return
		}
		apperr.Log(r.Context(), "updateAvatar", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
		Data: map[string]any{
			"image": s3ID,
		},
	})
}

// RoutesUser /api/user
func (h *Handler) RoutesUser(authChain alice.Chain) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /me", h.getUser)
	mux.HandleFunc("PATCH /name", h.patchUserName)
	mux.HandleFunc("DELETE /me", h.deleteUser)
	mux.HandleFunc("PUT /me/avatar", h.updateAvatar)

	return authChain.Then(mux)
}

package lot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/cmd/httpx"
	"github.com/ArtemChadaev/Auction/cmd/mimeType"
	"github.com/ArtemChadaev/Auction/cmd/storage"
	"github.com/justinas/alice"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type reqNewLot struct {
	ItemID       int64           `json:"item_id"`
	Status       Status          `json:"status"`
	StartAmount  int64           `json:"start_amount"`
	MinStep      int64           `json:"min_step"`
	Title        string          `json:"title"`
	Description  json.RawMessage `json:"description"`
	StartedAt    *time.Time      `json:"started_at"`
	EndedAt      *time.Time      `json:"ends_at"`
	AutoActivate bool            `json:"auto_activate,omitempty"`
	AutoActive   bool            `json:"auto_active,omitempty"`
}

func (r *reqNewLot) check() error {
	if r.ItemID <= 0 {
		return errors.New("invalid item_id: must be greater than 0")
	}
	if strings.TrimSpace(r.Title) == "" {
		return errors.New("title is required")
	}
	if r.StartAmount < 100 ||
		r.MinStep < 50 ||
		r.MinStep > 500 ||
		r.StartAmount%100 != 0 ||
		r.MinStep%50 != 0 {
		return errors.New("invalid req body: start_amount must be >= 100 and a multiple of 100, min_step must be between 50 and 500 and a multiple of 50")
	}
	if r.Status == "" {
		r.Status = Draft
	}
	if r.Status != Draft && r.Status != PendingReview {
		return errors.New("invalid status: seller can only set status to 'draft' or 'pending_review'")
	}
	if !(r.StartedAt == nil && r.EndedAt == nil || r.StartedAt != nil && r.EndedAt != nil) {
		return errors.New("invalid req body: started_at and ends_at must either both be set or both be omitted")
	}
	if r.StartedAt != nil && r.EndedAt != nil && !r.EndedAt.After(*r.StartedAt) {
		return errors.New("ends_at must be after started_at")
	}
	if len(r.Description) == 0 {
		r.Description = json.RawMessage("{}")
	}
	dec := json.NewDecoder(bytes.NewReader(r.Description))
	dec.DisallowUnknownFields()
	var desc Description
	if err := dec.Decode(&desc); err != nil {
		return fmt.Errorf("invalid description: %w", err)
	}
	if desc.Type != "" {
		switch desc.Type {
		case mimeType.Image, mimeType.Model3D, mimeType.Audio, mimeType.Video, mimeType.Document, mimeType.Archive:
		default:
			return errors.New("invalid description type")
		}
	}
	if r.AutoActivate || r.AutoActive || (desc.AutoActivate != nil && *desc.AutoActivate) || (desc.AutoActive != nil && *desc.AutoActive) {
		t := true
		r.AutoActivate = true
		r.AutoActive = true
		desc.AutoActivate = &t
		desc.AutoActive = &t
		if (r.StartedAt == nil || r.EndedAt == nil) && r.Status == PendingReview {
			return errors.New("started_at and ends_at are required when auto_activate is true")
		}
		updatedDesc, err := json.Marshal(desc)
		if err == nil {
			r.Description = updatedDesc
		}
	}
	return nil
}

// list возвращает список активных и завершенных лотов.
//
// @Summary Получить список лотов
// @Description Возвращает список лотов аукциона со статусами active и finished
// @Tags lot
// @Produce json
// @Success 200 {object} httpx.Response{data=[]Lot}
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/lot/ [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	lots, err := h.service.list(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.list", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}
	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
		Data: lots,
	})
}

// getByID возвращает детальную информацию о лоте по его ID.
//
// @Summary Получить лот по ID
// @Description Возвращает информацию о лоте по его числовому идентификатору
// @Tags lot
// @Produce json
// @Param id path int true "ID лота"
// @Success 200 {object} httpx.Response{data=Lot}
// @Failure 400 {object} httpx.Response "Неверный ID лота"
// @Failure 404 {object} httpx.Response "Лот не найден"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/lot/{id} [get]
func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
		return
	}

	lot, err := h.service.getByID(r.Context(), id)
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.getByID", err)
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, ErrLotNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
			return
		}
		if errors.Is(err, ErrInvalidID) {
			httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
			return
		}
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
		Data: lot,
	})
}

// getMyLots возвращает список всех лотов текущего авторизованного пользователя (продавца).
//
// @Summary Получить мои лоты
// @Description Возвращает все лоты, созданные текущим пользователем
// @Tags lot
// @Security BearerAuth
// @Produce json
// @Success 200 {object} httpx.Response{data=[]Lot}
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/lot/my [get]
func (h *Handler) getMyLots(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.getMyLots.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	lots, err := h.service.getMyLots(r.Context(), uid)
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.getMyLots", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
		Data: lots,
	})
}

// create создает новый лот для предмета текущего пользователя.
//
// @Summary Создать новый лот
// @Description Создает лот на аукционе для принадлежащего пользователю предмета со статусом draft или pending_review
// @Tags lot
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body reqNewLot true "Параметры нового лота"
// @Success 201 {object} httpx.Response{data=Lot}
// @Failure 400 {object} httpx.Response "Некорректные параметры запроса"
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 404 {object} httpx.Response "Предмет не найден или не принадлежит пользователю"
// @Failure 409 {object} httpx.Response "Лот с данным предметом уже активен"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/lot/ [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.create.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	req, err := httpx.DecodeJSON[reqNewLot](w, r)
	if err != nil {
		return
	}

	if err := req.check(); err != nil {
		apperr.Log(r.Context(), "lotHandler.create.check", err)
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusBadRequest,
			Error: err.Error(),
		})
		return
	}

	lot, err := h.service.createLot(r.Context(), uid, req)
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.create.service", err)
		if errors.Is(err, storage.ErrNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
			return
		}
		if errors.Is(err, storage.ErrUniqueViolation) {
			httpx.WriteResponse(w, httpx.ErrRespFieldIsTaken)
			return
		}
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusCreated,
		Data: lot,
	})
}

// update обновляет параметры существующего лота продавцом.
//
// @Summary Обновить лот
// @Description Обновляет параметры лота продавцом по его ID (статус draft или pending_review)
// @Tags lot
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID лота"
// @Param input body reqNewLot true "Обновленные параметры лота"
// @Success 200 {object} httpx.Response{data=Lot}
// @Failure 400 {object} httpx.Response "Неверный ID или некорректные параметры"
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 404 {object} httpx.Response "Лот не найден или не принадлежит пользователю"
// @Failure 409 {object} httpx.Response "Конфликт уникальности предмета"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/lot/{id} [put]
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.update.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
		return
	}

	req, err := httpx.DecodeJSON[reqNewLot](w, r)
	if err != nil {
		return
	}

	if err := req.check(); err != nil {
		apperr.Log(r.Context(), "lotHandler.update.check", err)
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusBadRequest,
			Error: err.Error(),
		})
		return
	}

	lot, err := h.service.updateLot(r.Context(), id, uid, req)
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.update.service", err)
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, ErrLotNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
			return
		}
		if errors.Is(err, ErrInvalidID) {
			httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
			return
		}
		if errors.Is(err, storage.ErrUniqueViolation) {
			httpx.WriteResponse(w, httpx.ErrRespFieldIsTaken)
			return
		}
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
		Data: lot,
	})
}

// delete отменяет лот текущего продавца.
//
// @Summary Отменить лот
// @Description Отменяет лот продавца, если он еще не завершен и до старта больше часа
// @Tags lot
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID лота"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response "Неверный ID лота"
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 404 {object} httpx.Response "Лот не найден, чужой или не может быть отменен"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/lot/{id} [delete]
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.delete.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
		return
	}

	err = h.service.cancelLot(r.Context(), id, uid)
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.delete", err)
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, ErrLotNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
			return
		}
		if errors.Is(err, ErrInvalidID) {
			httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
			return
		}
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
	})
}

type activateReq struct {
	StartedAt *time.Time `json:"started_at,omitempty"`
	EndedAt   *time.Time `json:"ends_at,omitempty"`
}

// activate переводит одобренный лот (со статусом approved) в активный статус (active).
//
// @Summary Активировать лот
// @Description Переводит лот продавца со статусом approved в статус active для старта торгов
// @Tags lot
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID лота"
// @Param input body activateReq false "Опциональные даты начала и окончания торгов"
// @Success 200 {object} httpx.Response{data=Lot}
// @Failure 400 {object} httpx.Response "Неверный ID лота или лот не в статусе approved"
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 403 {object} httpx.Response "Доступ запрещен"
// @Failure 404 {object} httpx.Response "Лот не найден"
// @Failure 409 {object} httpx.Response "Лот с данным предметом уже активен"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/lot/{id}/activate [post]
func (h *Handler) activate(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.activate.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
		return
	}

	req, err := httpx.DecodeJSON[activateReq](w, r)
	if err != nil {
		return
	}

	lot, err := h.service.activateLot(r.Context(), id, uid, req.StartedAt, req.EndedAt)
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.activate", err)
		if errors.Is(err, ErrLotNotApproved) {
			httpx.WriteResponse(w, httpx.Response{
				Code:  http.StatusBadRequest,
				Error: ErrLotNotApproved.Error(),
			})
			return
		}
		if errors.Is(err, ErrForbidden) {
			httpx.WriteResponse(w, httpx.Response{Code: http.StatusForbidden, Error: "forbidden"})
			return
		}
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, ErrLotNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
			return
		}
		if errors.Is(err, ErrInvalidID) || errors.Is(err, ErrInvalidReqBody) {
			httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
			return
		}
		if errors.Is(err, storage.ErrUniqueViolation) {
			httpx.WriteResponse(w, httpx.ErrRespFieldIsTaken)
			return
		}
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
		Data: lot,
	})
}

// RouterLot api/lot/
func (h *Handler) RouterLot(authChain alice.Chain) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.list)
	mux.HandleFunc("GET /{id}", h.getByID)
	mux.Handle("GET /my", authChain.ThenFunc(h.getMyLots))

	mux.Handle("POST /", authChain.ThenFunc(h.create))
	mux.Handle("PUT /{id}", authChain.ThenFunc(h.update))
	mux.Handle("DELETE /{id}", authChain.ThenFunc(h.delete))
	mux.Handle("POST /{id}/activate", authChain.ThenFunc(h.activate))
	return mux
}

type reqBid struct {
	Amount         int       `json:"amount"`
	IdempotencyKey uuid.UUID `json:"idempotency_key"`
}

func (h *Handler) bid(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.activate.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
		return
	}

	req, err := httpx.DecodeJSON[reqBid](w, r)
	if err != nil {
		return
	}

	if err = h.service.bid(r.Context(), id, uid, req.IdempotencyKey, req.Amount); err != nil {
		apperr.Log(r.Context(), "lotHandler.bid", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
		//TODO:
	}
}

// RouterBid api/bid/
func (h *Handler) RouterBid(authChain alice.Chain) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /{id}", h.bid)

	return authChain.Then(mux)
}

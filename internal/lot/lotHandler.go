package lot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	ItemID      int64       `json:"item_id"`
	Status      Status      `json:"status"`
	StartAmount int64       `json:"start_amount"`
	MinStep     int64       `json:"min_step"`
	Title       string      `json:"title"`
	Description Description `json:"description"`
	StartedAt   *time.Time  `json:"started_at"`
	EndedAt     *time.Time  `json:"ends_at"`
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
	if r.Description.Type != "" {
		switch r.Description.Type {
		case mimeType.Image, mimeType.Model3D, mimeType.Audio, mimeType.Video, mimeType.Document, mimeType.Archive:
		default:
			return errors.New("invalid description type")
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
// @Param Idempotency-Key header string false "Ключ идемпотентности"
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
// @Param Idempotency-Key header string false "Ключ идемпотентности"
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
		if errors.Is(err, storage.ErrForbidden) {
			httpx.WriteResponse(w, httpx.ErrForbidden)
			return
		}
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
// @Param id path int true "ID лота"
// @Param Idempotency-Key header string false "Ключ идемпотентности"
// @Accept json
// @Produce json
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
// @Param id path int true "ID лота"
// @Param Idempotency-Key header string false "Ключ идемпотентности"
// @Produce json
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
// @Param id path int true "ID лота"
// @Param Idempotency-Key header string false "Ключ идемпотентности"
// @Accept json
// @Produce json
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

func (h *Handler) approved(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
		return
	}
	if err = h.service.approve(r.Context(), id); err != nil {
		apperr.Log(r.Context(), "lotHandler.approved", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
	}
	httpx.WriteResponse(w, httpx.Response{Code: http.StatusOK})
}

// RouterLot api/lot/
func (h *Handler) RouterLot(authChain alice.Chain, adminChain alice.Chain) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.list)
	mux.HandleFunc("GET /{id}", h.getByID)
	mux.Handle("GET /my", authChain.ThenFunc(h.getMyLots))

	mux.Handle("POST /", authChain.ThenFunc(h.create))
	mux.Handle("PUT /{id}", authChain.ThenFunc(h.update))
	mux.Handle("DELETE /{id}", authChain.ThenFunc(h.delete))
	mux.Handle("POST /{id}/activate", authChain.ThenFunc(h.activate))

	mux.Handle("POST /{id}/approved", adminChain.ThenFunc(h.approved))
	return mux
}

type reqBid struct {
	Amount int `json:"amount"`
}

// bid размещает новую ставку на активный лот.
//
// @Summary Разместить ставку на лот
// @Description Размещает новую ставку авторизованного пользователя на активный лот аукциона
// @Tags bid
// @Security BearerAuth
// @Param id path int true "ID лота"
// @Param Idempotency-Key header string false "Ключ идемпотентности"
// @Accept json
// @Produce json
// @Param input body reqBid true "Параметры ставки"
// @Success 200 {object} httpx.Response "Ставка успешно принята"
// @Failure 400 {object} httpx.Response "Некорректная сумма ставки, меньший шаг или нехватка баланса"
// @Failure 401 {object} httpx.Response "Не авторизован"
// @Failure 403 {object} httpx.Response "Продавец не может делать ставки на свой лот"
// @Failure 404 {object} httpx.Response "Лот не найден или не активен"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/bid/{id} [post]
func (h *Handler) bid(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "lotHandler.bid.GetUID", err)
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

	if err = h.service.bid(r.Context(), id, uid, req.Amount); err != nil {
		apperr.Log(r.Context(), "lotHandler.bid", err)
		if errors.Is(err, ErrLider) || errors.Is(err, ErrAmountSmall) || errors.Is(err, ErrInsufficientBalance) || errors.Is(err, ErrInvalidReqBody) {
			httpx.WriteResponse(w, httpx.Response{
				Code:  http.StatusBadRequest,
				Error: err.Error(),
			})
			return
		}
		if errors.Is(err, ErrForbidden) || errors.Is(err, storage.ErrForbidden) {
			httpx.WriteResponse(w, httpx.Response{
				Code:  http.StatusForbidden,
				Error: "forbidden: seller cannot bid on own lot",
			})
			return
		}
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, ErrLotNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
			return
		}
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
	})
}

// streamBids транслирует историю ставок лота через SSE каждые 2 секунды.
//
// @Summary SSE поток ставок лота
// @Description Устанавливает Server-Sent Events соединение и отправляет список ставок каждые 2 секунды
// @Tags bid
// @Produce text/event-stream
// @Param id path int true "ID лота"
// @Success 200 {array} Bid "Поток ставок"
// @Failure 400 {object} httpx.Response "Неверный ID лота"
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/bid/{id}/stream [get]
func (h *Handler) streamBids(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteResponse(w, httpx.Response{
			Code:  http.StatusInternalServerError,
			Error: "Streaming unsupported",
		})
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sendBids := func() error {
		bids, err := h.service.getLotBids(r.Context(), id)
		if err != nil {
			return err
		}
		data, err := json.Marshal(bids)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	if err := sendBids(); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(r.Context().Err(), context.Canceled) {
			apperr.Log(r.Context(), "lotHandler.streamBids.initial", err)
		}
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := sendBids(); err != nil {
				if !errors.Is(err, context.Canceled) && !errors.Is(r.Context().Err(), context.Canceled) {
					apperr.Log(r.Context(), "lotHandler.streamBids.tick", err)
				}
				return
			}
		}
	}
}

// RouterBid api/bid/
func (h *Handler) RouterBid(authChain alice.Chain) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /{id}", h.bid)
	mux.HandleFunc("GET /{id}/stream", h.streamBids)

	return authChain.Then(mux)
}

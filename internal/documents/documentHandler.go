package documents

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/cmd/httpx"
	"github.com/ArtemChadaev/Auction/cmd/storage"
	"github.com/justinas/alice"
)

type service interface {
	GetActiveDocuments(ctx context.Context) ([]Document, error)
	GetDocumentByID(ctx context.Context, id int64) (Document, error)
	GetActiveDocumentByName(ctx context.Context, name string) (Document, error)

	CreateConsent(ctx context.Context, uid uuid.UUID, docID int64) (UserConsent, error)
	GetUserConsents(ctx context.Context, uid uuid.UUID) ([]UserConsent, error)
	GetConsentByDocumentID(ctx context.Context, docID int64, uid uuid.UUID) (UserConsent, error)
}

type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request) {
	if name := strings.TrimSpace(r.URL.Query().Get("name")); name != "" {
		doc, err := h.service.GetActiveDocumentByName(r.Context(), name)
		if err != nil {
			apperr.Log(r.Context(), "listDocuments.GetActiveDocumentByName", err)
			if errors.Is(err, storage.ErrNotFound) || errors.Is(err, ErrDocumentNotFound) {
				httpx.WriteResponse(w, httpx.ErrRespNotFound)
				return
			}
			httpx.WriteResponse(w, httpx.ErrRespInternalServer)
			return
		}
		httpx.WriteResponse(w, httpx.Response{Code: http.StatusOK, Data: doc})
		return
	}

	docs, err := h.service.GetActiveDocuments(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "listDocuments.GetActiveDocuments", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}
	httpx.WriteResponse(w, httpx.Response{Code: http.StatusOK, Data: docs})
}

func (h *Handler) getDocumentByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		appErr := fmt.Errorf("getDocumentByID.ParseInt: %w", ErrInvalidID)
		apperr.Log(r.Context(), "getDocumentByID.ParseInt", appErr)
		httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
		return
	}

	doc, err := h.service.GetDocumentByID(r.Context(), id)
	if err != nil {
		apperr.Log(r.Context(), "getDocumentByID.service", err)
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, ErrDocumentNotFound) {
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
	httpx.WriteResponse(w, httpx.Response{Code: http.StatusOK, Data: doc})
}

func (h *Handler) getActiveDocumentByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		appErr := fmt.Errorf("getActiveDocumentByName.emptyName: %w", ErrDocumentNotFound)
		apperr.Log(r.Context(), "getActiveDocumentByName.emptyName", appErr)
		httpx.WriteResponse(w, httpx.ErrRespNotFound)
		return
	}

	doc, err := h.service.GetActiveDocumentByName(r.Context(), name)
	if err != nil {
		apperr.Log(r.Context(), "getActiveDocumentByName.service", err)
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, ErrDocumentNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
			return
		}
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}
	httpx.WriteResponse(w, httpx.Response{Code: http.StatusOK, Data: doc})
}

func (h *Handler) createConsent(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "createConsent.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	req, err := httpx.DecodeJSON[struct {
		DocumentID int64 `json:"document_id"`
	}](w, r)
	if err != nil {
		apperr.Log(r.Context(), "createConsent.DecodeJSON", err)
		return
	}

	consent, err := h.service.CreateConsent(r.Context(), uid, req.DocumentID)
	if err != nil {
		apperr.Log(r.Context(), "createConsent.service", err)
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, storage.ErrForeignKeyViolation) || errors.Is(err, ErrDocumentNotFound) {
			httpx.WriteResponse(w, httpx.ErrRespNotFound)
			return
		}
		if errors.Is(err, ErrMissingDocumentID) {
			httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
			return
		}
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{Code: http.StatusCreated, Data: consent})
}

func (h *Handler) getUserConsents(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "getUserConsents.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	consents, err := h.service.GetUserConsents(r.Context(), uid)
	if err != nil {
		apperr.Log(r.Context(), "getUserConsents.service", err)
		httpx.WriteResponse(w, httpx.ErrRespInternalServer)
		return
	}

	httpx.WriteResponse(w, httpx.Response{Code: http.StatusOK, Data: consents})
}

func (h *Handler) getConsentByDocumentID(w http.ResponseWriter, r *http.Request) {
	uid, err := cfg.GetUID(r.Context())
	if err != nil {
		apperr.Log(r.Context(), "getConsentByDocumentID.GetUID", err)
		httpx.WriteResponse(w, httpx.ErrRespInvalidAuth)
		return
	}

	idStr := r.PathValue("id")
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		appErr := fmt.Errorf("getConsentByDocumentID.ParseInt: %w", ErrInvalidID)
		apperr.Log(r.Context(), "getConsentByDocumentID.ParseInt", appErr)
		httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
		return
	}

	consent, err := h.service.GetConsentByDocumentID(r.Context(), docID, uid)
	if err != nil {
		apperr.Log(r.Context(), "getConsentByDocumentID.service", err)
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, ErrConsentNotFound) {
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

	httpx.WriteResponse(w, httpx.Response{Code: http.StatusOK, Data: consent})
}

// Routes returns routes for /api/documents/
func (h *Handler) Routes(authChain alice.Chain) http.Handler {
	mux := http.NewServeMux()

	// Public document endpoints
	mux.HandleFunc("GET /{$}", h.listDocuments)
	mux.HandleFunc("GET /name/{name}", h.getActiveDocumentByName)
	mux.HandleFunc("GET /{id}", h.getDocumentByID)

	// User consent endpoints
	mux.Handle("POST /consents", authChain.ThenFunc(h.createConsent))
	mux.Handle("GET /consents", authChain.ThenFunc(h.getUserConsents))
	mux.Handle("GET /consents/{id}", authChain.ThenFunc(h.getConsentByDocumentID))

	return mux
}

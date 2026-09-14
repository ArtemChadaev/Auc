package documents

import (
	"context"
	"encoding/json"
	"net/http"
	"net/netip"
)

type DocStore interface {
	AccessUserDocument(ctx context.Context, userId int64, docID int64, ip *netip.Addr, source string) error
}

type DocHandler struct {
	store DocStore
}

func NewDocHandler(store DocStore) *DocHandler {
	return &DocHandler{store: store}
}

type docReq struct {
	DocID  int64  `json:"doc_id"`
	Source string `json:"source"`
}

func (h *DocHandler) AccessUserDoc(w http.ResponseWriter, r *http.Request) {
	var req docReq

	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)) // максимум 1 МБ
	dec.DisallowUnknownFields()                                   // опечатка в поле → ошибка, а не тишина
	if err := dec.Decode(&req); err != nil {
		return
	}
	//TODO: Надо middleware для получения userID/ip
	var id int64 = 1

	if err := h.store.AccessUserDocument(context.Background(), id, req.DocID, nil, req.Source); err != nil {
		http.Error(w, "err", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

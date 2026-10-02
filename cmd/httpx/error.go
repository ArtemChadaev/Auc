package httpx

import (
	"net/http"
)

// Строки ошибок
var (
	ReqEntityTooLarge = "request entity too large"
	BadFile           = "bad file"
)

var (
	ErrRespNotFound       = Response{Code: http.StatusNotFound, Error: "not found"}
	ErrRespInvalidReqBody = Response{Code: http.StatusBadRequest, Error: "invalid request body"}
	ErrRespInvalidAuth    = Response{Code: http.StatusUnauthorized, Error: "invalid authorization"}
	ErrRespInternalServer = Response{Code: http.StatusInternalServerError, Error: "internal server error"}
	ErrRespFieldIsTaken   = Response{Code: http.StatusConflict, Error: "field is taken"}
)

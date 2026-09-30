package httpx

import (
	"errors"
)

var (
	ErrReqBody = errors.New("invalid request body")
)

// Строки ошибок
var (
	NotFound          = "not found"
	FieldIsTaken      = "field is taken"
	InvalidReqBody    = "invalid request body"
	InvalidAuth       = "invalid authorization"
	InvalidToken      = "invalid token"
	InternalServer    = "internal server error"
	ReqEntityTooLarge = "request entity too large"
	BadFile           = "bad file"
)

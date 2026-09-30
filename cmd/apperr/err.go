package apperr

import "errors"

var (
	// ErrError Для ошибок требующих логирования Error опасные вляют на всю программу
	ErrError = errors.New("error")
	// ErrWarn Для ошибок требующих логирования Warn влияют на часть программы
	ErrWarn = errors.New("warn")
	// ErrDebug Для ошибок от пользователя требующих логирования только при тесте
	ErrDebug = errors.New("debug")

	ErrUnauthorized    = errors.New("unauthorized")
	ErrNotFoundContext = errors.New("not found in context")
	ErrForbidden       = errors.New("forbidden")

	// ErrTimeout для контекста
	ErrTimeout = errors.New("timeout")

	ErrEntityTooLarge = errors.New("entity too large")
)

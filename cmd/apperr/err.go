package apperr

import "errors"

var (
	// ErrError Для ошибок требующих логирования Error опасные вляют на всю программу
	ErrError = errors.New("error")
	// ErrWarn Для ошибок требующих логирования Warn влияют на часть программы
	ErrWarn = errors.New("warn")

	ErrInvalidRequest = errors.New("invalid request")
)

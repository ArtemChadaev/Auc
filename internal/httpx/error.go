package httpx

import "net/http"

type ErrorType int

const (
	Fatal ErrorType = iota
	Internal
	Client
)

type AppError struct {
	Type       ErrorType
	StatusCode int
	Err        error
}

func (e *AppError) Error() string {
	return e.Err.Error()
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewErrorFatal(err error) error {
	return &AppError{
		Type:       Fatal,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

func NewErrorInternal(err error) error {
	return &AppError{
		Type:       Internal,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}
func NewErrorClient(code int, err error) error {
	return &AppError{
		Type:       Client,
		StatusCode: code,
		Err:        err,
	}
}

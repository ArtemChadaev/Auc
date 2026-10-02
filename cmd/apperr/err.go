package apperr

import (
	"context"
	"errors"
	"log/slog"
	"runtime"
	"time"
)

var (
	ErrUnauthorized    = NewAppErrorString("unauthorized", -4)
	ErrNotFoundContext = NewAppErrorString("not found in context", -4)
	ErrForbidden       = NewAppErrorString("forbidden", -4)

	// ErrTimeout для контекста
	ErrTimeout = NewAppErrorString("timeout", -4)

	ErrEntityTooLarge = NewAppErrorString("entity too large", -4)
)

type AppError struct {
	Err   error
	Level slog.Level
}

func (e AppError) Error() string {
	return e.Err.Error()
}

func (e AppError) Unwrap() error {
	return e.Err
}

func NewAppError(err error, level slog.Level) AppError {
	return AppError{Err: err, Level: level}
}

func NewAppErrorString(err string, level slog.Level) AppError {
	return AppError{Err: errors.New(err), Level: level}
}

// Log Выводит в логи ошибку смотря на её уровень (использовать при неявном error)
func Log(ctx context.Context, msg string, err error, args ...any) {
	logger := slog.Default()

	// Специально 6 левел, значит ошибка но какая хз, т.к. забыл точно указать лвл
	var level slog.Level = 6

	//Очень странная ситуация, поэтому и 7 лвл
	if err == nil {
		level = 7
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		level = appErr.Level
	}

	if !logger.Enabled(ctx, level) {
		return
	}

	//Чтобы строка вызыва не тут
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])

	record := slog.NewRecord(time.Now(), level, msg, pcs[0])

	if err != nil {
		record.Add("error", err)
	}
	record.Add(args...)
	_ = logger.Handler().Handle(ctx, record)
}

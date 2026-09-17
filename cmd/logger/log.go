package logger

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
)

// Ошибки валидируются в handler (при api)
// Все ошибки создаются в виде кастомного типа error Example: ErrNotFound
// При возврате существующей ошибки используется fmt.Errorf("имяФайла.Метод: %w", err)
// Example: fmt.Errorf("userRepo.findAccessToken: %w", err)
// Если ошибка изза пользователя -> debug, чтото не так в программе Warn,
// Ошибка мешает работать полностью (отказ db) Error (при возможности починки перезапуск)

// Логи старта приложения и запуска Info, остальное debug
type ContextHandler struct {
	slog.Handler
}

func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if email, ok := ctx.Value(cfg.Email).(string); ok {
		r.AddAttrs(slog.String(string(cfg.Email), email))
	}
	if ip, ok := ctx.Value(cfg.IP).(string); ok {
		r.AddAttrs(slog.String(string(cfg.IP), ip))
	}
	if uid, ok := ctx.Value(cfg.UID).(string); ok {
		r.AddAttrs(slog.String(string(cfg.UID), uid))
	}
	if path, ok := ctx.Value(cfg.Path).(string); ok {
		r.AddAttrs(slog.String(string(cfg.Path), path))
	}
	return h.Handler.Handle(ctx, r)
}

func Init() {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == string(cfg.Email) {
				parts := strings.Split(a.Key, "@")
				if len(parts) != 2 {
					a.Key = "***"
				} else {
					runes := []rune(parts[0])
					a.Key = fmt.Sprintf("%c***%c@%s", runes[0], runes[len(runes)-1], parts[1])
				}
			}
			if a.Key == string(cfg.IP) {
				if ip := net.ParseIP(a.Key).To4(); ip != nil {
					a.Key = ip.Mask(net.CIDRMask(24, 32)).String() + "/24"
				} else {
					a.Key = "0.0.0.0"
				}
			}
			return a
		}, // Скрываем ip и email чтоб не дое..ставали
	}

	slog.SetDefault(slog.New(ContextHandler{Handler: slog.NewJSONHandler(os.Stdout, opts)}))
}

package logger

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
)

// Ошибки валидируются в handler (при api)
// Все ошибки создаются в виде кастомного типа error Example: ErrNotFound
// При возврате существующей ошибки используется fmt.Errorf("имяФайла.Метод: %w", err)
// Example: fmt.Errorf("userRepo.findAccessToken: %w", err)
// Если ошибка изза пользователя -> debug, чтото не так в программе Warn,
// Ошибка мешает работать полностью (отказ db) Error (при возможности починки перезапуск)

// Логи старта приложения и запуска Info, остальное debug

func Init() {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "email" {
				parts := strings.Split(a.Key, "@")
				if len(parts) != 2 {
					a.Key = "***"
				} else {
					runes := []rune(parts[0])
					a.Key = fmt.Sprintf("%c***%c@%s", runes[0], runes[len(runes)-1], parts[1])
				}
			}
			if a.Key == "ip" {
				if ip := net.ParseIP(a.Key).To4(); ip != nil {
					a.Key = ip.Mask(net.CIDRMask(24, 32)).String() + "/24"
				} else {
					a.Key = "0.0.0.0"
				}
			}
			return a
		}, // Скрываем ip и email чтоб не дое..ставали
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, opts)))
}

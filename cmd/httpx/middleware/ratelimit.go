package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/cmd/httpx"
	"github.com/ArtemChadaev/Auction/cmd/valkey"
	"github.com/ArtemChadaev/Auction/internal/user"
	"github.com/justinas/alice"
	"github.com/redis/go-redis/v9"
)

// ClientInfo содержит подробные сведения о клиенте для точной идентификации и учёта лимитов
type ClientInfo struct {
	ID        string // "user:<uid>" для авторизованных, "guest:<guest_id>:<fp>" для гостей
	IsAuth    bool
	IP        string
	UserAgent string
	GuestID   string
}

// extractIP извлекает реальный IP адрес клиента с учетом обратных прокси
func extractIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		if comma := strings.Index(ip, ","); comma != -1 {
			ip = strings.TrimSpace(ip[:comma])
		}
	}
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			ip = host
		} else {
			ip = r.RemoteAddr
		}
	}
	return ip
}

// getClientInfo выполняет точную идентификацию клиента:
//  1. Для авторизованных пользователей: "user:<uid>"
//  2. Для гостей: объединяет постоянный cookie идентификатор устройства (guest_id),
//     IP-адрес и цифровой отпечаток браузера (User-Agent, Accept-Language, Sec-CH-UA).
//     Это позволяет независимым гостям за одним NAT/Wi-Fi иметь раздельные квоты,
//     одновременно связывая сессию с конкретным браузером и окружением.
func getClientInfo(w http.ResponseWriter, r *http.Request) ClientInfo {
	ip := extractIP(r)
	ua := r.UserAgent()

	// 1. Проверяем, есть ли UID в контексте (если мидлварь вызвана после AuthAccessToken)
	if uid, err := cfg.GetUID(r.Context()); err == nil {
		return ClientInfo{
			ID:        "user:" + uid.String(),
			IsAuth:    true,
			IP:        ip,
			UserAgent: ua,
		}
	}

	// 2. Если мидлварь вызвана до AuthAccessToken (в globalChain), проверяем Authorization header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if uid, _, err := user.VerifyAccessToken(token); err == nil {
			return ClientInfo{
				ID:        "user:" + uid.String(),
				IsAuth:    true,
				IP:        ip,
				UserAgent: ua,
			}
		}
	}

	// 3. Для неавторизованного гостя: извлекаем или генерируем устойчивый guest_id
	var guestID string
	if cookie, err := r.Cookie("guest_id"); err == nil && cookie.Value != "" {
		guestID = strings.TrimSpace(cookie.Value)
	} else if headerVal := r.Header.Get("X-Device-Id"); headerVal != "" {
		guestID = strings.TrimSpace(headerVal)
	} else if headerVal := r.Header.Get("X-Guest-Id"); headerVal != "" {
		guestID = strings.TrimSpace(headerVal)
	}

	if guestID == "" {
		guestID = uuid.NewV7().String()
		if w != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     "guest_id",
				Value:    guestID,
				Path:     "/",
				MaxAge:   365 * 24 * 3600, // 1 год
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}
		// Добавляем cookie в r, чтобы последующие мидлвари в запросе видели его
		r.AddCookie(&http.Cookie{
			Name:  "guest_id",
			Value: guestID,
		})
	}

	// Вычисляем цифровой отпечаток браузера (Fingerprint)
	lang := r.Header.Get("Accept-Language")
	chUA := r.Header.Get("Sec-CH-UA")
	chPlatform := r.Header.Get("Sec-CH-UA-Platform")

	fpRaw := fmt.Sprintf("%s|%s|%s|%s|%s", ip, ua, lang, chUA, chPlatform)
	sum := sha256.Sum256([]byte(fpRaw))
	fp := hex.EncodeToString(sum[:8]) // 16 hex-символов

	id := fmt.Sprintf("guest:%s:%s", guestID, fp)

	return ClientInfo{
		ID:        id,
		IsAuth:    false,
		IP:        ip,
		UserAgent: ua,
		GuestID:   guestID,
	}
}

// recordViolation фиксирует нарушение лимита запросов:
// 1. Помещает детальное событие в очередь Redis (ratelimit:violations:queue) для последующей выгрузки в PostgreSQL и S3.
// 2. Инкрементирует общие и персональные счетчики в Redis.
// 3. Записывает метаданные последнего нарушения с TTL 30 дней.
// 4. Логирует предупреждение через slog.Warn.
func recordViolation(
	ctx context.Context,
	vk *valkey.Client,
	id, ip, ua, method, path string,
	currentCount int64,
	limit int,
	violationType string,
) {
	if vk == nil || vk.Raw() == nil {
		slog.Warn("rate_limit_violation",
			"identifier", id,
			"ip", ip,
			"endpoint", method+" "+path,
			"current_requests", currentCount,
			"limit", limit,
			"type", violationType,
		)
		return
	}

	rdb := vk.Raw()
	now := time.Now().UTC()

	// 1. Событие для очереди периодической выгрузки
	rec := ViolationRecord{
		Identifier:    id,
		IP:            ip,
		UserAgent:     ua,
		Endpoint:      method + " " + path,
		RequestCount:  currentCount,
		LimitCount:    limit,
		ViolationType: violationType,
		CreatedAt:     now,
	}
	jsonData, _ := json.Marshal(rec)

	pipe := rdb.Pipeline()

	// 2. Счётчик нарушений пользователя/гостя
	pipe.HIncrBy(ctx, "ratelimit:violations:total", id, 1)
	pipe.Incr(ctx, "ratelimit:violations:"+id)

	// 3. Детали последнего нарушения (путь, время, IP, User-Agent, тип)
	pipe.HSet(ctx, "ratelimit:violation_info:"+id,
		"last_path", method+" "+path,
		"last_ip", ip,
		"last_ua", ua,
		"last_time", now.Format(time.RFC3339),
		"type", violationType,
	)
	pipe.Expire(ctx, "ratelimit:violation_info:"+id, 30*24*time.Hour)

	// 4. Помещаем в очередь на выгрузку (с ограничением размера очереди)
	pipe.RPush(ctx, "ratelimit:violations:queue", string(jsonData))
	pipe.LTrim(ctx, "ratelimit:violations:queue", -50000, -1)

	totalViolations, err := pipe.Exec(ctx)
	var totalCount int64
	if err == nil && len(totalViolations) > 0 {
		if val, ok := totalViolations[0].(*redis.IntCmd); ok {
			totalCount = val.Val()
		}
	}

	slog.Warn("rate_limit_violation",
		"identifier", id,
		"ip", ip,
		"endpoint", method+" "+path,
		"current_requests", currentCount,
		"limit", limit,
		"type", violationType,
		"total_violations", totalCount,
	)
}

// GlobalRateLimit ограничивает суммарное число запросов в минуту на все эндпоинты:
// - authLimit: лимит для авторизованных пользователей (например, 60 запросов/мин)
// - guestLimit: лимит для неавторизованных гостей с точной идентификацией (например, 20 запросов/мин)
// Также содержит дополнительный предохранитель по IP (100 запросов/мин на всю подсеть/IP),
// чтобы злоумышленник не мог обходить лимит частым сбросом cookie.
func GlobalRateLimit(vk *valkey.Client, authLimit, guestLimit int) alice.Constructor {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if vk == nil || vk.Raw() == nil {
				next.ServeHTTP(w, r)
				return
			}

			info := getClientInfo(w, r)
			limit := guestLimit
			if info.IsAuth {
				limit = authLimit
			}

			// Минутный бакет
			minuteBucket := time.Now().UTC().Unix() / 60
			redisKey := fmt.Sprintf("ratelimit:global:%s:%d", info.ID, minuteBucket)

			count, err := vk.Raw().Incr(r.Context(), redisKey).Result()
			if err != nil {
				// Fail-open при временном сбое Redis, чтобы не ронять сервис
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				_ = vk.Raw().Expire(r.Context(), redisKey, 70*time.Second).Err()
			}

			if count > int64(limit) {
				recordViolation(r.Context(), vk, info.ID, info.IP, info.UserAgent, r.Method, r.URL.Path, count, limit, "global")
				w.Header().Set("Retry-After", "60")
				httpx.WriteResponse(w, httpx.Response{
					Code:  http.StatusTooManyRequests,
					Error: "global rate limit exceeded: too many requests per minute across all endpoints",
				})
				return
			}

			// Дополнительный предохранитель по IP для неавторизованных гостей
			if !info.IsAuth {
				ipKey := fmt.Sprintf("ratelimit:ip_guest:%s:%d", info.IP, minuteBucket)
				ipCount, ipErr := vk.Raw().Incr(r.Context(), ipKey).Result()
				if ipErr == nil {
					if ipCount == 1 {
						_ = vk.Raw().Expire(r.Context(), ipKey, 70*time.Second).Err()
					}
					if ipCount > 100 {
						recordViolation(r.Context(), vk, "ip:"+info.IP, info.IP, info.UserAgent, r.Method, r.URL.Path, ipCount, 100, "ip_safety")
						w.Header().Set("Retry-After", "60")
						httpx.WriteResponse(w, httpx.Response{
							Code:  http.StatusTooManyRequests,
							Error: "ip rate limit exceeded: too many guest requests from this IP",
						})
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// EndpointRateLimit ограничивает число запросов в минуту к конкретному эндпоинту:
// - endpointName: имя или категория эндпоинта (например, "upload_photo", "list_lots")
// - authLimit: лимит для авторизованных пользователей (например, 15 запросов/мин)
// - guestLimit: лимит для неавторизованных пользователей (например, 5 запросов/мин)
func EndpointRateLimit(vk *valkey.Client, endpointName string, authLimit, guestLimit int) alice.Constructor {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if vk == nil || vk.Raw() == nil {
				next.ServeHTTP(w, r)
				return
			}

			info := getClientInfo(w, r)
			limit := guestLimit
			if info.IsAuth {
				limit = authLimit
			}

			minuteBucket := time.Now().UTC().Unix() / 60
			redisKey := fmt.Sprintf("ratelimit:ep:%s:%s:%d", endpointName, info.ID, minuteBucket)

			count, err := vk.Raw().Incr(r.Context(), redisKey).Result()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				_ = vk.Raw().Expire(r.Context(), redisKey, 70*time.Second).Err()
			}

			if count > int64(limit) {
				recordViolation(r.Context(), vk, info.ID, info.IP, info.UserAgent, r.Method, r.URL.Path, count, limit, "endpoint:"+endpointName)
				w.Header().Set("Retry-After", "60")
				httpx.WriteResponse(w, httpx.Response{
					Code:  http.StatusTooManyRequests,
					Error: fmt.Sprintf("endpoint rate limit exceeded: maximum %d requests per minute allowed", limit),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/storage"
	"github.com/ArtemChadaev/Auction/cmd/valkey"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

const (
	UserStatusCacheTTL = 24 * time.Hour
	UserStatusDeleted  = "1"
	UserStatusActive   = "0"
)

// UserDeletedKey returns the Redis key used to cache user deleted status.
// Key format is kept concise ("u:del:<uuid>") to minimize memory consumption.
func UserDeletedKey(uid uuid.UUID) string {
	return "u:del:" + uid.String()
}

// SetUserDeletedCache updates user deleted status in Redis with a 24-hour TTL.
// isDeleted = true sets "1", isDeleted = false sets "0".
func SetUserDeletedCache(ctx context.Context, vk *valkey.Client, uid uuid.UUID, isDeleted bool) error {
	if vk == nil || vk.Raw() == nil {
		return nil
	}
	val := UserStatusActive
	if isDeleted {
		val = UserStatusDeleted
	}
	return vk.Raw().Set(ctx, UserDeletedKey(uid), val, UserStatusCacheTTL).Err()
}

// CheckUserDeleted checks if a user is deleted, first consulting Redis cache.
// On cache miss, it queries PostgreSQL and stores the status in Redis for 24 hours.
// Returns true if user is deleted or does not exist, false if active.
func CheckUserDeleted(ctx context.Context, vk *valkey.Client, db storage.DBTX, uid uuid.UUID) (bool, error) {
	if vk != nil && vk.Raw() != nil {
		val, err := vk.Raw().Get(ctx, UserDeletedKey(uid)).Result()
		if err == nil {
			return val == UserStatusDeleted, nil
		}
		if !errors.Is(err, redis.Nil) {
			slog.WarnContext(ctx, "failed to get user deleted status from redis", "uid", uid, "error", err)
		}
	}

	if db == nil {
		return false, nil
	}

	var deletedAt *time.Time
	err := db.QueryRow(ctx, "SELECT deleted_at FROM users WHERE id = $1", uid).Scan(&deletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// User does not exist in DB -> treat as deleted / invalid and cache as deleted
			if vk != nil && vk.Raw() != nil {
				_ = vk.Raw().Set(ctx, UserDeletedKey(uid), UserStatusDeleted, UserStatusCacheTTL).Err()
			}
			return true, storage.ErrNotFound
		}
		return false, fmt.Errorf("user.CheckUserDeleted: %w", storage.ErrRepo(err))
	}

	isDeleted := deletedAt != nil
	val := UserStatusActive
	if isDeleted {
		val = UserStatusDeleted
	}

	if vk != nil && vk.Raw() != nil {
		if err := vk.Raw().Set(ctx, UserDeletedKey(uid), val, UserStatusCacheTTL).Err(); err != nil {
			slog.WarnContext(ctx, "failed to cache user deleted status in redis", "uid", uid, "error", err)
		}
	}

	return isDeleted, nil
}

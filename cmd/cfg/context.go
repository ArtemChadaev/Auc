package cfg

import (
	"context"
	"fmt"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
)

type contextKey string

var (
	UID   contextKey = "id"
	Email contextKey = "email"
	IP    contextKey = "ip"
	Path  contextKey = "path"
	Role  contextKey = "role"
)

func GetUID(ctx context.Context) (uuid.UUID, error) {
	if uid, ok := ctx.Value(UID).(uuid.UUID); ok {
		return uid, nil
	}
	return uuid.Nil(), fmt.Errorf("context.GetUID: %w", apperr.ErrNotFoundContext)
}

func GetRole(ctx context.Context) (string, error) {
	if role, ok := ctx.Value(Role).(string); ok {
		return role, nil
	}
	return "", fmt.Errorf("context.GetRole: %w", apperr.ErrNotFoundContext)
}

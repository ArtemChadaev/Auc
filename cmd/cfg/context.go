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
)

func GetUID(ctx context.Context) (uuid.UUID, error) {
	if uid, ok := ctx.Value(UID).(uuid.UUID); ok {
		return uid, nil
	}
	return uuid.Nil(), fmt.Errorf("context.GetUID: %w", apperr.ErrNotFoundContext)
}

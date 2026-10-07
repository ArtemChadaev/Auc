package valkey

import (
	"context"
	"fmt"
	"time"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

type Config struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Port == "" {
		cfg.Port = "6379"
	}
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to valkey at %s: %w", addr, apperr.NewAppError(err, 4))
	}

	return &Client{rdb: rdb}, nil
}

func (c *Client) Raw() *redis.Client {
	return c.rdb
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

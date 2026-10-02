package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/cmd/httpx/middleware"
	"github.com/ArtemChadaev/Auction/cmd/logger"
	"github.com/ArtemChadaev/Auction/cmd/storageS3"
	"github.com/ArtemChadaev/Auction/internal/item"
	"github.com/ArtemChadaev/Auction/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justinas/alice"
)

func main() {
	// logger
	logger.Init()
	if err := cfg.Init(); err != nil {
		slog.Error("main.config", err)
		return
	}

	// ctx
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// db
	pool, err := pgxpool.New(context.Background(), "postgresql://test:password@"+cfg.Cfg.HostDB+":5432/db")
	if err != nil {
		slog.Error("main.db", err)
		return
	}
	if err = pool.Ping(ctx); err != nil {
		slog.Error("main.db", err)
		return
	}
	defer pool.Close()

	// s3
	s3, err := storageS3.NewClient(ctx, storageS3.Config{
		Endpoint:  cfg.Cfg.Endpoint,
		Region:    cfg.Cfg.Region,
		AccessKey: cfg.Cfg.AccessKey,
		SecretKey: cfg.Cfg.SecretKey,
		Bucket:    cfg.Cfg.Bucket,
	})

	// http
	mux := http.NewServeMux()
	globalChain := alice.New(middleware.Logger)
	authChain := alice.New(middleware.AuthAccessToken)
	standardChain := alice.New(middleware.MaxBodySize(1 << 20))

	userHandler := user.NewHandler(user.NewService(user.NewRepo(pool)))
	mux.Handle("/api/auth/", http.StripPrefix("/api/auth", standardChain.Then(userHandler.RoutesAuth(authChain))))
	mux.Handle("/api/user/", http.StripPrefix("/api/user", standardChain.Then(authChain.Then(userHandler.RoutesUser()))))

	itemHandler := item.NewHandler(item.NewService(pool, s3, item.NewRepo(pool)))
	mux.Handle("/api/item/", http.StripPrefix("/api/item", standardChain.Then(itemHandler.Router(authChain))))
	mux.Handle("/api/upload/", http.StripPrefix("/api/upload", alice.New(middleware.MaxBodySize(50<<20)).Then(itemHandler.RouterUpload())))

	// М.б добавть сначала globalChain а потом все очень странные роутеры (по типу upload, -> отдельно)
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           globalChain.Then(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       10 * time.Second,
	}

	go func() {
		_ = srv.ListenAndServe()
	}()

	<-ctx.Done()
}

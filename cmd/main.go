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
	"github.com/ArtemChadaev/Auction/cmd/valkey"
	"github.com/ArtemChadaev/Auction/internal/documents"
	"github.com/ArtemChadaev/Auction/internal/item"
	"github.com/ArtemChadaev/Auction/internal/lot"
	"github.com/ArtemChadaev/Auction/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justinas/alice"
)

// @title Auction API
// @version 1.0
// @description REST API документация платформы аукциона Auction.
// @description Для всех эндпоинтов, требующих авторизации (BearerAuth), поддерживается заголовок Idempotency-Key для защиты от повторной обработки (идемпотентность).
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите токен авторизации в формате: Bearer {token}
func main() {
	// logger
	logger.Init()
	if err := cfg.Init(); err != nil {
		slog.Error("main.config", "error", err)
		return
	}

	// ctx
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// db
	pool, err := pgxpool.New(ctx, "postgresql://test:password@"+cfg.Cfg.HostDB+":5432/db")
	if err != nil {
		slog.Error("main.db", "error", err)
		return
	}
	if err = pool.Ping(ctx); err != nil {
		slog.Error("main.db", "error", err)
		return
	}
	defer pool.Close()

	// s3 global (items)
	globalS3, err := storageS3.NewClient(ctx, storageS3.Config{
		Endpoint:  cfg.Cfg.Endpoint,
		Region:    cfg.Cfg.Region,
		AccessKey: cfg.Cfg.GlobalAccessKey,
		SecretKey: cfg.Cfg.GlobalSecretKey,
		Bucket:    cfg.Cfg.GlobalBucket,
	})
	if err != nil {
		slog.Error("main.s3.global", "error", err)
		return
	}

	// s3 photos (user avatar)
	photosS3, err := storageS3.NewClient(ctx, storageS3.Config{
		Endpoint:  cfg.Cfg.Endpoint,
		Region:    cfg.Cfg.Region,
		AccessKey: cfg.Cfg.PhotosAucAccessKey,
		SecretKey: cfg.Cfg.PhotosSecretKey,
		Bucket:    cfg.Cfg.PhotosBucket,
	})
	if err != nil {
		slog.Error("main.s3.photos", "error", err)
		return
	}

	// valkey
	vk, err := valkey.NewClient(ctx, valkey.Config{
		Host: cfg.Cfg.HostValkey,
		Port: cfg.Cfg.PortValkey,
	})
	if err != nil {
		slog.Error("main.valkey", "error", err)
		return
	}
	defer vk.Close()

	// http
	mux := http.NewServeMux()
	globalChain := alice.New(middleware.Logger, middleware.MaxBodySize(5<<20), middleware.Idempotency(vk))
	authChain := alice.New(middleware.AuthAccessToken)
	adminChain := alice.New(middleware.AuthAdmin)
	// user
	userService := user.NewService(user.NewRepo(pool), photosS3)
	if err = userService.CreateAdmin(); err != nil {
		slog.Error("main.user", "error", err)
		return
	}
	userHandler := user.NewHandler(userService)
	mux.Handle("/api/auth/", http.StripPrefix("/api/auth", userHandler.RoutesAuth(authChain)))
	mux.Handle("/api/user/", http.StripPrefix("/api/user", userHandler.RoutesUser(authChain)))

	// item
	itemService := item.NewService(pool, globalS3, item.NewRepo(pool), vk)
	go itemService.RecoverPendingUploads(ctx)
	itemHandler := item.NewHandler(itemService)
	mux.Handle("/api/upload/", http.StripPrefix("/api/upload", itemHandler.RouterUpload(authChain)))
	mux.Handle("/api/item/", http.StripPrefix("/api/item", itemHandler.RouterItem(authChain)))

	// doc
	docHandler := documents.NewHandler(documents.NewService(documents.NewRepo(pool)))
	mux.Handle("/api/document/", http.StripPrefix("/api/document", docHandler.Routes(authChain)))

	// lot
	lotService := lot.NewService(lot.NewRepo(pool), pool)
	go lotService.FinishLots(ctx)
	lotHandler := lot.NewHandler(lotService)
	mux.Handle("/api/lot/", http.StripPrefix("/api/lot", lotHandler.RouterLot(authChain, adminChain)))
	mux.Handle("/api/bid/", http.StripPrefix("/api/bid", lotHandler.RouterBid(authChain)))

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

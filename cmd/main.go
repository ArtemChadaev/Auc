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
	"github.com/ArtemChadaev/Auction/cmd/logger"
	"github.com/ArtemChadaev/Auction/internal/httpx/middleware"
	"github.com/ArtemChadaev/Auction/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justinas/alice"
)

func main() {
	logger.Init()
	if err := cfg.Init(); err != nil {
		slog.Error("main.config", err)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	host := "localhost"
	if cfg.Cfg.Deploy {
		host = "postgres"
	}
	pool, err := pgxpool.New(context.Background(), "postgresql://test:password@"+host+":5432/db")
	if err != nil {
		slog.Error("main.db", err)
		return
	}
	if err = pool.Ping(ctx); err != nil {
		slog.Error("main.db", err)
		return
	}
	defer pool.Close()

	mux := http.NewServeMux()
	globalChain := alice.New(middleware.Logger)
	userHandler := user.NewHandler(user.NewService(user.NewRepo(pool)))
	authChain := alice.New(middleware.AuthAccessToken)
	mux.Handle("/api/auth/", http.StripPrefix("/api/auth", userHandler.RoutesAuth(authChain)))
	mux.Handle("/api/user/", http.StripPrefix("/api/user", authChain.Then(userHandler.RoutesUser())))
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

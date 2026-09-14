package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := cfg.Init(); err != nil {
		log.Fatal("error parsing config", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	host := "localhost"
	if cfg.Cfg.Deploy {
		host = "postgres"
	}
	pool, err := pgxpool.New(context.Background(), "postgresql://test:password@"+host+":5432/db")
	if err != nil {
		panic(err)
	}
	if err := pool.Ping(ctx); err != nil {
		panic(err)
	}
	defer pool.Close()

	mux := http.NewServeMux()

	user.NewHandler(user.NewService(user.NewRepo(pool))).Routes(mux)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       10 * time.Second,
	}

	go func() {
		_ = srv.ListenAndServe()
	}()

	<-ctx.Done()
}

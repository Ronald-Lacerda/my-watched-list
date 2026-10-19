package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"mywatchedlist/api/internal/httpapi"
	"mywatchedlist/internal/config"
	"mywatchedlist/internal/metadata"
)

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return errors.New("configuração de banco inválida")
	}
	defer pool.Close()
	options := httpapi.Options{Origins: strings.Split(os.Getenv("APP_ORIGINS"), ","), NetflixMetadata: metadata.NewNetflix()}
	if token := os.Getenv("TMDB_READ_ACCESS_TOKEN"); token != "" {
		options.Metadata = metadata.New(token)
	}
	handler, err := httpapi.NewApp(pool, options)
	if err != nil {
		return err
	}
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	done := make(chan error, 1)
	go func() { done <- srv.ListenAndServe() }()
	slog.Info("API iniciada", "endereço", cfg.HTTPAddr)
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			return errors.New("não foi possível servir a API")
		}
		return nil
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdown)
	}
}

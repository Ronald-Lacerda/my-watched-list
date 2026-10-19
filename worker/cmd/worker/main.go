package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"mywatchedlist/internal/config"
)

func main() {
	if _, err := config.Load(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	slog.Info("worker iniciado; coleta e agenda serão implementadas na Fase 1")
	<-ctx.Done()
	slog.Info("worker encerrado")
}

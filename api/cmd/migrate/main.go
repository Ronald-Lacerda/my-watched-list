package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"mywatchedlist/api/migrations"
	"mywatchedlist/internal/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}
	if len(os.Args) > 2 || (direction != "up" && direction != "down") {
		return errors.New("uso: migrate [up|down]; down reverte a última migration")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	conn, err := pgx.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return errors.New("configuração de banco inválida")
	}
	db := stdlib.OpenDB(*conn)
	defer db.Close()
	p, err := migrations.New(db)
	if err != nil {
		return errors.New("não foi possível preparar as migrations")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if direction == "down" {
		_, err = p.Down(ctx)
	} else {
		_, err = p.Up(ctx)
	}
	if err != nil {
		return errors.New("falha nas migrations; verifique o banco e a versão do schema")
	}
	slog.Info("migrations concluídas", "direção", direction)
	return nil
}

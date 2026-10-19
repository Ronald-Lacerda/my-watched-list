//go:build integration

// Package testdb cria schemas descartáveis somente em TEST_DATABASE_URL.
package testdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"mywatchedlist/api/migrations"
)

func New(t *testing.T) (*pgxpool.Pool, string) { return At(t, 0) }

func At(t *testing.T, version int64) (*pgxpool.Pool, string) {
	t.Helper()
	ctx := context.Background()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL obrigatória; nunca usar o banco pessoal")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("TEST_DATABASE_URL inválida")
	}
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("app_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg.ConnConfig)
	p, err := migrations.New(db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if version == 0 {
		_, err = p.Up(ctx)
	} else {
		_, err = p.UpTo(ctx, version)
	}
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return pool, u.String()
}

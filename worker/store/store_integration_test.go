//go:build integration

package store

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"mywatchedlist/api/migrations"
	"mywatchedlist/internal/session"
	"mywatchedlist/worker/collector"
)

func TestPersistence(t *testing.T) {
	ctx := context.Background()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL obrigatória")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("store_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg.ConnConfig)
	defer db.Close()
	p, err := migrations.New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Up(ctx); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	s := Store{Pool: pool}
	c, _ := session.New(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	uid, err := s.AddUser(ctx, "R", "r@example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.Connect(ctx, uid, "principal", "perfil-1", "R", []byte(`{"cookies":[]}`), c)
	if err != nil {
		t.Fatal(err)
	}
	accounts, err := s.Accounts(ctx, false)
	if err != nil || len(accounts) != 1 || accounts[0].ID != id {
		t.Fatal("conexão não persistida", err)
	}
	a := accounts[0]
	plain, err := c.Open(id, a.EncryptedSession)
	if err != nil || string(plain) != `{"cookies":[]}` {
		t.Fatal("sessão inválida")
	}
	e := collector.Event{SourceKey: "netflix:1:1725233400000", SourceTitleID: "1", SourceTitle: "Teste", MatchTitle: "Teste", WatchedOn: "2024-09-01"}
	l, err := s.Lock(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		other, e := s.Lock(ctx, id)
		if !errors.Is(e, ErrBusy) {
			if other != nil {
				other.Close()
			}
			t.Error("execução concorrente não bloqueada")
		}
	}()
	wg.Wait()
	for i := 0; i < 2; i++ {
		if _, err = l.Save(ctx, a, []collector.Event{e}, nil); err != nil {
			t.Fatal(err)
		}
	}
	l.Close()
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM watch_events`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("replay: %d %v", count, err)
	}
	l, err = s.Lock(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	revisit := e
	revisit.SourceKey = "netflix:1:1725319800000"
	if _, err = l.Save(ctx, a, []collector.Event{revisit}, nil); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM watch_events`).Scan(&count); err != nil || count != 2 {
		t.Fatal("revisita perdida")
	}
	bad := e
	bad.SourceKey = "bad"
	bad.SourceTitle = ""
	third := e
	third.SourceKey = "terceiro"
	var before, after time.Time
	if err = pool.QueryRow(ctx, `SELECT last_sync_at FROM streaming_accounts WHERE id=$1`, id).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if _, err = l.Save(ctx, a, []collector.Event{third, bad}, nil); err == nil {
		t.Fatal("lote inválido aceito")
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM watch_events`).Scan(&count); err != nil || count != 2 {
		t.Fatal("lote parcial persistido")
	}
	if err = l.Failure(ctx, id, "error", "parser_changed"); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT last_sync_at FROM streaming_accounts WHERE id=$1`, id).Scan(&after); err != nil || !after.Equal(before) {
		t.Fatal("falha alterou último sucesso")
	}
	statuses, err := s.Accounts(ctx, true)
	if err != nil || statuses[0].Status != "error" || *statuses[0].ErrorCode != "parser_changed" {
		t.Fatal("status de erro perdido")
	}
}

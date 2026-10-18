//go:build integration

package syncer

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"mywatchedlist/api/migrations"
	"mywatchedlist/internal/session"
	"mywatchedlist/worker/netflix"
	"mywatchedlist/worker/store"
)

func serveStore(t *testing.T) (store.Store, *session.Cipher, string) {
	t.Helper()
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
	schema := fmt.Sprintf("serve_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg.ConnConfig)
	m, err := migrations.New(db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	_, err = m.Up(ctx)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	s := store.Store{Pool: pool}
	cipher, _ := session.New(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	uid, err := s.AddUser(ctx, "Pessoa", "pessoa@example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	return s, cipher, uid
}

func TestServeStartupAndEligibility(t *testing.T) {
	s, cipher, uid := serveStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, status := range []string{"active", "error", "needs_login", "disabled"} {
		id, err := s.Connect(ctx, uid, "principal", status, status, []byte(`{"cookies":[]}`), cipher)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.Pool.Exec(ctx, `UPDATE streaming_accounts SET session_status=$2 WHERE id=$1`, id, status); err != nil {
			t.Fatal(err)
		}
	}
	seen := make(chan string, 4)
	r := Runner{Store: s, Cipher: cipher, Fetcher: fetchFunc(func(_ context.Context, raw []byte, profile string) (netflix.Batch, error) {
		seen <- profile
		if string(raw) != `{"cookies":[]}` {
			t.Error("sessão inválida")
		}
		events, err := netflix.Normalize([]byte(`{"viewedItems":[{"movieID":1,"title":"Teste","date":1725233400000}]}`), time.UTC)
		return netflix.Batch{Events: events, State: raw}, err
	})}
	done := make(chan error, 1)
	go func() { done <- r.Serve(ctx, "0 4 * * *", time.UTC) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("worker não encerrou")
		}
	})
	deadline := time.Now().Add(3 * time.Second)
	var count int
	for {
		if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM watch_events`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("startup não persistiu conexões elegíveis: %d", count)
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
		done <- nil
	case <-time.After(3 * time.Second):
		t.Fatal("cancelamento ignorado")
	}
	close(seen)
	profiles := map[string]int{}
	for p := range seen {
		profiles[p]++
	}
	if len(profiles) != 2 || profiles["active"] != 1 || profiles["error"] != 1 {
		t.Fatalf("seleção incorreta: %v", profiles)
	}
	var successes, ineligible int
	if err := s.Pool.QueryRow(context.Background(), `SELECT count(*) FILTER (WHERE last_sync_at IS NOT NULL),count(*) FILTER (WHERE session_status IN ('needs_login','disabled') AND last_sync_at IS NOT NULL) FROM streaming_accounts`).Scan(&successes, &ineligible); err != nil || successes != 2 || ineligible != 0 {
		t.Fatalf("sucesso/estado incorreto: %d %d %v", successes, ineligible, err)
	}
}

func TestServeCancellationReleasesLease(t *testing.T) {
	s, cipher, uid := serveStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	id, err := s.Connect(ctx, uid, "principal", "r", "R", []byte(`{"cookies":[]}`), cipher)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	r := Runner{Store: s, Cipher: cipher, Fetcher: fetchFunc(func(c context.Context, _ []byte, _ string) (netflix.Batch, error) {
		close(started)
		<-c.Done()
		return netflix.Batch{}, c.Err()
	})}
	done := make(chan error, 1)
	go func() { done <- r.Serve(ctx, "0 4 * * *", time.UTC) }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("coleta não iniciou")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("coleta não cancelou")
	}
	lease, err := s.Lock(context.Background(), id)
	if err != nil {
		t.Fatal("lock ficou retido", err)
	}
	lease.Close()
	var count int
	var last *time.Time
	if err = s.Pool.QueryRow(context.Background(), `SELECT count(*) FROM watch_events`).Scan(&count); err != nil || count != 0 {
		t.Fatal("cancelamento persistiu lote parcial", count, err)
	}
	if err = s.Pool.QueryRow(context.Background(), `SELECT last_sync_at FROM streaming_accounts WHERE id=$1`, id).Scan(&last); err != nil || last != nil {
		t.Fatal("cancelamento marcou sucesso", err)
	}
}

type shortSchedule struct{}

func (shortSchedule) Next(now time.Time) time.Time { return now.Add(20 * time.Millisecond) }

// O relógio da agenda é controlado, mas cada ciclo persiste no PostgreSQL real.
func TestServeRepeatsCycle(t *testing.T) {
	s, cipher, uid := serveStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	id, err := s.Connect(ctx, uid, "principal", "r", "R", []byte(`{"cookies":[]}`), cipher)
	if err != nil {
		t.Fatal(err)
	}
	iteration := 0
	r := Runner{Store: s, Cipher: cipher, Fetcher: fetchFunc(func(_ context.Context, raw []byte, _ string) (netflix.Batch, error) {
		iteration++
		body := fmt.Sprintf(`{"viewedItems":[{"movieID":1,"title":"Teste","date":%d}]}`, int64(1725233400000)+int64(iteration))
		events, err := netflix.Normalize([]byte(body), time.UTC)
		return netflix.Batch{Events: events, State: raw}, err
	})}
	done := make(chan error, 1)
	go func() { done <- r.serve(ctx, shortSchedule{}) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("worker não encerrou")
		}
	})
	deadline := time.Now().Add(3 * time.Second)
	for {
		var count int
		if err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM watch_events WHERE streaming_account_id=$1`, id).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("agenda não repetiu o ciclo: %d", count)
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
		done <- nil
	case <-time.After(3 * time.Second):
		t.Fatal("worker não cancelou")
	}
	if iteration < 2 {
		t.Fatal("ciclo não foi repetido")
	}
}

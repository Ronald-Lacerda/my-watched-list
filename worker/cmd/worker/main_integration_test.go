//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"mywatchedlist/api/migrations"
	"mywatchedlist/internal/session"
	"mywatchedlist/worker/netflix"
	"mywatchedlist/worker/store"
)

type scriptedBrowser struct{ t *testing.T }

const syntheticState = `{"cookies":[],"origins":[],"synthetic":"private-test-session"}`

func (b scriptedBrowser) LoginWithState(_ context.Context, name string, state []byte, confirm func() error) (netflix.Profile, []byte, error) {
	if name != "R" && (name != "Gabriela" || string(state) != syntheticState) {
		b.t.Error("perfil/estado de reconexão incorreto")
	}
	if err := confirm(); err != nil {
		return netflix.Profile{}, nil, err
	}
	return netflix.Profile{ID: "guid-" + name, Name: name}, []byte(syntheticState), nil
}
func (b scriptedBrowser) Fetch(_ context.Context, state []byte, guid string) (netflix.Batch, error) {
	if string(state) != syntheticState || guid != "guid-R" {
		b.t.Error("sync não recebeu a sessão/perfil salvos")
	}
	events, err := netflix.Normalize([]byte(`{"viewedItems":[{"movieID":1,"title":"Filme sintético","date":1725233400000}]}`), time.UTC)
	return netflix.Batch{Events: events, State: []byte(syntheticState)}, err
}

func TestAdministrativeCLI(t *testing.T) {
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
	schema := fmt.Sprintf("cli_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg.ConnConfig)
	defer db.Close()
	m, err := migrations.New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Up(ctx); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	connectionURL, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := connectionURL.Query()
	query.Set("search_path", schema)
	connectionURL.RawQuery = query.Encode()
	t.Setenv("DATABASE_URL", connectionURL.String())
	t.Setenv("SESSION_ENCRYPTION_KEY", key)
	t.Setenv("TMDB_READ_ACCESS_TOKEN", "")
	t.Setenv("SYNC_TIMEZONE", "America/Sao_Paulo")
	var out bytes.Buffer
	command := func(args ...string) error {
		out.Reset()
		return runWithBrowser(ctx, args, strings.NewReader("\n"), &out, func(zone *time.Location, channel string) browserClient {
			if zone.String() != "America/Sao_Paulo" {
				t.Error("fuso incorreto")
			}
			return scriptedBrowser{t}
		})
	}
	if err = command("user-add", "--name", "Pessoa", "--email", "pessoa@example.invalid"); err != nil {
		t.Fatal(err)
	}
	uid := strings.TrimSpace(out.String())
	var name, password string
	if err = pool.QueryRow(ctx, `SELECT name,password_hash FROM users WHERE id=$1`, uid).Scan(&name, &password); err != nil || name != "Pessoa" || password != "!disabled" {
		t.Fatalf("identidade administrativa inválida: %q %q %v", name, password, err)
	}
	if err = command("login", "--user-id", uid, "--profile", "R"); err != nil {
		t.Fatal(err)
	}
	list, err := (store.Store{Pool: pool}).Accounts(ctx, true)
	if err != nil || len(list) != 1 || list[0].Name != "R" || list[0].ProfileID != "guid-R" || list[0].UserID != uid || list[0].Status != "active" {
		t.Fatalf("conexão incorreta: %+v %v", list, err)
	}
	a := list[0]
	cipher, _ := session.New(key)
	raw, err := cipher.Open(a.ID, a.EncryptedSession)
	if err != nil || string(raw) != syntheticState || bytes.Contains(a.EncryptedSession, []byte("private-test-session")) || !strings.Contains(out.String(), "Conexão salva: "+a.ID) {
		t.Fatal("sessão não foi cifrada/confirmada corretamente")
	}
	for i := 0; i < 2; i++ {
		if err = command("sync", "--connection", a.ID); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	var last *time.Time
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM watch_events WHERE streaming_account_id=$1`, a.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("sync manual duplicou/perdeu evento", count, err)
	}
	if err = pool.QueryRow(ctx, `SELECT last_sync_at FROM streaming_accounts WHERE id=$1`, a.ID).Scan(&last); err != nil || last == nil {
		t.Fatal("sync sem último sucesso", err)
	}
	if err = command("status"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), a.ID+"\tR\tactive\t") || strings.Contains(out.String(), "nunca") || strings.Contains(out.String(), "private-test-session") || strings.Contains(out.String(), "cookies") {
		t.Fatalf("status incorreto/inseguro: %q", out.String())
	}
	if err = command("login", "--user-id", uid, "--profile", "R"); err != nil {
		t.Fatal(err)
	}
	list, _ = (store.Store{Pool: pool}).Accounts(ctx, true)
	if len(list) != 1 || list[0].ID != a.ID {
		t.Fatal("reconexão recriou conexão")
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM watch_events WHERE streaming_account_id=$1`, a.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("reconexão perdeu histórico", err)
	}
	if err = command("login", "--user-id", uid, "--profile", "Gabriela", "--reuse-connection", a.ID); err != nil {
		t.Fatal(err)
	}
	list, _ = (store.Store{Pool: pool}).Accounts(ctx, true)
	if len(list) != 2 || list[1].ProfileID != "guid-Gabriela" || list[1].ID == a.ID {
		t.Fatal("perfil independente não foi salvo")
	}
}

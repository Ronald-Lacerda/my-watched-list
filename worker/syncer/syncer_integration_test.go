//go:build integration

package syncer

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"mywatchedlist/api/migrations"
	"mywatchedlist/internal/session"
	"mywatchedlist/worker/collector"
	"mywatchedlist/worker/netflix"
	"mywatchedlist/worker/store"
	"mywatchedlist/worker/tmdb"
	"os"
	"strings"
	"testing"
	"time"
)

type fetchFunc func(context.Context, []byte, string) (netflix.Batch, error)

func (f fetchFunc) Fetch(c context.Context, s []byte, p string) (netflix.Batch, error) {
	return f(c, s, p)
}

type matchFunc func(context.Context, collector.Event) (store.Title, error)

func (f matchFunc) Match(c context.Context, e collector.Event) (store.Title, error) { return f(c, e) }
func TestPipeline(t *testing.T) {
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
	schema := fmt.Sprintf("pipeline_%d", time.Now().UnixNano())
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
	s := store.Store{Pool: pool}
	cipher, _ := session.New(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	uid, err := s.AddUser(ctx, "Pessoa", "pessoa@example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"success", "tmdb_failed", "login", "parser", "driver_error"} {
		t.Run(mode, func(t *testing.T) {
			id, err := s.Connect(ctx, uid, "principal", mode, mode, []byte(`{"cookies":[]}`), cipher)
			if err != nil {
				t.Fatal(err)
			}
			var account store.Account
			list, _ := s.Accounts(ctx, true)
			for _, a := range list {
				if a.ID == id {
					account = a
				}
			}
			fetch := fetchFunc(func(c context.Context, raw []byte, p string) (netflix.Batch, error) {
				if string(raw) != `{"cookies":[]}` && string(raw) != `{"cookies":[],"origins":[]}` {
					t.Error("sessão não decifrada")
				}
				if p != mode {
					t.Error("perfil incorreto")
				}
				if mode == "login" {
					return netflix.Batch{}, collector.ErrLoginRequired
				}
				if mode == "parser" {
					return netflix.Batch{}, collector.ErrParserChanged
				}
				if mode == "driver_error" {
					return netflix.Batch{}, errors.New("NetflixId=SEGREDO-SINTÉTICO")
				}
				events, e := netflix.Normalize([]byte(`{"viewedItems":[{"movieID":101,"title":"Filme","date":1725233400000,"videoType":"movie"}]}`), time.UTC)
				return netflix.Batch{Events: events, State: []byte(`{"cookies":[],"origins":[]}`)}, e
			})
			good := matchFunc(func(context.Context, collector.Event) (store.Title, error) {
				return store.Title{TMDBID: 99, MediaType: "movie", Name: "Filme", Overview: "Sinopse"}, nil
			})
			r := Runner{Store: s, Cipher: cipher, Fetcher: fetch, Matcher: good}
			if mode == "tmdb_failed" {
				r.Matcher = matchFunc(func(context.Context, collector.Event) (store.Title, error) { return store.Title{}, tmdb.ErrUnavailable })
			}
			err = r.Sync(ctx, account)
			var count, linked int
			var status string
			var last *time.Time
			if e := pool.QueryRow(ctx, `SELECT count(*),count(title_id) FROM watch_events WHERE streaming_account_id=$1`, id).Scan(&count, &linked); e != nil {
				t.Fatal(e)
			}
			if e := pool.QueryRow(ctx, `SELECT session_status,last_sync_at FROM streaming_accounts WHERE id=$1`, id).Scan(&status, &last); e != nil {
				t.Fatal(e)
			}
			if mode == "success" || mode == "tmdb_failed" {
				if err != nil || count != 1 || status != "active" || last == nil {
					t.Fatalf("histórico perdido %d %s %v", count, status, err)
				}
				if mode == "tmdb_failed" {
					if linked != 0 {
						t.Fatal("match indevido")
					}
					r.Matcher = good
					if _, e := pool.Exec(ctx, `UPDATE watch_events SET enrich_after=now() WHERE streaming_account_id=$1`, id); e != nil {
						t.Fatal(e)
					}
				} else if linked != 1 {
					t.Fatal("metadados não vinculados")
				}
				if err = r.Sync(ctx, account); err != nil {
					t.Fatal(err)
				}
				if e := pool.QueryRow(ctx, `SELECT count(*),count(title_id) FROM watch_events WHERE streaming_account_id=$1`, id).Scan(&count, &linked); e != nil || count != 1 || linked != 1 {
					t.Fatal("replay/retry incorreto", e)
				}
			} else {
				want := "error"
				if mode == "login" {
					want = "needs_login"
				}
				if err == nil || strings.Contains(err.Error(), "SEGREDO") || count != 0 || status != want || last != nil {
					t.Fatalf("falha não isolada %d %s %v", count, status, err)
				}
			}
		})
	}
}

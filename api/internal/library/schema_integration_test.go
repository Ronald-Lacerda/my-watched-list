//go:build integration

package library

import (
	"context"
	"github.com/jackc/pgx/v5/stdlib"
	"mywatchedlist/api/migrations"
	"mywatchedlist/internal/testdb"
	"testing"
)

func TestMigrationPreservesData(t *testing.T) {
	ctx := context.Background()
	pool, _ := testdb.At(t, 2)
	var uid, tid, eid string
	if err := pool.QueryRow(ctx, `INSERT INTO users(name,email,password_hash) VALUES('Pessoa','pessoa@example.invalid','!disabled') RETURNING id::text`).Scan(&uid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO titles(tmdb_id,media_type,title) VALUES(1,'movie','Filme') RETURNING id::text`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ratings(user_id,title_id,score,comment) VALUES($1,$2,8.5,'Original')`, uid, tid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `WITH a AS (INSERT INTO streaming_accounts(user_id,service,account_ref,profile_id,profile_name) VALUES($1,'netflix','principal','r','R') RETURNING id) INSERT INTO watch_events(user_id,streaming_account_id,source_event_key,source_title,watched_on) SELECT $1,id,'key','Filme','2026-10-18' FROM a RETURNING id::text`, uid).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer db.Close()
	p, err := migrations.New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Up(ctx); err != nil {
		t.Fatal(err)
	}
	var score float64
	var comment, got string
	if err = pool.QueryRow(ctx, `SELECT score,comment FROM ratings WHERE user_id=$1 AND title_id=$2`, uid, tid).Scan(&score, &comment); err != nil || score != 8.5 || comment != "Original" {
		t.Fatal("nota anterior perdida", err)
	}
	if err = pool.QueryRow(ctx, `SELECT id::text FROM watch_events WHERE id=$1`, eid).Scan(&got); err != nil || got != eid {
		t.Fatal("evento anterior perdido", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO ratings(user_id,source_key,score) VALUES($1,'netflix:work:123',0)`, uid); err != nil {
		t.Fatal("nota sem TMDB rejeitada", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO ratings(user_id,score) VALUES($1,4)`, uid); err == nil {
		t.Fatal("nota sem identidade aceita")
	}
}

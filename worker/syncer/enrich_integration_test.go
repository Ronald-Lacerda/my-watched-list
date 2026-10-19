//go:build integration

package syncer

import (
	"context"
	"mywatchedlist/internal/testdb"
	"mywatchedlist/worker/collector"
	"mywatchedlist/worker/store"
	"mywatchedlist/worker/tmdb"
	"testing"
)

func TestMetadataOnly(t *testing.T) {
	ctx := context.Background()
	pool, _ := testdb.New(t)
	exec := func(sql string) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO users(id,name,email,password_hash) VALUES('00000000-0000-0000-0000-000000000001','Pessoa','p@example.invalid','!disabled')`)
	exec(`INSERT INTO streaming_accounts(id,user_id,service,account_ref,profile_id,profile_name,session_status) VALUES('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','netflix','a','p','Pessoa','needs_login')`)
	exec(`INSERT INTO watch_events(user_id,streaming_account_id,source_event_key,source_title_id,source_title,watched_on,enrichment_status,enrich_after) SELECT '00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002',n::text,n::text,'Teste','2026-10-01','not_found',now()+interval '1 day' FROM generate_series(1,501) n`)
	calls := 0
	r := Runner{Store: store.Store{Pool: pool}, Matcher: matchFunc(func(context.Context, collector.Event) (store.Title, error) {
		calls++
		return store.Title{}, tmdb.ErrNotFound
	})}
	// Cipher e Fetcher nil: até conexões expiradas podem receber metadados.
	if err := r.Enrich(ctx, "", false); err != nil || calls != 0 {
		t.Fatal("backoff ignorado", calls, err)
	}
	if err := r.Enrich(ctx, "", true); err != nil || calls != 500 {
		t.Fatal("lote incorreto", calls, err)
	}
	var count int
	var status string
	var last *string
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM watch_events`).Scan(&count); err != nil || count != 501 {
		t.Fatal("enrich alterou eventos", count, err)
	}
	if err := pool.QueryRow(ctx, `SELECT session_status,last_sync_at::text FROM streaming_accounts`).Scan(&status, &last); err != nil || status != "needs_login" || last != nil {
		t.Fatal("enrich alterou sessão/coleta", status, last, err)
	}
	if err := r.Enrich(ctx, "00000000-0000-0000-0000-000000000099", true); err == nil {
		t.Fatal("conexão inexistente aceita")
	}
	if err := (Runner{Store: r.Store}).Enrich(ctx, "", true); err == nil {
		t.Fatal("enrich sem matcher aceito")
	}
}

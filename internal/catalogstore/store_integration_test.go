//go:build integration

package catalogstore_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"mywatchedlist/internal/catalogstore"
	"mywatchedlist/internal/metadata"
	"mywatchedlist/internal/testdb"
	"mywatchedlist/worker/store"
)

func TestDurableIdentification(t *testing.T) {
	ctx := context.Background()
	pool, _ := testdb.New(t)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO users(id,name,email,password_hash) VALUES('00000000-0000-0000-0000-000000000001','Um','um@example.invalid','!disabled'),('00000000-0000-0000-0000-000000000002','Dois','dois@example.invalid','!disabled')`)
	exec(`INSERT INTO streaming_accounts(id,user_id,service,account_ref,profile_id,profile_name) VALUES('00000000-0000-0000-0000-000000000011','00000000-0000-0000-0000-000000000001','netflix','a','um','Um'),('00000000-0000-0000-0000-000000000012','00000000-0000-0000-0000-000000000002','netflix','a','dois','Dois'),('00000000-0000-0000-0000-000000000013','00000000-0000-0000-0000-000000000001','other','a','um','Um')`)
	exec(`INSERT INTO watch_events(id,user_id,streaming_account_id,source_event_key,source_title_id,source_work_id,source_title,match_title,media_type_hint,watched_on) SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid,CASE WHEN n=102 THEN '00000000-0000-0000-0000-000000000002'::uuid ELSE '00000000-0000-0000-0000-000000000001'::uuid END,CASE WHEN n=102 THEN '00000000-0000-0000-0000-000000000012'::uuid WHEN n=104 THEN '00000000-0000-0000-0000-000000000013'::uuid ELSE '00000000-0000-0000-0000-000000000011'::uuid END,n::text,n::text,CASE WHEN n=103 THEN '20' WHEN n=105 THEN '' ELSE '10' END,'Origem: episódio','Origem','tv','2026-10-01' FROM generate_series(101,105) n`)
	exec(`INSERT INTO ratings(user_id,source_key,score,comment) VALUES('00000000-0000-0000-0000-000000000001','netflix:work:10',0,'zero preservado'),('00000000-0000-0000-0000-000000000002','netflix:work:10',10,'dez preservado')`)
	var rawBefore, notesBefore string
	if err := pool.QueryRow(ctx, `SELECT jsonb_agg(jsonb_build_array(id,user_id,source_title,source_work_id,watched_on) ORDER BY id)::text FROM watch_events`).Scan(&rawBefore); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT jsonb_agg(to_jsonb(r) ORDER BY user_id)::text FROM ratings r`).Scan(&notesBefore); err != nil {
		t.Fatal(err)
	}
	event := "00000000-0000-0000-0000-000000000101"
	actor := "00000000-0000-0000-0000-000000000001"
	good := metadata.Title{TMDBID: 99, MediaType: "tv", Name: "Confirmada", Overview: "Sinopse oficial", Poster: "/confirmed.jpg", Language: "pt-BR", Genres: []string{"Drama"}}
	if err := catalogstore.ConfirmTMDB(ctx, pool, event, actor, good); err != nil {
		t.Fatal(err)
	}
	var linked int
	var got string
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM watch_events WHERE title_id IS NOT NULL`).Scan(&linked); err != nil || linked != 2 {
		t.Fatal("confirmação não isolou obra/serviço", linked, err)
	}
	for _, bad := range []metadata.Title{{TMDBID: 1, MediaType: "movie", Name: "Errada"}, {TMDBID: 0, MediaType: "tv", Name: "Errada"}} {
		if err := catalogstore.ConfirmTMDB(ctx, pool, event, actor, bad); !errors.Is(err, catalogstore.ErrType) {
			t.Fatal("tipo/ID incorreto aceito", err)
		}
	}
	if err := catalogstore.ConfirmTMDB(ctx, pool, "00000000-0000-0000-0000-000000000105", actor, good); !errors.Is(err, catalogstore.ErrIdentity) {
		t.Fatal("série sem chave aceita", err)
	}
	// Resultado de uma busca iniciada antes da confirmação chega depois dela.
	lease, err := (store.Store{Pool: pool}).Lock(ctx, "00000000-0000-0000-0000-000000000011")
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err = lease.Link(ctx, store.PendingEvent{ID: event}, metadata.Title{TMDBID: 777, MediaType: "tv", Name: "Automática errada"}); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT t.title FROM titles t JOIN watch_events e ON e.title_id=t.id WHERE e.id=$1`, event).Scan(&got); err != nil || got != "Confirmada" {
		t.Fatal("resultado atrasado substituiu escolha", got, err)
	}
	exec(`INSERT INTO watch_events(id,user_id,streaming_account_id,source_event_key,source_title_id,source_work_id,source_title,media_type_hint,watched_on) VALUES('00000000-0000-0000-0000-000000000106','00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000011','novo','106','10','Novo episódio','tv','2026-10-02')`)
	if ok, err := lease.ApplyOverride(ctx, "00000000-0000-0000-0000-000000000106"); err != nil || !ok {
		t.Fatal("episódio futuro perdeu override", ok, err)
	}
	// Concorrência real: worker espera o mesmo lock da confirmação, sem row lock anterior.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = catalogstore.LockIdentity(ctx, tx, event); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- lease.Link(ctx, store.PendingEvent{ID: event}, metadata.Title{TMDBID: 888, MediaType: "tv", Name: "Concorrente"})
	}()
	select {
	case err := <-done:
		t.Fatal("worker ignorou lock", err)
	case <-time.After(40 * time.Millisecond):
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("deadlock")
	}
	if err = pool.QueryRow(ctx, `SELECT t.tmdb_id::text FROM titles t JOIN watch_events e ON e.title_id=t.id WHERE e.id=$1`, event).Scan(&got); err != nil || got != "99" {
		t.Fatal("concorrência trocou escolha", got, err)
	}
	exec(`DELETE FROM watch_events WHERE id='00000000-0000-0000-0000-000000000106'`)
	var after string
	if err = pool.QueryRow(ctx, `SELECT jsonb_agg(jsonb_build_array(id,user_id,source_title,source_work_id,watched_on) ORDER BY id)::text FROM watch_events`).Scan(&after); err != nil || after != rawBefore {
		t.Fatal("origem alterada", err)
	}
	if err = pool.QueryRow(ctx, `SELECT jsonb_agg(to_jsonb(r) ORDER BY user_id)::text FROM ratings r`).Scan(&after); err != nil || after != notesBefore {
		t.Fatal("notas alteradas", err)
	}
	// Complemento oficial não inventa TMDB e não aceita CDN/ID de outra obra.
	nf := metadata.NetflixTitle{NetflixID: "20", Name: "Oficial", Overview: "Sinopse Netflix", PosterURL: "https://occ-0.nflxso.net/art.jpg", MediaType: "tv", Language: "pt-BR"}
	other := "00000000-0000-0000-0000-000000000103"
	if err = catalogstore.ConfirmNetflix(ctx, pool, other, actor, nf); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT provider||':'||overview||':'||poster_url FROM metadata_overrides WHERE source_key='netflix:work:20' AND title_id IS NULL`).Scan(&got); err != nil || got != "netflix:Sinopse Netflix:https://occ-0.nflxso.net/art.jpg" {
		t.Fatal("complemento perdido", got, err)
	}
	// A classificação confirmada continua conhecida mesmo sem pista na coleta.
	exec(`UPDATE watch_events SET media_type_hint='' WHERE id=$1`, other)
	if err = catalogstore.ConfirmTMDB(ctx, pool, other, actor, metadata.Title{TMDBID: 123, MediaType: "movie", Name: "Filme incompatível"}); !errors.Is(err, catalogstore.ErrType) {
		t.Fatal("série Netflix conhecida aceitou filme", err)
	}
	nf.PosterURL = "https://evil.invalid/art.jpg"
	if err = catalogstore.ConfirmNetflix(ctx, pool, other, actor, nf); !errors.Is(err, catalogstore.ErrType) {
		t.Fatal("CDN inválida aceita", err)
	}
	nf.PosterURL = ""
	nf.NetflixID = "999"
	if err = catalogstore.ConfirmNetflix(ctx, pool, other, actor, nf); !errors.Is(err, catalogstore.ErrIdentity) {
		t.Fatal("ID incorreto aceito", err)
	}
	pending, err := lease.PendingRetry(ctx, "00000000-0000-0000-0000-000000000011", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pending {
		if p.ID == other {
			t.Fatal("fallback será reprocessado indefinidamente")
		}
	}
}

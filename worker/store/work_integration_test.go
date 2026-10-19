//go:build integration

package store

import (
	"context"
	"encoding/base64"
	"mywatchedlist/internal/session"
	"mywatchedlist/internal/testdb"
	"mywatchedlist/worker/collector"
	"testing"
)

func TestWorkReplayPreservesRating(t *testing.T) {
	ctx := context.Background()
	pool, _ := testdb.New(t)
	s := Store{Pool: pool}
	cipher, _ := session.New(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	uid, err := s.AddUser(ctx, "R", "r@example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.Connect(ctx, uid, "principal", "r", "R", []byte(`{"cookies":[]}`), cipher)
	if err != nil {
		t.Fatal(err)
	}
	list, err := s.Accounts(ctx, true)
	if err != nil || len(list) != 1 {
		t.Fatal(err)
	}
	l, err := s.Lock(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	e := collector.Event{SourceKey: "event", SourceTitleID: "201", SourceTitle: "Série: episódio", MatchTitle: "Série", MediaType: "tv", WatchedOn: "2026-10-18"}
	if _, err = l.Save(ctx, list[0], []collector.Event{e}, nil); err != nil {
		t.Fatal(err)
	}
	var tid string
	if err = pool.QueryRow(ctx, `INSERT INTO titles(tmdb_id,media_type,title) VALUES(1,'tv','Série') RETURNING id::text`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE watch_events SET title_id=$1,enrichment_status='matched'`, tid); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO ratings(user_id,source_key,score,comment) VALUES($1,'netflix:work:200',9,'Preservar')`, uid); err != nil {
		t.Fatal(err)
	}
	e.SourceWorkID = "200"
	for i := 0; i < 2; i++ {
		if _, err = l.Save(ctx, list[0], []collector.Event{e}, nil); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	var work, linked string
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM watch_events`).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicação", count, err)
	}
	if err = pool.QueryRow(ctx, `SELECT source_work_id,title_id::text FROM watch_events`).Scan(&work, &linked); err != nil || work != "200" || linked != tid {
		t.Fatal("obra ou metadados perdidos", work, linked, err)
	}
	var score float64
	var comment string
	if err = pool.QueryRow(ctx, `SELECT score,comment FROM ratings WHERE user_id=$1`, uid).Scan(&score, &comment); err != nil || score != 9 || comment != "Preservar" {
		t.Fatal("nota perdida", err)
	}
}

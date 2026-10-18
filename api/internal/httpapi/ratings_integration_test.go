//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"mywatchedlist/api/internal/library"
	"strings"
	"sync"
	"testing"
)

func mutate(t *testing.T, f appFixture, method, id, body string, actor int) library.RatingSummary {
	t.Helper()
	w := f.call(method, "/api/events/"+id+"/rating", body, actor, f.origin)
	if w.Code != 200 {
		t.Fatal("avaliação falhou", w.Code, w.Body.String())
	}
	var value library.RatingSummary
	if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestIndividualRatingsAndPending(t *testing.T) {
	f, ids := historyFixture(t)
	ctx := context.Background()
	if h := readHistory(t, f, "?pending=true", 0); h.Total != 3 {
		t.Fatal("pendentes não agrupam obras", h.Total)
	}
	r := mutate(t, f, "PUT", ids[0], `{"score":0,"comment":"Minha nota zero"}`, 0)
	if r.Count != 1 || r.Average == nil || *r.Average != 0 || len(r.Ratings) != 1 || r.Ratings[0].UserID != f.users[0].ID || r.Ratings[0].Score != 0 || r.Ratings[0].Comment != "Minha nota zero" {
		t.Fatal("nota zero perdida", r)
	}
	if h := readHistory(t, f, "?pending=true", 0); h.Total != 2 {
		t.Fatal("zero continua pendente")
	}
	if h := readHistory(t, f, "?pending=true", 1); h.Total != 3 {
		t.Fatal("pendentes da outra pessoa foram alterados")
	}
	r = mutate(t, f, "PUT", ids[1], `{"score":10,"comment":"Nota da Gabriela"}`, 1)
	if r.Count != 2 || r.Average == nil || *r.Average != 5 {
		t.Fatal("média 0/10 incorreta", r)
	}
	r = mutate(t, f, "PUT", ids[1], `{"score":8.5,"comment":"Editei a minha"}`, 0)
	if r.Count != 2 || r.Average == nil || *r.Average != 9.25 {
		t.Fatal("edição não atualizou média", r)
	}
	for _, rating := range r.Ratings {
		if rating.UserID == f.users[1].ID && (rating.Score != 10 || rating.Comment != "Nota da Gabriela") {
			t.Fatal("nota da outra pessoa alterada")
		}
	}
	mutate(t, f, "PUT", ids[2], `{"score":7}`, 0)
	mutate(t, f, "PUT", ids[3], `{"score":8}`, 0)
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM ratings WHERE user_id=$1 AND source_key='netflix:work:200'`, f.users[0].ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("nota por episódio duplicou", count, err)
	}
	if h := readHistory(t, f, "?pending=true", 0); h.Total != 1 {
		t.Fatal("série avaliada continua pendente")
	}
	h := readHistory(t, f, "?from=2026-10-18&to=2026-10-18", 0)
	for _, item := range h.Items {
		if item.Count != 2 || item.Average == nil || *item.Average != 9.25 {
			t.Fatal("nota não foi preservada na leitura")
		}
	}
	r = mutate(t, f, "DELETE", ids[0], "", 0)
	if r.Count != 1 || r.Average == nil || *r.Average != 10 || r.Ratings[0].UserID != f.users[1].ID {
		t.Fatal("exclusão afetou a outra pessoa", r)
	}
	r = mutate(t, f, "DELETE", ids[0], "", 0)
	if r.Count != 1 {
		t.Fatal("exclusão não idempotente")
	}
	if h := readHistory(t, f, "?pending=true", 0); h.Total != 2 {
		t.Fatal("exclusão não devolveu pendente")
	}
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM watch_events`).Scan(&count); err != nil || count != 6 {
		t.Fatal("avaliação alterou histórico", err)
	}
}

func TestRatingValidationAndAuthorization(t *testing.T) {
	f, ids := historyFixture(t)
	ctx := context.Background()
	mutate(t, f, "PUT", ids[0], `{"score":6,"comment":"Preservar"}`, 0)
	for _, body := range []string{`{"score":-0.5}`, `{"score":10.5}`, `{"score":7.2}`, `{"score":null}`, `{}`, `{"score":NaN}`, `{"score":5,"comment":"` + strings.Repeat("x", 2001) + `"}`, `{"score":5,"user_id":"` + f.users[1].ID + `"}`} {
		if w := f.call("PUT", "/api/events/"+ids[0]+"/rating", body, 0, f.origin); w.Code != 400 {
			t.Fatal("nota inválida aceita", w.Code)
		}
	}
	for _, method := range []string{"PUT", "DELETE"} {
		path := "/api/events/" + ids[0] + "/rating"
		body := `{"score":5}`
		if method == "DELETE" {
			body = ""
		}
		if w := f.call(method, path, body, -1, f.origin); w.Code != 401 {
			t.Fatal("mutação sem sessão")
		}
		for _, origin := range []string{"", "http://evil.test"} {
			if w := f.call(method, path, body, 0, origin); w.Code != 403 {
				t.Fatal("mutação cross-origin")
			}
		}
	}
	for _, test := range []struct {
		id   string
		code int
	}{{"bad", 400}, {"00000000-0000-0000-0000-000000000099", 404}, {ids[5], 409}} {
		if w := f.call("PUT", "/api/events/"+test.id+"/rating", `{"score":5}`, 0, f.origin); w.Code != test.code {
			t.Fatal("identidade inválida aceita", test.code, w.Code)
		}
	}
	if w := f.call("DELETE", "/api/events/"+ids[0]+"/rating", `{"user_id":"`+f.users[1].ID+`"}`, 0, f.origin); w.Code != 400 {
		t.Fatal("usuário na exclusão aceito")
	}
	if w := f.call("DELETE", "/api/events/"+ids[0]+"/rating?user_id="+f.users[1].ID, "", 0, f.origin); w.Code != 400 {
		t.Fatal("query de ownership aceita")
	}
	var score float64
	var comment string
	if err := f.pool.QueryRow(ctx, `SELECT score,comment FROM ratings WHERE user_id=$1`, f.users[0].ID).Scan(&score, &comment); err != nil || score != 6 || comment != "Preservar" {
		t.Fatal("falha alterou nota anterior", err)
	}
}

func TestRatingConcurrentUpsertAndLegacy(t *testing.T) {
	f, ids := historyFixture(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := f.call("PUT", "/api/events/"+ids[i]+"/rating", fmt.Sprintf(`{"score":%d}`, i+1), 0, f.origin)
			codes <- w.Code
		}(i)
	}
	wg.Wait()
	close(codes)
	for code := range codes {
		if code != 200 {
			t.Fatal("upsert concorrente falhou", code)
		}
	}
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM ratings WHERE user_id=$1`, f.users[0].ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("upsert duplicou nota", err)
	}
	var tid string
	if err := f.pool.QueryRow(ctx, `INSERT INTO titles(tmdb_id,media_type,title) VALUES(1,'movie','Filme enriquecido') RETURNING id::text`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE watch_events SET title_id=$1 WHERE source_title_id='101'`, tid); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `DELETE FROM ratings`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO ratings(user_id,title_id,score) VALUES($1,$2,6)`, f.users[0].ID, tid); err != nil {
		t.Fatal(err)
	}
	r := mutate(t, f, "PUT", ids[1], `{"score":10}`, 1)
	if r.Count != 2 || r.Average == nil || *r.Average != 8 {
		t.Fatal("nota anterior TMDB não apareceu", r)
	}
	r = mutate(t, f, "PUT", ids[0], `{"score":7}`, 0)
	if r.Count != 2 || r.Average == nil || *r.Average != 8.5 {
		t.Fatal("conversão de nota anterior incorreta", r)
	}
	r = mutate(t, f, "DELETE", ids[0], "", 0)
	if r.Count != 1 || *r.Average != 10 {
		t.Fatal("nota antiga reapareceu após excluir", r)
	}
}

func TestRatingRollbackOnFailure(t *testing.T) {
	f, ids := historyFixture(t)
	ctx := context.Background()
	var tid string
	if err := f.pool.QueryRow(ctx, `INSERT INTO titles(tmdb_id,media_type,title) VALUES(1,'movie','Filme') RETURNING id::text`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE watch_events SET title_id=$1 WHERE source_title_id='101'`, tid); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO ratings(user_id,title_id,score,comment) VALUES($1,$2,6,'Original')`, f.users[0].ID, tid); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `CREATE FUNCTION reject_legacy_delete() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'SYNTHETIC_PRIVATE_FAILURE'; END $$`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `CREATE TRIGGER fail_delete BEFORE DELETE ON ratings FOR EACH ROW EXECUTE FUNCTION reject_legacy_delete()`); err != nil {
		t.Fatal(err)
	}
	w := f.call("PUT", "/api/events/"+ids[0]+"/rating", `{"score":7}`, 0, f.origin)
	if w.Code != 500 || strings.Contains(w.Body.String(), "SYNTHETIC_PRIVATE_FAILURE") {
		t.Fatal("falha de banco exposta ou ignorada", w.Code, w.Body.String())
	}
	var count int
	var score float64
	var key *string
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM ratings`).Scan(&count); err != nil || count != 1 {
		t.Fatal("upsert parcial persistiu", count, err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT score,source_key FROM ratings`).Scan(&score, &key); err != nil || score != 6 || key != nil {
		t.Fatal("rollback não preservou nota original", err)
	}
}

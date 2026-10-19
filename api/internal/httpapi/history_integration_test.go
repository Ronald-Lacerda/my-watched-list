//go:build integration

package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mywatchedlist/api/internal/library"
	"net/url"
	"strings"
	"testing"
)

func historyFixture(t *testing.T) (appFixture, []string) {
	t.Helper()
	f := fixtureApp(t)
	ctx := context.Background()
	accounts := make([]string, 3)
	for i, service := range []string{"netflix", "netflix", "prime_video"} {
		person := i % 2
		status := "active"
		if i == 1 {
			status = "needs_login"
		}
		if err := f.pool.QueryRow(ctx, `INSERT INTO streaming_accounts(user_id,service,account_ref,profile_id,profile_name,session_status,session_ciphertext,session_key_version,last_sync_at) VALUES($1,$2,'principal',$3,$3,$4,$5,1,'2026-10-18T12:00:00Z') RETURNING id::text`, f.users[person].ID, service, f.users[person].Name, status, []byte("DO-NOT-EXPOSE-COOKIES")).Scan(&accounts[i]); err != nil {
			t.Fatal(err)
		}
	}
	var ids []string
	for i, e := range []struct {
		person, account                int
		source, work, title, kind, day string
	}{{0, 0, "101", "101", "Filme A", "movie", "2026-10-18"}, {1, 1, "101", "101", "Filme A", "movie", "2026-10-18"}, {0, 0, "201", "200", "Série A: episódio 1", "tv", "2026-10-17"}, {1, 1, "202", "200", "Série A: episódio 2", "tv", "2026-10-16"}, {0, 2, "303", "303", "Filme B", "movie", "2026-09-30"}, {1, 1, "999", "", "Série antiga: episódio", "tv", "2026-10-15"}} {
		match := e.title
		if e.work == "200" {
			match = "Série A"
		}
		var id string
		if err := f.pool.QueryRow(ctx, `INSERT INTO watch_events(user_id,streaming_account_id,source_event_key,source_title_id,source_work_id,source_title,match_title,media_type_hint,watched_on) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::date) RETURNING id::text`, f.users[e.person].ID, accounts[e.account], fmt.Sprintf("event-%d", i), e.source, e.work, e.title, match, e.kind, e.day).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return f, ids
}
func readHistory(t *testing.T, f appFixture, query string, actor int) library.History {
	t.Helper()
	w := f.call("GET", "/api/history"+query, "", actor, "")
	if w.Code != 200 {
		t.Fatal("histórico falhou", w.Code, w.Body.String())
	}
	var h library.History
	if err := json.Unmarshal(w.Body.Bytes(), &h); err != nil {
		t.Fatal(err)
	}
	return h
}

func TestHistoryFiltersAndPagination(t *testing.T) {
	f, _ := historyFixture(t)
	h := readHistory(t, f, "", 0)
	if h.Total != 6 || len(h.Items) != 6 || h.NextCursor != nil {
		t.Fatal("histórico incompleto", h.Total, len(h.Items))
	}
	for _, q := range []struct {
		value string
		count int
	}{{"?person=" + f.users[0].ID + "&service=netflix&from=2026-10-17&to=2026-10-18", 2}, {"?from=2026-10-18&to=2026-10-18", 2}, {"?service=prime_video", 1}, {"?person=" + f.users[1].ID, 3}, {"?person=00000000-0000-0000-0000-000000000099", 0}} {
		h = readHistory(t, f, q.value, 0)
		if h.Total != q.count || len(h.Items) != q.count {
			t.Fatal("filtro incorreto", q.value, h.Total)
		}
	}
	seen := map[string]bool{}
	cursor := ""
	lastDay, lastID := "9999-12-31", "ffffffff-ffff-ffff-ffff-ffffffffffff"
	for i := 0; i < 6; i++ {
		h = readHistory(t, f, "?limit=1&cursor="+url.QueryEscape(cursor), 0)
		if h.Total != 6 || len(h.Items) != 1 {
			t.Fatal("página incorreta")
		}
		item := h.Items[0]
		if seen[item.ID] || item.WatchedOn > lastDay || (item.WatchedOn == lastDay && item.ID >= lastID) {
			t.Fatal("ordem instável ou item repetido")
		}
		seen[item.ID] = true
		lastDay, lastID = item.WatchedOn, item.ID
		if i == 5 {
			if h.NextCursor != nil {
				t.Fatal("cursor extra no fim")
			}
		} else {
			if h.NextCursor == nil {
				t.Fatal("cursor ausente")
			}
			cursor = *h.NextCursor
		}
	}
	if len(seen) != 6 {
		t.Fatal("paginação perdeu item")
	}
	for _, q := range []string{"?person=bad", "?service=bad%20service", "?from=2026-02-30", "?from=2026-10-18&to=2026-10-17", "?cursor=bad", "?limit=0", "?limit=101", "?pending=yes", "?limit=1&limit=2", "?unexpected=1"} {
		if w := f.call("GET", "/api/history"+q, "", 0, ""); w.Code != 400 {
			t.Fatal("filtro inválido aceito", q, w.Code)
		}
	}
}

func TestReadEndpointsPrivacy(t *testing.T) {
	f, _ := historyFixture(t)
	for _, path := range []string{"/api/history", "/api/filters", "/api/connections"} {
		if w := f.call("GET", path, "", -1, ""); w.Code != 401 {
			t.Fatal("rota sem auth", path, w.Code)
		}
		w := f.call("GET", path, "", 0, "")
		if w.Code != 200 {
			t.Fatal(path, w.Body.String())
		}
		for _, secret := range []string{"DO-NOT-EXPOSE-COOKIES", base64.StdEncoding.EncodeToString([]byte("DO-NOT-EXPOSE-COOKIES")), "password_hash", "token_hash", "session_ciphertext", "DATABASE_URL", "example.invalid"} {
			if strings.Contains(w.Body.String(), secret) {
				t.Fatal("campo privado exposto", path, secret)
			}
		}
	}
	w := f.call("GET", "/api/filters", "", 0, "")
	var filters library.Filters
	if err := json.Unmarshal(w.Body.Bytes(), &filters); err != nil || len(filters.People) != 2 || len(filters.Services) != 2 {
		t.Fatal("filtros ausentes", err)
	}
	w = f.call("GET", "/api/connections", "", 0, "")
	var result struct{ Connections []library.Connection }
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || len(result.Connections) != 3 {
		t.Fatal("conexões ausentes", err)
	}
	found := false
	for _, c := range result.Connections {
		if c.Person == "Gabriela" && c.Status == "needs_login" && c.Profile == "Gabriela" && c.Service == "netflix" && c.LastSync != nil {
			found = true
		}
	}
	if !found {
		t.Fatal("status de reconexão perdido")
	}
}

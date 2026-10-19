//go:build integration

package httpapi

import (
	"context"
	"net/url"
	"testing"

	"mywatchedlist/api/internal/library"
)

func seriesItem(t *testing.T, h library.History, key string) library.Item {
	t.Helper()
	var found []library.Item
	for _, item := range h.Items {
		if item.RatingKey != nil && *item.RatingKey == key {
			found = append(found, item)
		}
	}
	if len(found) != 1 {
		t.Fatal("esperada uma entrada por série", key, len(found))
	}
	return found[0]
}

func TestSeriesGroupingAndFilters(t *testing.T) {
	f, ids := historyFixture(t)
	h := readHistory(t, f, "?group_series=true", 0)
	if h.Total != 5 || len(h.Items) != 5 || h.NextCursor != nil {
		t.Fatal("agrupamento deve preservar três filmes e duas séries", h.Total, len(h.Items))
	}
	item := seriesItem(t, h, "netflix:work:200")
	if item.ID != ids[2] || item.WatchedOn != "2026-10-17" || item.WatchCount != 2 || item.Title != "Série A" || len(item.People) != 2 || item.PosterPath != nil {
		t.Fatal("resumo da série sem TMDB incorreto", item)
	}
	people := map[string]string{}
	for _, person := range item.People {
		people[person.ID] = person.Name
	}
	for _, person := range f.users {
		if people[person.ID] != person.Name {
			t.Fatal("participante incorreto", people)
		}
	}
	for _, query := range []string{"", "?group_series=false"} {
		if raw := readHistory(t, f, query, 0); raw.Total != 6 || len(raw.Items) != 6 {
			t.Fatal("contrato bruto alterado", query, raw.Total)
		}
	}
	for _, test := range []struct {
		query, id, day string
		count, total   int
		person         int
	}{
		{"&person=" + f.users[1].ID, ids[3], "2026-10-16", 1, 3, 1},
		{"&from=2026-10-16&to=2026-10-16", ids[3], "2026-10-16", 1, 1, 1},
		{"&person=" + f.users[0].ID + "&service=netflix&from=2026-10-17&to=2026-10-18", ids[2], "2026-10-17", 1, 2, 0},
	} {
		h = readHistory(t, f, "?group_series=true"+test.query, 0)
		item = seriesItem(t, h, "netflix:work:200")
		if h.Total != test.total || item.ID != test.id || item.WatchedOn != test.day || item.WatchCount != test.count || len(item.People) != 1 || item.People[0].ID != f.users[test.person].ID || item.People[0].Name != f.users[test.person].Name {
			t.Fatal("filtros devem preceder o agrupamento", test.query, h.Total, item)
		}
	}
	for _, query := range []string{"?group_series=maybe", "?group_series=true&group_series=false"} {
		if w := f.call("GET", "/api/history"+query, "", 0, ""); w.Code != 400 {
			t.Fatal("opção inválida aceita", w.Code)
		}
	}
}

func TestSeriesGroupingIdentity(t *testing.T) {
	f, ids := historyFixture(t)
	ctx := context.Background()
	for _, work := range []string{"2010", "200"} {
		accountEvent := ids[2]
		if work == "200" {
			accountEvent = ids[4]
		}
		if _, err := f.pool.Exec(ctx, `INSERT INTO watch_events(user_id,streaming_account_id,source_event_key,source_title_id,source_work_id,source_title,match_title,media_type_hint,watched_on) SELECT user_id,streaming_account_id,'identity-'||$2,'episode-'||$2,$2,'Série A: outro episódio','Série A','tv','2026-10-17' FROM watch_events WHERE id=$1`, accountEvent, work); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO watch_events(user_id,streaming_account_id,source_event_key,source_title_id,source_work_id,source_title,match_title,media_type_hint,watched_on) SELECT user_id,streaming_account_id,'legacy-extra','1000','','Série antiga: episódio','Série antiga: episódio','tv',watched_on FROM watch_events WHERE id=$1`, ids[5]); err != nil {
		t.Fatal(err)
	}
	h := readHistory(t, f, "?group_series=true", 0)
	if h.Total != 8 {
		t.Fatal("nome/serviço/série sem chave não podem reunir obras distintas", h.Total)
	}
	for _, key := range []string{"netflix:work:200", "netflix:work:2010", "prime_video:work:200"} {
		seriesItem(t, h, key)
	}
	legacy := 0
	for _, item := range h.Items {
		if item.RatingKey == nil {
			legacy++
			if item.WatchCount != 1 || len(item.People) != 1 {
				t.Fatal("série sem identidade foi agrupada", item)
			}
		}
	}
	if legacy != 2 {
		t.Fatal("registros legados não preservados", legacy)
	}
}

func TestSeriesGroupingPagination(t *testing.T) {
	f, ids := historyFixture(t)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `INSERT INTO watch_events(user_id,streaming_account_id,source_event_key,source_title_id,source_work_id,source_title,match_title,media_type_hint,watched_on) SELECT user_id,streaming_account_id,'series-page-'||n,(500+n)::text,'200','Série A: episódio '||n,'Série A','tv','2026-10-14' FROM watch_events CROSS JOIN generate_series(1,100) n WHERE id=$1`, ids[2]); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	cursor := ""
	lastDay, lastID := "9999-12-31", "ffffffff-ffff-ffff-ffff-ffffffffffff"
	for page := 0; page < 5; page++ {
		h := readHistory(t, f, "?group_series=true&limit=1&cursor="+url.QueryEscape(cursor), 0)
		if h.Total != 5 || len(h.Items) != 1 {
			t.Fatal("total/paginação deve operar sobre grupos", h.Total, len(h.Items))
		}
		item := h.Items[0]
		if seen[item.ID] || item.WatchedOn > lastDay || item.WatchedOn == lastDay && item.ID >= lastID {
			t.Fatal("ordem ou identidade repetida", item)
		}
		seen[item.ID] = true
		lastDay, lastID = item.WatchedOn, item.ID
		if item.RatingKey != nil && *item.RatingKey == "netflix:work:200" && (item.ID != ids[2] || item.WatchCount != 102) {
			t.Fatal("cursor cortou membros da série", item)
		}
		if page == 4 {
			if h.NextCursor != nil {
				t.Fatal("cursor extra")
			}
		} else {
			if h.NextCursor == nil {
				t.Fatal("cursor ausente")
			}
			cursor = *h.NextCursor
		}
	}
	if !seen[ids[2]] || seen[ids[3]] {
		t.Fatal("representante duplicado ou ausente", seen)
	}
}

func TestGroupedSeriesRatings(t *testing.T) {
	f, ids := historyFixture(t)
	mutate(t, f, "PUT", ids[2], `{"score":0}`, 0)
	mutate(t, f, "PUT", ids[3], `{"score":10}`, 1)
	item := seriesItem(t, readHistory(t, f, "?group_series=true", 0), "netflix:work:200")
	if item.Count != 2 || item.Average == nil || *item.Average != 5 {
		t.Fatal("resumo perdeu notas da série", item)
	}
	mutate(t, f, "PUT", ids[3], `{"score":8}`, 0)
	r := mutate(t, f, "DELETE", ids[2], "", 0)
	if r.Count != 1 || r.Average == nil || *r.Average != 10 || len(r.Ratings) != 1 || r.Ratings[0].UserID != f.users[1].ID {
		t.Fatal("remover nota da série afetou parceira", r)
	}
	item = seriesItem(t, readHistory(t, f, "?group_series=true&pending=true", 0), "netflix:work:200")
	if item.WatchCount != 2 || item.ID != ids[2] {
		t.Fatal("pendente perdeu agregado da série", item)
	}
	var events int
	if err := f.pool.QueryRow(context.Background(), `SELECT count(*) FROM watch_events`).Scan(&events); err != nil || events != 6 {
		t.Fatal("notas/agrupamento alteraram eventos", events, err)
	}
}

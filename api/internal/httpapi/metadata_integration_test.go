//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"mywatchedlist/internal/metadata"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type metadataStub struct {
	fail    bool
	queries []string
	ids     []string
}

func TestNetflixKnownSeriesHTTP(t *testing.T) {
	f, ids := historyFixture(t)
	stub := &metadataStub{}
	withMetadata(t, &f, stub)
	if _, err := f.pool.Exec(context.Background(), `UPDATE watch_events SET media_type_hint='' WHERE id=$1`, ids[0]); err != nil {
		t.Fatal(err)
	}
	path := "/api/events/" + ids[0] + "/metadata"
	if w := f.call("PUT", path, `{"provider":"netflix"}`, 0, f.origin); w.Code != 200 {
		t.Fatal("identificação Netflix falhou", w.Code, w.Body.String())
	}
	if w := f.call("PUT", path, `{"provider":"tmdb","tmdb_id":123,"media_type":"movie"}`, 0, f.origin); w.Code != 400 || !strings.Contains(w.Body.String(), "incompatible_type") {
		t.Fatal("série confirmada aceitou filme", w.Code, w.Body.String())
	}
	var provider, kind string
	var title *string
	if err := f.pool.QueryRow(context.Background(), `SELECT provider,media_type,title_id::text FROM metadata_overrides WHERE source_key='netflix:work:101'`).Scan(&provider, &kind, &title); err != nil || provider != "netflix" || kind != "tv" || title != nil {
		t.Fatal("rejeição alterou identificação", provider, kind, title, err)
	}
}

func (s *metadataStub) Search(_ context.Context, q, kind string) ([]metadata.Candidate, error) {
	s.queries = append(s.queries, q+":"+kind)
	if s.fail {
		return nil, errors.New("token=SEGREDO-SINTETICO")
	}
	year := 2019
	poster := "/official.jpg"
	return []metadata.Candidate{{ID: 99, MediaType: "tv", Title: "Série identificada", Year: &year, PosterPath: &poster, Overview: "Sinopse oficial"}}, nil
}
func (s *metadataStub) Details(_ context.Context, kind string, id int64) (metadata.Title, error) {
	if s.fail {
		return metadata.Title{}, errors.New("token=SEGREDO-SINTETICO")
	}
	year := 2019
	return metadata.Title{TMDBID: id, MediaType: kind, Name: "Série identificada", Overview: "Sinopse oficial", Poster: "/official.jpg", Year: &year, Language: "pt-BR", Genres: []string{"Drama"}}, nil
}
func (s *metadataStub) Lookup(_ context.Context, id string) (metadata.NetflixTitle, error) {
	s.ids = append(s.ids, id)
	if s.fail {
		return metadata.NetflixTitle{}, metadata.ErrUnavailable
	}
	return metadata.NetflixTitle{NetflixID: id, Name: "Série oficial Netflix", Overview: "Sinopse da Netflix", PosterURL: "https://occ-0.nflxso.net/official.jpg", MediaType: "tv", Language: "pt-BR"}, nil
}
func withMetadata(t *testing.T, f *appFixture, s *metadataStub) {
	t.Helper()
	var err error
	f.handler, err = NewApp(f.pool, Options{Origins: []string{f.origin}, Metadata: s, NetflixMetadata: s})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMetadataHTTP(t *testing.T) {
	f, ids := historyFixture(t)
	stub := &metadataStub{}
	withMetadata(t, &f, stub)
	path := "/api/events/" + ids[2] + "/metadata"
	for _, route := range []struct{ method, path, body string }{{"GET", "/api/metadata/search?query=Serie&type=tv", ""}, {"PUT", path, `{"provider":"tmdb","tmdb_id":99,"media_type":"tv"}`}} {
		if w := f.call(route.method, route.path, route.body, -1, f.origin); w.Code != 401 {
			t.Fatal("metadados sem auth", w.Code)
		}
	}
	for _, query := range []string{"?query=a", "?query=" + strings.Repeat("a", 201), "?query=Serie&type=collection", "?query=Serie&extra=1", "?query=Serie&query=Outra"} {
		if w := f.call("GET", "/api/metadata/search"+query, "", 0, ""); w.Code != 400 {
			t.Fatal("busca inválida aceita", query, w.Code)
		}
	}
	w := f.call("GET", "/api/metadata/search?query=Serie&type=tv", "", 0, "")
	var found struct{ Items []metadata.Candidate }
	if err := json.Unmarshal(w.Body.Bytes(), &found); err != nil || w.Code != 200 || len(found.Items) != 1 || found.Items[0].ID != 99 || found.Items[0].MediaType != "tv" || found.Items[0].Title != "Série identificada" || found.Items[0].Year == nil || *found.Items[0].Year != 2019 || found.Items[0].PosterPath == nil || *found.Items[0].PosterPath != "/official.jpg" || found.Items[0].Overview != "Sinopse oficial" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("candidatos incompletos", w.Body.String(), err)
	}
	for _, origin := range []string{"", "https://evil.invalid"} {
		if w := f.call("PUT", path, `{"provider":"tmdb","tmdb_id":99,"media_type":"tv"}`, 0, origin); w.Code != 403 {
			t.Fatal("origem indevida aceita", w.Code)
		}
	}
	for _, body := range []string{"null", "{}", "{}{}", `{"provider":"tmdb","tmdb_id":0,"media_type":"tv"}`, `{"provider":"tmdb","tmdb_id":99,"media_type":"collection"}`, `{"provider":"tmdb","tmdb_id":99,"media_type":"movie"}`, `{"provider":"tmdb","tmdb_id":99,"media_type":"tv","poster_url":"https://evil.invalid"}`, `{"provider":"netflix","tmdb_id":99}`} {
		if w := f.call("PUT", path, body, 0, f.origin); w.Code != 400 {
			t.Fatal("JSON/tipo inválido aceito", body, w.Code, w.Body.String())
		}
	}
	r := httptest.NewRequest("PUT", path, strings.NewReader(`{}`))
	r.AddCookie(f.cookies[0])
	r.Header.Set("Origin", f.origin)
	r.Header.Set("Content-Type", "text/plain")
	out := httptest.NewRecorder()
	f.handler.ServeHTTP(out, r)
	if out.Code != http.StatusUnsupportedMediaType {
		t.Fatal("JSON não exigido", out.Code)
	}
	if w := f.call("PUT", "/api/events/bad/metadata", `{}`, 0, f.origin); w.Code != 400 {
		t.Fatal("UUID inválido aceito", w.Code)
	}
	if w := f.call("PUT", "/api/events/00000000-0000-0000-0000-000000000099/metadata", `{"provider":"tmdb","tmdb_id":99,"media_type":"tv"}`, 0, f.origin); w.Code != 404 {
		t.Fatal("evento inexistente aceito", w.Code)
	}
	if w := f.call("PUT", "/api/events/"+ids[5]+"/metadata", `{"provider":"tmdb","tmdb_id":99,"media_type":"tv"}`, 0, f.origin); w.Code != 409 {
		t.Fatal("série sem obra aceita", w.Code)
	}
	if w := f.call("PUT", path, `{"provider":"tmdb","tmdb_id":99,"media_type":"tv"}`, 0, f.origin); w.Code != 200 {
		t.Fatal("confirmação falhou", w.Code, w.Body.String())
	}
	h := readHistory(t, f, "", 1)
	matched := 0
	for _, item := range h.Items {
		if item.ID == ids[2] || item.ID == ids[3] {
			matched++
			if item.Title != "Série identificada" || item.Overview != "Sinopse oficial" || item.PosterPath == nil || *item.PosterPath != "/official.jpg" || item.MetadataProvider != "tmdb" || item.OverviewLanguage != "pt-BR" || item.PosterURL != "" || item.SourceTitle == item.Title || item.Year == nil || *item.Year != 2019 || len(item.Genres) != 1 {
				t.Fatal("histórico não refletiu metadados", item)
			}
		}
	}
	if matched != 2 {
		t.Fatal("outra pessoa não recebeu identificação", matched)
	}
	if w := f.call("PUT", path, `{"provider":"netflix"}`, 1, f.origin); w.Code != 200 {
		t.Fatal("fallback Netflix falhou", w.Code, w.Body.String())
	}
	if len(stub.ids) != 1 || stub.ids[0] != "200" {
		t.Fatal("fallback usou episódio/ID do cliente", stub.ids)
	}
	for _, item := range readHistory(t, f, "?group_series=true", 0).Items {
		if item.RatingKey != nil && *item.RatingKey == "netflix:work:200" {
			if item.Title != "Série oficial Netflix" || item.Overview != "Sinopse da Netflix" || item.MetadataProvider != "netflix" || item.PosterURL != "https://occ-0.nflxso.net/official.jpg" || item.WatchCount != 2 {
				t.Fatal("complemento perdido", item)
			}
		}
	}
	stub.fail = true
	for _, req := range []struct{ method, path, body string }{{"GET", "/api/metadata/search?query=Serie", ""}, {"PUT", path, `{"provider":"tmdb","tmdb_id":88,"media_type":"tv"}`}, {"PUT", path, `{"provider":"netflix"}`}} {
		w := f.call(req.method, req.path, req.body, 0, f.origin)
		if w.Code != 503 || strings.Contains(w.Body.String(), "SEGREDO") {
			t.Fatal("erro remoto inseguro", w.Code, w.Body.String())
		}
	}
	var count int
	if err := f.pool.QueryRow(context.Background(), `SELECT count(*) FROM metadata_overrides`).Scan(&count); err != nil || count != 1 {
		t.Fatal("falha criou associação parcial", count, err)
	}
	f.handler, _ = NewApp(f.pool, Options{Origins: []string{f.origin}})
	if w := f.call("GET", "/api/metadata/search?query=Serie", "", 0, ""); w.Code != 503 {
		t.Fatal("token ausente sem resposta controlada", w.Code)
	}
	if w := f.call("GET", "/api/history", "", 0, ""); w.Code != 200 {
		t.Fatal("token ausente bloqueou diário", w.Code)
	}
}

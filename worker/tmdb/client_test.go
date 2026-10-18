package tmdb

import (
	"context"
	"errors"
	"fmt"
	"mywatchedlist/worker/collector"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMatch(t *testing.T) {
	for _, mode := range []string{"movie", "tv", "ambiguous", "not_found", "failure", "multi_page"} {
		t.Run(mode, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer synthetic-token" || r.URL.Query().Get("language") != "pt-BR" {
					t.Error("autorização/idioma ausentes")
				}
				w.Header().Set("Content-Type", "application/json")
				if mode == "failure" {
					w.WriteHeader(429)
					fmt.Fprint(w, `{"secret":"não deve vazar"}`)
					return
				}
				if strings.HasPrefix(r.URL.Path, "/search/") {
					if mode == "not_found" {
						fmt.Fprint(w, `{"results":[],"total_pages":1}`)
						return
					}
					if mode == "ambiguous" {
						fmt.Fprint(w, `{"results":[{"id":1,"title":"Teste"},{"id":2,"title":"Teste"}],"total_pages":1}`)
						return
					}
					if mode == "multi_page" && r.URL.Query().Get("page") == "1" {
						fmt.Fprint(w, `{"results":[{"id":1,"title":"Teste"}],"total_pages":2}`)
						return
					}
					id := 1
					if mode == "multi_page" {
						id = 2
					}
					fmt.Fprintf(w, `{"results":[{"id":%d,"title":"Teste","name":"Teste"}],"total_pages":1}`, id)
					return
				}
				if strings.Contains(r.URL.Path, "/episode/") {
					fmt.Fprint(w, `{"id":8,"season_number":1,"episode_number":2,"name":"Ep","runtime":40}`)
					return
				}
				fmt.Fprint(w, `{"id":1,"title":"Teste","name":"Teste","original_title":"Original","original_name":"Original","release_date":"2024-01-01","first_air_date":"2024-01-01","overview":"Sinopse","poster_path":"/poster.jpg","genres":[{"name":"Drama"}],"runtime":100}`)
			}))
			defer srv.Close()
			c := New("synthetic-token")
			c.baseURL = srv.URL
			e := collector.Event{MatchTitle: "Teste", MediaType: "movie"}
			if mode == "tv" {
				e.MediaType = "tv"
				season, episode := 1, 2
				e.SeasonNumber = &season
				e.EpisodeNumber = &episode
			}
			title, err := c.Match(context.Background(), e)
			switch mode {
			case "movie", "tv":
				if err != nil || title.TMDBID != 1 || title.Name != "Teste" || title.Overview != "Sinopse" || title.Poster != "/poster.jpg" || len(title.Genres) != 1 || title.Genres[0] != "Drama" || *title.Year != 2024 {
					t.Fatalf("metadados: %+v %v", title, err)
				}
				if mode == "tv" && (!title.EpisodeValid || title.EpisodeName != "Ep" || *title.EpisodeRuntime != 40) {
					t.Fatal("episódio perdido")
				}
			case "ambiguous", "multi_page":
				if !errors.Is(err, ErrAmbiguous) {
					t.Fatal("ambiguidade não preservada", err)
				}
			case "not_found":
				if !errors.Is(err, ErrNotFound) {
					t.Fatal(err)
				}
			case "failure":
				if !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), "synthetic-token") {
					t.Fatal("erro inseguro", err)
				}
			}
		})
	}
}

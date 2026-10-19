package metadata

import (
	"context"
	"errors"
	"fmt"
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
			e := MatchInput{MatchTitle: "Teste", MediaType: "movie"}
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

// AC 9–10: um candidato aproximado ou episódio incompatível nunca é associado.
func TestConservativeMatching(t *testing.T) {
	for _, mode := range []string{"inexact", "episode_mismatch", "episode_id", "episode_season", "unknown_ambiguous", "unknown_unique", "unavailable_title", "movie_original", "tv_original"} {
		t.Run(mode, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if mode == "unavailable_title" {
					t.Error("título indisponível não deve consultar TMDB")
				}
				if strings.HasPrefix(r.URL.Path, "/search/") {
					if mode == "unknown_unique" && r.URL.Path == "/search/tv" {
						fmt.Fprint(w, `{"results":[],"total_pages":1}`)
					} else if mode == "inexact" {
						fmt.Fprint(w, `{"results":[{"id":1,"title":"Teste parecido"}],"total_pages":1}`)
					} else if strings.HasSuffix(mode, "_original") {
						fmt.Fprint(w, `{"results":[{"id":1,"title":"Tradução","name":"Tradução","original_title":"Teste","original_name":"Teste"}],"total_pages":1}`)
					} else {
						// Mesmo ID numérico nas duas mídias ainda representa dois títulos.
						fmt.Fprint(w, `{"results":[{"id":1,"title":"Teste","name":"Teste"}],"total_pages":1}`)
					}
					return
				}
				if strings.HasSuffix(r.URL.Path, "/alternative_titles") {
					fmt.Fprint(w, `{"id":1,"titles":[],"results":[]}`)
					return
				}
				if mode == "inexact" || mode == "unknown_ambiguous" {
					t.Error("candidato não resolvido não deve buscar detalhes")
				}
				if strings.Contains(r.URL.Path, "/episode/") {
					id, season, episode := 8, 1, 2
					switch mode {
					case "episode_id":
						id = 0
					case "episode_season":
						season = 2
					default:
						episode = 3
					}
					fmt.Fprintf(w, `{"id":%d,"season_number":%d,"episode_number":%d,"name":"Outro episódio"}`, id, season, episode)
					return
				}
				if strings.HasSuffix(mode, "_original") {
					fmt.Fprint(w, `{"id":1,"title":"Tradução","name":"Tradução","original_title":"Teste","original_name":"Teste"}`)
					return
				}
				fmt.Fprint(w, `{"id":1,"title":"Teste","name":"Teste"}`)
			}))
			defer srv.Close()
			c := New("synthetic-token")
			c.baseURL = srv.URL
			e := MatchInput{MatchTitle: "Teste", MediaType: "movie"}
			if strings.HasPrefix(mode, "episode_") {
				e.MediaType = "tv"
				season, episode := 1, 2
				e.SeasonNumber, e.EpisodeNumber = &season, &episode
			}
			if mode == "tv_original" {
				e.MediaType = "tv"
			}
			if strings.HasPrefix(mode, "unknown_") {
				e.MediaType = ""
			}
			if mode == "unavailable_title" {
				e.MatchTitle = ""
			}
			title, err := c.Match(context.Background(), e)
			switch mode {
			case "inexact", "unavailable_title":
				if !errors.Is(err, ErrNotFound) || title.TMDBID != 0 {
					t.Fatalf("associação indevida: %+v %v", title, err)
				}
			case "episode_mismatch", "episode_id", "episode_season":
				if !errors.Is(err, ErrUnavailable) || title.EpisodeValid || title.TMDBID != 0 {
					t.Fatalf("episódio incompatível aceito: %+v %v", title, err)
				}
			case "unknown_ambiguous":
				if !errors.Is(err, ErrAmbiguous) || title.TMDBID != 0 {
					t.Fatalf("ambiguidade entre mídias ignorada: %+v %v", title, err)
				}
			case "unknown_unique":
				if err != nil || title.TMDBID != 1 || title.MediaType != "movie" || title.Name != "Teste" {
					t.Fatalf("match único entre mídias: %+v %v", title, err)
				}
			case "movie_original", "tv_original":
				if err != nil || title.TMDBID != 1 || title.Name != "Tradução" || title.Original != "Teste" || title.MediaType != e.MediaType {
					t.Fatalf("título original não reconhecido: %+v %v", title, err)
				}
			}
		})
	}
}

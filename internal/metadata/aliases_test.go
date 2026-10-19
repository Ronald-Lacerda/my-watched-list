package metadata

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestOfficialAliases(t *testing.T) {
	for _, mode := range []string{"movie", "tv", "canonical", "inexact", "ambiguous", "truncated", "many"} {
		t.Run(mode, func(t *testing.T) {
			details := 0
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				if r.Header.Get("Authorization") != "Bearer synthetic-only" || r.URL.Query().Get("language") != "pt-BR" {
					t.Error("autorização/idioma ausentes")
				}
				if strings.HasPrefix(r.URL.Path, "/search/") {
					if mode == "canonical" {
						fmt.Fprint(w, `{"results":[{"id":1,"title":"Plano de Voo"}],"total_pages":1}`)
					} else if mode == "many" {
						fmt.Fprint(w, `{"results":[`)
						from, to := 1, 20
						if r.URL.Query().Get("page") == "2" {
							from, to = 21, 21
						}
						for id := from; id <= to; id++ {
							if id > from {
								fmt.Fprint(w, ",")
							}
							fmt.Fprintf(w, `{"id":%d,"title":"Outro"}`, id)
						}
						fmt.Fprint(w, `],"total_pages":2}`)
					} else if mode == "ambiguous" {
						fmt.Fprint(w, `{"results":[{"id":1,"title":"Outro A"},{"id":2,"title":"Outro B"}],"total_pages":1}`)
					} else {
						fmt.Fprint(w, `{"results":[{"id":1,"title":"Nome Oficial","name":"Nome Oficial"}],"total_pages":1}`)
					}
					return
				}
				if strings.HasSuffix(r.URL.Path, "/alternative_titles") {
					parts := strings.Split(r.URL.Path, "/")
					id, _ := strconv.Atoi(parts[2])
					alias := "Nome Alternativo"
					if mode == "inexact" {
						alias += " Parecido"
					}
					field := "titles"
					if mode == "tv" {
						field = "results"
					}
					fmt.Fprintf(w, `{"id":%d,%q:[{"title":%q}]}`, id, field, alias)
					return
				}
				details++
				fmt.Fprint(w, `{"id":1,"title":"Nome Oficial","name":"Nome Oficial","overview":"Sinopse oficial","poster_path":"/cover.jpg","release_date":"2020-01-01","first_air_date":"2020-01-01"}`)
			}))
			defer srv.Close()
			client := New("synthetic-only")
			client.baseURL = srv.URL
			input := MatchInput{MatchTitle: "Nome Alternativo", MediaType: "movie"}
			if mode == "tv" {
				input.MediaType = "tv"
			}
			if mode == "canonical" {
				input.MatchTitle = "Plano-de-vôo!"
			}
			if mode == "truncated" {
				input.MatchTitle += "..."
			}
			value, err := client.Match(context.Background(), input)
			switch mode {
			case "movie", "tv", "canonical":
				if err != nil || value.TMDBID != 1 || value.Name != "Nome Oficial" || value.Poster != "/cover.jpg" || value.Overview != "Sinopse oficial" || details != 1 {
					t.Fatal("alias oficial não resolveu dados corretos", value, err, details)
				}
			case "ambiguous", "many":
				if !errors.Is(err, ErrAmbiguous) || details != 0 {
					t.Fatal("ambiguidade associada", value, err, details)
				}
			case "inexact", "truncated":
				if !errors.Is(err, ErrNotFound) || details != 0 {
					t.Fatal("aproximação associada", value, err, details)
				}
				if mode == "truncated" && calls != 0 {
					t.Fatal("título truncado não deve ser associado automaticamente")
				}
			}
		})
	}
}

func TestMetadataSearchAndFallback(t *testing.T) {
	english := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/search/") {
			if r.URL.Query().Get("query") != "Nome truncado" || r.URL.Query().Get("include_adult") != "false" {
				t.Error("busca assistida incorreta")
			}
			fmt.Fprint(w, `{"results":[{"id":11,"title":"Filme","release_date":"2022-03-04","poster_path":"/poster.jpg","overview":"Sinopse"}]}`)
			return
		}
		if r.URL.Query().Get("language") == "en-US" {
			english++
			fmt.Fprint(w, `{"id":11,"title":"Movie","overview":"Official overview","poster_path":"/english.jpg"}`)
			return
		}
		fmt.Fprint(w, `{"id":11,"title":"Título em português","overview":"","poster_path":null,"genres":[{"name":"Drama"}]}`)
	}))
	defer srv.Close()
	client := New("synthetic-only")
	client.baseURL = srv.URL
	items, err := client.Search(context.Background(), "Nome truncado...", "movie")
	if err != nil || len(items) != 1 || items[0].ID != 11 || items[0].MediaType != "movie" || items[0].Title != "Filme" || items[0].Year == nil || *items[0].Year != 2022 || items[0].Overview != "Sinopse" || items[0].PosterPath == nil || *items[0].PosterPath != "/poster.jpg" {
		t.Fatal("candidato incompleto", items, err)
	}
	value, err := client.Details(context.Background(), "movie", 11)
	if err != nil || english != 1 || value.Name != "Título em português" || value.Overview != "Official overview" || value.Language != "en-US" || value.Poster != "/english.jpg" || len(value.Genres) != 1 || value.Genres[0] != "Drama" {
		t.Fatal("fallback não preservou fonte/título", value, err, english)
	}
	for _, kind := range []string{"person", "../../x"} {
		if _, err := client.Search(context.Background(), "abc", kind); !errors.Is(err, ErrInvalid) {
			t.Fatal("tipo inválido aceito", kind, err)
		}
	}
	if _, err := client.Details(context.Background(), "movie", -1); !errors.Is(err, ErrInvalid) {
		t.Fatal("ID inválido aceito", err)
	}
}

func TestMetadataPayloadErrors(t *testing.T) {
	for _, mode := range []string{"status", "oversize", "invalid", "wrong-id"} {
		t.Run(mode, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch mode {
				case "status":
					w.WriteHeader(429)
					fmt.Fprint(w, "synthetic-secret")
				case "oversize":
					fmt.Fprint(w, `{"id":1,"title":"`+strings.Repeat("x", (2<<20)+1)+`"}`)
				case "invalid":
					fmt.Fprint(w, `{"id":`)
				case "wrong-id":
					fmt.Fprint(w, `{"id":2,"title":"Outro","overview":"Outro","poster_path":"/other.jpg"}`)
				}
			}))
			defer srv.Close()
			client := New("synthetic-secret")
			client.baseURL = srv.URL
			if _, err := client.Details(context.Background(), "movie", 1); !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), "synthetic-secret") {
				t.Fatal("erro inseguro ou payload aceito", err)
			}
		})
	}
}

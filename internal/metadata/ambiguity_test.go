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

func TestCanonicalAndAliasUniqueness(t *testing.T) {
	for _, mode := range []string{"canonical_vs_alias", "cross_media", "canonical_limit", "unique_primary", "alias_unavailable", "two_primary"} {
		t.Run(mode, func(t *testing.T) {
			details := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasPrefix(r.URL.Path, "/search/") {
					if mode == "canonical_limit" {
						fmt.Fprint(w, `{"results":[`)
						from, to := 1, 20
						if r.URL.Query().Get("page") == "2" {
							from, to = 21, 21
						}
						for id := from; id <= to; id++ {
							if id > from {
								fmt.Fprint(w, ",")
							}
							name := "Outra obra"
							if id == 1 {
								name = "Nome Comum"
							}
							fmt.Fprintf(w, `{"id":%d,"title":%q}`, id, name)
						}
						fmt.Fprint(w, `],"total_pages":2}`)
					} else if mode == "cross_media" {
						if r.URL.Path == "/search/movie" {
							fmt.Fprint(w, `{"results":[{"id":1,"title":"Nome Comum"}],"total_pages":1}`)
						} else {
							fmt.Fprint(w, `{"results":[{"id":1,"name":"Outra obra"}],"total_pages":1}`)
						}
					} else if mode == "two_primary" {
						fmt.Fprint(w, `{"results":[{"id":1,"title":"Nome Comum"},{"id":2,"title":"Nome Comum"}],"total_pages":1}`)
					} else {
						fmt.Fprint(w, `{"results":[{"id":1,"title":"Nome Comum"},{"id":2,"title":"Outra obra"}],"total_pages":1}`)
					}
					return
				}
				if strings.HasSuffix(r.URL.Path, "/alternative_titles") {
					if mode == "alias_unavailable" {
						w.WriteHeader(429)
						return
					}
					id, field := 2, "titles"
					if mode == "cross_media" {
						id, field = 1, "results"
					}
					alias := "Nome Comum"
					if mode == "unique_primary" {
						alias = "Nome Diferente"
					}
					fmt.Fprintf(w, `{"id":%d,%q:[{"title":%q}]}`, id, field, alias)
					return
				}
				details++
				fmt.Fprint(w, `{"id":1,"title":"Nome Comum","overview":"Sinopse confirmada","poster_path":"/correct.jpg"}`)
			}))
			defer srv.Close()
			client := New("synthetic-only")
			client.baseURL = srv.URL
			input := MatchInput{MatchTitle: "Nome Comum", MediaType: "movie"}
			if mode == "cross_media" {
				input.MediaType = ""
			}
			got, err := client.Match(context.Background(), input)
			switch mode {
			case "unique_primary":
				if err != nil || got.TMDBID != 1 || got.Name != "Nome Comum" || got.Overview != "Sinopse confirmada" || details != 1 {
					t.Fatal("nome único não resolveu dados", got, err, details)
				}
			case "alias_unavailable":
				if !errors.Is(err, ErrUnavailable) || got.TMDBID != 0 || details != 0 {
					t.Fatal("incerteza no alias não pode confirmar", got, err, details)
				}
			default:
				if !errors.Is(err, ErrAmbiguous) || got.TMDBID != 0 || details != 0 {
					t.Fatal("ambiguidade/cap com canônico foi ignorado", got, err, details)
				}
			}
		})
	}
}

// Package tmdb consulta somente metadados oficiais, sem delegar coleta.
package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"mywatchedlist/worker/collector"
	"mywatchedlist/worker/store"
)

var (
	ErrUnavailable = errors.New("TMDB indisponível")
	ErrNotFound    = errors.New("título não encontrado")
	ErrAmbiguous   = errors.New("título ambíguo")
)

type Client struct {
	token   string
	baseURL string
	http    *http.Client
}

func New(token string) *Client {
	return &Client{token: token, baseURL: "https://api.themoviedb.org/3", http: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	if c.token == "" {
		return ErrUnavailable
	}
	if query == nil {
		query = url.Values{}
	}
	query.Set("language", "pt-BR")
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path+"?"+query.Encode(), nil)
	if err != nil {
		return ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ErrUnavailable
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(out); err != nil {
		return ErrUnavailable
	}
	return nil
}

type candidate struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	Name          string `json:"name"`
	OriginalTitle string `json:"original_title"`
	OriginalName  string `json:"original_name"`
	kind          string
}

func exact(a, b string) bool {
	return strings.EqualFold(strings.Join(strings.Fields(a), " "), strings.Join(strings.Fields(b), " "))
}
func (c *Client) Match(ctx context.Context, e collector.Event) (store.Title, error) {
	if strings.TrimSpace(e.MatchTitle) == "" {
		return store.Title{}, ErrNotFound
	}
	kinds := []string{"movie", "tv"}
	if e.MediaType != "" {
		if e.MediaType != "movie" && e.MediaType != "tv" {
			return store.Title{}, ErrNotFound
		}
		kinds = []string{e.MediaType}
	}
	matches := map[string]candidate{}
	for _, kind := range kinds {
		for page := 1; ; page++ {
			var result struct {
				Results    []candidate `json:"results"`
				TotalPages int         `json:"total_pages"`
			}
			err := c.get(ctx, "/search/"+kind, url.Values{"query": {e.MatchTitle}, "include_adult": {"false"}, "page": {strconv.Itoa(page)}}, &result)
			if err != nil {
				return store.Title{}, err
			}
			if result.Results == nil {
				return store.Title{}, ErrUnavailable
			}
			if result.TotalPages > 10 {
				return store.Title{}, ErrAmbiguous
			}
			for _, item := range result.Results {
				if item.ID <= 0 {
					return store.Title{}, ErrUnavailable
				}
				if exact(item.Title, e.MatchTitle) || exact(item.Name, e.MatchTitle) || exact(item.OriginalTitle, e.MatchTitle) || exact(item.OriginalName, e.MatchTitle) {
					item.kind = kind
					matches[fmt.Sprintf("%s:%d", kind, item.ID)] = item
				}
			}
			if page >= result.TotalPages {
				break
			}
		}
	}
	if len(matches) == 0 {
		return store.Title{}, ErrNotFound
	}
	if len(matches) != 1 {
		return store.Title{}, ErrAmbiguous
	}
	var found candidate
	for _, v := range matches {
		found = v
	}
	var detail struct {
		ID            int64  `json:"id"`
		Title         string `json:"title"`
		Name          string `json:"name"`
		OriginalTitle string `json:"original_title"`
		OriginalName  string `json:"original_name"`
		Release       string `json:"release_date"`
		FirstAir      string `json:"first_air_date"`
		Overview      string `json:"overview"`
		Poster        string `json:"poster_path"`
		Runtime       int    `json:"runtime"`
		Genres        []struct {
			Name string `json:"name"`
		} `json:"genres"`
	}
	if err := c.get(ctx, fmt.Sprintf("/%s/%d", found.kind, found.ID), nil, &detail); err != nil {
		return store.Title{}, err
	}
	m := store.Title{TMDBID: found.ID, MediaType: found.kind, Name: detail.Title, Original: detail.OriginalTitle, Overview: detail.Overview, Poster: detail.Poster}
	date := detail.Release
	if found.kind == "tv" {
		m.Name = detail.Name
		m.Original = detail.OriginalName
		date = detail.FirstAir
	}
	if detail.ID != found.ID || strings.TrimSpace(m.Name) == "" {
		return store.Title{}, ErrUnavailable
	}
	if len(date) >= 4 {
		year, err := strconv.Atoi(date[:4])
		if err == nil && year >= 1800 && year <= 9999 {
			m.Year = &year
		}
	}
	if detail.Runtime > 0 {
		m.Runtime = &detail.Runtime
	}
	for _, g := range detail.Genres {
		if g.Name != "" {
			m.Genres = append(m.Genres, g.Name)
		}
	}
	if found.kind == "tv" && e.SeasonNumber != nil && e.EpisodeNumber != nil {
		var episode struct {
			ID      int64  `json:"id"`
			Name    string `json:"name"`
			Runtime int    `json:"runtime"`
			Season  int    `json:"season_number"`
			Number  int    `json:"episode_number"`
		}
		if err := c.get(ctx, fmt.Sprintf("/tv/%d/season/%d/episode/%d", found.ID, *e.SeasonNumber, *e.EpisodeNumber), nil, &episode); err != nil {
			return store.Title{}, err
		}
		if episode.ID <= 0 || episode.Season != *e.SeasonNumber || episode.Number != *e.EpisodeNumber {
			return store.Title{}, ErrUnavailable
		}
		m.EpisodeValid = true
		m.EpisodeName = episode.Name
		if episode.Runtime > 0 {
			m.EpisodeRuntime = &episode.Runtime
		}
	}
	return m, nil
}

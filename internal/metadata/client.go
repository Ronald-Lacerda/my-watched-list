// Package metadata consulta metadados oficiais do TMDB para API e worker.
package metadata

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
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var (
	ErrUnavailable = errors.New("TMDB indisponível")
	ErrNotFound    = errors.New("título não encontrado")
	ErrAmbiguous   = errors.New("título ambíguo")
	ErrInvalid     = errors.New("consulta de metadados inválida")
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
	if query.Get("language") == "" {
		query.Set("language", "pt-BR")
	}
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if err != nil || len(body) > 2<<20 {
		return ErrUnavailable
	}
	if err = json.Unmarshal(body, out); err != nil {
		return ErrUnavailable
	}
	return nil
}

type candidate struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	Name          string  `json:"name"`
	OriginalTitle string  `json:"original_title"`
	OriginalName  string  `json:"original_name"`
	Release       string  `json:"release_date"`
	FirstAir      string  `json:"first_air_date"`
	Overview      string  `json:"overview"`
	Poster        *string `json:"poster_path"`
	kind          string
}

func exact(a, b string) bool {
	canonical := func(value string) string {
		var out strings.Builder
		for _, char := range norm.NFD.String(strings.ToLower(value)) {
			if unicode.IsLetter(char) || unicode.IsDigit(char) {
				out.WriteRune(char)
			}
		}
		return out.String()
	}
	left, right := canonical(a), canonical(b)
	return left != "" && left == right
}
func (c *Client) Match(ctx context.Context, e MatchInput) (Title, error) {
	if strings.TrimSpace(e.MatchTitle) == "" || strings.HasSuffix(strings.TrimSpace(e.MatchTitle), "...") || strings.HasSuffix(strings.TrimSpace(e.MatchTitle), "…") {
		return Title{}, ErrNotFound
	}
	kinds := []string{"movie", "tv"}
	if e.MediaType != "" {
		if e.MediaType != "movie" && e.MediaType != "tv" {
			return Title{}, ErrNotFound
		}
		kinds = []string{e.MediaType}
	}
	matches := map[string]candidate{}
	candidates := map[string]candidate{}
	for _, kind := range kinds {
		for page := 1; ; page++ {
			var result struct {
				Results    []candidate `json:"results"`
				TotalPages int         `json:"total_pages"`
			}
			err := c.get(ctx, "/search/"+kind, url.Values{"query": {e.MatchTitle}, "include_adult": {"false"}, "page": {strconv.Itoa(page)}}, &result)
			if err != nil {
				return Title{}, err
			}
			if result.Results == nil {
				return Title{}, ErrUnavailable
			}
			if result.TotalPages > 10 {
				return Title{}, ErrAmbiguous
			}
			for _, item := range result.Results {
				if item.ID <= 0 {
					return Title{}, ErrUnavailable
				}
				item.kind = kind
				candidates[fmt.Sprintf("%s:%d", kind, item.ID)] = item
				if exact(item.Title, e.MatchTitle) || exact(item.Name, e.MatchTitle) || exact(item.OriginalTitle, e.MatchTitle) || exact(item.OriginalName, e.MatchTitle) {
					matches[fmt.Sprintf("%s:%d", kind, item.ID)] = item
				}
			}
			if page >= result.TotalPages {
				break
			}
		}
	}
	// Um nome canônico não elimina um alias oficial de outra obra. Verificar
	// os demais candidatos antes de aceitar a associação única.
	if len(matches) <= 1 {
		if len(candidates) > 20 {
			return Title{}, ErrAmbiguous
		}
		for key, item := range candidates {
			if _, matched := matches[key]; matched {
				continue
			}
			var aliases struct {
				ID     int64 `json:"id"`
				Titles []struct {
					Title string `json:"title"`
				} `json:"titles"`
				Results []struct {
					Title string `json:"title"`
				} `json:"results"`
			}
			if err := c.get(ctx, fmt.Sprintf("/%s/%d/alternative_titles", item.kind, item.ID), nil, &aliases); err != nil {
				return Title{}, err
			}
			if aliases.ID != item.ID || aliases.Titles == nil && aliases.Results == nil {
				return Title{}, ErrUnavailable
			}
			for _, alias := range append(aliases.Titles, aliases.Results...) {
				if exact(alias.Title, e.MatchTitle) {
					matches[key] = item
				}
			}
		}
	}
	if len(matches) == 0 {
		return Title{}, ErrNotFound
	}
	if len(matches) != 1 {
		return Title{}, ErrAmbiguous
	}
	var found candidate
	for _, v := range matches {
		found = v
	}
	m, err := c.Details(ctx, found.kind, found.ID)
	if err != nil {
		return Title{}, err
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
			return Title{}, err
		}
		if episode.ID <= 0 || episode.Season != *e.SeasonNumber || episode.Number != *e.EpisodeNumber {
			return Title{}, ErrUnavailable
		}
		m.EpisodeValid = true
		m.EpisodeName = episode.Name
		if episode.Runtime > 0 {
			m.EpisodeRuntime = &episode.Runtime
		}
	}
	return m, nil
}

// Details não aceita URLs do chamador; os caminhos usam apenas tipo e ID validados.
func (c *Client) Details(ctx context.Context, kind string, id int64) (Title, error) {
	if (kind != "movie" && kind != "tv") || id <= 0 {
		return Title{}, ErrInvalid
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
	if err := c.get(ctx, fmt.Sprintf("/%s/%d", kind, id), nil, &detail); err != nil {
		return Title{}, err
	}
	m := Title{TMDBID: id, MediaType: kind, Name: detail.Title, Original: detail.OriginalTitle, Overview: detail.Overview, Poster: detail.Poster}
	date := detail.Release
	if kind == "tv" {
		m.Name = detail.Name
		m.Original = detail.OriginalName
		date = detail.FirstAir
	}
	if detail.ID != id || strings.TrimSpace(m.Name) == "" {
		return Title{}, ErrUnavailable
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

	m.Language = "pt-BR"
	if m.Overview == "" || m.Poster == "" {
		var fallback struct {
			ID       int64  `json:"id"`
			Overview string `json:"overview"`
			Poster   string `json:"poster_path"`
		}
		if err := c.get(ctx, fmt.Sprintf("/%s/%d", kind, id), url.Values{"language": {"en-US"}}, &fallback); err != nil {
			return Title{}, err
		}
		if fallback.ID != id {
			return Title{}, ErrUnavailable
		}
		if m.Overview == "" && fallback.Overview != "" {
			m.Overview = fallback.Overview
			m.Language = "en-US"
		}
		if m.Poster == "" {
			m.Poster = fallback.Poster
		}
	}
	return m, nil
}

// Search serve à escolha explícita; seus candidatos não são associações automáticas.
func (c *Client) Search(ctx context.Context, query, kind string) ([]Candidate, error) {
	query = strings.TrimSpace(query)
	query = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(query, "..."), "…"))
	if utf8.RuneCountInString(query) < 2 || utf8.RuneCountInString(query) > 200 || kind != "" && kind != "movie" && kind != "tv" {
		return nil, ErrInvalid
	}
	kinds := []string{"movie", "tv"}
	if kind != "" {
		kinds = []string{kind}
	}
	items := []Candidate{}
	seen := map[string]bool{}
	for _, mediaType := range kinds {
		var result struct {
			Results []candidate `json:"results"`
		}
		if err := c.get(ctx, "/search/"+mediaType, url.Values{"query": {query}, "include_adult": {"false"}, "page": {"1"}}, &result); err != nil {
			return nil, err
		}
		if result.Results == nil || len(result.Results) > 20 {
			return nil, ErrUnavailable
		}
		for _, row := range result.Results {
			name, date := row.Title, row.Release
			if mediaType == "tv" {
				name, date = row.Name, row.FirstAir
			}
			if row.ID <= 0 || strings.TrimSpace(name) == "" {
				return nil, ErrUnavailable
			}
			key := fmt.Sprintf("%s:%d", mediaType, row.ID)
			if seen[key] {
				continue
			}
			seen[key] = true
			var year *int
			if len(date) >= 4 {
				if value, err := strconv.Atoi(date[:4]); err == nil && value >= 1800 && value <= 9999 {
					year = &value
				}
			}
			items = append(items, Candidate{ID: row.ID, MediaType: mediaType, Title: name, Year: year, PosterPath: row.Poster, Overview: row.Overview})
		}
	}
	return items, nil
}

package metadata

type MatchInput struct {
	MatchTitle, MediaType       string
	SeasonNumber, EpisodeNumber *int
}

type Title struct {
	TMDBID         int64    `json:"tmdb_id"`
	MediaType      string   `json:"media_type"`
	Name           string   `json:"title"`
	Original       string   `json:"original_title"`
	Overview       string   `json:"overview"`
	Language       string   `json:"overview_language"`
	Poster         string   `json:"poster_path"`
	Year           *int     `json:"release_year"`
	Runtime        *int     `json:"runtime_minutes"`
	Genres         []string `json:"genres"`
	EpisodeValid   bool     `json:"-"`
	EpisodeName    string   `json:"-"`
	EpisodeRuntime *int     `json:"-"`
}

type Candidate struct {
	ID         int64   `json:"id"`
	MediaType  string  `json:"media_type"`
	Title      string  `json:"title"`
	Year       *int    `json:"year"`
	PosterPath *string `json:"poster_path"`
	Overview   string  `json:"overview"`
}

package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"mywatchedlist/api/internal/library"
	"mywatchedlist/internal/catalogstore"
	"mywatchedlist/internal/metadata"
)

type MetadataClient interface {
	Search(context.Context, string, string) ([]metadata.Candidate, error)
	Details(context.Context, string, int64) (metadata.Title, error)
}
type NetflixMetadataClient interface {
	Lookup(context.Context, string) (metadata.NetflixTitle, error)
}

func (a *app) searchMetadata(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := strings.TrimSpace(q.Get("query"))
	kind := q.Get("type")
	valid := utf8.RuneCountInString(query) >= 2 && utf8.RuneCountInString(query) <= 200 && (kind == "" || kind == "movie" || kind == "tv")
	for key, values := range q {
		if (key != "query" && key != "type") || len(values) != 1 {
			valid = false
		}
	}
	if !valid {
		apiError(w, 400, "invalid_search", "Busque um título de 2 a 200 caracteres, como filme ou série.")
		return
	}
	if a.metadata == nil {
		a.metadataError(w, metadata.ErrUnavailable)
		return
	}
	items, err := a.metadata.Search(r.Context(), query, kind)
	if err != nil {
		a.metadataError(w, err)
		return
	}
	if items == nil {
		items = []metadata.Candidate{}
	}
	reply(w, 200, map[string]any{"items": items})
}

func (a *app) confirmMetadata(w http.ResponseWriter, r *http.Request) {
	if !a.origin(w, r) {
		return
	}
	id := chi.URLParam(r, "id")
	if !library.ValidID(id) || r.URL.RawQuery != "" {
		apiError(w, 400, "invalid_metadata", "Identificação inválida.")
		return
	}
	var body struct {
		Provider  string `json:"provider"`
		ID        int64  `json:"tmdb_id"`
		MediaType string `json:"media_type"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if (body.Provider != "tmdb" && body.Provider != "netflix") || (body.Provider == "tmdb" && (body.ID <= 0 || (body.MediaType != "movie" && body.MediaType != "tv"))) || (body.Provider == "netflix" && (body.ID != 0 || body.MediaType != "")) {
		apiError(w, 400, "invalid_metadata", "Escolha uma identificação válida.")
		return
	}
	identity, err := catalogstore.ReadIdentity(r.Context(), a.pool, id)
	if err != nil {
		a.metadataError(w, err)
		return
	}
	if identity.Key == "" {
		a.metadataError(w, catalogstore.ErrIdentity)
		return
	}
	actor := currentUser(r).ID
	if body.Provider == "tmdb" {
		if identity.MediaType == "tv" && body.MediaType != "tv" {
			a.metadataError(w, catalogstore.ErrType)
			return
		}
		if a.metadata == nil {
			a.metadataError(w, metadata.ErrUnavailable)
			return
		}
		title, e := a.metadata.Details(r.Context(), body.MediaType, body.ID)
		if e != nil {
			a.metadataError(w, e)
			return
		}
		// Não confiar em um resultado remoto incompatível com a seleção.
		if title.TMDBID != body.ID || title.MediaType != body.MediaType {
			a.metadataError(w, metadata.ErrUnavailable)
			return
		}
		err = catalogstore.ConfirmTMDB(r.Context(), a.pool, id, actor, title)
	} else {
		if identity.Service != "netflix" {
			a.metadataError(w, catalogstore.ErrIdentity)
			return
		}
		if a.netflixMetadata == nil {
			a.metadataError(w, metadata.ErrUnavailable)
			return
		}
		title, e := a.netflixMetadata.Lookup(r.Context(), identity.SourceID)
		if e != nil {
			a.metadataError(w, e)
			return
		}
		if title.Overview == "" && title.PosterURL == "" {
			a.metadataError(w, metadata.ErrNotFound)
			return
		}
		err = catalogstore.ConfirmNetflix(r.Context(), a.pool, id, actor, title)
	}
	if err != nil {
		a.metadataError(w, err)
		return
	}
	reply(w, 200, map[string]bool{"updated": true})
}

func (a *app) metadataError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, metadata.ErrNotFound):
		apiError(w, 404, "metadata_not_found", "Não encontramos informações oficiais para esta obra.")
	case errors.Is(err, catalogstore.ErrIdentity):
		apiError(w, 409, "sync_required", "Esta obra precisa de uma nova sincronização antes de corrigir informações.")
	case errors.Is(err, catalogstore.ErrType):
		apiError(w, 400, "incompatible_type", "Escolha uma série para este registro.")
	default:
		apiError(w, 503, "metadata_unavailable", "A fonte de informações está indisponível agora. Tente novamente.")
	}
}

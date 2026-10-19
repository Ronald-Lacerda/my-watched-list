// Package catalogstore compartilha a identidade e a confirmação de metadados.
package catalogstore

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"mywatchedlist/internal/metadata"
)

var ErrIdentity = errors.New("obra sem identidade confiável; sincronize novamente")
var ErrType = errors.New("tipo incompatível com a obra de origem")

type Identity struct{ Key, Service, SourceID, MediaType string }
type Querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

const SourceKey = `CASE WHEN e.source_work_id<>'' THEN a.service||':work:'||e.source_work_id WHEN e.media_type_hint<>'tv' AND nullif(e.source_title_id,'') IS NOT NULL THEN a.service||':work:'||e.source_title_id ELSE NULL END`

func ReadIdentity(ctx context.Context, q Querier, id string) (Identity, error) {
	var i Identity
	err := q.QueryRow(ctx, `SELECT coalesce(`+SourceKey+`,''),a.service,coalesce(nullif(e.source_work_id,''),e.source_title_id,''),CASE WHEN e.media_type_hint='tv' OR t.media_type='tv' OR o.media_type='tv' THEN 'tv' ELSE coalesce(nullif(e.media_type_hint,''),o.media_type,t.media_type,'') END FROM watch_events e JOIN streaming_accounts a ON a.id=e.streaming_account_id LEFT JOIN titles t ON t.id=e.title_id LEFT JOIN metadata_overrides o ON o.source_key=(`+SourceKey+`) WHERE e.id=$1`, id).Scan(&i.Key, &i.Service, &i.SourceID, &i.MediaType)
	return i, err
}

// A ordem é lock de obra, depois linhas. Nunca segurar uma linha enquanto
// espera pelo lock de obra: a confirmação atualiza várias pessoas de uma vez.
func LockIdentity(ctx context.Context, tx pgx.Tx, id string) (Identity, error) {
	i, err := ReadIdentity(ctx, tx, id)
	if err != nil {
		return i, err
	}
	key := i.Key
	if key == "" {
		key = "event:" + id
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,1))`, key); err != nil {
		return i, err
	}
	var locked string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM watch_events WHERE id=$1 FOR UPDATE`, id).Scan(&locked); err != nil {
		return i, err
	}
	current, err := ReadIdentity(ctx, tx, id)
	if err == nil && current != i {
		err = ErrIdentity
	}
	return current, err
}

func UpsertTitle(ctx context.Context, tx pgx.Tx, m metadata.Title) (string, error) {
	var id string
	genres := m.Genres
	if genres == nil {
		genres = []string{}
	}
	language := m.Language
	if language == "" {
		language = "pt-BR"
	}
	err := tx.QueryRow(ctx, `INSERT INTO titles(tmdb_id,media_type,title,original_title,release_year,overview,poster_path,genres,runtime_minutes,overview_language) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(tmdb_id,media_type) DO UPDATE SET title=excluded.title,original_title=excluded.original_title,release_year=excluded.release_year,overview=excluded.overview,poster_path=excluded.poster_path,genres=excluded.genres,runtime_minutes=excluded.runtime_minutes,overview_language=excluded.overview_language,enriched_at=now() RETURNING id::text`, m.TMDBID, m.MediaType, m.Name, m.Original, m.Year, m.Overview, m.Poster, genres, m.Runtime, language).Scan(&id)
	return id, err
}

// ApplyOverride deve ser chamado sob LockIdentity, inclusive após uma busca
// remota: uma confirmação feita durante a busca prevalece sobre seu resultado.
func ApplyOverride(ctx context.Context, tx pgx.Tx, i Identity, eventID string) (bool, error) {
	var provider string
	var titleID *string
	err := tx.QueryRow(ctx, `SELECT provider,title_id::text FROM metadata_overrides WHERE source_key=$1`, i.Key).Scan(&provider, &titleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if provider == "tmdb" {
		_, err = tx.Exec(ctx, `UPDATE watch_events SET episode_id=CASE WHEN title_id=$2::uuid THEN episode_id ELSE NULL END,title_id=$2,enrichment_status='matched' WHERE id=$1`, eventID, titleID)
	}
	return true, err
}

func ConfirmTMDB(ctx context.Context, pool *pgxpool.Pool, eventID, actor string, m metadata.Title) error {
	if m.TMDBID <= 0 || (m.MediaType != "movie" && m.MediaType != "tv") || strings.TrimSpace(m.Name) == "" {
		return ErrType
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	i, err := LockIdentity(ctx, tx, eventID)
	if err != nil {
		return err
	}
	if i.Key == "" {
		return ErrIdentity
	}
	if i.MediaType == "tv" && m.MediaType != "tv" {
		return ErrType
	}
	id, err := UpsertTitle(ctx, tx, m)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO metadata_overrides(source_key,provider,title_id,updated_by,media_type) VALUES($1,'tmdb',$2,$3,$4) ON CONFLICT(source_key) DO UPDATE SET provider='tmdb',media_type=excluded.media_type,title_id=excluded.title_id,display_title='',overview='',overview_language='',poster_url='',updated_by=excluded.updated_by,updated_at=now()`, i.Key, id, actor, m.MediaType)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE watch_events e SET episode_id=CASE WHEN e.title_id=$2::uuid THEN e.episode_id ELSE NULL END,title_id=$2,enrichment_status='matched' FROM streaming_accounts a WHERE a.id=e.streaming_account_id AND (`+SourceKey+`)=$1`, i.Key, id)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func ConfirmNetflix(ctx context.Context, pool *pgxpool.Pool, eventID, actor string, m metadata.NetflixTitle) error {
	if m.Name == "" || (m.PosterURL != "" && !metadata.ValidNetflixImageURL(m.PosterURL)) || (m.MediaType != "movie" && m.MediaType != "tv") {
		return ErrType
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	i, err := LockIdentity(ctx, tx, eventID)
	if err != nil {
		return err
	}
	if i.Key == "" || i.Service != "netflix" || m.NetflixID != i.SourceID {
		return ErrIdentity
	}
	if i.MediaType == "tv" && m.MediaType != "tv" {
		return ErrType
	}
	// Um fallback automático iniciado antes de uma escolha explícita não a apaga.
	if actor == "" {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM metadata_overrides WHERE source_key=$1)`, i.Key).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return tx.Commit(ctx)
		}
	}
	var owner any
	if actor != "" {
		owner = actor
	}
	_, err = tx.Exec(ctx, `INSERT INTO metadata_overrides(source_key,provider,display_title,overview,overview_language,poster_url,updated_by,media_type) VALUES($1,'netflix',$2,$3,$4,$5,$6,$7) ON CONFLICT(source_key) DO UPDATE SET provider='netflix',media_type=excluded.media_type,title_id=NULL,display_title=excluded.display_title,overview=excluded.overview,overview_language=excluded.overview_language,poster_url=excluded.poster_url,updated_by=excluded.updated_by,updated_at=now()`, i.Key, m.Name, m.Overview, m.Language, m.PosterURL, owner, m.MediaType)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

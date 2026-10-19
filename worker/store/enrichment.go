package store

import (
	"context"
	"mywatchedlist/internal/metadata"
	"mywatchedlist/worker/collector"
)

type PendingEvent struct {
	ID    string
	Event collector.Event
}
type Title = metadata.Title

func (l *Lease) Pending(ctx context.Context, accountID string) ([]PendingEvent, error) {
	rows, err := l.Conn.Query(ctx, `SELECT id::text,source_title_id,source_title,match_title,media_type_hint,season_number,episode_number FROM watch_events WHERE streaming_account_id=$1 AND title_id IS NULL AND enrich_after<=now() ORDER BY enrich_after LIMIT 500`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []PendingEvent
	for rows.Next() {
		var p PendingEvent
		var sourceID *string
		err = rows.Scan(&p.ID, &sourceID, &p.Event.SourceTitle, &p.Event.MatchTitle, &p.Event.MediaType, &p.Event.SeasonNumber, &p.Event.EpisodeNumber)
		if err != nil {
			return nil, err
		}
		if sourceID != nil {
			p.Event.SourceTitleID = *sourceID
		}
		list = append(list, p)
	}
	return list, rows.Err()
}
func (l *Lease) DeferMatch(ctx context.Context, id, status string) error {
	_, err := l.Conn.Exec(ctx, `UPDATE watch_events SET enrichment_status=$2,enrich_after=now()+CASE WHEN $2='pending' THEN interval '15 minutes' ELSE interval '1 day' END WHERE id=$1 AND title_id IS NULL`, id, status)
	return err
}
func (l *Lease) Link(ctx context.Context, p PendingEvent, m Title) error {
	tx, err := l.Conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	var titleID string
	genres := m.Genres
	if genres == nil {
		genres = []string{}
	}
	err = tx.QueryRow(ctx, `INSERT INTO titles(tmdb_id,media_type,title,original_title,release_year,overview,poster_path,genres,runtime_minutes) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(tmdb_id,media_type) DO UPDATE SET title=excluded.title,original_title=excluded.original_title,release_year=excluded.release_year,overview=excluded.overview,poster_path=excluded.poster_path,genres=excluded.genres,runtime_minutes=excluded.runtime_minutes,enriched_at=now() RETURNING id::text`, m.TMDBID, m.MediaType, m.Name, m.Original, m.Year, m.Overview, m.Poster, genres, m.Runtime).Scan(&titleID)
	if err != nil {
		return err
	}
	var episodeID *string
	e := p.Event
	if m.EpisodeValid && m.MediaType == "tv" && e.SeasonNumber != nil && e.EpisodeNumber != nil {
		var id string
		err = tx.QueryRow(ctx, `INSERT INTO episodes(title_id,season_number,episode_number,title,runtime_minutes) VALUES($1,$2,$3,$4,$5) ON CONFLICT(title_id,season_number,episode_number) DO UPDATE SET title=excluded.title,runtime_minutes=excluded.runtime_minutes RETURNING id::text`, titleID, *e.SeasonNumber, *e.EpisodeNumber, m.EpisodeName, m.EpisodeRuntime).Scan(&id)
		if err != nil {
			return err
		}
		episodeID = &id
	}
	_, err = tx.Exec(ctx, `UPDATE watch_events SET title_id=$2,episode_id=$3,enrichment_status='matched' WHERE id=$1`, p.ID, titleID, episodeID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

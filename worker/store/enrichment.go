package store

import (
	"context"
	"mywatchedlist/internal/catalogstore"
	"mywatchedlist/internal/metadata"
	"mywatchedlist/worker/collector"
)

type PendingEvent struct {
	ID    string
	Event collector.Event
}
type Title = metadata.Title

func (l *Lease) Pending(ctx context.Context, accountID string) ([]PendingEvent, error) {
	return l.PendingRetry(ctx, accountID, false)
}
func (l *Lease) PendingRetry(ctx context.Context, accountID string, retry bool) ([]PendingEvent, error) {
	rows, err := l.Conn.Query(ctx, `SELECT e.id::text,e.source_title_id,e.source_title,e.match_title,e.media_type_hint,e.season_number,e.episode_number,e.source_work_id FROM watch_events e JOIN streaming_accounts a ON a.id=e.streaming_account_id WHERE e.streaming_account_id=$1 AND e.title_id IS NULL AND (e.enrich_after<=now() OR ($2 AND e.enrichment_status IN ('not_found','ambiguous'))) AND NOT EXISTS(SELECT 1 FROM metadata_overrides o WHERE o.source_key=(`+catalogstore.SourceKey+`) AND o.provider='netflix') ORDER BY e.enrich_after,e.id LIMIT 500`, accountID, retry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []PendingEvent
	for rows.Next() {
		var p PendingEvent
		var sourceID *string
		err = rows.Scan(&p.ID, &sourceID, &p.Event.SourceTitle, &p.Event.MatchTitle, &p.Event.MediaType, &p.Event.SeasonNumber, &p.Event.EpisodeNumber, &p.Event.SourceWorkID)
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
	i, err := catalogstore.LockIdentity(ctx, tx, p.ID)
	if err != nil {
		return err
	}
	if applied, err := catalogstore.ApplyOverride(ctx, tx, i, p.ID); err != nil {
		return err
	} else if applied {
		return tx.Commit(ctx)
	}
	titleID, err := catalogstore.UpsertTitle(ctx, tx, m)
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

func (l *Lease) ApplyOverride(ctx context.Context, id string) (bool, error) {
	tx, err := l.Conn.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(context.Background())
	i, err := catalogstore.LockIdentity(ctx, tx, id)
	if err != nil {
		return false, err
	}
	applied, err := catalogstore.ApplyOverride(ctx, tx, i, id)
	if err != nil {
		return false, err
	}
	return applied, tx.Commit(ctx)
}

-- +goose Up
ALTER TABLE watch_events
    ADD COLUMN match_title text NOT NULL DEFAULT '',
    ADD COLUMN media_type_hint text NOT NULL DEFAULT '' CHECK (media_type_hint IN ('', 'movie', 'tv')),
    ADD COLUMN season_number integer CHECK (season_number >= 0),
    ADD COLUMN episode_number integer CHECK (episode_number > 0),
    ADD COLUMN enrichment_status text NOT NULL DEFAULT 'pending' CHECK (enrichment_status IN ('pending','matched','not_found','ambiguous')),
    ADD COLUMN enrich_after timestamptz NOT NULL DEFAULT now();
CREATE INDEX watch_events_enrich_due ON watch_events (streaming_account_id, enrich_after) WHERE title_id IS NULL;

-- +goose Down
DROP INDEX watch_events_enrich_due;
ALTER TABLE watch_events DROP COLUMN match_title, DROP COLUMN media_type_hint,
    DROP COLUMN season_number, DROP COLUMN episode_number,
    DROP COLUMN enrichment_status, DROP COLUMN enrich_after;

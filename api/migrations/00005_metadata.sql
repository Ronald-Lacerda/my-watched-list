-- +goose Up
ALTER TABLE titles ADD COLUMN overview_language text NOT NULL DEFAULT 'pt-BR';
CREATE TABLE metadata_overrides (
    source_key text PRIMARY KEY CHECK (btrim(source_key) <> ''),
    provider text NOT NULL CHECK (provider IN ('tmdb','netflix')),
    media_type text NOT NULL CHECK (media_type IN ('movie','tv')),
    title_id uuid REFERENCES titles(id),
    display_title text NOT NULL DEFAULT '',
    overview text NOT NULL DEFAULT '',
    overview_language text NOT NULL DEFAULT '',
    poster_url text NOT NULL DEFAULT '',
    updated_by uuid REFERENCES users(id),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((provider='tmdb' AND title_id IS NOT NULL AND poster_url='') OR
           (provider='netflix' AND display_title<>''))
);
CREATE INDEX watch_events_source_work ON watch_events (source_work_id) WHERE source_work_id<>'';

-- +goose Down
-- Recusar rollback destrutivo quando houver identificações confirmadas.
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM metadata_overrides) THEN
  RAISE EXCEPTION 'metadata_overrides contém identificações; preserve-as antes do rollback';
 END IF;
END $$;
-- +goose StatementEnd
DROP TABLE metadata_overrides;
DROP INDEX watch_events_source_work;
ALTER TABLE titles DROP COLUMN overview_language;

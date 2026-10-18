-- +goose Up
ALTER TABLE watch_events ADD COLUMN source_work_id text NOT NULL DEFAULT '';
CREATE INDEX watch_events_timeline ON watch_events (watched_on DESC, id DESC);

ALTER TABLE ratings ALTER COLUMN title_id DROP NOT NULL;
ALTER TABLE ratings ADD COLUMN source_key text CHECK (source_key IS NULL OR btrim(source_key) <> '');
ALTER TABLE ratings ADD CONSTRAINT ratings_identity CHECK (title_id IS NOT NULL OR source_key IS NOT NULL);
ALTER TABLE ratings ADD CONSTRAINT ratings_user_source UNIQUE (user_id, source_key);
CREATE INDEX ratings_source ON ratings (source_key) WHERE source_key IS NOT NULL;

CREATE TABLE auth_sessions (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL CHECK (expires_at > created_at)
);
CREATE INDEX auth_sessions_user ON auth_sessions (user_id);
CREATE INDEX auth_sessions_expiry ON auth_sessions (expires_at);
CREATE TABLE auth_invitations (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL CHECK (expires_at > created_at)
);
CREATE TABLE auth_attempts (
    identifier_hash bytea PRIMARY KEY CHECK (octet_length(identifier_hash) = 32),
    window_start timestamptz NOT NULL DEFAULT now(),
    attempts integer NOT NULL CHECK (attempts > 0)
);
CREATE INDEX auth_attempts_window ON auth_attempts (window_start);

-- +goose Down
-- Falha sem apagar notas se houver avaliações que dependem da identidade própria.
ALTER TABLE ratings ALTER COLUMN title_id SET NOT NULL;
DROP TABLE auth_attempts;
DROP TABLE auth_invitations;
DROP TABLE auth_sessions;
DROP INDEX ratings_source;
ALTER TABLE ratings DROP CONSTRAINT ratings_user_source, DROP CONSTRAINT ratings_identity, DROP COLUMN source_key;
DROP INDEX watch_events_timeline;
ALTER TABLE watch_events DROP COLUMN source_work_id;

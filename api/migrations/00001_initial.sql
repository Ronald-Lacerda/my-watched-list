-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL CHECK (btrim(name) <> ''),
    email text NOT NULL CHECK (btrim(email) <> ''),
    password_hash text NOT NULL CHECK (password_hash <> ''),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_unique ON users (lower(email));

-- Uma conexão representa um perfil atribuído a uma pessoa.
CREATE TABLE streaming_accounts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id),
    service text NOT NULL CHECK (btrim(service) <> ''),
    account_ref text NOT NULL CHECK (btrim(account_ref) <> ''),
    profile_id text NOT NULL CHECK (btrim(profile_id) <> ''),
    profile_name text NOT NULL CHECK (btrim(profile_name) <> ''),
    session_status text NOT NULL DEFAULT 'needs_login'
        CHECK (session_status IN ('active', 'needs_login', 'error', 'disabled')),
    session_ciphertext bytea,
    session_key_version integer,
    last_sync_at timestamptz,
    last_attempt_at timestamptz,
    last_error_code text,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, service, account_ref, profile_id),
    UNIQUE (id, user_id),
    CHECK ((session_ciphertext IS NULL AND session_key_version IS NULL) OR
           (session_ciphertext IS NOT NULL AND octet_length(session_ciphertext) > 0 AND session_key_version IS NOT NULL AND session_key_version > 0)),
    CHECK (session_status <> 'active' OR session_ciphertext IS NOT NULL)
);

CREATE TABLE titles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tmdb_id bigint NOT NULL CHECK (tmdb_id > 0),
    media_type text NOT NULL CHECK (media_type IN ('movie', 'tv')),
    title text NOT NULL CHECK (btrim(title) <> ''),
    original_title text,
    release_year integer CHECK (release_year BETWEEN 1800 AND 9999),
    overview text,
    poster_path text,
    genres text[] NOT NULL DEFAULT '{}',
    runtime_minutes integer CHECK (runtime_minutes > 0),
    enriched_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tmdb_id, media_type),
    UNIQUE (id, media_type)
);

CREATE TABLE episodes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title_id uuid NOT NULL,
    media_type text NOT NULL DEFAULT 'tv' CHECK (media_type = 'tv'),
    season_number integer NOT NULL CHECK (season_number >= 0),
    episode_number integer NOT NULL CHECK (episode_number > 0),
    title text,
    runtime_minutes integer CHECK (runtime_minutes > 0),
    FOREIGN KEY (title_id, media_type) REFERENCES titles(id, media_type),
    UNIQUE (title_id, season_number, episode_number),
    UNIQUE (id, title_id)
);

CREATE TABLE watch_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id),
    streaming_account_id uuid NOT NULL,
    source_event_key text NOT NULL CHECK (btrim(source_event_key) <> ''),
    source_title_id text,
    source_title text NOT NULL CHECK (btrim(source_title) <> ''),
    title_id uuid REFERENCES titles(id),
    episode_id uuid,
    watched_on date NOT NULL,
    watched_at timestamptz,
    origin text NOT NULL DEFAULT 'collector' CHECK (origin IN ('collector', 'assisted', 'manual')),
    collected_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (streaming_account_id, user_id) REFERENCES streaming_accounts(id, user_id),
    FOREIGN KEY (episode_id, title_id) REFERENCES episodes(id, title_id),
    CHECK (episode_id IS NULL OR title_id IS NOT NULL),
    UNIQUE (streaming_account_id, source_event_key)
);
CREATE INDEX watch_events_user_date ON watch_events (user_id, watched_on DESC, id);
CREATE INDEX watch_events_account_date ON watch_events (streaming_account_id, watched_on DESC);
CREATE INDEX watch_events_title ON watch_events (title_id);
CREATE INDEX watch_events_unmatched ON watch_events (collected_at) WHERE title_id IS NULL;

CREATE TABLE ratings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id),
    title_id uuid NOT NULL REFERENCES titles(id),
    score numeric NOT NULL CHECK (score >= 0 AND score <= 10 AND mod(score, 0.5) = 0),
    comment text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, title_id)
);
CREATE INDEX ratings_title ON ratings (title_id);

-- +goose Down
DROP TABLE ratings;
DROP TABLE watch_events;
DROP TABLE episodes;
DROP TABLE titles;
DROP TABLE streaming_accounts;
DROP TABLE users;

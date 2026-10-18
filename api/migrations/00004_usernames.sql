-- +goose Up
-- Contas anteriores recebem um identificador provisório único. O administrador
-- atribui o login desejado sem alterar senha, UUID, convites ou sessões.
ALTER TABLE users ADD COLUMN username text NOT NULL
    DEFAULT ('user_' || replace(gen_random_uuid()::text, '-', ''))
    CHECK (username ~ '^[a-z][a-z0-9_]{2,39}$');
CREATE UNIQUE INDEX users_username_unique ON users (username);

-- +goose Down
ALTER TABLE users DROP COLUMN username;

-- +goose Up
CREATE TABLE api_keys (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace   VARCHAR(255) NOT NULL DEFAULT 'default',
    description VARCHAR(255) NOT NULL DEFAULT '',
    key_hash    CHAR(64)     NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ,

    CONSTRAINT uq_api_keys_hash UNIQUE(key_hash)
);

CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);

-- +goose Down
DROP TABLE IF EXISTS api_keys;

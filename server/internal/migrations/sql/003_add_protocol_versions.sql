-- +goose Up

CREATE TABLE protocol_versions (
    id            UUID      PRIMARY KEY DEFAULT uuidv7(),
    service_id    UUID      NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    protocol_type SMALLINT  NOT NULL,
    version_number INT      NOT NULL,
    content_hash  CHAR(64)  NOT NULL,
    file_count    INT       NOT NULL DEFAULT 0,
    published_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_protocol_versions_service_type_version UNIQUE(service_id, protocol_type, version_number)
);

CREATE INDEX idx_protocol_versions_service_type ON protocol_versions(service_id, protocol_type, version_number DESC);

-- +goose Down

DROP TABLE IF EXISTS protocol_versions;

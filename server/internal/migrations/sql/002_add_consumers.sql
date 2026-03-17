-- +goose Up

-- Migrate existing tables to UUID v7 and remove created_at
ALTER TABLE services ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE services DROP COLUMN created_at;

ALTER TABLE protocols ALTER COLUMN id SET DEFAULT uuidv7();

-- New table
CREATE TABLE consumers (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    consumer_service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    server_service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    protocol_type SMALLINT NOT NULL,
    content_hash CHAR(64) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_consumers_consumer_server_type UNIQUE(consumer_service_id, server_service_id, protocol_type)
);
CREATE INDEX idx_consumers_server_type ON consumers(server_service_id, protocol_type);

-- +goose Down
DROP TABLE IF EXISTS consumers;
ALTER TABLE protocols ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE services ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE services ALTER COLUMN id SET DEFAULT gen_random_uuid();

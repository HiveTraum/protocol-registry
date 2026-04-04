-- +goose Up
ALTER TABLE protocols ADD COLUMN version VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE protocols DROP CONSTRAINT uq_protocols_service_type;
ALTER TABLE protocols ADD CONSTRAINT uq_protocols_service_type_version UNIQUE(service_id, protocol_type, version);

ALTER TABLE consumers ADD COLUMN version VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE consumers DROP CONSTRAINT uq_consumers_consumer_server_type;
ALTER TABLE consumers ADD CONSTRAINT uq_consumers_consumer_server_type_version UNIQUE(consumer_service_id, server_service_id, protocol_type, version);
DROP INDEX idx_consumers_server_type;
CREATE INDEX idx_consumers_server_type_version ON consumers(server_service_id, protocol_type, version);

-- +goose Down
DROP INDEX IF EXISTS idx_consumers_server_type_version;
CREATE INDEX idx_consumers_server_type ON consumers(server_service_id, protocol_type);
ALTER TABLE consumers DROP CONSTRAINT uq_consumers_consumer_server_type_version;
ALTER TABLE consumers ADD CONSTRAINT uq_consumers_consumer_server_type UNIQUE(consumer_service_id, server_service_id, protocol_type);
ALTER TABLE consumers DROP COLUMN version;

ALTER TABLE protocols DROP CONSTRAINT uq_protocols_service_type_version;
ALTER TABLE protocols ADD CONSTRAINT uq_protocols_service_type UNIQUE(service_id, protocol_type);
ALTER TABLE protocols DROP COLUMN version;

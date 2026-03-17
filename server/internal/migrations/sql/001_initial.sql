-- +goose Up
CREATE TABLE services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_services_name UNIQUE(name)
);

CREATE TABLE protocols (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    protocol_type SMALLINT NOT NULL,
    content_hash CHAR(64) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_protocols_service_type UNIQUE(service_id, protocol_type)
);

CREATE INDEX idx_protocols_service_id ON protocols(service_id);

-- +goose Down
DROP TABLE IF EXISTS protocols;
DROP TABLE IF EXISTS services;

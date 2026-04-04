package implementations

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/protocol_registry/internal/entities"
)

type ProtocolRepositoryPostgres struct {
	pool *pgxpool.Pool
}

func NewProtocolRepositoryPostgres(pool *pgxpool.Pool) *ProtocolRepositoryPostgres {
	return &ProtocolRepositoryPostgres{pool: pool}
}

func (r *ProtocolRepositoryPostgres) GetByServiceAndType(ctx context.Context, serviceID uuid.UUID, protocolType entities.ProtocolType, version string) (*entities.Protocol, error) {
	var p entities.Protocol
	err := r.pool.QueryRow(ctx,
		`SELECT id, service_id, protocol_type, version, content_hash, updated_at
		 FROM protocols
		 WHERE service_id = $1 AND protocol_type = $2 AND version = $3`,
		serviceID, protocolType, version,
	).Scan(&p.ID, &p.ServiceID, &p.Type, &p.Version, &p.ContentHash, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProtocolRepositoryPostgres) Upsert(ctx context.Context, serviceID uuid.UUID, protocolType entities.ProtocolType, version string, contentHash string) (*entities.Protocol, bool, error) {
	var p entities.Protocol
	var created bool

	err := r.pool.QueryRow(ctx,
		`INSERT INTO protocols (service_id, protocol_type, version, content_hash)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (service_id, protocol_type, version)
		 DO UPDATE SET content_hash = EXCLUDED.content_hash, updated_at = NOW()
		 RETURNING id, service_id, protocol_type, version, content_hash, updated_at, (xmax = 0) AS created`,
		serviceID, protocolType, version, contentHash,
	).Scan(&p.ID, &p.ServiceID, &p.Type, &p.Version, &p.ContentHash, &p.UpdatedAt, &created)
	if err != nil {
		return nil, false, err
	}
	return &p, created, nil
}

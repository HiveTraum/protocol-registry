package implementations

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/protocol_registry/internal/entities"
)

const defaultPageSize = 20
const maxPageSize = 100

type ProtocolRepositoryPostgres struct {
	pool *pgxpool.Pool
}

func NewProtocolRepositoryPostgres(pool *pgxpool.Pool) *ProtocolRepositoryPostgres {
	return &ProtocolRepositoryPostgres{pool: pool}
}

func (r *ProtocolRepositoryPostgres) GetByServiceAndType(ctx context.Context, serviceID uuid.UUID, protocolType entities.ProtocolType) (*entities.Protocol, error) {
	var p entities.Protocol
	err := r.pool.QueryRow(ctx,
		`SELECT id, service_id, protocol_type, content_hash, updated_at
		 FROM protocols
		 WHERE service_id = $1 AND protocol_type = $2`,
		serviceID, protocolType,
	).Scan(&p.ID, &p.ServiceID, &p.Type, &p.ContentHash, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProtocolRepositoryPostgres) Upsert(ctx context.Context, serviceID uuid.UUID, protocolType entities.ProtocolType, contentHash string) (*entities.Protocol, bool, error) {
	var p entities.Protocol
	var created bool

	err := r.pool.QueryRow(ctx,
		`INSERT INTO protocols (service_id, protocol_type, content_hash)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (service_id, protocol_type)
		 DO UPDATE SET content_hash = EXCLUDED.content_hash, updated_at = NOW()
		 RETURNING id, service_id, protocol_type, content_hash, updated_at, (xmax = 0) AS created`,
		serviceID, protocolType, contentHash,
	).Scan(&p.ID, &p.ServiceID, &p.Type, &p.ContentHash, &p.UpdatedAt, &created)
	if err != nil {
		return nil, false, err
	}
	return &p, created, nil
}

func (r *ProtocolRepositoryPostgres) InsertVersion(ctx context.Context, serviceID uuid.UUID, protocolType entities.ProtocolType, contentHash string, fileCount int) (*entities.ProtocolVersion, error) {
	var v entities.ProtocolVersion
	err := r.pool.QueryRow(ctx,
		`INSERT INTO protocol_versions (service_id, protocol_type, version_number, content_hash, file_count)
		 VALUES ($1, $2,
		     COALESCE((SELECT MAX(version_number) FROM protocol_versions WHERE service_id = $1 AND protocol_type = $2), 0) + 1,
		     $3, $4)
		 RETURNING id, service_id, protocol_type, version_number, content_hash, file_count, published_at`,
		serviceID, protocolType, contentHash, fileCount,
	).Scan(&v.ID, &v.ServiceID, &v.Type, &v.VersionNumber, &v.ContentHash, &v.FileCount, &v.PublishedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *ProtocolRepositoryPostgres) ListVersionsByServiceAndType(ctx context.Context, serviceID uuid.UUID, protocolType entities.ProtocolType, offset, limit int) ([]entities.ProtocolVersion, int, error) {
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM protocol_versions WHERE service_id = $1 AND protocol_type = $2`,
		serviceID, protocolType,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, service_id, protocol_type, version_number, content_hash, file_count, published_at
		 FROM protocol_versions
		 WHERE service_id = $1 AND protocol_type = $2
		 ORDER BY version_number DESC
		 LIMIT $3 OFFSET $4`,
		serviceID, protocolType, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var versions []entities.ProtocolVersion
	for rows.Next() {
		var v entities.ProtocolVersion
		if err := rows.Scan(&v.ID, &v.ServiceID, &v.Type, &v.VersionNumber, &v.ContentHash, &v.FileCount, &v.PublishedAt); err != nil {
			return nil, 0, err
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return versions, total, nil
}

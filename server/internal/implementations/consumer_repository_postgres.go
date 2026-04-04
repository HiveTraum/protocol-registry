package implementations

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/protocol_registry/internal/entities"
)

type ConsumerRepositoryPostgres struct {
	pool *pgxpool.Pool
}

func NewConsumerRepositoryPostgres(pool *pgxpool.Pool) *ConsumerRepositoryPostgres {
	return &ConsumerRepositoryPostgres{pool: pool}
}

func (r *ConsumerRepositoryPostgres) Upsert(ctx context.Context, consumerServiceID, serverServiceID uuid.UUID, protocolType entities.ProtocolType, version string, contentHash string) (*entities.Consumer, bool, error) {
	var c entities.Consumer
	var created bool

	err := r.pool.QueryRow(ctx,
		`INSERT INTO consumers (consumer_service_id, server_service_id, protocol_type, version, content_hash)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (consumer_service_id, server_service_id, protocol_type, version)
		 DO UPDATE SET content_hash = EXCLUDED.content_hash, updated_at = NOW()
		 RETURNING id, consumer_service_id, server_service_id, protocol_type, version, content_hash, updated_at, (xmax = 0) AS created`,
		consumerServiceID, serverServiceID, protocolType, version, contentHash,
	).Scan(&c.ID, &c.ConsumerServiceID, &c.ServerServiceID, &c.ProtocolType, &c.Version, &c.ContentHash, &c.UpdatedAt, &created)
	if err != nil {
		return nil, false, err
	}
	return &c, created, nil
}

func (r *ConsumerRepositoryPostgres) Delete(ctx context.Context, consumerServiceID, serverServiceID uuid.UUID, protocolType entities.ProtocolType, version string) error {
	result, err := r.pool.Exec(ctx,
		`DELETE FROM consumers
		 WHERE consumer_service_id = $1 AND server_service_id = $2 AND protocol_type = $3 AND version = $4`,
		consumerServiceID, serverServiceID, protocolType, version,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ConsumerRepositoryPostgres) ListByServerAndType(ctx context.Context, serverServiceID uuid.UUID, protocolType entities.ProtocolType, version string) ([]entities.Consumer, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, consumer_service_id, server_service_id, protocol_type, version, content_hash, updated_at
		 FROM consumers
		 WHERE server_service_id = $1 AND protocol_type = $2 AND version = $3`,
		serverServiceID, protocolType, version,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consumers []entities.Consumer
	for rows.Next() {
		var c entities.Consumer
		if err := rows.Scan(&c.ID, &c.ConsumerServiceID, &c.ServerServiceID, &c.ProtocolType, &c.Version, &c.ContentHash, &c.UpdatedAt); err != nil {
			return nil, err
		}
		consumers = append(consumers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return consumers, nil
}

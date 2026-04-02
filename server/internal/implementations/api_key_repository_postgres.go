package implementations

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/protocol_registry/internal/entities"
)

type APIKeyRepositoryPostgres struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepositoryPostgres(pool *pgxpool.Pool) *APIKeyRepositoryPostgres {
	return &APIKeyRepositoryPostgres{pool: pool}
}

func (r *APIKeyRepositoryPostgres) Create(ctx context.Context, namespace, description, keyHash string) (*entities.APIKey, error) {
	var k entities.APIKey
	err := r.pool.QueryRow(ctx,
		`INSERT INTO api_keys (namespace, description, key_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, namespace, description, key_hash, created_at, revoked_at`,
		namespace, description, keyHash,
	).Scan(&k.ID, &k.Namespace, &k.Description, &k.KeyHash, &k.CreatedAt, &k.RevokedAt)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *APIKeyRepositoryPostgres) FindByHash(ctx context.Context, keyHash string) (*entities.APIKey, error) {
	var k entities.APIKey
	err := r.pool.QueryRow(ctx,
		`SELECT id, namespace, description, key_hash, created_at, revoked_at
		 FROM api_keys
		 WHERE key_hash = $1`,
		keyHash,
	).Scan(&k.ID, &k.Namespace, &k.Description, &k.KeyHash, &k.CreatedAt, &k.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

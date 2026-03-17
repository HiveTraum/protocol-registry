package implementations

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/protocol_registry/internal/entities"
)

type ServiceRepositoryPostgres struct {
	pool *pgxpool.Pool
}

func NewServiceRepositoryPostgres(pool *pgxpool.Pool) *ServiceRepositoryPostgres {
	return &ServiceRepositoryPostgres{pool: pool}
}

func (r *ServiceRepositoryPostgres) GetByName(ctx context.Context, name string) (*entities.Service, error) {
	var s entities.Service
	err := r.pool.QueryRow(ctx,
		`SELECT id, name FROM services WHERE name = $1`,
		name,
	).Scan(&s.ID, &s.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ServiceRepositoryPostgres) GetByID(ctx context.Context, id uuid.UUID) (*entities.Service, error) {
	var s entities.Service
	err := r.pool.QueryRow(ctx,
		`SELECT id, name FROM services WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ServiceRepositoryPostgres) Create(ctx context.Context, name string) (*entities.Service, error) {
	var s entities.Service
	err := r.pool.QueryRow(ctx,
		`INSERT INTO services (name) VALUES ($1) RETURNING id, name`,
		name,
	).Scan(&s.ID, &s.Name)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ServiceRepositoryPostgres) ListAll(ctx context.Context) ([]entities.Service, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name FROM services ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []entities.Service
	for rows.Next() {
		var s entities.Service
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, rows.Err()
}

func (r *ServiceRepositoryPostgres) GetOrCreate(ctx context.Context, name string) (*entities.Service, bool, error) {
	var s entities.Service
	var created bool

	err := r.pool.QueryRow(ctx,
		`INSERT INTO services (name) VALUES ($1)
		 ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id, name, (xmax = 0) AS created`,
		name,
	).Scan(&s.ID, &s.Name, &created)
	if err != nil {
		return nil, false, err
	}
	return &s, created, nil
}

package entities

import (
	"context"
	"time"
)

type APIKey struct {
	ID          string
	Namespace   string
	Description string
	KeyHash     string
	CreatedAt   time.Time
	RevokedAt   *time.Time
}

type APIKeyRepository interface {
	Create(ctx context.Context, namespace, description, keyHash string) (*APIKey, error)
	FindByHash(ctx context.Context, keyHash string) (*APIKey, error)
}

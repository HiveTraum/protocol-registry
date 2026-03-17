package unregister_consumer

import (
	"context"

	"github.com/google/uuid"
	"github.com/user/protocol_registry/internal/entities"
)

type ServiceRepository interface {
	GetByName(ctx context.Context, name string) (*entities.Service, error)
}

type ConsumerRepository interface {
	Delete(ctx context.Context, consumerServiceID, serverServiceID uuid.UUID, protocolType entities.ProtocolType) error
}

type ConsumerStorage interface {
	DeleteConsumer(ctx context.Context, consumerName, serverName string, protocolType entities.ProtocolType) error
}

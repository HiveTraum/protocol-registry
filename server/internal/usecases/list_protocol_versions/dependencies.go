package list_protocol_versions

import (
	"context"

	"github.com/google/uuid"
	"github.com/user/protocol_registry/internal/entities"
)

type ServiceRepository interface {
	GetByName(ctx context.Context, name string) (*entities.Service, error)
}

type ProtocolRepository interface {
	GetByServiceAndType(ctx context.Context, serviceID uuid.UUID, protocolType entities.ProtocolType) (*entities.Protocol, error)
	ListVersionsByServiceAndType(ctx context.Context, serviceID uuid.UUID, protocolType entities.ProtocolType, offset, limit int) ([]entities.ProtocolVersion, int, error)
}

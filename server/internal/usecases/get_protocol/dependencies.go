package get_protocol

import (
	"context"

	"github.com/google/uuid"
	"github.com/user/protocol_registry/internal/entities"
)

type ServiceRepository interface {
	GetByName(ctx context.Context, name string) (*entities.Service, error)
}

type ProtocolRepository interface {
	GetByServiceAndType(ctx context.Context, serviceID uuid.UUID, protocolType entities.ProtocolType, version string) (*entities.Protocol, error)
}

type Storage interface {
	DownloadFileSet(ctx context.Context, serviceName string, version string, protocolType entities.ProtocolType) (entities.ProtoFileSet, error)
}

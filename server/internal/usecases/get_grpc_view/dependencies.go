package get_grpc_view

import (
	"context"

	"github.com/google/uuid"
	"github.com/user/protocol_registry/internal/entities"
)

type ServiceRepository interface {
	GetByName(ctx context.Context, name string) (*entities.Service, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Service, error)
}

type ProtocolRepository interface {
	GetByServiceAndType(ctx context.Context, serviceID uuid.UUID, protocolType entities.ProtocolType, version string) (*entities.Protocol, error)
}

type Storage interface {
	DownloadFileSet(ctx context.Context, serviceName string, version string, protocolType entities.ProtocolType) (entities.ProtoFileSet, error)
}

type ConsumerRepository interface {
	ListByServerAndType(ctx context.Context, serverServiceID uuid.UUID, protocolType entities.ProtocolType, version string) ([]entities.Consumer, error)
}

type ConsumerStorage interface {
	DownloadConsumerFileSet(ctx context.Context, consumerName, serverName string, version string, protocolType entities.ProtocolType) (entities.ProtoFileSet, error)
}

type ProtoInspector interface {
	Inspect(ctx context.Context, fileSet entities.ProtoFileSet) ([]entities.ProtoServiceView, error)
}

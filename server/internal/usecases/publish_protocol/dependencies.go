package publish_protocol

import (
	"context"

	"github.com/google/uuid"
	"github.com/user/protocol_registry/internal/entities"
)

type ServiceRepository interface {
	GetOrCreate(ctx context.Context, name string) (*entities.Service, bool, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Service, error)
}

type ProtocolRepository interface {
	GetByServiceAndType(ctx context.Context, serviceID uuid.UUID, protocolType entities.ProtocolType) (*entities.Protocol, error)
	Upsert(ctx context.Context, serviceID uuid.UUID, protocolType entities.ProtocolType, contentHash string) (*entities.Protocol, bool, error)
}

type Storage interface {
	UploadFileSet(ctx context.Context, serviceName string, protocolType entities.ProtocolType, fileSet entities.ProtoFileSet) error
}

type ConsumerRepository interface {
	ListByServerAndType(ctx context.Context, serverServiceID uuid.UUID, protocolType entities.ProtocolType) ([]entities.Consumer, error)
}

type ConsumerStorage interface {
	DownloadConsumerFileSet(ctx context.Context, consumerName, serverName string, protocolType entities.ProtocolType) (entities.ProtoFileSet, error)
}

type SyntaxValidator interface {
	Validate(fileSet entities.ProtoFileSet) error
}

type BreakingChangesValidator interface {
	Validate(ctx context.Context, previous, current entities.ProtoFileSet) error
}

package register_consumer

import (
	"context"

	"github.com/google/uuid"
	"github.com/user/protocol_registry/internal/entities"
)

type ServiceRepository interface {
	GetByName(ctx context.Context, name string) (*entities.Service, error)
	GetOrCreate(ctx context.Context, name string) (*entities.Service, bool, error)
}

type ProtocolRepository interface {
	GetByServiceAndType(ctx context.Context, serviceID uuid.UUID, protocolType entities.ProtocolType) (*entities.Protocol, error)
}

type ConsumerRepository interface {
	Upsert(ctx context.Context, consumerServiceID, serverServiceID uuid.UUID, protocolType entities.ProtocolType, contentHash string) (*entities.Consumer, bool, error)
}

type Storage interface {
	DownloadFileSet(ctx context.Context, serviceName string, protocolType entities.ProtocolType) (entities.ProtoFileSet, error)
}

type ConsumerStorage interface {
	UploadConsumerFileSet(ctx context.Context, consumerName, serverName string, protocolType entities.ProtocolType, fileSet entities.ProtoFileSet) error
}

type SyntaxValidator interface {
	Validate(protocolType entities.ProtocolType, fileSet entities.ProtoFileSet) error
}

type BreakingChangesValidator interface {
	Validate(ctx context.Context, protocolType entities.ProtocolType, previous, current entities.ProtoFileSet) error
}

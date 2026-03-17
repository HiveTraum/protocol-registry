package publish_protocol

import (
	"context"

	"github.com/user/protocol-registry-cli/internal/entities"
)

type RegistryClient interface {
	PublishProtocol(ctx context.Context, serviceName string, protocolType entities.ProtocolType, files []ProtoFile, entryPoint string) (*Output, error)
}

type FileReader interface {
	ReadProtoFiles(dir string) ([]ProtoFile, error)
}

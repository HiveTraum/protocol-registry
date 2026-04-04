package register_consumer

import (
	"context"

	"github.com/user/protocol-registry-cli/internal/entities"
)

type RegistryClient interface {
	RegisterConsumer(ctx context.Context, consumerName, serverName string, protocolType entities.ProtocolType, files []ProtoFile, entryPoint string, serverVersions []string) (*Output, error)
}

type FileReader interface {
	ReadProtoFiles(dir string) ([]ProtoFile, error)
}

package get_protocol

import (
	"context"

	"github.com/user/protocol-registry-cli/internal/entities"
)

type RegistryClient interface {
	GetProtocol(ctx context.Context, serviceName string, protocolType entities.ProtocolType) (*Output, error)
}

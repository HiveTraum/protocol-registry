package unregister_consumer

import (
	"context"

	"github.com/user/protocol-registry-cli/internal/entities"
)

type RegistryClient interface {
	UnregisterConsumer(ctx context.Context, consumerName, serverName string, protocolType entities.ProtocolType) error
}

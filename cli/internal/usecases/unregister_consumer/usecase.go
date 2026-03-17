package unregister_consumer

import (
	"context"
	"fmt"

	"github.com/user/protocol-registry-cli/internal/entities"
)

type UseCase struct {
	registryClient RegistryClient
}

func New(registryClient RegistryClient) *UseCase {
	return &UseCase{
		registryClient: registryClient,
	}
}

func (uc *UseCase) Execute(ctx context.Context, input Input) error {
	protocolType := entities.ParseProtocolType(input.ProtocolType)

	if err := uc.registryClient.UnregisterConsumer(ctx, input.ConsumerName, input.ServerName, protocolType); err != nil {
		return fmt.Errorf("unregister: %w", err)
	}

	return nil
}

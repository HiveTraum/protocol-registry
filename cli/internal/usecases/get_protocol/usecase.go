package get_protocol

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

func (uc *UseCase) Execute(ctx context.Context, input Input) (*Output, error) {
	protocolType := entities.ParseProtocolType(input.ProtocolType)

	output, err := uc.registryClient.GetProtocol(ctx, input.ServiceName, protocolType)
	if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}

	return output, nil
}

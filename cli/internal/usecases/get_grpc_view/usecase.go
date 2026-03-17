package get_grpc_view

import (
	"context"
	"fmt"
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
	output, err := uc.registryClient.GetGrpcView(ctx, input.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("grpc-view: %w", err)
	}

	return output, nil
}

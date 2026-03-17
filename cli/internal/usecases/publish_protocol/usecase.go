package publish_protocol

import (
	"context"
	"fmt"

	"github.com/user/protocol-registry-cli/internal/entities"
)

type UseCase struct {
	registryClient RegistryClient
	fileReader     FileReader
}

func New(registryClient RegistryClient, fileReader FileReader) *UseCase {
	return &UseCase{
		registryClient: registryClient,
		fileReader:     fileReader,
	}
}

func (uc *UseCase) Execute(ctx context.Context, input Input) (*Output, error) {
	protocolType := entities.ParseProtocolType(input.ProtocolType)

	files, err := uc.fileReader.ReadProtoFiles(input.ProtoDir)
	if err != nil {
		return nil, fmt.Errorf("collect proto files: %w", err)
	}

	output, err := uc.registryClient.PublishProtocol(ctx, input.ServiceName, protocolType, files, input.EntryPoint)
	if err != nil {
		return nil, fmt.Errorf("publish: %w", err)
	}

	return output, nil
}

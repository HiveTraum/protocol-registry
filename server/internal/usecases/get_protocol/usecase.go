package get_protocol

import (
	"context"
	"fmt"

	"github.com/user/protocol_registry/internal/entities"
)

type UseCase struct {
	serviceRepo  ServiceRepository
	protocolRepo ProtocolRepository
	storage      Storage
}

func New(
	serviceRepo ServiceRepository,
	protocolRepo ProtocolRepository,
	storage Storage,
) *UseCase {
	return &UseCase{
		serviceRepo:  serviceRepo,
		protocolRepo: protocolRepo,
		storage:      storage,
	}
}

func (uc *UseCase) Execute(ctx context.Context, input Input) (*Output, error) {
	svc, err := uc.serviceRepo.GetByName(ctx, input.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("get service: %w", err)
	}
	if svc == nil {
		return nil, entities.NewServiceNotFoundError(input.ServiceName)
	}

	existing, err := uc.protocolRepo.GetByServiceAndType(ctx, svc.ID, input.ProtocolType)
	if err != nil {
		return nil, fmt.Errorf("get protocol: %w", err)
	}
	if existing == nil {
		return nil, entities.NewProtocolNotFoundError(input.ServiceName)
	}

	fileSet, err := uc.storage.DownloadFileSet(ctx, input.ServiceName, input.ProtocolType)
	if err != nil {
		return nil, fmt.Errorf("download protocol: %w", err)
	}

	return &Output{
		ServiceName:  input.ServiceName,
		ProtocolType: input.ProtocolType,
		FileSet:      fileSet,
	}, nil
}

package list_protocol_versions

import (
	"context"
	"fmt"

	"github.com/user/protocol_registry/internal/entities"
)

type UseCase struct {
	serviceRepo  ServiceRepository
	protocolRepo ProtocolRepository
}

func New(serviceRepo ServiceRepository, protocolRepo ProtocolRepository) *UseCase {
	return &UseCase{
		serviceRepo:  serviceRepo,
		protocolRepo: protocolRepo,
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

	versions, total, err := uc.protocolRepo.ListVersionsByServiceAndType(ctx, svc.ID, input.ProtocolType, input.Offset, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("list protocol versions: %w", err)
	}

	out := &Output{
		ServiceName:  input.ServiceName,
		ProtocolType: input.ProtocolType,
		Total:        total,
		Versions:     make([]VersionInfo, len(versions)),
	}
	for i, v := range versions {
		out.Versions[i] = VersionInfo{
			VersionNumber: v.VersionNumber,
			ContentHash:   v.ContentHash,
			FileCount:     v.FileCount,
			PublishedAt:   v.PublishedAt,
		}
	}
	return out, nil
}

package register_consumer

import (
	"context"
	"fmt"

	"github.com/user/protocol_registry/internal/entities"
)

type UseCase struct {
	serviceRepo              ServiceRepository
	protocolRepo             ProtocolRepository
	consumerRepo             ConsumerRepository
	storage                  Storage
	consumerStorage          ConsumerStorage
	syntaxValidator          SyntaxValidator
	breakingChangesValidator BreakingChangesValidator
}

func New(
	serviceRepo ServiceRepository,
	protocolRepo ProtocolRepository,
	consumerRepo ConsumerRepository,
	storage Storage,
	consumerStorage ConsumerStorage,
	syntaxValidator SyntaxValidator,
	breakingChangesValidator BreakingChangesValidator,
) *UseCase {
	return &UseCase{
		serviceRepo:              serviceRepo,
		protocolRepo:             protocolRepo,
		consumerRepo:             consumerRepo,
		storage:                  storage,
		consumerStorage:          consumerStorage,
		syntaxValidator:          syntaxValidator,
		breakingChangesValidator: breakingChangesValidator,
	}
}

func (uc *UseCase) Execute(ctx context.Context, input Input) (*Output, error) {
	if err := uc.syntaxValidator.Validate(input.FileSet); err != nil {
		return nil, fmt.Errorf("syntax validation failed: %w", err)
	}

	versions := input.ServerVersions
	if len(versions) == 0 {
		versions = []string{"default"}
	}

	serverSvc, err := uc.serviceRepo.GetByName(ctx, input.ServerName)
	if err != nil {
		return nil, fmt.Errorf("get server service: %w", err)
	}
	if serverSvc == nil {
		return nil, entities.NewServiceNotFoundError(input.ServerName)
	}

	consumerSvc, _, err := uc.serviceRepo.GetOrCreate(ctx, input.ConsumerName)
	if err != nil {
		return nil, fmt.Errorf("get or create consumer service: %w", err)
	}

	contentHash := input.FileSet.ContentHash()
	anyNew := false

	for _, version := range versions {
		existing, err := uc.protocolRepo.GetByServiceAndType(ctx, serverSvc.ID, input.ProtocolType, version)
		if err != nil {
			return nil, fmt.Errorf("get server protocol (version %s): %w", version, err)
		}
		if existing == nil {
			return nil, entities.NewProtocolNotFoundError(input.ServerName)
		}

		serverFileSet, err := uc.storage.DownloadFileSet(ctx, input.ServerName, version, input.ProtocolType)
		if err != nil {
			return nil, fmt.Errorf("download server protocol (version %s): %w", version, err)
		}

		if err := uc.breakingChangesValidator.Validate(ctx, input.FileSet, serverFileSet); err != nil {
			return nil, fmt.Errorf("consumer proto is not a subset of server proto (version %s): %w", version, err)
		}

		if err := uc.consumerStorage.UploadConsumerFileSet(ctx, input.ConsumerName, input.ServerName, version, input.ProtocolType, input.FileSet); err != nil {
			return nil, fmt.Errorf("upload consumer proto (version %s): %w", version, err)
		}

		_, isNew, err := uc.consumerRepo.Upsert(ctx, consumerSvc.ID, serverSvc.ID, input.ProtocolType, version, contentHash)
		if err != nil {
			return nil, fmt.Errorf("upsert consumer (version %s): %w", version, err)
		}
		if isNew {
			anyNew = true
		}
	}

	return &Output{
		ConsumerName: input.ConsumerName,
		ServerName:   input.ServerName,
		IsNew:        anyNew,
	}, nil
}

package publish_protocol

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/user/protocol_registry/internal/entities"
)

type UseCase struct {
	serviceRepo              ServiceRepository
	protocolRepo             ProtocolRepository
	storage                  Storage
	consumerRepo             ConsumerRepository
	consumerStorage          ConsumerStorage
	syntaxValidator          SyntaxValidator
	breakingChangesValidator BreakingChangesValidator
}

func New(
	serviceRepo ServiceRepository,
	protocolRepo ProtocolRepository,
	storage Storage,
	consumerRepo ConsumerRepository,
	consumerStorage ConsumerStorage,
	syntaxValidator SyntaxValidator,
	breakingChangesValidator BreakingChangesValidator,
) *UseCase {
	return &UseCase{
		serviceRepo:              serviceRepo,
		protocolRepo:             protocolRepo,
		storage:                  storage,
		consumerRepo:             consumerRepo,
		consumerStorage:          consumerStorage,
		syntaxValidator:          syntaxValidator,
		breakingChangesValidator: breakingChangesValidator,
	}
}

func (uc *UseCase) Execute(ctx context.Context, input Input) (*Output, error) {
	if err := uc.syntaxValidator.Validate(input.FileSet); err != nil {
		return nil, fmt.Errorf("syntax validation failed: %w", err)
	}

	versions := input.Versions
	if len(versions) == 0 {
		versions = []string{"default"}
	}

	contentHash := input.FileSet.ContentHash()

	svc, _, err := uc.serviceRepo.GetOrCreate(ctx, input.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("get or create service: %w", err)
	}

	anyNew := false
	for _, version := range versions {
		existing, err := uc.protocolRepo.GetByServiceAndType(ctx, svc.ID, input.ProtocolType, version)
		if err != nil {
			return nil, fmt.Errorf("get existing protocol (version %s): %w", version, err)
		}

		if existing != nil && existing.ContentHash == contentHash {
			continue
		}

		if err := uc.validateAgainstConsumers(ctx, svc.ID, input.ServiceName, input.ProtocolType, version, input.FileSet); err != nil {
			return nil, err
		}

		if err := uc.storage.UploadFileSet(ctx, input.ServiceName, version, input.ProtocolType, input.FileSet); err != nil {
			return nil, fmt.Errorf("upload protocol (version %s): %w", version, err)
		}

		_, isNew, err := uc.protocolRepo.Upsert(ctx, svc.ID, input.ProtocolType, version, contentHash)
		if err != nil {
			return nil, fmt.Errorf("upsert protocol (version %s): %w", version, err)
		}
		if isNew {
			anyNew = true
		}
	}

	return &Output{
		ServiceName: input.ServiceName,
		IsNew:       anyNew,
	}, nil
}

func (uc *UseCase) validateAgainstConsumers(ctx context.Context, serverServiceID uuid.UUID, serverName string, protocolType entities.ProtocolType, version string, newFileSet entities.ProtoFileSet) error {
	consumers, err := uc.consumerRepo.ListByServerAndType(ctx, serverServiceID, protocolType, version)
	if err != nil {
		return fmt.Errorf("list consumers: %w", err)
	}

	if len(consumers) == 0 {
		return nil
	}

	var violations []entities.ConsumerBreakingChange

	for _, consumer := range consumers {
		consumerSvc, err := uc.serviceRepo.GetByID(ctx, consumer.ConsumerServiceID)
		if err != nil {
			return fmt.Errorf("get consumer service: %w", err)
		}

		consumerFileSet, err := uc.consumerStorage.DownloadConsumerFileSet(ctx, consumerSvc.Name, serverName, version, protocolType)
		if err != nil {
			return fmt.Errorf("download consumer proto for %q: %w", consumerSvc.Name, err)
		}

		if err := uc.breakingChangesValidator.Validate(ctx, consumerFileSet, newFileSet); err != nil {
			var domainErr *entities.DomainError
			if errors.As(err, &domainErr) && domainErr.Code == entities.ErrorCodeBreakingChanges {
				violations = append(violations, entities.ConsumerBreakingChange{
					ConsumerName: consumerSvc.Name,
					Changes:      domainErr.BreakingChanges(),
				})
			} else {
				return fmt.Errorf("validate against consumer %q: %w", consumerSvc.Name, err)
			}
		}
	}

	if len(violations) > 0 {
		return entities.NewConsumerBreakingChangesError(violations)
	}

	return nil
}

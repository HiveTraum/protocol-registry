package validate_protocol

import (
	"context"
	"errors"
	"fmt"

	"github.com/user/protocol_registry/internal/entities"
)

type UseCase struct {
	serviceRepo              ServiceRepository
	consumerRepo             ConsumerRepository
	consumerStorage          ConsumerStorage
	syntaxValidator          SyntaxValidator
	breakingChangesValidator BreakingChangesValidator
}

func New(
	serviceRepo ServiceRepository,
	consumerRepo ConsumerRepository,
	consumerStorage ConsumerStorage,
	syntaxValidator SyntaxValidator,
	breakingChangesValidator BreakingChangesValidator,
) *UseCase {
	return &UseCase{
		serviceRepo:              serviceRepo,
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

	versions := input.AgainstVersions
	if len(versions) == 0 {
		versions = []string{"default"}
	}

	svc, err := uc.serviceRepo.GetByName(ctx, input.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("get service: %w", err)
	}
	if svc == nil {
		return nil, entities.NewServiceNotFoundError(input.ServiceName)
	}

	var allViolations []VersionViolation

	for _, version := range versions {
		consumers, err := uc.consumerRepo.ListByServerAndType(ctx, svc.ID, input.ProtocolType, version)
		if err != nil {
			return nil, fmt.Errorf("list consumers (version %s): %w", version, err)
		}

		var consumerViolations []entities.ConsumerBreakingChange

		for _, consumer := range consumers {
			consumerSvc, err := uc.serviceRepo.GetByID(ctx, consumer.ConsumerServiceID)
			if err != nil {
				return nil, fmt.Errorf("get consumer service: %w", err)
			}

			consumerFileSet, err := uc.consumerStorage.DownloadConsumerFileSet(ctx, consumerSvc.Name, input.ServiceName, version, input.ProtocolType)
			if err != nil {
				return nil, fmt.Errorf("download consumer proto for %q: %w", consumerSvc.Name, err)
			}

			if err := uc.breakingChangesValidator.Validate(ctx, consumerFileSet, input.FileSet); err != nil {
				var domainErr *entities.DomainError
				if errors.As(err, &domainErr) && domainErr.Code == entities.ErrorCodeBreakingChanges {
					consumerViolations = append(consumerViolations, entities.ConsumerBreakingChange{
						ConsumerName: consumerSvc.Name,
						Changes:      domainErr.BreakingChanges(),
					})
				} else {
					return nil, fmt.Errorf("validate against consumer %q: %w", consumerSvc.Name, err)
				}
			}
		}

		if len(consumerViolations) > 0 {
			allViolations = append(allViolations, VersionViolation{
				Version:   version,
				Consumers: consumerViolations,
			})
		}
	}

	return &Output{
		Valid:      len(allViolations) == 0,
		Violations: allViolations,
	}, nil
}

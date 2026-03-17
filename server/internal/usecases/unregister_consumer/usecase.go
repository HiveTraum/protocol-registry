package unregister_consumer

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/user/protocol_registry/internal/entities"
)

type UseCase struct {
	serviceRepo     ServiceRepository
	consumerRepo    ConsumerRepository
	consumerStorage ConsumerStorage
}

func New(
	serviceRepo ServiceRepository,
	consumerRepo ConsumerRepository,
	consumerStorage ConsumerStorage,
) *UseCase {
	return &UseCase{
		serviceRepo:     serviceRepo,
		consumerRepo:    consumerRepo,
		consumerStorage: consumerStorage,
	}
}

func (uc *UseCase) Execute(ctx context.Context, input Input) error {
	consumerSvc, err := uc.serviceRepo.GetByName(ctx, input.ConsumerName)
	if err != nil {
		return fmt.Errorf("get consumer service: %w", err)
	}
	if consumerSvc == nil {
		return entities.NewConsumerNotFoundError(input.ConsumerName, input.ServerName)
	}

	serverSvc, err := uc.serviceRepo.GetByName(ctx, input.ServerName)
	if err != nil {
		return fmt.Errorf("get server service: %w", err)
	}
	if serverSvc == nil {
		return entities.NewServiceNotFoundError(input.ServerName)
	}

	if err := uc.consumerRepo.Delete(ctx, consumerSvc.ID, serverSvc.ID, input.ProtocolType); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.NewConsumerNotFoundError(input.ConsumerName, input.ServerName)
		}
		return fmt.Errorf("delete consumer: %w", err)
	}

	if err := uc.consumerStorage.DeleteConsumer(ctx, input.ConsumerName, input.ServerName, input.ProtocolType); err != nil {
		return fmt.Errorf("delete consumer proto: %w", err)
	}

	return nil
}

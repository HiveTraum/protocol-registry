package list_services

import (
	"context"
	"fmt"
)

type UseCase struct {
	serviceRepo ServiceRepository
}

func New(serviceRepo ServiceRepository) *UseCase {
	return &UseCase{serviceRepo: serviceRepo}
}

func (uc *UseCase) Execute(ctx context.Context) (*Output, error) {
	services, err := uc.serviceRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}

	out := &Output{
		Services: make([]Service, len(services)),
	}
	for i, s := range services {
		out.Services[i] = Service{
			ID:   s.ID.String(),
			Name: s.Name,
		}
	}
	return out, nil
}

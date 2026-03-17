package list_services

import (
	"context"

	"github.com/user/protocol_registry/internal/entities"
)

type ServiceRepository interface {
	ListAll(ctx context.Context) ([]entities.Service, error)
}

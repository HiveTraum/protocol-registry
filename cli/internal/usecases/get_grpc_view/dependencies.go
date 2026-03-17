package get_grpc_view

import "context"

type RegistryClient interface {
	GetGrpcView(ctx context.Context, serviceName string) (*Output, error)
}

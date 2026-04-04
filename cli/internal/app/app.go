package app

import (
	"fmt"

	"github.com/user/protocol-registry-cli/internal/config"
	"github.com/user/protocol-registry-cli/internal/implementations"
	"github.com/user/protocol-registry-cli/internal/usecases/get_grpc_view"
	"github.com/user/protocol-registry-cli/internal/usecases/get_protocol"
	"github.com/user/protocol-registry-cli/internal/usecases/publish_protocol"
	"github.com/user/protocol-registry-cli/internal/usecases/register_consumer"
	"github.com/user/protocol-registry-cli/internal/usecases/unregister_consumer"
	"github.com/user/protocol-registry-cli/internal/usecases/validate_protocol"
	registryv1 "github.com/user/protocol_registry/pkg/api/registry/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type App struct {
	PublishUC    *publish_protocol.UseCase
	GetUC        *get_protocol.UseCase
	RegisterUC   *register_consumer.UseCase
	UnregisterUC *unregister_consumer.UseCase
	GrpcViewUC   *get_grpc_view.UseCase
	ValidateUC   *validate_protocol.UseCase
	conn         *grpc.ClientConn
}

func New(cfg config.Config) (*App, error) {
	conn, err := grpc.NewClient(cfg.ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	grpcClient := registryv1.NewProtocolRegistryClient(conn)
	registryClient := implementations.NewRegistryClientGRPC(grpcClient)
	publishFileReader := implementations.NewPublishFileReader()
	registerFileReader := implementations.NewRegisterFileReader()
	validateFileReader := implementations.NewValidateFileReader()

	return &App{
		PublishUC:    publish_protocol.New(registryClient, publishFileReader),
		GetUC:        get_protocol.New(registryClient),
		RegisterUC:   register_consumer.New(registryClient, registerFileReader),
		UnregisterUC: unregister_consumer.New(registryClient),
		GrpcViewUC:   get_grpc_view.New(registryClient),
		ValidateUC:   validate_protocol.New(registryClient, validateFileReader),
		conn:         conn,
	}, nil
}

func (a *App) Close() error {
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

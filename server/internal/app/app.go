package app

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/user/protocol_registry/internal/config"
	grpccontroller "github.com/user/protocol_registry/internal/controllers/grpc"
	"github.com/user/protocol_registry/internal/implementations"
	"github.com/user/protocol_registry/internal/usecases/get_grpc_view"
	"github.com/user/protocol_registry/internal/usecases/get_protocol"
	"github.com/user/protocol_registry/internal/usecases/list_services"
	"github.com/user/protocol_registry/internal/usecases/publish_protocol"
	"github.com/user/protocol_registry/internal/usecases/register_consumer"
	"github.com/user/protocol_registry/internal/usecases/unregister_consumer"
	registryv1 "github.com/user/protocol_registry/pkg/api/registry/v1"
)

type App struct {
	cfg        *config.Config
	grpcServer *grpc.Server
	pool       *pgxpool.Pool
}

func New(cfg *config.Config) *App {
	return &App{cfg: cfg}
}

func (a *App) Run(ctx context.Context) error {
	pool, err := pgxpool.New(ctx, a.cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	a.pool = pool

	if err = pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	log.Println("Connected to PostgreSQL")

	s3Client := s3.New(s3.Options{
		BaseEndpoint: aws.String(a.cfg.S3Endpoint),
		Region:       a.cfg.S3Region,
		Credentials:  credentials.NewStaticCredentialsProvider(a.cfg.S3AccessKey, a.cfg.S3SecretKey, ""),
		UsePathStyle: true,
	})
	log.Println("S3 client initialized")

	// Implementations
	serviceRepo := implementations.NewServiceRepositoryPostgres(pool)
	protocolRepo := implementations.NewProtocolRepositoryPostgres(pool)
	consumerRepo := implementations.NewConsumerRepositoryPostgres(pool)
	protocolStorage := implementations.NewProtocolStorageS3(s3Client, a.cfg.S3Bucket)
	syntaxValidator := implementations.NewProtocolSyntaxValidatorProtocompile()
	breakingChangesValidator := implementations.NewBreakingChangesValidatorProtocompile()
	protoInspector := implementations.NewProtoInspectorProtocompile()

	// Use cases
	publishUC := publish_protocol.New(serviceRepo, protocolRepo, protocolStorage, consumerRepo, protocolStorage, syntaxValidator, breakingChangesValidator)
	getUC := get_protocol.New(serviceRepo, protocolRepo, protocolStorage)
	registerUC := register_consumer.New(serviceRepo, protocolRepo, consumerRepo, protocolStorage, protocolStorage, syntaxValidator, breakingChangesValidator)
	unregisterUC := unregister_consumer.New(serviceRepo, consumerRepo, protocolStorage)
	grpcViewUC := get_grpc_view.New(serviceRepo, protocolRepo, protocolStorage, consumerRepo, protocolStorage, protoInspector)
	listServicesUC := list_services.New(serviceRepo)

	// Controller
	handler := grpccontroller.NewHandler(publishUC, getUC, registerUC, unregisterUC, grpcViewUC, listServicesUC)

	a.grpcServer = grpc.NewServer()
	registryv1.RegisterProtocolRegistryServer(a.grpcServer, handler)
	reflection.Register(a.grpcServer)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", a.cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	log.Printf("gRPC server listening on :%d", a.cfg.GRPCPort)
	return a.grpcServer.Serve(listener)
}

func (a *App) Shutdown() {
	if a.grpcServer != nil {
		a.grpcServer.GracefulStop()
	}
	if a.pool != nil {
		a.pool.Close()
	}
}

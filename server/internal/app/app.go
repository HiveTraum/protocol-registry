package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3service "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/user/protocol_registry/internal/config"
	grpccontroller "github.com/user/protocol_registry/internal/controllers/grpc"
	"github.com/user/protocol_registry/internal/implementations"
	"github.com/user/protocol_registry/internal/usecases/get_grpc_view"
	"github.com/user/protocol_registry/internal/usecases/get_protocol"
	"github.com/user/protocol_registry/internal/usecases/list_protocol_versions"
	"github.com/user/protocol_registry/internal/usecases/list_services"
	"github.com/user/protocol_registry/internal/usecases/publish_protocol"
	"github.com/user/protocol_registry/internal/usecases/register_consumer"
	"github.com/user/protocol_registry/internal/usecases/unregister_consumer"
	registryv1 "github.com/user/protocol_registry/pkg/api/registry/v1"
)

type App struct {
	cfg        *config.Config
	log        *slog.Logger
	grpcServer *grpc.Server
	httpServer *http.Server
	pool       *pgxpool.Pool
	s3Client   *s3service.Client
}

func New(cfg *config.Config) *App {
	return &App{
		cfg: cfg,
		log: newLogger(cfg.LogLevel),
	}
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
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
	a.log.Info("connected to PostgreSQL")

	s3Client := s3service.New(s3service.Options{
		BaseEndpoint: aws.String(a.cfg.S3Endpoint),
		Region:       a.cfg.S3Region,
		Credentials:  credentials.NewStaticCredentialsProvider(a.cfg.S3AccessKey, a.cfg.S3SecretKey, ""),
		UsePathStyle: true,
	})
	a.s3Client = s3Client
	a.log.Info("S3 client initialized")

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
	listVersionsUC := list_protocol_versions.New(serviceRepo, protocolRepo)

	// Controller
	handler := grpccontroller.NewHandler(publishUC, getUC, registerUC, unregisterUC, grpcViewUC, listServicesUC, listVersionsUC)

	// gRPC server with logging interceptor
	a.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(a.loggingUnaryInterceptor()),
	)
	registryv1.RegisterProtocolRegistryServer(a.grpcServer, handler)

	// gRPC health service
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(a.grpcServer, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(a.grpcServer)

	// HTTP health server
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", a.handleLiveness())
	mux.HandleFunc("/readyz", a.handleReadiness())
	a.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.cfg.HTTPPort),
		Handler: mux,
	}

	grpcErrCh := make(chan error, 1)
	httpErrCh := make(chan error, 1)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", a.cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	go func() {
		a.log.Info("gRPC server listening", "port", a.cfg.GRPCPort)
		if err := a.grpcServer.Serve(listener); err != nil {
			grpcErrCh <- err
		}
	}()

	go func() {
		a.log.Info("HTTP health server listening", "port", a.cfg.HTTPPort)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpErrCh <- err
		}
	}()

	select {
	case err := <-grpcErrCh:
		return fmt.Errorf("gRPC server: %w", err)
	case err := <-httpErrCh:
		return fmt.Errorf("HTTP server: %w", err)
	case <-ctx.Done():
		return nil
	}
}

func (a *App) Shutdown() {
	a.log.Info("shutting down")

	if a.grpcServer != nil {
		a.grpcServer.GracefulStop()
	}

	if a.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
			a.log.Error("HTTP server shutdown error", "error", err)
		}
	}

	if a.pool != nil {
		a.pool.Close()
	}

	a.log.Info("shutdown complete")
}

func (a *App) loggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		a.log.Info("gRPC request",
			"method", info.FullMethod,
			"duration_ms", time.Since(start).Milliseconds(),
			"error", err,
		)
		return resp, err
	}
}

func (a *App) handleLiveness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func (a *App) handleReadiness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.pool != nil {
			if err := a.pool.Ping(r.Context()); err != nil {
				a.log.Warn("readiness check: postgres ping failed", "error", err)
				http.Error(w, "postgres unavailable", http.StatusServiceUnavailable)
				return
			}
		}

		if a.s3Client != nil {
			_, err := a.s3Client.HeadBucket(r.Context(), &s3service.HeadBucketInput{
				Bucket: aws.String(a.cfg.S3Bucket),
			})
			if err != nil {
				a.log.Warn("readiness check: S3 bucket check failed", "error", err)
				http.Error(w, "s3 unavailable", http.StatusServiceUnavailable)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}

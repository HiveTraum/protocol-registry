package integration

import (
	"context"
	"database/sql"
	"embed"
	"log"
	"net"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpccreds "google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	grpccontroller "github.com/user/protocol_registry/internal/controllers/grpc"
	"github.com/user/protocol_registry/internal/implementations"
	"github.com/user/protocol_registry/internal/migrations"
	"github.com/user/protocol_registry/internal/usecases/get_grpc_view"
	"github.com/user/protocol_registry/internal/usecases/get_protocol"
	"github.com/user/protocol_registry/internal/usecases/list_services"
	"github.com/user/protocol_registry/internal/usecases/publish_protocol"
	"github.com/user/protocol_registry/internal/usecases/register_consumer"
	"github.com/user/protocol_registry/internal/usecases/unregister_consumer"
	registryv1 "github.com/user/protocol_registry/pkg/api/registry/v1"
)

const (
	bufSize    = 1024 * 1024
	bucketName = "test-protocols"
)

//go:embed testdata
var testdata embed.FS

var (
	pool     *pgxpool.Pool
	s3Client *s3.Client
	client   registryv1.ProtocolRegistryClient
	srv      *grpc.Server
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// --- Postgres ---
	pgContainer, err := tcpostgres.Run(ctx, "postgres:18-alpine",
		tcpostgres.WithDatabase("protocol_registry"),
		tcpostgres.WithUsername("registry"),
		tcpostgres.WithPassword("registry"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("start postgres container: %v", err)
	}
	defer pgContainer.Terminate(ctx)

	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("get postgres DSN: %v", err)
	}

	pool, err = pgxpool.New(ctx, pgDSN)
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer pool.Close()

	goose.SetBaseFS(migrations.FS)
	db, err := sql.Open("pgx", pgDSN)
	if err != nil {
		log.Fatalf("open db for migrations: %v", err)
	}
	if err := goose.Up(db, "sql"); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}
	db.Close()

	// --- MinIO ---
	minioContainer, err := tcminio.Run(ctx, "minio/minio")
	if err != nil {
		log.Fatalf("start minio container: %v", err)
	}
	defer minioContainer.Terminate(ctx)

	minioEndpoint, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("get minio endpoint: %v", err)
	}

	s3Client = s3.New(s3.Options{
		BaseEndpoint: aws.String("http://" + minioEndpoint),
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", ""),
		UsePathStyle: true,
	})

	if _, err := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}); err != nil {
		log.Fatalf("create S3 bucket: %v", err)
	}

	// --- gRPC server on bufconn ---
	listener := bufconn.Listen(bufSize)

	serviceRepo := implementations.NewServiceRepositoryPostgres(pool)
	protocolRepo := implementations.NewProtocolRepositoryPostgres(pool)
	consumerRepo := implementations.NewConsumerRepositoryPostgres(pool)
	protocolStorage := implementations.NewProtocolStorageS3(s3Client, bucketName)
	syntaxValidator := implementations.NewProtocolSyntaxValidatorProtocompile()
	breakingChangesValidator := implementations.NewBreakingChangesValidatorProtocompile()
	protoInspector := implementations.NewProtoInspectorProtocompile()

	publishUC := publish_protocol.New(serviceRepo, protocolRepo, protocolStorage, consumerRepo, protocolStorage, syntaxValidator, breakingChangesValidator)
	getUC := get_protocol.New(serviceRepo, protocolRepo, protocolStorage)
	registerUC := register_consumer.New(serviceRepo, protocolRepo, consumerRepo, protocolStorage, protocolStorage, syntaxValidator, breakingChangesValidator)
	unregisterUC := unregister_consumer.New(serviceRepo, consumerRepo, protocolStorage)
	grpcViewUC := get_grpc_view.New(serviceRepo, protocolRepo, protocolStorage, consumerRepo, protocolStorage, protoInspector)
	listServicesUC := list_services.New(serviceRepo)

	handler := grpccontroller.NewHandler(publishUC, getUC, registerUC, unregisterUC, grpcViewUC, listServicesUC)

	srv = grpc.NewServer()
	registryv1.RegisterProtocolRegistryServer(srv, handler)

	go func() {
		if err := srv.Serve(listener); err != nil {
			log.Printf("grpc serve: %v", err)
		}
	}()

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(grpccreds.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("dial bufconn: %v", err)
	}
	defer conn.Close()

	client = registryv1.NewProtocolRegistryClient(conn)

	// --- run tests ---
	code := m.Run()

	srv.GracefulStop()
	os.Exit(code)
}

// resetState truncates all tables and clears all objects in the S3 bucket.
func resetState(ctx context.Context, t *testing.T) {
	t.Helper()
	r := require.New(t)

	_, err := pool.Exec(ctx, "TRUNCATE services, protocols, consumers CASCADE")
	r.NoError(err, "truncate tables")

	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		r.NoError(err, "list objects for cleanup")
		if len(page.Contents) == 0 {
			continue
		}
		objects := make([]types.ObjectIdentifier, len(page.Contents))
		for i, obj := range page.Contents {
			objects[i] = types.ObjectIdentifier{Key: obj.Key}
		}
		_, err = s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucketName),
			Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(true)},
		})
		r.NoError(err, "delete objects for cleanup")
	}
}

func TestPublishAndGetProtocol(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	protoContent, err := testdata.ReadFile("testdata/test/v1/service.proto")
	r.NoError(err)

	// 1. PublishProtocol → is_new == true
	pubResp, err := client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  "test-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files: []*registryv1.ProtoFile{
			{Path: "test/v1/service.proto", Content: protoContent},
		},
		EntryPoint: "test/v1/service.proto",
	})
	r.NoError(err, "PublishProtocol (first)")
	r.True(pubResp.IsNew, "expected is_new == true on first publish")
	r.Equal("test-service", pubResp.ServiceName)

	// 2. GetProtocol → files and entry_point match
	getResp, err := client.GetProtocol(ctx, &registryv1.GetProtocolRequest{
		ServiceName:  "test-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
	})
	r.NoError(err, "GetProtocol")
	r.Equal("test/v1/service.proto", getResp.EntryPoint)
	r.Len(getResp.Files, 1)
	r.Equal("test/v1/service.proto", getResp.Files[0].Path)
	r.Equal(protoContent, getResp.Files[0].Content)

	// 3. Repeat PublishProtocol with same content → is_new == false (idempotency)
	pubResp2, err := client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  "test-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files: []*registryv1.ProtoFile{
			{Path: "test/v1/service.proto", Content: protoContent},
		},
		EntryPoint: "test/v1/service.proto",
	})
	r.NoError(err, "PublishProtocol (repeat)")
	r.False(pubResp2.IsNew, "expected is_new == false on repeat publish with same content")
}

func TestRegisterAndUnregisterConsumer(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	protoContent, err := testdata.ReadFile("testdata/test/v1/service.proto")
	r.NoError(err)

	// Publish server protocol
	_, err = client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  "orders-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files: []*registryv1.ProtoFile{
			{Path: "test/v1/service.proto", Content: protoContent},
		},
		EntryPoint: "test/v1/service.proto",
	})
	r.NoError(err)

	// Register consumer
	regResp, err := client.RegisterConsumer(ctx, &registryv1.RegisterConsumerRequest{
		ConsumerName: "billing-service",
		ServerName:   "orders-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files: []*registryv1.ProtoFile{
			{Path: "test/v1/service.proto", Content: protoContent},
		},
		EntryPoint: "test/v1/service.proto",
	})
	r.NoError(err)
	r.True(regResp.IsNew)
	r.Equal("billing-service", regResp.ConsumerName)
	r.Equal("orders-service", regResp.ServerName)

	// Unregister consumer
	_, err = client.UnregisterConsumer(ctx, &registryv1.UnregisterConsumerRequest{
		ConsumerName: "billing-service",
		ServerName:   "orders-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
	})
	r.NoError(err)

	// Unregister again → NotFound
	_, err = client.UnregisterConsumer(ctx, &registryv1.UnregisterConsumerRequest{
		ConsumerName: "billing-service",
		ServerName:   "orders-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
	})
	r.Error(err)
	r.Equal(codes.NotFound, status.Code(err))
}

func TestPublishBreakingChangeBlockedByConsumer(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	protoContent, err := testdata.ReadFile("testdata/test/v1/service.proto")
	r.NoError(err)
	breakingContent, err := testdata.ReadFile("testdata/test/v1/service_v2_breaking.proto")
	r.NoError(err)

	// Publish v1
	_, err = client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  "payments-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files: []*registryv1.ProtoFile{
			{Path: "test/v1/service.proto", Content: protoContent},
		},
		EntryPoint: "test/v1/service.proto",
	})
	r.NoError(err)

	// Register consumer
	_, err = client.RegisterConsumer(ctx, &registryv1.RegisterConsumerRequest{
		ConsumerName: "checkout-service",
		ServerName:   "payments-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files: []*registryv1.ProtoFile{
			{Path: "test/v1/service.proto", Content: protoContent},
		},
		EntryPoint: "test/v1/service.proto",
	})
	r.NoError(err)

	// Publish v2 with breaking changes → FailedPrecondition
	_, err = client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  "payments-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files: []*registryv1.ProtoFile{
			{Path: "test/v1/service.proto", Content: breakingContent},
		},
		EntryPoint: "test/v1/service.proto",
	})
	r.Error(err)
	r.Equal(codes.FailedPrecondition, status.Code(err))
}

func TestGetProtocolNotFound(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	_, err := client.GetProtocol(ctx, &registryv1.GetProtocolRequest{
		ServiceName:  "nonexistent-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
	})
	r.Error(err)
	r.Equal(codes.NotFound, status.Code(err))
}

// readTestdata is a helper that reads an embedded testdata file and fails the test on error.
func readTestdata(t *testing.T, path string) []byte {
	t.Helper()
	data, err := testdata.ReadFile("testdata/" + path)
	require.NoError(t, err)
	return data
}

// publishProto is a helper that publishes a single-file protocol and fails the test on error.
func publishProto(t *testing.T, ctx context.Context, serviceName, filePath string, content []byte) {
	t.Helper()
	_, err := client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  serviceName,
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files:        []*registryv1.ProtoFile{{Path: filePath, Content: content}},
		EntryPoint:   filePath,
	})
	require.NoError(t, err)
}

// registerConsumer is a helper that registers a single-file consumer and fails the test on error.
func registerConsumer(t *testing.T, ctx context.Context, consumerName, serverName, filePath string, content []byte) {
	t.Helper()
	_, err := client.RegisterConsumer(ctx, &registryv1.RegisterConsumerRequest{
		ConsumerName: consumerName,
		ServerName:   serverName,
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files:        []*registryv1.ProtoFile{{Path: filePath, Content: content}},
		EntryPoint:   filePath,
	})
	require.NoError(t, err)
}

func TestPublishInvalidSyntax(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	invalidContent := readTestdata(t, "test/v1/invalid.proto")

	_, err := client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  "bad-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files:        []*registryv1.ProtoFile{{Path: "test/v1/invalid.proto", Content: invalidContent}},
		EntryPoint:   "test/v1/invalid.proto",
	})
	r.Error(err)
	r.Equal(codes.InvalidArgument, status.Code(err))
}

func TestPublishValidationErrors(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)

	protoContent := readTestdata(t, "test/v1/service.proto")

	t.Run("empty service_name", func(t *testing.T) {
		r := require.New(t)
		_, err := client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
			ServiceName:  "",
			ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
			Files:        []*registryv1.ProtoFile{{Path: "test/v1/service.proto", Content: protoContent}},
			EntryPoint:   "test/v1/service.proto",
		})
		r.Error(err)
		r.Equal(codes.InvalidArgument, status.Code(err))
	})

	t.Run("empty files", func(t *testing.T) {
		r := require.New(t)
		_, err := client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
			ServiceName:  "svc",
			ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
			Files:        []*registryv1.ProtoFile{},
			EntryPoint:   "test/v1/service.proto",
		})
		r.Error(err)
		r.Equal(codes.InvalidArgument, status.Code(err))
	})

	t.Run("entry_point not in files", func(t *testing.T) {
		r := require.New(t)
		_, err := client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
			ServiceName:  "svc",
			ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
			Files:        []*registryv1.ProtoFile{{Path: "test/v1/service.proto", Content: protoContent}},
			EntryPoint:   "nonexistent.proto",
		})
		r.Error(err)
		r.Equal(codes.InvalidArgument, status.Code(err))
	})
}

func TestRegisterConsumerServerNotFound(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	protoContent := readTestdata(t, "test/v1/service.proto")

	_, err := client.RegisterConsumer(ctx, &registryv1.RegisterConsumerRequest{
		ConsumerName: "my-consumer",
		ServerName:   "nonexistent-server",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files:        []*registryv1.ProtoFile{{Path: "test/v1/service.proto", Content: protoContent}},
		EntryPoint:   "test/v1/service.proto",
	})
	r.Error(err)
	r.Equal(codes.NotFound, status.Code(err))
}

func TestRegisterConsumerNotSubset(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	protoContent := readTestdata(t, "test/v1/service.proto")
	supersetContent := readTestdata(t, "test/v1/consumer_superset.proto")

	// Publish server protocol (has only GetItem)
	publishProto(t, ctx, "api-service", "test/v1/service.proto", protoContent)

	// Register consumer with superset proto (has GetItem + DeleteItem) → error
	_, err := client.RegisterConsumer(ctx, &registryv1.RegisterConsumerRequest{
		ConsumerName: "greedy-consumer",
		ServerName:   "api-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files:        []*registryv1.ProtoFile{{Path: "test/v1/consumer_superset.proto", Content: supersetContent}},
		EntryPoint:   "test/v1/consumer_superset.proto",
	})
	r.Error(err)
	r.Equal(codes.FailedPrecondition, status.Code(err))
}

func TestRegisterConsumerIdempotent(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	protoContent := readTestdata(t, "test/v1/service.proto")

	publishProto(t, ctx, "idempotent-server", "test/v1/service.proto", protoContent)

	// First registration
	resp1, err := client.RegisterConsumer(ctx, &registryv1.RegisterConsumerRequest{
		ConsumerName: "idempotent-consumer",
		ServerName:   "idempotent-server",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files:        []*registryv1.ProtoFile{{Path: "test/v1/service.proto", Content: protoContent}},
		EntryPoint:   "test/v1/service.proto",
	})
	r.NoError(err)
	r.True(resp1.IsNew)

	// Second registration with same content → is_new == false
	resp2, err := client.RegisterConsumer(ctx, &registryv1.RegisterConsumerRequest{
		ConsumerName: "idempotent-consumer",
		ServerName:   "idempotent-server",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files:        []*registryv1.ProtoFile{{Path: "test/v1/service.proto", Content: protoContent}},
		EntryPoint:   "test/v1/service.proto",
	})
	r.NoError(err)
	r.False(resp2.IsNew)
}

func TestPublishCompatibleUpdate(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	v1Content := readTestdata(t, "test/v1/service.proto")
	v2Content := readTestdata(t, "test/v1/service_v2_compatible.proto")

	// Publish v1
	publishProto(t, ctx, "evolving-service", "test/v1/service.proto", v1Content)

	// Register consumer on v1
	registerConsumer(t, ctx, "stable-consumer", "evolving-service", "test/v1/service.proto", v1Content)

	// Publish v2 (compatible: adds ListItems, keeps GetItem) → success, is_new == false (update)
	pubResp, err := client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  "evolving-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files:        []*registryv1.ProtoFile{{Path: "test/v1/service.proto", Content: v2Content}},
		EntryPoint:   "test/v1/service.proto",
	})
	r.NoError(err)
	r.False(pubResp.IsNew, "is_new == false because protocol record already exists (update, not insert)")

	// GetProtocol returns v2 content
	getResp, err := client.GetProtocol(ctx, &registryv1.GetProtocolRequest{
		ServiceName:  "evolving-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
	})
	r.NoError(err)
	r.Len(getResp.Files, 1)
	r.Equal(v2Content, getResp.Files[0].Content)
}

func TestPublishBreakingChangeAfterUnregister(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	v1Content := readTestdata(t, "test/v1/service.proto")
	breakingContent := readTestdata(t, "test/v1/service_v2_breaking.proto")

	// Publish v1 and register consumer
	publishProto(t, ctx, "free-service", "test/v1/service.proto", v1Content)
	registerConsumer(t, ctx, "temp-consumer", "free-service", "test/v1/service.proto", v1Content)

	// Breaking change blocked while consumer exists
	_, err := client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  "free-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files:        []*registryv1.ProtoFile{{Path: "test/v1/service.proto", Content: breakingContent}},
		EntryPoint:   "test/v1/service.proto",
	})
	r.Equal(codes.FailedPrecondition, status.Code(err))

	// Unregister the consumer
	_, err = client.UnregisterConsumer(ctx, &registryv1.UnregisterConsumerRequest{
		ConsumerName: "temp-consumer",
		ServerName:   "free-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
	})
	r.NoError(err)

	// Now breaking change succeeds
	pubResp, err := client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  "free-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files:        []*registryv1.ProtoFile{{Path: "test/v1/service.proto", Content: breakingContent}},
		EntryPoint:   "test/v1/service.proto",
	})
	r.NoError(err)
	r.False(pubResp.IsNew, "is_new == false because protocol record already exists (update, not insert)")
}

func TestPublishMultipleFiles(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	serviceContent := readTestdata(t, "multi/v1/service.proto")
	typesContent := readTestdata(t, "multi/v1/types.proto")

	// Publish with two files
	pubResp, err := client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  "multi-file-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files: []*registryv1.ProtoFile{
			{Path: "multi/v1/service.proto", Content: serviceContent},
			{Path: "multi/v1/types.proto", Content: typesContent},
		},
		EntryPoint: "multi/v1/service.proto",
	})
	r.NoError(err)
	r.True(pubResp.IsNew)

	// GetProtocol returns both files
	getResp, err := client.GetProtocol(ctx, &registryv1.GetProtocolRequest{
		ServiceName:  "multi-file-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
	})
	r.NoError(err)
	r.Equal("multi/v1/service.proto", getResp.EntryPoint)
	r.Len(getResp.Files, 2)

	filesByPath := make(map[string][]byte)
	for _, f := range getResp.Files {
		filesByPath[f.Path] = f.Content
	}
	r.Equal(serviceContent, filesByPath["multi/v1/service.proto"])
	r.Equal(typesContent, filesByPath["multi/v1/types.proto"])
}

func TestMultipleConsumersCompatibleUpdate(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	v1Content := readTestdata(t, "test/v1/service.proto")
	v2Content := readTestdata(t, "test/v1/service_v2_compatible.proto")

	publishProto(t, ctx, "shared-service", "test/v1/service.proto", v1Content)
	registerConsumer(t, ctx, "consumer-a", "shared-service", "test/v1/service.proto", v1Content)
	registerConsumer(t, ctx, "consumer-b", "shared-service", "test/v1/service.proto", v1Content)

	// Compatible update with two consumers → success
	pubResp, err := client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  "shared-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files:        []*registryv1.ProtoFile{{Path: "test/v1/service.proto", Content: v2Content}},
		EntryPoint:   "test/v1/service.proto",
	})
	r.NoError(err)
	r.False(pubResp.IsNew, "is_new == false because protocol record already exists (update, not insert)")
}

func TestMultipleConsumersBreakingChange(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	v1Content := readTestdata(t, "test/v1/service.proto")
	breakingContent := readTestdata(t, "test/v1/service_v2_breaking.proto")

	publishProto(t, ctx, "popular-service", "test/v1/service.proto", v1Content)
	registerConsumer(t, ctx, "consumer-x", "popular-service", "test/v1/service.proto", v1Content)
	registerConsumer(t, ctx, "consumer-y", "popular-service", "test/v1/service.proto", v1Content)

	// Breaking change with two consumers → FailedPrecondition
	_, err := client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  "popular-service",
		ProtocolType: registryv1.ProtocolType_PROTOCOL_TYPE_GRPC,
		Files:        []*registryv1.ProtoFile{{Path: "test/v1/service.proto", Content: breakingContent}},
		EntryPoint:   "test/v1/service.proto",
	})
	r.Error(err)
	r.Equal(codes.FailedPrecondition, status.Code(err))
}

func TestGetGrpcView(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	// Server proto: TestService with GetItem (id, name in response)
	serverContent := readTestdata(t, "test/v1/service.proto")
	publishProto(t, ctx, "view-server", "test/v1/service.proto", serverContent)

	// Consumer A: full proto (uses GetItem, all fields)
	fullConsumerContent := readTestdata(t, "test/v1/service.proto")
	registerConsumer(t, ctx, "consumer-full", "view-server", "test/v1/service.proto", fullConsumerContent)

	// Consumer B: subset proto (uses GetItem, only id field in response)
	subsetConsumerContent := readTestdata(t, "test/v1/consumer_subset.proto")
	registerConsumer(t, ctx, "consumer-subset", "view-server", "test/v1/consumer_subset.proto", subsetConsumerContent)

	// Call GetGrpcView
	resp, err := client.GetGrpcView(ctx, &registryv1.GetGrpcViewRequest{
		ServiceName: "view-server",
	})
	r.NoError(err)
	r.Equal("view-server", resp.ServiceName)

	// Should have one service: TestService
	r.Len(resp.Services, 1)
	svc := resp.Services[0]
	r.Equal("TestService", svc.Name)

	// Should have one method: GetItem
	r.Len(svc.Methods, 1)
	method := svc.Methods[0]
	r.Equal("GetItem", method.Name)

	// Both consumers use GetItem
	r.ElementsMatch([]string{"consumer-full", "consumer-subset"}, method.Consumers)

	// Input message: GetItemRequest with field "id" (number=1)
	r.Equal("GetItemRequest", method.Input.Name)
	r.Len(method.Input.Fields, 1)
	r.Equal("id", method.Input.Fields[0].Name)
	r.Equal(uint32(1), method.Input.Fields[0].Number)
	// Both consumers use field "id" in the request
	r.ElementsMatch([]string{"consumer-full", "consumer-subset"}, method.Input.Fields[0].Consumers)

	// Output message: GetItemResponse with fields "id" (1) and "name" (2)
	r.Equal("GetItemResponse", method.Output.Name)
	r.Len(method.Output.Fields, 2)

	fieldsByName := make(map[string]*registryv1.GrpcFieldView)
	for _, f := range method.Output.Fields {
		fieldsByName[f.Name] = f
	}

	// Field "id" — used by both consumers
	idField := fieldsByName["id"]
	r.NotNil(idField)
	r.Equal(uint32(1), idField.Number)
	r.ElementsMatch([]string{"consumer-full", "consumer-subset"}, idField.Consumers)

	// Field "name" — used only by consumer-full (consumer-subset doesn't have it)
	nameField := fieldsByName["name"]
	r.NotNil(nameField)
	r.Equal(uint32(2), nameField.Number)
	r.Equal([]string{"consumer-full"}, nameField.Consumers)
}

func TestGetGrpcViewNotFound(t *testing.T) {
	ctx := context.Background()
	resetState(ctx, t)
	r := require.New(t)

	_, err := client.GetGrpcView(ctx, &registryv1.GetGrpcViewRequest{
		ServiceName: "nonexistent-service",
	})
	r.Error(err)
	r.Equal(codes.NotFound, status.Code(err))
}

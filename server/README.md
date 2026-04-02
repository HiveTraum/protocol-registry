# Protocol Registry

A centralized registry for storing, versioning, and validating gRPC protocol definitions. Services publish their `.proto` files on deploy, and the registry ensures backward compatibility by validating changes against registered consumers.

## Features

- **Protocol publishing** — upload `.proto` files tied to a service name
- **Consumer registration** — services declare which methods and fields they use from other services
- **Consumer-aware compatibility** — server can remove methods/fields not used by any consumer; changes breaking a consumer are rejected with the consumer name
- **Syntax validation** — proto files are validated before storage
- **Idempotent publishing** — re-publishing the same content returns the current version without creating a duplicate
- **Protocol retrieval** — clients can fetch the latest proto definition for any registered service
- **S3 storage** — proto files are stored in S3 (or any S3-compatible storage like MinIO)
- **gRPC transport** — the registry itself exposes a gRPC API

### Breaking change detection

The following incompatible changes are detected:

| Category     | Detected changes                                                            |
|--------------|-----------------------------------------------------------------------------|
| **Messages** | Removal, field removal, field type change, cardinality change, field rename |
| **Enums**    | Removal, value removal                                                      |
| **Services** | Removal, method removal, input/output type change, streaming mode change    |

When publishing a new server proto, breaking changes are validated **against all registered consumers**, not against the previous version. This means a server can freely remove methods and fields as long as no consumer depends on them.

## Architecture

```
┌─────────────────┐       gRPC             ┌──────────────────────────┐
│  Server CI/CD   │ ────────────────────▶  │    Protocol Registry     │
│                 │   PublishProtocol      │                          │
└─────────────────┘                        │  ┌────────┐ ┌─────────┐  │
                                           │  │Validate│ │  Store  │  │
┌─────────────────┐       gRPC             │  └───┬────┘ └────┬────┘  │
│  Consumer CI/CD │ ────────────────────▶  │      │           │       │
│                 │   RegisterConsumer     │      ▼           ▼       │
│                 │   GetProtocol          │  PostgreSQL      S3      │
└─────────────────┘                        └──────────────────────────┘
```

### Flow

```
Server publishes proto
  → Find all consumers of this service
  → For each consumer: verify its proto is a valid subset of the new server proto
  → If a consumer uses a removed method/field → error naming the consumer
  → If all ok → store new version

Consumer registers
  → Validate consumer proto syntax
  → Verify consumer proto is a subset of the current server proto
  → Store consumer proto in S3, record in DB

Consumer unregisters
  → Remove record from DB + proto from S3
```

## Tech stack

- **Go 1.24**
- **gRPC** — transport
- **PostgreSQL 18** — service, protocol, and consumer metadata (UUID v7)
- **S3 / MinIO** — proto file storage
- **protocompile** — syntax validation and breaking change detection
- **goose** — database migrations (embedded in the binary)
- **Buf** — proto linting and code generation

## Quick start

### Prerequisites

- Go 1.24+
- Docker and Docker Compose
- [Buf CLI](https://buf.build/docs/installation) (for code generation)
- [grpcurl](https://github.com/fullstorydev/grpcurl) (optional, for manual testing)

### 1. Start infrastructure

```bash
make docker-up
```

This starts PostgreSQL and MinIO.

### 2. Run migrations

```bash
make migrate-up
```

### 3. Create S3 bucket

Open MinIO Console at http://localhost:9001 (login: `minioadmin` / `minioadmin`) and create a bucket named `protocol-registry`.

### 4. Start the server

```bash
cp .env.example .env
make run
```

The gRPC server starts on port `50051`.

### 5. Publish a protocol

```bash
grpcurl -plaintext \
  -d "{
    \"service_name\": \"user-service\",
    \"proto_content\": \"$(base64 -i your_service.proto)\",
    \"protocol_type\": \"PROTOCOL_TYPE_GRPC\"
  }" \
  localhost:50051 registry.v1.ProtocolRegistry/PublishProtocol
```

### 6. Register a consumer

```bash
grpcurl -plaintext \
  -d "{
    \"consumer_name\": \"order-service\",
    \"server_name\": \"user-service\",
    \"protocol_type\": \"PROTOCOL_TYPE_GRPC\",
    \"proto_content\": \"$(base64 -i consumer.proto)\"
  }" \
  localhost:50051 registry.v1.ProtocolRegistry/RegisterConsumer
```

The consumer proto should be a subset of the server proto — only the methods and fields the consumer actually uses.

### 7. Unregister a consumer

```bash
grpcurl -plaintext \
  -d '{
    "consumer_name": "order-service",
    "server_name": "user-service",
    "protocol_type": "PROTOCOL_TYPE_GRPC"
  }' \
  localhost:50051 registry.v1.ProtocolRegistry/UnregisterConsumer
```

### 8. Retrieve a protocol

```bash
grpcurl -plaintext \
  -d '{
    "service_name": "user-service",
    "protocol_type": "PROTOCOL_TYPE_GRPC"
  }' \
  localhost:50051 registry.v1.ProtocolRegistry/GetProtocol
```

## API

### PublishProtocol

Publishes a proto file for a service. If the service doesn't exist, it is created automatically. Breaking changes are validated against all registered consumers.

| Field           | Type   | Description               |
|-----------------|--------|---------------------------|
| `service_name`  | string | Unique service identifier |
| `proto_content` | bytes  | Raw `.proto` file content |
| `protocol_type` | enum   | `PROTOCOL_TYPE_GRPC`      |

**Response:**

| Field          | Type   | Description                                                |
|----------------|--------|------------------------------------------------------------|
| `service_name` | string | Service name                                               |
| `is_new`       | bool   | `true` if service was just created or protocol was updated |

**Errors:**
- `FAILED_PRECONDITION` — breaking changes affect one or more consumers (field violations include `consumer:{name}/{subject}`)
- `INVALID_ARGUMENT` — proto syntax errors

### GetProtocol

Retrieves the latest proto definition for a service.

| Field           | Type   | Description          |
|-----------------|--------|----------------------|
| `service_name`  | string | Service to look up   |
| `protocol_type` | enum   | `PROTOCOL_TYPE_GRPC` |

**Response:**

| Field           | Type   | Description               |
|-----------------|--------|---------------------------|
| `service_name`  | string | Service name              |
| `protocol_type` | enum   | Protocol type             |
| `content`       | bytes  | Raw `.proto` file content |

### RegisterConsumer

Registers a service as a consumer of another service's protocol. The consumer proto must be a valid subset of the server proto.

| Field           | Type   | Description                                  |
|-----------------|--------|----------------------------------------------|
| `consumer_name` | string | Name of the consuming service                |
| `server_name`   | string | Name of the service being consumed           |
| `protocol_type` | enum   | `PROTOCOL_TYPE_GRPC`                         |
| `proto_content` | bytes  | Consumer's proto (subset of server's proto)  |

**Response:**

| Field           | Type   | Description                             |
|-----------------|--------|-----------------------------------------|
| `consumer_name` | string | Consumer service name                   |
| `server_name`   | string | Server service name                     |
| `is_new`        | bool   | `true` if this is a new registration    |

### UnregisterConsumer

Removes a consumer registration.

| Field           | Type   | Description                        |
|-----------------|--------|------------------------------------|
| `consumer_name` | string | Name of the consuming service      |
| `server_name`   | string | Name of the service being consumed |
| `protocol_type` | enum   | `PROTOCOL_TYPE_GRPC`               |

## Configuration

Environment variables (or `.env` file):

| Variable        | Default     | Description                                          |
|-----------------|-------------|------------------------------------------------------|
| `GRPC_PORT`     | `50051`     | gRPC server port                                     |
| `POSTGRES_DSN`  | —           | PostgreSQL connection string                         |
| `S3_BUCKET`     | —           | S3 bucket name                                       |
| `S3_ENDPOINT`   | —           | S3 endpoint (e.g. `http://localhost:9000` for MinIO) |
| `S3_ACCESS_KEY` | —           | S3 access key                                        |
| `S3_SECRET_KEY` | —           | S3 secret key                                        |
| `S3_REGION`     | `us-east-1` | S3 region                                            |

## Development

```bash
make generate      # regenerate protobuf code
make build         # compile binary to bin/server
make test          # run tests
make lint          # run buf lint + golangci-lint
make migrate-up    # apply database migrations
make docker-up     # start PostgreSQL + MinIO
make docker-down   # stop infrastructure
```

Migrations are embedded into the binary via `go:embed`. You can also run them directly:

```bash
./bin/server migrate up
```

## Project structure

```
├── api/proto/                  # protobuf definitions
├── cmd/server/                 # CLI entrypoint (serve, migrate up)
├── internal/
│   ├── app/                    # application bootstrap and DI
│   ├── config/                 # configuration loading
│   ├── controllers/grpc/       # gRPC handlers
│   ├── entities/               # domain models and errors
│   ├── implementations/        # postgres, s3, validators
│   ├── migrations/             # embedded SQL migrations (goose)
│   │   └── sql/               # *.sql migration files
│   └── usecases/               # business logic
└── pkg/api/                    # generated protobuf Go code
```

## Roadmap

- [x] Protocol version history and listing
- [ ] Protocol dependency graph — visual map of which consumers use which methods/fields
- [ ] OpenAPI and AsyncAPI support
- [ ] Web UI for browsing registered protocols
- [ ] Authentication and authorization
- [x] Multi-file proto support (imports)

## License

MIT

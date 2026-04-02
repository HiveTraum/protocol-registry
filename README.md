# Protocol Registry

A centralized contract broker for gRPC services. Publish proto definitions, register consumers, and catch breaking changes before they reach production.

## The Problem

In microservice architectures, breaking changes to proto definitions silently break consumers. Teams discover this at deploy time — or worse, in production. Protocol Registry prevents this by validating every publish against all registered consumers.

## How It Works

```
Server publishes .proto → Registry finds all consumers
                        → Validates new proto against each consumer's declared usage
                        → Rejects if any consumer would break (names the consumer)
                        → Stores new version if all clear
```

**Core capabilities:**
- Publish versioned `.proto` files per service
- Register consumers with their subset of the API they actually use
- Detect breaking changes: removed methods, changed field types, removed enum values
- Version history for audit and rollback decisions
- Multi-file proto support with imports

## Quick Start

```bash
# Start infrastructure (PostgreSQL + MinIO)
cd server && make docker-up

# Run migrations
make migrate-up

# Create S3 bucket (open http://localhost:9001, login minioadmin/minioadmin)

# Start the server
cp .env.example .env
make run
# gRPC on :50051, health on :8080
```

## CLI

```bash
cd cli && make build

# Initialize a service
prctl init service --service-name my-service
prctl init protocol

# Publish server proto
prctl publish

# Register as consumer
prctl get --service-name other-service
prctl register --service-name other-service

# View API with consumer usage
prctl grpc-view --service-name other-service
```

See [cli/README.md](cli/README.md) for full CLI reference.

## Architecture

| Component         | Tech                          |
|-------------------|-------------------------------|
| Server            | Go 1.25, gRPC                 |
| Database          | PostgreSQL 18 (UUID v7)       |
| File storage      | S3 / MinIO                    |
| Validation        | protocompile                  |
| Migrations        | goose (embedded)              |

See [server/README.md](server/README.md) for API reference, configuration, and development guide.

## CI/CD

GitHub Actions pipeline: lint, test, build, Docker image. Triggers on push to `main` and PRs.

## Roadmap

- [x] Core publish/validate/consume flow
- [x] Multi-file proto support with imports
- [x] Breaking change detection (messages, enums, services)
- [x] Protocol version history
- [x] CI/CD pipeline
- [x] Health endpoints and structured logging
- [ ] Authentication and authorization
- [ ] OpenAPI/Swagger support
- [ ] Web UI and dependency graph
- [ ] Public documentation site

## License

MIT

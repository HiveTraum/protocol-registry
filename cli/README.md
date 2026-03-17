# prctl — Protocol Registry CLI

CLI for managing gRPC protocol contracts between services.

## Prerequisites

Every project that uses `prctl` must have a `service.toml` file in the root directory:

```toml
[service]
name = "my-service"
```

This file identifies the current service for all commands. For legacy services without `service.toml`, use `prctl init service` to create it.

## Directory convention

```
protocols/
  grpc/
    server/              # Published server protocol
      my_service.proto
    clients/
      other-service/     # Downloaded client protocol
        other_service.proto
```

## Commands

### init service

Create `service.toml` for a legacy service.

```bash
prctl init service --service-name my-service
```

### init protocol

Generate a proto template from `service.toml`.

```bash
prctl init protocol
```

Creates `protocols/grpc/server/<service_name>.proto` with a basic service definition.

### publish

Publish the server protocol to the registry.

```bash
prctl publish
```

| Flag              | Default                            | Description                 |
|-------------------|------------------------------------|-----------------------------|
| `--proto-dir`     | `protocols/<protocol-type>/server` | Directory with .proto files |
| `--entry-point`   | `default.proto`                    | Entry point .proto file     |
| `--protocol-type` | `grpc`                             | Protocol type               |

### get

Download a service's protocol into the clients directory.

```bash
prctl get --service-name other-service
```

Files are written to `protocols/<protocol-type>/clients/<service-name>/`.

| Flag              | Default    | Description                           |
|-------------------|------------|---------------------------------------|
| `--service-name`  | *required* | Service to download the protocol from |
| `--protocol-type` | `grpc`     | Protocol type                         |

### register

Register the current service as a consumer using local client proto files.

```bash
prctl register --service-name other-service
```

Uploads proto files from `protocols/<protocol-type>/clients/<service-name>/` to the registry.

| Flag              | Default         | Description             |
|-------------------|-----------------|-------------------------|
| `--service-name`  | *required*      | Service to consume      |
| `--entry-point`   | `default.proto` | Entry point .proto file |
| `--protocol-type` | `grpc`          | Protocol type           |

### unregister

Unregister the current service as a consumer.

```bash
prctl unregister --service-name other-service
```

| Flag              | Default    | Description                |
|-------------------|------------|----------------------------|
| `--service-name`  | *required* | Service to unregister from |
| `--protocol-type` | `grpc`     | Protocol type              |

### grpc-view

View a service's gRPC API with consumer usage info.

```bash
prctl grpc-view --service-name other-service
```

## User flow

### Publishing a protocol (server side)

```bash
# 1. Initialize the project (one-time)
prctl init service --service-name my-service
prctl init protocol

# 2. Edit protocols/grpc/server/my_service.proto

# 3. Publish to the registry
prctl publish
```

### Consuming a protocol (client side)

```bash
# 1. Download the server's protocol
prctl get --service-name other-service

# 2. Edit proto files in protocols/grpc/clients/other-service/ as needed

# 3. Register as a consumer
prctl register --service-name other-service
```

### Stopping consumption

```bash
prctl unregister --service-name other-service
```

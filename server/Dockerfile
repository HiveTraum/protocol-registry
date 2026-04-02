# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/bin/server ./cmd/server

# Run stage
FROM gcr.io/distroless/static-debian12

COPY --from=builder /app/bin/server /server

EXPOSE 50051

ENTRYPOINT ["/server"]

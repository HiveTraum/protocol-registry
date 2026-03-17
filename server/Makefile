.PHONY: generate build run test lint migrate-up docker-up docker-down

generate:
	buf generate

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test -v ./...

lint:
	buf lint
	golangci-lint run

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

migrate-up:
	go run ./cmd/server migrate up

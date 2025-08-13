.PHONY: help build run test clean dev service cli docker-up docker-down docker-build

help:
	@echo "Available targets:"
	@echo "  build       - Build all binaries"
	@echo "  service     - Build and run the EDI service"
	@echo "  cli         - Build the EDI CLI tool"
	@echo "  run         - Run the EDI service"
	@echo "  dev         - Run service in development mode with auto-reload"
	@echo "  test        - Run all tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  migrate     - Run database migrations"
	@echo "  docker-up   - Start services with Docker Compose"
	@echo "  docker-down - Stop Docker Compose services"
	@echo "  docker-build- Build Docker images"

build: cli service-build

cli:
	GOWORK=off GOCACHE=$$(pwd)/.gocache go build -o bin/edi-cli ./cmd/edi-cli

service-build:
	GOWORK=off GOCACHE=$$(pwd)/.gocache go build -o bin/edi-service ./cmd/edi-service

service: service-build
	./bin/edi-service

run:
	go run ./cmd/edi-service

dev:
	@if [ ! -f .env ]; then cp .env.example .env; fi
	go run ./cmd/edi-service

test:
	go test -v ./...

clean:
	rm -rf bin/ .gocache/

migrate:
	@echo "Migrations are run automatically on service startup"

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker-compose build

docker-logs:
	docker-compose logs -f edi-service
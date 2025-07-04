.PHONY: help build run test clean docker-up docker-down migrate-up migrate-down migrate-status dev install-tools import-osm

# Default target
.DEFAULT_GOAL := help

# Variables
GOOSE := $(shell which goose || echo $(HOME)/go/bin/goose)
DB_URL := postgres://routing:routing@localhost:5433/routing?sslmode=disable
DOCKER_COMPOSE := docker-compose
GO := go
BINARY_NAME := routing-server
IMPORTER_NAME := routing-importer

## help: Display this help message
help:
	@echo "Routing Service - Available Commands"
	@echo "===================================="
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

## dev: Start development environment (DB, Redis, and server with hot reload)
dev: docker-up
	@echo "Starting development server..."
	@$(GO) run cmd/server/main.go

## run: Run the server
run:
	@echo "Starting server..."
	@$(GO) run cmd/server/main.go

## build: Build the application binaries
build:
	@echo "Building binaries..."
	@$(GO) build -o bin/$(BINARY_NAME) cmd/server/main.go
	@$(GO) build -o bin/$(IMPORTER_NAME) cmd/importer/main.go

## test: Run tests (excluding importer that requires zlib)
test:
	@echo "Running tests..."
	@$(GO) test -v ./internal/graph/... ./internal/api/... ./internal/storage/... ./cmd/server/...

## test-all: Run all tests (requires zlib/pkg-config for importer)
test-all:
	@echo "Running all tests..."
	@echo "Note: Requires zlib development headers (apt-get install zlib1g-dev or brew install zlib)"
	@$(GO) test -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@$(GO) test -v -cover -coverprofile=coverage.out ./internal/graph/... ./internal/api/... ./internal/storage/...
	@$(GO) tool cover -html=coverage.out -o coverage.html

## test-graph: Run graph algorithm tests only
test-graph:
	@echo "Running graph tests..."
	@$(GO) test -v ./internal/graph/... -run="Test"

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

##@ Database

## migrate-up: Run all pending migrations
migrate-up:
	@echo "Running migrations..."
	@cd internal/database && $(GOOSE) -dir migrations postgres "$(DB_URL)" up

## migrate-down: Rollback last migration
migrate-down:
	@echo "Rolling back migration..."
	@cd internal/database && $(GOOSE) -dir migrations postgres "$(DB_URL)" down

## migrate-status: Show migration status
migrate-status:
	@echo "Migration status:"
	@cd internal/database && $(GOOSE) -dir migrations postgres "$(DB_URL)" status

## migrate-create: Create a new migration (usage: make migrate-create name=migration_name)
migrate-create:
	@echo "Creating migration: $(name)"
	@cd internal/database && $(GOOSE) -dir migrations create $(name) sql

##@ Docker

## docker-up: Start Docker containers (PostgreSQL and Redis)
docker-up:
	@echo "Starting Docker containers..."
	@$(DOCKER_COMPOSE) up -d postgres redis

## docker-down: Stop Docker containers
docker-down:
	@echo "Stopping Docker containers..."
	@$(DOCKER_COMPOSE) down

## docker-build: Build Docker image for the service
docker-build:
	@echo "Building Docker image..."
	@$(DOCKER_COMPOSE) build routing

## docker-logs: Show logs from Docker containers
docker-logs:
	@$(DOCKER_COMPOSE) logs -f

## docker-reset: Reset Docker environment (removes volumes)
docker-reset: docker-down
	@echo "Resetting Docker environment..."
	@$(DOCKER_COMPOSE) down -v
	@$(DOCKER_COMPOSE) up -d postgres redis
	@sleep 5
	@make migrate-up

##@ Data Import

## import-osm-ca: Import California OSM data
import-osm-ca: build
	@echo "Importing California OSM data..."
	@./bin/$(IMPORTER_NAME) -url https://download.geofabrik.de/north-america/us/california-latest.osm.pbf

## import-osm: Import OSM data (usage: make import-osm file=/path/to/file.pbf)
import-osm: build
	@echo "Importing OSM data from $(file)..."
	@./bin/$(IMPORTER_NAME) -file $(file)

##@ Utilities

## deps-check: Check if required system dependencies are installed
deps-check:
	@echo "Checking system dependencies..."
	@which pkg-config > /dev/null 2>&1 || (echo "ERROR: pkg-config not found. Install with:" && echo "  Ubuntu/Debian: sudo apt-get install pkg-config" && echo "  macOS: brew install pkg-config" && exit 1)
	@pkg-config --exists zlib || (echo "ERROR: zlib not found. Install with:" && echo "  Ubuntu/Debian: sudo apt-get install zlib1g-dev" && echo "  macOS: brew install zlib" && exit 1)
	@echo "✓ All system dependencies are installed"

## install-tools: Install required development tools
install-tools:
	@echo "Installing development tools..."
	@$(GO) install github.com/pressly/goose/v3/cmd/goose@latest
	@$(GO) install github.com/air-verse/air@latest

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...

## lint: Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

## mod-tidy: Tidy Go modules
mod-tidy:
	@echo "Tidying modules..."
	@$(GO) mod tidy

## check-health: Check service health
check-health:
	@echo "Checking service health..."
	@curl -s http://localhost:8084/health | jq

## test-api: Test the distance API
test-api:
	@echo "Testing distance API..."
	@curl -s "http://localhost:8084/api/v1/route/distance?origin_zip=90001&dest_zip=94102&vehicle_type=truck" | jq
# Trenova Makefile
# Transportation Management System

# Variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint
GOVET=$(GOCMD) vet
KAFKA_SCRIPTS=./scripts/kafka
BASH_RUN=bash

# Binary names
CLI_BINARY=trenova-cli

# Directories
CLI_DIR=./cmd/cli
BUILD_DIR=./build

# Docker compose files
DOCKER_COMPOSE=docker-compose
DOCKER_COMPOSE_LOCAL=$(DOCKER_COMPOSE) -f docker-compose-local.yml

# Colors for output
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: all build clean test coverage lint fmt vet help

## help: Display this help message
help:
	@echo "Trenova - Transportation Management System"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "  ${GREEN}%-20s${NC} %s\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)


## build: Build the CLI binary
build: build-cli

## build-cli: Build the unified CLI tool (includes API, worker, and database commands)
build-cli:
	@echo "$(GREEN)Building unified CLI tool...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -o $(BUILD_DIR)/$(CLI_BINARY) ./cmd/cli
	@echo "$(GREEN)✓ Unified CLI tool built: $(BUILD_DIR)/$(CLI_BINARY)$(NC)"

## run: Run the API server via CLI
run: build-cli
	@echo "$(GREEN)Starting API server...$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) api run

## run-watch: Run the API server with hot reload
run-watch:
	@echo "$(GREEN)Starting API server with hot reload...$(NC)"
	@air api run

## run-worker: Run the worker service
run-worker:
	@echo "$(GREEN)Starting worker service...$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) worker run

## clean: Remove build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "$(GREEN)✓ Clean complete$(NC)"

## test: Run all tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	@$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	@$(GOTEST) -v -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report generated: coverage.html$(NC)"

## test-short: Run short tests
test-short:
	@echo "$(GREEN)Running short tests...$(NC)"
	@$(GOTEST) -short ./...

## test-race: Run tests with race detector
test-race:
	@echo "$(GREEN)Running tests with race detector...$(NC)"
	@$(GOTEST) -race ./...

## benchmark: Run benchmarks
benchmark:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	@$(GOTEST) -bench=. -benchmem ./...

## lint: Run linter
lint:
	@echo "$(GREEN)Running linter...$(NC)"
	@$(GOLINT) run ./...

## fmt: Format code
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	@$(GOFMT) -s -w .
	@echo "$(GREEN)✓ Code formatted$(NC)"

## vet: Run go vet
vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	@$(GOVET) ./...

## tidy: Tidy and verify module dependencies
tidy:
	@echo "$(GREEN)Tidying module dependencies...$(NC)"
	@$(GOMOD) tidy
	@$(GOMOD) verify
	@echo "$(GREEN)✓ Dependencies tidied$(NC)"

## deps: Download dependencies
deps:
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	@$(GOMOD) download
	@echo "$(GREEN)✓ Dependencies downloaded$(NC)"

## update: Update dependencies
update:
	@echo "$(GREEN)Updating dependencies...$(NC)"
	@$(GOGET) -u ./...
	@$(GOMOD) tidy
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

## generate: Run go generate
generate:
	@echo "$(GREEN)Running code generation...$(NC)"
	@$(GOCMD) generate ./...
	@echo "$(GREEN)✓ Code generation complete$(NC)"

## generate-seeds: Regenerate seed registry
generate-seeds:
	@echo "$(GREEN)Regenerating seed registry...$(NC)"
	@$(GOCMD) generate ./internal/infrastructure/database/seeder/...
	@echo "$(GREEN)✓ Seed registry regenerated$(NC)"

# Permission commands
## perm-generate: Generate permission manifest
perm-generate:
	@echo "$(GREEN)Generating permission manifest...$(NC)"
	@$(build_dir)/$(cli_binary) codegen generate
	@echo "$(GREEN)✓ Permission manifest generated$(NC)"

## perm-generate-enum: Generate permission enum from registry
perm-generate-enum:
	@echo "$(GREEN)Generating permission enum from registry...$(NC)"
	@$(build_dir)/$(cli_binary) codegen enum
	@echo "$(GREEN)✓ Permission enum generated$(NC)"

## perm-generate-types: Generate permission types from registry
perm-generate-types:
	@echo "$(GREEN)Generating permission types from registry...$(NC)"
	@$(build_dir)/$(cli_binary) codegen types
	@echo "$(GREEN)✓ Permission types generated$(NC)"

# Docker commands
## docker-up: Start all services with docker-compose
docker-up:
	@echo "$(GREEN)Starting Docker services...$(NC)"
	@$(DOCKER_COMPOSE_LOCAL) up -d
	@echo "$(GREEN)✓ Services started$(NC)"

## docker-down: Stop all services
docker-down:
	@echo "$(YELLOW)Stopping Docker services...$(NC)"
	@$(DOCKER_COMPOSE_LOCAL) down
	@echo "$(GREEN)✓ Services stopped$(NC)"

## docker-restart: Restart all services
docker-restart: docker-down docker-up

## docker-logs: View docker logs
docker-logs:
	@$(DOCKER_COMPOSE_LOCAL) logs -f

## docker-ps: List running containers
docker-ps:
	@$(DOCKER_COMPOSE_LOCAL) ps

## docker-build: Build docker images
docker-build:
	@echo "$(GREEN)Building Docker images...$(NC)"
	@$(DOCKER_COMPOSE_LOCAL) build
	@echo "$(GREEN)✓ Images built$(NC)"

# Database commands - ensure CLI is built first
## db-migrate: Run database migrations
db-migrate: build-cli
	@echo "$(GREEN)Running database migrations...$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) db migrate
	@echo "$(GREEN)✓ Migrations complete$(NC)"

## db-rollback: Rollback database migrations
db-rollback: build-cli
	@echo "$(YELLOW)Rolling back last migration...$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) db rollback
	@echo "$(GREEN)✓ Rollback complete$(NC)"

## db-seed: Seed the database
db-seed: build-cli
	@echo "$(GREEN)Seeding database...$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) db seed
	@echo "$(GREEN)✓ Database seeded$(NC)"

## db-reset: Reset database (drop, create, migrate, seed)
db-reset: build-cli
	@echo "$(RED)Resetting database...$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) db reset --force
	@echo "$(GREEN)✓ Database reset complete$(NC)"

## db-setup: Setup database (migrate + seed)
db-setup: build-cli
	@echo "$(GREEN)Setting up database...$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) db setup
	@echo "$(GREEN)✓ Database setup complete$(NC)"

## db-status: Show migration and seed status
db-status: build-cli
	@echo "$(GREEN)Database status:$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) db status

## db-create-seed: Create a new seed file
db-create-seed: build-cli
	@if [ -z "$(name)" ]; then \
		echo "$(RED)Error: Please provide a seed name$(NC)"; \
		echo "Usage: make db-create-seed name=my_seed_name [env=dev|test]"; \
		exit 1; \
	fi
	@if [ "$(env)" = "dev" ]; then \
		$(BUILD_DIR)/$(CLI_BINARY) db create-seed $(name) --dev; \
	elif [ "$(env)" = "test" ]; then \
		$(BUILD_DIR)/$(CLI_BINARY) db create-seed $(name) --test; \
	else \
		$(BUILD_DIR)/$(CLI_BINARY) db create-seed $(name); \
	fi
	@echo "$(GREEN)✓ Seed created$(NC)"

## db-seed-sync: Synchronize seed registry
db-seed-sync: build-cli
	@echo "$(GREEN)Synchronizing seed registry...$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) db seed-sync
	@echo "$(GREEN)✓ Registry synchronized$(NC)"

## db-seed-check: Check for orphaned seeds
db-seed-check: build-cli
	@echo "$(GREEN)Checking for orphaned seeds...$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) db seed-check

## db-seed-clean: Clean orphaned seed history
db-seed-clean: build-cli
	@echo "$(YELLOW)Cleaning orphaned seed history...$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) db seed-clean --force
	@echo "$(GREEN)✓ Orphaned seeds cleaned$(NC)"

## db-seed-watch: Watch seed directories for changes
db-seed-watch: build-cli
	@echo "$(GREEN)Watching seed directories...$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) db seed-watch

## redis-flush: Flush Redis cache
redis-flush: build-cli
	@echo "$(YELLOW)Flushing Redis cache...$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) redis flush
	@echo "$(GREEN)✓ Redis cache flushed$(NC)"

# Security commands
## security-check: Run security checks
security-check:
	@echo "$(GREEN)Running security checks...$(NC)"
	@gosec ./...
	@echo "$(GREEN)✓ Security check complete$(NC)"

## audit: Audit dependencies for vulnerabilities
audit:
	@echo "$(GREEN)Auditing dependencies...$(NC)"
	@$(GOCMD) list -json -m all | nancy sleuth
	@echo "$(GREEN)✓ Audit complete$(NC)"

# Development helpers
## watch: Run with hot reload (requires air)
watch:
	@echo "$(GREEN)Starting with hot reload...$(NC)"
	@air

## dev: Start development environment
dev: docker-up
	@echo "$(GREEN)Starting development environment...$(NC)"
	@$(MAKE) -j2 watch watch-ui

## watch-ui: Run UI with hot reload
watch-ui:
	@echo "$(GREEN)Starting UI with hot reload...$(NC)"
	@cd ui && npm run dev

# UI commands
## ui-install: Install UI dependencies
ui-install:
	@echo "$(GREEN)Installing UI dependencies...$(NC)"
	@cd ui && pnpm install
	@echo "$(GREEN)✓ UI dependencies installed$(NC)"

## ui-build: Build UI for production
ui-build:
	@echo "$(GREEN)Building UI for production...$(NC)"
	@cd ui && pnpm run build
	@echo "$(GREEN)✓ UI built$(NC)"

## ui-lint: Lint UI code
ui-lint:
	@echo "$(GREEN)Linting UI code...$(NC)"
	@cd ui && pnpm run lint
	@echo "$(GREEN)✓ UI lint complete$(NC)"

## ui-test: Run UI tests
ui-test:
	@echo "$(GREEN)Running UI tests...$(NC)"
	@cd ui && pnpm test
	@echo "$(GREEN)✓ UI tests complete$(NC)"

# Git hooks
## install-hooks: Install git hooks
install-hooks:
	@echo "$(GREEN)Installing git hooks...$(NC)"
	@cp .githooks/* .git/hooks/
	@chmod +x .git/hooks/*
	@echo "$(GREEN)✓ Git hooks installed$(NC)"

# CI/CD helpers
## ci: Run CI pipeline locally
ci: lint vet test security-check
	@echo "$(GREEN)✓ CI pipeline complete$(NC)"

## check: Run all checks (lint, vet, test)
check: lint vet test
	@echo "$(GREEN)✓ All checks passed$(NC)"

# Config commands
## config-validate: Validate configuration
config-validate: build-cli
	@echo "$(GREEN)Validating configuration...$(NC)"
	@$(BUILD_DIR)/$(CLI_BINARY) config validate
	@echo "$(GREEN)✓ Configuration valid$(NC)"

## config-show: Show current configuration
config-show: build-cli
	@$(BUILD_DIR)/$(CLI_BINARY) config show

# Version info
## version: Show version information
version:
	@echo "Trenova Transportation Management System"
	@echo "Version: $$(git describe --tags --always --dirty)"
	@echo "Go version: $$(go version)"
	@echo "Build date: $$(date -u +%Y-%m-%d_%H:%M:%S)"

# Tools installation
## tools: Install required tools
tools:
	@echo "$(GREEN)Installing required tools...$(NC)"
	@$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@$(GOCMD) install github.com/securego/gosec/v2/cmd/gosec@latest
	@$(GOCMD) install github.com/cosmtrek/air@latest
	@$(GOCMD) install github.com/sonatype-nexus-community/nancy@latest
	@echo "$(GREEN)✓ Tools installed$(NC)"

# Kafka commands
## kafka-reset: Reset Kafka
kafka-reset:
	@echo "$(GREEN)Resetting Kafka...$(NC)"
	$(BASH_RUN) $(KAFKA_SCRIPTS)/reset-kafka.sh
	@echo "$(GREEN)✓ Kafka reset complete$(NC)"

## kafka-setup: Setup Kafka
kafka-setup:
	@echo "$(GREEN)Setting up Kafka...$(NC)"
	$(BASH_RUN) $(KAFKA_SCRIPTS)/setup-debezium-connector.sh
	@echo "$(GREEN)✓ Kafka setup complete$(NC)"

## reset: Reset all services
reset-all: db-reset db-seed redis-flush kafka-reset kafka-setup
	@echo "$(GREEN)✓ All services reset$(NC)"

# Quick commands
## quick-start: Quick start for new developers
quick-start: deps tools docker-up db-setup
	@echo "$(GREEN)✓ Quick start complete!$(NC)"
	@echo ""
	@echo "To start the development server, run:"
	@echo "  make dev"
	@echo ""
	@echo "API will be available at: http://localhost:8080"
	@echo "UI will be available at: http://localhost:5173"

.DEFAULT_GOAL := help
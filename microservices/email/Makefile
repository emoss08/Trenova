.PHONY: build run test docker-build docker-up docker-down docker-logs test-email build-ui build-go dev ui-deps ui-dev

# Build the application
build: build-ui build-go

# Build the Go application
build-go:
	@echo "Building email microservice..."
	go build -o bin/email main.go

# Build the Svelte UI
build-ui:
	@echo "Building Svelte UI..."
	cd ui && npm install && npm run build

# Run the application
run: build
	@echo "Running email microservice..."
	./bin/email

# Run the microservice in development mode
dev: build-ui
	@echo "Running email microservice in development mode..."
	go run main.go

# Run the tests
test:
	go test ./...

# Build Docker image
docker-build:
	docker-compose build

# Start services with Docker Compose
docker-up:
	docker-compose up -d

# Stop Docker Compose services
docker-down:
	docker-compose down

# Show Docker Compose logs
docker-logs:
	docker-compose logs -f

# Send a test email
test-email:
	@if [ -z "$(to)" ]; then \
		echo "Error: 'to' parameter is required"; \
		echo "Usage: make test-email to=email@example.com [template=welcome] [subject=\"Test Email\"]"; \
		exit 1; \
	fi
	./scripts/send_test_email.sh --to $(to) \
		$(if $(template),--template $(template),) \
		$(if $(subject),--subject "$(subject)",)

# Install UI dependencies
ui-deps:
	@echo "Installing UI dependencies..."
	cd ui && npm install

# Start UI development server (for UI-only development)
ui-dev:
	@echo "Starting UI development server..."
	cd ui && npm run dev

# Help command
help:
	@echo "Available commands:"
	@echo "  make build         - Build the application"
	@echo "  make run           - Run the application"
	@echo "  make test          - Run tests"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-up     - Start services with Docker Compose"
	@echo "  make docker-down   - Stop Docker Compose services"
	@echo "  make docker-logs   - Show Docker Compose logs"
	@echo "  make test-email to=email@example.com [template=welcome] [subject=\"Test Email\"] - Send a test email"
	@echo "  make build-ui      - Build the Svelte UI"
	@echo "  make build-go      - Build the Go application"
	@echo "  make dev           - Run the application in development mode"
	@echo "  make ui-deps       - Install UI dependencies"
	@echo "  make ui-dev        - Start UI development server" 
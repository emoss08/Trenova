.PHONY: build run test docker-build docker-up docker-down docker-logs

# Build the application
build:
	go build -o workflow-service .

# Run the application
run:
	go run main.go

# Run the tests
test:
	go test ./...

# Build Docker image
docker-build:
	docker-compose build

# Start services with Docker Compose
docker-up:
	docker compose -f docker-compose.yml up -d

# Stop Docker Compose services
docker-down:
	docker compose down

# Show Docker Compose logs
docker-logs:
	docker compose logs -f

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
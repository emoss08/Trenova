.PHONY: build run test docker-build docker-up docker-down docker-logs test-email

# Build the application
build:
	go build -o email-service .

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
help: ## Show this help.
	@echo "usage: make \033[36m<target>\033[0m"
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

test: ## Run tests.
	gotest -v ./...

test-coverage: ## Generate test coverage report.
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

go-lint: ## Run linter.
	golangci-lint run --timeout 5m

go-format: ## Format the Go code.
	go fmt ./...

go-vet: ## Vet the Go code.
	go vet ./...

tidy: ## Tidy up the module.
	go mod tidy

build: ## Build the project.
	go build -o bin/ ./...

generate: ## Run go generate.
	go generate .

new-entity: ## Create a new entity. Usage: make new-entity ENTITY_NAME=<name>
	ifndef ENTITY_NAME
		$(error ENTITY_NAME is undefined. Usage: make new-entity ENTITY_NAME=entity_name)
	endif
	go run -mod=mod entgo.io/ent/cmd/ent new --target ./internal/ent/schema/ new ${ENTITY_NAME}

migrate-create: ## Create a new migration. Usage: make migrate-create MIGRATION_NAME=<name>
	ifndef MIGRATION_NAME
		$(error MIGRATION_NAME is undefined. Usage: make migrate-create MIGRATION_NAME=migration_name)
	endif
	atlas migrate diff ${MIGRATION_NAME} \
              --dir "file://internal/ent/migrate/migrations" \
              --to "ent://internal/ent/schema" \
              --dev-url "docker://postgres/15/test?search_path=public"

migrate-lint: ## Lint migrations.
	atlas migrate lint \
            --dev-url="docker://postgres/15/test?search_path=public" \
            --dir="file://internal/ent/migrate/migrations" \
            --latest=1

migrate-hash: ## Generate a hash for migrations.
	atlas migrate hash --dir file://internal/ent/migrate/migrations

migrate-status: ## Show migration status.
	atlas migrate status \
	--dir "file://internal/ent/migrate/migrations" \
	--url "postgresql://postgres:postgres@localhost:5432/trenova_go_db?sslmode=disable"

migrate-custom: ## Create a custom migration. Usage: make migrate-custom MIGRATION_NAME=<name>
	ifndef MIGRATION_NAME
		$(error MIGRATION_NAME is undefined. Usage: make migrate-custom MIGRATION_NAME=migration_name)
	endif
	atlas migrate new ${MIGRATION_NAME} \
      --dir "file://internal/ent/migrate/migrations"

migrate: ## Apply migrations.
	atlas migrate apply \
  --dir "file://internal/ent/migrate/migrations" \
  --url "postgresql://postgres:postgres@localhost:5432/trenova_go_db?sslmode=disable"

describe-schema: ## Describe the ENT schema.
	go run -mod=mod entgo.io/ent/cmd/ent describe ./ent/schema

run: ## Run the application.
	go run cmd/app.go

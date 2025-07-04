# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Trenova is an AI-driven transportation management system for trucking companies in the United States, built with Go backend, React/TypeScript frontend, and microservices architecture following hexagonal/clean architecture patterns.

## Essential Commands

### Backend Development (Go)

```bash
# Development
task run                    # Run main API server
task test                   # Run all Go tests
task test-pretty           # Run tests with formatted output
task format                # Format Go code
task lint                  # Run linting
task check                 # Run security checks

# Database
task db-reset              # Reset database
task db-migrate            # Run migrations
task db-seed               # Seed test data
task redis-flushall        # Clear Redis cache
```

### Frontend Development (React/TypeScript)

```bash
cd ui/
npm run dev                # Start Vite dev server (port 5173)
npm run build              # Build for production
npm run lint               # Run ESLint
npm run preview            # Preview production build
```

### Full Development Environment

```bash
# Start infrastructure (PostgreSQL, Redis, etc.)
docker-compose -f docker-compose-local.yml up -d

# Then run frontend and backend separately
cd ui && npm run dev       # Terminal 1
task run                   # Terminal 2
```

### Microservices

```bash
# Email service (Go + Svelte UI)
cd microservices/email/
make dev                   # Development
make build                 # Build
make test                  # Test
```

## Architecture

### Hexagonal Architecture Structure

- `internal/core/domain/` - Business entities and enums
- `internal/core/ports/` - Interface definitions (repositories, services)
- `internal/core/services/` - Business logic implementations
- `internal/infrastructure/` - External adapters (database, cache, messaging)
- `internal/api/handlers/` - HTTP handlers
- `internal/bootstrap/` - Dependency injection and app initialization

### Key Patterns

- **Dependency Injection**: Uses `uber-go/fx` for container management
- **Repository Pattern**: Database abstraction via interfaces
- **Service Layer**: Business logic separation
- **Event-Driven**: RabbitMQ for inter-service communication
- **Microservices**: Separate email and workflow services

### Frontend Architecture

- **State Management**: Zustand for global state, TanStack Query for server state
- **UI Components**: Custom components built on Radix UI primitives
- **Styling**: Tailwind CSS with custom design system
- **Routing**: React Router with protected routes

## Domain Concepts

The system manages transportation logistics including:

- **Shipments**: Core transport unit with moves, stops, and commodities
- **Equipment**: Tractors, trailers, and equipment types
- **Personnel**: Workers, drivers with compliance tracking
- **Customers**: Billing profiles and service configurations
- **Compliance**: Hazmat regulations, DOT requirements
- **Documents**: Upload, processing, and workflow management

## Technology Stack

### Backend

- **Framework**: Go Fiber (HTTP), Bun (ORM)
- **Database**: PostgreSQL with PGBouncer connection pooling
- **Cache**: Redis
- **Messaging**: RabbitMQ
- **Storage**: MinIO (S3-compatible)
- **Monitoring**: Prometheus, OpenTelemetry

### Frontend

- **Framework**: React 19 with TypeScript
- **Build**: Vite
- **UI**: Radix UI, Tailwind CSS, Lucide icons
- **Data**: TanStack Query, React Hook Form
- **Charts**: Recharts

### Infrastructure

- **Containerization**: Docker with multi-stage builds
- **Reverse Proxy**: Caddy with automatic HTTPS
- **Orchestration**: Docker Compose for local development

## Development Best Practices

- Always follow golang best practices for go 1.24

## Code Optimization Recommendations

- Instead of sync.WaitGroup use conc.WaitGroup("github.com/sourcegraph/conc")
- Use sonic instead of the standard library json package when possible

## Code Comments and Documentation

- When writing inline comments (comments inside functions) make sure to prefix them with _for example //_ this is a comment.
- If the comment is important and notes some type of behavior that may be unexpected or just an important note use the prefix ! for example // ! this is an important comment.

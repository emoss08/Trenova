# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Trenova is an AI-driven transportation management system for trucking companies in the United States, built with Go backend, React/TypeScript frontend, and microservices architecture following hexagonal/clean architecture patterns.

## Essential Commands

### Backend Development (Go)

```bash
# Development
make run                    # Run main API server
make test                   # Run all Go tests
make test-pretty           # Run tests with formatted output
make check                 # Run security checks

# Database
make db-reset              # Reset database
make db-migrate            # Run migrations
make db-seed               # Seed test data
make redis-flushall        # Clear Redis cache
```

### Frontend Development (React/TypeScript)

```bash
cd ui/
pnpm run dev                # Start Vite dev server (port 5173)
pnpm run build              # Build for production
pnpm run lint               # Run ESLint
pnpm run preview            # Preview production build
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
- **Event-Driven**: Kafka for inter-service communication
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

- **Framework**: Gin (HTTP), Bun (ORM)
- **Database**: PostgreSQL
- **Cache**: Redis
- **Messaging**: Kafka
- **Storage**: MinIO (S3-compatible)
- **Workflow Orchestration & Background Jobs**: Temporal
- **Monitoring**: Prometheus, OpenTelemetry

### Frontend

- **Framework**: React 19 with TypeScript & React Compiler
- **Build**: Vite
- **UI**: Radix UI, Tailwind CSS, Lucide icons, Fontawesome icons
- **Data**: TanStack Query, React Hook Form
- **Charts**: Recharts

### Infrastructure

- **Containerization**: Docker with multi-stage builds
- **Reverse Proxy**: Traefik with automatic HTTPS
- **Orchestration**: Docker Compose for local development

## Development Best Practices

- Always follow golang best practices for go 1.25+
- Ensure using correct syntax for go 1.25+ for example `interface{}` is `any` you can also discover this with the modernize tool that is in the editor

## Code Optimization Recommendations

- Instead of sync.WaitGroup use conc.WaitGroup("github.com/sourcegraph/conc")
- Use sonic instead of the standard library json package when possible

## Development Workflow

- Always check for errors in the current file before moving forward

## Implementation Guidelines

- Do not put in a placeholder note for implementation. If something needs to be done, implement the actual feature or functionality directly

## Development Warnings

- Never run npm run dev
- Never build the frontend with npm run buildj
- Never run npm run preview

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Trenova is a Transportation Management System (TMS) built as a Go monorepo with multiple services. The system uses PostgreSQL with PostGIS, Redis, and Meilisearch for search functionality.

## Repository Structure

```
trenova-2/
├── services/
│   ├── tms/                    # Main TMS service (API, worker, CLI)
│   ├── gtc/                    # CDC pipeline (Postgres → Redis/Meilisearch)
│   └── samsara-sim/            # Samsara simulator service
├── shared/                     # Shared Go packages across services
├── client/                     # Frontend Turborepo (apps/* + packages/*, pnpm)
├── go.work                     # Go workspace configuration
└── docker-compose-local.yml    # Local development infrastructure
```

## Common Commands

### TMS Service (from `services/tms/`)

```bash
task run-watch          # Run API server with hot reload (air)
task test               # Run unit tests
task test-integration   # Run integration tests (requires Docker)
task lint               # Run golangci-lint
task db-migrate         # Run database migrations
task db-seed            # Seed database (auto-regenerates seed IDs)
task db-reset           # Drop, create, migrate, and seed database
task docker-up          # Start infrastructure (postgres, redis, meilisearch, minio)
task docker-down        # Stop infrastructure
task quick-start        # Full setup for new developers
```

### Client (pnpm workspace + Turborepo, from `client/`)

The client is a Turborepo monorepo. `apps/*` are deployable applications and
`packages/*` are shared libraries consumed by name (`@trenova/*`).

```
client/
├── apps/
│   ├── web/          # @trenova/web  — main TMS application
│   └── dash/         # @trenova/dash — driver portal
└── packages/
    ├── shared/       # @trenova/shared  — design system (components/ui) + shared
    │                 #   lib/types/services/stores/hooks/styles used by both apps
    ├── graphql/      # @trenova/graphql — generated GraphQL client + codegen
    └── config/       # @trenova/config  — shared base tsconfigs
```

```bash
# From client/ — Turborepo runs the task across every app/package
pnpm dev              # turbo run dev (all apps)
pnpm build            # turbo run build
pnpm lint             # turbo run lint (oxlint)
pnpm typecheck        # turbo run typecheck (tsc -b)
pnpm test             # turbo run test (vitest)

# Scope to one project
pnpm --filter @trenova/dash dev        # run only the Dash dev server
pnpm --filter @trenova/web build       # build only the TMS app
```

GraphQL codegen lives in `@trenova/graphql` (`pnpm --filter @trenova/graphql codegen`).

**Shared code rule:** anything imported by more than one app — UI components,
utilities, types, base services — belongs in `packages/shared` and is imported
as `@trenova/shared/...`. App-specific (TMS-only) code stays in `apps/web`.
Never reach across apps or import an app package from another app.

### Running a Single Test

```bash
# Go
go test -v -run TestFunctionName ./path/to/package
go test -v -run TestFunctionName ./internal/core/services/organization/...

# Client
pnpm vitest run src/path/to/file.test.ts
```

## Architecture

### TMS Service - Hexagonal/Ports & Adapters with DDD

```
services/tms/
├── cmd/cli/                    # CLI entry point (Cobra)
│   ├── api/                    # API server command
│   └── db/                     # Database management commands
├── internal/
│   ├── api/                    # HTTP layer (Gin framework)
│   │   ├── handlers/           # Request handlers
│   │   ├── middleware/         # HTTP middleware
│   │   └── router.go           # Route registration
│   ├── bootstrap/              # Uber FX dependency injection modules
│   ├── core/                   # Business logic (pure domain)
│   │   ├── domain/             # Domain entities and value objects
│   │   ├── ports/              # Interface definitions
│   │   │   ├── repositories/   # Data access contracts
│   │   │   └── services/       # Service port definitions
│   │   └── services/           # Business logic implementations
│   └── infrastructure/         # Technical implementations
│       ├── postgres/           # PostgreSQL adapters (Bun ORM)
│       ├── redis/              # Redis cache adapters
│       ├── database/           # Migrations and seeding
│       └── config/             # Configuration management
└── pkg/                        # Public packages
    ├── errortypes/             # Structured error handling
    ├── domaintypes/            # Shared domain types
    └── validationframework/    # Validation engine
```

### Key Patterns

- **Dependency Injection**: Uber FX for compile-time DI with modular providers
- **ORM**: Bun (lightweight SQL builder on pgx)
- **IDs**: PULID (Prefix-based ULID) for distributed unique identifiers
- **Validation**: Ozzo validation + custom validation framework
- **Error Handling**: Structured errors with field-level validation (see `pkg/errortypes/`)

### Request Flow

```
HTTP Request → Handler → Service → Validator → Repository → Database/Cache
```

## Database Seeding

The seeding system supports environment-aware seeding with dependency management:

```bash
task db-seed                           # Seed for current environment
task db-seed env=development           # Override environment
task db-create-seed name=MySeed        # Create new base seed
task db-create-seed name=MySeed env=dev  # Create development seed
task generate-seeds                    # Regenerate seed ID constants
```

Seeds are registered in `internal/infrastructure/database/seeder/seeds/register.go`. Typed `SeedID` constants are auto-generated in `pkg/seedhelpers/seed_ids_gen.go`.

## Infrastructure (Docker Compose)

Local development uses `docker-compose-local.yml`:

| Service | Port | Purpose |
|---------|------|---------|
| PostgreSQL (PostGIS) | 5432 | Primary database |
| Redis 8 | 6379 | Cache + JSON/stream support |
| Redis Insight | 5540 | Redis inspection UI |
| Meilisearch | 7700 | Full-text search |
| MinIO | 9000, 9001 | Object storage |
| GTC | - | CDC pipeline (Postgres → Redis/Meilisearch) |

## Error Handling

Use the `errortypes` package for structured errors that map to HTTP status codes and frontend forms:

```go
multiErr := errortypes.NewMultiError()
multiErr.Add("email", errortypes.ErrRequired, "Email is required")
if multiErr.HasErrors() {
    return multiErr  // Returns 422 with field-level errors
}
```

Supports nested paths (`user.address.street`) and array indices (`items[0].name`).

## Code Style

### General Principles
- **Production-grade, fully featured code**: This is an enterprise application. Never write "v1", "MVP", or simplified versions of a feature. Every feature must be implemented completely — no stubs, no shortcuts, no "can be improved later" placeholders. If a feature needs error handling, edge cases, validation, proper UX states, or integration with existing systems, implement all of it in the first pass. Do not simplify or reduce scope unless explicitly told to.
- **Secure and correct**: All code must be secure (no injection vectors, no unvalidated input, no leaked secrets, proper authz checks) and free of bugs — handle every error path and edge case explicitly.
- **DRY**: Do not repeat yourself — extract shared logic rather than duplicating code
- **SOLID**: Follow SOLID principles strictly (single responsibility, open/closed, Liskov substitution, interface segregation, dependency inversion)
- **Performance**: Write the most efficient and performant code possible — avoid unnecessary allocations, prefer stack over heap, minimize copies, use appropriate data structures
- **Utility functions**: Never duplicate a utility — if a function that does the same thing already exists, reuse it. Backend: place reusable utilities in the `shared/` package (e.g., `shared/stringutils`, `shared/sliceutils`, `shared/intutils`); do NOT scatter utility/helper functions in domain or service files; if a utility package doesn't exist for the category, create one in `shared/`. Frontend: utilities shared across apps go in `client/packages/shared/src/lib/` (`utils.ts`, `date.ts`, etc.) and are imported as `@trenova/shared/lib/*`; app-only utilities go in that app's `src/lib/`. Do NOT define them inline in components, hooks, or routes, and do NOT duplicate a utility that already exists in `@trenova/shared`

### Go
- Follow the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) as the baseline for all Go code
- Do not add comments to code
- Use Bun ORM for database operations
- Use Ozzo validation for struct validation
- Follow hexagonal architecture — keep domain logic in `core/`, adapters in `infrastructure/`
- Use `sonic` for JSON — `encoding/json` is forbidden by lint
- Format with `goimports` and `golines`
- Use `t.Context()` instead of `context.Background()` in tests (Go 1.25+)
- When a function signature exceeds ~3-4 parameters, group them into a named struct type (e.g., `type CreateShipmentParams struct { ... }`)
- Prefer value receivers unless the method mutates state or the struct is large
- Avoid `interface{}` / `any` when a concrete or generic type is possible
- Use `errors.New` / `fmt.Errorf` with `%w` — never discard errors silently
- Preallocate slices/maps when the size is known (`make([]T, 0, n)`)

### TypeScript/React
- Format with OxFmt (`client/.oxfmtrc.json`): double quotes, semicolons, trailing commas, 100 char width
- Lint with OxLint (`client/.oxlintrc.json`): strict React hooks, TanStack Query exhaustive deps, no console.log
- Prefer named exports over default exports
- Extract repeated logic into custom hooks or shared utilities

## Bun ORM

For help with Bun ORM, look in the [docs](docs/bun/).
When writing repositories, always use the generated column helpers in `services/tms/pkg/buncolgen/` — never hand-write column references. Read [docs/bun/buncolgen.md](docs/bun/buncolgen.md) for the full method reference, canonical repository patterns, and regeneration workflow before writing any repository code.

## DO NOT
- **Processes**: Do not run high usage tasks that will max out CPU, Disk and/or memory usage.
- **Mockery**: Do not run mockery against the entire codebase — manually adjust mocks in the codebase.


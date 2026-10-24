# Backend patterns

How Go code in `services/tms` and `shared/` is written. [AGENTS.md](../../../AGENTS.md) sets the
stance (scale, no backwards compatibility, complete features, disagree when you have something
better); these files are how that turns into code. Read the file for each layer you touch before
writing it, then check the subsystem's own doc in the [index](../README.md).

| File | Covers |
|---|---|
| [scale.md](scale.md) | the checklist every read, write, job and integration passes |
| [data-access.md](data-access.md) | entities, repositories, pagination, locking, migrations, caching, seeding |
| [keel-migration.md](keel-migration.md) | the Bun-to-keel move: shared pools and transactions, `Keel(ctx)`, error mapping, what converted repositories do |
| [keel-conversion.md](keel-conversion.md) | the playbook for converting a module: every repository pattern in Bun and in keel, `pkg/listquery`, validators, tests, the checklist |
| [services.md](services.md) | application services, validation, errors, authorization, audit, after-commit work |
| [api.md](api.md) | GraphQL schema, resolvers, dataloaders, limits, REST handlers |
| [jobs.md](jobs.md) | Temporal workflows, activities, schedules, sweeps |
| [testing.md](testing.md) | unit and integration tests, fixtures, running them |
| [go.md](go.md) | Go style, utilities, generics and build time, dependencies |
| [module-architecture.md](../module-architecture.md) | modules, levels, ports, where a new package goes |

## Where code goes

A request flows handler or resolver → service → validator → repository → Postgres, Redis or
Temporal. Each business area is a module, `internal/modules/<module>/`:

| Layer | Package | Holds |
|---|---|---|
| Domain | `domain/<entity>/` | Bun models, value objects, enums, entity rules; no I/O |
| Ports | `ports/<entity>ports/` | repository and service interfaces, request and result types |
| Application | `app/<name>service/` | use cases, validators, plans |
| Adapters | `adapters/postgres/<name>repository/` | repositories |
| Jobs | `jobs/<name>jobs/`, `<name>contract/` | workflows, activities, schedules; the contract others import |
| Edge | `internal/api/{graphql,handlers}` | resolvers and handlers: authorize, map, call a service |
| Wiring | `module.go`, `jobs/jobs.go` | fx options for shared, API and worker graphs |
| Technical adapters | `internal/infrastructure/*` | Postgres, Redis, Temporal, imaging, config, observability |
| Cross-module types | `pkg/*` | `errortypes`, `validationframework`, `pagination`, `querybuilder`, `dbhelper`, … |
| Domain-free utilities | `shared/*` | string, slice, map, time, money, pulid, vendor clients |

A module imports only strictly lower modules' `domain`, `ports` and `<name>contract`; an upward
need is a port declared low and provided by the higher module.

## Adding an entity end to end

1. Domain model and enums (`data-access.md#entities`), migration with indexes
   (`data-access.md#migrations`), then regenerate column helpers (`buncolgen.md`).
2. Ports: repository interface with `GetByID`, `GetByIDs`, `ListConnection`, writes.
3. Repository (`data-access.md#repositories`) with an integration test that includes a second
   tenant.
4. Permission resource and operations in the registry (`permissions.md`), record kind if it is
   one (`record-kinds.md`).
5. Validator and service (`services.md`), audit included.
6. GraphQL schema, resolvers, loader, projection; REST only where GraphQL does not fit
   (`api.md`).
7. Agent tool or write-coverage entry (`agent-write-coverage.md`).
8. Client screen through the `trenova-frontend` skill; seeds if development needs data.

## Local infrastructure

`task docker-up` starts `dev/compose.yml`: PostgreSQL with PostGIS (5432), Redis 8 (6379), Redis
Insight (5540), Meilisearch (7700), MinIO (9000/9001), the imaging worker (gRPC 50051) and GTC.
`task db-reset` recreates the database from `trenova-devdb`'s migrated template, then reconciles
and seeds it.

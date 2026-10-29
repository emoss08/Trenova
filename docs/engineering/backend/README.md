# Backend patterns

How Go code in `services/tms` and `shared/` is written. [AGENTS.md](../../../AGENTS.md) sets the
stance (scale, no backwards compatibility, complete features, disagree when you have something
better); these files are how that turns into code. Read the file for each layer you touch before
writing it, then check the subsystem's own doc in the [index](../README.md).

| File | Covers |
|---|---|
| [scale.md](scale.md) | the checklist every read, write, job and integration passes |
| [data-access.md](data-access.md) | keel entities, repositories, lists (`pkg/listquery`), error mapping, pools and transactions, locking, migrations, caching, seeding |
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
| Domain | `domain/<entity>/` | keel models, value objects, enums, entity rules; no I/O |
| Ports | `ports/<entity>ports/` | repository and service interfaces, request and result types |
| Application | `app/<name>service/` | use cases, validators, plans |
| Adapters | `adapters/postgres/<name>repository/` | repositories |
| Jobs | `jobs/<name>jobs/`, `<name>contract/` | workflows, activities, schedules; the contract others import |
| Edge | `internal/api/{graphql,handlers}` | resolvers and handlers: authorize, map, call a service |
| Wiring | `module.go`, `jobs/jobs.go` | fx options for shared, API and worker graphs |
| Technical adapters | `internal/infrastructure/*` | Postgres, Redis, Temporal, imaging, config, observability |
| Cross-module types | `pkg/*` | `errortypes`, `validationframework`, `pagination`, `listquery`, `dberror`, … |
| Domain-free utilities | `shared/*` | string, slice, map, time, money, pulid, vendor clients |

A module imports only strictly lower modules' `domain`, `ports` and `<name>contract`; an upward
need is a port declared low and provided by the higher module.

## Adding an entity end to end

1. Domain model and enums (`data-access.md#entities`), migration with indexes
   (`data-access.md#migrations`), then regenerate the keel tables (`task keel-gen`).
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

## Configuration

`internal/infrastructure/config` holds the root `config.Config`, the loader (files, `.env`, the
`TRENOVA_` environment) and the validation order. Each config section is its own package under it,
`config/<section>config` (`databaseconfig`, `aiconfig`, `securityconfig`, ...), holding the
section's `Config`, its `Defaults()` (keys relative to the section), its rules (`Validate`,
`ValidateProduction`) and its getters. The root imports the sections; a section never imports the
root.

- **Consumers take their section from fx, not the root.** A constructor names the section in its
  `Params` (`AIConfig *aiconfig.Config`, `StorageConfig *storageconfig.Config`), which
  `config.SectionProviders` supplies, so an edit to one section recompiles only the code that
  reads it (`infrastructure/minio/client.go`). Take `*config.Config` only for work that spans
  sections, such as `GetDSN` or `PublicLinkBaseURL`.
- **A hand-built fx graph** (a CLI command, a wiring test) that supplies `*config.Config` also
  adds `config.SectionProviders`.
- **A new section** is a package with `Config`, `Defaults()` and `Validate()` when it has rules, a
  field on `config.Config`, its line in `setDefaults` and `validateConfig` (`loader.go`) and its
  provider in `sectionproviders.go`.
- Edition sections (`config.Section`, registered through `config.SectionsOption`) are how the
  Cloud overlay adds paths the public build does not own; they decode into `Config.Extension`.

## Local infrastructure

`task docker-up` starts `dev/compose.yml`: PostgreSQL with PostGIS (5432), Redis 8 (6379), Redis
Insight (5540), Meilisearch (7700), MinIO (9000/9001), the imaging worker (gRPC 50051) and GTC.
`task db-reset` recreates the database from `trenova-devdb`'s migrated template, then reconciles
and seeds it.

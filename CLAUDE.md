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
task gqlgen             # Regenerate GraphQL server code from internal/api/graphql/schema/*.graphqls
task gqlschema-diff     # Report GraphQL schema changes vs origin/master; fails on breaking changes
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

## Design System

Tokens live in `client/packages/shared/src/styles/tokens.css`; components consume them and
define no colour, size or elevation of their own. **Read
[docs/engineering/design-system.md](docs/engineering/design-system.md) before writing styles.**

`pnpm lint:design` runs in CI and fails on the seven ways the old set was bypassed: raw
Tailwind palette classes (`bg-red-500`), arbitrary font sizes (`text-[11px]` — `text-xs` *is*
11px and brings a line-height), hex colours in `className`/`style`, hand-rolled focus rings
(`focus-visible:ring-*` — use `ui-focus-ring`), any `shadow-*` class or inline `boxShadow` that draws a shadow, and
retired Badge variants. Each message
names the token to use instead.

- Colour is a **tone** (`neutral`/`brand`/`info`/`success`/`warning`/`danger`) when the set has
  a severity ordering, or a **categorical accent** (`accent-teal`, `accent-violet`, …) when it
  does not — a HOS duty status and a pricing method are categories, not severities.
- Status maps declare a lifecycle **phase** (`draft`/`queued`/`active`/`awaiting`/`attention`/
  `complete`/`closed`/`failed`) and the tone follows, so a new status cannot pick a colour.
- The product is drawn in ink: the primary button is `--ink` (the foreground colour), never
  a hue. Every grey carries `--hue-neutral` (cool slate, 260). The brand is cobalt at 262
  and is spent only on links, focus, selection and the active nav row or tab. The blue arc
  is pinned apart — info 222 → sky 232 → brand 262 → indigo 283 — and so is the warm one —
  danger 25 → warning 78 → amber 98.
- Labels are sentence case; no `uppercase tracking-wider` on section labels, column heads
  or badges. Form controls spend `ui-field`, filled controls `ui-press`, skeletons
  `ui-shimmer`. Motion answers an action — nothing loops on a working screen.
- **Nothing that sits *in* the page draws a shadow.** The `--elevation-*` tokens are all
  flat and must stay that way; separate surfaces with a border or `ring-1
  ring-foreground/10`. Zero-blur outlines (`shadow-[0_0_0_1px_var(--brand)]`) and the focus
  ring are lines, not shadows, and are fine. The few things that float *above* the page — a
  floating composer, a card in flight, a menu — spend `ui-lift-whisper` / `ui-lift` /
  `ui-lift-float`, named by where they are allowed rather than by size. A table row with a
  lift is still the defect the rule was written against.
- Everything that floats from a trigger (menus, selects, popovers, hover cards, tooltips) is
  inverted — dark in light mode. The primitives set `dark` on the positioner; never write it
  by hand, and build popover content from tokens only. Dialogs and sheets follow the theme.
- Form controls are filled: `--field` never matches the canvas or the card. Every control
  spends `ui-field` (rest, hover, open and disabled in one place); a `Button`-built trigger
  takes `fieldTriggerClass` and an invalid one `fieldInvalidClass`, both from
  `@trenova/shared/lib/variants/field`. Never hand-write `border-input bg-muted` or a
  `data-pressed:ring-*` on a field.
- Pages: always `PageLayout` with `pageHeaderProps` (never mount `PageHeader` by hand, never
  `p-0` plus re-added `mx-4`). A list page is `PageLayout` > `DataTableLazyComponent` > table
  with no wrapper `div`, so the table bleeds edge to edge. Create buttons read "New {thing}".
  Figures are `KpiStrip`/`KpiStripItem`, titled blocks are `SectionPanel`, read-only
  label/value is `DescriptionList`, inline callouts are `<Alert size="sm">`, and dialogs take
  `size` rather than an arbitrary max-width. See "Page anatomy" in the design-system doc.
- Machine suggestions are marked with `AssistMark` from `@trenova/shared/components/ui/assist-mark`
  (a real `LucideIcon`). Never import `Sparkles`, `WandSparkles` or `Wand2`.
- Two radii: `--radius-control` (6px) for controls, `--radius-surface` (8px) for containers.
  The whole `rounded-*` scale points at them; a badge is `rounded-full`.
- Weight means something: 400 body, 500 label, 600 heading. A value in a cell takes no
  weight class at all.
- Row rhythm is `--row-h` / `--row-h-compact` / `--row-head-h` / `--cell-px`. A denser table
  repoints the token; it does not patch padding onto every cell.
- Missing colour? Add a token to `tokens.css` — in **both** `:root` and `.dark` — rather than
  reaching for the palette. An incomplete set is what caused the drift in the first place.
- Genuine exceptions take `design-tokens-ignore: <reason>` in a comment on or above the line.
- `pnpm lint:design` also reads the oklch values out of `tokens.css` and fails on a contrast
  pair below AA, a value outside sRGB, a hairline invisible against its surface, or either
  hue arc closing up. `--foreground-on-solid` is near-white in light and near-**black** in dark,
  because a dark theme's tone fills are the light end of their ramp. Text on a solid
  warning is `text-warning-on-solid` in both themes, because the solid is a true amber.
- Editing `tokens.css`: never let a comment-terminator sequence appear inside a comment body.
  It ends the comment early and every `@utility` after it silently stops emitting — which
  removes focus indicators without failing anything. `pnpm lint:design` compiles the file and
  asserts each declared `@utility` reaches the output.

## Generated Code

Several generators in this repo fail on code that compiles and passes every test, and the
CI failure names the command without saying why the generator refused. **Before changing a
GraphQL schema, removing a database column, or editing user-facing text, read
[docs/engineering/generated-artifacts.md](docs/engineering/generated-artifacts.md).** It
lists the five `Codegen Checks` steps with the exact commands to run them locally, and the
traps that have actually broken `master` — most often `projection.yml`, which requires every
GraphQL field to resolve to a column, an inferred relation, or a declared override, so
retiring a column while keeping its deprecated field breaks the build until the field is
added to `virtuals`.

## Product Guide

Agents answer "where is…" and "how do I…" from a catalog generated from the web app and the
guides in `docs/product-guide/`. **Adding a page, renaming a button a guide names, or changing a
record link means regenerating it** (`pnpm --filter @trenova/web guide:generate`); CI fails
otherwise. Record links are built from one registry, `client/apps/web/src/config/record-links.ts`
(`recordPath`), never by hand. Read [docs/engineering/product-guide.md](docs/engineering/product-guide.md).

## GraphQL (gqlgen)

- Schema lives in `services/tms/internal/api/graphql/schema/*.graphqls`; regenerate with `task gqlgen` (it retries once, because gqlgen can miss the `models_gen.go` it just wrote when a schema adds a model).
- When a domain struct field uses an initialism (`ShipmentBOL`, `DOTNumber`, `PTOType`), add the initialism to `go_initialisms` in `gqlgen.yml` rather than a per-field `fieldName:` override. Keep initialisms to 3+ letters: 2-letter ones (`AR`, `PO`, `MC`, `CC`) prefix-match SCREAMING enum values and mangle their Go constants (`DETENTION_POLICY` became `DetentionPOLicy`), so those few fields keep explicit overrides.
- Bind GraphQL enums to their domain type in `gqlgen.yml` (`RotaDayState: model: ...worker.RotaDayState`) when the string values match; that removes the need for a conversion resolver.
- Every root resolver must reach a permission check or be listed, with a reason, in `internal/api/graphql/authzlint`; self-scoped operations are named `My<Thing>`. `task gqlschema-diff` fails on breaking schema changes; swapping a field between wire-compatible scalars (`Int`↔`Timestamp`, `String`↔`Decimal`) is reported as dangerous, not breaking.
- Use the semantic scalars from `shared.graphqls`: `Timestamp` for Unix-second instants (never bare `Int` for a `*At`/`*Date` field) and `Decimal` for money and other exact quantities (never `String` or `Float`). Both bind to the same Go types as `Int`/`String`, so no mapper changes are needed.
- A field resolver that fetches per parent row must go through a per-request dataloader in `internal/api/graphql/loaders` (`batchByIDFunc` for entities, `batchCountFunc` for counts, `batchGroupFunc` for child lists) backed by a `...ByIDs` repository method; never call a single-ID getter from an object-type resolver. Register the factory in `loaders/loaders.go` and `bootstrap/modules/api/graphql.go`.
- Connection repositories only run their `COUNT` when `req.Cursor.IncludeTotalCount` is set; the GraphQL layer clears it when `totalCount` is not selected. New `ListConnection` methods must follow that gate.
- Patch inputs (`*PatchInput`) mark fields `@goField(omittable: true)`: absent leaves the value alone, explicit `null` clears it (or fails validation for fields the entity requires). Generated mappers come from `resolver/mappergen`.
- Clients send persisted operations by hash only; outside production the server re-reads `persisted-documents.json` on an unknown hash (`security.graphql.persistedDocumentsPath`), so run `pnpm --filter @trenova/graphql codegen` (or `pnpm dev`, which watches) after editing an operation. `/graphql` enforces a body-size limit and a per-user operation-cost budget (`security.graphql.*`).
- Run GraphQL package tests with `go test -tags nofitz ./internal/api/graphql/...` on machines without `libmupdf`.

## Agent Runtime

How every AI feature executes on Temporal — the queues and why they are split,
the agent loop in workflow code (one activity per model and tool call), the
HTTP-derived retry mapping, the step ledger that makes a retried write safe, the
turn stream, per-agent schedules, the one-shot and batch workflows, and the
`GetVersion` branches waiting on in-flight executions to drain.
**Read [docs/engineering/agent-runtime.md](docs/engineering/agent-runtime.md)
before changing anything under `internal/core/services/agentruntime/`,
`assistantservice/`, or `internal/core/temporaljobs/{agentflow,modelcall,agentjobs,assistantjobs,completionjobs,importassistantjobs}/`.**
Extensions (web search and anything like it) add agent-only tools that an organization turns
on with its own vendor account; how they are gated, metered, and why a turn that read outside
content proposes every later write, is in
[docs/engineering/agent-extensions.md](docs/engineering/agent-extensions.md).
An agent handing a task to another agent (`delegate_task`, the per-agent
allowlist, one level only, same person) is described in
[docs/engineering/agent-delegation.md](docs/engineering/agent-delegation.md).

## Realtime

Live updates are server-sent events from the API, fanned out through sharded Redis
Streams; there is no third-party realtime vendor. Publish with
`RealtimeService.PublishResourceInvalidation` (it only queues, so it is safe on hot
paths) and set `AudienceUserID` for anything addressed to one person. **Read
[docs/engineering/realtime.md](docs/engineering/realtime.md) before changing
`realtimeservice`, `infrastructure/realtimebroker`, the stream endpoint, or the
browser `realtimeClient`**, and before adding a presence or typing scope.

## AI Audit Trail

`ai_audit_events` is an append-only, per-tenant hash chain, signed with keys kept outside the
database (`aiAudit.chain.*`). One projector writes it from rows the agent runtime already
records; nothing else may write it. **Read
[docs/engineering/ai-audit-trail.md](docs/engineering/ai-audit-trail.md) before changing
`aiauditservice`, `aiauditjobs`, `domain/aiaudit` (the canonical form is hashed, so a field
change is a `hash_version` change), or any agent source table the projector reads.** A
retention sweep that deletes agent rows must stay behind `SourcePruneHorizon`. Never remove a
chain key while rows signed with it are retained.

## Bun ORM

For help with Bun ORM, look in the [docs](docs/bun/).
When writing repositories, always use the generated column helpers in `services/tms/pkg/buncolgen/` — never hand-write column references. Read [docs/bun/buncolgen.md](docs/bun/buncolgen.md) for the full method reference, canonical repository patterns, and regeneration workflow before writing any repository code.

## DO NOT
- **Processes**: Do not run high usage tasks that will max out CPU, Disk and/or memory usage.
- **Mockery**: Do not run mockery against the entire codebase — manually adjust mocks in the codebase.


# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Trenova is a Transportation Management System (TMS) built as a Go monorepo with multiple services. The system uses PostgreSQL with PostGIS, Redis, and Meilisearch for search functionality.

## Repository Structure

```
trenova/
├── services/tms/          # Main TMS service (API, worker, CLI)
├── client/                # pnpm + Turborepo: apps/web, apps/dash, packages/*
├── desktop/capture/       # Windows scanning and printing companion (Rust)
├── shared/                # Shared Go packages (its own module in go.work)
├── i18n/                  # Translation catalogs and their tooling
├── dev/                   # Local environment: compose.yml, config/, samsara-sim, edi-partner-sim
├── deploy/                # Dockerfiles, self-host compose, Caddy, GTC routing, observability
├── tools/                 # Not shipped: extraction-finetune (Python), screenshots, smack, release
├── docs/                  # engineering/, product-guide/, self-hosting/, notes/
└── go.work                # Go workspace: services/tms, shared, dev/samsara-sim, dev/edi-partner-sim
```

GTC, the change-data-capture pipeline, is a separate project (`github.com/emoss08/gtc`); this
repository runs its image and keeps its routing in `deploy/gtc/sinks.yaml`. Each top-level
directory names a role, not a language: put a new project where its role is, and keep `dev/`
for what only a developer's machine runs and `tools/` for what is never shipped.

## Common Commands

### TMS Service (from `services/tms/`)

```bash
task run-watch          # Run API server with hot reload (air)
task test               # Run unit tests
task test-integration   # Run integration tests (requires Docker)
task lint               # Run golangci-lint
task gqlgen             # Regenerate GraphQL models, resolver stubs and the sharded executor (never `go tool gqlgen generate`)
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
go test -v -run TestFunctionName ./internal/modules/identity/app/organizationservice/...

# Client
pnpm vitest run src/path/to/file.test.ts
```

## Architecture

### TMS Service - Hexagonal/Ports & Adapters with DDD

```
services/tms/
├── cmd/
│   ├── cli/                    # CLI entry point (Cobra): api run, worker run, db, redis, ...
│   ├── api/                    # API-only binary
│   └── worker/                 # Worker-only binary
├── internal/
│   ├── api/                    # HTTP layer (Gin framework) and GraphQL: inbound adapters
│   │   ├── handlers/           # Request handlers
│   │   ├── middleware/         # HTTP middleware
│   │   └── router.go           # Route registration
│   ├── bootstrap/              # Uber FX graphs: core (shared), apiapp, workerapp
│   ├── modules/<module>/       # One hexagon per business module
│   │   ├── domain/             # Domain entities and value objects
│   │   ├── ports/<entity>ports # Repository and service interfaces per entity
│   │   ├── app/                # Application services
│   │   ├── adapters/postgres/  # Repositories
│   │   ├── jobs/               # Temporal workflows and activities; <name>contract packages
│   │   └── module.go           # The module's fx options
│   └── infrastructure/         # Cross-cutting technical adapters
│       ├── postgres/           # PostgreSQL adapters (Bun ORM)
│       ├── temporaljobs/       # Temporal client, worker runtime, schedules, workflow starter
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

Local development uses `dev/compose.yml`:

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
- Follow the modular hexagonal architecture — domain, ports, app, adapters and jobs live in their module under `internal/modules/<module>/` (see "Modules" below)
- Use `sonic` for JSON — `encoding/json` is forbidden by lint
- Format with `goimports` and `golines`
- Use `t.Context()` instead of `context.Background()` in tests (Go 1.25+)
- When a function signature exceeds ~3-4 parameters, group them into a named struct type (e.g., `type CreateShipmentParams struct { ... }`)
- Prefer value receivers unless the method mutates state or the struct is large
- Avoid `interface{}` / `any` when a concrete or generic type is possible
- Use `errors.New` / `fmt.Errorf` with `%w` — never discard errors silently
- Preallocate slices/maps when the size is known (`make([]T, 0, n)`)
- Never let an exported type, or the signature of an exported function or method, mention a concrete generic type that has methods (`*validationframework.TenantedValidator[*X]`, `*dataloadgen.Loader[K, V]`). Go compiles that type's methods again in every package that can see it, so hold it behind a generic interface (`validationframework.TenantedEntityValidator[*X]`) or in an unexported type reached only through package-level functions; an unexported field of an exported type still spreads it, and so does a pointer to an unexported struct that holds one. `sync.Map` counts: it is a generic hash trie underneath, so a widely reachable type uses a plain map behind a mutex instead. See "Generics and build time" in [docs/engineering/module-architecture.md](docs/engineering/module-architecture.md).

### TypeScript/React
- **Use the `trenova-frontend` skill** (`.claude/skills/trenova-frontend/`) for any change under
  `client/`: where code goes, page and loading anatomy, data and query keys, forms, i18n,
  styling, bundle discipline, tests and what to verify. When one of its rules is wrong, fix the
  rule in the same change.
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
  a hue. The palette is Vercel's Geist: every grey is a true neutral (chroma 0), the canvas
  is #fafafa under white panels, and the ink is #171717. The brand is Geist blue at 258 and
  is spent only on links, focus and selection; the active nav row is a neutral fill and the
  active tab is underlined in ink. Every colour is a step on a Geist ramp (200 tint, 400
  hairline, 700 solid, 900 text); info is Geist blue itself, as on Vercel. Sky and indigo stay
  20° from the brand, and the warm arc is pinned apart — danger 25 → warning 76 → amber 98.
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
- Form controls are filled: `--field` never matches the canvas. It is Vercel's panel fill in
  both themes (white in light, #0a0a0a in dark), and the `--input` hairline carries it on a
  panel. Every control spends `ui-field` (rest, hover, open and disabled in
  one place); a `Button`-built trigger takes `fieldTriggerClass` and an invalid one `fieldInvalidClass`, both from
  `@trenova/shared/lib/variants/field`. Never hand-write `border-input bg-muted` or a
  `data-pressed:ring-*` on a field.
- The shell is one layout: a 48px module rail, the current module's panel beside it, and the
  page's own title bar as the only bar above the body. There is no global header; a page never
  draws its own top bar, and `PageHeader`'s `leading`/`trailing` slots belong to the shell. See
  "Shell anatomy" in the design-system doc.
- Pages: always `PageLayout` with `pageHeaderProps` (never mount `PageHeader` by hand, never
  `p-0` plus re-added `mx-4`). A list page is `PageLayout` > `DataTableLazyComponent` > table
  with no wrapper `div`, so the table bleeds edge to edge. Create buttons read "New {thing}".
  Figures are `KpiStrip`/`KpiStripItem`, titled blocks are `SectionPanel`, read-only
  label/value is `DescriptionList`, inline callouts are `<Alert size="sm">`, and dialogs take
  `size` rather than an arbitrary max-width. See "Page anatomy" in the design-system doc.
- Toasts are for news the screen cannot show: a failure with nowhere else to go, work that
  finished out of sight, a bulk outcome, or a result with a next step (Undo, Download, Open).
  A save, delete, toggle or approval the person watches happen takes no success toast, and a
  realtime notification toasts only when urgent or when it delivers what the person asked
  for. See "Toasts" in the design-system doc.
- Loading states are the layout of what loads. `DataTableLazyComponent` wraps only components
  that render `DataTable`; everything else is `LazyComponent` with a required `fallback` that is
  its own `<name>-skeleton.tsx`, and the component's own data-loading state draws the same
  skeleton. `pnpm lint:skeletons` runs in CI and fails a table skeleton around anything that is
  not a table, or a `<Suspense>` without a fallback. See "Loading states" in the design-system doc.
- Icons come from `@trenova/shared/components/icons`: every Untitled UI icon as `<Name>Icon`
  (`Truck01Icon`, `SearchLgIcon`) plus Trenova's custom set drawn on the same grid, typed
  `IconComponent`. Never import `lucide-react` or `@untitledui/icons` directly; lint refuses both.
  A missing icon is drawn into the custom set, not borrowed from another library. See "Icons" in the
  design-system doc.
- Machine suggestions are marked with `AssistMark` from `@trenova/shared/components/ui/assist-mark`
  (a real `IconComponent`). The sparkle and magic-wand glyphs are not exported; never draw one.
- Nova is marked with `NovaBadge` from `@trenova/shared/components/ui/nova-badge`, sized with
  `size` (`xs`…`xl`), never redrawn in a stylesheet. See "The Nova badge" in the design-system doc.
- Tight corners, as on Vercel: `rounded-badge` (4px) for badges, counts and chips,
  `--radius-control` (6px) for controls, `--radius-surface` (6px) for containers. The whole
  `rounded-*` scale points at them. A label is never a pill: `rounded-full` is only for what
  is actually round (avatars, dots, switch tracks, icon-only round buttons).
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
lists the `Codegen Checks` steps with the exact commands to run them locally, and the
traps that have actually broken `master` — most often `projection.yml`, which requires every
GraphQL field to resolve to a column, an inferred relation, or a declared override, so
retiring a column while keeping its deprecated field breaks the build until the field is
added to `virtuals`.

## Web Build

The React Compiler runs in Rust (`react({ compiler: true })`, `oxc-transform-react`), `pnpm build`
bundles without typechecking, and CI fails a build whose first-visit download passes
`client/apps/web/bundle-budget.json` (`pnpm --filter @trenova/web bundle:check`). Chunking lives in
`client/apps/web/vite/chunk-groups.ts`, and a group with a `maxSize`, a pattern that is not anchored
to `node_modules/<name>/`, or a feature imported statically from the shell has each already broken
the app or put a whole feature on every page. Query key factories are imported from their own
module (`import { shipment } from "@/lib/queries/shipment"`); there is no merged `queries`
object, and lint rejects an import of `@/lib/queries`. PDFs are shown by the browser's own
viewer through `PdfFrame` (`components/elements/pdf-frame.tsx`); never add a PDF rendering
library, and every CSP keeps `frame-src 'self' blob:`. A service worker (`/sw.js`) serves the app shell
and hashed assets from cache, and the shell's queries persist to IndexedDB from an allow-list
(`SHELL_QUERIES` in `lib/session-cache.ts`) that must never hold business records or personal
details; every session end goes through `clearSessionCache()`. Open tabs reload onto a new
release at their next change of page unless a reload guard (`lib/reload-guards.ts`) holds the
page; anything new that keeps in-memory work a reload would lose must register one. **Read
[docs/engineering/web-build.md](docs/engineering/web-build.md) before changing
`apps/web/vite.config.ts`, `chunk-groups.ts`, the budget, `main.tsx`'s imports, the service
worker, the persisted query cache, or the GraphQL codegen hooks (`strip-fragment-text`,
`split-documents`).**

## Product Guide

Agents answer "where is…" and "how do I…" from a catalog generated from the web app and the
guides in `docs/product-guide/`. **Adding a page, renaming a button a guide names, or changing a
record link means regenerating it** (`pnpm --filter @trenova/web guide:generate`); CI fails
otherwise. Record links are built from one registry, `client/apps/web/src/config/record-links.ts`
(`recordPath`), never by hand. Read [docs/engineering/product-guide.md](docs/engineering/product-guide.md).

## Integration Logos

Every integration logo comes from Brandfetch, never drawn or traced by hand. Look the brand up
with the Brand API (`GET https://api.brandfetch.io/v2/brands/{domain}`, `Authorization: Bearer
<key>`), download the SVG it lists (PNG only when it has no SVG) unmodified into
`client/apps/web/public/integrations/logos/`, and point both the client vendor entry and the
server catalog (`integrationports`) at it. Brandfetch's `theme` is the tone of the logo itself,
not of the background it sits on (Xero's `light` variant is the white mark), so its `dark`
variant is our `*-light` file and its `light` variant our `*-dark` one. When Brandfetch has no mark for the product itself, or none for one of the two
backgrounds, use the parent company's (Business Central uses Microsoft's; QuickBooks shows
Intuit's in dark mode). The API key and client ID are the
maintainer's: ask for them or read `BRANDFETCH_API_KEY`, and never commit them.

## GraphQL (gqlgen + gqlexec)

- Schema lives in `services/tms/internal/api/graphql/schema/*.graphqls`; regenerate with `task gqlgen`. It runs `internal/api/graphql/gqlexec/gen`, which keeps gqlgen's binder, models and resolver stubs but writes the executor as one package per schema file (`internal/api/graphql/generated/exec/<file>exec`) on a shared runtime (`internal/api/graphql/gqlexec`). Never run `go tool gqlgen generate`; it writes gqlgen's single-package executor and breaks the build. Read [docs/engineering/graphql-executor.md](docs/engineering/graphql-executor.md) before changing the generator or runtime.
- Resolvers live in one package per schema file (`resolver/<file>resolver`). Each gets a generated `Deps` holding `*base.Core` plus only the `Services` fields its code names; add a new dependency to `Params`, `Services` and `newServices` in `resolver/services.go`, use it as `r.X`, and run `task gqlgen`. Resolver packages never import each other; put shared code in `base` only if it needs nothing but `Core`. In tests build `&QueryResolver{&Deps{Core: &base.Core{...}, ...}}`.
- When a domain struct field uses an initialism (`ShipmentBOL`, `DOTNumber`, `PTOType`), add the initialism to `go_initialisms` in `gqlgen.yml` rather than a per-field `fieldName:` override. Keep initialisms to 3+ letters: 2-letter ones (`AR`, `PO`, `MC`, `CC`) prefix-match SCREAMING enum values and mangle their Go constants (`DETENTION_POLICY` became `DetentionPOLicy`), so those few fields keep explicit overrides.
- Bind GraphQL enums to their domain type in `gqlgen.yml` (`RotaDayState: model: ...worker.RotaDayState`) when the string values match; that removes the need for a conversion resolver.
- Every root resolver must reach a permission check or be listed, with a reason, in `internal/api/graphql/authzlint`; self-scoped operations are named `My<Thing>` and each is listed in `selfScoped` under the reason it touches only the caller's own records (the name alone earns no exemption). `task gqlschema-diff` fails on breaking schema changes; swapping a field between wire-compatible scalars (`Int`↔`Timestamp`, `String`↔`Decimal`) is reported as dangerous, not breaking.
- Use the semantic scalars from `shared.graphqls`: `Timestamp` for Unix-second instants (never bare `Int` for a `*At`/`*Date` field) and `Decimal` for money and other exact quantities (never `String` or `Float`). Both bind to the same Go types as `Int`/`String`, so no mapper changes are needed.
- A field resolver that fetches per parent row must go through a per-request dataloader in `internal/api/graphql/loaders` (`batchByIDFunc` for entities, `batchCountFunc` for counts, `batchGroupFunc` for child lists) backed by a `...ByIDs` repository method; never call a single-ID getter from an object-type resolver. Register the factory in `loaders/loaders.go` and `bootstrap/modules/api/graphql.go`, add its field to `set` and its accessor and `Attach` to `loaders/access.go`. Resolvers read loaders only through those accessors (`loaders.Available(ctx)`, `loaders.CustomerByID(ctx, id)`), and tests attach one with the factory's `Attach`. No exported type in `loaders` may have a field or method, even an unexported one, that mentions `dataloadgen.Loader`, and no exported function may take or return one: Go compiles a generic type's methods again in every package that can see the type, and the old exported `Loaders` struct cost about 1,000 compiled functions in each of 40 resolver packages.
- Connection repositories only run their `COUNT` when `req.Cursor.IncludeTotalCount` is set; the GraphQL layer clears it when `totalCount` is not selected. New `ListConnection` methods must follow that gate.
- Patch inputs (`*PatchInput`) mark fields `@goField(omittable: true)`: absent leaves the value alone, explicit `null` clears it (or fails validation for fields the entity requires). Generated mappers come from `resolver/mappergen`.
- Clients send persisted operations by hash only; outside production the server re-reads `persisted-documents.json` on an unknown hash (`security.graphql.persistedDocumentsPath`), so run `pnpm --filter @trenova/graphql codegen` (or `pnpm dev`, which watches) after editing an operation. `/graphql` enforces a body-size limit and a per-user operation-cost budget (`security.graphql.*`).

## Agent Runtime

How every AI feature executes on Temporal — the queues and why they are split,
the agent loop in workflow code (one activity per model and tool call), the
HTTP-derived retry mapping, the step ledger that makes a retried write safe, the
turn stream, per-agent schedules, the one-shot and batch workflows, and the
`GetVersion` branches waiting on in-flight executions to drain.
**Read [docs/engineering/agent-runtime.md](docs/engineering/agent-runtime.md)
before changing anything under `internal/modules/agent/app/agentruntime/`,
`assistantservice/`, or `internal/modules/agent/jobs/{agentflow,modelcall,agentjobs,assistantjobs,completionjobs,importassistantjobs}/`.**
Extensions (web search and anything like it) add agent-only tools that an organization turns
on with its own vendor account; how they are gated, metered, and why a turn that read outside
content proposes every later write, is in
[docs/engineering/agent-extensions.md](docs/engineering/agent-extensions.md).
Portal automation (`list_portal_sites`, `read_portal`, `act_on_portal` on a cloud browser):
administrator-set sites, the navigation allow-list and the post-hoc page check, sign-ins and
authenticator keys sent as vendor secrets, texted or emailed sign-in codes relayed by the person
into the same browser, Trenova's own address refused as a site, actions run from an approval and followed by `PortalRunWorkflow` and a
`PortalRun` wait, is in
[docs/engineering/agent-portal-automation.md](docs/engineering/agent-portal-automation.md).
Read it before changing `portal_tasks.go`, `portal_sites.go`, `portaljobs`, `portaltools` or
`browseruseconnector`.
Data analysis (`analyze_data`: Python over the caller's report runs and attached files in an E2B
sandbox with all egress denied, charts kept with the run and drawn as the Desk's `analysis`
artifact) is in [docs/engineering/agent-data-analysis.md](docs/engineering/agent-data-analysis.md).
Read it before changing `analysis.go` in `agentextensionservice`, `analysis_tool.go`,
`e2bconnector` or `shared/e2b`.
How an agent looks back over a quiet conversation or a settled run and keeps what the work
taught (`agentreflectionservice`, `reflectionjobs`, the `agent_reflections` table) is under
"Learning from the work" in agent-runtime.md.
What agents are told about the organization's business and about the person they are
talking to — profiles worked out from the records, written up by a model, steered by people's
guidance and never edited by hand (`agentprofileservice`, `agentprofilejobs`, the
`agent_profiles` and `agent_profile_guidance` tables, the `guide_profile` tool) — is in
[docs/engineering/agent-profiles.md](docs/engineering/agent-profiles.md). Read it before
adding a fact, changing the writing prompt, or touching the profile sections of the prompt.
An agent parking its work until something happens in the world (`wait_until`, a truck
reaching a stop, a reply, an appointment or free time coming round, drive time running low,
a time) and picking it up as a new turn or run is described in
[docs/engineering/agent-waits.md](docs/engineering/agent-waits.md).
A Desk conversation about a shipment, invoice or dispute is a case: its state (working,
waiting, snoozed, settled) worked out on read, snoozes that follow the appointment or ETA,
waiting on the customer's or a carrier's reply, and the ready-to-bill checklist are in
[docs/engineering/desk-cases.md](docs/engineering/desk-cases.md).
An agent handing a task to another agent (`delegate_task`, the per-agent
allowlist, one level only, same person) is described in
[docs/engineering/agent-delegation.md](docs/engineering/agent-delegation.md).
The ledger, fiscal period and accounting system tools, and who holds them, are in
[docs/engineering/agent-accounting-tools.md](docs/engineering/agent-accounting-tools.md).
The fuel purchase, IFTA, rate agreement, rate sheet, fuel index price and report
administration tools, and who holds them, are in
[docs/engineering/agent-rates-tools.md](docs/engineering/agent-rates-tools.md).
The worker safety, compliance, HR, permit and driver pay tools, what is left to a person's
signature, and who holds them are in
[docs/engineering/agent-workforce-tools.md](docs/engineering/agent-workforce-tools.md).
How an agent drafts a whole EDI connection from a trading partner's implementation guide (cited
values checked against their pages, held values, secrets never taken, a change set on a revised
guide, one proposal applied in one transaction) is in
[docs/engineering/edi-spec-onboarding.md](docs/engineering/edi-spec-onboarding.md). Read it before
changing `ediconnectiondraft`, `draft_edi_connection`, `apply_edi_connection_draft` or the
`edi_connection_draft` Desk artifact.
The carrier, customer, commodity, hazardous material, location and equipment tools, document,
change alert and watchtower housekeeping, capture filing, and who holds them are in
[docs/engineering/agent-master-data-tools.md](docs/engineering/agent-master-data-tools.md).
The Triage queue tools (owners, parking, senders, reply and load drafts, filing, exceptions,
detention disputes and conflicts), the plans their previews render, and who holds them are in
[docs/engineering/agent-triage-tools.md](docs/engineering/agent-triage-tools.md).
What a person approves is a preview of the write, computed for them from the world as it is
now, and the approval carries its digest; tools implement `ToolPreviewer` with
`services/toolpreview`, and the filing baseline stays off `PendingAction`. **Read
[docs/engineering/proposal-previews.md](docs/engineering/proposal-previews.md) before
changing `toolpreview`, `proposalpreviewservice`, a tool's `Preview`, or how a proposal is
decided.**
Every write a person can make has an agent tool, a reasoned exemption or a `pending` entry in
`services/tms/internal/api/writecoverage/writecoverage.yml`, and a new mutation or write route
fails CI until it has one; see
[docs/engineering/agent-write-coverage.md](docs/engineering/agent-write-coverage.md).
Every run, turn, delegate's task and evaluation is one trace named by its id, rooted in an
`invoke_agent` span its finishing activity emits (never a span in workflow code); the span
catalogue, anchors, sampling, the link columns that tie proposals, steps, decisions and usage
rows to their spans, and the known limits are in
[docs/engineering/ai-tracing.md](docs/engineering/ai-tracing.md). Read it before adding a span,
a trace attribute or a link column.

## Formula Language Service

The Formula Studio's editor parses and checks the formula language in the browser
(`client/apps/web/src/components/formula-editor/expr-parser.ts`, `expr-check.ts`): every finding
carries its span and, where one exists, its fix, and the same analysis feeds the editor's
squiggles, the problems panel, the rate trace and Nova's cards. **Read
[docs/engineering/formula-language-service.md](docs/engineering/formula-language-service.md)
before changing the parser, a check, a diagnostic message, or anything that reads
`analyzeExpression`.** Diagnostic messages are ICU patterns: never put a bare brace in one.

## Desk Bench

`trenova desk ask "…"` and `trenova desk run` (`task desk-ask`, `task desk-bench`) drive the
Desk's agents through real turns on this checkout's code, on an in-process worker in its own
Temporal namespace, and write every model request, tool call, verdict, refusal, proposal and
follow-up to `.deskbench/runs/<time>/` (`summary.md`, `transcript.md`, `calls/`, `prompts/`,
`tools/`). Use it to find and confirm fixes to tool descriptions, schemas and prompts;
scenarios live in `services/tms/deskbench/scenarios/`, and every problem a run finds is recorded
in `services/tms/deskbench/findings.yaml` (`trenova desk findings` lists and exports them). **Read
[docs/engineering/desk-bench.md](docs/engineering/desk-bench.md) before changing
`internal/deskbench` or writing scenarios.** It approves real writes, so run it against a
seeded development database only.

## Desk Artifacts

Each Desk artifact kind has its own folder under `routes/desk/_components/artifacts/kinds/`.
Resource cards (a record drawn for what it is read for, such as a formula template, opened
from a table's row or looked up by the agent) live one folder each under `artifacts/cards/`,
are lazy-loaded through `cards/registry.tsx`, and each ships a skeleton drawn in its own
layout. **Read
[docs/engineering/desk-artifacts.md](docs/engineering/desk-artifacts.md) before adding
an artifact card or a part shared between cards.**

## Record Kinds

The server's permission registry owns which kinds of record exist; every other list of
kinds (record links, page context and mentions, agent subjects, @ search tabs, Desk cards,
memory) is generated from it, typed against it, or tested against it. **Read
[docs/engineering/record-kinds.md](docs/engineering/record-kinds.md) before adding a kind
of record or writing any table keyed by one**: never add another hand-kept list of kind names.

## Row-Level Security

PostgreSQL enforces tenant isolation itself: every tenant table is under a FORCE
row-level security policy keyed to a signed, transaction-local scope, and the
application role cannot bypass it. Repository methods run their body in
`dbtx.Read`/`dbtx.Write` and query through `DBForContext(ctx)`, never `.DB()`; a new
tenant table ends its migration with `SELECT trenova_rls.reconcile();`; code that must
see every tenant declares `dbscope.WithSystem(ctx, reason)` and is listed in
`systemscopelint`. **Read [docs/engineering/row-level-security.md](docs/engineering/row-level-security.md)
before adding a table, a repository method, a background job, or anything that runs
before a tenant is known.**

## Permissions

Roles grant `(resource, operations, data scope)` from the registry in
`domain/permission/resources_*.go`; one engine answers every check; every writer that grants
access runs the delegation check; every change goes through `PermissionChangeNotifier`, which
advances an organization permission epoch instead of deleting cache keys. Administrators get
new registry operations from `trenova db migrate`, never from a hand-written migration.
Grants are stored as made and implied operations are applied when evaluated. An "own" or
"team" scope exists only where a kind declares it; shipments, orders, invoices and workers
are narrowed by restrictive `read_scope` policies fed from a per-request `pkg/readscope`
holder, so they are enforced only while RLS is installed.
**Read [docs/engineering/permissions.md](docs/engineering/permissions.md) before adding a
resource or operation, a permission check, a data scope, a way of granting access, or
anything that invalidates cached permissions.**

## Security Audit Trail

Sign-ins, SSO callbacks, sign-outs and password resets write `auth_events` through
`iamports.AuthEventRecorder`; API key, role, permission, IAM and membership changes write
critical `audit_entries` through `permissionports.SecurityAuditor`; document downloads and views
and refused cross-tenant requests (`tenantboundary.Report`) are recorded too, each with the
request ID, client IP and user agent. Both tables are append-only and only the retention
sweep may delete from them. **Read [docs/engineering/security-audit.md](docs/engineering/security-audit.md)
before adding a mutation that changes who can do what, a new way to read a document, a
new cross-tenant refusal, or anything that deletes audit rows.**

## Realtime

Live updates are server-sent events from the API, fanned out through sharded Redis
Streams; there is no third-party realtime vendor. Publish with
`RealtimeService.PublishResourceInvalidation` (it only queues, so it is safe on hot
paths) and set `AudienceUserID` for anything addressed to one person. **Read
[docs/engineering/realtime.md](docs/engineering/realtime.md) before changing
`realtimeservice`, `infrastructure/realtimebroker`, the stream endpoint, or the
browser `realtimeClient`**, and before adding a presence or typing scope.

## Idempotency

A write sent with an `Idempotency-Key` header is claimed in Redis by
`IdempotencyMiddleware` and its response replayed for a repeat; a 5xx or transient
GraphQL error releases the key. Shipment create also stores the scoped key under a
partial unique index, so it stays exactly-once when Redis is down. **Read
[docs/engineering/idempotency.md](docs/engineering/idempotency.md) before changing the
middleware, `pkg/idempotency`, or adding a write that must be safe to retry.**

## Tracking Events

A stop's actual arrival and departure are settled from `tracking_events`, one row per report
from dispatchers, agents, drivers, telematics and EDI, applied in event-time order under the
move lock so out-of-order and repeated reports give the same result. People's reports go
through `TrackingEventService.RecordReported`, machine reports through `RecordObserved`; never
call `RecordStopActual` for a new source. **Read
[docs/engineering/tracking-events.md](docs/engineering/tracking-events.md) before changing
`trackingevent`, `trackingeventservice`, `ReconcileStopActuals`, the telematics stop-visit
path, the EDI 214 parser, or adding anything that records a stop actual.**

## Check Calls

A brokered load is followed through `check_calls`: dispatchers log calls (`logCheckCall`), the
load monitor logs them with `log_check_call`, and the carrier posts through a per-move token
link at `/carrier-tracking/:token` without signing in. An arrival or departure also goes through
tracking events (people strict, the carrier tolerant as source `Carrier`). With monitoring on in
dispatch controls, an in-transit carrier load quiet past its window (customer override, else the
organization's) raises a `NoCheckCall` watchtower item, swept every 15 minutes and cleared the
moment a check call lands. **Read [docs/engineering/check-calls.md](docs/engineering/check-calls.md)
before changing `checkcall`, `checkcallservice`, the carrier tracking link or its public page,
or the no-check-call sweep.**

## Projected Arrivals

Every open stop of an active shipment has a stored projected arrival in `stop_etas`, worked out
by the pure `etaprojection` package from the truck's position, the carrier's check calls or the
last departure, with routed drive time (PC*MILER when configured, else the planning speed),
dwell and the driver's hours. Stop, assignment and check call events signal a debounced
per-shipment refresh workflow, and a five-minute sweep picks up positions and hours; the board,
`Shipment.eta`, `GET /shipments/:id/eta/`, the agents and `etaSlip` all read the stored rows,
and an at-risk load raises an `EtaAtRisk` watchtower item. **Read
[docs/engineering/eta.md](docs/engineering/eta.md) before changing `stopeta`, `etaprojection`,
`shipmentetaservice`, `etajobs`, `etatrigger`, the route estimator or the arrival settings.**

## Customer Tracking

A customer follows a shipment through a token link to `/shipment-tracking/:token`, which shows only
the fields its `customer_tracking_profiles` row allows (holds and comments only when marked for the
customer), and through status emails (picked up, delayed, delivered, POD ready) its rules turn on.
Each owed email is one row in `customer_status_notifications`, unique per shipment, event and stop,
sent with an idempotency key and settled only while pending, so it goes out exactly once however
often the shipment is evaluated. Shipment events and ETA refreshes trigger a debounced evaluation,
and a ten-minute sweep catches PODs and retries. **Read
[docs/engineering/customer-tracking.md](docs/engineering/customer-tracking.md) before changing
`customertracking`, `customertrackingservice`, `customerstatustrigger`, `customertrackingjobs`, the
public tracking endpoints or page, or the customer tracking email templates.**

## Tender Negotiation

A tender runs one workflow per round (`tendercontract.WorkflowID(id, bounceCount)`). A carrier
backing out after accepting (`Bounce`, or an EDI 990 declining the accepted offer) cancels the
carrier assignment, revokes the offer and inserts **new** offer rows for the carriers the acceptance
cut short, because delivery is idempotent per offer. A counter-offer is judged against the
organization's counter floor in dispatch controls; one below it needs `tender:approve`, and
dispatch takes or refuses it with Accept or Decline on the respond route. The carrier is paid
`AgreedRate()`, never `rate`. An exhausted tender escalates in the service, not the workflow, per
`tender_exhausted_action`. **Read [docs/engineering/tender-negotiation.md](docs/engineering/tender-negotiation.md)
before changing `tender` (`retender.go`, `counter.go`, `escalation.go`), `tenderservice`,
`tenderjobs`, the tender routes, or the tender settings on dispatch controls.**

## Exception Queue

Every watchtower item is an owned exception: a status (Open, Acknowledged, InProgress, Snoozed,
Resolved), an owner, a due time by severity, a resolution code, and an append-only history in
`watchtower_item_events`, each change audited. New items are routed to the shipment's owner or
the default owner in dispatch controls (only someone who can see the item), overdue ones
escalate every few minutes up to a cap, and missed pickups, missed deliveries, missing PODs and
exhausted tenders are raised on their own. A person's resolution holds until the source clears;
a source never overwrites the owner, status or resolution. **Read
[docs/engineering/exception-queue.md](docs/engineering/exception-queue.md) before changing
`watchtower` (`exception.go`), `watchtowerservice`, `watchtowerrepository`,
`exceptionsourcerepository`, the shipment exception sources, the exception settings or the
exception tools.**

## AI Training Export

Operators export consenting organizations' AI corrections for fine-tuning with
`trenova ai training-export`; examples are anonymized in the worker before anything is
written, consent is read at export time and again before an organization's files are kept,
and every example is recorded so a withdrawal can be honoured later. **Read
[docs/engineering/ai-training-export.md](docs/engineering/ai-training-export.md) before
changing `domain/aitraining`, the Cloud edition's `internal/cloud/aitraining` packages
(`aitrainingservice`, `aitrainingjobs`), or what a correction stores.** Datasets are rendered from an export with the production prompt builder and scored
with the production reply reader; `tools/extraction-finetune` (Python) owns the training recipe — how a
confirmed answer becomes a target, which examples become preference pairs — and the training
itself. **Read [docs/engineering/extraction-fine-tuning.md](docs/engineering/extraction-fine-tuning.md)
before changing the extraction prompt, reply schema, `aicorrection.Score`, or that pipeline** — a
prompt or schema change means rendering again and retraining, and a schema change also means
regenerating the pipeline's schema fixture.

## AI Audit Trail

`ai_audit_events` is an append-only, per-tenant hash chain, signed with keys kept outside the
database (`aiAudit.chain.*`). One projector writes it from rows the agent runtime already
records; nothing else may write it. **Read
[docs/engineering/ai-audit-trail.md](docs/engineering/ai-audit-trail.md) before changing
`aiauditservice`, `aiauditjobs`, `domain/aiaudit` (the canonical form is hashed, so a field
change is a `hash_version` change), or any agent source table the projector reads.** A
retention sweep that deletes agent rows must stay behind `SourcePruneHorizon`. Never remove a
chain key while rows signed with it are retained.

## AI Analytics

AI control's Analytics tab reads one query, `agentAnalytics`, over rollup tables
(`ai_work_units`, `ai_work_unit_tools`, `ai_provider_hourly_stats`) that the five-minute
`ai-analytics-refresh` job folds in from usage records, run steps and runs. Every threshold
("few", ranking, suggested routes, the Nova sentences) lives on the server in
`aianalytics/metrics.go`, answers are cached per tenant epoch in Redis, and series colours are
assigned server-side so a model keeps its hue under every filter. **Read
[docs/engineering/ai-analytics.md](docs/engineering/ai-analytics.md) before changing
`aianalyticsservice`, `aianalyticsrepository`, `aianalyticsjobs`, the `aianalytics` domain, or
what the completion router or step ledger records about a call.**

## Editions (self-hosted and Trenova Cloud)

This repository builds the self-hosted product. Trenova Cloud code lives in a private repository
and is overlaid at build time into `services/tms/internal/cloud/` plus one registration file,
`services/tms/cmd/cli/edition_cloud.go`, which calls `edition.Register(cloud.Edition())`. The
build must pass with and without that overlay, so **public code never imports
`internal/cloud`**. The seam is `internal/bootstrap/edition`: an `Edition` adds fx options to
every process (`Options`), the API (`APIOptions`) or the worker (`WorkerOptions`), Cobra
commands, Postgres migration sets (run after the public ones) and configuration sections the
loader decodes strictly on the edition's behalf. The public graph always provides the
self-hosted implementation of every platform port (`PlanService`, `QuotaGuard`,
`EntitlementProvider`, `BillingProvider`, `UsageProvider`, `EditionInfo`,
`AITrainingHistoryService`); an edition swaps them with root-level `fx.Decorate`. Routes and
middleware join through the `internal/api/routegroup` value groups, GraphQL extensions through
`edition_graphql_extensions`, and `GET /system/public-config` fields through
`public_config_contributors`. Decide behaviour from an injected port (`PlanService.EnforcesPlans`,
`EditionInfo.SharedTenancy`, an onboarding row existing), never from `platform.mode`.

**Binaries and edition registration.** Besides `cmd/cli` (the production image's `api run`
and `worker run`), the service builds `cmd/api` and `cmd/worker`, which link only the API's
or the worker's graph (`internal/bootstrap/core` plus `apiapp` or `workerapp`). Each main
registers its edition separately, so an overlay must also add `cmd/api/edition_cloud.go` and
`cmd/worker/edition_cloud.go`, each calling `edition.Register(cloud.Edition())`; without them
those two binaries run the self-hosted edition.

## Modules

`services/tms` is split into levelled modules, `internal/modules/<module>/{domain,ports,app,adapters/postgres,jobs}`
with a `module.go` (`SharedOptions`, `APIOptions`) and a `jobs/jobs.go` (`WorkerOptions`). A module may import only
the `domain`, `ports` and `<name>contract` packages of strictly lower modules, plus `pkg/*` and `internal/infrastructure/*`;
an upward need is an inversion port declared low and provided by the higher `module.go`. `go run ./scripts/archlint`
(`task lint`, CI) enforces it against `.go-arch-lint.yml`. Every package belongs to a module; `internal/core` no longer
exists, and new code goes in its module.
**Read [docs/engineering/module-architecture.md](docs/engineering/module-architecture.md) before adding a module, an entity, a port, a service or a job.**

## Bun ORM

For help with Bun ORM, look in the [docs](docs/engineering/bun/).
When writing repositories, always use the generated column helpers in `services/tms/pkg/buncolgen/` — never hand-write column references. Read [docs/engineering/bun/buncolgen.md](docs/engineering/bun/buncolgen.md) for the full method reference, canonical repository patterns, and regeneration workflow before writing any repository code.

## DO NOT
- **Processes**: Do not run high usage tasks that will max out CPU, Disk and/or memory usage.
- **Mockery**: Do not run mockery against the entire codebase — manually adjust mocks in the codebase.


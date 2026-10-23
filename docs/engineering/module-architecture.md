# Module architecture

How `services/tms` is divided into modules, which module may import which, where new code
goes, how the API and worker binaries are assembled from the modules, and how to measure what
a change costs the build.

The service was one hexagon. Every domain package lived in `internal/core/domain`, every
interface in two hub packages (`internal/core/ports/repositories` and
`internal/core/ports/services`), every service in `internal/core/services`, every column helper
in one `pkg/buncolgen` package, and one bootstrap linked the API and the worker together. A new
method on any port therefore recompiled almost every service, a new column recompiled every
repository, and the API binary carried every workflow and activity. The modules put a boundary
around each part of the business, and the rules below keep the import graph a DAG, so an
edit recompiles only the packages that actually depend on it.

## Layout

```text
internal/modules/<module>/
├── domain/<entity>/                 # entities and value objects
├── ports/<entity>ports/             # repository and service interfaces, request types
├── app/<name>service/               # services, validators, other application logic
├── adapters/postgres/<name>repository/
├── jobs/
│   ├── jobs.go                      # WorkerOptions(): the module's worker-only graph
│   └── <name>jobs/                  # Temporal workflows, activities, registry, schedules
│       └── <name>contract/          # workflow names, payloads, signals, starters
├── <module>test/                    # test helpers (mock-backed builders, fixtures)
└── module.go                        # SharedOptions(), APIOptions()
```

| Piece | Where |
|---|---|
| Module levels and package assignments | `services/tms/docs/modules/module-map.yml` |
| Dependency rules | `services/tms/.go-arch-lint.yml`, checked by `go run ./scripts/archlint` |
| Shared, domain-free port types (`WorkflowStarter`, `AfterCommit`, storage, search, actors) | `internal/modules/kernel/ports/kernelports` |
| Mocks | `internal/testutil/mocks/<entity>portsmock`, configured in `.mockery.yaml` |
| Column helpers | `pkg/buncolgen` (shared `Column` and `TableInfo`) and `pkg/buncolgen/<domain>col` |
| Graph shared by every process | `internal/bootstrap/core` |
| API-only and worker-only graphs | `internal/bootstrap/apiapp`, `internal/bootstrap/workerapp` |
| Binaries | `cmd/api`, `cmd/worker`, `cmd/cli` |
| Ledgers for the Cloud overlay | `services/tms/docs/modules/moves.tsv`, `symbol-moves.tsv` |
| Overlay import rewriter | `scripts/buildbench/rewrite-imports` |
| Build benchmark | `scripts/buildbench`, `task build-bench` |

Package names did not change when packages moved: `shipmentservice` is still
`shipmentservice`, now at `internal/modules/operations/app/shipmentservice`. Domain package
names must stay unique across modules, because the column generator names its output after the
package's base name (`domain/shipment` becomes `pkg/buncolgen/shipmentcol`).

## Modules and levels

Every module has a level. A module depends only on modules at a strictly lower level, so the
levels are a topological order of the module DAG.

| Level | Module | Holds |
|---|---|---|
| 0 | `kernel` | platform catalog, setting versions, US states, entitlements and usage, `kernelports` |
| 1 | `identity` | tenants, IAM, permissions, API keys, sessions, onboarding, plans, organization controls |
| 2 | `workspace` | audit, notifications, email, custom fields, reports, table layouts, integrations |
| 3 | `reference` | commodities, hazardous materials, service and shipment types, accessorials, exchange rates, document types |
| 4 | `places` | locations, geofences, distance and mileage, jurisdiction rules |
| 4 | `equipment` | tractors, trailers, equipment types and manufacturers, fleet codes |
| 4 | `parties` | customers, carriers, carrier capacity and intelligence |
| 5 | `workforce` | workers, permits, telematics |
| 5 | `documents` | documents, uploads, capture, templates, extraction, inbound messages |
| 6 | `rating` | rate agreements, matrices, quotes, fuel surcharges, formula templates, rate confirmations |
| 7 | `operations` | shipments, orders, tenders, dispatch control, detention, service failures |
| 8 | `edi` | EDI documents, transport and X12 |
| 8 | `accounting` | GL accounts, journal entries, fiscal periods and years, accounting sync |
| 9 | `billing` | invoices, billing queue, customer payments, bank receipts, late charges |
| 9 | `settlement` | driver and carrier settlement, driver pay, fuel purchases, IFTA |
| 10 | `agent` | agents, conversations, AI audit, corrections and usage, watchtower, briefings |

`module-map.yml` lists every package each module owns. It was regenerated from the tree once
every package had a module, so its `unassigned` list is empty; keep it in step with the tree
when you add or move a package.

## Dependency rules

1. A module may import, from another module, only that module's `domain`, `ports` and
   contract (`jobs/<name>jobs/<name>contract`) packages, and only when the other module's
   level is strictly lower.
2. Modules at the same level may not import each other. `billing` and `settlement` are both
   level 9 and share nothing directly.
3. A module's `app`, `adapters`, `jobs` and `module.go` are never imported by another module.
   Reach a lower module's behaviour through its port, not its service struct.
4. When a lower module needs something only a higher module can do, the lower module declares
   an inversion port in its own `ports` package and the higher module provides it from its
   `module.go`. See [Inversion ports](#inversion-ports).
5. Any module may import `pkg/*` and `internal/infrastructure/*`.
6. `internal/api` may import any module's `domain`, `ports`, `app` and contract packages, never
   its `adapters` or `jobs`.
7. `pkg` imports only `pkg`. `kernel` imports no other module.

Inside a module the layers are ordered too:

| Component | May import from its own module | May import from lower modules |
|---|---|---|
| `domain` | `domain` | `domain` |
| `ports` | `domain`, `ports` | `domain`, `ports` |
| `app` | `domain`, `ports`, `app`, contracts | `domain`, `ports`, contracts |
| `adapters` | `domain`, `ports`, `adapters` | `domain`, `ports` |
| contracts | `domain`, `ports`, `app`, contracts | `domain`, `ports`, contracts |
| `jobs` | everything except `module.go` and `<module>test` | `domain`, `ports`, contracts |
| `module.go` | everything except `jobs` and `<module>test` | `domain`, `ports` |
| `<module>test` | anything | anything |

`domain` may import `pkg` but not `internal/infrastructure`; every other component may import
both. The Temporal runtime is infrastructure: the client and its connection settings, the
worker manager, registry, schedules, interceptors, health check and replay test helpers are in
`internal/infrastructure/temporaljobs`, and the workflow starter is in
`internal/infrastructure/temporaljobs/workflowstarter`. A module's `jobs` reach them through the
general rule that every component except `domain` may import infrastructure.

`internal/infrastructure` may import every module's `domain`, `ports` and contract packages.
`internal/bootstrap`, `cmd`, `internal/testutil`, `internal/deskbench` and `scripts` may import
anything.

### Enforcement

`.go-arch-lint.yml` declares one component per module layer (`operations-domain`,
`operations-ports`, `operations-app`, `operations-adapters`, `operations-jobs`,
`operations-contracts`, `operations-module`, `operations-test`) plus `api`, `bootstrap`, `cmd`,
`testutil`, `infrastructure`, `deskbench`, `scripts` and `pkg`, and lists for each what it may
depend on. `_test.go` files are excluded. No generator for the file is checked in, so edit it
directly. Its header requires any exception to carry a `TODO(build-arch)` reason, and the number
of exceptions must not grow.

Run the check with `go run ./scripts/archlint` from `services/tms`; `task lint` runs it after
golangci-lint, and CI runs it in `test-tms.yml`. The script runs `go tool go-arch-lint check
--json` and fails on any dependency, unmatched-file, deep-scan or configuration warning, except
imports of the sibling `shared` module (`github.com/emoss08/trenova/shared/...`). go-arch-lint
treats every import under the module path as project code and cannot map `shared` to a
component, so it would report every such import as a violation. Running `go tool go-arch-lint
check` directly therefore prints those false reports.

Test helpers have a second guard. The depguard rule `testcode` in the repository's
`.golangci.yml` forbids non-test files from importing `internal/testutil`, testify's `mock`,
`golang/mock`, and each listed `<name>test` helper package, because anything a production file
imports is linked into the binary. Files under a directory whose name ends in `test` are exempt,
which is what lets a `<module>test` package build mocks.

### Inversion ports

An upward need is turned around with an interface. The consumer's module declares the narrow
interface it needs; the higher module implements it and binds it in its `module.go`. Neither
module imports the other's `app`.

`rating` (level 6) re-derives a move's cost events when carrier coverage changes, which is
settlement's work (level 9):

```go
// internal/modules/parties/ports/carrierports/carriercostaccrual.go (level 4)
type CarrierCostAccrual interface {
	ReaccrueMove(ctx context.Context, tenantInfo pagination.TenantInfo, moveID pulid.ID) error
}

// internal/modules/rating/app/carrierassignmentservice (level 6) takes it as a dependency
CostAccrual carrierports.CarrierCostAccrual `optional:"true"`

// internal/modules/settlement/module.go (level 9) provides it
func(s *carriersettlementservice.Service) carrierports.CarrierCostAccrual { return s },
```

When many higher modules contribute to one lower consumer, use an fx value group. Document
templates render invoices and detention notices without importing either: the group is
declared as `documenttemplateports.ContextBuilderGroup`, and `billing` and `operations` each
provide a builder with `fx.ResultTags(documenttemplateports.ContextBuilderGroup)`.

An `optional:"true"` dependency resolves to nil when nothing provides it, and the service then
silently does nothing. If the port must be present in a process, add it to
`TestWiring_BestEffortPortsAreActuallyProvided` in `internal/bootstrap/wiring_test.go`, which
asks for it without the tag in every graph.

## Binaries

| Binary | Graph | Watch task |
|---|---|---|
| `cmd/api` | `core.OptionsFor(e)` + `apiapp.OptionsFor(e)` | `task run-watch` (`.air.toml`) |
| `cmd/worker` | `core.OptionsFor(e)` + `workerapp.Modules()` + `workerapp.OptionsFor(e)`; `--queues` limits task queues | `task run-worker-watch` |
| `cmd/cli` | `bootstrap.Options()` (core plus `workerapp.Modules()`), with `apiapp` for `api run` or `workerapp` for `worker run` | none; the production image runs it |

`internal/bootstrap/core` holds what every process needs: configuration, infrastructure, the
Temporal client, every module's `SharedOptions()` and the edition's `Options`. `apiapp` adds the
HTTP server, GraphQL, handlers, middleware, every module's `APIOptions()` and the edition's
`APIOptions`. `workerapp.Modules()` adds every module's `jobs.WorkerOptions()` and the schedule
module; `workerapp.OptionsFor` adds the worker runtime (`temporaljobs.WorkerModule` from
`internal/infrastructure/temporaljobs`), the agent runtime's worker modules and the edition's
`WorkerOptions`.

`run-watch` and `run-worker-watch` build with `-gcflags='github.com/emoss08/trenova/...=-N -l'`
and link with `-ldflags='-s -w'`. The first keeps a private edit from changing a package's
export data, so it recompiles only its direct importers; the second leaves out the symbol
table and DWARF, which only a debugger reads, and shortens the link on every reload. The
binaries run unoptimised code, so do not benchmark against them.

Each main registers its edition separately; see "Server seams" in [editions.md](editions.md).

### Why `WorkerOptions` live in the jobs package

`module.go` is imported by `internal/bootstrap/core`, and through it by `cmd/api`. If
`module.go` referenced the module's workflows, the API would link every workflow and activity
and recompile whenever one changed. So each module's worker graph is a separate package,
`internal/modules/<module>/jobs` (`package jobs`, function `WorkerOptions()`), imported only
by `internal/bootstrap/workerapp`. The arch-lint rules keep it that way: `<module>-module` and
`<module>-app` may not depend on `<module>-jobs`.

API-side code still has to start, signal and cancel workflows. It does that through the
contract package next to the workflow, `jobs/<name>jobs/<name>contract`, which holds only the
workflow name constants, payload types, signal names, workflow ID helpers and, where useful, a
starter function. The service imports the contract and calls `kernelports.WorkflowStarter` with
the name; the workflow registers under the same constant. Keeping the name in one place matters:
before the contracts existed, the API started bulk approvals as `BulkApprovalWorkflow`, a type
no worker registered, while the worker registered `BillingQueueBulkApprovalWorkflow`.

To confirm a binary does not link something, list the chain that pulls it in:

```bash
task build-bench -- --deps-of api 'internal/modules/[a-z]+/jobs(/[a-z]+)?$'
```

The pattern matches each module's `jobs` package and the job packages under it, but not their
contract packages. Every match it prints, with its import chain from `cmd/api`, is worker code
linked into the API and a leak to fix.

## Adding code

New code goes in its module. `internal/core` and `internal/infrastructure/postgres/repositories`
no longer exist; do not recreate them. `internal/infrastructure` is for technical adapters such
as the Temporal runtime, `m2msync`, and the storage and search clients.

### A module

1. Choose its level in `services/tms/docs/modules/module-map.yml`: above every module it needs, below every
   module that needs it. If a lower module would need it, that module gets an inversion port
   instead. Add the module, its level and its packages to the map.
2. Create `internal/modules/<module>/` with `domain`, `ports`, `app` and `adapters/postgres`,
   and a `module.go` in `package <module>` exporting `SharedOptions()` and `APIOptions()`
   (return `fx.Options()` when there is nothing API-only).
3. If it has workflows, create `jobs/jobs.go` (`package jobs`) exporting `WorkerOptions()`.
4. Register it: `<module>.SharedOptions()` in `core.OptionsFor`
   (`internal/bootstrap/core/core.go`), `<module>.APIOptions()` in `apiapp.OptionsFor`, and
   `<module>worker.WorkerOptions()` in `workerapp.Modules()`. Keep each list in level order.
5. In `.go-arch-lint.yml`, add the eight components (`<module>-domain`, `-ports`, `-app`,
   `-adapters`, `-jobs`, `-contracts`, `-module`, `-test`) with the same `in:` globs as the
   existing modules, and their `deps` entries naming the lower modules. Add the new module's
   `-domain`, `-ports` and `-contracts` to the `deps` lists of every higher module's
   components, to `api` (with `-app`) and to `infrastructure`.
6. If it has a `<module>test` package, add it to the `testcode` deny list in `.golangci.yml`.
7. Run `task lint`.

### An entity

1. Domain: `internal/modules/<module>/domain/<entity>/`. It imports only its own module's and
   lower modules' domain packages and `pkg`.
2. Write the migration, and follow [row-level-security.md](row-level-security.md) for a tenant
   table.
3. Regenerate the column helpers with `task generate-columns`. The generator reads every
   `internal/modules/*/domain/*` package and writes one package per domain package,
   `pkg/buncolgen/<domain>col`. Repositories import that package (`shipmentcol.ShipmentColumns`),
   not `pkg/buncolgen` itself, so a column change recompiles only the repositories that use it. If the entity is reportable, run
   `task generate-reportcatalog`; it reads the same directories.
4. Add its ports, repository and service as below.

### A port

1. Put repository and service interfaces, with their request and option types, in
   `internal/modules/<module>/ports/<entity>ports/`. Where a package has both, repositories are
   in `<entity>.go` and services in `<entity>_services.go`.
2. A type with no domain dependency that several modules share (a starter, a hook, a storage
   or search client) goes in `kernelports`. Do not put anything there that names a domain type.
3. An interface a lower module needs and a higher module implements is an inversion port: it
   goes in the lower module's ports package.
4. Add the package to `.mockery.yaml` with `config: all: true`; its mocks land in
   `internal/testutil/mocks/<entity>portsmock`. Generate mocks for that package alone, or write
   the mock by hand; do not run mockery over the whole tree.

### A service

1. Create `internal/modules/<module>/app/<name>service/` with a constructor taking an `fx.In`
   params struct. Depend on ports: your own module's, or a lower module's. Never import another
   module's `app` package.
2. Register the constructor (and validator) in the module's `SharedOptions()`. Bind it to its
   port in the same list, either `func(s *<name>service.Service) <entity>ports.<Name>Service {
   return s }` or `fx.Annotate(<name>service.New, fx.As(new(<entity>ports.<Name>Service)))`. A
   provider that only the API needs goes in `APIOptions()`.
3. If a GraphQL resolver uses it, add it to `resolver/services.go` and run `task gqlgen`; see
   [graphql-executor.md](graphql-executor.md).
4. To start a workflow, import the workflow's contract package, never its jobs package.

### A repository

1. Create `internal/modules/<module>/adapters/postgres/<entity>repository/` with a `New`
   constructor that returns the port interface.
2. Use the generated `<domain>col` helpers ([docs/engineering/bun/buncolgen.md](bun/buncolgen.md)) and run
   every exported method in `dbtx.Read` or `dbtx.Write`
   ([row-level-security.md](row-level-security.md)).
3. Register `<entity>repository.New` in the module's `SharedOptions()`.

### A job

1. Create `internal/modules/<module>/jobs/<name>jobs/` with the workflow, activities, a
   `registry.go` (a `registry.DomainConfig` naming a task queue from `pkg/temporaltype`, and a
   `NewRegistry`), and a `module.go` exporting `var Module = fx.Module(...)` that provides the
   activities and the registry into the `worker_registries` group as a
   `registry.WorkerRegistry`. A scheduled job also provides
   `schedule.AsProvider(NewScheduleProvider)`.
2. If anything outside the worker starts or signals it, create
   `<name>jobs/<name>contract/` with the workflow name, payload, signal names and workflow ID
   helpers, and register the workflow under the contract's name constant. The payload must
   carry its tenant in a form the worker's interceptor can bind; see "A request or job" in
   [row-level-security.md](row-level-security.md).
3. Add `<name>jobs.Module` to the module's `jobs/jobs.go`. If the module had no jobs package,
   create one and add it to `workerapp.Modules()`.
4. For agent workflows, read [agent-runtime.md](agent-runtime.md) first.

## The Cloud overlay

The private Cloud overlay compiles against public packages, so every move breaks its imports.
Two ledgers record the moves so the overlay can be rewritten mechanically.

| Ledger | Lines | Format | Records |
|---|---|---|---|
| `services/tms/docs/modules/moves.tsv` | 849 | `old/import/path<TAB>new/import/path` | each package moved from `internal/core/{domain,services,temporaljobs}` and `internal/infrastructure/postgres/repositories`: into a module, including the packages once left unassigned, or into `internal/infrastructure` (the Temporal runtime and workflow starter to `temporaljobs`, `m2msync` to `postgres/m2msync`) |
| `services/tms/docs/modules/symbol-moves.tsv` | 11,838 | `old/path.Symbol<TAB>new/path.Symbol` | each symbol that changed package: the split of `internal/core/ports/{repositories,services,storage}` and the root `ports` package into `<entity>ports` and `kernelports`, the mocks into `<entity>portsmock`, contract types out of jobs packages, test helpers into `<name>test` packages, the generated column helpers into `pkg/buncolgen/<domain>col`, and the split of `agenttoolservice` into `agenttools/<family>tools` and `agenttools/common` |

`scripts/buildbench/rewrite-imports` applies both to a tree of Go files, in place. Run it from
`services/tms`:

```bash
go run ./scripts/buildbench/rewrite-imports <dir>...
go run ./scripts/buildbench/rewrite-imports -moves <file> -symbols <file> <dir>...
```

It rewrites moved import paths (following a package that moved twice), rewrites each
`oldpkg.Symbol` selector to `newpkg.Symbol`, adds the new imports, and formats the file with
goimports, which drops imports left unused. It handles named imports; it does not handle dot
imports. It prints how many files it changed.

When you move a package, append its line to `moves.tsv`; when you move a symbol between
packages, append it to `symbol-moves.tsv`. The column helper split is in the symbol ledger too,
so `buncolgen.ShipmentColumns` in overlay code becomes `shipmentcol.ShipmentColumns`.

## Generics and build time

Go compiles the methods of an instantiated generic type in every package that can see the
type, and the linker throws the copies away afterwards. A package "sees" a type when anything it
reads from another package mentions it: an exported struct's fields, even unexported ones, and
the signatures of an exported type's methods, even unexported ones. The search is transitive:
a pointer to an unexported struct is followed into that struct, and an interface's method
signatures are followed into the types they name. Plain package-level functions are not
followed, and neither is an interface whose methods never mention the generic type, so those
stop the spread.

`sync.Map` is generic underneath: since Go 1.24 it wraps `internal/sync.HashTrieMap[any, any]`,
so a package that can reach a `sync.Map` field compiles about 130 functions (roughly 15 KB of
machine code and 60 ms of compile time) whether or not it touches the map. Keep `sync.Map` out
of any type that many packages can reach, including through unexported fields; a plain map
behind a `sync.Mutex` or `sync.RWMutex` costs nothing outside its own package.

These designs did this at scale before they were changed:

| Source | What every importer compiled | Fix |
|---|---|---|
| `twilio-go` | about 15,000 functions per package that reached its client | replaced with a direct call to Twilio's Messages API (`internal/infrastructure/sms`) |
| `loaders.Loaders` | about 1,000 functions in each of 40 resolver packages | loaders reached through typed accessors; the set and its builders are unexported (`loaders/access.go`) |
| service `Validator` structs | the generic validator of every entity a package could see: 1,400 to 1,700 functions in each GraphQL and bootstrap package | fields typed `validationframework.TenantedEntityValidator[T]` / `ScopedEntityValidator[T]` |
| `pagination.EncodeCursorFromEntity*`, `ExtractCursorValues` | four generic encoders, called by `CursorListResult[T].NextCursor`, in each of the 707 packages that mention a `CursorListResult` | the encoders take `any`; they only ever used reflection and type assertions on the item |
| `auditservice.SensitiveDataManager`, the RLS scope reporter behind `postgres.Connection` | `sync.Map`'s generic internals in every package that reaches a service or a repository | mutex-guarded maps |
| `*lru.Cache` / `*expirable.LRU` fields in the template engine, the rate table cache and six other packages | golang-lru's cache, list and entry methods, about 17 KB in each of 30 packages | fields typed `lrucache.Cache[K, V]` from `shared/lrucache`, built with `lrucache.New` or `lrucache.NewExpirable` |

Bun does the same to every package that mentions any Bun type, which includes every domain
package with a `BeforeAppendModel(ctx, bun.Query)` hook: `schema.Table` holds a `sync.Map` and
`schema.Tables` an `xsync.MapOf`, both reachable from `bun.Query` through `schema.Dialect`. That
is about 180 functions (24 KB) in each of about 1,500 packages, close to half of all the generic
code the API's first-party packages compile. Hiding the two maps behind unexported interfaces in
Bun cut a cold API build by 15% and a rebuild after an exported edit to `domain/shipment` by
about 20% on a 4-core machine. Nothing on this side of the import can stop it, so the fix lives
in Trenova's fork of Bun, [`emoss08/bun`](https://github.com/emoss08/bun), whose `master` is
upstream `master` plus that change and a fix for slice inserts that dropped a later row's value
when the first row took the column default ([uptrace/bun#1394](https://github.com/uptrace/bun/issues/1394)). `services/tms/go.mod` and `shared/go.mod` replace every
`github.com/uptrace/bun` module with the fork at one commit; the Bun modules are separate Go
modules, so a new fork commit moves all of the `replace` lines together, never one of them.

go-redis did it to every package that holds a `*redis.Client` or a `redis.UniversalClient`, at
about 0.7 seconds each. `baseClient` kept a `weak.Pointer[Client]`, and because `Client` embeds
`cmdable`, the stand-in type the compiler builds for that instantiation carries every promoted
command method, so each of those packages compiled about 1,480 wrapper functions. Several
`sync.Map` fields reachable from the client added the hash trie on top. Only `redis.Cmdable` stays
clear of both. Trenova builds against
[`emoss08/go-redis`](https://github.com/emoss08/go-redis), whose `master` is upstream `master`
plus a change that holds the weak reference and the maps behind types whose method signatures
name no generic type. A package holding a `*redis.Client` now compiles 114 functions in about
66 ms instead of 1,722 in about 690 ms. The core module, `extra/redisotel` and `extra/rediscmd`
are replaced at one commit, under the same rule as Bun's modules.

These measurements come from probe packages: a package that imports one dependency and mentions
one of its types in a function signature, compiled on its own. A probe that compiles thousands of
functions for a type it only passes along is a dependency worth looking into.

To check a package, count the generic functions its object file defines:

```bash
go tool nm -type "$(go list -tags nomsgpack -export -f '{{.Export}}' ./internal/api/graphql)" \
  | grep -E ' [Tt] ' | grep -c '\['
```

A package that defines thousands of functions for a library it only passes values through is
paying for a type it can see but does not need.

## Build benchmark

`task build-bench` measures what an edit costs. It builds a target, then applies a probe edit
to a real file, rebuilds, and restores the file byte for byte, once per scenario.

| Scenario | Edit |
|---|---|
| `cold` / `warm` | first build, from an empty (`--cold`) or existing benchmark cache |
| S1 | none (no-op rebuild) |
| S2 | unexported func added to `shipmentservice` |
| S3 | unexported func added to `shipmentresolver` |
| S4 | exported func added to `domain/shipment` |
| S5 | exported type added to the package that declares `ShipmentRepository` |
| S6 | exported func added to the least-depended-on domain package the target links |

```bash
task build-bench -- --target api --mode air           # what a run-watch reload costs
task build-bench -- --target worker --mode release
task build-bench -- --target cli --mode release --cold
task build-bench -- --fanin                           # the 30 most depended-on packages
task build-bench -- --deps-of api '<regexp>'          # why the API links matching packages
```

| Option | Effect |
|---|---|
| `--target cli\|api\|worker` | the main package to build (`./cmd/<target>`); default `cli` |
| `--mode release\|air` | `release` is `go build -tags nomsgpack`; `air` adds the watch tasks' `-gcflags=<module>/...=-N -l` and `-ldflags=-s -w` |
| `--strip` | adds `-ldflags=-s -w` to a release build; `air` always has it |
| `--cold` | empties the benchmark's cache first |
| `--fanin` | ranks first-party packages by how many first-party packages depend on them, directly or transitively |
| `--deps-of <target> <regexp>` | lists every first-party package the target links whose import path matches, with the shortest import chain to it |

Each scenario reports wall time, link time, first-party and total packages recompiled, binary
size and the slowest compile actions. Results go to
`scripts/buildbench/results/<sha>-<target>-<mode>[-strip][-cold].json`, which git ignores.
Compare S1 to S6 across warm runs: the first edit after a cold build recompiles more than the
same edit does later.

The benchmark builds with its own cache, `$(go env GOCACHE)-buildbench`, so it neither reuses
nor evicts the cache your editor and `task` share. It runs under `nice` with `-p` set to one
less than the CPU count. It refuses to probe a file with uncommitted changes and never runs
`git checkout`, `git restore` or `git stash`; an interrupted run restores the probed file from
its backup on exit.

## Agent tools

The action tools live in one package per family under
`internal/modules/agent/app/agenttools/<family>tools` (`invoicetools`, `workforcetools`,
`masterdatatools`, ...), named with a `tools` suffix so they never collide with the domain
package of the same name. Code that several families use is in `agenttools/common`; test
helpers that several families' tests share are in `agenttools/agenttoolstest`, a test-only
package. `agenttoolservice` keeps the public API: `Module`, `ToolProviders()` and the registry,
which collects every tool from the `agent_tools` fx value group. `ToolProviders()` lists the
providers in their original order, because the order of the group is the order of the
catalog the model is shown.

A family package may import `common` and other families as long as the graph stays acyclic;
nothing imports `agenttoolservice` except its callers outside the tools. A test file lives in
the package it tests unless a package it imports depends on that package, in which case it
lives in `agenttoolservice`. Fields that had to be exported across packages carry
`json:"-"`, so the split changed no tool's name, schema, policy, descriptor or preview.

To add a tool: put it in its family package, export its provider, and add the provider to
the family's place in `ToolProviders()` (or the family's own provider list, such as
`operationsToolProviders()`).

## Measured results

Measured on 2026-10-11 with `task build-bench` on a 16-core WSL2 machine (15 GB), comparing
`master` at `70f77e005` with this branch. Four configurations ran interleaved, one discarded
priming round and then three measured rounds each; the table gives the median wall time and the
first-party packages recompiled. The three runs of each cell agreed within a few percent.

| Edit | A: `master` cli, unstripped (old `run-watch`) | B: `master` cli, stripped | C: branch cli, stripped | D: branch api, stripped (new `run-watch`) |
|---|---|---|---|---|
| S1 no-op rebuild | 0.7s | 0.7s | 0.8s | 0.7s |
| S2 unexported func in `shipmentservice` | 13.9s (6) | 8.5s (6) | 6.5s (6) | 6.3s (6) |
| S3 unexported func in `shipmentresolver` | 13.6s (2) | 8.4s (2) | 7.8s (2) | 6.8s (2) |
| S4 exported func in `domain/shipment` | 100.6s (1,079) | 95.8s (1,079) | 55.2s (627) | 47.8s (598) |
| S5 exported type beside `ShipmentRepository` | 100.9s (1,056) | 94.8s (1,056) | 38.4s (451) | 36.0s (423) |
| S6 exported func in the least-depended-on domain | 13.2s (12) | 7.6s (12) | 5.8s (6) | 5.6s (5) |
| Link | 10.6s | 5.1s | 4.6s | 4.4s |
| Binary | 347 MB | 268 MB | 245 MB | 233 MB |

- A to D is what a reload costs now against before: about half for a small edit, and 52% and
  64% less for the two edits that ripple through the shipment domain.
- B to C is the code alone, with the same binary and flags: the module split, the contract
  packages and the three generic fixes in "Generics and build time" cut S4 by 42% and S5 by 59%,
  because those edits recompile 40-57% fewer packages and the packages that recompile define far
  fewer generic functions.
- A to B is stripping the watch builds (`-ldflags='-s -w'`) alone: it halves the link, which is
  most of a small edit's time, and barely moves the large edits.

The runs and the script that summarises them are reproducible with `task build-bench -- --target
<cli|api> --mode air` on each tree; compare medians of several warm runs, since single runs on
this machine varied by up to 2x.

## Not done yet

- The 348 architecture exceptions in `.go-arch-lint.yml` (`TODO(build-arch)`) record imports
  that break the module rules today; each needs an inversion port or a decision before it can
  be removed. The count rose from 319 when the last unassigned packages moved into modules and
  became subject to the module rules.

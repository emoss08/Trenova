# Trenova

Trenova is a transportation management system: Go services, a React client and Rust workers in
one repository. This file is the working agreement for every coding agent (Claude Code loads it
through `CLAUDE.md`). A rule lives here only if it applies to most changes; everything else lives
in the doc for its area, indexed in [docs/engineering/README.md](docs/engineering/README.md).

## How to work here

You are a senior engineer on this codebase, not a transcriber of instructions.

- **Disagree when you have something better.** If the request, the existing design or a rule in
  these docs is worse than an alternative you can name, say so, build the better one and explain
  the trade-off in your report. Agree only when you agree. When the choice is genuinely the
  maintainer's (product behaviour, scope, spend), ask, and lead with your recommendation.
- **Build for scale.** Design every feature for hundreds of thousands of users across thousands
  of tenants: bounded queries, batched reads, an index behind every query path, slow or
  fan-out work on Temporal, idempotent writes, nothing on a request path that grows with the
  size of a tenant's data. [backend/scale.md](docs/engineering/backend/scale.md) is the checklist.
- **No backwards compatibility.** Trenova has not launched. Rename, reshape or delete anything
  (tables, columns, GraphQL fields, routes, workflow payloads, config keys) when the new shape is
  better. Change every caller in the same change and delete the old path: no aliases, shims,
  deprecated fields, dual writes, flags for the old behaviour or workflow version gates.
- **Complete, not minimal.** No "v1", MVP, stub, TODO or "later". A feature ships with its
  validation, authorization, error paths, edge cases, audit, realtime updates, tests and every
  UI state. Don't shrink scope unless told to.
- **Inventive where it pays.** When the obvious design has a known ceiling (N+1 reads, a workflow
  per row, a global lock, polling, a table scan), use the mechanism that removes it (a batch, a
  stream, a rollup table, a debounced workflow, a stored projection) even if nothing like it
  exists here yet. Then write it into the patterns so the next change reuses it.
- **DRY and SOLID, judged against this codebase.** Search before writing a helper, port, query
  or component; reuse or extend what exists. One responsibility per type, depend on ports, keep
  interfaces narrow. Every abstraction has a caller today: generality nobody uses is not scale.
- **Never gate a feature behind the bench.** Ship it as a setting (off by default when it costs
  money or touches data in a new way), and use the bench to measure and recommend.
- **When a rule is wrong, change the rule** in the same change and say so. Never work around a
  rule or a lint silently, and never add an `ignore` without a reason a reviewer would accept.
- **Leave it better.** If code you touch breaks these patterns, fix it when the fix is small and
  related; otherwise name it in your report.

## Repository

| Path | What |
|---|---|
| `services/tms/` | the TMS (Go): API, worker and CLI; hexagonal modules in `internal/modules/<module>/` |
| `services/imaging/`, `services/crates/`, `services/proto/` | Rust imaging worker, shared crates, gRPC contracts (buf) |
| `shared/` | shared Go packages (`stringutils`, `sliceutils`, …); its own module in `go.work` |
| `client/` | pnpm + Turborepo: `apps/web` (TMS), `apps/dash` (driver portal), `packages/{shared,graphql,config}` |
| `desktop/capture/` | Windows scanning and printing companion (Rust) |
| `i18n/`, `dev/`, `deploy/`, `tools/`, `docs/` | translations; local environment; images and self-host; unshipped tooling; docs |

Each top-level directory names a role, not a language: put new code where its role is. GTC, the
change-data-capture pipeline, is a separate project; its routing is `deploy/gtc/sinks.yaml`.

## Commands

From `services/tms/`: `task run-watch` (API, hot reload), `task test`, `task test-integration`
(Docker), `task lint`, `task gqlgen` (never `go tool gqlgen generate`), `task gqlschema-diff`,
`task db-migrate` (`trenova-devdb up`, then `trenova db reconcile`), `task db-seed`, `task db-reset`,
`task db-create-migration name=…`, `task db-stamp`, `task docker-up`, `task quick-start`.
One test: `GOFLAGS=-tags=nomsgpack go test -run TestName ./path/...`.

From `client/`: `pnpm dev`, `pnpm build`, `pnpm lint`, `pnpm typecheck`, `pnpm test`; scope with
`pnpm --filter @trenova/web …`. GraphQL client codegen: `pnpm --filter @trenova/graphql codegen`.

From the root: `task rust:fmt`, `task rust:lint`, `task rust:test`, `task rust:proto`.

## Backend

Read [docs/engineering/backend/README.md](docs/engineering/backend/README.md) and the pattern file
for the layer you are in before writing Go. The rules below are the ones that compile, pass tests
and still break production or the build.

- **Modules.** Code lives in its module; a module imports only strictly lower modules'
  `domain`, `ports` and `<name>contract` packages (`archlint`). Public code never imports
  `internal/cloud`.
- **Tenancy.** Repository methods run in `dbtx.Read`/`dbtx.Write` and query through
  `DBForContext(ctx)` (Bun) or `Keel(ctx)` (keel), never `.DB()`; while Bun remains, start
  transactions with `dbtx`, never `keel.Write` ([keel-migration.md](docs/engineering/backend/keel-migration.md)).
  `trenova-db` runs `SELECT trenova_rls.reconcile()` after every migration run, which protects a
  table with non-null `organization_id` and `business_unit_id`; any other shape calls
  `trenova_rls.apply_policy` or joins `trenova_rls.global_tables` in its migration. Cross-tenant
  code declares `dbscope.WithSystem` and is listed in `rlslint`'s allowlist.
- **Columns** come from `pkg/buncolgen` helpers, never hand-written names.
- **Authorization.** Every root resolver and route reaches a permission check or is listed with a
  reason in `authzlint`; every new write has an agent tool, exemption or `pending` entry in
  `writecoverage.yml`; access changes are audited through `SecurityAuditor`.
- **Generics.** No exported type or signature mentions a concrete generic type that has methods,
  and no widely reachable type holds a `sync.Map`; see "Generics and build time" in
  [module-architecture.md](docs/engineering/module-architecture.md).
- **Go style.** [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md);
  no comments; `sonic`, never `encoding/json`; `errortypes` for errors the caller sees; a params
  struct past three or four parameters; `goimports` and `golines`.
- **Migrations** are keel files in `services/tms/schema/migrations`, applied by `trenova-db`
  (`cmd/db`); add one with `task db-create-migration` and run `task db-stamp` in the same change.
  See "Migrations" in [data-access.md](docs/engineering/backend/data-access.md).
- **Generated code.** Before changing a GraphQL schema, removing a column or editing
  user-facing text, read [generated-artifacts.md](docs/engineering/generated-artifacts.md).
- **The TMS does no document or image computation in process**; it calls the imaging worker
  through `kernelports.Imaging`.

## Frontend

Use the `trenova-frontend` skill (`.claude/skills/trenova-frontend/SKILL.md`; agents without
skills read it directly) for any change under `client/`. Tokens, page anatomy and loading states
are in [design-system.md](docs/engineering/design-system.md); `pnpm lint:design` and
`pnpm lint:skeletons` enforce them.

## Rust

Read [rust.md](docs/engineering/rust.md) before adding or changing Rust, and
[imaging-worker.md](docs/engineering/imaging-worker.md) before touching the worker, its contract
or anything that renders, inspects or assembles a document.

## Before you hand off

Review your own diff for bugs, security holes (injection, missing authz, unvalidated input,
leaked secrets, cross-tenant reads), swallowed errors, duplicated logic, architecture drift,
unbounded work on a request path, and helpers placed outside `shared/` or `lib/`. Fix what you
find. Then report what changed, any rule you changed and why, what you would do next, and the
exact checks the change needs (tests, lints, generators), so the person opening the PR can run
them.

## Commits and pull requests

Concise imperative messages with an optional area prefix (`fix: …`, `permissions: …`). Keep
refactors and behaviour changes in separate commits. A PR says what changed and why, the checks
run, migrations and config changes, and screenshots for UI.

## Do not

- Run tasks that saturate CPU, disk or memory (whole-repo builds in parallel, fat-LTO release
  builds on repeat).
- Run mockery over the whole codebase; adjust the affected mocks.
- Run a formatter over a directory; format the files you changed, by path.

# Reporting

How a report is described, authorized, compiled, run, exported and delivered. Trenova does not
build report SQL itself: the catalog, the definition (IR), the compiler, the runner and its
guards, the formatter and the file writers are keel's `analytics` packages. Trenova owns what it
configures and the product around them: which models are reportable, who may use what
(`reportaccess`), saved and canned reports, runs on Temporal, PDF, schedules, alerts, email,
dashboards, GraphQL and the agent tools.

keel's guide ("Analytics: report definitions and formulas", "formatting and exports", "caching
and rollups", "the engine") and its `docs/analytics.md` describe the IR, the compiler and the
guards themselves. This page is what Trenova decides on top of them. What a person sees is in the
[product guide](../product-guide/reports/index.md).

## The shape of it

```
domain model: keel tags + doc comments ──task keel-gen──▶ <domain>db, internal/keeltables
                                                               │ analytics.Build (startup)
                                                               ▼
                       analytics.Catalog ──go generate──▶ analytics.messages.json, keel-analytics.ts
                               │
GraphQL / REST / agent tool ──▶ reporting.Service.Prepare ──▶ reportaccess.Access (analytics.Policy)
                               │                               + tenant scope + read scope
                               ▼
                 reportengine.Engine ──▶ keel analytics.Runner (limits, cost, slots, cache)
                               │                       │
        Preview, Page, Drill (in the request)          Run (a stream) ──▶ reportjobs (Temporal)
                                                         ──▶ render ──▶ object storage ──▶ email, in-app
```

| Piece | Where |
|---|---|
| Catalog, engine (two keel runners), fx module | `internal/infrastructure/reportengine/` |
| Generators: message catalog and client module | `reportengine/generate.go`, `reportengine/gen` |
| Engine and renderer ports | `internal/modules/workspace/ports/reportports/reporting.go` |
| Repository ports | `reportports/report.go` |
| Access policy: entity kinds, decisions, refusals | `internal/modules/workspace/app/reporting/reportaccess/` |
| Reporting service: prepare, preview, drill, runs, downloads, rows, schedules, views, dashboards, canned reports | `internal/modules/workspace/app/reporting/` |
| Canned reports (embedded JSON) | `app/reporting/canned/reports/*.json` |
| Table export as a report | `internal/modules/workspace/app/tableexportservice/` |
| Models: definitions, revisions, runs, schedules, views, dashboards | `internal/modules/workspace/domain/report/` |
| Repositories | `internal/modules/workspace/adapters/postgres/reportrepository/` |
| Run workflow, schedule dispatch, delivery, cleanup | `internal/modules/workspace/jobs/reportjobs/` |
| Renderers: CSV, JSON, XLSX (keel), PDF (Gotenberg), dashboard workbook (excelize) | `internal/infrastructure/reporting/render/` |
| Reporting connection pool | `postgres.ReportingConnection` (`internal/infrastructure/postgres/connection.go`) |
| Configuration and defaults | `reporting:` in config, `config.ReportingConfig` |
| GraphQL | `internal/api/graphql/schema/report.graphqls`, `resolver/reportresolver/` |
| REST: download, dashboard export | `internal/api/handlers/reporthandler/` |
| Agent tools | `agent/app/agentquerytoolservice/report*.go`, `agent/app/agenttools/{reporttools,alerttools,masterdatatools}` |
| Client: generated IR module, builder, grid, dashboards | `client/packages/shared/src/generated/keel-analytics.ts`, `client/apps/web/src/routes/reports/`, `lib/report-definition.ts` |

## The catalog

`reportengine.NewCatalog` builds the catalog once at startup: `analytics.Build(keeltables.All(),
…)` over every generated table, keeping those whose table tag carries `report`. Nothing about a
field is declared twice: keel gen derives labels, types, measures, aggregations, enum values and
edges from the model, and a tag option is written only where the derived default is wrong. A
graph error (an edge to a model that is not reportable, a bad pick, two entities with one key)
is a `catalog` error and the process does not start.

81 models are reportable, by category: Accounting 16, Billing 20, Configuration 2, Equipment 10,
Freight 2, Operations 6, Parties 13, Payroll 10, Reporting 2. `reportaccess/kinds.go` has one row
for each (see [Access](#access)).

### Keys

- **Entity key** is the Go type name in snake case: `Shipment` is `shipment`, `ShipmentMove` is
  `shipment_move`, `ReportDefinition` is `report_definition`. It is unique across every
  reportable model.
- **Field key** is the Go field name in lower camel case: `TotalChargeAmount` is
  `totalChargeAmount`.
- **Edge name** is the relation's field name in lower camel case: `Customer` is `customer`,
  `OriginStop` is `originStop`. A definition names a field by its path from the entity:
  `customer.name`, `moves.stops.id`.

Saved definitions, canned reports, dashboards and the agent all store these keys, so renaming a
reportable Go type, field or relation renames a catalog key.

### Tag options

The storage options (`pk`, `tenant`, `nullzero`, `version`, relations) are keel's; these are the
report ones, each with a use from the models:

| Option | Does | Example |
|---|---|---|
| `report=<Category>` on the table tag | makes the model an entity, in that category | `keel.Table` `keel:"shipments,report=Operations"` (`operations/domain/shipment`) |
| `label=` | replaces the label derived from the name | `keel:"external_reference,nullzero,label=Customer Reference"`; on a table, `keel:"report_definitions,report=Reporting,label=Report"`; on a relation, `keel:"belongs:trailer_type_id,label=Trailer Equipment Type"` |
| `format=` | the display kind: `money`, `weight`, `distance`, `count`, `percent` | `keel:"total_charge_amount,label=Total Charge,format=money"`, `keel:"weight,format=weight"`, `keel:"deadhead_goal,format=percent"` (`fleetcode`) |
| `noreport` on a field | keeps the column out of the catalog | `keel:"idempotency_key,nullzero,noreport"`, `keel:"definition,jsonb,noreport"` |
| `noreport` on a relation | drops the edge | `Owner *tenant.User` `keel:"belongs:owner_id,noreport"` |
| `dimension` | groups by a numeric column instead of summing it | `keel:"current_revision,label=Revision,format=count,dimension"`, `keel:"period_number,label=Period,dimension"` |
| `type=epoch` | an `int64` time column whose name does not end in `_at`, `_date` or `_time` | `keel:"period_start,type=epoch"` (`invoice`), `keel:"adjustment_deadline,type=epoch"` (`fiscalperiod`) |

`-acronyms` in `KEEL_GEN_ARGS` (`services/tms/Taskfile.yml`) decides how names split into
labels (`ProNumber` is "PRO Number"). Tenant, version and bytea columns and the foreign keys a
relation already exposes are left out without a tag.

### Descriptions are doc comments

The doc comment on a reportable model is its catalog description, and so are the doc comments on
its fields and its enum constants. They are the one place a Go comment is wanted
([backend/go.md](backend/go.md)):

```go
// Freight shipments and their lifecycle state
type Shipment struct {
	keel.Table `keel:"shipments,report=Operations"`
```

becomes `"shipment@description": "Freight shipments and their lifecycle state"` in
`analytics.messages.json`, and the builder, the catalog query and the agent's dataset tools show
it. Write it for a person choosing a dataset.

### Relations are edges

Every `belongs:`, `many:` and `pick:` relation of a reportable model is an edge unless it is
`noreport`. A `many:` edge is to-many: its fields are aggregated, never grouped by. A `pick:` edge
picks one row of a list: `OriginStop` is
`keel:"pick:moves.stops,first=moves.sequence|stops.sequence"`. **An edge must lead to a
reportable model.** A relation to one that is not (a `tenant.User`, a formula template) fails
`analytics.Build` until the relation carries `noreport` or its target is made reportable.

### Labels and languages

Labels and descriptions are translation keys with the derived English as fallback
(`shipment.totalChargeAmount`, `shipment.status.InTransit`, `@category.Operations`).
`reportengine.Translate` looks the English text up in the request's language through
`shared/i18n`. `go generate ./internal/infrastructure/reportengine/...` writes every one of them
to `analytics.messages.json`, which `task i18n` extracts into the catalogs like any other message
([i18n/README.md](../../i18n/README.md)).

`Catalog.Version()` hashes everything a stored definition depends on. A definition and each of
its revisions store the version they were saved against (`catalog_version`).

## Access

`reportaccess` is the catalog's `analytics.Policy` and the only place report access is decided.
`reportaccess.New` maps each catalog entity to a permission resource through `entityKinds`
(`kinds.go`). It fails, and with it `reporting.Service` and the process, when an entity has no
row, names an unregistered resource or names an ownership column the table lacks.

A row says three things:

- `resource`: the kind whose grant decides the entity (`"customer": {resource:
  permission.ResourceCustomer}`);
- `withinRecord`: the entity is part of another record (`invoice_line` is part of an invoice).
  It is decided by the parent's grant, never uses a reference view, and its protected-group path
  is prefixed with the parent's edge to it;
- `ownership`: the column an `own` data scope narrows by (`shipment`, `order` and
  `report_definition` name `owner_id`).

`Resolver.For(ctx, tenant, mode)` loads the caller's grants on every report kind in one
`GetResourcePermissionsBatch` call and returns the `*Access` for one request. keel asks it once
per entity, field and edge the request touches.

| Decision | When |
|---|---|
| `available` | allowed; the field carries its registry sensitivity, which every output column reading it inherits |
| `needs_kind_read` | the caller cannot read the kind and the field or edge is not in its [reference view](reference-views.md), or the report's root kind is unreadable |
| `withheld_by_ceiling` | the field's sensitivity is above the caller's ceiling for the kind |
| `needs_field_group` | the field is in a protected field group the caller's role does not grant |
| `needs_export` | export mode, and the caller reads the kind without its `export` |
| `own_scope_unsupported` | the caller's scope on the kind is `own` and the kind has no ownership column |

- **Read and export modes.** Previews, drill-through and saving a definition compile in
  `ModeRead`. Runs, the CSV stream, dashboard exports and serving a stored run compile in
  `ModeExport`, which also needs each kind's `export` (reference-view fields excepted).
- **Rows.** On an entity with an ownership column, `Access.Rows` adds `owner_id = <caller>` when
  the caller's scope on its kind is `own`. keel applies it to every reference a statement makes
  to the table: the base, joins, picks, EXISTS and pre-aggregations. The reporting pool is under
  row-level security like the main one, so the `read_scope` policies on shipments, orders,
  invoices and workers narrow as well ([permissions.md](permissions.md#data-scopes)).
- **Tenancy** is not the policy's job. `Service.Prepare` puts `dbscope.WithTenant` on the
  context, and keel applies the tenant columns to every table reference by construction.
- **Refusals.** keel returns `denied` with the decision as its reason; `Access.Refusal` turns
  the request's denials into one translated authorization error ("you do not have read access
  to Customers, which "Credit Limit" on Invoices needs; …").
- **One decision everywhere.** The engine, the `reportCatalog` query (`CatalogFor`, which
  returns `FieldUse` and `EdgeUse` for every field and edge) and the agent's dataset tools all
  ask `reportaccess`, so the builder offers exactly what the compiler accepts.
  `TestReportSurfacesDecideAccessOnlyThroughReportaccess` fails if a report surface calls a
  permission primitive (`CanAccess`, `Decide`, `InReferenceView`, `ProtectedGroupAt`, …) itself.

## The engine

`reportengine.Engine` implements `reportports.ReportEngine` over two keel runners on the
reporting pool, built when fx starts:

- **interactive**, with no result cache: `Preview`, `Page`, `Count`, `Drill`, `Compile`, and
  `Run` for a request whose read scope narrows (an `own` or `team` grant, so `readscope.Encode`
  is not empty);
- **cached**, with a Redis result cache (prefix `report:analytics:`, TTL `resultCacheTtl`):
  `Run` for every other request. Without Redis, both are the interactive runner.

The cache key covers the statement and its arguments (the tenant and the policy's row rules
among them), the output, and keel's data version of every table the statement reads, which keel
bumps after a write through it commits
([backend/keel-migration.md](backend/keel-migration.md#transactions)). The read scope is a
session setting rather than an argument, which is why a narrowed run never reads the cache. The
TTL is the backstop for writes keel does not see.

No rollups are declared and the runners have no rollup router, so every report reads its base
tables. `Primary: true`: reports run on the primary, not a replica.

### Preparing a request

`Service.Prepare` is the way into the engine. It resolves the caller's `Access`, the time zone
(the organization's, or `UTC`), `Now`, the tenant scope, and the supersede key: a preview sent
with `supersede: true` carries `preview:<userID>`, so starting one cancels that person's previous
preview in every process. `Service.Check` prepares and compiles without running. It validates a
save, a canned run, a stored run before it is served, and an agent's proposal
(`ValidateDefinition`).

### Guards and limits

Every entry point passes keel's guards. Trenova sets them from `reporting:`:

| Key | Default | Bounds |
|---|---|---|
| `maxDimensions`, `maxPivotColumns`, `maxPathDepth`, `maxDefinitionLimit` | 6, 50, 3, 100,000 | the definition (`ir.Limits`) |
| `maxToOneJoins`, `maxToManySubqueries` | 6, 3 | compiled complexity |
| `explainCostLimit`, `explainRowLimit` | 5,000,000, 10,000,000 | `EXPLAIN` total cost and estimated rows |
| `maxConcurrentStatementsPerOrg` | 4 | report statements one tenant runs at once on a server (advisory-lock slots, a 2 s wait, then `busy`) |
| `previewStatementTimeout` | 10 s | Preview, Page, Count and Drill |
| `statementTimeout` | 5 min | Run, and the reporting pool's session `statement_timeout` |
| `previewRowLimit` | 100 | preview rows; the most a page or a drill-through returns |
| `maxRows` | 5,000,000 | rows a run streams |
| `poolMaxOpenConns` | 4 | connections in the reporting pool |
| `resultCacheTtl` | 15 min | cached results |

`EngineError` maps keel's codes to what the caller sees: `denied` to the refusal above;
`invalid_definition` to one validation error per JSON pointer
(`definition.columns[2].dimension.field`, with keel's "did you mean" suggestions); `parameter`
and `cursor` to validation errors; `not_allowed`, `too_complex`, `too_costly` and `timeout` to
business errors; `busy` to a rate limit; `superseded` to `context.Canceled`.

### The reporting pool

`postgres.ReportingConnection` is a second pool on the same database and role, under the same
row-level security guard as the main pool, with its own size (`poolMaxOpenConns`) and
`statementTimeout` as its session timeout, so a long report never holds a connection the API
needs.

## Definitions and the records around them

| Table | Model | What |
|---|---|---|
| `report_definitions` | `ReportDefinition` | a saved report: name, owner, `private` or `shared`, status, kind (`custom`, `canned_fork`), the IR as JSONB, `catalog_version` |
| `report_definition_revisions` | `ReportDefinitionRevision` | one per save; a run binds the head revision, so it runs what was saved when it was queued |
| `report_runs` | `ReportRun` | one run: format, parameters, status, row count, artifact and rows keys, expiry, Temporal ids |
| `report_schedules` | `ReportSchedule` | cron, zone, formats, delivery, alert, `run_as_id` |
| `report_views` | `ReportView` | a named set of parameter values for one report |
| `report_dashboards` | `Dashboard` | a layout of tiles, each pointing at a saved or canned report; a tile never holds a definition of its own |

- **Visibility.** A definition is read by its owner or, when `shared`, by everyone in the tenant;
  only the owner updates, deletes or resets it. `ListDefinitions` filters visibility in the
  query.
- **Canned reports** are the JSON files in `canned/reports/`, embedded and loaded at startup
  (`canned.Default`); each file's name is its key. `forkCannedReport` copies one into a
  `canned_fork` definition and `resetCannedFork` puts the built-in definition back.
- **Status** is `draft`, `active`, `archived` or `needs_attention`. A run refuses `archived` and
  `needs_attention`; a schedule runs only an `active` definition. Nothing sets `needs_attention`
  (`UpdateStatus` has no caller), and nothing sweeps definitions whose `catalog_version`
  differs from the catalog's.
- `report_definitions` and `report_runs` are themselves reportable (category `Reporting`).

## Runs and exports

### A run

`runReport`, the agent's `report_run`, a table export and every schedule create a `queued` run
and start `RunReportWorkflow` on `report-queue` with the workflow id `report-run/<runID>`. Before
it enqueues, `enforceEnqueueGate` refuses a tenant with `maxConcurrentRunsPerOrg` (2) runs
running or `maxQueuedRunsPerOrg` (10) queued. A canned run is compiled in `ModeExport` before it
is queued.

The workflow:

1. `PrepareRunActivity` marks the run `running`, resolves its revision or canned entry, and
   checks it as the requester in `ModeExport`. A validation or authorization failure is not
   retried.
2. `ExecuteAndRenderActivity` attaches the requester's read scope (`readscope.With`; a run that
   cannot be narrowed is refused), prepares in the requester's locale and the organization's
   zone, calls `Engine.Run`, and streams each batch to the format's renderer and, alongside, to
   the JSON rows sidecar. It heartbeats every 5,000 rows, and its timeout is `maxRunDuration`
   (30 min). `busy` is retried after 15 s; a cost, row-limit or size failure is not retried.
3. `FinalizeRunActivity` records the outcome and the artifact's expiry (`artifactRetention`, 7
   days), writes an audit entry, notifies the requester, publishes a `report-run` realtime
   invalidation and updates the schedule's failure streak. A failed or canceled workflow still
   finalizes, on a disconnected context, so a run never stays `queued` or `running`.
4. A scheduled run then delivers (below). A delivery failure never fails the run.

The artifact is `<artifactPrefix>/<orgID>/<runID>/<attempt>/report.<ext>` and the sidecar is the
same key plus `.rows.json`. Each upload is capped near `maxArtifactBytes` (1 GiB).

| Format | Renderer |
|---|---|
| `csv` | keel `export.CSV`: formatted cells, formula-injection escaping, a BOM when `csvIncludeBom` |
| `xlsx` | keel `export.XLSX`: streaming, native numbers and dates, an info sheet with the parameters in declared order |
| `json` | keel `export.JSON`: the raw-value envelope, the same shape as the sidecar |
| `pdf` | an HTML table rendered by Gotenberg (`documenttemplateports.PDFRenderer`); more than `pdfMaxRows` (5,000) detail rows fails the run without a retry |

Every renderer formats through each column's resolved display, in the run's locale and zone: the
contract the browser formats with too
([rules-engine.md](rules-engine.md#report-formatting-report)).

### Reading a run back

- **Download** (`GET /api/v1/reports/runs/:runID/download/`, report `export`) redirects to a
  60-second presigned URL and is audited as `artifact_downloaded`.
- **Rows** (`Service.ReadRunRows`) read the sidecar, at most 24 MiB, as an `export.Envelope`,
  audited as `rows_read`. It is the only way to read a run's data on the server; the agent's run
  comparison and data analysis use it.
- Both serve only the requester's own run, and both compile its definition again against the
  requester's grants as they are now (`authorizeStoredRun`), so access withdrawn since the run
  withdraws its file and its rows.

### Synchronous exports

- **Dashboard export** (`POST /api/v1/reports/dashboards/:dashboardID/export/`, report
  `export`) previews each tile in `ModeExport` and builds one XLSX workbook in memory with
  excelize, a sheet per tile. A tile's sheet holds at most `previewRowLimit` rows.
- **`Service.StreamCSV`** runs a definition and streams CSV to a writer; the assistant's artifact
  export uses it.
- **Table export** (`exportTableView`, `scheduleTableView`) saves what a data table shows as a
  private definition on the table's root entity (`tableexportservice.Build`: the shown columns
  as dimensions, the table's filters and sort; a search query is refused, and a to-many or
  ungroupable column is skipped and named), then runs or schedules it like any report.

### Maintenance

`reportjobs.ScheduleProvider` registers three Temporal schedules: dispatching due report
schedules every 5 minutes; deleting expired artifacts and sidecars hourly, which marks the run
`expired`; and, hourly, failing runs left `queued` or `running` for more than `maxRunDuration`
plus an hour (`ZOMBIE`). Their cross-tenant reads are `dbscope.WithSystem`.

## Schedules, alerts and delivery

- **Dispatch** takes up to 200 due schedules a tick. Each advances its `next_run_at` first, so a
  broken schedule cannot fire every tick, then queues one run per format as `run_as_id`. A
  definition that cannot be loaded or is not `active` adds to the failure streak; five in a row
  disable the schedule and notify its owner.
- **Alerts** hold a delivery back unless the run's row count (`threshold`) or, with `columnId`,
  a measure's grand total (`value`) satisfies the operator. A measure alert or an inline email
  table makes the run keep a digest: the first 20 rows (`MaxDigestRows`) and the total, handed to
  delivery through the workflow and never stored. `suppressWhileFiring` mails only the
  transition into firing.
- **Email** renders the organization's `report.delivery.email` template (falling back to the
  built-in), attaches the artifact up to `emailMaxAttachmentBytes` (10 MiB) and sends with the
  idempotency key `report-run-<runID>`. A template or configuration failure is reported to the
  schedule's owner instead of retried. The email's link uses `deliveryLinkBaseUrl`.
- **In-app** delivery notifies `notifyUserIds` (the owner already has the run notification).
  Recipients must be members of the tenant; a schedule takes at most 20 email and 20 in-app
  recipients.
- Only a schedule's owner (`run_as_id`) changes or deletes it. `reportSchedules` lists a person's
  own schedules and those on reports they own.

## GraphQL and REST

Every root resolver requires the `report` resource: `read` for the catalog, the lists, previews,
drill-through, views and canceling a run; `create`, `update` and `delete` for definitions and
dashboards; `export` for `runReport` and the schedule mutations. `exportTableView` and
`scheduleTableView` require `export` on the table's own resource. Agent coverage of each mutation
is in `writecoverage.yml` ([agent-write-coverage.md](agent-write-coverage.md)).

| Queries | Mutations |
|---|---|
| `reportCatalog`, `cannedReports`, `reportDefinition`, `reportDefinitions`, `reportDefinitionRevisions`, `reportRun`, `reportRuns`, `previewReport`, `drillThroughReport`, `reportSchedules`, `reportViews`, `reportView`, `reportDashboards`, `reportDashboard` | `create`/`update`/`deleteReportDefinition`, `forkCannedReport`, `resetCannedFork`, `runReport`, `cancelReportRun`, `create`/`update`/`deleteReportSchedule`, `create`/`update`/`deleteReportView`, `create`/`update`/`deleteReportDashboard`, `exportTableView`, `scheduleTableView` |

- **A definition is `JSON`**, keel IR version 2, never a GraphQL type tree.
  `reporting.DecodeDefinition` decodes and validates it (`ir.Decode`, `ir.Validate`); the engine
  then resolves it against the catalog.
- **`reportCatalog`** is the caller's view: `version`, `rules` (keel's `RuleSet` as JSON:
  operators per value type, aggregations, buckets, transforms, windows, charts, the display
  vocabulary, formula functions, limits) and every entity with `readable` and `exportable` and,
  per field and edge, `access` (`ReportFieldAccess`), `inReferenceView`, `protection` and
  `needsReadOf`.
- **`ReportPreview`** carries the columns with their resolved `display` and `sensitivity`, raw
  `rows`, `levels`, `totals`, `truncated`, `cursor` (when paged with `pageSize`), `timezone`,
  `rollup` and `cached`.
- `runReport` with `viewId` takes the view's report, parameters and format on the server;
  explicit `params` or `format` still win.

## The client

- `client/packages/shared/src/generated/keel-analytics.ts` is keel's IR module, written by
  `go generate ./internal/infrastructure/reportengine/...`: the definition types (`Definition`,
  `Column`, `Filter`, `Parameter`, `DashboardFilter`, …), the operator, chart, transform and
  formula tables, `LIMITS`, `DEFINITION_SCHEMA` and `applyFilters`. Never edit it.
- The builder edits its own model (`ReportIR`, `types/report.ts`). `lib/report-definition.ts`
  converts it to and from the keel `Definition` the server stores, and carries through untouched
  what the builder cannot edit (windows, top-N, subtotals, free formulas). The operations are in
  `client/packages/graphql/src/operations/reports/`; their types come from
  `pnpm --filter @trenova/graphql codegen` over the server schema.
- Cells are formatted from each column's `display` by the rules module
  (`@trenova/shared/rules/report`).
- A dashboard tile loads its report and calls `previewReport` itself. It folds the dashboard's
  filters into the report with its own `applyDashboardFilters`
  (`routes/reports/dashboards/_components/dashboard-state.ts`) on `ReportIR`; the server's
  dashboard export folds them with keel's `ir.ApplyFilters` (`ResolveDashboardTiles`).

## Agent tools

The tools are a schema over `reporting.Service`; none of them builds a query or decides access.

| Tools | Where |
|---|---|
| `report_list`, `report_get`, `report_list_datasets`, `report_get_dataset`, `report_preview`, `report_run`, `report_get_run`, `report_compare_runs`, `report_list_runs`, `report_list_schedules`, `report_list_dashboards`, `report_compose_table_view` | `agentquerytoolservice` |
| `report_create`, `report_update`, `report_fork`, `report_delete`, `report_reset_fork`, `report_cancel_run`, `report_update_schedule`, `report_delete_schedule`, `report_delete_dashboard` | `agenttools/reporttools` |
| `report_create_schedule` | `agenttools/alerttools` |
| `report_create_dashboard`, `report_add_dashboard_tile`, `report_save_table_view` | `agenttools/masterdatatools` |
| `report_analyze_data` | [agent-data-analysis.md](agent-data-analysis.md) |

- **Definitions.** `report_create` and `report_update` take the definition as JSON constrained by
  keel's JSON Schema (`report.DefinitionToolSchema`) and check it with `ValidateDefinition`, as
  the person, before anyone approves it.
- **Datasets.** `report_list_datasets` and `report_get_dataset` read `CatalogFor`, so an agent
  sees the access decisions the builder sees.
- **`report_run`** asks for report `export` itself (a query tool is otherwise authorized as
  `read`). It corrects only the container shape of the parameters (`reportparams.go`: a wrapped
  list is unwrapped, a scalar becomes a one-item list for a multi-valued parameter and a
  one-item list a scalar for a single-valued one; a value is never changed), refuses a missing
  required parameter with its allowed values, and waits a few seconds for a quick run before it
  returns.
- **`report_preview`** returns at most 20 rows.
- The `report_*_table_change_alert` tools belong to table change alerts, a separate feature
  ([agent-master-data-tools.md](agent-master-data-tools.md)). Each tool's class and default are in
  [ai-tool-safety.md](ai-tool-safety.md).

## Adding a reportable model or field

1. **Tag the model** `report=<Category>` on its `keel.Table` tag, in an existing category unless
   a new one is meant, and write its doc comment for a person choosing a dataset.
2. **Fix what derivation gets wrong**: `label=`, `format=`, `dimension`, `type=epoch`, and
   `noreport` on internal columns (idempotency keys, hashes, raw JSON) and on every relation to a
   model that is not reportable.
3. **Generate** with `task keel-gen` (`<domain>db/keel_gen.go`, `internal/keeltables`,
   `pkg/keelkinds`).
4. **Give it a kind**: a row in `reportaccess/kinds.go` with its resource, `withinRecord` when
   it is part of another record, and `ownership` when its kind has an `own` scope.
5. **Check the reference view** of its resource (`resources_*.go`) names the fields a reader of
   a linking record may see ([reference-views.md](reference-views.md)).
6. **Regenerate** with `go generate ./internal/infrastructure/reportengine/...`, then run
   `task i18n` from the repository root for the new labels and descriptions.
7. **Commit** the regenerated `analytics.messages.json` and `keel-analytics.ts` with the model.

A new field on a reportable model needs steps 2, 3, 6 and 7. Renaming or removing a reportable
field changes a catalog key: update every canned report in `canned/reports/*.json` that names
the old path in the same change.

## Generators and checks

From `services/tms/`:

```bash
task keel-gen                                          # <domain>db packages, internal/keeltables, pkg/keelkinds
go generate ./internal/infrastructure/reportengine/... # analytics.messages.json, client/.../generated/keel-analytics.ts
GOFLAGS=-tags=nomsgpack go test ./internal/infrastructure/reportengine/... \
  ./internal/modules/workspace/app/reporting/... ./internal/modules/workspace/jobs/reportjobs/... \
  ./internal/infrastructure/reporting/render/... ./internal/api/graphql/resolver/reportresolver/...
```

Then `task i18n` and `task i18n-check` from the root, and `pnpm --filter @trenova/graphql codegen`
from `client/` after a change to `report.graphqls`. `task keel-gen-check` and the `Codegen Checks`
job ([generated-artifacts.md](generated-artifacts.md)) fail when a generated file is stale.

| Test | Fails when |
|---|---|
| `TestEveryReportableEntityHasAKind` (`reportengine`) | a reportable model has no row in `kinds.go`, or names an unregistered resource or a missing ownership column |
| `TestClientDefinitionModuleIsCurrent`, `TestMessageCatalogIsCurrent` (`reportengine`) | `keel-analytics.ts` or `analytics.messages.json` is stale |
| `TestCatalogReadsInTheRequestersLanguage` (`reportengine`) | the Spanish catalog has no translation for the customer entity's plural |
| `TestReportSurfacesDecideAccessOnlyThroughReportaccess`, `TestTheEngineAndTheCatalogShareTheDecision` (`reportaccess`) | a report surface decides access itself, or the engine or the catalog stops calling the shared decision |
| `TestServerFormattingMatchesTheBrowsersContract` (`render`) | server formatting departs from the browser's contract |
| `TestBuiltInLayoutsReferenceRealCannedReports` (`homelayoutservice`) | a home layout preset names a canned report that does not exist |

## Rules that bite

- **A relation from a reportable model to one that is not reportable stops the process at
  startup.** Mark it `noreport`. `go test ./internal/infrastructure/reportengine/...` shows the
  failure before a deploy does.
- **A new reportable model needs its row in `kinds.go`**, or `reporting.Service` cannot be
  built.
- **Decide report access only in `reportaccess`.** A report surface that checks a permission, a
  reference view or a field group itself fails the guard test, and the builder would stop
  matching the compiler.
- **Run a tag change through both generators.** `task keel-gen` alone leaves the client module
  and the message catalog stale, and CI fails.
- **Catalog keys are Go names.** Renaming a reportable type, field or relation renames what
  stored definitions, canned reports and dashboards refer to, and they stop compiling until they
  are updated. No test resolves the canned reports against the catalog.
- **Every way into the engine goes through `Service.Prepare`**, which attaches the policy, the
  tenant scope and the zone. A run on a worker also needs the requester's read scope
  (`readscope.With`), or own- and team-scoped grants would not narrow it.
- **Use `ModeExport` for anything that leaves as a file** (runs, streams, dashboard exports,
  serving a stored run); `ModeRead` is for what is shown on a screen.
- **Dashboard filter semantics live in two places**: `ir.ApplyFilters` on the server (dashboard
  export) and `applyDashboardFilters` in the client (tiles on screen). Change both together.
- **A dashboard export is built in memory and holds at most the preview rows per tile.** Large
  data belongs in a run, which streams.
- **`ReportRun.cacheHit` and `ReportPreview.cached` are always false**: previews use the
  uncached runner, and a run does not record whether its stream came from the cache.

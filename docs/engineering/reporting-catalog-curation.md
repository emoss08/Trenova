# Reporting Catalog Curation Guide

How to make a domain entity reportable — and what the generator refuses to let
you get wrong.

## How the catalog works

The report builder and compiler operate exclusively on a generated semantic
catalog (`services/tms/pkg/reportcatalog/catalog_gen.go`). Nothing in the
reporting engine ever takes an identifier from user input: every table, column,
and join pair in emitted SQL comes from this catalog, which is generated at
build time from two inputs:

1. **Domain bun tags** — SQL identity (table name, alias, column names, types,
   nullability, relation join pairs, m2m through-tables, enum values) is parsed
   from the Go structs in `internal/core/domain/`. It is never hand-written.
2. **The curation manifest** — `internal/infrastructure/database/reportcatalog/reportcatalog.yml`
   selects *which* entities/edges are exposed and supplies display metadata.

Entities are **opt-in**, fields are **opt-out**. A forgotten inclusion is a
missing feature; a forgotten exclusion would be a silent data leak — the
asymmetry is deliberate.

Field-level sensitivity is *not* stored in the catalog. It resolves at
authorize time from `permission.Registry.FieldSensitivities` — the permission
registry stays the single source of truth (see "Classify sensitive fields"
below).

## Making an entity reportable

### 1. Add the entity to the manifest

```yaml
entities:
  my_entity:                      # catalog key — snake_case, stable forever
    struct: mypkg.MyEntity        # Go type as <package>.<Type> under core/domain
    resource: my_entity           # permission.Resource — MUST exist in the registry
    label: My Entity
    pluralLabel: My Entities
    description: One line shown in the builder's catalog browser
    category: Operations          # builder grouping (Operations/Billing/Compliance/...)
    ownershipColumn: owner_id     # optional — enables DataScopeOwn (see below)
    excludeFields: [internalNotes] # opt-out fields by JSON name
    fields:                       # per-field overrides (all optional)
      totalAmount: { label: "Total", format: money }
      dob: { label: "Date of Birth", type: epoch }
      status: { enumLabels: { InTransit: "In Transit" } }
      rawPayload: { filterable: false, groupable: false }
    edges:                        # opt-in relations by the Go field's JSON name
      customer: { label: Customer }
      lineItems: { label: "Line Items" }
      legacyRef: { traversable: false }  # visible in graph docs, not walkable
```

Field manifest keys: `label`, `description`, `type` (override inference —
mainly `epoch` for BIGINT timestamps the parser can't classify), `format`
(display hint: `money`, `weight`, `percent`, ...), `enumLabels`,
`aggregations` (can only *remove* legal aggregations, never add),
`filterable`, `groupable`.

### 2. Mind the tenancy rules — the generator enforces them

- Every reportable entity must carry **both** `organization_id` and
  `business_unit_id` columns, or **neither** (untenanted reference data like
  `us_state`). An entity with exactly one is a **hard build error** — the
  compiler cannot tenant-scope it safely. This is why `user` is not in the
  catalog v1: it carries `business_unit_id` but not `organization_id`.
- The compiler stamps org/BU predicates on *every* alias in every query (join
  ON clauses, lateral subqueries, EXISTS, m2m through-tables). You get this
  for free; you cannot opt out.

### 3. Ownership column (DataScopeOwn)

If users can be granted "own records only" scope on this resource, set
`ownershipColumn` to the column holding the owning user's ID. When a runner
has `DataScopeOwn`, the compiler adds `t<i>.<ownershipColumn> = ?` bound to
their user ID. If the entity declares **no** ownership column, own-scoped
runners are **denied** (fail closed) — never silently widened.

### 4. Edges

- Edge names are the Go relation field's **JSON name** — that's how report
  definitions reference joins (`["customer"]`, `["moves","assignment"]`).
  Renaming a relation field is a breaking change for saved definitions.
- Only declare an edge if its **target entity is also in the manifest**; the
  generator fails otherwise.
- Cardinality comes from the bun tag: to-one edges are usable in dimensions,
  filters, and sorts; to-many edges are measure/EXISTS-only (the compiler
  pre-aggregates them in `LEFT JOIN LATERAL` so the primary grain never
  inflates — SUM cannot double-count).
- Do **not** add edges to half-tenanted or unregistered targets (e.g. the
  `owner`/`enteredBy` user edges are deferred for this reason).

### 5. Classify sensitive fields

Reporting makes field-level sensitivity load-bearing. If the entity carries
PII or financial data, add explicit `FieldSensitivities` to its resource in
`internal/core/domain/permission/registry.go` (`confidential` for identity
documents/DOB, `restricted` for PII and money, `internal` for operational
data). The generator **warns** on unclassified fields of `restricted`+
resources; the authorize stage fails closed when a runner's `MaxSensitivity`
cannot access a referenced field.

### 6. Regenerate and verify

```bash
task generate-reportcatalog          # regenerates pkg/reportcatalog/catalog_gen.go
task generate-reportcatalog-check    # CI guard: regen + git diff --exit-code
task test-integration                # TestCatalogMatchesLiveSchema validates every
                                     # catalog column against information_schema
```

The catalog `Version` is a content hash; saved definitions and revisions store
the version they were authored against.

### 7. Renames and removals

Renaming or removing a reportable field/edge breaks saved definitions that
reference it. Today this is caught fail-closed at the two validation points:
saving a definition re-validates against the current catalog, and `PrepareRun`
re-validates before execution — a stale definition fails its run with
structured field-level diagnostics (never wrong data), and scheduled runs
feed the schedule's auto-disable streak. The definition rows carry `status`
(`needs_attention`) and `diagnostics` columns as the vehicle for the designed
deploy-time reconcile job (auto-rewrite via a manifest rename map, flagging
removals proactively instead of at run time) — that job is not built yet, so
treat catalog renames as breaking changes and coordinate them deliberately.

## Leakage suite

`internal/core/services/reporting/compiler/leakage_integration_test.go` seeds
two organizations with canary values and executes the full corpus (tripwire,
integration, and every canned report) in both directions, asserting zero
cross-tenant cells. If you add an entity with seedable fixtures, extend the
corpus with a definition that selects its string fields — the suite asserts
its own canaries stay reachable, so it can never pass vacuously.

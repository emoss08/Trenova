# Canned Report Authoring Guide

Canned reports are the product-shipped report library: code-resident, compiled
against the current catalog in CI, and forkable per tenant.

## Why code, not seeds

Canned definitions live in a Go registry
(`internal/core/services/reporting/canned/registry.go`), not in tenant
databases. The registry is instantiated and compiled against the live catalog
by a CI gate test (`canned/registry_test.go`) — schema drift breaks the build,
not the customer. Seeded rows would strand stale copies in every tenant DB.

## Adding a canned report

### 1. Write the entry

Add a constructor to `registry.go` and register it in `Default()`:

```go
func lateDeliveries() *Entry {
    return &Entry{
        Key:           "late-deliveries",        // stable forever — used as provenance
        Version:       "1.0.0",                  // semver of THIS definition
        Name:          "Late Deliveries",
        Description:   "Shipments delivered outside their scheduled window",
        Category:      "Operations",
        Tags:          []string{"shipments", "service"},
        DefaultFormat: report.FormatXLSX,
        Definition:    &report.Definition{ /* IR */ },
    }
}
```

### 2. IR authoring rules

The definition is ordinary compiler IR — everything the builder can express,
subject to the same validation:

- **One primary entity = the grain.** Dimensions, filters, and sorts may only
  traverse to-one edges; measures over to-many paths compile to lateral
  pre-aggregation (see `order-revenue-summary` for a cross-shipment SUM).
- **Any measure ⇒ every non-measure column is a GROUP BY dimension.**
- **Parameters over hardcoded values.** Use `ParameterDef` + `Param`-bound
  filters (`windowDays`, `horizonDays` are established conventions) so tenants
  can run the report without editing it. Give every required parameter a
  `Default` — the leakage suite and preview both execute canned entries with
  their declared defaults.
- Reference only catalog fields the target audience can plausibly access;
  runners are authorized per referenced entity/field at run time, and a runner
  lacking access gets a structured failure, never silently narrowed results.
- Reuse the file's existing constants (`windowDaysParam`, `statusFieldKey`,
  ...) — goconst is enforced.

### 3. Versioning

Bump `Version` whenever the definition changes meaningfully. The version is
recorded on every run (`canned_key` + `canned_version`) and on every fork, so
run history stays reproducible and the UI can show "based on v1.0.0 — v1.1.0
available" on customized copies.

### 4. Verify

```bash
go test ./internal/core/services/reporting/canned/...      # CI compile gate
go test -tags integration -run TestNoCrossTenantLeakage \
  ./internal/core/services/reporting/compiler/             # executes every canned
                                                           # entry on real Postgres
```

The leakage suite automatically picks up new registry entries — every canned
report is executed as two different organizations with canary data and
asserted to return zero cross-tenant cells.

## How canned reports behave at run time

- **Run as-is**: the run row stores `canned_key`/`canned_version` and no
  definition ID; the worker resolves the IR from the registry at
  `PrepareRun`, so a deployed binary always runs its own catalog-compatible
  copy.
- **Customize**: `forkCannedReport` materializes the IR into a tenant
  `report_definitions` row (`kind = canned_fork`) with provenance. Forks are
  ordinary custom definitions to the engine — no special-case paths.
- **Reset to default**: `resetCannedFork` re-materializes the current registry
  IR over the fork and appends a new revision.

## Removing a canned report

Never reuse a retired `Key`. Existing forks are unaffected (they own their
IR); pending runs referencing the removed key fail at `PrepareRun` with the
structured error `canned report %q is not registered in this build`, and
"reset to default" on a surviving fork fails the same way.

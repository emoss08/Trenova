# TMS Metrics

This package exposes Prometheus metrics for HTTP traffic, GraphQL operations, errors, Temporal, audit buffering, and database concurrency behavior.

## Database Concurrency Metric

The concurrency hardening work emits:

- `trenova_db_concurrency_total{kind,entity,code}`

Label meanings:

- `kind`: logical concurrency outcome. Current values:
  - `version_mismatch`
  - `retryable_transaction`
- `entity`: entity name when available, otherwise `unknown`
- `code`: Postgres SQLSTATE when available, otherwise `unknown`

Examples:

- A stale optimistic-lock update on `Shipment` increments:
  - `trenova_db_concurrency_total{kind="version_mismatch",entity="shipment",code="unknown"}`
- A `FOR UPDATE NOWAIT` conflict increments:
  - `trenova_db_concurrency_total{kind="retryable_transaction",entity="unknown",code="55p03"}`

## Database Runtime Metrics

Registered by the tenant pool (`postgres.Connection.runtimeStats`) beside the
`trenova_db_pool_*` connection gauges:

- `trenova_db_cache_hits_total`, `trenova_db_cache_misses_total`,
  `trenova_db_cache_bypasses_total`, `trenova_db_cache_errors_total`: reads
  that asked for keel's result cache, by outcome; errors are shared-store,
  listener and decoder failures (each a miss or a bypass).
- `trenova_db_cache_bytes`: what the in-process LRU holds
  (`database.cache.localBytes` bounds it).
- `trenova_db_cache_versions`: (tenant, table) data versions held.
- `trenova_db_cache_listening`: 1 while the data version listener is
  connected; at 0 every cached read goes to the database.
- `trenova_db_pool_tenant_waits_total`: transactions and terminals that waited
  for one of their organization's `database.tenantConcurrency` slots.
- `trenova_db_pool_read_commit_failures_total`: connections closed because the
  reply to a read transaction's unawaited `COMMIT` failed (tenant and system
  pools).
- `trenova_db_slow_waits_total{component,pool,kind}`: waits past
  `database.slowWaitWarning` for an organization's slot (`tenant_slot`), a pool
  connection (`connection`) or another request's read of the same cache entry
  (`cache_fill`). Each also logs a warning on the `slow-wait` logger with the
  waiter's stack and, for a slot, every holder's.

A hit ratio (`hits / (hits + misses)`) that falls after a deploy names a read
that stopped matching its entries; `listening` at 0 for more than a few
seconds names a listener that cannot reach PostgreSQL (behind a
transaction-mode pooler, set `database.cache.listenHost`); a rising
`tenant_waits_total` names a tenant at its bound; any `slow_waits_total` names
work stuck behind a holder that is slow or never lets go, and its warning names
the holder.

## GraphQL Metrics

Every GraphQL request arrives on a single `POST /graphql` route, so the HTTP
metrics cannot tell one operation from another. These metrics restore
per-operation visibility and are emitted by the `TrenovaObservability`
gqlgen extension in `internal/api/graphql/observability.go`.

Always on:

- `trenova_graphql_operations_total{operation,type,status}`
- `trenova_graphql_operation_duration_seconds{operation,type}`
- `trenova_graphql_operation_errors_total{operation,type,code}`
- `trenova_graphql_operation_cost{operation,type}`
- `trenova_graphql_active_operations`
- `trenova_graphql_rejections_total{reason}`
- `trenova_graphql_parse_duration_seconds`
- `trenova_graphql_validation_duration_seconds`

Behind `monitoring.graphql.resolverMetrics` (off by default, because each
resolver-backed field adds a label pair):

- `trenova_graphql_resolver_calls_total{object,field}`
- `trenova_graphql_resolver_duration_seconds{object,field}`
- `trenova_graphql_resolver_errors_total{object,field}`

Label meanings:

- `operation`: the client operation name. Names that are empty become
  `anonymous`, names that are not valid GraphQL identifiers become `unknown`,
  and once `monitoring.graphql.maxTrackedOperations` distinct names have been
  seen every further name collapses to `other`. This cap is what stops a client
  from blowing up Prometheus cardinality with generated operation names.
- `type`: `query`, `mutation`, `subscription`, or `unknown`.
- `status`: `success` or `failure`.
- `code`: the `code` extension set by the error presenter, or `unknown`.
- `reason`: why a request was refused before it ever executed. Current values
  are `api_key` and `persisted_operation`.

### Slowest Operations

```promql
topk(10,
  histogram_quantile(0.95,
    sum by (operation, le) (
      rate(trenova_graphql_operation_duration_seconds_bucket[5m])
    )
  )
)
```

This is the query that replaces per-route REST latency. Use it to find which of
the several hundred queries and mutations is actually slow.

### Error Rate By Operation And Code

```promql
sum by (operation, code) (
  rate(trenova_graphql_operation_errors_total[5m])
)
```

### N+1 Detection

Requires `resolverMetrics: true`. Resolver calls per operation:

```promql
sum by (object, field) (rate(trenova_graphql_resolver_calls_total[5m]))
  /
scalar(sum(rate(trenova_graphql_operations_total[5m])))
```

A field whose ratio tracks the page size rather than staying near 1 is being
called once per row and wants a dataloader.

### Query Cost Headroom

`trenova_graphql_operation_cost` is the complexity gqlgen computed for the
operation, so you can watch headroom against the limit rather than waiting for
a rejection:

```promql
histogram_quantile(0.99,
  sum by (operation, le) (rate(trenova_graphql_operation_cost_bucket[1h]))
)
```

Costs are enforced in `internal/api/graphql/querycost`:

- A field returning a connection costs `clamp(first) * childCost`, where
  `clamp` is the same `pagination.ClampLimit` the resolvers use, so the cost
  tracks the rows that will actually be returned.
- Fields *inside* a connection (`edges`, `nodes`) fall through to the default
  cost, because the connection already charged for the page.
- Any other field returning a list has no client-controlled page size, so it
  costs `UnpaginatedListWeight * childCost`.
- Everything else is gqlgen's default, `1 + childCost`.

Rejections surface as `operation_errors_total` with code
`COMPLEXITY_LIMIT_EXCEEDED` or `OPERATION_DEPTH_LIMIT_EXCEEDED`, and are
returned to the client as HTTP 422.

Operations are also capped at `MaxOperationDepth`. Introspection meta-fields
(`__schema`, `__type`, `__typename`) are excluded from the depth measurement:
the standard introspection query is 13 levels deep, so counting it would block
schema tooling. Introspection is disabled outside development anyway.

The limits are calibrated against every persisted operation by
`TestPersistedDocumentBudget`, which measures each document at the maximum page
size and fails the build if one would be rejected. Raise the limit only after
that test tells you what the new worst case actually is.

### Dataloader Batching

Dataloaders (`internal/api/graphql/loaders`) run on `vikstrous/dataloadgen`
with a 1ms batching window and a batch capacity equal to the maximum page
size, so one page of rows resolves a relation in one query. When tracing is on
each batch emits `dataloadgen.wait` and fetch spans under the
`trenova.graphql.loaders` tracer, nested inside the operation span, which is
the quickest way to see whether a nested field is actually batching.

### Rejected Requests

```promql
sum by (reason) (rate(trenova_graphql_rejections_total[5m]))
```

A rise in `persisted_operation` normally means a client shipped with a document
the server does not know, so the client and server builds are out of step.

### Known Gap

Documents that fail to parse or validate never reach the extension, so they are
counted only under `rejections_total` when the persisted-operation layer refuses
them. In production and staging that layer rejects anything not in the manifest,
so unparseable documents cannot reach the executor at all.

## WebAssembly Module Metrics

Each WebAssembly module the TMS runs in wazero (`pkg/rules`, docs/engineering/rules-engine.md;
`pkg/x12`, docs/engineering/edi-x12.md) runs on a pool of instances, one call per instance at a
time, through `pkg/wasmmodule`. That package cannot import this one, so it reports through its
`wasmmodule.Observer` interface, which `WasmModule` implements; `NewWasmModule` builds one per
module under its own subsystem (`Rules` as `trenova_rules_*`, `EDIModule` as
`trenova_edi_module_*`), and `internal/bootstrap/infrastructure/wasmmodules.go` hands each to its
default runtime at start. The rules module's series are below; the EDI module's have the same
names under `trenova_edi_module_` (operations `x12.read`, `x12.acknowledge`, ...).

- `trenova_rules_calls_total{op,outcome}`: every call, by operation and how it ended
- `trenova_rules_call_duration_seconds{op}`: how long a call took, including the wait for a
  free instance
- `trenova_rules_instances_in_use`: instances running a call right now
- `trenova_rules_instances_discarded_total{reason}`: instances thrown away instead of reused

Label meanings:

- `op`: the operation (`formula.check`, `allocation.split`, ...). A name the module does not
  know, or one not shaped like `area.name`, is `unknown`, so a caller bug cannot grow the label
  set.
- `outcome`: `ok`; `request_error` (the module refused the request: a caller bug);
  `trap` (the module panicked: an internal bug, never retried); `context_done` (the caller's
  context ended while it waited or ran); `unavailable` (the module failed to compile or
  instantiate).
- `reason`: `trap`; `context_done` (the deadline closed the instance mid-call); `memory` (its
  linear memory grew past the recycle threshold, 64 MiB by default).

Any `trap` is a bug to fix. `instances_in_use` pinned at the pool size (`GOMAXPROCS`) while
call latency climbs means callers are queueing for instances; a steady `memory` discard rate
means some operation's input is large enough that it should be split or streamed.

```promql
sum by (op, outcome) (rate(trenova_rules_calls_total{outcome!="ok"}[5m]))
```

```promql
histogram_quantile(0.99,
  sum by (op, le) (rate(trenova_rules_call_duration_seconds_bucket[5m]))
)
```

## Suggested Grafana Panels

Use rate-based panels for operational visibility and raw increase panels for incident review.

### Version Mismatch Rate By Entity

```promql
sum by (entity) (
  rate(trenova_db_concurrency_total{kind="version_mismatch"}[5m])
)
```

Use this to spot entities with frequent stale writes. A sustained rise usually means user workflows are editing the same records concurrently.

### Retryable Transaction Rate By Postgres Code

```promql
sum by (code) (
  rate(trenova_db_concurrency_total{kind="retryable_transaction"}[5m])
)
```

Recommended interpretation:

- `55p03`: lock not available / `NOWAIT` contention
- `40p01`: deadlock detected
- `40001`: serialization failure

### Top Concurrency Outcomes

```promql
topk(10,
  sum by (kind, entity, code) (
    increase(trenova_db_concurrency_total[1h])
  )
)
```

Use this during incident review to see which entities and SQLSTATEs are driving concurrency pressure.

## Suggested Alerts

Tune thresholds to your actual traffic. Start conservative and tighten after a week of data.

### Deadlock Alert

```promql
sum(increase(trenova_db_concurrency_total{kind="retryable_transaction",code="40p01"}[10m])) > 0
```

This should normally be zero. Any sustained non-zero value is worth investigating.

### Lock Contention Surge

```promql
sum(rate(trenova_db_concurrency_total{kind="retryable_transaction",code="55p03"}[5m])) > 0.2
```

This means the app is failing fast on lock acquisition more than once every 5 seconds across the fleet.

### Version Mismatch Surge

```promql
sum by (entity) (
  rate(trenova_db_concurrency_total{kind="version_mismatch"}[15m])
) > 0.5
```

This is useful when a UI workflow or batch job starts creating excessive stale writes.

## Operational Notes

- `version_mismatch` usually indicates user or worker concurrency, not database distress.
- `55p03` is expected in small amounts when using fail-fast locking. Alert on sustained increases, not isolated events.
- `40p01` is the highest-signal concurrency error and usually points to lock ordering or unexpectedly long transactions.
- If retryable transaction metrics rise after deployment, inspect:
  - long-running transactions
  - new multi-row update flows
  - missing indexes on lock/filter predicates

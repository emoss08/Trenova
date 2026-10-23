# Building for scale

Design every change as if it already serves hundreds of thousands of users across thousands of
organizations, some with millions of shipments. Go through this list for any new read, write,
job or integration, and fix what fails before handing off. When you meet existing code that
fails it, fix it in the same change if it is small, or name it in your report.

## Request paths

- **Bounded.** No request does work that grows with the size of a tenant's data. Lists are
  keyset-paginated connections; counts run only when asked for; exports, reports, recalculations
  and bulk changes start a workflow and return.
- **Batched.** No query per row: dataloaders over `...ByIDs` in GraphQL, `IN (…)` and joins in
  repositories, one statement for a bulk update. Loops that call a repository are a defect.
- **Indexed.** Every `WHERE`, `ORDER BY` and join a repository runs has an index that serves it,
  tenant columns leading, partial when the query is. Write the migration with the query.
- **Projected.** Select the columns asked for, not `*`, on anything a table or connection reads.
- **History in one pass.** A balance or state "as of" a series of dates is never a correlated
  subquery per row per date. Read the changes once, fold them into segments with a window
  function, expand each segment onto the dates it covers, and read only rows open now or
  touched inside the window (`accountsreceivablerepository/openasof.go`). Every source that
  moves the stored value must be an event, or the series disagrees with today's figure.
- **No third party in the request.** Calls to PC*MILER, FMCSA, model providers, accounting
  systems, email or the imaging worker run in activities, or behind a cache, unless the person is
  waiting for exactly that answer; then they carry a timeout and a fallback.
- **Short transactions.** No network call inside a transaction; lock rows in a fixed order;
  set a lock timeout when a write may wait.

## State and coordination

- **Store the answer, don't recompute it per read.** When a value is read far more than its
  inputs change (an ETA, a board row, analytics, a rate), compute it on change into a table or
  cache and read the stored row (`stop_etas`, `shipmentboardcache`, `ai_work_units`).
- **Invalidate by epoch** ([data-access.md](data-access.md#caching)).
- **Debounce per entity.** Bursts of events for one record collapse into one workflow
  ([jobs.md](jobs.md)).
- **Idempotent everywhere.** HTTP writes take `Idempotency-Key`; activities and webhook
  receivers tolerate replay through unique keys and conditional updates.
- **Fair across tenants.** Fan-out passes a tenant fairness key; per-tenant work is batched so one
  large organization cannot starve the rest; public endpoints are rate limited.
- **Realtime is invalidation, not payload.** Publish that a record changed
  (`PublishResourceInvalidation`) and let clients refetch through the cache; never broadcast
  records.
- **No in-process state that assumes one replica.** API and worker processes are stateless and
  horizontally scaled; shared state lives in Postgres, Redis or Temporal. An in-process cache
  (`shared/lrucache`, `ttlcache`) is only an accelerator with a bound and an expiry.

## Data growth

- Append-only tables (events, audit, usage, tracking) have a retention or rollup plan and an
  index on what the sweep deletes or folds by.
- Large text and files live in object storage through `storedobject`, never in a row.
- Anything that grows without bound per tenant (history, notifications, messages) is paginated
  everywhere it is read.

## Observability

- A new hot path or job gets a Prometheus metric in
  `internal/infrastructure/observability/metrics/` (a typed struct per domain) and a span through
  `observability.RunWithSpanReturn` where latency matters.
- Logs are zap, named per component (`p.Logger.Named("service.commodity")`), structured fields,
  no document content or personal data, and never log-and-return.

## Cost and limits

- An expensive GraphQL field declares its cost; never raise the global ceiling to fit one query.
- Model calls go through the completion router
  (`internal/infrastructure/agentcompletion/completionrouter`), which meters them; a new feature
  never calls a provider SDK on its own.
- Measure before and after when a change targets performance (`EXPLAIN (ANALYZE, BUFFERS)` on
  the query, a benchmark for a hot function) and put the numbers in the PR.

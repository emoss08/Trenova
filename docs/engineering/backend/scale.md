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
  tenant columns leading, partial when the query is. Write the migration with the query, and give
  the query an integration test so the plan guard explains it: CI fails on a sequential scan of a
  large table or a page sorted without an index
  ([data-access.md](data-access.md#plan-guard)).
- **Within budget.** No request sends one statement text in more round trips than
  `database.budget.maxRepeats` (10): every REST request and GraphQL operation is budgeted and logs
  the repeated SQL and its call sites when one does ([data-access.md](data-access.md#budgets)).
  Treat that warning as a defect, and pin a hot path with a hard `keel.Budget` in its test.
- **Projected.** Select the columns asked for, not `*`, on anything a table or connection reads.
- **Each policied table once.** Under row-level security every reference to a table brings its
  policy's checks: an InitPlan that verifies the signed tenant scope (about 30 µs, once per
  statement) and, for the heavier policies (invoices, with their owner-scope checks), subplans
  over other policied tables. Two `EXISTS` against the same table pay twice. Read what several
  predicates need from a policied table once, in a CTE limited to the rows in play, and join it:
  detention intelligence's recovery read takes 36 ms this way against 69 ms with three `EXISTS`
  (`detentionrepository/intelligence.go`, `invoiced`). Measure under the application role, not
  as the superuser, who skips the policies
  ([row-level-security.md](../row-level-security.md#what-the-scope-costs)).
- **A per-row fact from a small set in one pass.** A fact each row of a large list needs from a
  small live set (a driver's open work among the tenant's open moves) is a grouped or
  `DISTINCT ON` CTE over that set, joined to the list, never a correlated subquery or lateral
  that walks each row's whole history to keep the few live rows. The dispatch driver read's
  open-work count and final drop (`dispatchconsolerepository`, `openWork`, `finalDrops`): the
  near-ordered read takes 69 ms this way against 548 ms as per-driver probes through about 26
  historical assignments each, under the application role, and the plain board read 10-23 ms
  against 100-130 ms.
- **History in one pass.** A balance or state "as of" a series of dates is never a correlated
  subquery per row per date. Read the changes once, fold them into segments with a window
  function, expand each segment onto the dates it covers, and read only rows open now or
  touched inside the window (`accountsreceivablerepository/openasof.go`). Walk back from the
  stored value rather than summing every change ever made: read only the changes after the
  window start, by an index on (tenant, creation time), and take them off today's figure, so
  the read grows with the window, not with the history. Every source that moves the stored
  value must be an event, or the series disagrees with today's figure.
- **No third party in the request.** Calls to PC*MILER, FMCSA, model providers, accounting
  systems, email or the imaging worker run in activities, or behind a cache, unless the person is
  waiting for exactly that answer; then they carry a timeout and a fallback.
- **Short transactions.** No network call inside a transaction; lock rows in a fixed order;
  set a lock timeout when a write may wait.

## State and coordination

- **Store the answer, don't recompute it per read.** When a value is read far more than its
  inputs change (an ETA, a board row, analytics, a rate), compute it on change into a table or
  cache and read the stored row (`stop_etas`, `shipmentboardcache`, `ai_work_units`).
- **Invalidate by epoch** ([data-access.md](data-access.md#caching)); for a repository read of a
  rarely written table, keel's result cache invalidates by data version on its own
  (`Query.Cache`, [data-access.md](data-access.md#result-cache)), never on a user-owned table.
- **Fair across a pool.** `database.tenantConcurrency` bounds one organization's concurrent
  transactions; code that fans out in goroutines under one request shares its tenant's slots, so
  batch instead of fanning out.
- **Debounce per entity.** Bursts of events for one record collapse into one workflow
  ([jobs.md](jobs.md)).
- **Idempotent everywhere.** HTTP writes take `Idempotency-Key`; activities and webhook
  receivers tolerate replay through unique keys and conditional updates.
- **Throttle a per-record side effect with a conditional claim.** One `UPDATE … WHERE last_at IS
  NULL OR last_at <= now - cooldown RETURNING` stamps the record and claims the right to act; no
  row back is a `RateLimitError` with `RetryAfter`. Never read-then-write, never an in-process
  limiter (replicas do not share it). Release the claim if the side effect fails
  (`ClaimTestSend` in `inboundmessagerepository/mailbox.go`).
- **Fair across tenants.** Fan-out passes a tenant fairness key; per-tenant work is batched so one
  large organization cannot starve the rest; public endpoints are rate limited.
- **Hear a record change off the stream, never by polling.** Work that must wake when a
  record changes registers the record with `realtimeports.RecordWatchFeed` and follows it
  from a worker: every write already publishes an invalidation, and the feed reads only each
  entry's header until a tenant is known to watch something, so the work beyond that grows
  with the writes of tenants that watch, not with every write ([agent-waits.md](../agent-waits.md#hearing-a-record-change)).
- **Realtime is invalidation, not payload.** Publish that a record changed
  (`PublishResourceInvalidation`) and let clients refetch through the cache; never broadcast
  records.
- **No in-process state that assumes one replica.** API and worker processes are stateless and
  horizontally scaled; shared state lives in Postgres, Redis or Temporal. An in-process cache
  (`shared/lrucache`, `ttlcache`) is only an accelerator with a bound and an expiry, for what
  Postgres does not hold; a database read is cached with keel's result cache, which every
  process's writes invalidate.

## Data growth

- A read that adds up a tenant's history reads a rollup kept by day and month, not the rows
  ([data-access.md](data-access.md#rollups)).
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
  never calls a provider SDK on its own. Model work nobody waits on goes through
  `llmports.BatchCompletionService` at the provider's batch discount, falling back to one
  call at a time on `ErrBatchUnsupported`.
- Measure before and after when a change targets performance (`EXPLAIN (ANALYZE, BUFFERS)` on
  the query, a benchmark for a hot function) and put the numbers in the PR. Explain a tenant
  query as the application sees it: in one transaction, set a signed `trenova.scope`, `SET LOCAL
  ROLE trenova_tenant` and `SET LOCAL jit = off` (as the pools do), and keep costs on, since
  `COSTS OFF` hides the JIT section. A superuser plan skips every policy and is not the plan
  production runs ([row-level-security.md](../row-level-security.md#what-the-scope-costs)).

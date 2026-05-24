# PostgreSQL Timeout Incident: Nested pgxpool Under PgBouncer

## Summary

On 2026-05-24, the production TMS API showed sporadic 502/504 failures across
unrelated endpoints. Shipment list requests, location lookups, shipment events,
runtime integration config, document lookups, and billing-readiness requests
could all fail during the same bursts.

The failure was not caused by one slow endpoint. Small endpoints such as
`/api/v1/integrations/OpenWeatherMap/runtime-config/` were timing out only
because they shared the same database access path as heavier shipment queries.

The fix was to remove the nested `pgxpool` layer under `database/sql` and use
Bun's PostgreSQL driver directly:

```go
sqldb := sql.OpenDB(pgdriver.NewConnector(
    pgdriver.WithDSN(dsn),
    pgdriver.WithConnParams(c.databaseConnParams()),
))

db := bun.NewDB(sqldb, pgdialect.New())
```

This shipped in `v0.0.18` with commit `aa4a0da4c`.

## Symptoms

Production logs initially showed request bursts failing exactly at the app
deadline:

```json
{
  "msg": "http: Handler timeout",
  "status": 504,
  "method": "GET",
  "path": "/api/v1/shipments/",
  "latency": "1m0.000..."
}
```

After tightening the request deadline to 55 seconds, failures moved to exactly
55 seconds:

```json
{
  "msg": "Request timed out",
  "status": 504,
  "error": "request exceeded app deadline of 55s"
}
```

The failures were sporadic. The same query shape could be fast immediately
before or after a timeout:

- `/api/v1/shipments/?limit=20&...status=Delayed...` could finish in 10-25 ms.
- A near-identical request could later run until the 55 second context deadline.
- `/api/v1/integrations/OpenWeatherMap/runtime-config/` could complete in 1-2 ms,
  but also time out during a burst.

This made the issue look like a global app hang, a goroutine leak, or a database
lock issue.

## What We Instrumented

The first useful change was to stop relying only on request logs and add database
slow-query diagnostics. The `v0.0.17` release added:

- request timeout shorter than write timeout: `55s` vs `75s`
- app-controlled timeout responses instead of upstream 502 ambiguity
- slow database query logging from Bun query hooks
- request logs with `database/sql` pool stats
- pprof server wiring, disabled unless explicitly configured

The pprof server logs its status on startup. In production it was disabled:

```text
pprof server disabled
```

When enabled, it listens on the configured host and port. The default endpoint is
host-local:

```bash
curl -fsS 'http://127.0.0.1:6060/debug/pprof/goroutine?debug=2' \
  >/tmp/trenova.goroutines.txt
```

## Evidence

The decisive logs were slow database query logs. They showed the app was not
spinning in Go code. It was blocked in database query execution until the request
context expired:

```json
{
  "logger": "slow-query",
  "msg": "slow database query",
  "duration": "54.993...",
  "operation": "SELECT",
  "query_hash": "e04ff32199279e4c",
  "query": "SELECT count(*) FROM \"shipments\" AS \"sp\" WHERE ...",
  "error": "context deadline exceeded",
  "context_error": "context deadline exceeded"
}
```

The same timeout burst also showed `database/sql` pool stats that did not look
saturated:

```json
{
  "db_pool_open": 4,
  "db_pool_in_use": 0,
  "db_pool_idle": 4,
  "db_pool_wait_count": 0,
  "db_pool_wait_duration": "0s",
  "db_pool_max_open": 75
}
```

That was the contradiction that mattered:

- requests were waiting on database work for 55 seconds
- `database/sql` reported no wait and spare idle connections

Those two facts cannot both describe the whole database access path. Something
below `database/sql` was controlling access or blocking work without being
visible in `database/sql` stats.

## Root Cause

The TMS API was using two client-side pools in front of PgBouncer:

```text
Gin request
  -> Bun
  -> database/sql pool
  -> pgx stdlib adapter
  -> pgxpool
  -> PgBouncer
  -> PostgreSQL
```

The code created a `pgxpool.Pool` and then wrapped it with
`stdlib.OpenDBFromPool(pool)`. After that, the application configured the outer
`database/sql` pool:

```go
sqldb.SetMaxOpenConns(c.cfg.Database.MaxOpenConns)
sqldb.SetMaxIdleConns(c.cfg.Database.MaxIdleConns)
```

That configuration did not make `pgxpool` disappear. The inner `pgxpool` still
had its own capacity, lifecycle, acquisition behavior, and connection state. The
request logs and metrics only showed the outer `database/sql` pool, so the
operational picture was misleading.

In practice, the app had this behavior:

- `database/sql` could report idle connections and zero wait.
- Work could still be blocked or constrained below it by `pgxpool` or PgBouncer.
- Slow logs attached to Bun queries reported the final context cancellation,
  not the hidden pool's internal wait state.
- The app's configured statement timeout was not the first timeout to fire; the
  request deadline was.

This is why unrelated endpoints could time out together. They were not all slow.
They were sharing a hidden database access bottleneck.

## Why PgBouncer Made This More Fragile

PgBouncer is already a server-side pool. It should be paired with one clear
client-side pool in the application.

With PgBouncer in the path, stacking `database/sql` and `pgxpool` added
complexity without useful isolation:

```text
database/sql pool -> pgxpool -> PgBouncer pool
```

That shape makes it harder to reason about:

- where a request is waiting
- which pool's max connection setting is effective
- whether a session-level setting is still present
- which metrics represent real saturation

PgBouncer transaction pooling also means session-level settings should not be
treated as the only timeout guard. Role-level PostgreSQL settings remain the
most reliable backstop:

```sql
ALTER ROLE trenova SET statement_timeout = '10s';
ALTER ROLE trenova SET lock_timeout = '5s';
ALTER ROLE trenova SET idle_in_transaction_session_timeout = '30s';
```

## Fix

The fix was to use Bun's native PostgreSQL driver and keep a single visible
application-side pool:

```text
Gin request
  -> Bun
  -> database/sql pool
  -> pgdriver
  -> PgBouncer
  -> PostgreSQL
```

The production connection code now uses `pgdriver.NewConnector`:

```go
sqldb := sql.OpenDB(pgdriver.NewConnector(
    pgdriver.WithDSN(dsn),
    pgdriver.WithConnParams(c.databaseConnParams()),
))
```

The existing `database/sql` settings now govern the only client-side pool:

```go
sqldb.SetMaxOpenConns(c.cfg.Database.MaxOpenConns)
sqldb.SetMaxIdleConns(c.cfg.Database.MaxIdleConns)
sqldb.SetConnMaxLifetime(c.cfg.Database.ConnMaxLifetime)
sqldb.SetConnMaxIdleTime(c.cfg.Database.ConnMaxIdleTime)
```

The connection parameters are still applied on connection creation:

```go
map[string]any{
    "statement_timeout":                   "10000ms",
    "lock_timeout":                        "5000ms",
    "idle_in_transaction_session_timeout": "30000ms",
}
```

The DSN was also changed from pgx-style `connect_timeout=10` to Bun pgdriver's
documented `dial_timeout=10s`.

Reference: Bun's PostgreSQL driver documentation shows `pgdriver.NewConnector`
with `sql.OpenDB`, documents `dial_timeout`, and supports `WithConnParams` for
connection-created PostgreSQL parameters:
<https://bun.uptrace.dev/postgres/#pgdriver>.

## Why This Fixed It

After the change, there is no hidden pgxpool between `database/sql` and
PgBouncer. That means:

- `database/sql` metrics describe the real app-side pool.
- app pool settings are actually the pool settings.
- request logs are no longer blind to a second pool.
- PgBouncer remains the server-side pooling layer.
- query timeout parameters are applied by the driver used by the app.

The production symptom disappeared after deploying `v0.0.18`, which confirms the
nested pool architecture was the cause.

## What Was Not The Root Cause

The following were investigated and did not explain the incident:

- HTTP server write timeout: it needed correction, but was not the root cause.
- Cloudflare: it reported failures but was not causing them.
- One specific runtime config endpoint: it was a victim of shared DB pressure.
- Go goroutine leak: logs pointed to database query execution instead.
- `database/sql` pool saturation: visible stats showed no wait and idle
  connections.
- Missing shipment status indexes: indexes may still need normal performance
  review, but they did not explain fast and slow executions of the same query
  shape around the same time.

## Follow-Up Checks

After deploying a fix for this class of issue, check:

```bash
curl -fsS https://api.trenova.app/api/v1/system/version
docker compose -f docker-compose.api.yml logs --tail=120 tms-api
```

Expected:

- startup logs show the intended release version
- `PostgreSQL connection established`
- normal request latencies stay in milliseconds for small endpoints
- slow query logs do not appear at the exact app request deadline

If slow query logs still appear, check whether PostgreSQL itself is blocked:

```sql
SELECT
    blocked.pid AS blocked_pid,
    blocker.pid AS blocking_pid,
    blocked.wait_event_type,
    blocked.wait_event,
    now() - blocked.xact_start AS blocked_xact_age,
    now() - blocker.xact_start AS blocker_xact_age,
    left(blocked.query, 500) AS blocked_query,
    left(blocker.query, 500) AS blocking_query
FROM pg_stat_activity blocked
JOIN pg_locks blocked_locks
    ON blocked_locks.pid = blocked.pid
JOIN pg_locks blocker_locks
    ON blocker_locks.locktype = blocked_locks.locktype
    AND blocker_locks.database IS NOT DISTINCT FROM blocked_locks.database
    AND blocker_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
    AND blocker_locks.page IS NOT DISTINCT FROM blocked_locks.page
    AND blocker_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
    AND blocker_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
    AND blocker_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
    AND blocker_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
    AND blocker_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
    AND blocker_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
    AND blocker_locks.pid <> blocked_locks.pid
JOIN pg_stat_activity blocker
    ON blocker.pid = blocker_locks.pid
WHERE NOT blocked_locks.granted
ORDER BY blocked_xact_age DESC;
```

Check PgBouncer separately:

```sql
SHOW POOLS;
SHOW STATS;
SHOW CLIENTS;
SHOW SERVERS;
```

Focus on `cl_waiting`, `avg_wait_time`, `sv_active`, and server connection
exhaustion.

## Guardrails

- Do not put `pgxpool` behind `database/sql`.
- Do not stack client-side pools in front of PgBouncer.
- Keep request timeout shorter than write timeout.
- Log slow database queries with query hash, operation, context error, and pool
  stats.
- Treat `database/sql` stats as authoritative only when `database/sql` owns the
  client-side pool.
- Keep PostgreSQL role-level timeout defaults in production even when the app
  also sets driver connection parameters.
- When small endpoints time out during heavy request bursts, inspect shared
  dependencies before optimizing the small endpoint.

## Code References

- `services/tms/internal/infrastructure/postgres/connection.go`
- `services/tms/internal/infrastructure/postgres/slow_query_hook.go`
- `services/tms/internal/infrastructure/config/config.go`
- `services/tms/config/config.prod.yaml`
- `docs/operations-guides/high-concurrency-control-lookups-runbook.md`

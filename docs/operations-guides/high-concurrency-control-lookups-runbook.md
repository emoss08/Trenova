# High-Concurrency Control Lookup Runbook

Use this when API latency rises, worker creation or shipment validation slows down, or Cloudflare reports 524s.

For the 2026-05-24 production timeout incident caused by a nested `pgxpool`
under `database/sql` and PgBouncer, see
`docs/operations-guides/postgres-pgxpool-timeout-incident.md`.

## Fast Checks

Verify the API and monitoring listener:

```bash
curl -fsS http://127.0.0.1:8080/api/v1/system/version
curl -fsS http://127.0.0.1:9090/internal/metricsz >/tmp/trenova.metrics
```

If pprof is explicitly enabled, capture goroutines from the host only:

```bash
curl -fsS 'http://127.0.0.1:6060/debug/pprof/goroutine?debug=2' >/tmp/trenova.goroutines.txt
```

Verify PostgreSQL role-level timeout defaults for the application database user. These role
settings are the most reliable guard when PgBouncer transaction pooling is in front of
PostgreSQL:

```sql
SELECT rolname, rolconfig
FROM pg_roles
WHERE rolname = current_user;

ALTER ROLE app_database_user SET statement_timeout = '10s';
ALTER ROLE app_database_user SET lock_timeout = '5s';
ALTER ROLE app_database_user SET idle_in_transaction_session_timeout = '30s';
```

Check for blocked PostgreSQL sessions:

```sql
SELECT
    blocked.pid AS blocked_pid,
    blocker.pid AS blocking_pid,
    blocked.state AS blocked_state,
    blocker.state AS blocking_state,
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

Check PgBouncer pressure:

```sql
SHOW POOLS;
SHOW STATS;
SHOW CLIENTS;
SHOW SERVERS;
```

Focus on high `cl_waiting`, long `avg_wait_time`, saturated `sv_active`, and clients waiting on the application database.

Use a dedicated PgBouncer admin or stats identity for these commands. The app database user should not be listed in `admin_users` or `stats_users`.

```ini
admin_users = pgbouncer_admin
stats_users = pgbouncer_admin, pgbouncer_stats
```

See `config/pgbouncer.ini.example` and `config/pgbouncer.userlist.example.txt` for the deployment template. This observability setup does not require changing `pool_mode`, pool sizing, or application database connection settings.

## Control Tables

These controls are tenant-scoped and must have one row per `(organization_id, business_unit_id)`:

- `dispatch_controls`
- `data_entry_controls`
- `document_controls`
- `shipment_controls`

Check for bad duplicate data before or after migrations:

```sql
SELECT 'dispatch_controls' AS table_name, organization_id, business_unit_id, count(*)
FROM dispatch_controls
GROUP BY organization_id, business_unit_id
HAVING count(*) > 1
UNION ALL
SELECT 'data_entry_controls', organization_id, business_unit_id, count(*)
FROM data_entry_controls
GROUP BY organization_id, business_unit_id
HAVING count(*) > 1
UNION ALL
SELECT 'document_controls', organization_id, business_unit_id, count(*)
FROM document_controls
GROUP BY organization_id, business_unit_id
HAVING count(*) > 1
UNION ALL
SELECT 'shipment_controls', organization_id, business_unit_id, count(*)
FROM shipment_controls
GROUP BY organization_id, business_unit_id
HAVING count(*) > 1;
```

These controls are organization-scoped by repository access pattern:

- `accounting_controls`
- `billing_controls`
- `invoice_adjustment_controls`

## Application Logs

Search recent API logs by request ID and slow request message:

```bash
journalctl -u trenova-api --since "30 minutes ago" | grep -E "Slow request detected|retryable_transaction|request_id"
```

For lock and timeout errors, keep the request ID and compare the timestamp with `pg_stat_activity`, PgBouncer `SHOW POOLS`, and the `trenova_db_concurrency_total` metrics.

# TMS Metrics

This package exposes Prometheus metrics for HTTP traffic, errors, Temporal, audit buffering, and database concurrency behavior.

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

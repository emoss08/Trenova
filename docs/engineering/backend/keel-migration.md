# Persistence during the keel migration

Trenova is moving its data layer from Bun to [keel](https://github.com/emoss08/keel). Modules
convert one repository at a time, so until the last one lands both layers run in one process,
against the same pools, under the same row-level security and inside the same transactions.
This page is what that foundation guarantees and what converted code must do. Read
[data-access.md](data-access.md) and [row-level-security.md](../row-level-security.md) first.
Paths are under `services/tms/`.

## One pool per role, two layers on it

| Role | Login | Pool |
|---|---|---|
| tenant | `database.user` | `Connection` (and `ReportingConnection`, its own pool) |
| system | `database.system.user` | the same `Connection`, for contexts declared with `dbscope.WithSystem` |

`dbpool.Open` (`internal/infrastructure/postgres/dbpool`) builds each pool once: keel opens the
`pgxpool.Pool` with Trenova's `keel.Config`, and Bun runs on that same pool through pgx's
`database/sql` driver (`stdlib.GetPoolConnector`) wrapped in the scoped driver. `database/sql`
keeps no idle connections of its own, so the pgx pool is the only pool and keel never waits on
a connection Bun parked.

| Concern | Where it lives |
|---|---|
| Session timeouts (`statement_timeout`, `lock_timeout`, `idle_in_transaction_session_timeout`) | pgx runtime parameters, from `database.*Timeout` (`connection.go`, `sessionParams`) |
| `application_name`, `connect_timeout`, `sslmode` | the DSN (`config.Config.GetDSN`) |
| Pool size and lifetimes | `maxOpenConns`, `connMaxLifetime`, `connMaxIdleTime` (and `database.system.maxOpenConns`) |
| Enum types | keel registers every enum and its array type on each new connection with one catalog query |
| Slow queries, tracing, query timing | one `slowQueryLog` per pool for both layers; `bunotel` for Bun, `tracingObserver` (same `go.sql.query_timing` histogram) for keel |
| Pool metrics | `trenova_db_pool_*` from `pgxpool.Stat` (`metrics.Database.RegisterPoolStats`) |

Bun statements without arguments (all of them: Bun formats its own SQL) go over the simple
protocol, as `pgdriver` sent them, so they never fill the per-connection prepared-statement cache
that keel's statements use, and a multi-statement string still runs.

## Getting the keel DB

Repositories keep `Params{DB *postgres.Connection}` and take the keel DB for the context from
it, the way Bun code takes `DBForContext(ctx)`:

```go
func (r *repository) GetByID(ctx context.Context, req commodityports.GetRequest) (*commodity.Commodity, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*commodity.Commodity, error) {
		row, err := commoditydb.Commodities.Get(ctx, r.db.Keel(ctx), req.ID)
		return row, commodityErrors.Map(err, req.ID.String())
	})
}
```

`Keel(ctx)` returns the system pool's DB for a system scope and the tenant pool's DB otherwise.
`rlslint` holds `r.db.Keel(ctx)` to the same rule as `DBForContext`: an exported method that
touches it outside `dbtx` fails.

## Transactions

`dbtx.Read`/`Write` (and `Connection.WithTx`, `RunScoped`, `RunDetached`) open one transaction
that both layers use:

1. a dedicated `database/sql` connection, and through it the `*pgx.Conn` under the driver;
2. Bun begins on it, and the scoped driver sends `BEGIN; SET LOCAL trenova.scope …` (and the
   read scope);
3. `keel.JoinTx` puts a keel transaction on the callback's context, on that connection;
4. after Bun commits or rolls back, keel is told how it ended, so its `Committed` hooks,
   `keel.AfterCommit` callbacks and data versions follow the real outcome.

Nesting is unchanged: a call with the same scope joins, a call with another tenant's scope
opens a second transaction on another connection (and a second keel transaction shadows the
first), and `RunDetached` always opens its own. `dbtx.Savepoint` uses `keel.Savepoint` inside a
shared transaction. `kernelports.AfterCommit` and `keel.AfterCommit` both run after the shared
commit; prefer `kernelports.AfterCommit` in services, which do not know which layer wrote.

`BEGIN` is detached from the context's cancellation: `database/sql` would otherwise roll back
from its own goroutine while a keel statement is running on the same pgx connection. A cancelled
context still fails the next statement, and `WithTx` checks the context before `COMMIT`.

Outside `dbtx`, keel runs every terminal in an implicit transaction whose `Config.OnBegin` sets
the same signed scope, from the same `scopeGuard` the driver uses, so `keel.Write`/`keel.Read`
called directly are row-level-security scoped too. But a `keel.Write` started directly is
keel's alone: a Bun repository called inside it opens its own transaction. **Until Bun is gone,
start transactions with `dbtx` or `WithTx`, never `keel.Write`/`keel.Read`.**

## Errors

keel returns Postgres errors as `*pgconn.PgError`, which is also what Bun now returns, so every
`pkg/dberror` predicate answers both. keel's own errors map like Bun's did:

| keel | Helper | Caller sees |
|---|---|---|
| `keel.ErrNotFound` | `HandleNotFoundError(err, "Commodity")` / `IsNotFoundError` | `NotFoundError` "Commodity not found within your organization" |
| `keel.ErrConflict` (`*keel.ConflictError` from `Save`) | `HandleVersionConflict(err, entity, id)` / `IsVersionConflict` | the `version` validation error `CheckRowsAffected` gives |
| unique violation (`keel.IsUnique`) | `Mapping.Unique` / `MapUniqueViolation` | validation error on the mapped field, `DUPLICATE` |
| foreign key violation (`keel.IsForeignKey`) | `Mapping.ForeignKey` / `MapForeignKeyViolation` | validation error on the mapped field, `INVALID_REFERENCE` |
| `keel.ErrReadOnly` | `IsReadOnlyTransaction` | as a Postgres read-only refusal |
| `keel.ErrTenantMismatch` | `IsTenantBoundaryViolation` | authorization error and a tenant-boundary report at the edge, like a row-level security refusal |
| `ErrNoScope`, `ErrAllTenantsRequired`, `ErrSystemScopeRequired` | none | a programming error: an internal error |

A repository declares its mapping once and calls it on every keel error:

```go
var commodityErrors = dberror.Mapping{
	Entity: "Commodity",
	Unique: []dberror.Constraint{
		{Name: "idx_commodities_name", Field: "name", Message: "Commodity with this name already exists"},
	},
}
```

An unmapped constraint passes through, so `NormalizeDatabaseError` still turns it into the
generic duplicate conflict at the edge.

## What converted repository code does

- Builds on generated keel handles; its method bodies run in `dbtx.Read`/`Write` and pass
  `r.db.Keel(ctx)` to every terminal.
- Writes versioned rows with `Save`, which checks the version, and maps errors through a
  `dberror.Mapping`.
- Declares `dbscope.WithSystem` before `dbtx` for cross-tenant work, and says `AllTenants()` on
  each such query (keel refuses a system-scope query without it).
- Tests against the real pools: the integration tests in `internal/infrastructure/postgres`
  (`keel_integration_test.go`) and the test model in `internal/testutil/keeltest` show the
  pattern, a second tenant included. `go generate -tags integration ./internal/testutil/keeltest/`
  regenerates that model's handles; its files build only with the `integration` tag, so the
  application's `keel gen` run never sees it.

## When Bun is gone

The Bun half is deleted, not adapted: `dbpool`'s `database/sql` driver adapter and Bun's pool,
`Connection.DB`/`DBForContext`, the `bun.Tx` argument of `WithTx` and `kernelports.DBConnection`'s
Bun methods. `dbtx.Read`/`Write` then call `keel.ReadValue`/`WriteValue` on `Keel(ctx)`, keeping
two things keel does not do by itself: a context with another scope, or a detached one, must get a
new transaction rather than join the one on the context. Until keel can start a transaction that
shadows the context's, `dbtx` does that with `keel.JoinTx` on a connection it begins itself.
`kernelports.AfterCommit` becomes `keel.AfterCommit`.

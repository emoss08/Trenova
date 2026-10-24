# Data access

Entities, repositories, migrations, locking and caching. The reference vertical is commodity:
`internal/modules/reference/{domain/commodity,adapters/postgres/commodityrepository}`. Paths are
under `services/tms/`. Read [row-level-security.md](../row-level-security.md) and
[bun/buncolgen.md](../bun/buncolgen.md) before writing a repository.

## Entities

A domain entity is a Bun model in `domain/<entity>/` (`domain/commodity/commodity.go`):

- Asserts the interfaces it implements: `bun.BeforeAppendModelHook`,
  `validationframework.TenantedEntity`, and `domaintypes.PostgresSearchable` when it is searchable.
- `pulid.ID` with a per-entity prefix (`pulid.MustNew("com_")`), primary key
  `(id, organization_id, business_unit_id)`.
- `Version int64` for optimistic locking; `CreatedAt`/`UpdatedAt int64` Unix seconds, set in
  `BeforeAppendModel` with `timeutils.NowUnix()`.
- Typed string enums with `String()` and `IsValid()`, backed by a `CHECK` constraint.
- Entity-level rules in `Validate(multiErr)` (ozzo); rules that need the database belong to the
  service's validator ([services.md](services.md#validation)).

## Repositories

Shape: an fx `Params{DB *postgres.Connection; Logger}`, an unexported `repository`, and
`New(p) <module>ports.XRepository`.

- **Every method body runs in `dbtx.Read` or `dbtx.Write`** (or `ReadErr`/`WriteErr`,
  `Read2`/`Write2`, `Savepoint`) and queries through `r.db.DBForContext(ctx)`, which joins the
  transaction already on the context. Use `Read` for reads; a read in `dbtx.Write` takes a
  writable transaction it does not need.
- **Columns from `pkg/buncolgen/<entity>col`**: `cols.ID.Eq()`, `.In()`, `.Set()`, `.OrderDesc()`,
  `XScopeTenant(q, tenantInfo)`, `XRelations.Y`, `XTable.Alias`. Never a column name in a string.
- **Not found** goes through `dberror.HandleNotFoundError(err, "Commodity")`.
- **Batch reads are first-class.** Every entity read per parent row has a `GetByIDs`/`...ByIDs`
  method (`cols.ID.In(), bun.List(ids)`) so a dataloader can batch it ([api.md](api.md)).
- **Lists**: `ListConnection` is keyset-paginated (`dbhelper.CursorList`,
  `querybuilder.ApplyCursorFilters`) and runs its `COUNT` only when
  `req.Cursor.IncludeTotalCount` is set. Filters and sorts go through `querybuilder`, which
  allow-lists fields; never build a `WHERE` or `ORDER BY` from input. Offset `List`
  (`SafeLimit`/`SafeOffset`) exists for small, bounded sets only; anything a person scrolls is a
  connection.
- **Projection**: a connection selects the columns the GraphQL query asked for
  (`req.<Entity>Columns`), not `*`.
- **Select options** use `dbhelper.SelectOptions`.
- **Bulk writes** are one statement (`Set(...).Where(cols.ID.In(), ...)`) checked with
  `dberror.CheckBulkRowsAffected`, never a loop of single updates.
- **Versioned child sets** (a published version's rows copied from a draft) are copied in one
  `INSERT … SELECT` built from the generated `…InsertableColumns`, with new ids made in SQL
  (`ratematrixrepository.CopyVersion`), never read into Go and written back. Rows of an
  immutable version are keyed by `(parent, version_number)`, and a cache of them needs no
  invalidation (`pkg/ratetablecache`).

## Concurrency

- **Optimistic locking on every update**: keep `ov := e.Version`, `e.Version++`, update
  `WherePK().Where(cols.Version.Eq(), ov).Returning("*")`, then `dberror.CheckRowsAffected`. Zero
  rows is a `version` validation error the client shows as a conflict. Where the person needs to
  see what changed underneath them, use `editconflict` (`kernel/app/editconflict`) and
  `errortypes.NewEditConflictError`.
- **Row locks** when a write depends on what it read: a `ForUpdate` flag on the get request
  (`query.For("UPDATE")`), taken first inside the transaction
  (`operations/app/shipmentmoveservice/service.go`). Lock in a fixed order (parent before child)
  so two writers cannot deadlock.
- **Advisory locks** serialize work that has no row to lock:
  `pg_advisory_xact_lock(hashtextextended(?, 0))`, transaction-scoped, never session-scoped.
- **Lock timeouts**: a transaction that may wait passes `kernelports.TxOptions{LockTimeout: …}`.
- Keep transactions short: no network call (HTTP, Temporal, Redis, the imaging worker) inside
  one. Work that must follow a commit goes through `kernelports.AfterCommit`.

## Migrations

Migrations are [keel](https://github.com/emoss08/keel) files in `schema/migrations/`, one per
change, `<version>_<snake_name>.sql`; `task db-create-migration name=add_holds [kind=notx]` writes
one with the version from the clock (older files number versions within a day as counters, which
keel accepts). A transactional file runs whole, in one transaction with its row in
`keel_migrations`, so it needs no separators. There are no down migrations: fix forward with a new
migration, and rebuild a development database with `task db-reset`. `trenova-db up` (`cmd/db`) is
the deploy step that applies them, `trenova-devdb up` (`cmd/devdb`, `task db-migrate`) the
development one; each migration waits at most 5s for a lock and is retried, and after a run that
applied anything `SELECT trenova_rls.reconcile()` runs once, so a new migration does not call it
(the files written before keel still do). A new table (`20261231009920_agent_web_watches.sql` is
the model, less its closing `reconcile()`):

- composite primary key `pk_<table>` on `(id, organization_id, business_unit_id)`, named `fk_`
  and `ck_` constraints, `version BIGINT NOT NULL DEFAULT 0`, Unix-second `BIGINT` timestamps;
- an index for every query path the repository runs, tenant columns leading, partial where the
  query is (`WHERE status = 'Active'`), named `idx_<table>_<purpose>`;
- non-null `organization_id` and `business_unit_id`, which the run's `reconcile()` protects; any
  other shape calls `trenova_rls.apply_policy` or joins `trenova_rls.global_tables` in the same
  migration ([row-level-security.md](../row-level-security.md)).

An index on an existing table goes in its own file that starts with `--keel:notx`, one
`CREATE INDEX CONCURRENTLY IF NOT EXISTS` (or `DROP INDEX CONCURRENTLY IF EXISTS`) per
`--keel:split` part, as does `ALTER TYPE … ADD VALUE`. Pre-launch, reshape a table freely; write
the migration that gets there directly, with no dual-write or backfill phase for the old shape.

- **Stamp.** `task db-stamp` rewrites `schema/version_gen.go`; commit it with the migration. The API
  and worker link only that constant and wait at startup (`postgres/schemaversion`, at most 10
  minutes) until the database has applied it, so a server never runs against a schema behind it.
  CI and `dbcli.TestTheSchemaStampIsCurrent` fail a stale stamp.
- **Order and edits.** A migration older than the newest applied one is refused, and so is an
  applied migration whose file changed; CI's base-branch check (`migrate check -base-dir`) fails
  both on the pull request. Give the migration a newer version, or write a new one.
- **Lint.** `task db-lint` runs keel's rules and Trenova's (`schema/migrations/lint.go`:
  `bun-directive`, `reconcile-automatic`, `notx-idempotent`, `tenant-rls`) over pending migrations;
  `TestMigrationsSinceTheMoveToKeelPassLint` fails CI on an error finding in a new migration.
  `--keel:lint_ignore rule` at the top of a file skips a rule, with the reason in the PR.
- **Tests.** `seedtest` copies every test database from a keel template (`migrate.Template`, key
  `seedtest.TemplateKey` joined to the migrations' fingerprint), built once per server. A test of
  a migration's content reads it with `migrations.File("<version>_<name>")` or the `Set`.
- **Editions** add migration sources through `edition.Edition.Migrations`, merged into one version
  line; the db binaries read them, the server binaries never link them
  ([editions.md](../editions.md)).
- **Bun-migrated databases** are adopted on `trenova-db`'s first connection
  (`database/bunadopt`): every version in `bun_migrations` is recorded in `keel_migrations`, and a
  database Bun left gaps in is refused with the missing migrations named.

## Caching

- **Invalidate by epoch, not by key.** A cache that depends on many writes keys its entries
  under a per-tenant epoch and a write bumps it with `INCR` (`permissioncache.go`,
  `shipmentboardcache.go`, `aianalyticscache.go`). Deleting keys races with readers and misses
  keys you did not know about.
- **CDC-fed read caches** (`cache:customers|shipments|workers|documents`, `syncedread.go`) are
  written by GTC (`deploy/gtc/sinks.yaml`), never by Go. Go reads them, falls back to Postgres on
  a miss, and skips them when a read scope narrows the result.
- `pkg/redishelpers` for Redis JSON; `shared/lrucache` and `shared/ttlcache` in process. A cache
  never holds data a narrower permission or read scope could not see.

## Seeding

Seeds live in `internal/infrastructure/database/seeds/{base,development}` and are registered in
the generated `seeds/register_gen.go`; typed `SeedID` constants are generated into
`pkg/seedhelpers/seed_ids_gen.go`. `task db-create-seed name=MySeed [env=dev]` creates one,
`task generate-seeds` regenerates the IDs and `task db-seed` runs them.

Volume data for load and UI testing is not a seed: `task db-seed-volume [shipments=N months=M
seed=S]` (`trenova db seed-volume`, package `internal/infrastructure/database/volumeseed`) adds a
large, consistent data set to the development organization on top of the development seeds.
It numbers records through `pkg/seqgen` like the application, batches every insert, and
back-dates rows with `pulid.MustNewAt` and one `unnest` update per table, because model hooks
stamp the current time on insert. When an entity gains a required relation or a derived row the
application always writes (as every shipment gets an auto-order), generate it there too; write
an optional relation only in the states where the application would have written it (a
billing queue item only once a shipment is ready to invoice, a draft invoice only once the
item is approved, an email only for a customer with an email profile).

# Data access

Entities, repositories, transactions, migrations, locking, caching and seeds. Trenova's data layer
is [keel](https://github.com/emoss08/keel); keel's own guide (`docs/guide.md` in the keel
repository) is the API, and this page is how Trenova uses it. The reference vertical is equipment
(`internal/modules/equipment`). Paths are under `services/tms/`. Read
[row-level-security.md](../row-level-security.md) before writing a repository.

## Entities

A domain entity is a keel model in `domain/<entity>/` (`domain/tractor/tractor.go`):

- `keel.Table` names the table (`keel:"tractors,report=Equipment"`); every stored field carries a
  `keel:"<column>,…"` tag. `task keel-gen` writes the `<entity>db` package beside it: the typed
  table (`tractordb.Tractors`), its columns and relations, and its schema
  (`tractordb.TractorsSchema()`). Never write a column name as a string.
- `pulid.ID` with a per-entity prefix (`keel:"id,pk,autoid:trac_"`), primary key
  `(id, organization_id, business_unit_id)`; the tenant columns are tagged `pk,tenant` and keel
  fills them from the scope.
- `Version int64` tagged `version` for optimistic locking; `CreatedAt`/`UpdatedAt int64` Unix
  seconds tagged `autocreate`/`autoupdate`. keel stamps `autocreate` only when the field is zero, so
  a back-dated row keeps the value the caller set.
- Defaults a row needs on insert (a status, an entry method) go in a `BeforeCreate(ctx) error` hook
  on the model; keel writes zero values as they are and never substitutes a database default for
  them. `BeforeUpdate`, `BeforeSave` and `AfterLoad` exist for the other moments.
- Relations a repository loads or a list filters through are tagged: `belongs:<fk>`, `one:<fk>`,
  `many:<fk>`, `m2m:<join>.<local>.<remote>`, and `pick:<path>,first=…|last=…[,where=…]` for one row
  chosen along a to-many path (a shipment's first pickup stop). The tenant parents
  (`BusinessUnit`, `Organization`) are tagged too when something loads them.
- Values read through a join or a lateral subquery are `computed` fields (`nullzero` when the value
  may be NULL and the field is not a pointer); JSON documents are `jsonb`, decoded with
  `json.Number` for numbers held in an interface.
- Typed string enums with `String()` and `IsValid()`, backed by a `CHECK` constraint.
- Entity-level rules in `Validate(multiErr)` (ozzo); rules that need the database belong to the
  service's validator ([services.md](services.md#validation)).
- A reportable model's doc comments are its catalog descriptions ([reporting.md](../reporting.md)).

## Repositories

Shape: an fx `Params{DB *postgres.Connection; Logger}`, an unexported `repository`, and
`New(p) <module>ports.XRepository`.

- **Every method body runs in `dbtx.Read` or `dbtx.Write`** (or `ReadErr`/`WriteErr`,
  `Read2`/`Write2`, `Savepoint`) and passes `r.db.Keel(ctx)` to every terminal. `Keel(ctx)` is the
  system pool's DB under a system scope and the tenant pool's otherwise, and every terminal joins
  the transaction already on the context. `rlslint` fails an exported method that touches
  `Keel(ctx)` outside `dbtx`. Use `Read` for reads; a read in `dbtx.Write` takes a writable
  transaction it does not need.
- **The tenant is the scope.** A statement on a tenant table compares its tenant columns with the
  context's scope (`dbscope.WithTenant`, set by the HTTP and Temporal bindings) and fails with
  `keel.ErrNoScope` without one. `pagination.TenantInfo` on a request is not a filter: never add
  `organization_id`/`business_unit_id` predicates. Cross-tenant code declares `dbscope.WithSystem`,
  says `AllTenants()` on each query (keel refuses a system-scope query without it) and filters the
  tenant itself; it is listed in `rlslint`'s allowlist. A query that relies on a row-level security
  policy wider than the tenant scope (a person reads their own memberships in every organization)
  writes the policy's rule into the query and runs under a system scope
  (`userrepository.membershipReadScope`).
- **Errors.** keel returns every constraint violation as a `*keel.ConstraintError` resolved
  against the model: the constraint's columns without the tenant columns (index expressions such
  as `lower(code)` and partial-index predicates included), their JSON field names, and the message
  the model declares. Declare the constraints a user can hit on the model, with the migration's
  names; keel gen records them and the schemalint test (`keel.Verify`) fails when a name drifts:

  ```go
  func (Tractor) Constraints() []keel.Constraint {
  	return []keel.Constraint{
  		{Name: "idx_tractors_code", Message: "Tractor with this code already exists in your organization"},
  		{Name: "fk_tractors_fleet_code", Field: "fleetCodeId", Message: "Fleet code does not exist"},
  	}
  }
  ```

  `Field` is needed only when the form field is not the constraint's first non-tenant column (a key
  led by a reference, a child row reported under its parent's field, a column hidden from JSON).
  Each repository keeps one `dberror.Mapping{Entity: "Tractor"}` and passes every keel error
  through `Map(err, id)`; a constraint without a declared message gets a default for its kind.

  | keel | Caller sees |
  |---|---|
  | `keel.ErrNotFound` | `NotFoundError` "Tractor not found within your organization" |
  | `keel.ErrConflict` (`Save`, a version mismatch) | the `version` validation error the client shows as a conflict |
  | unique or exclusion violation | validation error on the constraint's field, `DUPLICATE` |
  | foreign key violation | validation error on the constraint's field, `INVALID_REFERENCE` |
  | not-null violation | validation error on the column's field, `REQUIRED` |
  | check violation | validation error on the constraint's field, `INVALID` |
  | `keel.ErrReadOnly` | a read-only refusal (`dberror.IsReadOnlyTransaction`) |
  | `keel.ErrTenantMismatch` | an authorization error and a tenant-boundary report, like a row-level security refusal |
  | `ErrNoScope`, `ErrAllTenantsRequired`, `ErrSystemScopeRequired` | a programming error: an internal error |

  Every violation maps, declared or not; code that needs to branch on one constraint checks
  `dberror.ExtractConstraintName(err)`, and that name is declared on the model too.
- **PostGIS.** Geometry and geography fields are `*postgis.Point` and `*postgis.Geometry` from
  `github.com/emoss08/keel/postgis`; their generated columns carry spatial methods
  (`DWithin`, `Distance`, `Contains`, `Within`, `Intersects`, `Area`, `SetExpr`), and
  `postgis.Pt`, `postgis.Buffer`, `postgis.FromGeoJSON` and `postgis.DistanceSphere` build values.
  Write no `ST_*` SQL by hand. GeoJSON in Go goes through `pkg/geojson`; a constant that a partial
  GiST index's predicate names stays inline with `keel.DynConst`.
- **Partial and expression indexes under generic plans.** A prepared statement settles on a plan
  without its parameters' values, so a predicate reaches a partial index only through a constant.
  keel writes a declared constant of an enum column into the statement (`s.Status.Eq(shipment.StatusNew)`
  is `status = 'New'`); a fixed constant of a plain string column takes `keel.DynConst`. A lookup an
  index on `lower(code)` serves compares `t.Code.Lower().Eq(strings.ToLower(v))`, never `t.Code.Eq(v)`.
  Pin a hot path with `keeltest.UsesIndex` in `schemalint/index_plans_integration_test.go`.
- **Vectors and text search.** pgvector distances come from `github.com/emoss08/keel/vector`
  (`CosineDistance(HalfVec(col, n), HalfVec(query, n))`, `EFSearch`); full-text search is
  `TSVectorCol.Match` with `SearchModes` (`AnyWebSearch`, `TSQuery`) and `SearchAlso`. Neither is
  written as SQL text.
- **Literals and grouping.** A bound value takes the type of its Go value where nothing else types
  it (a picked literal, an all-literal CASE, `concat` arguments), so `.Cast` is for another type
  only. Group and order by the same expression variable you picked: keel writes `GROUP BY 2`.
- **Enum assignments.** A field whose column is a Postgres enum and is assigned from a text
  expression declares `sqltype=<enum>` in its keel tag; `ToExpr` casts to it and `keel.Verify` holds
  the tag to the column.
- **Batch reads are first-class.** Every entity read per parent row has a `GetByIDs`/`…ByIDs`
  method so a dataloader can batch it ([api.md](api.md)):
  `t.Where(t.ID.In(ids...)).Limit(len(ids)).List(ctx, db)`. `In` binds one array, so 3 or 3,000
  IDs are one statement shape. `List` refuses a query without `Limit`; `All` is for sets bounded by
  the input.

### Reads

```go
row, err := tractordb.Tractors.Get(ctx, r.db.Keel(ctx), req.ID)
```

With relations or projected columns, build the query and end with `One`
(`tractorrepository.tractorQuery`). Belongs-to and has-one loads join into the statement; each
level of to-many loads is one more round trip for every parent. A pick relation joins as a
`LEFT JOIN LATERAL (… LIMIT 1)` for a one-row read and loads in one statement for all parents
otherwise.

```go
et := t.EquipmentType()
var loads listquery.Loads
loads.Add(includes.IncludeEquipmentType, func() (keel.Loader, error) {
	return listquery.Load(et.Query(), includes.EquipmentTypeColumns,
		listquery.When(includes.IncludeTenantDetails, et.BusinessUnit(), et.Organization())...)
})
loaders, err := loads.Result()
q = q.Load(loaders...)
```

A value from another table is a lateral join computed into the row:

```go
ec := equipmentcontinuitydb.EquipmentContinuities.As("lkc")
current := ec.Pick(ec.CurrentLocationID).Where(ec.EquipmentID.EqCol(t.ID), ec.IsCurrent.IsTrue()).Limit(1)
q = q.LeftJoinLateral(current).Compute(t.LastKnownLocationID, ec.CurrentLocationID)
```

A table of a module this one may not import is reached through a column port (see "Column
ports" in [module-architecture.md](../module-architecture.md#column-ports)): the port carries
that table's generated columns, and the query starts from `keel.Handle[*keel.Row]{Ref: col.Ref}`
(`lastknownlocation.Of`). Never `keel.Raw` for it. Aggregates use `keel.Into` over `Group(col).Pick(col, keel.Count())`;
`Count`, `CountUpTo` and `Exists` are terminals; `keel.Into[row]` decodes a `Pick` into a small
struct.

### Lists

- **Connections** (`listquery.Connection`) are keyset-paginated, and anything a person scrolls is
  one. The order is the request's sorts (relation paths included, through a queryable to-one or
  pick relation), `createdAt desc` when there are none, then `id`; keel appends the rest of the
  primary key, and nullable or joined sort keys put NULLs last. The cursor
  (`pagination.EncodeKeysetCursor`) carries keel's page cursor and the sort shape; a cursor read
  under another sort is a validation error on `after`. Every edge gets its own cursor
  (`Page.CursorAt`, surfaced as `CursorListResult.EdgeCursors`). The page and its exact count share
  one round trip, and the count runs only when `IncludeTotalCount` is set. `Columns` (the GraphQL
  projection's SQL names) narrows the row with `Pick`; keel adds the keys and order columns.

  ```go
  fc := fleetcodedb.FleetCodes
  return listquery.Connection(ctx, r.db.Keel(ctx), fc.Where(statusFilter), &listquery.Request[fleetcode.FleetCode]{
  	Filter:  req.Filter,
  	Cursor:  req.Cursor,
  	Columns: req.FleetCodeColumns,
  	Shape:   func(q *keel.Query[*fleetcode.FleetCode]) *keel.Query[*fleetcode.FleetCode] { return q.Load(fc.Manager()) },
  })
  ```

  Predicates the repository owns (a status, a class) go on the query passed in, so the page and the
  count both see them. `Shape` touches only the rows (loads, computed fields, lateral joins), never
  the count.
- **Offset lists** (`listquery.List`, `listquery.Offset`) return rows and the count in one
  `keel.Batch`, for small, bounded sets (agent tools, exports, REST lists through
  `pagination.List`, whose `after` is an opaque offset cursor). `Then` orders after the request's
  sorts and the search rank.
- **Select options** use `listquery.SelectOptions` with the search predicate the picker needs
  (`ContainsFold` over the code and description).
- **Filters, search and relation paths.** `listquery` resolves a request's JSON field names
  through the generated schema: a field filters when it has a JSON name and is a stored, selectable
  column that is not JSON, an array, bytea or a `tsvector`. `relation.field` resolves through a
  to-one relation (belongs, one or pick) the entity lists as `Queryable` in
  `GetPostgresSearchConfig().Relationships`, and joins it with its tenant condition. Values are
  converted to the column's kind and bound; enums compare as text; `contains`/`startswith`/`endswith`
  escape `%` and `_`. A filter it cannot apply is skipped. Search uses the entity's `tsvector`
  column under the `simple` and `english` configurations (`websearch` plus prefix matching while
  typing), or its configured searchable fields with `ILIKE`. Never build a `WHERE` or `ORDER BY`
  from input.
- **Filtering through another kind's record** (a column showing a linked record's number): a
  pick or belongs relation the entity lists as queryable. A belongs relation makes every column of
  the target filterable, which answers questions about fields outside the target's reference view;
  prefer a relation whose target is the narrow record.

### Writes

- **Create**: `Create` (one row), `CreateMany` (one round trip, `unnest`), `Copy` for imports. keel
  fills the `autoid`, the tenant columns from the scope (and refuses a row of another tenant),
  `created_at` and `updated_at`, after the model's `BeforeCreate`.
- **Update with optimistic locking**: `Save` compares the version the entity was read at, writes
  every updatable column, bumps the version into the entity and stamps `updated_at`. Zero rows is
  `keel.ErrConflict`, mapped to the `version` validation error. Name columns to write only those:
  `ec.Save(ctx, db, row, ec.IsCurrent, ec.SupersededAt)`. The entity is the full new state. Where
  the person needs to see what changed underneath them, use `editconflict`
  (`kernel/app/editconflict`) and `errortypes.NewEditConflictError`.
- **Bulk writes** are one statement, never a loop of single updates:
  `t.Where(t.ID.In(ids...)).UpdateReturning(ctx, db, t.Status.To(status))`, checked with
  `dberror.CheckBulkAffected(int64(len(rows)), "Tractor", ids)`; `Update` returns the count only,
  `Delete` deletes. A bulk update bumps `version` and stamps `updated_at`. Per-row values in one
  statement: `keel.Unnest` joined into the update. `listquery.OptionalInt64` assigns a nullable
  column from a pointer.
- **Upsert**: `Upsert(ctx, db, rows, keel.OnConflict(cols…).Update(cols…))`; `CreateOrSkip` is
  `ON CONFLICT … DO NOTHING`. A partial unique index is matched with `.Where(…)` on the conflict,
  with constants only.
- **Delete**: `DeleteRow(ctx, db, &x.X{ID: id})`, mapped through the entity's `Mapping`.
- **Versioned child sets** (a published version's rows copied from a draft) are copied in one
  `INSERT … SELECT` with new ids made in SQL (`ratematrixrepository.CopyVersion`), never read into
  Go and written back. Rows of an immutable version are keyed by `(parent, version_number)`, and a
  cache of them needs no invalidation (`pkg/ratetablecache`).

### Raw SQL

`keel.Exec`, `QueryValue`, `QueryAll` and `QueryRows` join the context's transaction but add no
tenant predicate: write it, and keep the text constant with values as `$n`. Inside a typed query,
`keel.Raw("…?…", col, value)` writes columns and binds values. A CTE is `keel.CTE(name, columns,
query)` declared with `With`; a recursive one is `keel.RecursiveCTE`.

### Validators

Validators build one checker over the entity's table:

```go
checker := validationframework.NewKeelChecker(p.DB, tractordb.TractorsSchema())
validationframework.NewTenantedValidatorBuilder[*tractor.Tractor]().
	WithUniquenessChecker(checker).
	WithReferenceChecker(checker)
```

The checker answers references to the entity's table and every table it relates to; a table it
does not relate to gets a `WithCustomReferenceCheck` with `validationframework.KeelReferenceCheck`,
or, when the table belongs to a module this one may not import,
`validationframework.KeelTableReferenceCheck(p.DB, "gl_accounts")` (one constant `EXISTS` over the
table's `id`, `organization_id` and `business_unit_id`). A uniqueness check that must see rows
row-level security hides (other organizations of the business unit) wraps the checker in a
`UniquenessChecker` that puts `dbscope.WithSystem` on the context and pins the request's
`BusinessUnitID` to the caller's (`organizationservice.businessUnitWideChecker`).

## Pools and transactions

| Role | Login | Pool |
|---|---|---|
| tenant | `database.user` | `Connection` (and `ReportingConnection`, its own pool) |
| system | `database.system.user` | the same `Connection`, for contexts declared with `dbscope.WithSystem` |

`dbpool.Open` (`internal/infrastructure/postgres/dbpool`) opens one keel DB per role on a
`pgxpool.Pool`. Its `OnBegin` guard (`scopeguard.go`) sends `SET LOCAL trenova.scope …` with the
signed scope (and the read scope) at the start of every transaction, so every statement is row-level
security scoped.

| Concern | Where it lives |
|---|---|
| Session timeouts (`statement_timeout`, `lock_timeout`, `idle_in_transaction_session_timeout`) | pgx runtime parameters, from `database.*Timeout` (`connection.go`, `sessionParams`) |
| `application_name`, `connect_timeout`, `sslmode` | the DSN (`config.Config.GetDSN`) |
| Pool size and lifetimes | `maxOpenConns`, `connMaxLifetime`, `connMaxIdleTime` (and `database.system.maxOpenConns`) |
| Enum types | keel registers every enum and its array type on each new connection with one catalog query |
| Slow queries, tracing, query timing | one `slowQueryLog` per pool and `tracingObserver` (`go.sql.query_timing`) |
| Development query log | `database.queryLog` (`off`, `errors`, `all`; development only): `keel/keellog` prints each statement to stderr with its values, caller, transaction and outcome; `errors` prints failed and slow ones (`keelobserve.go`) |
| Constraint catalog, generic plans | keel reads the constraints of `keeltables.All()` at open (`dbpool.Options.Tables`); `keeltest.UsesIndex` asserts a hot-path query reaches its index under a generic plan (`schemalint/index_plans_integration_test.go`) |
| Pool metrics | `trenova_db_pool_*` from `pgxpool.Stat` (`metrics.Database.RegisterPoolStats`) |

`dbtx.Read`/`Write` and services' `kernelports.Transactor.Transact(ctx, opts, fn)` go through
`Connection.Transact`:

- A call whose context already holds a transaction of the same DB and the same scope joins it
  (applying a `LockTimeout` and `Settings` as `keel.Setting`s for the call, restored after it); any other call opens a new, detached transaction
  (`keel.Detached()`), so a context with another tenant's scope, or a `RunDetached` one, never
  writes in the caller's transaction.
- `TxOptions` carries `ReadOnly` (a `keel.Read` transaction), `Isolation` (read committed,
  repeatable read or serializable; anything else is `ErrUnsupportedIsolation`) and `LockTimeout`.
- `dbtx.Savepoint` uses `keel.Savepoint` inside a transaction and runs the function directly
  outside one.
- `kernelports.AfterCommit` callbacks run after the outermost commit, never after a rollback.
  Services, which do not know how a repository writes, use it rather than `keel.AfterCommit`.

Start transactions with `dbtx` or `Transact`, never `keel.Write`/`keel.Read` directly: those skip
the same-scope join rule and the after-commit hooks.

## Concurrency

- **Optimistic locking on every update** through `Save` (above).
- **Row locks** when a write depends on what it read: `Lock(keel.ForUpdate)` on the read, taken
  first inside `dbtx.Write` (`operations/app/shipmentmoveservice/service.go`). Lock in a fixed
  order (parent before child) so two writers cannot deadlock. `keel.NoWait` fails with SQLSTATE
  `55P03`; `keel.SkipLocked` claims work.
- **Advisory locks** serialize work that has no row to lock: `keel.LockKey(ctx, db, key)` and
  `keel.TryLockKey`, transaction-scoped, never session-scoped.
- **Lock timeouts**: a transaction that may wait passes `kernelports.TxOptions{LockTimeout: …}`.
- Keep transactions short: no network call (HTTP, Temporal, Redis, the imaging worker) inside
  one. Work that must follow a commit goes through `kernelports.AfterCommit`.

## Migrations

Migrations are keel files in `schema/migrations/`, one per change, `<version>_<snake_name>.sql`;
`task db-create-migration name=add_holds [kind=notx]` writes one with the version from the clock
(older files number versions within a day as counters, which keel accepts). Until the clock passes
the newest version (the `20261231…` counters run ahead of it), rename the file to the next counter
after the newest, or it is refused as older than an applied migration. A transactional file
runs whole, in one transaction with its row in `keel_migrations`, so it needs no separators. There
are no down migrations: fix forward with a new migration, and rebuild a development database with
`task db-reset`. `trenova-db up` (`cmd/db`) is the deploy step that applies them, `trenova-devdb up`
(`cmd/devdb`, `task db-migrate`) the development one; each migration waits at most 5s for a lock
and is retried, and after a run that applied anything `SELECT trenova_rls.reconcile()` runs once,
so a new migration does not call it (the files written before keel still do). A new table
(`20261231009920_agent_web_watches.sql` is the model, less its closing `reconcile()`):

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
A new column, relation or table needs `task keel-gen` in the same change, and
`keel_verify_integration_test.go` fails when a model and the migrated schema disagree.

- **Stamp.** `task db-stamp` rewrites `schema/version_gen.go`; commit it with the migration. The API
  and worker link only that constant and wait at startup (`postgres/schemaversion`, at most 10
  minutes) until the database has applied it, so a server never runs against a schema behind it.
  CI and `dbcli.TestTheSchemaStampIsCurrent` fail a stale stamp.
- **Order and edits.** A migration older than the newest applied one is refused, and so is an
  applied migration whose file changed; CI's base-branch check (`migrate check -base-dir`) fails
  both on the pull request. Give the migration a newer version, or write a new one.
- **Lint.** `task db-lint` runs keel's rules and Trenova's (`schema/migrations/lint.go`:
  `reconcile-automatic`, `notx-idempotent`, `tenant-rls`) over pending migrations;
  `TestMigrationsSinceTheMoveToKeelPassLint` fails CI on an error finding in a new migration.
  `--keel:lint_ignore rule` at the top of a file skips a rule, with the reason in the PR.
- **Editions** add migration sources through `edition.Edition.Migrations`, merged into one version
  line; the db binaries read them, the server binaries never link them
  ([editions.md](../editions.md)).

## Tests

- **SQL**: render statements with `keeltest.Render(t, q)` (`internal/testutil/keeltest`), a keel DB
  that never connects, under a test tenant; assert the fragments that matter
  (`tractorrepository/tractor_test.go`). A test that pins a whole statement uses
  `keeltest.Snapshot(t, q)`: the formatted SQL and its values live in
  `testdata/keel/<test>.sql`, rewritten with `KEEL_UPDATE=1 go test ./…` when keel's rendering
  changes, so the change is one reviewed diff. A unit test that needs a database is an integration test.
- **Integration**: `seedtest` copies every test database from a keel template (`migrate.Template`,
  key `seedtest.TemplateKey` joined to the migrations' and the seeds' fingerprints), built once per
  server: `SetupTestDB` from migrations plus reference seeds, `seededtest.SetupTestDB` with the
  fixtures too. `equipmenttest.NewDatabase(t)` gives a migrated database with two tenants (`A`,
  `B`, each with a scoped `Ctx`), an RLS-enforcing `Conn` and an `Admin` keel DB for setup. Assert
  the second tenant sees nothing, page through every row with `NextCursor` and an edge cursor, and
  map every constraint the repository declares. Run with
  `TRENOVA_TEST_POSTGRES_DSN=… go test -tags=integration,nomsgpack ./internal/modules/<module>/...`.
  A test of a migration's content reads it with `migrations.File("<version>_<name>")` or the `Set`.
- Service tests fake `kernelports.Transactor` with `dbtest.NopConnection`, and list fakes build
  pages with `internal/testutil/pagetest`.

## Caching

- **Invalidate by epoch, not by key.** A cache that depends on many writes keys its entries
  under a per-tenant epoch and a write bumps it with `INCR` (`permissioncache.go`,
  `shipmentboardcache.go`, `aianalyticscache.go`). Deleting keys races with readers and misses
  keys you did not know about.
- **CDC-fed read caches** (`cache:customers|shipments|workers|documents`, `syncedread.go`) are
  written by GTC (`deploy/gtc/sinks.yaml`), never by Go. Go reads them (decoding rows by their
  keel column names with `pkg/columnmap`), falls back to Postgres on a miss, and skips them when a
  read scope narrows the result.
- `pkg/redishelpers` for Redis JSON; `shared/lrucache` and `shared/ttlcache` in process. A cache
  never holds data a narrower permission or read scope could not see.

## Seeding

Seeds are [keel/seed](https://github.com/emoss08/keel) seeds in two registries, written with keel
against the generated tables (never raw SQL):

- **Reference** (`internal/seeds/reference`, `seed.Reference`): data every environment needs.
  Global rows (`us_states`, `dot_hazmat_references`, `ifta_jurisdictions` from embedded YAML in
  `reference/data`, upserted on their natural keys; `jurisdiction_rules_baseline`) and per-tenant
  defaults (`PerTenant`: `chart_of_accounts`, `system_document_types`,
  `service_failure_reason_codes`, `document_template_starters`, `system_agent_definitions`,
  `table_change_alert_allowlist`, `administrator_permissions`), plus `rls_scope_key`, which
  installs the configured row-level security key. `trenova-db up` applies them after the
  migrations on every deploy; a seed runs only when its fingerprint (name, `Version`, `Inputs`
  and `Fingerprint` function) changed, so an unchanged deploy does no seed work. A seed that
  inserts only missing rows bumps `Version` when its rows change; one built from a Go catalog
  or the permission registry fingerprints that catalog instead. Per-tenant seeds create
  nothing for an organization made later: `tenantbootstrap.CreateReferenceData` gives every new
  organization the same rows when it is created.
- **Fixtures** (`internal/seeds/fixtures`, `seed.Fixture`): the development data set (the admin
  account `admin` / `admin123!`, the Trenova Logistics and Trenova Transportation
  organizations, workers, equipment, shipments, ledgers, agent activity). Only `trenova-devdb`
  links them; each runs once per database, and a database marked production refuses them.
  `requireDevelopment` refuses publicly known credentials on staging too.

Seed tenancy comes from `seedenv.Env`: the system scope for bookkeeping and global seeds, and an
RLS tenant scope per organization for `PerTenant` seeds, with the loaded configuration on the
context (`seedenv.Config`). `task db-migrate` (`trenova-devdb up -env development`) migrates and
applies every changed seed, `task db-seed [name=x]` applies seeds alone, `task db-seed-plan` says
which would run and why, and `task db-reset` recreates the database from the migrated template
(which carries the reference seeds) and applies the fixtures. To add a seed, add it to the right
registry with `Seeds.Add(seed.Seed{…})`, name its `After` dependencies, and list any data file
under `Inputs` so editing the file reruns it.

Volume data for load and UI testing is not a seed: `task db-seed-volume [shipments=N months=M
seed=S]` (`trenova-devdb volume`, package `internal/seeds/volume`) adds a large, consistent data
set to the development organization on top of the fixtures, in a tenant scope, and refuses a
database not marked development or test. It streams rows with `COPY` (`Handle.Copy`), numbers
records through `pkg/seqgen` like the application (over keel-backed sequence and format ports),
and back-dates rows with `pulid.MustNewAt` and one `keel.Unnest` update per table, because keel
stamps `autoupdate` columns with the current time on every write. An update that must not move a
back-dated row's `updated_at` assigns `keepUpdated(h)`. When an entity gains a required relation
or a derived row the application always writes (as every shipment gets an auto-order), generate it
there too; write an optional relation only in the states where the application would have written
it (a billing queue item only once a shipment is ready to invoice, a draft invoice only once the
item is approved, an email only for a customer with an email profile).

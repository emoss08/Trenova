# Converting a module to keel

The playbook for moving one module's data access from Bun to [keel](https://github.com/emoss08/keel).
The equipment module (`internal/modules/equipment`) is the converted reference: read its
repositories beside this page. [keel-migration.md](keel-migration.md) is what the shared pools and
transactions guarantee; keel's own guide (`docs/guide.md` in the keel repository, "Coming from
Bun" first) is the API. Paths are under `services/tms/`.

A module is converted when nothing under it imports `github.com/uptrace/bun`, `pkg/querybuilder`,
`pkg/dbhelper` or `pkg/buncolgen`, except a domain package that still needs `bun.BaseModel`, `bun:`
tags or a `BeforeAppendModel` hook because Bun code in another module still writes the model.

## What changes, in one table

| Bun | keel |
|---|---|
| `r.db.DBForContext(ctx).NewSelect().Model(&e)` | `fleetcodedb.FleetCodes.Get(ctx, r.db.Keel(ctx), id)`, inside `dbtx.Read` |
| `buncolgen` columns, `?TableAlias`, column strings | the generated table: `fc := fleetcodedb.FleetCodes; fc.Code.Eq(v)` |
| `WhereGroup(… ScopeTenant(q, req.TenantInfo))` | nothing: the scope on the context is the tenant |
| `querybuilder.ApplyCursorFilters` + `dbhelper.CursorList` + a separate `Count` | `listquery.Connection` (page and count in one round trip) |
| `querybuilder.ApplyFilters` + `ScanAndCount` | `listquery.List` (rows and count in one round trip) |
| `dbhelper.SelectOptions` | `listquery.SelectOptions` |
| `Relation("A").Relation("A.B", dbhelper.RelationColumns(cols))` | `Load(a.Pick(…).Load(a.B()))`, or `listquery.Load(a.Query(), cols, a.B())` |
| `Version++` + `Where("version = ?")` + `CheckRowsAffected` | `Save`, errors through a `dberror.Mapping` |
| `Returning("*")` on a bulk `Update` | `listquery.UpdateReturning` |
| `NewBunUniquenessCheckerScoped` / `NewBunReferenceCheckerScoped` | `validationframework.NewKeelChecker(p.DB, xdb.XsSchema())` |
| `kernelports.DBConnection` + `WithTx(…, func(ctx, bun.Tx))` in a service | `kernelports.Transactor` + `Transact(ctx, opts, func(ctx))` |
| `pagination.CursorValueSet` embedded in the model | nothing: keel's keyset pages carry the order values |
| a sqlmock unit test of the SQL Bun wrote | a `keeltest.Render` test of the SQL keel writes, and an integration test |

## The tenant is the scope

A keel statement on a tenant table compares its tenant columns with the context's scope
(`dbscope.WithTenant`, set by the HTTP and Temporal bindings) and fails with `keel.ErrNoScope`
without one. `pagination.TenantInfo` on a repository request is not a filter any more: drop the
`organization_id`/`business_unit_id` predicates, never re-add them. A system-scope caller says
`AllTenants()` on each query and filters the tenant explicitly. Tests put the scope on the
context (`equipmenttest.Database.A.Ctx`).

## Repository shape

Unchanged: `Params{DB *postgres.Connection; Logger}`, one `dbtx.Read`/`Write` per exported method,
`r.db.Keel(ctx)` passed to every terminal, the generated handle (`fc := fleetcodedb.FleetCodes`)
bound to a short local. Declare one `dberror.Mapping` per entity with the constraints a caller can
hit (names from the migration):

```go
var tractorErrors = dberror.Mapping{
	Entity: "Tractor",
	Unique: []dberror.Constraint{
		{Name: "idx_tractors_code", Field: "code", Message: "Tractor with this code already exists in your organization"},
	},
}
```

`Map` turns `keel.ErrNotFound` into the not-found error, `keel.ErrConflict` into the version
mismatch, and a mapped constraint into a field validation error; anything else passes through.

## Patterns

Each pattern shows the Bun code it replaces and the keel code that replaced it.

### Get by ID

```go
entity := new(fleetcode.FleetCode)
err := r.db.DBForContext(ctx).NewSelect().Model(entity).
	WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where("fc.id = ?", req.ID).
			Where("fc.organization_id = ?", req.TenantInfo.OrgID).
			Where("fc.business_unit_id = ?", req.TenantInfo.BuID)
	}).Scan(ctx)
return entity, dberror.HandleNotFoundError(err, "FleetCode")
```

```go
entity, err := fleetcodedb.FleetCodes.Get(ctx, r.db.Keel(ctx), req.ID)
if err != nil {
	return nil, fleetCodeErrors.Map(err, req.ID.String())
}
return entity, nil
```

With relations or projected columns, build the query and end with `One`
(`tractorrepository.tractorQuery`).

### Get by IDs (dataloaders)

```go
if len(req.IDs) == 0 {
	return []*equipmenttype.EquipmentType{}, nil
}
et := equipmenttypedb.EquipmentTypes
return et.Where(et.ID.In(req.IDs...)).Limit(len(req.IDs)).List(ctx, r.db.Keel(ctx))
```

`In` binds one array, so a list of 3 or 3,000 IDs is one statement shape. `List` refuses a query
without `Limit`; `All` is for sets bounded by the input (`tractorrepository.GetByCodes`).

### Offset list with filters and a count

```go
total, err := r.db.DBForContext(ctx).NewSelect().Model(&entities).
	Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		q := querybuilder.ApplyFilters(sq, "fc", req.Filter, (*fleetcode.FleetCode)(nil))
		return q.Limit(…).Offset(…)
	}).ScanAndCount(ctx)
```

```go
fc := fleetcodedb.FleetCodes
return listquery.List(ctx, r.db.Keel(ctx), fc.Query(), &listquery.Request[fleetcode.FleetCode]{
	Filter: req.Filter,
	Shape:  func(q *keel.Query[fleetcode.FleetCode]) *keel.Query[fleetcode.FleetCode] { return q.Load(fc.Manager()) },
	Then:   []keel.Order{fc.CreatedAt.Desc()},
})
```

The rows and the count go out in one `keel.Batch`. `Shape` touches only the rows (loads, computed
fields, lateral joins), never the count; `Then` orders after the request's sorts and the search
rank. Offset lists are for small, bounded sets (agent tools, exports); anything a person scrolls is
a connection.

### Cursor connection

```go
var totalCount *int
if req.Cursor.IncludeTotalCount { /* a second, separately filtered Count query */ }
result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*fleetcode.FleetCode]{
	Filter: req.Filter, Cursor: req.Cursor, TotalCount: totalCount,
	Query: func(items *[]*fleetcode.FleetCode) *bun.SelectQuery { … },
	Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
		return querybuilder.ApplyCursorFilters(sq, "fc", req.Filter, req.Cursor, (*fleetcode.FleetCode)(nil))
	},
})
```

```go
fc := fleetcodedb.FleetCodes
return listquery.Connection(ctx, r.db.Keel(ctx), fc.Where(statusFilter), &listquery.Request[fleetcode.FleetCode]{
	Filter:  req.Filter,
	Cursor:  req.Cursor,
	Columns: req.FleetCodeColumns,
	Shape:   func(q *keel.Query[fleetcode.FleetCode]) *keel.Query[fleetcode.FleetCode] { return q.Load(fc.Manager()) },
})
```

- Predicates the repository owns (a status, a class) go on the query passed in, so the page and the
  count both see them (`trailerrepository.List`).
- The order is the request's sorts, `createdAt desc` when there are none, then `id`; keel appends
  the rest of the primary key. Nullable columns sort their NULLs last. A sort on a relation path is
  refused with a `sort.field` validation error (keel pages order by the table's own columns).
- The cursor (`pagination.EncodeKeysetCursor`) carries keel's page cursor and the sort shape; a
  cursor read under another sort is a validation error on `after`. Every edge gets a cursor
  (`CursorListResult.EdgeCursors`): the last edge carries keel's, the others resume by skipping
  rows after the page's own cursor (bounded by `pagination.MaxCursorSkip`).
- The page and its exact count share one round trip when `IncludeTotalCount` is set.
- `Columns` (the GraphQL projection's SQL names) narrows the row with `Pick`; keel adds the keys
  and the order columns.

### Filters, search and relation paths

`listquery` resolves a request's JSON field names through the generated schema: a field filters
when it has a JSON name, is a stored, selectable column and is not JSON, an array, bytea or a
`tsvector`. `relation.field` resolves through a to-one relation the entity lists as `Queryable` in
`GetPostgresSearchConfig().Relationships` (only the name matters; a custom relationship becomes a
keel `pick:` relation of the same name) and joins it with its tenant condition. Values are
converted to the column's kind and bound; enums compare as text; `contains`/`startswith`/`endswith`
escape `%` and `_`. A filter it cannot apply is skipped, as before. Search uses the entity's
`tsvector` column under the `simple` and `english` configurations (`websearch` plus prefix
matching while typing), or its configured searchable fields with `ILIKE`.

### Select options

```go
et := equipmenttypedb.EquipmentTypes
q := et.Pick(et.ID, et.Code, et.Description, et.Status).
	Where(et.Status.Eq(domaintypes.StatusActive), classFilter(req.Classes))
return listquery.SelectOptions(ctx, r.db.Keel(ctx), q, req.SelectQueryRequest, func(s string) keel.Pred {
	return keel.Or(et.Code.ContainsFold(s), et.Description.ContainsFold(s))
})
```

### Relation loads, nested, with columns

```go
q = q.Relation(rel.EquipmentType, dbhelper.RelationColumns(includes.EquipmentTypeColumns)).
	Relation(buncolgen.Rel(rel.EquipmentType, equipmentTypeRel.Organization))
```

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

Belongs-to and has-one loads join into the statement; each level of to-many loads is one more
round trip for every parent. A relation a repository loads needs a `keel:"belongs:…"` tag on the
model (the tenant parents `BusinessUnit`/`Organization` included); run `task keel-gen` after
adding one.

### A value from another table (computed field, lateral join)

```go
q.ColumnExpr("ec.current_location_id AS last_known_location_id").
	Join("LEFT JOIN equipment_continuity AS ec ON ec.equipment_id = trac.id AND …")
```

```go
ec := equipmentcontinuitydb.EquipmentContinuities.As("lkc")
current := ec.Pick(ec.CurrentLocationID).Where(ec.EquipmentID.EqCol(t.ID), ec.IsCurrent.IsTrue()).Limit(1)
q = q.LeftJoinLateral(current).Compute(t.LastKnownLocationID, ec.CurrentLocationID)
```

The model field is `keel:"…,computed"` (add `nullzero` when the value can be NULL and the field is
not a pointer). A table of a module this one may not import is reached with `keel.Raw` scalar
subqueries (`lastknownlocation.Of`).

### Create

```go
_, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx)
```

```go
if err := tractordb.Tractors.Create(ctx, r.db.Keel(ctx), entity); err != nil {
	return nil, tractorErrors.Map(err, entity.ID.String())
}
```

keel fills the `autoid`, the tenant columns from the scope (and refuses a row of another tenant),
`created_at` and `updated_at`. Many rows: `CreateMany` (one round trip, `unnest`), or `Copy` for
imports.

### Update with optimistic locking

```go
ov := entity.Version
entity.Version++
res, err := db.NewUpdate().Model(entity).WherePK().Where("version = ?", ov).Returning("*").Exec(ctx)
err = dberror.CheckRowsAffected(res, "Tractor", entity.ID.String())
```

```go
if err := tractordb.Tractors.Save(ctx, r.db.Keel(ctx), entity); err != nil {
	return nil, tractorErrors.Map(err, entity.ID.String())
}
```

`Save` compares the version the entity was read at, writes every updatable column, bumps the
version into the entity and stamps `updated_at`. Name columns to write only those:
`ec.Save(ctx, db, row, ec.IsCurrent, ec.SupersededAt)` (`equipmentcontinuityrepository`). Bun's
`OmitZero` has no equivalent: the entity is the full new state.

### Delete

```go
res, err := db.NewDelete().Model((*x.X)(nil)).Where("x.id = ?", id).Where(tenant…).Exec(ctx)
err = dberror.CheckFound(res, "X")
```

```go
err := xdb.Xs.DeleteRow(ctx, r.db.Keel(ctx), &x.X{ID: id})
return xErrors.Map(err, id.String())
```

### Bulk update and delete

```go
res, err := db.NewUpdate().Model(&entities).
	WhereGroup(" AND ", …tenant…, cols.ID.In(), bun.List(ids)).
	Set(cols.Status.Set(), status).Returning("*").Exec(ctx)
err = dberror.CheckBulkRowsAffected(res, "Tractor", ids)
```

```go
t := tractordb.Tractors
rows, err := listquery.UpdateReturning(ctx, r.db.Keel(ctx), t.Where(t.ID.In(ids...)), t.Status.To(status))
err = dberror.CheckBulkAffected(int64(len(rows)), "Tractor", ids)
```

A bulk update bumps `version` and stamps `updated_at`. Without the rows back:
`n, err := t.Where(…).Update(ctx, db, t.Status.To(s))`; delete: `t.Where(…).Delete(ctx, db)`. Per-row
values in one statement: `keel.Unnest` joined into the update.

### Upsert

```go
db.NewInsert().Model(&rows).On("CONFLICT (organization_id, business_unit_id, code) DO UPDATE").
	Set("name = EXCLUDED.name").Exec(ctx)
```

```go
c := xdb.Xs
inserted, err := c.Upsert(ctx, db, rows, keel.OnConflict(c.OrganizationID, c.BusinessUnitID, c.Code).Update(c.Name))
```

`CreateOrSkip(ctx, db, rows, targets...)` is `ON CONFLICT … DO NOTHING`. A partial unique index is
matched with `.Where(…)` on the conflict, with constants only.

### Raw SQL, CTEs and tables a module may not import

```go
count, err := db.NewSelect().Table("shipment_moves").Where("id = ?", id).Where(…).Count(ctx)
```

```go
done, err := keel.QueryValue[bool](ctx, r.db.Keel(ctx),
	"SELECT EXISTS (SELECT 1 FROM shipment_moves WHERE id = $1 AND organization_id = $2 AND business_unit_id = $3 AND status = 'Completed')",
	moveID.String(), orgID.String(), buID.String())
```

Raw statements (`keel.Exec`, `QueryValue`, `QueryAll`, `QueryRows`) join the context's
transaction but add no tenant predicate: write it, and keep the text constant with values as
`$n`. Inside a typed query, `keel.Raw("…?…", col, value)` writes columns and binds values. A CTE is
`keel.CTE(name, columns, query)` declared with `With`; a recursive one is `keel.RecursiveCTE`.

### Aggregates and counts

```go
var rows []statusCount
err := db.NewSelect().Model((*x.X)(nil)).ColumnExpr("status, count(*) AS n").Group("status").Scan(ctx, &rows)
```

```go
x := xdb.Xs
rows, err := keel.Into[statusCount](ctx, db, x.Group(x.Status).Pick(x.Status, keel.Count()))
n, err := x.Where(…).Count(ctx, db)
capped, err := x.Where(…).CountUpTo(ctx, db, 10_000)
```

`dbhelper.CountByID` becomes `keel.Into` over `Group(col).Pick(col, keel.Count())`.

### Exists

```go
exists, err := db.NewSelect().TableExpr("user_organization_memberships").ColumnExpr("1").Where(…).Exists(ctx)
```

```go
m := tenantdb.OrganizationMemberships
exists, err := m.Where(m.UserID.Eq(userID)).Exists(ctx, db.Keel(ctx))
```

### A projection into a struct

```go
var out []row
err := db.NewSelect().Model((*x.X)(nil)).Column("id", "code").Scan(ctx, &out)
```

```go
type row struct {
	ID   pulid.ID
	Code string
}
out, err := keel.Into[row](ctx, db, x.Pick(x.ID, x.Code).Where(…).Limit(100))
```

`Pick` with only the table's own columns still decodes into the model with the other fields zero.

### Row locks

```go
db.NewSelect().Model(e).Where(…).For("UPDATE").Scan(ctx)
```

```go
e, err := x.Where(x.ID.Eq(id)).Lock(keel.ForUpdate).One(ctx, db)
```

Inside `dbtx.Write`. `keel.NoWait` fails with SQLSTATE `55P03`; `keel.SkipLocked` claims work.

### Advisory locks

```go
db.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key)
```

```go
err := keel.LockKey(ctx, r.db.Keel(ctx), "invoice-run:"+orgID.String())
got, err := keel.TryLockKey(ctx, r.db.Keel(ctx), key)
```

### Services and validators

A service that only needs atomicity takes `kernelports.Transactor` and calls
`s.db.Transact(ctx, kernelports.TxOptions{}, func(ctx context.Context) error { … })`; repositories
called inside join that transaction. Validators build one checker over the entity's table:

```go
checker := validationframework.NewKeelChecker(p.DB, tractordb.TractorsSchema())
validationframework.NewTenantedValidatorBuilder[*tractor.Tractor]().
	WithUniquenessChecker(checker).
	WithReferenceChecker(checker)
```

The checker answers references to the entity's table and every table it relates to; a table it
does not relate to gets a `WithCustomReferenceCheck` with `validationframework.KeelReferenceCheck`.

## Models

- Keep `bun.BaseModel`, the `bun:` tags and `BeforeAppendModel` while Bun code anywhere writes the
  model (seeds, sample data and other modules' repositories do for equipment); keel stamps IDs and
  timestamps from its own tags and never calls the hook.
- Delete `pagination.CursorValueSet` from the model.
- Tag every relation a repository loads; a Bun-only `scanonly` value read through a join becomes a
  `computed` field.
- `fieldmap_gen.go` (buncolgen's querybuilder bridge) stays until buncolgen goes; converted code
  never calls it.

## Tests

- **SQL**: render statements with `keeltest.Render(t, q)` (`internal/testutil/keeltest`), a keel DB
  that never connects, under a test tenant; assert the SQL and the arguments
  (`tractorrepository/tractor_test.go`). sqlmock cannot see keel: a unit test that needs a
  database is an integration test.
- **Integration**: `equipmenttest.NewDatabase(t)` gives a migrated database with two tenants
  (`A`, `B`, each with a scoped `Ctx`), an RLS-enforcing `Conn` and an `Admin` keel DB for setup;
  `NewDevelopmentDatabase(t)` runs the development seeds first. Assert the second tenant sees
  nothing, page through every row with `NextCursor` and an edge cursor, and map every constraint
  the repository declares. Run with
  `TRENOVA_TEST_POSTGRES_DSN=… go test -tags=integration,nomsgpack ./internal/modules/<module>/...`.
- Service tests fake `kernelports.Transactor` with `dbtest.NopConnection`.

## Checklist

1. Tag every relation the module's repositories load, `computed` for joined values; `task keel-gen`.
2. Delete `pagination.CursorValueSet` from the models.
3. Rewrite each repository method on the generated tables inside `dbtx`, with a `dberror.Mapping`;
   drop every tenant predicate.
4. Lists through `listquery` (`Connection`, `List`, `SelectOptions`), loads through
   `listquery.Load`/`Loads`; bulk writes through `Update`/`UpdateReturning`.
5. Validators on `NewKeelChecker`; services on `kernelports.Transactor`.
6. A REST list that relied on `pagination.List`'s cursor side channel calls the connection through
   `pagination.CursorList`.
7. Replace Bun SQL tests with `keeltest` render tests; write an integration test per repository
   with a second tenant.
8. `grep -rn 'uptrace/bun\|querybuilder\|dbhelper\|buncolgen' internal/modules/<module>` lists only
   domain models Bun still writes.
9. Run the module's unit and integration tests, `go run ./scripts/archlint`, the `rlslint` tests,
   and `go vet` over the packages that import the module's ports.

## What goes when Bun goes

Kept beside the keel code only for unconverted modules, and deleted with Bun:

| Bun-only | Keel replacement |
|---|---|
| `pkg/querybuilder` | `pkg/listquery` |
| `pkg/dbhelper` | `pkg/listquery`, keel's `Count`/`Into`/`ContainsFold` |
| `pkg/buncolgen`, `fieldmap_gen.go`, `domaintypes.FieldConfiguration` | the generated keel tables |
| `pagination.CursorValueSet`, `CursorValueColumn`, `CursorValueCarrier`, `QueryOptions.CursorColumns`/`CursorSort`/`CursorError`/`UseCursor`, `Cursor.CreatedAt`/`Values`/`ID`, `EncodeCursorFromEntity*` | `Cursor.Keyset`/`Skip`, `CursorListResult.EdgeCursors` |
| `validationframework.NewBun*Checker*`, `ScopedDB`, `NewUSStateReferenceCheck` | `NewKeelChecker`, `KeelReferenceCheck` |
| `kernelports.DBConnection`'s `DB`/`DBForContext`/`WithTx`, `dbtest.NewSQLMock` | `kernelports.Transactor`, `Keel(ctx)` |
| `RelationshipDefintion` fields other than `Field` and `Queryable` | keel relation tags |

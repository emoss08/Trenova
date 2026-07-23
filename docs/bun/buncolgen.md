# buncolgen — Type-Safe Column Helpers for Bun ORM

`services/tms/pkg/buncolgen/` contains **generated** type-safe helpers for every domain entity: column references, table metadata, relation names, tenant scoping, filter builders, and field maps. **All repository code must use these helpers instead of hand-written column strings.** Raw string literals like `"wrk.status = ?"` or `.Relation("PrimaryWorker")` are forbidden in new code.

## Why

- **Compile-time safety**: renaming a column in the domain model breaks the build at every call site instead of failing at runtime.
- **Zero allocation**: every SQL fragment (`Eq()`, `OrderDesc()`, etc.) is pre-computed at package init; method calls are plain field returns.
- **Consistency**: every table has exactly one alias (from its `bun:"table:...,alias:..."` tag) and every fragment bakes it in — no alias drift between queries.
- **Injection safety**: `NewColumn` panics on any identifier outside `[a-zA-Z0-9_]`, so generated fragments can never carry injected SQL.

## Regenerating

The generator lives at `internal/infrastructure/database/colgen/gen` and scans `internal/core/domain/`:

```bash
cd services/tms && task generate-columns
# runs: go generate ./internal/infrastructure/database/colgen/...
```

Run this after **any** domain model change (new entity, added/renamed/removed field, changed bun tag). Never hand-edit `*_gen.go` files — they are overwritten. If the entity feeds GraphQL projections, regenerate buncolgen **first**, then the projection specs (they consume the generated field maps).

The `email` domain package is intentionally excluded (`excludedPackages` in `gen/main.go`) due to unqualified model-name collisions.

## Generated Artifacts (per entity `Worker` → `worker_gen.go`)

| Artifact | Type | Purpose |
|---|---|---|
| `WorkerTable` | `TableInfo` | Table name, alias, composite PK |
| `WorkerColumns` | struct of `Column` | Every column, alias pre-bound |
| `WorkerFieldMap` | `map[string]string` | JSON field name → DB column (used by QueryBuilder) |
| `WorkerInsertableColumns` | `[]string` | Columns valid for INSERT (excludes `scanonly` computed columns) |
| `WorkerRelations` | struct of `string` | Bun relation names for `.Relation()` |
| `WorkerScopeTenant` / `...Update` / `...Delete` | funcs | Tenant `WHERE` clauses per query type |
| `WorkerApplyTenant` | func | Tenant scoping closure for `.Apply()` |
| `WorkerFilter` | struct of funcs | `domaintypes.FieldFilter` builders with JSON names baked in |

## Column Method Reference

A `Column` knows its bare name and its alias-qualified form. Pick the method by context:

| Method | Returns | Use in |
|---|---|---|
| `String()` / `Bare()` | `first_name` | `q.Column(...)`, `Columns: []string{...}` — Bun qualifies it |
| `Qualified()` | `wrk.first_name` | Raw SQL where Bun won't add the alias |
| `Eq()` `Ne()` `NotEq()` `Gt()` `Gte()` `Lt()` `Lte()` | `wrk.col <op> ?` | `q.Where(...)` with one bind arg |
| `In()` / `NotIn()` | `wrk.col IN (?)` | `q.Where(col.In(), bun.List(values))` |
| `IsNull()` `IsNotNull()` `IsTrue()` `IsFalse()` | no-bind predicates | `q.Where(...)` |
| `Like()` `ILike()` `NotLike()` `NotILike()` `LowerLike()` `TextILike()` | pattern predicates | `q.Where(..., "%"+term+"%")` |
| `Between()` | `wrk.col BETWEEN ? AND ?` | `q.Where(col.Between(), lo, hi)` |
| `OrderAsc()` / `OrderDesc()` | `wrk.col ASC` | `q.Order(...)` |
| `As(label)` | `wrk.col AS label` | `q.ColumnExpr(...)` |
| `Set()` | `col = ?` | `q.Set(...)` — SET uses bare names |
| `SetNull()` / `SetExcluded()` | `col = NULL` / `col = EXCLUDED.col` | UPDATE / upsert `ON CONFLICT DO UPDATE` |
| `SetExpr(tpl)` / `Inc(n)` / `Dec(n)` | `col = <expr>` | self-referential updates, counters, version bumps |
| `Expr(tpl)` | template with `{}` → qualified name | `COALESCE`, `LOWER`, `BTRIM` wrappers |
| `EqColumn(other)` | `a.col = b.col` | joins, correlated subqueries |
| `WithAlias(a)` | same column, new alias | self-joins |

Package-level helpers: `Expr(tpl, cols...)` (multi-column templates with `{0}`, `{1}`), `Rel(segments...)` (dot-joined relation paths), and aggregates `Count`, `CountDistinct`, `CountFilter`, `Sum`, `Min`, `Max`, `Coalesce`.

## Repository Patterns

These are the canonical patterns — follow them exactly. Real examples: `tractorrepository/tractor.go`, `workerrepository/worker.go`, `shipmentrepository/shipment.go`.

### Local shorthands

When a function uses several helpers from the same entity, bind them once:

```go
cols := buncolgen.TractorColumns
rel := buncolgen.TractorRelations
```

### Get by ID (tenant-scoped)

```go
entity := new(tractor.Tractor)
err := dba.NewSelect().
    Model(entity).
    WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
        return buncolgen.TractorScopeTenant(sq, req.TenantInfo).
            Where(buncolgen.TractorColumns.ID.Eq(), req.ID)
    }).
    Scan(ctx)
```

### List by IDs

```go
Where(buncolgen.TractorColumns.ID.In(), bun.List(req.TractorIDs))
```

### Tenant scoping via Apply

```go
q.Apply(buncolgen.TractorApplyTenant(req.TenantInfo))
```

### Bulk update with tenant scope

```go
_, err := dba.NewUpdate().
    Model((*tractor.Tractor)(nil)).
    WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
        return buncolgen.TractorScopeTenantUpdate(uq, req.TenantInfo).
            Where(buncolgen.TractorColumns.ID.In(), bun.List(req.TractorIDs))
    }).
    Set(buncolgen.TractorColumns.Status.Set(), req.Status).
    Exec(ctx)
```

### Optimistic locking

```go
ov := entity.Version
entity.Version++
results, err := dba.NewUpdate().
    Model(entity).
    WherePK().
    Where(buncolgen.TractorColumns.Version.Eq(), ov).
    OmitZero().
    Returning("*").
    Exec(ctx)
```

### Column projection (all vs. selected)

```go
func applyTractorColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
    if len(columns) == 0 {
        return q.ColumnExpr(buncolgen.TractorTable.All()) // "tr.*"
    }
    return q.Column(columns...)
}
```

### Relations (never string literals)

```go
q.Relation(rel.PrimaryWorker)
q.Relation(buncolgen.Rel(rel.PrimaryWorker, buncolgen.WorkerRelations.FleetCode))
// → "PrimaryWorker.FleetCode"
```

### Ordering

```go
q.Order(buncolgen.TractorColumns.CreatedAt.OrderDesc())
```

### QueryBuilder filters (list endpoints)

Pass the generated alias, never a hardcoded string:

```go
querybuilder.ApplyFilters(q, buncolgen.TractorTable.Alias, req.Filter, (*tractor.Tractor)(nil))
querybuilder.ApplyCursorFilters(q, buncolgen.TractorTable.Alias, req.Filter, req.Cursor, (*tractor.Tractor)(nil))
```

Programmatic filters use the generated builders so JSON names stay in sync:

```go
req.Filter.FieldFilters = append(req.Filter.FieldFilters,
    buncolgen.WorkerFilter.Status(dbtype.OpEq, domaintypes.StatusActive))
```

### SelectOptions (typed config)

Use the `*Ref` fields with `buncolgen.Column` values — not the legacy string fields:

```go
dbhelper.SelectOptions[*tractor.Tractor](ctx, r.db.DB(), req.SelectOptionsRequest,
    &dbhelper.SelectOptionsConfig{
        ColumnRefs:       []buncolgen.Column{cols.ID, cols.Status, cols.Code},
        OrgColumnRef:     &cols.OrganizationID,
        BuColumnRef:      &cols.BusinessUnitID,
        SearchColumnRefs: []buncolgen.Column{cols.Code},
        EntityName:       "Tractor",
        QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
            return q.Where(cols.Status.Eq(), domaintypes.EquipmentStatusAvailable)
        },
    })
```

### Aggregates

```go
q.ColumnExpr(buncolgen.Count("total"))
q.ColumnExpr(buncolgen.CountFilter("active", cols.Status.Eq()), domaintypes.StatusActive)
q.ColumnExpr(buncolgen.Sum(buncolgen.ShipmentColumns.Weight, "total_weight"))
```

### SQL expression templates

```go
cols.ExternalID.Expr("NULLIF(BTRIM({}), '') IS NOT NULL")
buncolgen.Expr("CONCAT({0}, ' ', {1})", cols.FirstName, cols.LastName)
```

### Upserts

```go
q.On("CONFLICT (id) DO UPDATE").
    Set(buncolgen.DocumentContentColumns.Status.SetExcluded())
```

## Rules

1. **Never** write a raw column string (`"wrk.status"`, `"status = ?"`) in repository code — there is a helper for every case; if one seems missing, check the method table above first.
2. **Never** hand-edit `pkg/buncolgen/*_gen.go` — change the domain model and run `task generate-columns`.
3. **Never** hardcode a table alias — use `XTable.Alias` / `XTable.All()`.
4. **Never** pass a string literal to `.Relation()` — use `XRelations` and `buncolgen.Rel()` for nested paths.
5. **Always** tenant-scope queries with the generated `XScopeTenant*` / `XApplyTenant` helpers, never hand-written org/BU `WHERE` clauses.
6. **Always** use `bun.List()` with `In()` / `NotIn()`.
7. New entity? Add the domain model with proper `bun` tags (`table:...,alias:...`), run `task generate-columns`, then write the repository against the generated helpers.

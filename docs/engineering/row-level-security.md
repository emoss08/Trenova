# Row-level security

How PostgreSQL itself keeps one organization's rows away from another, what the
application has to do for that to hold, and what to do when you add a table, a
repository method, a background job or a code path that genuinely has to see
every tenant.

Tenant filters in Go (`WHERE organization_id = ?`) are still written everywhere.
Row-level security is the second, independent wall behind them: a missed filter,
a request field that overrode the session's tenant, or an injected SQL string can
no longer return or change another organization's rows, because the database
refuses them.

## The shape of it

```
request / activity ──▶ dbscope.WithTenant(ctx, org, bu, user)
                               │
repository method ──▶ dbtx.Read / dbtx.Write (one transaction per method)
                               │
scoped driver ──▶ BEGIN [READ ONLY]; SET LOCAL trenova.scope = 'v1.<key>.<org>.<bu>.<user>.<exp>.<hmac>'
                               │
PostgreSQL ──▶ policy: organization_id = (SELECT trenova_rls.org_id())
                       AND business_unit_id = (SELECT trenova_rls.bu_id())
                               │
               trenova_rls.claims() verifies the HMAC and expiry, or raises 42501
```

| Piece | Where |
|---|---|
| Scope on a context (`WithTenant`, `WithSystem`, `TenantOf`) | `pkg/dbscope` |
| Per-method transaction helpers | `internal/infrastructure/postgres/dbtx` |
| Signing, driver wrapper, system pool, startup checks | `internal/infrastructure/postgres/{rlsscope,scopeddriver,connection_rls}.go` |
| Key install, policy reconcile, role provisioning | `internal/infrastructure/postgres/rlsprovision.go`, `trenova db rls` |
| Roles, scope schema and every policy | migration `20261231007040_row_level_security` |
| HTTP binding | `pkg/authctx` (`SetSessionAuthContext`, `SetAPIKeyContext`, `SetCaptureDeviceContext`) |
| Temporal binding | `internal/core/temporaljobs/interceptors/tenantscope.go` |
| Guards | `schemalint/rls_integration_test.go`, `systemscopelint`, `postgres/rls_integration_test.go` |

## Roles

| Role | Login | Bound by policies | Used for |
|---|---|---|---|
| `trenova_tenant` | no | yes | group; the application role is a member |
| `trenova_rls_bypass` | no | no (`USING (true)`) | group; the system and migrator roles are members |
| `database.user` | yes | yes | API, worker, reporting pool |
| `database.system.user` | yes | no | only contexts declared with `dbscope.WithSystem` |
| `database.migrator.user` | yes | no | migrations, seeds, key install; owns the tables |

The bypass is a role, never a setting a session can change. Every table is
`FORCE ROW LEVEL SECURITY`, so even the owner is held to the policies unless it is
a bypass member. Startup in `enforce` mode refuses to run when the application
role is a superuser, has `BYPASSRLS`, is a bypass member, owns a table, or is not
a tenant member, and when the configured scope key is not installed.

## The signed scope

The scope is not a plain organization id. It is
`v1.<keyId>.<org>.<bu>.<user|->.<expiresUnix>.<hex hmac-sha256>`, signed with
`database.rls.scopeKey` and checked by `trenova_rls.claims()`, a
`SECURITY DEFINER` function that alone can read `trenova_rls.scope_keys`. A SQL
injection that runs `SET trenova.scope = …` cannot name another tenant without
the key. The key is installed as HMAC pads, never in the clear, by
`trenova db migrate`; rotate it by configuring a new `scopeKeyId`, deploying, and
`trenova db rls retire-key <old>` once nothing signs with the old one.

`SET LOCAL` keeps the scope to the transaction, which is what PgBouncer's
transaction pooling needs: the next transaction on the same server connection
starts with no scope and is refused.

## Modes

| `database.rls.mode` | Driver | When |
|---|---|---|
| `off` (default) | not installed; `dbtx` helpers call straight through | SQLite, and deployments not yet migrated |
| `observe` | signs scopes, logs and meters access that `enforce` would refuse | while rolling out |
| `enforce` | refuses unscoped statements and statements outside a scoped transaction | production |

`trenova_db_rls_scope_events_total{pool,event,outcome}` counts every decision.
In `observe`, the first sighting of each unscoped call site is logged at warn with
its caller; refusals are always logged at error.

## Writing code

**Repository methods** run their body in `dbtx.Read` (read-only transaction) or
`dbtx.Write`, which reuse the caller's transaction when it has the same scope.
Inside, use `r.db.DBForContext(ctx)` as before; never `r.db.DB()`, which is the raw
pool and runs outside the transaction.

```go
func (r *repository) GetByID(ctx context.Context, req GetRequest) (*thing.Thing, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*thing.Thing, error) {
		entity := new(thing.Thing)
		err := r.db.DBForContext(ctx).NewSelect().Model(entity).
			Where(cols.ID.Eq(), req.ID).
			Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Thing")
		}
		return entity, nil
	})
}
```

- A statement that is allowed to fail inside a transaction (insert, then read the
  winner on a unique violation) goes in `dbtx.Savepoint`; otherwise the failure
  aborts the transaction and the read after it fails too.
- Work that must commit on its own regardless of the caller's transaction
  (sequence allocation) uses `Connection.RunDetached`.
- Validators use `validationframework.NewBunUniquenessCheckerScoped(conn)` /
  `NewBunReferenceCheckerScoped(conn)` / `NewUSStateReferenceCheck(conn)`.

**A new tenant table** needs nothing when it has non-null `organization_id` and
`business_unit_id`: end the migration with `SELECT trenova_rls.reconcile();`, and
`trenova db migrate` reconciles again afterwards. Any other shape (nullable
tenant, two parties, owned through a parent) calls
`trenova_rls.apply_policy('public.t', '<read expr>', '<write expr>')` with
expressions that consult `trenova_rls.org_id()` / `bu_id()` / `user_id()`. A table
with no tenant data is added to `trenova_rls.global_tables` with its reason.
`schemalint` fails on a table with neither, on a tenant policy that does not read
the scope, on a view that is not `security_invoker`, and on a new
`SECURITY DEFINER` function.

**A request or job** gets its scope from the session, API key or capture device
(HTTP and GraphQL) or from its input (Temporal: a `TenantInfo`, an
organization/business unit pair, or a type implementing `dbscope.TenantScoped`).
Code that starts work any other way binds `dbscope.WithTenant` itself.

**Seeing every tenant** is `ctx = dbscope.WithSystem(ctx, reason)` in the one
method that needs it, before the `dbtx` call, with a constant reason. The
statement then runs on the system pool. `systemscopelint` fails on any call site
not listed, with a justification, in its `allowlist_test.go`. Prefer resolving
the tenant first (a share token, a webhook's account) and continuing under
`WithTenant`, so the system scope covers one lookup, not the request.

## Rolling out

1. `trenova db rls generate-key`; store the key; set `database.rls` with
   `mode: observe` and `database.migrator` to the role that owns the tables.
2. `trenova db migrate` (installs the key and reconciles policies and grants).
3. `trenova db rls provision-roles` creates the application and system login
   roles from `database.user` and `database.system`.
4. Point the application at the new `database.user`, run in `observe` until
   `rls_scope_events_total{outcome="observed"}` stays at zero, then `enforce`.
5. GTC reads every tenant: give its role `trenova_rls_bypass` (or run it as the
   system role); logical decoding ignores policies but its snapshot does not.

Rolling back is setting `mode: off` (no driver, no per-method transactions) or
pointing the application back at the owner role.

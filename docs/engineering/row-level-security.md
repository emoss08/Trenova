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
with no tenant data is added to `trenova_rls.global_tables` with its reason. A table
the application may only read and append to (an audit log) is added to
`trenova_rls.append_only_tables`, and `reconcile()` keeps `UPDATE` and `DELETE`
revoked from `trenova_tenant`; see [security-audit.md](security-audit.md).
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

## Getting the scope right

A missing scope fails closed: the statement is refused. A scope that is present
but names the wrong tenant fails *open* for any question of the form "is there a
row that forbids this?", because the row is simply invisible. Sign-in checks SSO
enforcement under the organization being signed into, not the user's current
one, for exactly this reason. When code asks whether something exists in order to
deny, make sure the scope is the tenant the question is about.

- **One lookup, then the tenant.** Public links, webhooks, AS2, capture device
  tokens and password resets resolve their token under the system scope and then
  continue under the token's tenant. Keep the system scope to that one lookup.
- **Sweeps.** A Temporal sweep lists work across tenants under the system scope
  (inside the listing repository method) and binds each item's tenant before
  working on it: `dbscope.WithTenant(ctx, tenantInfo.DBTenant())`.
  `pagination.TenantInfo` implements `dbscope.TenantScoped`, so an activity that
  takes one is bound by the interceptor. The same applies to a second pass over
  what the sweep collected (a map of workers to refresh, drivers to raise,
  digests to send): bind each entry's tenant before calling into a service.
- **Projectors and publishers.** Code that writes on behalf of a record it was
  handed (the watchtower projector, the agent event publisher, the AI audit
  projector's per-tenant pass) binds that record's tenant itself, so a caller
  that holds only a system scope or none cannot make it fail or write elsewhere.
- **Two organizations, one transaction.** An internal EDI approval or transfer
  change writes both organizations atomically. Those transactions run under the
  system scope, with the tenant filters still written in Go; a read of the other
  organization alone binds that organization's tenant instead.
- **Business-unit-wide reads.** Organization pickers, the organization loader and
  organization uniqueness checks span the business unit; they run under the system
  scope and filter by business unit themselves. The `organizations` policy only
  shows the scope's own organization, ones the user belongs to and EDI
  counterparties.
- **Transactions across scopes.** A transaction is reused only by a context with
  the same scope (any two system scopes count as the same). Switching tenant
  inside a transaction opens a second transaction on another connection, so do not
  do it for writes that must commit together.

## Verifying a deployment

`trenova db rls status` lists tables without forced RLS. Before enforcing, run in
`observe` against a copy of production, exercise sign-in, the main pages, the
public links and webhooks, and let the workers run their schedules; every
unscoped access is logged once per call site with its two calling frames. Then
switch to `enforce` and watch for `Refused database access` errors.

## Rolling out

`observe` is a dry run only while the application still connects as a role that
bypasses row-level security. The driver signs every scope and reports the
access `enforce` would refuse, but PostgreSQL lets it through. Connected as the
tenant-bound application role, PostgreSQL itself refuses an unscoped statement
on a tenant table (`no tenant scope is set for this transaction`) whatever the
mode, so switch roles and enforcement together.

1. `trenova db rls generate-key`; store the key; set `database.rls.scopeKeyId`,
   `database.rls.scopeKey` and `mode: observe`, and set `database.migrator` to
   the role that owns the tables. Leave `database.user` as it is.
2. `trenova db migrate` installs the key and reconciles policies and grants.
   Every role that already held table privileges, the current application role
   included, becomes a member of `trenova_rls_bypass`, so nothing is refused yet.
3. Create the login roles. `provision-roles` reads the application role from
   `database.user`, so run it with the new role's name and password in the
   environment:
   `TRENOVA_DATABASE_USER=trenova_app TRENOVA_DATABASE_PASSWORD=<app> trenova db rls provision-roles`
   (with `database.system` set to the system role).
4. Deploy in `observe`, still connected as the existing role, and exercise
   sign-in, the main pages, public links, webhooks and a full day of schedules.
   Each unscoped call site is logged once at warn with its two calling frames.
   Fix every one until `trenova_db_rls_scope_events_total{outcome="observed"}`
   stays at zero.
5. Switch `database.user` to the application role and `mode` to `enforce` in the
   same deploy. Startup refuses `enforce` while the application role can bypass
   row-level security, owns a table, or the key is not installed.
6. GTC reads every tenant: give its role `trenova_rls_bypass` (or run it as the
   system role); logical decoding ignores policies but its snapshot does not.

Rolling back is setting `mode: off` (no driver, no per-method transactions) or
pointing the application back at the owner role.

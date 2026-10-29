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
repository method ──▶ dbtx.Read / dbtx.Write (one keel transaction per method, joined by nested calls of the same scope)
                               │
scope guard ──▶ keel (Config.OnBegin): BEGIN [READ ONLY]; SELECT set_config('trenova.scope', 'v1.<key>.<org>.<bu>.<user>.<exp>.<hmac>', true)
                               │
PostgreSQL ──▶ policy: (organization_id, business_unit_id)
                       = (SELECT t.org_id, t.bu_id FROM trenova_rls.tenant() AS t)
                               │
               trenova_rls.claims() verifies the HMAC and expiry, or raises 42501
```

| Piece | Where |
|---|---|
| Scope on a context (`WithTenant`, `WithSystem`, `TenantOf`) | `pkg/dbscope` |
| Per-method transaction helpers | `internal/infrastructure/postgres/dbtx` |
| Signing, the scope guard, system pool, startup checks | `internal/infrastructure/postgres/{rlsscope,scopeguard,connection_rls}.go` |
| pgx pools and keel's configuration | `internal/infrastructure/postgres/dbpool` |
| Key install, policy reconcile, role provisioning | `internal/infrastructure/postgres/rlsprovision.go`, `trenova db rls`, the `rls_scope_key` reference seed (`internal/infrastructure/database/dbcli/scopekey.go`) |
| Policy reconcile after every migration run | `migrations.Options` (`schema/migrations`): `trenova-db` runs `SELECT trenova_rls.reconcile()` once a run applied anything |
| Roles, scope schema and every policy | migration `20261231007040_row_level_security` |
| HTTP binding | `pkg/authctx` (`SetSessionAuthContext`, `SetAPIKeyContext`, `SetCaptureDeviceContext`) |
| Temporal binding | `internal/infrastructure/temporaljobs/interceptors/tenantscope.go` |
| Guards | `schemalint/rls_integration_test.go`, `rlslint` (`TestEverySystemScopeIsReviewed`; repository methods run in a scoped transaction), `postgres/rls_integration_test.go` |

## Roles

| Role | Login | Bound by policies | Used for |
|---|---|---|---|
| `trenova_tenant` | no | yes | group; the application role is a member |
| `trenova_rls_bypass` | no | no (`USING (true)`) | group; the system and migrator roles are members |
| `database.user` | yes | yes | API, worker, reporting pool |
| `database.system.user` | yes | no | only contexts declared with `dbscope.WithSystem` |
| `database.migrator.user` | yes | no | `trenova-db`, seeds, key install; owns the tables |

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
the key. The key is installed as HMAC pads, never in the clear, by the `rls_scope_key`
reference seed that `trenova-db up` applies whenever the configured key changes (its
fingerprint hashes the key id and key); rotate it by configuring a new `scopeKeyId`, deploying, and
`trenova db rls retire-key <old>` once nothing signs with the old one.

`SET LOCAL` (and keel's `set_config(…, true)`, which is the same thing) keeps the
scope to the transaction, which is what PgBouncer's transaction pooling needs: the
next transaction on the same server connection starts with no scope and is
refused.

One `scopeGuard` per pool decides every transaction's scope: keel calls it through
`Config.OnBegin` for every transaction it begins, explicit (`dbtx`, `Transact`) or
implicit (a terminal outside a transaction, which keel runs in one whenever `OnBegin`
is set). A nested `dbtx` call with the same scope joins the open transaction and
sends no `OnBegin` statements; one with another scope begins its own. See
[backend/data-access.md](backend/data-access.md#pools-and-transactions).

## Modes

| `database.rls.mode` | Scope guard | When |
|---|---|---|
| `off` (default) | not installed; keel has no `OnBegin`, and `dbtx` still runs each method in one transaction | deployments not yet migrated |
| `observe` | signs scopes, logs and meters access that `enforce` would refuse | while rolling out |
| `enforce` | `OnBegin` refuses a transaction with no tenant or system scope | production |

`trenova_db_rls_scope_events_total{pool,event,outcome}` counts every decision.
In `observe`, the first sighting of each unscoped call site is logged at warn with
its caller; refusals are always logged at error.

## Writing code

**Repository methods** run their body in `dbtx.Read` (read-only transaction) or
`dbtx.Write`, which reuse the caller's transaction when it has the same scope.
Inside, pass `r.db.Keel(ctx)` to every terminal; it joins the transaction on the
context.

```go
func (r *repository) GetByID(ctx context.Context, req GetRequest) (*thing.Thing, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*thing.Thing, error) {
		entity, err := thingdb.Things.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, thingErrors.Map(err, req.ID.String())
		}
		return entity, nil
	})
}
```

- The scope belongs on the exported method, around everything it reads. A method
  whose unexported helpers run the queries is still an unscoped method: the rate
  engine's `ResolveRules` opened a transaction only in the last of its three helpers,
  and every quote from a caller without one of its own was refused.
  `rlslint.TestRepositoryMethodsRunInScopedTransactions` follows calls to the
  receiver's unexported helpers, transitively, and fails an exported method that
  reaches one touching the connection outside `dbtx`. A closure handed to another
  method is that method's to scope, so take the `ctx` the closure is given rather
  than the one around it.
- A statement that is allowed to fail inside a transaction (insert, then read the
  winner on a unique violation) goes in `dbtx.Savepoint`; otherwise the failure
  aborts the transaction and the read after it fails too.
- Work that must commit on its own regardless of the caller's transaction
  (sequence allocation) uses `Connection.RunDetached`.
- Validators use `keelcheck.NewChecker(conn, xdb.XsSchema())`, which
  runs its checks through `Keel(ctx)` in the caller's transaction.
- A keel query under a system scope must say
  `AllTenants()`, and keel fills and checks the tenant columns of every write from
  the scope, a second wall in Go beside the policies.

**A new tenant table** needs nothing when it has non-null `organization_id` and
`business_unit_id`: `trenova-db` runs `SELECT trenova_rls.reconcile()` after every run that
applied a migration, so the migration does not call it (`task db-lint` warns when one does).
Any other shape (nullable tenant, two parties, owned through a parent) calls
`trenova_rls.apply_policy('public.t', '<read expr>', '<write expr>')` with
expressions that consult `trenova_rls.tenant()`, `org_id()`, `bu_id()` or `user_id()`. Each
`(SELECT …)` over one of them is an InitPlan that verifies the signed scope once per table per
statement, about 60 µs each: compare a row with `(SELECT t.org_id, t.bu_id FROM
trenova_rls.tenant() AS t)` rather than writing two subqueries, as `standard_expr()` does. A table
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
Code that starts work any other way binds `dbscope.WithTenant` itself. A dynamic
activity is one such way: the interceptor sees only its opaque encoded arguments, so the
agent tool activity (`agentflow.runTool`) decodes its input and binds the run's tenant
before it dispatches, and refuses a call that names none. Work that outlives its request on
a detached context (a usage row written in the background, a decision follow-up) binds its
own tenant with `dbscope.WithValidTenant` rather than trusting the caller's scope.
A `RequestActor` is itself `TenantScoped`: an agent principal's scope names no user, even
though an unattended run's actor carries the system user as its `UserID` for attribution,
so `trenova_rls.user_id()` never widens a run to the system user's own rows.

**Seeing every tenant** is `ctx = dbscope.WithSystem(ctx, reason)` in the one
method that needs it, before the `dbtx` call, with a constant reason. The
statement then runs on the system pool. `rlslint`'s `TestEverySystemScopeIsReviewed` fails on any call site
not listed, with a justification, in its `allowlist_test.go`. Prefer resolving
the tenant first (a share token, a webhook's account) and continuing under
`WithTenant`, so the system scope covers one lookup, not the request.

**A role's data scope** rides beside the tenant scope. A request or agent tool call
that holds a `pkg/readscope` holder sends `SET LOCAL trenova.read_scope` with its
tenant scope, and restrictive `read_scope` policies on `shipments`, `orders`,
`invoices` and `workers` read it through `trenova_rls.read_scope(kind)` and
`trenova_rls.read_scope_ids(kind)`. It is unsigned because it can only narrow a read
below what the signed scope allows; `apply_policy` and `reconcile()` leave the
`read_scope` policies alone. It carries only `own` and `team` grants: a kind the
principal cannot read is left out, so its rows stay visible to the relations, loaders
and joins that reach them from a readable record, and the application decides what of
them shows (the kind's reference view) and refuses a root read of the kind before it
queries. See [permissions.md](permissions.md#data-scopes).

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
- **Two organizations, one transaction.** A transfer change review writes both
  organizations' linked shipments atomically, under the system scope, with the
  tenant filters still written in Go; a read of the other organization alone
  binds that organization's tenant instead. A write whose only other-tenant rows
  are rows both parties' policies admit (an EDI transfer, shipment link or tender
  recipient) runs under the tenant that owns the rest: rejecting or expiring an
  internal tender runs under the source organization's scope (`ediservice.transferSourceContext`),
  since it changes the source shipment. Work that must create rows in one
  organization through another module's service and change rows of the other
  cannot share a transaction (a system scope would need every repository it
  reaches to be system-aware): the load tender approval activity commits the
  receiving side under the target's scope, then accepts the source shipment's
  tender under the source's scope in a second transaction that skips what is
  already done, so a retried activity converges
  (`ediservice.ProcessLoadTenderApproval`). Replacing a user's
  organization memberships is the same shape: the user repository lists and rewrites
  them under the system scope, filtered to that user and business unit, and the user
  service first checks the caller holds `user:update` in every organization it adds or
  removes, judged by that organization's own roles (`dbscope.WithTenant` for it, and
  `authctx.WithoutSessionRoleActivation`, since the session's activated roles belong to
  the caller's current organization).
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
bypasses row-level security. The scope guard signs every scope and reports the
access `enforce` would refuse, but PostgreSQL lets it through. Connected as the
tenant-bound application role, PostgreSQL itself refuses an unscoped statement
on a tenant table (`no tenant scope is set for this transaction`) whatever the
mode, so switch roles and enforcement together.

1. `trenova db rls generate-key`; store the key; set `database.rls.scopeKeyId`,
   `database.rls.scopeKey` and `mode: observe`, and set `database.migrator` to
   the role that owns the tables. Leave `database.user` as it is.
2. `trenova-db up` reconciles policies and grants, and its `rls_scope_key` seed installs the key.
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

Rolling back takes both changes in the same deploy: point `database.user` back at
the owner role and set `mode` to `off` (no scope guard) or
`observe`. Neither is enough alone: the tenant-bound role with `mode: off` is
refused every unscoped statement by PostgreSQL, and the owner role under
`enforce` is refused at startup.

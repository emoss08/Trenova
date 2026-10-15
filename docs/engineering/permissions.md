# Permissions

How Trenova decides what a person, an API key or an agent may do, how a grant reaches the
people who hold it, and what to keep true when you add a resource, an operation, a route
or a way of changing access.

## The shape of it

```
role ──▶ resource_permissions (resource, operations[], data_scope)
  ▲            │
  │  inherits  │ union over every role a person holds, and every role those inherit
  │            ▼
user_role_assignments ──▶ permission engine ──▶ Check(resource, operation) ──▶ allow / deny
                              │       ▲
                    per-user cache    organization permission epoch
                    (Redis, hashed    (bumped on every change; an entry from an
                     by variant)       earlier epoch is never served)
```

| Piece | Where |
|---|---|
| Registry: every resource, its operations and field sensitivities | `internal/modules/identity/domain/permission/resources_*.go` |
| Engine (one port, used by REST, GraphQL, services and agent tools) | `internal/modules/identity/app/permission/engine.go` |
| Delegation check ("cannot grant what you do not hold") | `permissionports.EffectivePermissions.DelegationViolation` |
| Change notifier (cache epoch + realtime) | `internal/modules/identity/app/permissionchange` |
| Administrator role reconcile | `internal/infrastructure/postgres/tenantbootstrap/adminreconcile.go` |
| Declared-permission lints (REST, GraphQL, web client, route registry, agent tools) | `internal/api/permissionlint`, `internal/api/graphql/authzlint`, `permission/routeregistry_test.go`, `agenttoolpolicy/validate.go` |

## Adding a resource or an operation

1. Declare it in the registry (`resources_*.go`). A check that names an operation the
   registry does not declare can never pass, because no role can be granted it, so
   every place that names a permission is held to `Registry.Declares`:
   - REST handlers, GraphQL resolvers and middleware
     (`permissionlint.TestEveryCheckedPermissionIsDeclared`). A check whose resource is
     chosen at run time (a bulk edit, a table export) cannot be read statically and must
     be listed, with its reason, in `dynamicChecks`
     (`TestEveryPermissionCheckNamesItsPermission`); a resource held in a local variable
     assigned from a `permission.Resource…` constant is read.
   - The web and driver-portal clients: every `Resource.X, Operation.Y` gate
     (`TestEveryClientPermissionGateIsDeclared`).
   - Page access in the route registry (`TestRouteRegistry_EveryRequirementIsDeclared`).
   - Agent tool policies (`agenttoolpolicy.Validate`), for every registered tool.

   A resource nothing checks is not declared: it only makes the role editor longer and a
   grant of it allows nothing. Retire one with a migration that deletes its
   `resource_permissions` and `api_key_permissions` rows.
2. Do not write a migration to grant it to administrators. `trenova db migrate` runs
   `ReconcileAdminRoles` after the migrations: every system role named
   "Organization Administrator" is given every registry resource and operation it lacks.
   Grants beyond the registry are left alone for a person to review, and other system
   roles (the driver portal role) are never touched.
3. Check it where the work happens: `RequirePermission` on the route or in the resolver.
   A new write route also needs an entry in `writecoverage.yml`.

## Bundles and job templates

A role is composed from **capability bundles** rather than from 180 resources one at a
time. A bundle (`permission/bundles_catalog.go`) is a named set of grants for one job to be
done, such as "Dispatch board" or "Post invoices", and a **job template** (Dispatcher,
Billing clerk, Accountant, Driver manager, Read-only and others) is a list of bundles with
a field sensitivity ceiling. `GET /permissions/bundles` returns both, each grant with the
scope it will actually apply in this deployment (a scope the deployment does not enforce
comes back as organization), and each template already composed (`Registry.TemplateGrants`:
operations merged, the widest scope kept).

Bundles are a way of choosing grants, not a second kind of grant: choosing one writes its
grants into the role's ordinary `resource_permissions`, so evaluation, delegation and the
audit trail are unchanged, and a later change to a bundle does not reach roles built from
it. The role editor's Capabilities tab ticks bundles (full, partly granted, none); its
Advanced tab is the resource grid, where each row says which ticked bundles it came from.
Taking a bundle away removes its operations except where another fully ticked bundle
grants them. `TestBundles_GrantOnlyWhatTheRegistryDeclares` keeps every bundle grant
declared and every bundle scope one its kind supports.

A role and all its grants are saved in one request and one transaction: `POST /roles/`
creates both, and `PUT /roles/:id` replaces the grant set with the role's columns.

## Data scopes

A grant's data scope narrows which records of a kind it reaches. Organization (and the
legacy business unit and all, which read the same) reaches the whole organization. A
narrower scope means something only where a kind declares it, in its registry
definition's `Scopes`:

| Kind | Scope | Enforced by |
|---|---|---|
| `shipment`, `order` | own: records whose `owner_id` is the person | database |
| `invoice` | own: invoices on a shipment or order the person owns (the header's or a line's) | database |
| `worker` | team: workers the person manages, directly or through a fleet code, or who were delegated to them | database |
| `capture_batch`, `report` | own | application (`captureservice`, the report compiler) |
| `worker_schedule`, `timesheet`, `profile_change_request` | team | application (`RequireTeamScope`, `teamscope.Guard`) |

**Database enforcement.** Each authenticated request (`ReadScopeMiddleware`) and each
agent tool call (`agentflow.runTool`) carries a `pkg/readscope` holder. At its first
query, `readscopeservice` checks the principal's read grant on every database-scoped
kind in one `CheckBatch` and, for a team grant, resolves the team through
`OrgStructureService.ManagerIDs`. The scoped driver then sends the result with the
tenant scope as `SET LOCAL trenova.read_scope = '<kind>:<scope>[:<ids>]'`, and a
restrictive `read_scope` policy on `shipments`, `orders`, `invoices` and `workers`
applies it to every read: lists, lookups, loaders, search, insights and reports alike.
The value only narrows what the signed tenant scope allows, so it is not signed; work
with no holder (jobs, unattended agents) reads its whole tenant. Policies are
`FOR SELECT`, so an update or a `RETURNING` on a row outside the scope finds nothing.

These policies apply only while row-level security is installed (`database.rls.mode`
`observe` or `enforce`). With RLS off, a database-enforced scope is not enforced, so the
server refuses to grant one and the editor does not offer it.

**Granting.** Role create and update, a permission row's create and update, and API key
create and update refuse a narrow scope the kind does not enforce in this deployment
(`permissionports.ScopePolicy`). A role edit that leaves an existing grant's scope as it
was is accepted, so a deployment that turns RLS off can still rename its roles; the
editor shows such a grant with a warning that it reaches the whole organization.
`GET /permissions/resources` sends each kind's `enforceableScopes`, and the editors offer
only those (`components/permissions/data-scope-select.tsx`).

Adding a scope to a kind: declare it in the registry; for the database, add the
`read_scope` policy in a migration and keep the migration's kind lists equal to the
registry (`migrations/read_scope_test.go` fails otherwise); for the application, enforce
it where the kind is read before declaring it.

## Implied operations

Some operations bring others with them (`permission.Dependencies`): approve, reject,
assign, archive and the other state changes bring update and read; import and duplicate
bring create; manage brings update and delete. A grant is stored exactly as it was made
and the implication is applied when it is evaluated (`permission.Implied`), in the
engine's cached set, its effective permissions, the delegation check, API keys and
`ListUsersWithPermission` (which matches any stored operation that implies the one asked
for, `permission.GrantingOperations`). Nothing widens a grant on write, so the audit
trail and the role editor show what a person chose.

The registry sends each operation's implications with it (`OperationDefinition.Implies`),
and the role editor marks an operation that a selected one brings as included rather than
leaving it looking ungranted. Rows written before this change were stored already widened
and read the same either way.

The client's operation and data scope lists (`PERMISSION_OPERATIONS`, `DATA_SCOPES` in
`@trenova/shared/types/generated/permission-resources`) are generated from the Go
constants with the resource enum; a hand-kept list once kept the role form from saving any
role holding manage, delete or close.

## Organization security settings

Single sign-on, SCIM, access policies and the sign-in activity views each have their own
resource rather than riding on `organization`: `identity_provider`, `scim_directory`
(directories, tokens and group mappings) and `access_policy` take read, create, update
and delete; `provisioning_audit`, `auth_event`, `risk_decision`, `external_identity` and
`mfa_authenticator` are read-only. The organization settings page shows each tab and
control only to someone who holds its permission
(`organization-settings/_components/security-access/use-security-access.ts`).

## Granting access

Every writer that grants access goes through the delegation check, which refuses any
resource, operation, data scope or sensitivity the person saving does not hold:

- role create and update, including permissions a role inherits through its parents;
- role assignment, including what the assigned role inherits;
- role inheritance edges (the junior role's closure is checked, and a system role can
  never be made to inherit);
- API key create and update.

A refused grant is recorded in the security audit trail ("Refused a grant beyond the
actor's own permissions") with what was refused. An API key is also held, at request
time, to what its creator holds now; see [security-audit.md](security-audit.md).

`PUT /roles/:id` writes only the editable columns (name, description, core responsibility,
max sensitivity) and replaces the role's permission set in the same transaction when the
request carries one. Inheritance is changed only through the hierarchy endpoints.

Deleting a role (`DELETE /roles/:id`, `role:delete`) refuses a system role and a role
someone holds, unless `force=true`, which also needs `user:unassign` and records an
unassignment for each holder. Roles that inherited the deleted role lose it.

## When a grant changes

Every change to who can do what calls `permissionports.PermissionChangeNotifier`:

- `OrganizationChanged` advances the organization's permission epoch, so every cached
  permission set computed before the change stops being served at once, and publishes
  `permissions`/`changed` to the tenant;
- `UserChanged` drops one person's cached set and publishes the event to them alone.

The browser refetches its permission manifest on the event. The thirty-minute cache TTL
and the five-minute client poll are only safety nets. Never delete permission keys by
pattern or by holder: holders through inheritance are invisible to such a delete, which is
the bug the epoch replaced.

## Membership and sessions

A role assignment means nothing without a membership in its organization. Removing a
membership revokes the person's roles there, the hourly
`orphaned-organization-access-sweep` revokes any that a failed removal left behind, and a
sign-in session is re-checked against membership and account status every five minutes.
The details are in [security-audit.md](security-audit.md#when-access-ends).

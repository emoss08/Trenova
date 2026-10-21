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
   Give it the name the app gives the feature and file it under the module it belongs
   to. The role editor's Advanced tab groups resources by `Category`, in the order of
   `categoryOrder` in `registry.go`, which follows the module rail (Desk, Triage, Intake,
   Shipments, Dispatch, … Organization settings, Users & security, Protected fields);
   a category not listed there fails `TestRegistry_EveryResourceSitsInACategoryTheRoleEditorOrders`.
   A module's own settings resource (Billing control, Dispatch control) sits in that
   module, not in Organization settings, so a role author finds everything that opens or
   hides a module in one place. Names, descriptions, categories and operation labels are
   translated by `GET /permissions/resources` and `GET /permissions/bundles` from the
   request's locale; the client renders them as they arrive.
2. Do not write a migration to grant it to administrators. `trenova db migrate` runs
   `ReconcileAdminRoles` after the migrations: every system role named
   "Organization Administrator" is given every registry resource and operation it lacks.
   Grants beyond the registry are left alone for a person to review, and other system
   roles (the driver portal role) are never touched.
3. Check it where the work happens: `RequirePermission` on the route or in the resolver.
   A new write route also needs an entry in `writecoverage.yml`.
4. Name the action. A route or mutation named for an action (`POST /:id/post/`,
   `SendInvoiceEDI`, `ReleaseBillingQueueItem`) must check that action's own operation
   (approve, cancel, archive…) or one whose display name or description on the resource
   names it, so a role author can find "post" or "send" in the editor
   (`permissionlint.TestEveryActionIsGatedByAPermissionThatNamesIt`). Posting an invoice is
   its own operation, `invoice:post`; sending one is "Submit and send". When an existing
   operation already carries the action, say so in its description rather than adding a
   verb no role holds.

## Saved table views

Anyone may keep private views of a table they can read. Sharing a view with the
organization needs `table_configuration` create, and changing one that stays shared needs
update; the owner may always withdraw or delete their own view
(`tableconfigurationservice`, which the `report_save_table_view` agent tool goes through too). A
reader only sees views, shared or private, of tables whose resource they can read, so a
view's name and filters never reach someone who cannot open the table it is about.
`reporting.reports` carries the share and change grants for new roles.

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

Splitting a resource that gated two features (working detention, `detention`, from
configuring it, `detention_policy`) moves every check, agent tool policy, record kind,
watchtower source and audit entry for the split-off work onto the new resource, and a
migration gives each role and API key the new resource with the operations it already
held through the old one (`20261231009780_detention_work_permission`), so nobody loses
access on upgrade.

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
with no holder (jobs, unattended agents) reads its whole tenant.

Only `own` and `team` grants are sent. A kind the principal cannot read at all is left
out of the setting, so the database shows its rows to whatever reaches them from a record
the person can read (a tractor's driver, a shipment's order, a document's shipment), and
the application shows each as its kind's reference view: the GraphQL field guard, the
REST response guard, search, reports and analytics all decide it through `refview`, so a
linked record reads the same with row-level security on or off. (Until
`20261231010180_read_scope_linked_rows` such a kind was sent as `<kind>:none`, and the
four policies hid every row of it but the person's own worker and the shipments a move
assigned them to drive, which made a linked driver or order come back null under RLS.)

A root read of a kind the person cannot read is refused in the application before it
queries, and CI holds every path to it:

- GraphQL: every root field that returns a shipment, order, invoice or worker, directly
  or through a connection, checks read (or an operation implying it) on the kind in its
  resolver (`TestEveryRootOfADatabaseScopedKindChecksItsRead`). One that such a mutation's
  payload carries rather than returns (a training record's worker) is the root record only
  for a caller who reads its kind or when the mutation declares `@authorizes`; otherwise
  the field guard shows it as linked, and `TestEveryLinkedFieldIsDeclared` holds client
  operations to that (`applyCreditMemo` and `unapplyCreditMemoApplication` check invoice
  read for it).
- REST: only the kind's own handler, whose every route requires an operation on the kind
  or marks its records `responseguard.References()`, writes one as a response's own
  record (`TestEveryRESTRootOfADatabaseScopedKindIsWrittenByItsOwnHandler`).
- Search: global search drops the index of a kind the person cannot read and a document
  attached to one before it asks `RecordVisibility`, which answers only the narrowing.
- Agents, reports, insights and analytics check the kind's read (the tool's policy, the
  report compiler, each detector's `Resources()`, the analytics page) before they read.

The driver portal and the `My…` self-service reads never relied on the database here:
they read the worker linked to the caller's user and the assignments naming that worker
in SQL (`driverportalrepository`), so a driver still sees only their own records.
Every write operation implies read, so a role that writes a kind reads it. Policies are
`FOR SELECT`, so an update or a `RETURNING` on a row outside the scope finds nothing.

A cache cannot apply them either. The synced Redis caches (`cache:shipments`,
`cache:workers`, `cache:customers`, `cache:documents`) hold the tenant's rows, so each one
asks `readscope.Narrows` for the kind it holds and answers only a request its kind is not
narrowed for (an own or team grant); a request whose scope could not be resolved, or one
the kind is narrowed for, reads through the database.

A search index cannot apply them: Meilisearch holds every record of the tenant. Global
search therefore passes each shipment, order, invoice and worker hit, and each document
attached to one, through `kernelports.RecordVisibility`, which asks the database which of
those ids the request may read under its read scope; a hit it does not return, or a check
that fails, is not shown, and a narrowed group reports only the hits it shows as its total.
A kind the person cannot read at all is refused by the service itself (its index is not
searched and the documents attached to it are hidden without asking), because the
database would return every one of its rows. Every index query
names the fields it returns (`SearchRequest.Attributes`, the id alone when none are named)
and builds its filter with `shared/searchfilter`, which quotes each value.

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

## Role activation

By default every role a person holds is in force from the moment they sign in, and a role
assigned during a session takes effect on its next request (the session carries no
activation, so the engine reads every unexpired assignment). An organization that turns on
**role activation** (`organization_access_settings.require_role_activation`, the Policies
tab, `GET`/`PUT /organizations/:id/iam/access-settings`, `access_policy` read/update) has
its people choose the roles they work under at sign-in and after switching organization;
only those roles count, which is what dynamic separation of duty checks.

Activation cannot be turned off while the organization keeps a dynamic separation of duty
constraint, and such a constraint cannot be saved while activation is off. The migration
that added the setting turned it on for every organization that already had one.

`AuthenticateSession` reads the setting through `iamports.RoleActivationPolicy`
(`accesssettingsservice`, cached for 30 seconds per instance and dropped at once on the
instance that saved a change) and stamps it on the session for the request; the auth
middleware installs the session's activation into the request context only when it is
required. A change advances the permission epoch and is recorded in the security audit
trail.

## Security briefing

The Security tab of organization settings opens on a briefing:
`GET /organizations/:id/iam/security-briefing` (`iamhandler.getSecurityBriefing`,
`securitybriefingservice`). The route needs only `organization` read, because the briefing is
part of the settings page; each section is then gated by the read permission of the records
it summarizes, checked in one `CheckBatch`, and a section the caller may not read is `null`
rather than an empty count:

| Section | Needs read on |
|---|---|
| `members` | `user` |
| `mfa` (counts) | `mfa_authenticator` |
| `mfa.unenrolled` (names and emails) | `mfa_authenticator` and `user` |
| `signIns` | `auth_event`; `signIns.riskDenied` also `risk_decision` |
| `apiKeys` | `api_key` |
| `sso` | `identity_provider` |
| `provisioning` | `scim_directory` |
| `policies` | `access_policy` |

`session` is the deployment's own `security.session` configuration (lifetime, idle sign-out
and its warning), not an organization setting, so it is always present. The findings — each a
code, a severity (`critical`, `warning`, `notice`, `pass`) and the counts behind it — and every
threshold (seven days of sign-ins, ninety days without a sign-in or key use, keys expiring
within fourteen days, ten failed sign-ins, half the members without two-factor) are worked out
on the server in `domain/iam/securitybriefing.go`; the client only words them.

Two-factor coverage is the one read under the system scope
(`securityBriefingRepository.MFACoverage`, listed in `rlslint`). A person enrols once, and the
`mfa_authenticators` row carries the organization they enrolled from; its policy shows a row
only to that organization or its own user, so under the tenant's scope a member who enrolled
from another organization would count as unenrolled. The method filters the members to the
tenant's organization and business unit in Go and reads only whether each has an active TOTP.

The tab's Activity tables read four cursor connections (`authEvents`, `riskDecisions`,
`externalIdentities`, `mfaAuthenticators` in `security_activity.graphqls`), each gated by read
on its own resource (`auth_event`, `risk_decision`, `external_identity`, `mfa_authenticator`),
so a person sees only the views they hold. Unlike the briefing's coverage count, the
two-factor device table stays under the tenant's scope: it lists the devices enrolled from this
organization, which is what an administrator here can act on. None of the four exposes a secret
(raw claims, the TOTP secret or a credential id).

## Reference views

A record that links to another (a shipment's customer, a tractor's drivers) shows the linked
record's **reference view** to everyone who can read the record that links to it: the few fields a
linking screen needs to identify and use it (a customer's name, code and billing address, a
driver's name). Every other field of the linked record needs that kind's own read; without it a
nullable field is answered null and named in `extensions.withheldByAccess`, and a non-null one is
refused. The record a root resolver returns, records that are part of it (`ParentResource`), and
everything under a self-scoped resolver are not gated this way.

Views are declared on each resource definition (`ReferenceView` in `resources_*.go`) and decided by
one package, `refview`, which the GraphQL field guard asks for people and agents alike, and the
REST response guard (`internal/api/responseguard`) asks for every REST response. Global search,
shipment analytics and reports ask it through `permissionports.ReferenceViews`; an insight names
records beside figures about them, so its reader needs read on every kind its detector declares. They are
fixed in the code; an organization decides who sees whole linked records through bundles and job
templates, and the role's sensitivity ceiling still applies inside a view. Built-in bundles grant
a linked kind's read where the job needs the full record (billing receivables reads invoices and
disputes, carrier settlements read carriers). How to build a screen that shows another record,
and how to change a view, is in [reference-views.md](reference-views.md).

## Protected fields

A few fields stay hidden from people who can read the record they sit on. Each belongs to a
group, and a group is granted like any other permission, as `read` on its own resource in
the **Protected fields** category of the role editor:

| Group | Resource | Fields |
|---|---|---|
| Personal details | `protected_personal` | `WorkerProfile.dob`, `licenseNumber`, `twicCardNumber`, `lastDrugTest`; `WorkerCredential.number` |
| Margins | `protected_margins` | `RateAgreement.marginFloorPercent`, `maxPayPercentOfSell` (and the same values on a version) |

The schema marks each field `@protected(group: …)` (`shared.graphqls`; the directive is
`skip_runtime`). `FieldGuardExtension` (`internal/api/graphql/fieldguard.go`) reads the
directives once when the server starts, refuses one on a non-null field, and returns null
for a protected field to a reader without the group, without resolving it. It asks the
engine at most once per group per operation, through the same per-request memo as the
resolvers. A protected field is therefore nullable, and on the client null means hidden:
the worker form, credential form, rate agreement form and Desk cards show "Hidden" and
disable the input.

The REST routes withhold the same fields through the entity's `WithholdFields`
(`internal/api/fieldguard`): the response guard calls it on every record in every response, at
the root or nested inside another (a driver on a shipment), and an update keeps the stored value of any group the writer
cannot see (`KeepFields` in the worker, credential and rate agreement services), so a form
that never received a field cannot clear it. Creating a record takes the values as sent.
Workers are published to the realtime stream without their entity, so a subscriber refetches
through the guarded paths.

Every protected field is also classified confidential in the registry, which keeps it out
of agent tools, reports and proposal previews by tier. `fieldguard_test.go` fails if a field
is protected without that classification, without an entity that withholds it, or if the
set changes without the table being updated.

The migration that added the groups granted them to every role and API key that could read
workers (personal details) or rate agreements (margins), so nothing disappeared on upgrade.
New roles get them from bundles: `safety.drivers` and `hr.workers` carry personal details,
`billing.rates` carries margins.

To protect another field: add the directive to a nullable field, classify it confidential on
its resource, give the entity `WithholdFields` (the REST response guard calls it wherever the
entity appears) and `KeepFields`, call `KeepFields` from its update path, show "Hidden" where the client reads it, and add it to the tables in
`fieldguard_test.go`.

## Agents

In a conversation an agent acts as the person: every tool call is checked against the
person's grants. When an agent runs on its own (a schedule, an event, a loop) its principal
is `PrincipalTypeAgent` with the agent definition's id, and a check passes only where:

1. the fixed agent ceiling allows it (`permission.IsAgentAllowed`), which no organization
   can widen;
2. the role the organization chose for the agent, if any, grants it, through the role's
   parents too (`agent_definitions.unattended_role_id`, read through
   `permissionports.AgentRoleSource` and cached per agent under the organization epoch);
3. no access policy denies it. Policies bind agent principals as they bind people; match
   them with the `principalType` condition `agent` or `principalId` the agent's id.

`setAgentUnattendedRole` needs `agent_definition:update` and `role:update`, and runs the
delegation check over the role's whole closure: the person setting it must hold every grant
and the sensitivity ceiling the role carries. The change advances the organization epoch,
so the next unattended call reads the new role, and is recorded as a critical audit entry.

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

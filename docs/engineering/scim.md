# SCIM provisioning

How an identity provider (Okta, Entra ID or any SCIM 2.0 client) creates, updates and
deactivates people in an organization, and how directory groups become role assignments. Paths
are under `services/tms/internal/` unless they say otherwise. Who may configure a directory, its
tokens and its group mappings is in [permissions.md](permissions.md#organization-security-settings).

## Shape

```
identity provider ──▶ /api/v1/scim/:tenantSlug/v2/{Users,Groups,...}
                         │ scimhandler: body ≤ 2 MiB, bearer token, rate limit per directory
                         ▼
                      scimservice.Authenticate ──▶ Directory {id, organization, business unit, token}
                         │ every later read and write runs under that tenant scope
                         ▼
        Users ──▶ directoryaccounts (create, profile, access) ──▶ userservice exit path
        Groups ──▶ scim_groups / scim_group_members, raise roles_version of affected people
                         │ after commit
                         ▼
        scimjobs.ReconcileSCIMDirectoryGrantsWorkflow (debounced per directory)
                         │ pages of people whose roles_version moved
                         ▼
        roleservice.AssignRole / UnassignRole, recorded in scim_role_grants
```

| Piece | Where |
|---|---|
| Wire format: resources, filters, patch operations, errors, discovery documents | `shared/scimproto` |
| Routes and SCIM error responses | `api/handlers/scimhandler` |
| Authentication, Users, Groups, reconcile | `modules/identity/app/scimservice` |
| Creating, updating and ending a person's access for a directory | `modules/identity/app/directoryaccounts` |
| Tables | `scim_users`, `scim_groups`, `scim_group_members`, `scim_role_grants` (`domain/scim`), migration `20261231023200_scim_provisioning.sql` |
| Reconcile workflow and the 15-minute sweep | `modules/identity/jobs/scimjobs` |

## Authentication binds the tenant

A token is `scim_<prefix>.<secret>`. The prefix is looked up under the system scope
(`scimrepository.FindCredential`, listed in `rlslint`), the SHA-256 of the whole token is compared
with the stored hash in constant time, and the request is refused unless the token is active and
unexpired, its directory is enabled, the path's `:tenantSlug` equals the directory's slug and the
organization's plan includes SSO. Every refusal is the same 401 with `WWW-Authenticate: Bearer`.

From then on the directory's organization and business unit are the request's tenant scope; no
value in the path or body chooses a tenant. A user or group ID in the path is looked up as
`(directory_id, id)`, so a directory reaches only the people it provisioned or linked and the
groups it pushed. Another directory's people answer 404.

## Users

- **Create** links an existing member when the email belongs to someone who is already a member
  of the directory's organization (`scimrepository.FindMember`, under the tenant scope, so a
  person of another organization is never found), and otherwise creates the account with a
  membership in that organization only. An email or username taken elsewhere is a 409. The
  instance's system user is never linked. A created account has an unusable password and no
  invitation: people sign in through SSO or reset their password.
- **Profile changes** (name, email, title, phone, timezone) apply only when the person belongs to
  no other organization. Otherwise they are withheld and a `withheld` provisioning record says so,
  because a directory must not change where another organization's sign-in or reset email goes.
- **active=false** ends access in the directory's organization through the existing exit path
  (`userservice.ExitOrganization`: the membership goes, every role in that organization is
  revoked, sessions and Capture devices bound to it end). A person who belongs only to that
  organization is deactivated instead (`Status` inactive, every session ended). **active=true**
  restores the membership and reactivates only a person who belongs nowhere else. Only a change of
  the directory's own `active` acts; an administrator's manual deactivation is not undone by a
  later replace that repeats `active: true`.
- **Delete** does what deactivation does, takes back the directory's role grants and removes the
  link. The account is never deleted.

## Groups and roles

A group grants a role when a group role mapping of the same directory names it, by the group's
`externalId` or, case-insensitively, by its `displayName`. Membership changes, renames, deletes,
a person's `active` flipping and every mapping save raise `scim_users.roles_version` for the
people affected in the same transaction, then signal the directory's reconcile workflow after
commit. The workflow debounces, works through people whose `roles_reconciled_version` is behind in
pages of 100 and brings each up to date:

- A role is granted only while **the person who saved the mapping** (`saved_by_id`) is an active,
  unlocked member of the organization and `RequireCanGrantRole` still passes for them. This is the
  A1-12 delegation check run again at provisioning time; the assignment is made with that person
  as the actor, so `AssignRole` checks it a third time. A mapping saved before `saved_by_id`
  existed names nobody and grants nothing until it is saved again.
- A grant the groups no longer justify, or whose saver lost the authority, is unassigned when
  that person is next reconciled. The saver's own access counts too: `SaverWatch` hears every
  permission change through the `permissionChangeListeners` group and every loss of access through
  `userAccessRevokers`, and marks the people whose grants came from that saver's mappings (or, for
  an organization-wide change, every mapping there), so a demoted or removed saver's grants are
  re-checked at once. A role assignment that simply expires raises no event, so the sweep also marks
  the mappings of savers whose assignment ran out in the last 30 minutes. Only
  assignments recorded in `scim_role_grants` are ever taken away; a role a person assigned by hand
  stays, and `ErrRoleAlreadyAssigned` leaves it unrecorded.
- An administrator who unassigns a directory-granted role sees it come back on the next reconcile
  while the groups still grant it; change the mapping or the group instead.
- Deleting a directory unassigns everything it granted first, in the same transaction
  (`RevokeDirectoryGrants`); the `scim_role_grants` foreign key refuses the delete otherwise.

A signal that fails to start the workflow leaves the people marked; the `scim-pending-grants-sweep`
schedule restarts reconciles for any directory with marked people every 15 minutes.

## Bounds

| Bound | Value | Where |
|---|---|---|
| Request body | 2 MiB | `scimhandler.maxBodyBytes` |
| Page size (`count`) | 100 | `scimports.MaxPageSize` |
| Patch operations per request | 100 | `scimports.MaxPatchOperations` |
| Members named in one request | 10,000 | `scimports.MaxMembersPerRequest` |
| Members returned with a group | 1,000; lists never include members | `scimports.MaxMembersReturned` |
| Filter | `eq` joined by `and`, up to 8 conditions, 1,024 characters | `scimproto.ParseFilter` |
| Rate | `security.rateLimit.scim` (600/min, burst 100) per directory; an unauthenticated request spends the client IP's anonymous budget | `RateLimiter.Allow`, `AllowClientIP` |

Filters accepted: Users `id`, `userName`, `externalId`, `emails.value` (also
`emails[type eq "work"].value`), `active`; Groups `id`, `displayName`, `externalId`,
`members.value`. Anything else is a 400 `invalidFilter`. Bulk, sort, ETags and password change are
not offered, and `ServiceProviderConfig` says so.

## Audit

Every provisioning change writes a `provisioning_audit_records` row (the timeline on the
Provisioning tab, with `succeeded`, `refused`, `withheld` or `failed`) and a `SecurityAuditor`
entry on the directory, attributed to the system actor with `scimDirectoryId` and `scimTokenId` in
its metadata. Account changes are recorded on the user the same way, role assignments by
`roleservice` ("Role assigned" names the mapping's saver as the actor, since the grant is made on
their authority), and membership exits by `userservice`.

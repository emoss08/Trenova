# Security review

The defects a full audit of this repository kept finding, as rules. Each one compiled, passed its
tests and read as correct in isolation; what made it a defect was a second path, a concurrent
request or a caller the author did not picture. Read this before changing who may do what, money,
sign-in, a stored secret or anything shared between people. Paths are under
`services/tms/internal/`. Who can reach the system at all is in
[trust-boundaries.md](../trust-boundaries.md).

To fix a reported finding, use the `trenova-security-fix` skill.

## Check, then act

A precondition read before the write is a hope, not a check: two requests both read `Accrued` and
both write. The write itself has to prove the precondition.

- Put the precondition in the write's `Where`, and compare the rows it changed with the rows you
  expected. `Update` returns the count; `_, err :=` throws away the only evidence of a lost race.
  `MarkSettled` (`driversettlementrepository/payevent.go`) and `UpdateCreditMemoApplication`
  (`customerpaymentrepository/creditapplication.go`) are the pattern, and both answer with
  `errortypes.NewConflictError`.
- `Save` guards a row only when the table has a `version` column. A table without one has no
  optimistic lock, however much the code around it looks like it does.
- Locking the invoice does not lock the row you read before you took the lock. Read under the lock,
  or make the write conditional.
- A step that is two writes in two transactions leaves the first behind when the second fails, and
  a retry repeats it. Wrap them in one `Transact`; a nested one joins it when the scope matches.

## One control, many paths

A control guards a path; the effect it protects can usually be reached another way. Write-off
approval lived in the adjustment engine while a short pay posted to the same write-off account from
the payment form. SSO settings sat behind `organization:update` on one route and `identity_provider`
on another.

When you add or review a control, name the effect (a journal line, a status, a balance, who can
sign in) and find every writer of it: REST, GraphQL, bulk, agent tool, Temporal activity, import. Search for the column or the repository method, not for the feature's name. If two surfaces
configure one thing, they need one permission.

## A tool asks what its page asks

An agent tool's `Policy()` names the permission the page's resolver checks, or it is a way around
the page. Compare the two whenever a tool or a resolver changes.

- A tool that does more than one thing still declares one `Resource` and `Operation`. A further
  grant the page asks of a person goes in `PersonAlsoNeeds`, which the dispatcher and
  `proposalexecutor` check for a person only. `detention_send_notice` and `billing_start_review`
  are the examples.
- Don't raise the declared operation to get there. An agent runs under `agentceiling`, and a
  declared permission no agent may hold takes the tool away from every agent that runs on it
  (`TestEveryToolADeskRunsOnIsOneAnAgentMayCall` says so). Outbound tools stay declared on the
  communication they send (`TestOutboundTools_AreGatedOnTheCommunicationTheySend`).

## A read for a person needs the person

`pagination.FromAuth` carries the tenant and no `UserID`; `FromAuthAsUser` carries both. A rule that
reads `TenantInfo.UserID` under `FromAuth` sees nobody, so a person loses their own records with
no error at the point of the mistake. Anything private to a person (a conversation upload through
`VisibleToPerson`, a personal memory) is read with `FromAuthAsUser`, on every route that reaches it.

## A composite read is several reads

A resolver that checks one resource and then calls a service for another has read the second without
a check. `WorkerTotalCompensation` checked `benefit_plan` and returned pay; a worker read returned
disciplinary standing.

- Check each resource the result draws from, with `HasPermission` for the optional part.
- Build the result without what the reader may not see, rather than blanking fields afterwards, so
  no derived figure carries it. `ScorecardWithoutDiscipline` never reads the actions, so the score
  doesn't reflect them either.
- A resource's parent grants read of all of it. Data most readers of the parent shouldn't see has
  no parent ([permissions.md](../permissions.md)).

## Anything shared has one way in

A rule for what reaches other people applies to every writer, not the one the rule was written
beside. `planRemember` kept shared memory for people who manage it, while `RecordCorrection` wrote
an organization-wide memory from any rejection reason. Text a person types into a free field and
values copied from their own conversation are both input to everyone else's prompt. Carry
`Tainted` from whatever the text came from, and let a person who may not publish only offer.

## A secret stays with where it was saved

A stored secret must not follow a change of address, because the person who may edit the address
may not be able to read the secret. `requireSecretsForNewTarget` (EDI), `applySiteRequest`
(portals), and the integration and AI provider services all ask for it again.

- "In use" and "presented to the target" are different questions. An AS2 private key is in use and
  only signs here, so it stays (`SecretBoundToTarget`). `SecretInUse` answering false deletes the
  secret, so don't reach for it to mean "don't ask again".
- A key in a query string reaches logs through `*url.Error`. Return
  `restx.SanitizeTransportError(err)`, or better, send the key in a header.

## Sign-in vouches for one place

An identity provider is configured by one organization and vouches for people there, not for what
else they can reach.

- A federated session moves only into an organization that trusts the same issuer
  (`requireTargetSignIn`), and `accountguard.RequireCovers` keeps an administrator from acting on an
  account with access they don't hold themselves.
- Trust in another system's assurance is opt-in (`AllowFederatedMFA`).
- A session field that decides access belongs in the seal (`authstore/seal.go`). Tag a new one
  `omitempty`, so the seal of every live session stays valid while the field is empty.
- Someone from outside the organization signs in only as a principal of their own. A Trenova
  support session is the grant's `sup_` principal, never a person row, so a user id never
  credits a customer's person with what support did; a new person-only path (self-service,
  sign-in settings) must hold for it too, and a new sensitive resource must be classified in
  `supportaccess/policy.go` ([managed-mode.md](../managed-mode.md#support-access)).

## A shared server's support entry names one tenant

On a shared server the support keys are the server's, so a signature proves only that the control
plane minted the assertion, not for which company. The audience (the tenant's instance) and `org`
must name the same active tenant before any grant is looked up, and the grant must be that
organization's (`supportaccessservice.admitTenant`). A new way to open a session for staff, or a
new reader of the key set, keeps both checks; never compare `aud` with a server-wide value on a
shared server. `TestSharedSupport_AGrantOfOneCompanyNeverOpensAnother` holds it.

## Stepping outside the tenant

`dbscope.WithSystem` turns row-level security off for that context, so the query is all that stands
between tenants. A caller's ID, slug or token that reaches one has to be narrowed to a tenant or to
a value proving access: a token looked up by hash, a realm the webhook verified. See
[row-level-security.md](../row-level-security.md). A use that exists for a sweep or at startup takes
no input from a caller; keep it that way.

## Background work has a tenant's own actor

Work that acts for nobody is written under the system actor of the business unit it works in
(`UserRepository.SystemActorID(ctx, businessUnitID)`), never an install-wide user found by name:
on a shared server that user would be one tenant's row named by every other tenant's records. A
sweep over many tenants resolves the actor per tenant inside its loop. Nothing a person can set (a
username, an address) ever decides who the system actor is; reserved names are refused in one
place (`tenant.IsReservedUsername`, `IsReservedEmailAddress`).

## Roles go to members

A role is assigned only to a live member of the organization: the write locks the membership
(`tenantbootstrap.LockMember`, `FOR SHARE`) in the same transaction as the insert, and answers a
non-member, another tenant's person and the system actor exactly as an unknown user. Every writer
of `user_role_assignments` goes through it (`rolerepository.CreateAssignment`,
`tenantbootstrap.AssignRole`), and the row carries its organization's business unit, so the
database refuses another tenant's person too.

## An id from input is the caller's to prove

A foreign key checks that a row exists, not that it is the caller's: row-level security does not
apply to it. An id that arrives in a request, a tool call, an import or an EDI mapping is looked up
in the caller's tenant before it is written (a validator `WithReferenceCheck`, or a scoped
`GetByID`, as `assignmentservice.ensureParties` does), or the row would name two tenants. The
removal purge refuses to touch such a row and the nightly archive names it.

## Object keys come from the registry

Every object a tenant writes is under one of its families in `kernel/domain/objectkey`, so the
removal and the archive find all of it; `TestEveryObjectKeyIsBuiltByTheRegistry` fails on a key
built in place. A key that arrives from outside (an archive manifest) is accepted only under the
tenant's own prefixes (`objectkey.Rebase`).

## An archive carries one tenant and no install secret

A tenant archive (`internal/cloud`, [shared-tenancy.md](../shared-tenancy.md#the-tenant-archive))
reads and writes every table of one tenant under the system role, so the tenant filter in each
statement is the only wall: review that a new table shape is classified by `archiveplan` (the
catalog test runs the plan on the real schema), that an export refuses rows that refer to another
organization, that a new sealed column or object is in `internal/bootstrap/envelopes` (or it
travels as the source install's ciphertext and is unreadable on the destination), that a new
per-install keyed hash is handled on import, and that a new route or workflow keeps the archive
target sealed and out of history and logs.

A column that names an organization other than the row's own `organization_id` (a counterparty,
a recipient, a membership's user) must be held to the row's business unit by a composite foreign
key (`(<column>, business_unit_id) REFERENCES organizations(id, business_unit_id)`), a check or a
trigger, as `20261231023600` does for in-network EDI and memberships. Each tenant has its own
business unit, so that one constraint is what keeps a forged or buggy row from naming another
tenant; the catalog test in `internal/cloud/boundaries_integration_test.go` fails on a new
organization-naming column that has none. Archive keys and chain keys never leave an install: an
imported audit trail is signed again under this install's chain key with the source's values kept
beside it, never by importing the source's key, and a trail this server could not verify where it
was signed is marked so in what the hash covers (hash version 3 or 4), so every verifier reports it
as imported and not verified rather than valid. A new reader of chain validity shows that state;
a new writer of a chain never signs imported links as ordinary ones.

## Reading what belonged to someone else is its own door

A record one person owns (a Desk conversation, a draft, a file attached to a conversation) stays
behind its owner filter on every path that already reads it. When people with authority must read
everyone's, they get a separate door, never a loosened filter: its own permission, granted by no
template and no bundle that is not about exactly that, refused to Trenova support, agents and API
keys; its own service and repository methods whose only callers are that surface; reads that
write nothing the owner would see as activity; and an audit entry for every look. The Desk
conversations oversight page is the model ([agent-runtime.md](../agent-runtime.md#other-peoples-conversations)).

## What ships

- The operator runs only charts the release workflow signed, and published charts pin every image
  by digest ([operator.md](../operator.md)). A new image in the chart is pinned by the same step as
  long as its values hold `repository` and `tag`.
- Pin third-party actions by commit in a workflow that publishes or holds a token.

## A fix can break the thing it guards

Most of the regressions in that audit came from the fixes. Before calling one done, ask:

- Who else runs this path? An agent or system actor fails a check written for a person.
- What is already in the database? A new column's default can read as a change that never happened;
  `20261231023153_capture_device_session.sql` backfills for that reason.
- What did the default become? A policy that used to be skipped now applies at its default value,
  which may refuse almost everything. Say so in the handoff.
- Does the client depend on the status? A hidden record answers 404 where it used to answer 403.

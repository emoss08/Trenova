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

## Stepping outside the tenant

`dbscope.WithSystem` turns row-level security off for that context, so the query is all that stands
between tenants. A caller's ID, slug or token that reaches one has to be narrowed to a tenant or to
a value proving access: a token looked up by hash, a realm the webhook verified. See
[row-level-security.md](../row-level-security.md). A use that exists for a sweep or at startup takes
no input from a caller; keep it that way.

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

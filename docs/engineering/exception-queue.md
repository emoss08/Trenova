# Exception queue

Every item on the watchtower is an exception someone owns: it has a status, an owner, a due
time, a history that is never edited, and an escalation path when nobody deals with it in time.
This page covers the workflow on `watchtower_items`, the history in `watchtower_item_events`,
routing and escalation, the automatic sources (missed pickup, missed delivery, POD missing,
tender exhausted, no check call and ETA at risk), the GraphQL operations, the agent tools and
the web queue. Read it before changing `watchtower` (`exception.go`), `watchtowerservice`
(`exceptions.go`, `escalation.go`, `routing.go`, `projector_routing.go`,
`projector_subject.go`), `watchtowerrepository`, `exceptionsourcerepository`,
`watchtowersources/shipment_exceptions.go`, the exception settings on dispatch controls, or the
exception tools.

## The shape of it

```
Source event / sweep ─▶ Projector.Upsert ─┬─ new or reopened ─▶ route: due by severity,
                                          │                     owner by rule (checked)
                                          └─ already open ─────▶ title/summary refresh only
Source clears ─▶ Projector.Resolve ─┬─ open item ─▶ Resolved / Cleared (system)
                                    └─ person resolved it ─▶ source_cleared_at only

Person (GraphQL) ─┐                          ┌─ version check ─▶ watchtower_items
Agent tool ───────┴─ Exceptions.<action> ────┼─ events ────────▶ watchtower_item_events
                                             ├─ audit entry, realtime, notification
                                             └─ DryRun: preview only, nothing written

exception-attention (5 min) ─▶ wake snoozed items, escalate overdue ones (capped)
shipment-exception-sweep (10 min) ─▶ ReconcileKind for the four shipment kinds
Shipment event ─▶ ShipmentExceptionObserver ─▶ ReconcileSubject for that shipment
```

| Piece | Where |
|---|---|
| Workflow on the item, statuses, codes, events | `agent/domain/watchtower` (`item.go`, `exception.go`) |
| Migration | `20261231009210_exception_queue` (columns, history table, settings) |
| Repository | `agent/adapters/postgres/watchtowerrepository` |
| Source reads (missed stops, PODs, tenders) | `agent/adapters/postgres/exceptionsourcerepository` |
| Workflow service | `watchtowerservice.Exceptions` (`exceptions.go`), `Dismisser` delegates to it |
| Routing policy | `watchtowerservice.Router` (`routing.go`), read from dispatch controls |
| Routing on raise | `projector_routing.go` |
| Per-shipment reconcile | `projector_subject.go` (`WatchtowerSubjectReconciler`) |
| Escalation and wake | `escalation.go`, `ExceptionAttentionWorkflow` in `agent/jobs/watchtowerjobs` |
| Automatic sources and observer | `agent/app/watchtowersources/shipment_exceptions.go`, `shipment_exception_observer.go` |
| GraphQL | `watchtower.graphqls`, resolvers in `resolver/watchtowerresolver` |
| Agent tools | `agent/app/agenttools/receivabletools/exception_tools.go` |
| Web | Desk watchtower (`routes/desk/_components/watchtower`), dispatch controls "Exception queue" card |

## The workflow

| Status | Means | Gets there by |
|---|---|---|
| `Open` | Nobody has said they are on it | being raised, waking from a snooze, reopening |
| `Acknowledged` | Someone has seen it | `acknowledgeWatchtowerItem` (only from `Open`) |
| `InProgress` | Someone is working it now | `startWatchtowerItem` |
| `Snoozed` | Set aside until `snoozed_until` | `snoozeWatchtowerItem` (at most 30 days) |
| `Resolved` | Closed with a reason | `resolveWatchtowerItem`, or the source clearing |

Acknowledging or starting an item nobody owns makes it the reader's. A snoozed item wakes back
to `Acknowledged` if it had been acknowledged, else `Open`. The database holds the shape:
`(status = 'Resolved') = (resolved_at IS NOT NULL)`, a snoozed item has `snoozed_until`, and the
resolution code is one of the list below.

Every write takes the version the person was looking at (`expectedVersion`); a newer version
refuses the change with a version-mismatch error, so two people working the same exception
never overwrite each other silently. Every change appends one or more rows to
`watchtower_item_events` (a trigger refuses `UPDATE`), writes an audit entry for
`ResourceWatchtower`/`OpUpdate` with the before and after state, and publishes a realtime
invalidation on `watchtower`.

### Resolution codes

`Fixed`, `CarrierDelay`, `ShipperDelay`, `ReceiverDelay`, `CustomerChange`, `Rescheduled`,
`Weather`, `Equipment`, `DriverIssue`, `DocumentReceived`, `DataCorrected`, `Duplicate`,
`NoActionNeeded`, `Other` (needs a note) and `Cleared`, which only the system uses when a source
stops reporting the problem. The list is a fixed enum in the domain rather than a per-organization
table, so reports over resolutions mean the same thing in every organization.
`watchtowerResolutionCodes` serves the ones a person may pick, with their labels.

## A person's resolution and the source

The source stays authoritative about whether the problem exists; a person is authoritative about
whether it has been dealt with. The two meet in `planUpsert`:

- A person resolves an item while the source still reports the problem: the resolution holds.
  Re-reports from the source refresh nothing but the title and summary.
- The source then clears: `Resolve` stamps `source_cleared_at` and leaves the resolution alone.
- The problem comes back: the item reopens (`SourceReopened`), clearing the resolution,
  acknowledgement, snooze, due time and escalation, and **keeping the owner**.
- An item the system resolved (`Cleared`) reopens on the next report.

A source never overwrites the owner, the status or the resolution, and never lowers a severity
that escalation raised. Retention deletes resolved items after 30 days except one a person
resolved whose source is still open, so the resolution is not lost while the problem persists.

## Routing

New and reopened items, and open items that have never had a due time, are routed by the
projector:

- **Due time**: `now` plus the organization's minutes for the item's severity
  (`exception_due_critical_minutes` 60, `exception_due_warning_minutes` 240,
  `exception_due_info_minutes` 1440; each 5 to 10080). Due times depend on severity only.
- **Owner**: the shipment's owner (`WatchtowerItemInput.OwnerID`, filled by the shipment
  sources), else `exception_default_assignee_id`. Each candidate must hold watchtower read and
  read on the item's source resource; one who cannot see the item is skipped. A routed owner is
  notified.

The policy is read through `Router`, cached per tenant for a minute.

## Escalation

`ExceptionAttentionWorkflow` runs every five minutes on the system queue. It lists the tenants
with an overdue or waking item (`ListTenantsNeedingAttention`, a system-scope read listed in the
RLS allowlist) and for each:

1. Wakes snoozed items whose time has come (`Woke`), telling the owner.
2. Escalates each overdue item below `exception_max_escalations` (default 3, 0 turns it off):
   the level goes up, severity is raised one step (Info → Warning → Critical), the due time
   becomes now plus `exception_escalation_minutes` (default 60), an `Escalated` event is
   written, and the owner and `exception_escalation_contact_id` (if they can see it) are notified.

An item at the cap stays overdue and is no longer picked up by the sweep.

## Automatic sources

| Kind | Raised when | Source id | Clears when |
|---|---|---|---|
| `MissedPickup` | a pickup's window (end, else start) closed more than `exception_missed_stop_grace_minutes` (default 30) ago with no arrival, on an open shipment | stop | the arrival is recorded, the stop or shipment is cancelled |
| `MissedDelivery` | the same for a delivery stop | stop | the same |
| `PodMissing` | the last delivery arrived more than `exception_pod_grace_hours` (default 24) ago, within 90 days, and no current, unrejected document of type `POD` is attached to the shipment | shipment | a POD is filed |
| `TenderExhausted` | the move's latest tender is `Exhausted`, the move is still `New` with no live carrier assignment, and the shipment is open | tender | the move is covered or tendered again |
| `NoCheckCall` | see [check-calls.md](check-calls.md) | move | a check call lands |
| `EtaAtRisk` | see [eta.md](eta.md) | shipment | the projection recovers |

The four shipment kinds are raised by `shipment-exception-sweep` every ten minutes, which
reconciles every organization against `exceptionsourcerepository`. A shipment event (arrival,
departure, stop completed, status change, cancel, carrier or driver assignment, tender events,
ownership transfer) settles that shipment's items at once through `ReconcileSubject`, so an
arrival or a covering carrier takes the item off the queue without waiting for the sweep. All
six kinds carry the shipment owner for routing.

## GraphQL

`WatchtowerItem` carries `status`, `assignee`, `dueAt`, `overdue`, `snoozedUntil`,
`acknowledgedAt`, `resolutionCode`, `resolutionNote`, `resolvedBy`, `sourceClearedAt`,
`escalationLevel` and `lastEscalatedAt`. `watchtowerItems` filters by `statuses`,
`assignedToMe`, `assigneeIds`, `unassigned`, `overdueOnly` and `subjectId`, and `orderByDue`
reads soonest due first with undated items last (the cursor then encodes the due time).
`watchtowerCounts` adds `overdue`, `unassigned`, `mine` and `mineOverdue`.

Queries: `watchtowerItem`, `watchtowerItemHistory`, `watchtowerResolutionCodes`. Mutations:
`assignWatchtowerItem`, `acknowledgeWatchtowerItem`, `startWatchtowerItem`,
`snoozeWatchtowerItem`, `setWatchtowerItemDue`, `commentOnWatchtowerItem`,
`resolveWatchtowerItem`, `reopenWatchtowerItem`. Each needs watchtower read or update and read
on the item's source resource; the assignee check is the same pair, evaluated for the assignee.

## Agent tools

| Tool | Does | Tier |
|---|---|---|
| `assign_exception` | gives an exception to a person who can see it, or to nobody | Propose → Ask first |
| `snooze_exception` | sets it aside until a local time | Propose → Ask first |
| `set_exception_due` | moves or clears the due time | Propose → Ask first |
| `resolve_exception` | closes it with a code (and a note for `Other`) | Propose → Ask first |
| `reopen_exception` | reopens a resolved exception | Propose → Ask first |
| `comment_on_exception` | adds a line to the history | Propose → Ask first |

Each preview is computed by calling the same service method with `DryRun` set, so the person
approves exactly the change the tool would make. Exceptions about an agent's own work
(`OverseesAgents`) are refused and stay for a person. The load monitor holds the tools, which is
why the agent permission set grants watchtower update. `list_watchtower_items` shows each item's
status, owner, due time, overdue and escalations, and takes `assignedToMe` (ignored for an agent
on its own), `unassigned`, `overdueOnly` and `orderByDue`. Acknowledging and starting are a
person's attestations and have no tool.

## Web

The Desk watchtower has four views: All (newest first), Mine, Unassigned and Overdue (soonest
due first), with counts. Each row shows the status, owner and due time; its title opens the
exception sheet, which holds the details, the actions (acknowledge, start, assign, snooze,
change due time, resolve, reopen), a comment box and the history. Dispatch controls carry an
"Exception queue" card for the due times, escalation interval and cap, default owner,
escalation contact and the two source graces.

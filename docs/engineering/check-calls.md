# Check calls and carrier tracking links

How a brokered load is followed once a carrier has it: the check calls a dispatcher logs, the
link a carrier posts through without a login, the agent tool, and the sweep that raises a load
that has gone quiet. Read this before changing `checkcall`, `checkcallservice`,
`checkcallrepository`, `carriertrackingpublichandler`, the `NoCheckCall` watchtower source or
the check call settings on dispatch controls and customers. Stop actuals themselves are settled
by tracking events; see [tracking-events.md](tracking-events.md).

## The shape of it

```
Dispatcher (logCheckCall) ─┐                      ┌─ Arrival / Departure ─▶ RecordReported (strict)
Agent (log_check_call)  ───┤ CheckCallService.Log ┤
                           │                      └─ every kind ─▶ check_calls row
Carrier (tracking link) ───┴ PostByToken ─────────┬─ Arrival / Departure ─▶ RecordObserved (tolerant)
                                                  └─ every kind ─▶ check_calls row
                                                                        │ after commit
                                                                        ▼
                                                  CheckCallObserver ─▶ resolve NoCheckCall item

no-check-call-sweep (15 min) ─▶ ListOverdue per monitoring tenant ─▶ ReconcileKind(NoCheckCall)
```

| Piece | Where |
|---|---|
| Entities and limits | `internal/modules/operations/domain/checkcall` (`CheckCall`, `TrackingLink`, `enums.go`) |
| Ports | `operations/ports/checkcallports` (`CheckCallService`, `CheckCallObserver`, repositories) |
| Service | `operations/app/checkcallservice` (`record.go`, `link.go`, `public.go`, `overdue.go`) |
| Repositories and tables | `operations/adapters/postgres/checkcallrepository`, migration `20261231009050_check_calls` |
| Public routes | `internal/api/handlers/carriertrackingpublichandler` |
| Link email | document template kind `carriertracking.link.email` |
| Dead link purge | `operations/jobs/checkcalljobs` (schedule `carrier-tracking-link-retention`) |
| Watchtower source, observer | `agent/app/watchtowersources/checkcall.go`, `DescribeNoCheckCall` in `describe.go` |
| Sweep | `NoCheckCallSweepWorkflow` in `agent/jobs/watchtowerjobs` (schedule `no-check-call-sweep`) |
| GraphQL | `checkcall.graphqls`, resolvers in `resolver/checkcallresolver` |
| Agent tool | `log_check_call` in `agent/app/agenttools/movetools`, held by the load monitor |
| Web | "Check calls" under move details, tracking link controls on a carrier-covered move, `/carrier-tracking/:token` |

## The record

A `check_calls` row is one report about one move: who it came from (`Dispatcher`, `Carrier`,
`Agent`), what it says (`Update`, `Arrival`, `Departure`), when it happened (`reported_at`) and
when Trenova received it (`received_at`), an optional position (coordinates, city, state),
whether the load is running late and why, the expected arrival at the next stop, a note and
who was spoken to. An arrival or departure names its stop and links the tracking event it
settled the stop through; a carrier post also links the tracking link it came in on.

`(organization, business unit, source, source_key)` is unique, so a repeat is stored once and
answered with the stored row (`Replayed`). The key per source:

| Source | Key |
|---|---|
| Dispatcher | a new id per call (the GraphQL mutation carries no key) |
| Agent | the tool call's idempotency key |
| Carrier | `ctl:<tracking link id>:<clientEventId>` from the page, else a new id |

Times are checked the same way for everyone except where noted: a reported time more than 7
days back is refused, an expected arrival more than 30 days out is refused, and a reported
time in the future is refused for a dispatcher or an agent but read as now for a carrier,
whose phone clock is not ours to argue with. A canceled load takes no check calls.

## Stop visits

An arrival or departure is also a stop actual, and goes through tracking events in the same
transaction as the check call:

- A **dispatcher or agent** visit is `RecordReported` with source `Dispatcher` or `Agent`, so
  the strict stop rules answer it and nothing is stored when they refuse.
- A **carrier** visit is `RecordObserved` with source `Carrier` (precedence 25, between driver
  and telematics), match `Direct`, raw reference `carrier_tracking_link:<link id>`. A carrier is
  outside the organization, so its report is weighed against the others rather than obeyed: a
  refusal is recorded on the tracking event and the carrier is told the stop time did not
  change, but its check call is kept.

## The tracking link

A dispatcher issues a link for a move covered by an outside carrier (an active carrier
assignment on a move that is not completed or canceled). Issuing revokes the live link on the
move and creates a new one; only one is live at a time.

- The token comes from `shared/tokenutils`; only its SHA-256 is stored, so the link is shown
  once, in the issue response, and cannot be read back.
- The URL is `Config.PublicLinkBaseURL()` (`tendering.publicBaseUrl`, else `app.webBaseUrl`) +
  `/carrier-tracking/<token>`. Without a public base URL the link is refused rather than built
  against an internal host.
- With an email address the link is sent with the `carriertracking.link.email` template under
  the idempotency key `carrier-tracking-link-<link id>`. A failed send keeps the link and
  returns the reason, so the dispatcher can copy it instead.
- A link lives 21 days. Issuing and revoking write an audit entry on the move.
- Dead links (revoked or expired) are deleted 30 days after they died by the daily
  `carrier-tracking-link-retention` job.

The public routes are `GET /api/v1/carrier-tracking-links/:token/` (the load as the carrier
sees it) and `POST /api/v1/carrier-tracking-links/:token/check-calls/`. Both are throttled per
token. The token is resolved under a system scope (`GetByHash`) before the tenant is known and
everything after runs in the link's tenant. An unknown, revoked or expired token, a move that
is gone, and a carrier assignment that changed while the load is open all get the same answer,
"This tracking link is no longer valid", so the page leaks nothing about why. A finished or
canceled load still opens, read-only, and refuses posts with a business error. Each post marks
the link used.

What the carrier sees is the shipper and carrier names, the PRO number, the stops (place,
window, recorded times) and when the last check call came in. Nothing about the customer, the
rate or other loads.

## Monitoring

`dispatch_controls.enable_check_call_monitoring` turns it on for an organization and
`check_call_interval_hours` (1 to 72, default 4) is the window. A customer's
`check_call_interval_hours` overrides it for that customer's loads;
`DispatchControl.CheckCallWindowHours` chooses.

A monitored move is in transit, covered by a carrier, with a carrier assignment that is not
canceled. Its last contact is the latest of: a check call received, a tracking event received
from any source, a stop's actual arrival or departure, and the carrier's confirmation of the
assignment. A move with none of those is not judged. It is overdue once the last contact is
more than the window ago.

An overdue move is a `NoCheckCall` watchtower item keyed by the move id, linked to the
shipment, a warning, and critical once the silence passes twice the window. Its time is when
it went overdue, and its text does not change from sweep to sweep, so the feed is not
rewritten every fifteen minutes.

- `no-check-call-sweep` runs every 15 minutes on the system queue and calls
  `WatchtowerService.ReconcileKind(NoCheckCall)` for each organization in
  `ListCheckCallMonitoringTenants`: overdue moves are upserted, the rest resolved.
- The nightly reconcile reads the same source, which also clears the items of an organization
  that turned monitoring off.
- A check call resolves its move's item at once: `checkcallservice` calls every
  `CheckCallObserver` in the `check_call_observers` group after commit, among them the agent
  module's resolver and the ETA trigger that refreshes the shipment's projected arrivals (see
  [eta.md](eta.md)). A tracking event from telematics or EDI counts as contact
  too, and its move clears on the next sweep.

## Changing it

- A new source of check calls needs a value in `checkcall.Source`, the CHECK constraint on
  `check_calls.source`, the GraphQL enum and `use-check-call-labels.ts`, and a key that is the
  same when the same report is sent again.
- Never record a stop visit from a check call any way but through tracking events.
- Anything more a carrier can see goes into `PublicTrackingView` deliberately: the page is
  open to whoever holds the link.
- `issueCarrierTrackingLink` and `revokeCarrierTrackingLink` are `pending` in
  `writecoverage.yml`; an agent tool for them is a write that reaches a counterparty.

# Tender bounces, counter-offers and escalation

What happens to a carrier tender after the first offer goes out: a carrier that accepts and then
backs out, a carrier that answers with a rate of its own, and a tender every carrier passed on.
Read this before changing `tender` (`retender.go`, `counter.go`, `escalation.go`),
`tenderservice` (`bounce.go`, `counter.go`, `escalate.go`, `lifecycle.go`), `tenderjobs`, the
tender routes, or the tender settings on dispatch controls.

## The shape of it

```
                       ┌── Accept ──────────────▶ AcceptOffer ─▶ carrier assigned at AgreedRate()
Carrier / dispatch ────┤
(email link, EDI 990,  ├── Decline ─────────────▶ DeclineOffer ─▶ next rank
 respond route, agent) │
                       ├── Counter ─────────────▶ RecordCounter ─▶ Countered (decision deadline)
                       │                              ├─ above floor + auto-accept ─▶ AcceptOffer
                       │                              ├─ dispatch takes it ─────────▶ AcceptOffer
                       │                              ├─ dispatch refuses it ───────▶ CounterRejected ─▶ next rank
                       │                              └─ deadline passes ───────────▶ Expired ─▶ next rank
                       │
                       └── BackedOut / EDI decline of the accepted offer
                                          ▼
                       Bounce ─▶ cancel carrier assignment ─▶ Reopen (Revoked + reoffers)
                                          ─▶ start round N+1 of the tender workflow

Every rank passed ─▶ MarkExhausted ─▶ escalateExhausted (dispatch controls policy)
                                          ├─ Notify ──────▶ nothing more; TenderExhausted on the queue
                                          ├─ Rebroadcast ─▶ new SpotBroadcast tender, same carriers
                                          └─ Rewaterfall ─▶ new SpotSequential tender, same carriers
```

| Piece | Where |
|---|---|
| Rules (pure) | `operations/domain/tender`: `retender.go` (`CanBounce`, `ReofferPlan`), `counter.go` (`EvaluateCounter`), `escalation.go` (`EscalationPlan`, `EscalationRate`, `RateCeiling`) |
| Settings | `operations/domain/dispatchcontrol` (`TenderPolicy`, `TenderExhaustedAction`) |
| Service | `operations/app/tenderservice`: `bounce.go`, `counter.go`, `escalate.go`, `lifecycle.go` |
| Repository | `tenderrepository.Reopen`, `UpdateOfferStatus` with `Counter` and `CounterDecided` |
| Workflow | `operations/jobs/tenderjobs/workflow.go`, `RecordCounterActivity` |
| Migration | `20261231009320_tender_bounce_counter` |
| REST | `POST /tenders/:tenderID/bounce/`, the respond route's `Counter` action, public `POST /tender-offers/:token/counter/` |
| Agent tool | `record_tender_response` (`Accept`, `Decline`, `Counter`, `BackedOut`) |
| Web | dispatch console live tender panel and tender history, `/tender-offer/:token`, "Carrier tenders" on dispatch controls |

## Rounds and workflow IDs

A tender runs one Temporal workflow per round. Round 0 is `carrier-tender-<id>`; every bounce
increments `tenders.bounce_count` and starts `carrier-tender-<id>-r<N>`
(`tendercontract.WorkflowID`). A round's workflow finished when the carrier accepted, so it
cannot be signalled back to life. Starting a new ID for the next round means no workflow ever
has to rewind, and a late signal for the old round reaches a workflow that has already
finished instead of one that is waiting on a different offer. Every caller that signals or
cancels a tender builds the ID from the tender's own `bounce_count`; never build it by hand.

## Bounce: a carrier backs out after accepting

`Bounce` takes the accepted tender (or its accepted offer) and a reason. In order:

1. `planBounce` checks the tender can bounce (`Accepted` or `NeedsReview`, with the accepted
   offer still `Accepted`), that no other tender is live on the move, and that the move's carrier
   assignment, if any, belongs to the carrier backing out. A move someone has since covered with
   another carrier is left alone.
2. The carrier assignment is canceled through `CarrierMoveAssigner.Cancel`. That voids the rate
   confirmation and carries the usual cancel guards: it only succeeds while the move is still
   `Assigned`, so a carrier cannot back out of freight already moving.
3. `Reopen` runs in one transaction: a CAS from `Accepted`/`NeedsReview` to `Active` that clears
   the acceptance and bumps `bounce_count`, the accepted offer moved to `Revoked` with the reason
   and who recorded it, and the new offers inserted. Losing the CAS rolls everything back.
4. The next round's workflow starts, then the `TenderBounced` event, a dispatch notification and
   the audit entry.

The offers sent next come from `ReofferPlan`. The carriers the acceptance cut short get a new
offer: those whose offers were `Superseded`, and those `Skipped` without ever being sent because
the waterfall stopped above them. A carrier skipped for ineligibility keeps its reason and is
left out, as is any offer already re-offered and any carrier that has backed out of this tender.
They are **new rows** linked by `reoffered_from_id`, never the old rows reopened: delivery is
idempotent per offer (email `tender-offer-<offerID>`, one EDI 204 per offer), so a reopened
offer would never reach the carrier again. `MaxBounces` (100) caps the cycle.

An EDI 990 that declines the offer the carrier already accepted is a bounce, not a late decline.
`RecordResponse` checks for this first (`bounceOnCarrierDecline`), using the 990's reason.

## Counter-offers

A carrier answers an open offer with a rate (`Counter`, with an optional note of up to 500
characters). This is allowed only while counter-offers are on in dispatch controls.
`RecordCounter` moves the offer from `Sent` to `Countered` with a CAS, revokes the offer's
response tokens, and sets `expires_at` to the decision deadline (`tender_counter_decision_minutes`).
Losing the CAS files the counter as a late response.

`EvaluateCounter` judges the counter against the **organization's counter floor**
(`tender_counter_margin_floor_percent`), not a contract floor. Sell is the shipment's
`TotalChargeAmount`; buy is `CarrierBaseAmount(method, counterRate, move.Distance)`:

- No sell price means the margin cannot be known, so the counter needs approval.
- A margin below the floor needs approval, with the explanation stored on the event.
- Otherwise the counter is taken without anyone when `tender_auto_accept_counters` is on, and
  waits for dispatch when it is off.

The margin and the approval flag are stored on the offer (`counter_margin_percent`,
`counter_needs_approval`) so the panel, the agent and the preview all read the same verdict.

Dispatch decides through the same respond route, with no separate endpoint: `Accept` on a
`Countered` offer takes the counter, and `Decline` refuses it. `counterDecisionFor` turns those
into `DecideCounter`. Taking a counter that needs approval also requires `tender:approve`
(`OpApprove`, checked through the `PermissionEngine` with the request's actor). `tender:update`
is enough for one at or above the floor. Taking it is `AcceptOffer` from `Countered`: the carrier
is assigned at `AgreedRate()`, the counter rate, and the decision is stamped. Refusing moves the
offer to `CounterRejected` and the waterfall goes on. A counter nobody decides by its deadline
is expired by the workflow's timer like an unanswered offer, with a `TenderCounterDecided` event.

`AgreedRate()` is the counter rate whenever there is one and the offer is not `CounterRejected`,
else the offered rate. Pay previews, the carrier assignment and the GraphQL `agreedRate` field
all read it. Never read `rate` for what the carrier will be paid.

## Escalating an exhausted tender

When every rank has passed, `MarkExhausted` calls `escalateExhausted`. It runs in the service,
after the exhaustion is saved, so the workflow gains no new commands and in-flight executions
replay unchanged. The policy comes from dispatch controls:

| Setting | Meaning |
|---|---|
| `tender_exhausted_action` | `Notify` (the default: nothing more), `Rebroadcast` (a `SpotBroadcast` to the same carriers), `Rewaterfall` (a `SpotSequential` in the original rank order) |
| `tender_escalation_rate_increase_percent` | Raise each carrier's rate by up to this much (0–100) |
| `tender_escalation_offer_ttl_minutes` | How long each escalated offer stays open (5–10080) |
| `tender_max_escalations` | Rounds after the first tender (0–5, never above `MaxEscalationRound`) |

`EscalationPlan` makes one offer per carrier on the exhausted tender and leaves out any that
backed out. The raised rate never passes `RateCeiling`, the buy that would leave exactly the
counter floor's margin, and never falls below the original rate. With no sell price there is
no ceiling, so rates stay as they were. The new tender records `escalated_from_tender_id` and
`escalation_round`, which a CHECK keeps consistent, and goes through `persistAndStart`, so it is
held to every check a person's tender is: a move covered, held or canceled in the meantime is
not tendered again.

Escalation never fails the exhaustion it follows. A round that cannot start, or a policy at its
limit, notifies dispatch (`tender_escalation_stopped`), and the move stays on the exception queue
as `TenderExhausted` (see [exception-queue.md](exception-queue.md)). `TenderBounced` and
`TenderEscalated` re-evaluate that source.

## Events

| Type | Severity | Metadata |
|---|---|---|
| `TenderBounced` | danger | `carrierName`, `reason`, `source`, `reoffers`, `round` |
| `TenderCountered` | brand | `carrierName`, `offeredRate`, `counterRate`, `marginPercent`, `needsApproval`, `autoAccepted`, `source` |
| `TenderCounterDecided` | success / info | `carrierName`, `counterRate`, `approved`, `reason` |
| `TenderEscalated` | brand | `mode`, `round`, `rateIncreasePercent`, `carriers` |

The ETA and customer-status triggers ignore all four. The web activity feed renders each one
(`event-renderer.tsx`), and the shipment event parity test keeps the client's list in step with
the server's.

## The agent

`DispatchAssistant` is at the template tool cap, so there is no new tool. `record_tender_response`
gained `Counter` (with `counterRate` and `counterNote`) and `BackedOut` (which bounces, using
`offerId` and `declineReason` as the reason). Recording an acceptance, a counter or a backout moves
money or takes the carrier off the move, so those are proposals a person decides. A decline runs
once a person approves it. Previews show the counter against the offered rate, whether taking it
needs approval, and for a bounce, the carrier unassigned and the carriers asked next.

## Web

- **Dispatch console, live tender:** a countered offer shows its rate, margin, note, a "Needs
  approval" badge and the time left to decide. "Take counter" is hidden from a user without
  `tender:approve` when the counter needs approval. "Refuse counter" asks for a reason. "Record
  response" can also record a counter heard by phone. A tender in review offers "Carrier backed
  out".
- **Tender history:** an accepted tender offers "Carrier backed out" to a user with
  `tender:update`. Each tender shows its escalation round, how often carriers backed out, and the
  agreed rate after a counter.
- **Public offer page:** "Propose a different rate" appears only when `counterAllowed` (counter
  offers on, offer still `Sent`). The carrier is told the load is not theirs until the broker
  confirms.
- **Dispatch controls:** the "Carrier tenders" card, shown to brokerages, holds the eight settings.

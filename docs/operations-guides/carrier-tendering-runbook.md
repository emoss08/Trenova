# Carrier Tendering Runbook

## Scope

This runbook covers the operational surfaces of carrier tendering:

- the tender lifecycle and the background sweeps that keep it moving
- the `NeedsReview` state and how a dispatcher clears it
- rate confirmation issuance failures and their recovery paths
- the public token lifecycle behind emailed offer and signing links

It is for dispatchers working the console, and for operators watching the
dispatch Temporal worker.

## Lifecycle Overview

A tender is created against one shipment move and holds that move's single
live-tender slot until it reaches a terminal state.

| Status | Meaning | Live? |
| --- | --- | --- |
| `Active` | Offers are being dispatched and awaited. | Yes |
| `Accepted` | A carrier accepted and the move was assigned. | No |
| `Exhausted` | Every carrier declined, expired, or failed delivery. | No |
| `Canceled` | Withdrawn by a dispatcher, or automatically when the move was covered outside the tender. | No |
| `NeedsReview` | A carrier accepted but auto-assignment failed. | Yes |

`NeedsReview` is deliberately not terminal. The workflow has stopped driving
the tender, but it still occupies the move's live-tender slot and still needs
a human.

Tenders run in three modes: `Waterfall` and `SpotSequential` walk their offers
one rank at a time; `SpotBroadcast` dispatches every offer at once and takes
the first acceptance.

### One workflow per tender

Every tender is driven by its own Temporal workflow, `CarrierTenderWorkflow`,
on the `dispatch-queue` task queue. The workflow identifier is deterministic —
`carrier-tender-<tenderID>` — so a tender is recoverable even if it crashed
before its workflow ID was recorded.

The workflow owns every state transition:

- it dispatches each offer and holds the expiry timer
- carrier responses reach it as `tender_signal` signals from the single
  service funnel, never as direct writes
- cancellation arrives on the same signal channel, so an in-flight response
  cannot race a withdrawal

Every activity is CAS-idempotent, so a restarted workflow rebuilds its
position from the database rows rather than from workflow state.

### `tender-stalled-sweep` (every 5 minutes)

`TenderSweepWorkflow` is the safety net under the workflow-per-tender model.
It lists `Active` tenders that look stuck — either their in-flight offer went
overdue more than 120 seconds ago, or no offer is in flight at all past that
grace window (a workflow that died before its first dispatch, or after its
last offer resolved without reaching a terminal status). For each of up to 200
such tenders it:

1. signals the workflow with a `SweepNudge`; if the signal lands, the workflow
   is healthy and takes it from there;
2. if the signal cannot be delivered, the workflow is gone — the sweep starts
   it again under the same deterministic ID.

The schedule uses overlap policy `SKIP`, so a slow run is never doubled up.
A run reports `Checked`, `Nudged`, `Recovered`, and `Failed`; a non-zero
`Failed` means workflows could neither be signaled nor restarted and is worth
paging on.

## NeedsReview

### What it means

A carrier accepted the offer — that acceptance is committed and is never
rolled back — but the system could not put that carrier on the move
automatically. The tender is parked in `NeedsReview` with the assignment
failure recorded as its reason. Sibling offers are closed out at the same
time, so no other carrier is still holding a live offer.

The reasons come straight from the assignment attempt:

- **Competing coverage** — `Move is already covered by another carrier`: the
  move gained a different carrier assignment between dispatch and acceptance.
- **Driver already assigned** — `Shipment move already has a driver assignment.
  Unassign the driver before assigning a carrier`.
- **A dispatch hold** on the shipment blocks any assignment.
- **Carrier eligibility** — the carrier failed the compliance/insurance gate at
  assignment time even though it passed at tender time.
- **Assignment unavailable** — `Carrier assignment is unavailable; assign the
  carrier manually` when the assigner is not wired.

Only deterministic (business) failures park the tender. Conflicts and infra
errors are retried by the activity instead, so `NeedsReview` never means "try
again in a second".

### How it surfaces

1. A high-priority global notification, event type `tender_needs_review`,
   titled *Tender acceptance needs review*, carrying the carrier name and the
   reason. It is categorized under Dispatch in the notification center and
   deep-links to `/dispatch/console?move=<moveID>`.
2. A `TenderNeedsReview` entry on the shipment's activity feed with the
   carrier and reason.
3. A warning banner on the tender panel in the dispatch console —
   *Carrier accepted, but auto-assignment failed* — with an **Assign manually**
   button and the cancel action.

### How a dispatcher clears it

Both paths end with the tender leaving `NeedsReview`, which frees the move's
live-tender slot.

**The carrier should keep the load:**

1. Open the move from the console banner and use **Assign manually** (the
   regular carrier-assign flow). This requires `ShipmentMove:Assign`.
2. Assign the accepting carrier. Resolve whatever blocked it first — remove the
   dispatch hold, unassign the driver, or clear the competing assignment.
3. Cancel the tender with a reason describing the manual assignment. This
   requires `Tender:Cancel`.

Assigning a carrier or a driver to the move also withdraws a live tender on
its own, so step 3 may already be done by the time you get to it.

**The carrier should not keep the load:** cancel the tender with a reason. A
`NeedsReview` tender no longer has a workflow, so the cancel is applied
directly to the rows; the same path is the fallback when a workflow is lost.

A cancellation reason is mandatory — the request is rejected without one.

## Rate Confirmation Issuance Failure

### What the notification means

When a tender is accepted, the acceptance is committed first and its paperwork
is issued afterwards, best-effort. If that issuance does not produce a rate
confirmation, dispatch gets a high-priority `rate_confirmation_issue_failed`
notification titled *Rate confirmation could not be issued*, carrying the
carrier name and the reason, and deep-linking to the move in the dispatch
console.

The acceptance and the carrier assignment still stand. Only the paperwork is
missing. Common reasons:

- **A dispatcher-managed revision already covers the move** — the auto-issuer
  refuses to overwrite it and reports it as not issued.
- **The move has no active carrier assignment to confirm**, or the move is
  covered by a different carrier than the one that accepted.
- **Deterministic issuance failures** such as unconfigured document templates.

Transient infrastructure failures are not notified here — they propagate and
the workflow activity retries into the issuer's idempotent state machine.

### Manual recovery

From the move's rate confirmation actions (carrier assignment panel):

1. **Generate** (or **Regenerate**) the rate confirmation.
2. **Send** it to the carrier's rate confirmation contacts.
3. Record the carrier's confirmation with **Confirm**, or **Void** the revision
   if it should not stand.

The same actions are available over REST:

| Action | Endpoint | Permission |
| --- | --- | --- |
| List for a move | `GET /api/v1/shipment-moves/{moveID}/rate-confirmations/` | `RateConfirmation:Read` |
| Generate | `POST /api/v1/shipment-moves/{moveID}/rate-confirmations/` | `RateConfirmation:Create` |
| Get | `GET /api/v1/rate-confirmations/{rateConfirmationID}/` | `RateConfirmation:Read` |
| Send | `POST /api/v1/rate-confirmations/{rateConfirmationID}/send/` | `RateConfirmation:Update` |
| Mark confirmed | `POST /api/v1/rate-confirmations/{rateConfirmationID}/confirm/` | `RateConfirmation:Update` |
| Void | `POST /api/v1/rate-confirmations/{rateConfirmationID}/void/` | `RateConfirmation:Update` |

### `rate-confirmation-issue-sweep` (every 30 minutes)

`RateConfirmationIssueSweepWorkflow` self-heals the common case, so a
notification that arrived because of a temporary condition usually resolves
itself before anyone acts on it.

The sweep selects accepted tenders where:

- acceptance happened at least 900 seconds ago — the grace window keeps the
  sweep clear of the owning workflow's own issuance retries;
- acceptance happened within the last 7 days;
- the move carries a non-canceled carrier assignment for the accepting
  carrier;
- and the move has **no non-voided rate confirmation**.

For each row (batch of 100) it re-runs the same idempotent issuer the
acceptance path uses, logging and continuing on failure. Overlap policy is
`SKIP`. A run reports `Checked`, `Issued`, and `Failed`.

**What the sweep will not touch:** any move that already carries a non-voided
rate confirmation. A dispatcher-managed revision — generated by hand, sent, or
already confirmed — is excluded by the query itself, so the sweep can never
overwrite or supersede it. That case stays a manual decision; if the
dispatcher-managed revision is wrong, void it and the next sweep run will
issue the tender revision.

The sweep also cannot help a move with no matching carrier assignment. Those
need the assignment fixed first.

## Token Lifecycle

Two kinds of hashed, single-use credentials reach external carriers by email.
Only the hash is stored; the raw token exists only in the emailed link.

### Offer tokens

- **Minted** when an offer is dispatched over the Email channel, one per
  dispatch, addressed to that offer's recipient.
- **Expiry** equals the offer's own expiry — `sent_at + offer TTL` — so an
  offer link dies exactly when the offer does.
- **Link shape**: `/tender-offer/<token>` for the preview, plus
  `/tender-offer/<token>/accept` and `/tender-offer/<token>/decline` for the
  intent links used in the email buttons.
- **Single use**: a recorded response marks the token used. The response is
  funneled through the workflow's CAS gate *first* and the token is burned
  *after*, so a transient failure never eats the carrier's only link.
- **Revoked** when the offer leaves play: accepted, declined, expired,
  withdrawn on cancellation, superseded by another carrier's acceptance, or
  marked delivery-failed.

### Rate confirmation signing tokens

- **Minted** when a rate confirmation is sent, addressed to the recipient.
  Sending revokes every prior live link for that agreement, so exactly one
  current link stands.
- **Expiry** is 14 days — long enough for a carrier's back office, short enough
  that a stale link in a forwarded inbox dies on its own.
- **Link shape**: `/rate-confirmation/<token>`.
- **Single use**: confirming writes the signature first and burns the token
  after.
- **Revoked** when the agreement dies — voided, superseded by a new revision,
  or invalidated because the carrier assignment was canceled. A voided
  agreement invalidates outstanding links even if revocation was missed.

If the tendering public base URL is not configured, an emailed offer cannot be
delivered — the link would be unreachable — so the offer is marked
`DeliveryFailed`, its tokens are revoked, dispatch gets a
`tender_delivery_failed` notification, and the plan moves to the next carrier.
Rate confirmations degrade more gently: the email still goes out with its PDF
attached, just without a signing link.

### Retention purge (daily)

`PurgeTenderTokensWorkflow` runs once a day, overlap policy `SKIP`, and
deletes tokens that can no longer authorize anything, 30 days after they died.
"Died" is measured from the terminal timestamp — used, revoked, or expired,
whichever applies — not from when the token was minted. Live tokens are never
eligible. Both token tables are purged in batches of 500 with activity
heartbeats; the run reports `offerTokensPurged` and `signTokensPurged`.

### Throttling on the public endpoints

Both public surfaces rate-limit **per token**, on top of the coarse global
per-IP limiter, so one leaked link cannot be hammered from many addresses:
10 requests per minute with a burst of 5. Over the limit the endpoint returns
`429` with `Retry-After: 60`. Idle token buckets are evicted after 30 minutes,
and the bucket table has a hard cap of 10,000 entries with oldest-first
eviction so the limiter itself cannot be used to exhaust memory.

Every invalid-token mode — missing, expired, used, revoked, voided — returns
one identical message, so nothing about a token is enumerable. See
[public-web-hardening.md](public-web-hardening.md) for the full public-surface
hardening posture.

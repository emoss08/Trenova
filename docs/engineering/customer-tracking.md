# Customer tracking

A customer follows a shipment two ways: a **tracking link** that opens a public page showing
only the fields the customer's settings allow, and **status emails** (picked up, delayed,
delivered, proof of delivery ready) sent by per-customer rules, exactly once per event. This page
covers the profile that configures both, the links and the public page, the notification ledger
that makes delivery exactly-once, the triggers and sweeps, the GraphQL operations, the agent
tools and the web screens. Read it before changing `customertracking`,
`customertrackingservice`, `customertrackingrepository`, `customerstatustrigger`,
`customertrackingjobs`, `shipmenttrackingpublichandler`, the two `customertracking.*.email`
template kinds, or the public page at `/shipment-tracking/:token`.

## The shape of it

```
Shipment event / stop ETAs refreshed ─▶ customerstatustrigger ─▶ EvaluateShipment (10 s debounce)
customer-status-sweep (10 min) ─▶ every notifying tenant: candidate shipments + due retries
                     │
                     ▼
Detect(facts, profile, now) ─▶ InsertOnce into customer_status_notifications (unique per event)
                     │
                     ▼
deliver ─▶ render customertracking.status.email ─▶ email (IdempotencyKey customer-status-<id>)
        └▶ Record: settles only a row still Pending

Person (GraphQL) / agent tool ─▶ IssueLink ─▶ customer_tracking_links (hash only) ─▶ optional email
Customer browser ─▶ GET /shipment-tracking-links/:token/ ─▶ PreviewByToken ─▶ the allowed fields
customer-tracking-link-retention (24 h) ─▶ delete links dead for 30 days
```

| Piece | Where |
|---|---|
| Domain (profile, link, notification, `Detect`) | `operations/domain/customertracking` |
| Migration | `20261231009240_customer_tracking` |
| Ports | `operations/ports/customertrackingports` |
| Repository (profiles, links, ledger, facts) | `operations/adapters/postgres/customertrackingrepository` |
| Service | `operations/app/customertrackingservice` |
| Triggers | `operations/app/customerstatustrigger` |
| Workflows and schedules | `operations/jobs/customertrackingjobs` |
| Public REST | `internal/api/handlers/shipmenttrackingpublichandler` |
| Email templates | `documenttemplate` kinds `customertracking.link.email`, `customertracking.status.email` |
| GraphQL | `customertracking.graphqls`, resolvers in `resolver/customertrackingresolver` |
| Agent tools | `agent/app/agenttools/shipmenttools/customer_tracking_tools.go` |
| Web | customer "Tracking" tab, shipment "Customer Tracking" tab, `routes/shipment-tracking-public` |

## The profile

One row per customer in `customer_tracking_profiles`; a customer without one reads as
`DefaultProfile` (links on, default fields, no rules, 30-day links).

| Setting | Means |
|---|---|
| `linkEnabled` | Links can be shared; turning it off stops every link already shared, at once |
| `linkTtlDays` | How long a new link lasts, 1 to 180 days |
| `visibleFields` | What the page shows (below) |
| `notifyPickedUp`, `notifyDelayed`, `notifyDelivered`, `notifyPodReady` | The four rules |
| `includeLink` | Each status email carries a fresh tracking link (needs `linkEnabled`) |
| `recipients` | Who gets status emails, at most 20 |

Recipients fall back in order: the profile's list, the customer's status update recipients, then
the To line of the customer's email profile. With nobody to send to, the event is recorded as
skipped (`NoRecipients`) rather than sent later to someone added afterwards.

`notificationsEnabledAt` is stamped by `Normalize` when the first rule goes on and cleared when
the last goes off, and a CHECK keeps the two in step. Detection ignores everything before it, so
switching a rule on never mails the customer about loads that already moved. Updates take the
profile's `version`; a stale one is refused.

### Visible fields

`Status`, `LocationNames`, `Windows`, `Actuals`, `Eta`, `References` (BOL and the customer's
reference), `Carrier`, `Equipment`, `Freight` (pieces and weight), `LastLocation`, `Holds`,
`Notes`, `ProofOfDelivery`. The default is Status, Windows, Actuals, Eta and References. Each
stop's city and state, its type and the PRO number are always shown. `Holds` shows only holds
with `VisibleToCustomer`, and `Notes` only comments marked for the customer; nothing internal
reaches the page by being merely present.

## Links

`customer_tracking_links` stores a link's SHA-256 hash, never the token (`tokenutils.New`); the
URL is the public link base + `/shipment-tracking/<token>` and is returned once, to the
person or agent that issued it. The base is `Config.PublicLinkBaseURL()`: `tendering.publicBaseUrl`
when an installation set it, else `app.webBaseUrl`. Carrier tracking links, tender offers and
public rate confirmations use the same resolver, so one setting serves every link a customer or
carrier opens. A link has a purpose (`Shared` by a person or agent, or
`Notification` when minted for a status email), an optional recipient, an expiry from the
profile's TTL, a revoke time, and a view count with the last view.

Issuing refuses a canceled shipment, a customer with links off, and an environment without a
public base URL (or without email, when an address is given). `PreviewIssueLink` and
`PreviewRevokeLink` run the same checks without writing. Revoking takes one link or, with no
link id, every live link on the shipment.

A link fails, always with the same "no longer valid" answer, when it is unknown, revoked or
expired, its shipment is gone or now belongs to another customer, or the customer's links are
turned off. The public endpoints are rate limited per token, answer `Cache-Control: no-store`
and `Referrer-Policy: no-referrer`, and resolve the tenant from the link
(`dbscope.WithTenant`), never from the request. The proof-of-delivery download works only when
`ProofOfDelivery` is visible. A daily retention schedule deletes links revoked or expired more
than 30 days ago.

### Preview

`GET /shipments/:shipmentID/customer-tracking/preview/` (shipment read) returns the view a link
sent now would show: the same `PublicTrackingView`, built by the same `buildView` the token path
uses, from the shipment's current state and the customer's profile (or the defaults), with
`expiresAt` set to now plus the profile's TTL. It mints no link, records no view, and answers
`Cache-Control: no-store`. It works while the customer's links are off, so staff can check what
turning them on would show.

## Status emails, exactly once

`Detect` reads the shipment as it stands, not a stream of changes, so asking twice, late or out
of order gives the same answer:

| Event | Happened when | Subject |
|---|---|---|
| `PickedUp` | the earliest departure from a pickup stop | the shipment |
| `Delayed` | a stop ETA needs attention and the truck has not arrived (not after delivery) | each stop |
| `Delivered` | the departure from the last delivery (else its arrival on a closed shipment) | the shipment |
| `PodReady` | a current, unrejected POD document is filed | the shipment |

An event counts only if its rule is on, it happened after `notificationsEnabledAt`, within the
last 48 hours, and not in the future. A canceled shipment owes nothing.

Each owed email is a row in `customer_status_notifications`, unique on
(organization, business unit, shipment, event, subject key), written with
`ON CONFLICT DO NOTHING`. That key is what makes it once: a second detection finds the row and
does nothing. Delivery then:

1. Re-checks the shipment and profile: canceled → `Skipped/ShipmentGone`, rule now off →
   `Skipped/RuleOff`, nobody to send to → `Skipped/NoRecipients`.
2. Renders `customertracking.status.email` (the customer's own template, else the built-in) and
   sends it with idempotency key `customer-status-<row id>`, so the email service's
   `CreateMessageOnce` refuses a second message for the same row even if the first send's
   outcome was lost.
3. Records the outcome only if the row is still `Pending`, so two workers racing on one row
   settle it once.

A failed send stays `Pending` with `next_attempt_at` backing off from 60 seconds, doubling up to
an hour, and becomes `Failed` after 8 attempts. Times in the email are in the stop's timezone.

## Triggers

`customerstatustrigger` joins `shipment_event_observers` (updates, status changes, cancel and
uncancel, move status, arrivals and departures, stop completed) and observes stop ETA refreshes,
and signals a debounced `EvaluateShipment` workflow per shipment. It first asks a one-minute
cache whether the tenant has any rule on, so organizations that use none pay nothing per event.
`customer-status-sweep` runs every ten minutes over every tenant with a rule on: it evaluates
candidate shipments (which also catches PODs, which raise no shipment event) and delivers rows
whose retry is due. The tenant list is a system-scope read listed in the RLS allowlist.

## GraphQL

Queries: `customerTrackingProfile(customerId)` (customer read), `customerTrackingLinks` and
`customerStatusNotifications(shipmentId)` (shipment read). Mutations:
`updateCustomerTrackingProfile` (customer update), `issueCustomerTrackingLink` and
`revokeCustomerTrackingLink(shipmentId, linkId)` (shipment update). Every change writes an audit
entry. In write coverage, issuing and revoking map to the agent tools; the profile is exempt as
customer configuration.

## Agent tools

| Tool | Does | Tier |
|---|---|---|
| `shipment_share_tracking_link` | issues a link and, given an address, emails it | Propose → Ask first |
| `shipment_revoke_tracking_link` | revokes every live link on a shipment | Propose → Ask first |

Both previews come from the service's preview methods. Sharing is egress `CustomerVisible`, plus
`ExternalRecipient` when it emails someone. Agents may not update shipments unattended, so the
tools are offered to the customer assistant in chat, where a person approves each one, and not
to the unattended customer update desk.

## Web

The customer form's **Tracking** tab edits the profile.

The shipment panel's **Customer Tracking** tab (`shipment-customer-tracking.tsx`) is sections
split by hairlines, not cards:

- **Shared with** names the customer, shows whether links are on and their lifetime, and how
  many details the page shows (the tooltip lists what is shown and what is hidden). **Preview**
  opens the public page, drawn from the preview endpoint, in a full-screen dialog with a
  preview bar in place of the live indicator and none of the link-only actions; **Settings**
  opens the customer's Tracking tab.
- **Tracking links** emails a new link or makes one to copy (the URL is shown once), lists live
  links with recipient, when sent, opens and expiry, revokes one or all, and folds expired and
  revoked links away. Canceled shipments and customers with links off say why nothing can be
  shared.
- **Status emails** draws the four events as a track: sent, retrying, failed, skipped, not yet,
  or off by the customer's rules (`statusEmailSteps`), with the server's last error for a
  retrying or failed email.

The public page at `/shipment-tracking/:token` needs no sign-in and draws only what the view
returns (`routes/shipment-tracking-public`, `TrackingView`):

- a serif headline with one sentence per status (`trackingHeadline`) and the expected arrival
  measured against the appointment window (`minutesOutsideWindow`), with **Add to calendar**
  (an `.ics` from `@trenova/shared/lib/calendar-file`) and **Copy link**;
- the route ribbon: stops spaced evenly, the driven part filled, and the truck placed by
  `journeyOf` between the last departure and the next expected arrival, labelled with the last
  reported location;
- the stops, the shipper's updates and holds, references, and the POD download;
- a glow and truck halo in the status's tone, refreshing every two minutes with a live
  indicator.

The mock API serves both surfaces (`client/apps/web/e2e/mock-api/handlers/customertracking.mjs`:
tokens `in-transit`, `delayed`, `delivered`, `pod-ready`, `planned`, `canceled`, `minimal`), and
`e2e/mock-api/customer-tracking-screenshots.mjs` captures every state.

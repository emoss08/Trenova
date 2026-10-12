# Triage redesign notes

Triage replaces `/inbox`. The spec is `design_handoff_triage/README.md`; this file records how it
was built, where the design and the server's rules disagreed, and how the queue performs.

## Shape

```
provider webhook ──▶ inboundhandler ──▶ inboundmessageservice.ReceiveWebhook
                                         │  thread key from Message-ID / In-Reply-To / References
                                         ▼
                              InboundMessageWorkflow (Temporal)
                                         │  classify (+ facts, + sender hints)
                                         │  match
                                         │  settle ──▶ TriageRuleEvaluator.ApplyRules
                                         │  conflicts (background activity)
                                         ▼
                              inbound_messages (lane = status + parked)
                                         │
       GraphQL ◀── loaders (owner, thread count, attachments, latest reply, shipment, customer)
       SSE     ◀── realtime invalidations (resource inbound_message) + triage presence scope
```

### Where things live

| Concern | Package |
| --- | --- |
| Message-level types (snooze, parking, facts, conflicts, action state, load draft) | `internal/modules/documents/domain/inboundmessage` (`triage.go`, `queue.go`) |
| Rules and the one evaluator, actions, replies, templates, targets, settings, views, activity, sender hints | `internal/modules/documents/domain/triage` |
| Ports shared across the triage services, and the types GraphQL aggregates bind to | `internal/modules/documents/ports/inboundmessageports/triage.go` |
| Queue core: lanes, due, owner, snooze and wake, waiting on them, threads, views, targets, settings, source health and test, hand-off | `inboundmessageservice`, `jobs/triagewakejobs` |
| Rules, preview, suggestions, corrections, sender hints, facts, conflicts, briefing | `triageruleservice`, `triagebriefingservice`, `triagecorrectionservice` |
| Replies and templates, Nova drafts, loads, filing and splitting, exceptions, blocking, stops, peeks, presence | `inboundreplyservice`, `triageworkservice`, `triagepeekservice`, `inboxpresencehandler` |
| GraphQL | `schema/inboundmessage.graphqls` (queue core), `schema/triage.graphqls` (rules, briefing, corrections), `schema/inboundreply.graphqls` (replies, work, peeks, actions) |
| Migrations | `postgres/migrations/20261231008800_triage.*` and its SQLite twin |

### Decisions

- **One lane model.** A message's lane is derived from `status` and `parked` (`Snoozed` or
  `AwaitingReply`), never stored twice. Waiting = `InReview|Quarantined` and not parked.
  `InboundMessage.IsWaiting/IsSnoozed/IsAwaitingReply` are the only definitions; the list query,
  `CountBreakdown` and the briefing use the same predicates.
- **One action derivation.** `triage.Actions` turns a message's state into its checklist. The
  reader, the floating action bar, the list row's next step and the briefing deck all read it,
  through the `actions` field or `triage.GroupFor`. The client renders copy only.
- **One rule evaluator.** `triage.Rule.Covers/Matches` and `triage.FirstMatch` decide a match;
  processing, the 30-day preview and the suggestions all call them (the preview narrows in SQL
  on kind, sender domain and source, then confirms each candidate with the same function).
- **Due.** `due_at = received_at + target(kind)`; targets are per organization with the
  README's defaults. A correction or a target change recomputes `due_at` for open messages.
- **Threads.** `message_key` is the normalized Message-ID. At ingest the thread key is the key
  of the first known message (inbound or one of our replies) named by In-Reply-To or References,
  else the message's own key. Collapsing in a folder keeps the newest message of each thread
  that is in the folder (`NOT EXISTS` probe on `(org, bu, thread_key, received_at)`).
- **Snooze and waiting on them** wake through a per-message Temporal timer workflow
  (`triage-wake:<message id>`) and, for events, through `TriageEventSink`, which the operations,
  documents and billing modules call when a shipment delivers, a document is attached or an
  invoice is paid. A reply on the thread wakes a message waiting on them at ingest. An hourly
  reconcile closes anything a lost timer missed. Nothing polls the list.
- **Briefing** is one aggregate per tenant cached in Redis for a short TTL and dropped on every
  queue write (`TriageBriefingInvalidator`, called from `TriageMessageNotifier`).
- **Activity trace** is `inbound_message_activities`; every write records its line in the same
  request, through `TriageActivityRecorder`.
- **Authorization.** Reading is `inbound_message:read`; working a message (assign to self, snooze,
  reply, file, correct, settle) is `inbound_message:update`; assigning to someone else and
  handing off is `inbound_message:assign`; rules, templates, response targets and settings are
  `inbound_message:manage`.

## Where the design and the server disagree

_Filled in as each area lands._

### Replies and work

- **Replies carry no attachments.** The prototype's "stand by" reply attaches the gate and ELD
  record. `sendInboundReply` sends text only (the email port has no attachment path for a reply
  that must thread under the sender's Message-ID), so Nova's stand-by draft cites the arrival,
  check-in and departure times and their source instead, and offers to send the record.
- **Send status arrives later.** Email goes out through the email workflow, so the mutation
  returns the reply as `Sending` unless the provider answered at once. `Sent` or `Failed` (a
  bounce, complaint or suppression included) lands through an email observer
  (`emailports.EmailMessageObserver`) and reaches the list as a realtime invalidation. The
  design shows "Sent" immediately.
- **A refused recipient suggests an alternate only when one is known**: the most recent other
  address at the same domain that wrote in or was replied to in the last 90 days. With none,
  `suggestedAddress` is null and the client should offer the address field.
- **Creating a load needs what a shipment needs.** The prototype calls an empty field "fine for a
  draft"; `createInboundLoad` needs a customer (or the message's matched customer), origin,
  destination, and pickup and delivery windows, and each one that does not resolve to exactly one
  record comes back at `fields[i].value`. Service and shipment type are taken from the customer's
  latest shipment, or the only active service type; with neither, the error asks for the shipment
  form. `Loads` (a multi-load tender) is kept on the draft but one call creates one shipment.
- **OS&D is a shipment hold** with the hold reason coded `OSD` (blocks billing). A tenant without
  that reason gets a business error naming it; there is no separate OS&D record.
- **Stop times have no check-in.** Detention occurrences record arrival, departure and the
  appointment window, so `checkedInAt` is always null; "Arrival source" comes from the arrival
  evidence (ELD, geofence, driver app, ...), "Not recorded" without one.
- **Queue for pay needs a reading.** A carrier invoice is queued into settlement's invoice intake
  only when the message matched a carrier and the document has an AI extraction (number and
  total come from it); otherwise it is filed and left for settlement to pick up by hand.
- **Underlying permissions apply.** On top of `inbound_message:update`, sending a reply also
  needs `customer_communication:create`; creating a load `shipment:create`; filing
  `document:update` (and `document:create` to split); logging an exception `shipment_hold:create`;
  deciding stops `detention_policy:update`. Peeks hide a shipment or customer the viewer cannot
  read rather than failing the message.
- **Sending does not park the message as waiting on them.** The design moves a replied message
  to "Waiting on them" automatically; the queue core parks a message only when asked, so the
  client sends the park after a successful send if the reply asks a question.

## Performance

_Filled in after profiling with a seeded tenant of 50,000 messages._

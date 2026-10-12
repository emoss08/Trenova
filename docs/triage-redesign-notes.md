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

## Performance

_Filled in after profiling with a seeded tenant of 50,000 messages._

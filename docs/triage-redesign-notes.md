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

- **Replies carry stored documents, not loose files.** A reply attaches documents:
  `inboundReplyAttachables(messageId)` lists the message's own files and the documents on its
  shipment, customer and carrier, grouped by `source`, and "Upload from computer" goes
  through the documents upload route first (onto the matched shipment, or the message itself
  when nothing is matched), so every file a reply sends is also on file.
  `sendInboundReply(input.documentIds)` and `saveInboundReplyDraft(documentIds)` resolve them
  through `documentports.DocumentMailAttachments` (tenant, sendable status, the person's
  read access to each document's record, at most 20 files and `emailports.MaxAttachmentBytes`
  in all, which `inboundReplyAttachmentLimit` returns for the composer's total), and the
  email goes out with the files by storage key beside the threading headers. The reply keeps
  a snapshot (`inbound_replies.attachments`: document, name, type, size) so a later rename
  does not change what it says it sent, the `Replied` activity's detail lists the file names,
  and a retry to another address sends the same documents (the composer keeps them and
  sends them again). Leaving `documentIds` out of a draft save keeps the draft's files.
- **Nova attaches the detention record when standing by.** `decideInboundStops(StandBy)` asks
  operations for the dispute's record (`TriageDetentionEvidenceSource`): the PDF of a
  detention notice that was sent (a Final one first), else a record stored earlier, else the
  Final notice rendered from the organization's `detention.notice.pdf` template and stored on
  the shipment as a `DETNOTICE` document, then attaches it to the draft. Its prompt lists
  exactly what is attached and says nothing is when nothing is, so the text never claims a
  file that is not there; without a record it offers to send the times instead. There is no
  separate gate log or ELD file in Trenova, so the record is the notice, which carries the
  arrival (with its source), departure, free time and billed time. Other drafts attach
  nothing: "Return it signed" is left to a person, because the file on the message is the
  unsigned rate confirmation.
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
- **Waiving a billed detention charge credits the invoice.** The detention domain still refuses
  to waive a billed occurrence ("issue an adjustment instead"), so `decideInboundStops(Waive)`
  asks billing first, through `detentionports.BilledDetentionCreditor` (declared in operations,
  provided by `billing/app/detentioncreditservice` from `billing/module.go`). Billing reads
  what the standing invoices still bill of the occurrence's shipment charge
  (`NetBilledByCharge`):
  - Nothing billed (the charge was never invoiced, or every invoice that billed it was voided
    or credited): the occurrence is waived as before. One still marked `Billed` with no invoice
    behind it (older data, the dev seed's `SEED-DET-008`) is released and waived in one write
    (`detentionservice.WaiveUninvoiced`), with both steps on its evidence chain.
  - Billed: the person needs `invoice:update` (the permission the invoice adjustment routes
    check), otherwise the mutation is refused with an authorization error. For every posted,
    non-credit-memo invoice with a line for the charge, billing submits a `CreditOnly`
    adjustment through `InvoiceAdjustmentService.Submit` for what is left to credit of those
    lines, with the organization's adjustment policy applied unchanged: when the policy needs
    an approver it lands `PendingApproval`, exactly as a person's submit from billing would.
    The adjustment carries the waiver (`metadata.detentionWaivers`, never bound from a request
    body); when it executes, now or on approval, the detention billing sync waives the
    occurrence instead of only releasing it, so the charge ends `Waived` with its reason and
    amount, not `Approved`. The idempotency key is per occurrence, invoice and attempt, so a
    second click returns the credit already waiting, and a rejected or failed credit lets the
    next attempt submit a new one.
  - Refused: a draft invoice, a preview error (policy, period, attachments) or a failed
    execution comes back as a field error on `decision`, "Invoice INV-… could not be credited
    for the waived detention: …", and nothing on the message changes.
  The message records what happened in `actionState.detentionWaiver` (amount, invoices, credit
  memos, pending) and the activity detail ("185.00 USD, credited on INV-20931 by CM-00012", or
  "… credit on INV-20931 waits for approval"). Nova's reply follows it: removed before
  invoicing, credited with the credit memo to follow, or a credit being processed (never
  claimed as issued). Standing by is refused while the credit waits for approval, and deciding
  the same way twice changes nothing unless a credit is still pending. The design's "a
  corrected invoice follows" is a credit memo here; a credit-only adjustment issues no
  replacement invoice.
- **Standing by keeps the dispute while the invoice still bills it.** `decideInboundStops(StandBy)`
  disputes the occurrence, and a dispute is a person's decision that the invoices do not
  override. The detention billing sync used to call `MarkBilled` on any occurrence an invoice
  carried that was not `Billed`, so a stood-by charge failed every later sync on that invoice (a
  credit on another line, a void, a rebill); its occurrence query also skipped `Disputed`, so
  waiving after standing by credited the invoice but left the occurrence disputed. The domain now
  decides it: `DetentionOccurrence.FollowInvoices(invoiced)` moves only a billable charge
  (pending/approved to billed, billed back to approved) and leaves accruing, disputed, waived and
  not-billable ones as they are, idempotently; the sync lists `detention.InvoiceSettledStatuses()`
  (billable plus disputed) so a credit carrying a waiver still waives a disputed charge. Whether
  a disputed charge is invoiced is read from the invoices (`NetBilledByCharge`), as the waiver
  path already does, rather than stored on the occurrence. A credit with no waiver that takes a
  stood-by charge off the invoice leaves the occurrence disputed: the dispute stays open until a
  person waives or settles it, and nothing waives it automatically (product decision).

### Queue core

- **Teaching a rule from a bulk handle needs `inbound_message:manage`.** The design offers "always
  do this for this sender" beside Handle; a rule changes what Nova does for everyone, so the
  `teach` half of `reviewInboundMessages` is refused without manage while the handle itself only
  needs update.
- **A snooze keeps the message's due time.** Waking (by time, event or a person) returns the
  message with its original `dueAt`, so something snoozed past its target comes back overdue.
  `responseTargetMinutes` is the organization's target for the kind; a settled message has no
  `dueAt`.
- **Search is a substring match** on subject, sender address and sender name (the existing
  `querybuilder` behaviour), not the body: "S-10" or "acme" match inside a PRO or a domain, which
  the stored `search_vector` (word tokens; an address is one token) would not.
- **Seeded dev data.** The prototype's Claims dispute arrives on Claims within the day while
  Claims has "received nothing in 7 days"; the seed puts the FreshHaul dispute on Intake and
  leaves Claims' last message 7 days 2 hours old. Readable seeded attachments have a `documents`
  row, a `document_contents` reading (and a shipment draft with a rate for rate cons, an AI
  extraction for the carrier invoice), so filing, splitting, conflicts and load drafting have data;
  the files themselves are not in object storage, so preview and download fail for them (each
  row's description says so). The prototype's shipments and parties are mapped onto seeded ones:
  the detention dispute is on `SEED-DET-008` (FreshHaul, a billed $1,200 detention with geofence
  arrival and departure), the overlapping tender load repeats the lane and pickup day of the open
  `SEED-SHP-001`, the carrier is Ironhorse Freight Lines, the driver is the seeded worker Carlos
  Rivera, and Blue Ridge Brokerage is added as a broker customer (broker vetting on). The seed
  also adds the `OSD` hold reason exception logging needs, routes `InboundClassification` to the
  local model so Nova reads as on, and adds an address rule (FreshHaul claims assigned to Dana)
  next to the prototype's four. The second saved view searches "SEED-" rather than "S-10".

### Reader

- **"Download original" is "Download as .eml".** The provider's copy of a message is kept in
  object storage (`html_key`) and no route serves it, so the reader's menu writes the stored
  headers and plain text as an RFC 5322 file in the browser (`lib/triage/eml.ts`) and names the
  item for what it is.
- **Assignees carry no role.** The prototype's assign menu shows each teammate's role
  ("Billing", "Dispatch"); `triageAssignees` returns the user and how many messages they have
  waiting, so the menu shows that count, and "Has it now" on the current owner.
- **"Not phishing" opens the link dialog.** There is no mutation that releases a held message
  back to waiting; as in the prototype, the button asks which record the message is about.
  "Ignore" settles it as phishing with that note.
- **An exception's count and unit are read client-side.** `LogInboundExceptionInput` needs a
  kind, a quantity and a unit; the action carries only the kind (and the mock leaves it null), so
  the step reads "2 pallets short" from the message's facts or the document reading and lets the
  person correct it before logging.
- **A reply the message did not ask for has no step.** R on a message whose actions include no
  `Reply` (a POD, the EDI status file) opens a composer appended to the checklist, unnumbered by
  the server, so the prototype's "reply to anything" still works.
- **A settled message lists only the steps that were done.** The server keeps deriving open
  steps for a handled message (a status request still "asks" for an ETA); the reader hides the
  open ones once the message is settled.
- **Document types are matched by name.** Select options for `DOCUMENT_TYPE` carry the code as
  the label and the name as the description; the reader shows and matches the name
  (`lib/triage/document-types.ts`), since the reading's `documentType` and a page's
  `suggestedType` are names.

### Briefing, rules and overlays

- **The ring is how much of today is cleared.** The briefing returns what is waiting and the
  points you cleared, not a target or a start-of-day count, so the ring fills to
  cleared / (cleared + waiting) and reads full on a day with nothing to do.
- **A new source has no address of its own until someone types one.** The server has no inbound
  domain setting. The connect flow suggests `name@domain` from the domain most of the
  organization's other sources receive on and asks for the whole address on a first source.
  It creates the mailbox to review anything below 0.8 confidence. For Postmark it generates
  credentials and stores the signing secret; Resend's signing secret only exists in Resend, so
  that step links to the source's settings to paste it. The webhook URL comes from the same
  helper the admin page uses.
- **The document viewer highlights by text, not by box.** A reading's fields carry values but no
  page coordinates, so the viewer marks matches in the PDF's text layer. A scanned page with no
  text layer shows the page without marks, and the list of read fields still shows.
- **Desk hand-off chips sit above the thread.** `DeskThread` builds its own user bubble, so the
  open actions the message carries are a strip above the conversation instead of chips inside
  the first message. The message and its shipment go in as hand-off mentions, so the agent sees
  the same context. The agent is picked by matching the message's kind to each agent's name,
  template, description and tools, falls back to the last agent used, and can be changed in the
  header.
- **The hand-off line is gender-neutral.** The prototype's "he'll see them first thing" names a
  pronoun the server does not know; the copy reads "they'll".

## Performance

Measured on the development database with 50,055 messages in one tenant (50,000 synthetic across
three mailboxes over 120 days: 7% waiting on a person, 0.5% snoozed, 0.5% waiting on them, 70%
handled, 22% ignored; ~3 messages per thread; 100,000 activity lines), PostgreSQL as superuser
(so without the RLS predicates, which are equalities on the same leading columns),
`EXPLAIN (ANALYZE, BUFFERS)`, page size 50 (`LIMIT 51`), three runs each, median shown. The
synthetic rows were deleted afterwards (50,000 rows and their activity lines cascade in ~2 s).

| Query | Plan | ms |
| --- | --- | --- |
| Waiting, by due, page 1 | Index Scan `idx_inbound_messages_waiting_queue`, stops at 51 | 0.4 |
| Waiting, by due, page 2 (keyset) | same index, cursor predicate in the scan | 0.3 |
| Waiting, threads collapsed | `waiting_queue` + `NOT EXISTS` probe on `idx_inbound_messages_thread_key` | 1.1 |
| Everything, newest | Index Scan `idx_inbound_messages_received` | 0.4 |
| Everything, newest, collapsed | `received` + thread-key probe | 1.0 |
| Handled, newest | Index Scan `idx_inbound_messages_lane` | 0.3 |
| Assigned to me, by due | Index Scan `idx_inbound_messages_owner` | 1.6 |
| Snoozed, newest | Index Scan `idx_inbound_messages_parked` | 0.3 |
| Kind folder (Invoice), waiting by due | `waiting_queue` with a kind filter | 0.4 |
| Mailbox folder (Billing), everything | `received` with a mailbox filter | 0.4 |
| Search "invoice", everything newest | `received`, filter until 51 match | 1.3 |
| One thread / thread reader | Index Scan `idx_inbound_messages_thread_key` | 0.2 |
| Thread counts for a page (50 keys) | Index Only Scan `thread_key` | 0.5 |
| Activity for a page (50 messages) | Index Scan `idx_inbound_message_activities_message` | 1.1 |
| Census (`CountBreakdown`) | open lanes: Bitmap on `idx_inbound_messages_census`; settled: Index Only Scan `census` | 21 |
| Saved views' waiting counts (2 views) | Bitmap on `census`, waiting rows only | 22 |
| Wake reconcile (all tenants) | `snoozed_until` + `awaiting_until` partial indexes | 0.6 |
| Total count, waiting | Index Only Scan `waiting_queue` | 0.9 |
| Total count, everything / handled / mailbox | Index Only Scan `census` | 7 / 4 / 3 |
| Total count, waiting collapsed | hash anti join over the waiting rows | 12 |
| Total count, everything collapsed | parallel seq scan + hash anti join | 89 |
| Total count, search | seq scan (`ILIKE` on five columns) | 133 |

What changed after the first run (`20261231008890_triage_queue_indexes` and its SQLite twin):

- **Waiting by due was a sort of every waiting row** (bitmap scan of `idx_inbound_messages_lane`,
  top-N heapsort of 3,508 rows, 14 ms; `idx_inbound_messages_waiting_due` leads with `status`, so
  two statuses cannot give one due order). It is now
  `idx_inbound_messages_waiting_queue (org, bu, due_at ASC NULLS LAST, received_at DESC, id DESC)
  WHERE status IN ('InReview','Quarantined') AND parked IS NULL`, which matches the sort exactly:
  0.4 ms and constant in the size of the queue. It replaces `waiting_due`. The collapsed waiting
  folder went from a parallel seq scan (32 ms) to the same index plus a probe (1.1 ms).
- **The census was a seq scan of the tenant** (47 ms) grouping every row by six expressions. It is
  now two branches under `UNION ALL`: open statuses grouped by lane, owner and lateness (bitmap on
  the small open set), and Actioned/Ignored grouped by status, kind and mailbox only, read from
  `idx_inbound_messages_census (org, bu, status, classification, mailbox_id)` without touching the
  heap (0 heap fetches). 21 ms, of which ~14 ms is aggregating the 46,000 settled index entries.
  The same index answers the gated `totalCount` of the settled folders.

Known linear costs, all behind the gated `totalCount` (the list itself stays index-bound):

- **Collapsed "Everything" total** counts the newest message of every thread in the tenant
  (89 ms at 50k). The client should take the folder's number from `inboundMessageCounts` and only
  select `totalCount` on a collapsed folder when it shows it.
- **Search total** evaluates `ILIKE '%term%'` on every row of the tenant (133 ms at 50k); the page
  itself stops at the 51st match. A trigram index per searched column, or moving search to the
  stored `search_vector` with the sender's domain as its own token, would make it index-bound;
  both change `querybuilder` behaviour shared by every list, so neither is done here.

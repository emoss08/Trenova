# Desk cases

A Desk conversation about a shipment, an invoice or an invoice dispute is a
**case**. It has a lifecycle state, can be snoozed, can wait on the customer or
a carrier, and shows what stands between its record and what comes next.

Read this before changing `domain/deskcase`, `assistantcaseservice`,
`assistantcaserepository`, the `/assistant/threads/:threadID/case/` routes, or
the Desk's `routes/desk/_components/case/`.

## Binding

A case is a thread whose `subject_type`/`subject_id` names a `Shipment`,
`Invoice` or `InvoiceDispute` (`deskcase.IsSubject`). A thread opened from a
record is a case from the start; any other Desk thread can be bound later
(`PUT …/case/`), moved to another record, or unbound (`DELETE …/case/`).
Binding checks that the id has the kind's prefix (`SubjectType.CheckID`), that
the person may read that kind of record (`subjectaccess`), and that the record
is the organization's (`AgentSubjectRepository.Exists`). A page-bound thread
(import, formula) cannot be rebound. Moving or unbinding clears the snooze,
which was set against the old record.

## State

State is never stored. `deskcase.Resolve` works it out on every read from the
record, the conversation's open waits and its snooze, strongest first:

| State | When |
|---|---|
| `Settled` | the record closed: a shipment `Invoiced` or `Canceled`, an invoice `Paid` or `Voided`, a dispute `Resolved` or `Withdrawn` |
| `Snoozed` | the snooze ends in the future |
| `Waiting` | the conversation has an open wait; `waitingOn` says on whom (customer, carrier, a reply about the record, or another event) |
| `Working` | anything else |

Because nothing is stored, a payment that announces nothing still settles its
invoice's case the next time anyone looks. A shipment settles when it is
invoiced or cancelled rather than when it delivers, because the ready-to-bill
work happens after delivery.

`AssistantCaseStates.Attach` sets `Thread.Case` on every case in a page of the
thread list in a fixed number of queries: one per kind of record
(`ListRecords`), one for the open waits (`ListOpenByThreads`), and one for ETAs
only when a case is snoozed to one. `GetThread` and `ListThreads` both attach.

## Snooze

`POST …/case/snooze/` takes an anchor:

- `Time`: until a time, at least a minute and at most 30 days ahead.
- `Appointment` (shipments): the next stop not yet reached
  (`snooze_stop_id`). Read again on every read, so a moved window moves the
  snooze, and the snooze ends when that stop is reached.
- `ETA` (shipments): the shipment's estimate from `ShipmentEtaReader`, read
  again on every read; `until` is kept as the fallback while there is none.

`POST …/case/wake/` ends a snooze. A wait that picks the conversation back up
also ends it (`agentwaitservice.wakeCase` → `WakeThread`): what the person put
it away for has arrived.

## Waiting on someone

`POST …/case/await-reply/` registers a `Reply` wait (see
[agent-waits.md](agent-waits.md)) on the record's customer or one of its
carriers. The party must be the record's own. The reply, matched to that party
by `inbound_message.classified`, ends the wait and starts a turn that picks the
case back up. The agent sets the same wait with `wait_until`, which is why the
route's write coverage names that tool.

## Readiness checklist

`GET …/case/` returns the summary, the parties the case can wait on, and a
checklist built by pure functions in `deskcase/checklist.go`:

- **Ready to bill** (shipments): delivered, proof of delivery received, the
  rest of the paperwork in, the rate matched to the rate confirmation (rate
  validation and the billing queue's open charge findings), each carrier's
  rate confirmation confirmed, accessorials approved (no detention charge pending
  or disputed), the customer emailed since delivery, nothing holding the bill
  (`credit_hold`, `unresolved_service_failures` …). Facts come from
  `GetBillingReadiness` and one query (`ShipmentFacts`).
- **Ready to close** (invoices): posted, sent, clear of dispute, paid. Payment
  not yet due is pending; past due it blocks.

Each item is `Done`, `Blocked` (something to do now), `Pending` (waiting on the
world) or `NotNeeded`. Blocked and pending items keep the record from being
ready. `next` is the first blocked item's step or, when nothing blocks, what the
record is ready for (`mark_ready`, `send_invoice`). The Desk words each step and
sends it to the case's agent as the person's own message.

## Customizing the checklist

`case_checklist_templates` holds how an organization lays out a kind of
checklist (`ReadyToBill`, `ReadyToClose`): the steps in order, each
`Required`, `Optional` (shown, never blocks, never the next step) or `Off`
(not shown), and steps it added. A customer's template replaces the
organization's; with neither, `DefaultItems` applies. The template that
applies is the **bill-to** customer's (a shipment's bill-to, else its
customer), read in one query (`TemplateFor`).

- **Locked steps** follow a rule kept elsewhere and stay required, though
  they can be moved: POD and paperwork (the billing profile's document
  types), the rate check (rate validation), delivered, billing holds,
  posted, dispute and paid. `LockedKeys` tells the settings screen which.
- **Added steps** (`custom:<key>`) are ticked by a person on the case
  (`Manual`, stored per record in `case_checklist_ticks`, so everyone
  working a case about the record sees it) or by an accepted document of a
  type being on file (`Document`, shipments only). Each carries its own
  button text and the request its button sends to the agent.
- `Normalize` brings a saved template up to date: a built-in step added
  since joins at its default position, a retired one leaves.
- Settings: `GET/PUT /case-checklists/`, `DELETE /case-checklists/:id/`
  (billing control read / update), audited under billing control, saved at
  the version read (`ErrChecklistTemplateStale` is a 409). The page is
  Billing → Configuration files → Case checklists.
- Ticks: `POST /assistant/threads/:threadID/case/ticks/`, only for a
  `Manual` added step of the template that applies.

## Realtime

- `assistant_case`, addressed to the owner, when a case is bound, moved,
  snoozed or woken.
- `agent_waits` already moves a waiting case.
- A change to the record itself reaches the case through the organization's
  ordinary events: `lib/case-realtime.ts` refetches the case and the rail only
  when an event's record (or the shipment, invoice or document it names) is one
  the person's cases are about. Payments now announce the invoices they apply
  to, and a detention hold names its shipment.

## The Desk

The top bar's case button is the shared popover. It binds a conversation to a
shipment, invoice or dispute from a picker that pages through every matching
record (`GET /assistant/mentions/?kind=&offset=&limit=`, one kind at a time,
one row past the page saying whether more follow), loads the next page as the
list nears its end, and draws only the rows in view (TanStack Virtual; the
file opts out of the React Compiler like the other virtualized lists). Nothing
is read until the popover opens. Disputes are a search kind only the picker
asks for, so the composer's @ search across all kinds leaves them out. The
same menu snoozes, waits on a party, wakes, moves and unbinds. The card above the
composer shows the record, its state and the checklist with the next step. The
rail shelves snoozed and settled cases below the calendar shelves and wakes a
snooze on time without polling.

## Not covered

Freight claims have no domain in Trenova yet, so a case cannot be about one.
Adding a claims domain adds a `SubjectType`, a record reader in
`assistantcaserepository` and, if it has one, a checklist.

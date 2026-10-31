# Nova harness

Nova does two kinds of work. The **Desk** is a conversation: an agent with a tool catalog,
guards and a person waiting on each turn (`agentruntime`, see [agent-runtime](agent-runtime.md)).
Everything else Nova does is **a task**: one job with its own instructions, its own task (which
picks the providers and the model) and its own usage feature (where its cost shows). Tasks run
through the harness, `internal/modules/agent/app/novaharness`, never through the Desk runtime,
so the Desk's guards and prompt never shape them.

The harness puts Nova's identity (`identity.go`: who Nova is, that it works inside Trenova, the
rules every task follows) in front of each task's instructions. A task's instructions start
"Your task: …" and say one thing.

## The ports

Modules reach the harness through narrow ports in `llmports`; the agent module provides
`novaharness.Harness` behind each.

| Port | For | Example |
|---|---|---|
| `TaskRunner` (`Run`, `Available`) | one structured answer, decoded into the caller's struct; `ErrOutsideSchema` when it does not decode (the result is still returned). A task may declare `Reads` (see below) | billing queue review, paperwork reading |
| `TaskConverser` (`Converse`) | a question answered from a few reads, streamed: a fixed `[]NovaTool` declared by the task, at most `MaxCalls` model calls, the last told to answer with what it has | command palette answers |
| `TaskBatcher` (`SubmitTasks`, `BatchStatus`, `CancelBatch`, `BatchResults`) | many tasks nobody is waiting on, through the provider's batch API (half the price, results within the batch window); decode each answer with `llmports.DecodeAnswer`. When someone starts waiting, `CancelBatch` stops what has not run (it is not charged) and the batch ends with what had, so the rest can run at once without paying twice | insight narration, the billing queue's bulk reviews |
| `TaskDeferrer` (`Defer`, `Collect`) | one long task a provider runs in the background while an activity polls | rate confirmation extraction |

**Where a call runs.** Inside a Temporal activity the harness calls the completion router
itself. On a request path it hands the call to a worker through the completion dispatcher. A
task therefore never waits on a second workflow for a slot on its own queue.

**Reads.** A `NovaTask` with `Reads` (`[]NovaTool`) may read before it answers. Each model
call is offered the reads and an `answer` tool whose parameters are the task's schema, so a task
whose context settles it still answers in one call; a read costs a call only when the model
asks for it. `MaxCalls` bounds the calls (three by default). When they run out, or a tool answer
does not decode, one structured call answers with the task's context plus what was read, as
untrusted sections. Reads run only on a worker; on a request path the task answers from its
context alone. Show a task what usually settles it and make the rest a read: the billing queue
review sees a short excerpt of each document and reads further parts (`read_document`) or one
charge across more of the payer's invoices (`charge_history`) only when it needs to.

**A task's files.** `NovaTask.Files` are document references; the worker reads the bytes and a
model that reads files gets each as itself, so a task never waits for docling.

## Tasks

| Task | Package | Port | Trigger |
|---|---|---|---|
| Paperwork reading | `agent/app/paperworkreader` | Run | every shipment document upload, and "Read with Nova" |
| Command palette answers | `agent/app/paletteanswer` | Converse | a question typed into the palette |
| Insight narration | `agent/app/insightservice/narrator` | Batcher (sweep), Run (fallback) | the six-hourly insight refresh |
| Billing queue review, brief, search | `billing/app/billingqueueassist` (the review's prompt: `reviewprompt`) | Run (the review with reads) | each recheck; the debounced brief; a sentence in the queue search |
| Factoring remittances | `billing/app/factoringremittanceservice` | Run | a PDF remittance |
| Inbound classification, replies, loads | `documents/app/inboundprocessing`, `triagenova` | Run | each message; a draft asked for |
| Accounting mapping | `accounting/app/accountingmappingscoring` | Run | records the matcher could not settle |
| Document routing and extraction | `agent/app/aidocumentservice` | Run, Deferrer | document intelligence |
| Formula studio | `agent/app/formulastudioservice` | Run | explain, describe, suggest tests, review notes |
| Briefings, chapter titles, insights, narrations, table queries | `agent/app/…` | Run | their own jobs and requests |

The Desk's own machinery (the scope guard, quality judging, drafting, profiles, reflection,
hand-off summaries, artifacts) stays on the Desk runtime.

A task lives in its own leaf package (instructions, schema, the service that runs it and records
the answer) and, when an event triggers it, a job under `agent/jobs/<task>jobs` with a contract
the triggering module calls through a port (`documentports.PaperworkReadRequester`).

## The command palette

A palette question is not a Desk turn. `paletteservice.Ask` starts `PaletteAnswerWorkflow` on
the chat queue, which hosts a Workflow Stream (`jobs/runstream`, the same host a Desk turn uses)
and runs `paletteanswer` through `Converse`. The answer streams the events a turn does, so the
palette card is unchanged, and ends with `done` naming only the question.

- **Tools.** `search_records` (global search, visibility-filtered), `get_record` (an id goes to
  its kind's get tool) and `list_records` (a kind and its filters go to its kind's list tool;
  the description lists each kind's filters, required and fixed-value ones first, paging left
  out). Every read is checked as the person and argument-checked the way the Desk checks a call
  (`argcheck.ContractCall`: aliases, "today", a refusal that names what each argument should
  be), and reads at the agent tier (`SensitivityInternal`), never the person's own when theirs
  is higher, since what Nova reads goes to a model provider. None writes.
- **Ownership.** The workflow id is built from the session's organization, user and the ask
  id, so a reader can only reach their own question.
- **The record.** Each question is a row in `assistant_palette_questions` (no content: who
  asked, how it ended, the Desk conversation it went to). The person's monthly AI allowance
  counts it (`turnlimits.Limits.PaletteQuestions`) unless it was carried into the Desk, where
  its message counts instead; the plan's assistant-message and spend meters are checked before
  it starts (`AssertMayAsk`); and a person has at most `palettequestion.MaxAsking` being
  answered at once. The assistant retention sweep removes rows after two months.
- **Open in Desk.** `Keep` claims the row (so a second press reuses the first conversation),
  reads the workflow's result and starts a Desk conversation with the general assistant that
  opens with the question and the answer. The palette's own reads are not replayed (the Desk
  holds none of those tools); the records they found ride on the question as mentions. A
  question with no answer opens an empty conversation and the client asks it there.
- **Access.** Asking needs a provider for `PaletteAnswer` (it borrows the Desk's until one is
  assigned), not the general assistant; carrying the answer into the Desk needs the general
  assistant.
- **Retention.** The workflow's input and result (the question, the answer's text and the
  records it read) stay in Temporal history for the namespace's retention period, which is
  what `Keep` reads.

## The billing queue review

The five checks run first and the biller sees what they found beside Nova's note, so the note
says only what the checks could not: a charge against the payer's past invoices, the paperwork
against the bill, time on site at the stops, the comments and the payer's billing notes and
terms, and how a charge was priced (the fuel program behind a surcharge, a rate override's
reason). The checks' findings are a section of their own, titled as what the biller already
sees. With nothing to add, Nova leaves the summary empty and the queue shows no note; a Concern
about something no check found must say why (`reviewprompt.Validate`). The review is re-run only
when what it is shown changes: `reviewprompt.Hash` hashes the rendered sections, so a save that
changes nothing Nova reads costs nothing, and a new section is in the hash without a second list.

**What it read.** Each review records, with its note, what it read (`agent_reads`,
`reviewprompt.Reads`): what its sections gave it (the invoice's charges, the stops, the
payer's terms and notes, the comments, the past invoices, each document) and what it read
further, kept by its read tools as it reads (`readLog`: the furthest part of each long
document, each charge looked up with how many of the payer's invoices carried it). A batched
review's reads are what `readAhead` gave it (`readAhead.taken`, from the invoices it already
read for the page). The note's "See what it read" opens them under the note, one line
each for the paperwork, the shipment, the payer and history, and a document named there opens
beside the item. The record is structured, not prose, so the client words it in the
reader's language.

### Bulk generation

A review runs on its own inside each item's recheck, because auto-approve may wait on it. A
queue with items Nova has not reviewed (review just turned on, a prompt change, a provider
outage) or no brief is backlog nobody waits on, so it is generated in bulk by
`BillingQueueNovaGenerationWorkflow`, one per organization (its id is the organization's).
Reading `GET /billing-queue/nova-generation/` (the queue page does on open) starts it when the
queue needs it, and the half-hourly `billing-queue-nova-generation-sweep` starts it for every
organization with a backlog Nova can review; a request while one runs makes it look again when it
is done. A generation checks, as it finishes, whether it left items unreviewed, and reports that
with its progress and as its result (`NovaGenerationProgress`). Opening the queue within ten
minutes of one that left items starts nothing: what it failed on waits for the sweep, because
every reader of a queue whose items keep failing would otherwise start another one the moment it
ends (its end publishes, the client reads, the backlog is still there). After one that finished
everything, whatever is unreviewed is new, and opening the queue starts a generation at once. While items rest, the endpoint answers the `Left` phase with `retryAt`, so the queue says Nova
could not review some items and when it tries again, and reads the endpoint again then. An item whose evidence cannot be read is logged and skipped
rather than failing its page. It pages through the unreviewed items
(`idx_billing_queue_items_nova_backlog`) a hundred at a time, and how it reviews them depends on
who asked (the signal's `NovaTrigger`). Someone who opened the queue is waiting, so their pass
reviews at once, eight items at a time, each review gathering its evidence and taking its reads as
it runs. The schedule's pass waits on nobody, so it goes to the provider's batch API at half the
price: it waits for each batch with a widening timer (a batch can take minutes or hours) and
records the answers as a review on its own would (an answer to evidence that moved, or an item
reviewed meanwhile, is dropped). A batch takes one answer, so a batched review cannot read;
`readAhead` makes its reads before the batch is sent instead (each charge on the invoice across
the payer's last invoices, read once per payer for the page, and the next parts of a long
document) and gives them to it as untrusted context. A provider with no batch API reviews the page one item at a time. Someone opening the queue while
the schedule's pass waits on its batches ends the wait: the batches are canceled, the answers that
had come back are recorded, and a pass that reviews at once takes the rest, with the brief written
after it. A canceled request is not charged, so nothing is paid for twice. Either way an item Nova agrees
with is offered to auto-approve, and the brief is written last.

**Showing it.** The workflow keeps its progress (phase, total, done) behind the `progress`
query; the endpoint reads it with Describe and Query, so a generation that died never shows as
running. Each step publishes the `billing-queue-nova` realtime resource (and `billing_queue` when
reviews or the brief were written); the client's generation query has its own root
(`billingQueueNova`), so the queue's item changes do not refetch it, and reads it again every
30 seconds only while one runs.

## Batches

A sweep's narration batch is submitted by the refresh and waited on by a child workflow it
abandons (`InsightNarrationWorkflow`): a widening timer checks the batch, the ended batch's
wording is checked by the number guard and applied to the findings still active. A provider
with no batch API answers the same requests one at a time. A finding whose figures have not
moved keeps the wording a model already wrote for it, so an unchanged operation costs nothing
to refresh.

A task a person is waiting on (paperwork reading, palette answers, search) never batches.

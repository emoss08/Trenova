# The agent runtime

How AI work actually executes: where it runs, what makes it safe to retry,
what survives a crash, and what is written down afterwards.

Read this before changing anything under `internal/core/services/agentruntime/`,
`internal/core/services/assistantservice/`, or any of the Temporal packages
named below.

## Everything is a workflow

Every AI feature runs on Temporal. The API authenticates the request, records
what it needs to, starts or signals a workflow, and either waits for its result
or relays its stream. Workers do the work. There is no in-request path and no
flag that brings one back: an outage of Temporal is reported as such, and the
client connects lazily, so the API still starts and the first call after
Temporal returns succeeds.

| Feature | Workflow | Queue |
|---|---|---|
| Assistant chat, Ask, decision follow-ups | `AssistantTurnWorkflow`, one per turn | `agent-chat-queue` |
| Import and formula assistants | `AssistantTurnWorkflow`, one per turn, on the page's own thread | `agent-chat-queue` |
| Table compose | `StructuredCompletionWorkflow` | `agent-chat-queue` |
| Briefing regenerate | `WriteBriefingWorkflow` | `agent-chat-queue` |
| AI provider test | `TestAIProviderWorkflow` | `agent-chat-queue` |
| Event-driven and scheduled agent runs | `AgentRunWorkflow` | `agent-background-queue` |
| An agent's schedule firing | `AgentScheduledRunWorkflow` | `agent-background-queue` |
| Agent evaluations (replays) | `AgentEvaluationWorkflow` | `agent-heavy-queue` |
| Reports and dispatch planning called as tools | the tool's activity | `agent-heavy-queue` |
| Insights, daily briefing | a parent plus one child per organization | `system-queue` |
| Document extraction | `ProcessDocumentAIExtractionWorkflow` | document intelligence |
| Inbound email | `ProcessInboundMessageWorkflow` | `system-queue` |
| Accounting mapping suggestions | `RefreshAccountingReferenceWorkflow`, one per connection (`accounting-reference:<connectionID>`), whose model pass is one activity at background priority | `integration-queue` |

The queues are split so one class of work cannot starve another: a person
watching a reply must not wait behind a ten-minute scheduled run, and an
evaluation sweep must not fill the queue a scheduled run needs. A worker polls
every queue unless told otherwise:

```
trenova worker run                                   # every queue
trenova worker run --queues=agent-chat-queue         # only what a person waits on
```

Inside a queue, work is ordered by **priority** and shared out by **fairness**.
Every AI start and activity carries a priority key (interactive 1, one-shot 2,
background 3, evaluation 5) and a fairness key, the organization, so one busy
tenant cannot queue everybody else's replies behind its own. Fairness needs
`matching.enableFairness=true` on the server; without it the keys are ignored
and nothing breaks.

## The agent loop runs in workflow code

`agentflow` drives the same loop the runtime has always had (`agentruntime.Drive`
over a `Turn`) from workflow code, through the `TurnEffects` seam:

- **Each model call is an activity** (`ModelCallActivity`). It streams the
  reply to the run's Workflow Stream when somebody is reading, and heartbeats
  on a ten-second timer, so a model thinking silently is not mistaken for a
  lost worker.
- **Each tool call is an activity** named for the tool: the worker's dynamic
  activity, so the Temporal UI and the SDK's metrics show each tool as itself.
  A tool that fails after its retries is reported to the model as a failed
  call and the turn goes on. `run_report`, `compare_report_runs` and
  `plan_dispatch` run on the heavy queue whichever queue called them.
- `find_tools` and `publish_artifact` are activities of their own. Call ids
  are minted through `workflow.SideEffect`, so a replay reads back the same id.
  An id an adapter made up from the call's position (Ollama, and an
  OpenAI-compatible stream that sends none) is marked `SynthesizedID` and
  always replaced: `call_0` of one completion would otherwise name the same
  artifact as `call_0` of an earlier one that has left the replayed history.
- Anything that reads a clock or the database happens in an activity. The
  workflow holds only the turn and what its activities returned. That includes
  the tools the agent holds: `TurnState.Held` is taken when the turn opens, and
  the loop decides dispatch or refusal from it rather than from the catalog,
  which a later release may have changed.
- Every message is stamped when it was produced, through `TurnEffects.Now`:
  `workflow.Now` in workflow code, which reads back the same instant on replay
  and records no command, so stamping takes no version gate. `AppendTurn`
  keeps a message's own stamp; one without a stamp (a turn begun before
  stamping, a closing note) takes the stamp of the message after it, or the
  save's, and no stamp runs ahead of the message that follows it.

A failure retries only the call that failed, and a lost worker's turn resumes
from its last completed step.

### Model failures are retried the way the provider says

`modelcall` is the one mapping every model activity uses, the Temporal AI
cookbook's retry-from-HTTP-response recipe:

- A rejected request (4xx other than 408, 409 and 429), a refusal and a missing
  provider are **not retried**: they fail the same way however often they are
  sent.
- A rate limit waits out the provider's `Retry-After`, capped at a minute.
  Resting providers are asked again after their rest.
- Everything else is retried on the policy's own backoff.

The kind of failure travels in the error's details (`modelcall.Failure`), so
whoever reads it afterwards — the saved turn, a request waiting on a one-shot
call — still knows whether the provider refused, was unreachable, timed out, or
no provider was configured, and a refusal keeps its own words.

An activity with a deterministic fallback — document routing, inbound email
classification — leaves a transient failure to Temporal and falls back only on
its last attempt (`modelcall.Transient`, `modelcall.FinalAttempt`).

## What makes a retry safe

A tool call is an activity, and an activity can run more than once. The tools do
not dedupe: a policy's `Idempotent` flag is checked for presence and, bar the two
that forward it to an email provider, never looked up.

`agent_run_steps` is what makes it safe. Every operation is **claimed before it
runs** and settled after:

- The key is derived from what the model asked for — owner, tool name,
  arguments, and the ordinal the workflow assigned — not from the provider's
  call id, which changes on every attempt.
- A claim that finds the key already settled returns the recorded outcome
  instead of running again.
- A claim that finds it still `Started` means the previous attempt died between
  executing and recording. The model is told the operation began and its outcome
  is unknown, rather than being silently replayed.

That last case is the honest limit: resume is **at-most-once, not lossless**.

Outcomes over 64 KiB are dropped rather than truncated, because half a fenced
JSON document handed back to a model is worse than none. The import assistant
creates nothing itself: `create_location` is an approval-gated action like any
other, so it runs once, after a person approves it, under the ledger.

## Taint: runs that have read outside content

A run that was started by, or has read, content written outside the
organization is **tainted**. A tainted run never runs a write whose egress class
leaves the organization (`customer_visible`, `driver_visible`,
`external_recipient`, `money`) on its own: `agenttoolpolicy.Decide` lowers it to
`ActWithApproval`. Money is the class whose tier moves, because the other three
already stop at `ActWithApproval`, but every one of them names `tainted` in
`HeldBy` whenever the rule applies, whatever held the call first, so
`agent_proposals.held_by` records that taint held it. `HeldBy` keeps the order the
checks run in (tool tier, agent ceiling, tool max, class, condition, taint) and
names each reason once.

A write that stays inside the organization can declare one further taint hold
(`ToolPolicy.TaintHold`: a description and the calls it applies to), which the same
rule honours: a tainted or nil-taint call it applies to is held at
`ActWithApproval` and names `tainted`. `remember` is the one tool that declares
it. An `Instruction` or a `Correction` recorded after outside content waits for a
person; a `Fact` is recorded and stays tainted. The hold shows in the safety
table (for an agent that runs `remember` on its own, the "after outside text"
column reads "Depends on the call" and names taint among what holds it), in
`ai-tool-safety.md` and in the tool snapshot (`taintHold`). A tainted proposal
held this way is executed only on a person's decision, like one that leaves.

`agent.RunTaint` is the run's marks, one per place the content came from
(source, tool, call id, record), deduplicated and bounded to sixteen. A turn
opens tainted when:

- its subject is an inbound message, a document, an EDI inbound file or a bank
  receipt, which is every run woken by `inbound_message.classified`,
  `document.extracted`, `edi.file_quarantined` or `bank_receipt.exception`
  (`agent.SubjectType.TaintSource`);
- its subject is a record written outside the organization, which the subject
  describer says on `RuntimeSubject.OutsideAuthored`: a shipment entered by EDI
  (`EntryMethod` `EDI`) is the trading partner's text, so a run woken by
  `shipment.created` or `service_failure.detected` on one opens with an `edi`
  mark on the `shipment` (`agent.OutsideAuthoredTaint`), and the subject block
  of the prompt says who wrote it;
- the person attached a file to the question;
- a memory read into its prompt was written by a tainted run (a person's approval
  of the `remember` that wrote it changes how it is rendered, not whether it taints);
- its conversation is already tainted (`assistant_threads.taint`), which a
  decision follow-up inherits because it is a turn on the same conversation;
- the agent that handed it a task was (`DelegateCall.Taint`).

Master-data names and notes written inside the organization are not sources.
A shipment comment written outside it is (`record_note`, below).

Web content is held more strictly: once a turn has read the web through an extension, every
later write is capped at `Propose`, whatever its class
([agent-extensions.md](agent-extensions.md)). That rule rides its own flag
(`TurnState.ExternalContent`) beside the taint, and the web read also adds a `web` mark to the
taint, so both record it.

During the turn, a tool whose policy reads outside content always
(`ReadsExternal: always`) adds a mark when it succeeds; a `marked` tool
(`recall_memory`, `get_agent_run`, `list_agent_runs`, `list_watchtower_items`,
`get_shipment`) adds one only for a returned record that carries taint
(`agent.TaintCarrier`). A result whose rows come from different places names
each row's source itself (`agent.SourcedTaintCarrier`, preferred when both are
present), and its policy lists every further source it may name
(`ToolPolicy.Sources`, checked at boot and printed in the safety table). A row
that names a source the policy does not declare is still marked, under the
policy's own source, so a mark is never dropped and the table never lies.

`get_shipment` returns the shipment's twenty newest top-level comments (when the
caller may read shipment comments) and marks each one written outside the
organization as a `record_note` (`shipment_comment` record). Comment origin is
not a column of its own, so `ShipmentComment.WrittenOutside` reads it from what
is recorded:

| Comment | Outside? |
|---|---|
| `source` `Integration` | yes: another system wrote it |
| `source` `AI` | only when `metadata.tainted` is set: `add_shipment_comment` (`CarriesTaint`) stamps it on a note a tainted or nil-taint run wrote on its own; a note a person approved is not stamped |
| `type` `DriverUpdate` | yes: what Dash writes for a driver, including rows written before Dash stamped its origin, and a note an employee filed under that type |
| `metadata.source` `dash` or `edi` | yes: Dash now stamps `dash`; EDI 214 status and 210 invoice comments carry `edi` |
| anything else (`User`, `System` from inside Trenova) | no |

Each returned comment carries `writtenOutside`, and the result carries a note
that such a comment reports, never instructs. No other query tool returns
comments or notes an outsider wrote: service-failure notes, detention evidence
and notices, and payment memos are written by the system or by people inside the
organization; a bank memo is read through the bank receipt tools, which always
taint. The marks travel on the tool's outcome and
in the step ledger, so a replayed step taints the run as the original did. The
loop folds them into the turn and emits `run_tainted` once for each new mark,
which reaches the stream and the trajectory. A delegate's taint is folded back
into the turn that asked when it ends, because its reply enters that turn's
context.

### Oversight tools

Four read tools let an agent look over the organization's own agents, and each
is written so that reading about work never launders the outside text that
work read.

- `list_watchtower_items` reads the feed through `WatchtowerService.List`, so a
  reader sees only the kinds whose source resource they may read. It marks each
  row on its own source: an inbound message row as `inbound_message`, a
  quarantined EDI file as `edi`, a weather alert as `weather`, and an agent row
  (a failed run, a proposal, a plan, an exception) as `run_record` only when the
  run behind it is tainted, found by batched reads of the proposals, plans,
  exceptions and runs. A row whose run cannot be found is marked, not trusted.
  It never marks anything seen; `unseenOnly` follows a person's own cursor and
  is ignored for an agent principal, which has none.
- `get_daily_briefing` reads the computed page (`ReadsExternal: never`) and drops
  every section whose source resource the reader may not read, and every
  watchtower line of a kind they may not read. When anything is dropped, the
  headline goes too and the remaining sections read in their computed wording,
  because the model's prose may cite a figure from what was dropped.
- `list_agent_runs` filters on status, trigger, subject, agent and date; `mine`
  keeps to the calling agent and is on by default when an agent principal runs
  unattended. A run on an assistant conversation is listed only to the person
  who owns that conversation (`ThreadOwnerRepository`, one batched read), and
  never to an agent principal or an API key, whose query leaves them out.
  Each tainted row is a `run_record` mark.
- `get_agent_run` applies the same ownership rule, answering "not found" rather
  than "forbidden", and can add the run's proposals (at most 20, when the reader
  may read proposals) and its last 50 recorded events, each reduced to its kind,
  tool and a short text: never a tool's whole result or its arguments.

What is kept:

| Where | Columns |
|---|---|
| `agent_runs` | `tainted`, `taint`, `tainted_at` |
| `agent_proposals` | `tainted` (the run had read outside content when the write was decided), `taint`, `egress_class`, `held_by` |
| `assistant_threads` | `taint`, `tainted_at` |
| `agent_memories` | `tainted`, `taint_run_id`, for a memory `remember` wrote from a tainted or nil-taint run (`CarriesTaint`); `source_proposal_id` and `created_by_user_id` when a person approved the `remember` that wrote it |
| `shipment_comments` | `metadata.tainted`, for a note `add_shipment_comment` wrote on its own after outside content |

The proposal executor refuses a tainted proposal whose class leaves the
organization, or that names `tainted` in `held_by`, unless a person decides it
(`ErrTaintedNeedsPerson`), checking the class as the write would run and as it
was proposed, and records the class it ran with. It hands the tool the proposal
id (`ToolExecuteParams.ProposalID`); a `CarriesTaint` tool reads a nil taint
outside a proposal as unknown and marks what it writes with the run's record.

### Memory in the prompt

The prompt renders memories in two sections. What the organization recorded, and
any memory a person approved (`Memory.ApprovedByPerson`: written from a proposal
a person decided), goes under "What this organization has recorded for its
agents", grouped under a heading per thing it is about ("About Acme Foods
(customer)", "About tool assign_move", "For the whole organization"), where an
Instruction is followed as if the person who recorded it were asking now. A
tainted memory nobody approved (`Memory.DrawnFromOutside`) goes under "Recorded
by agents after reading outside content", fenced as
`<memory_from_outside_content>`, each line naming the kind it was recorded as,
and framed as information drawn from outside text that is never followed. It
still taints the turn that reads it. Trust promotion is unaffected: the ceiling is applied in `Decide` on
every call, whatever tier was earned.

**What a turn reads.** `ContextBuilder.Build` collects the records the turn is
about (`RuntimeContext.MemoryRecords`: its subject, the page's record, the records
the person mentioned, and for a delegate the records of the turn that handed it
the task) and `agentmemoryservice` resolves them to at most twelve memory
subjects (`SubjectResolver`):

| Record | Subjects |
|---|---|
| customer, location, worker, carrier | itself, as the record the turn is about |
| shipment | its customer and bill-to, its stops' locations in order, its assigned workers, its carriers |
| shipment move | its shipment, then as above |
| invoice, billing queue item | its customer |
| inbound message | its matched customer and carrier, and its matched shipment as above |
| document | the record it is attached to, as above |

The records a turn is about come first; the ones they name follow in the order
a person reads the record. The links are batched reads
(`agentmemorysubjectrepository.ListRecordLinks`, one query per kind per hop). Up
to 500 active, unexpired candidates are then read, subject rows first, and
ordered by `services.MemoryRanker` — recency and use count until retrieval can
rank by meaning.

**What the prompt carries.** `OpenTurn` fits the candidates to the agent's
`memory_token_budget` (6,000 tokens by default, 1,000–16,000; estimated with
`shared/llmtokens`) in this order, skipping whatever does not fit and filling
what is left:

1. memories about a record the turn is about;
2. memories about a record it names;
3. organization-wide Instructions;
4. memories about a tool loaded this turn (every held tool when the turn
   disclosed none);
5. everything else — Facts, and Corrections to tools not loaded — in the
   ranker's order.

A memory over 1,200 characters is shown cut short with its id, and the prompt
tells the model to read the rest with `recall_memory`. Only what fits is counted
as used (`AgentMemoryService.RecordUse`), and only what fits taints the turn. A
preview built outside a turn (`PreviewPrompt`) applies the same fit and counts
nothing.

**Determinism.** All of it runs in activities: `Build` in the prepare and open
activities, the fit in `OpenTurn`. The records a delegate inherits travel as
optional data (`RunRequest.Records`, `TurnPlan.Records`, `agentflow.RunContext.Records`)
from an activity result into the delegate's open activity. Workflow code only
copies them; a history recorded before they existed replays with none, and the
delegate reads the memories of no record. No `GetVersion` gate.

**Recall.** `recall_memory` searches a generated `search_vector` on
`agent_memories` (`'simple'` configuration: subject label A, content B, tool C;
GIN index). A query is `websearch_to_tsquery` or'd with every word as a prefix,
ranked by `ts_rank_cd`; when nothing matches every word, any word's prefix is
tried, so a question finds the memories that share most of its words. A query
with websearch operators (quotes, `or`, `-word`) is taken as written. It returns
10 rows by default and at most 50, reads one memory whole by `id`, and marks each
row a search found `match: "words"`.

**Limits.** A memory holds at most 4,000 characters
(`agent.MaxMemoryContentChars`). An organization should keep at most 5,000
active memories; `agentMemoryUsage` reports the count and AI Control's Memory
section warns from 4,000. Recording the same sentence again returns the existing
row only when it is unexpired, kept for the same readers (organization, or the
same agent) and, for a clean write, not tainted.

### Taint is data

No `GetVersion` gate. Taint enters workflow code only from activity results
(`OpenTurn`'s state, the delegate's opening, each tool's outcome) and leaves only
through activity inputs (`DispatchCall.Taint`, `DelegateCall.Taint`, the finish
activity's `RunResult`). Whether a write is proposed is decided in the dispatch
activity. Workflow code merges marks into `RunResult.Taint` and publishes
`run_tainted` to the Workflow Stream, which records no command; it never chooses
a command from taint. A mark's source comes from the tool's result inside the
dispatch activity (`callTaint`), whether the policy names it or the result names
it per row, so a tool that marks rows from several sources changes what the
activity returns and nothing about the commands workflow code issues. A recorded history from before this release carries no
taint, so its turn replays with a nil taint, adds no mark and emits nothing, and
issues the same commands in the same order.

A nil taint is a turn opened before the release. It counts as tainted for every
class that leaves the organization and for a declared taint hold, which is the
safe reading; money tools and `remember`'s Instruction and Correction notice. A delegate opened for such a turn inherits the nil.

## Chat turns

Every assistant question is answered by `AssistantTurnWorkflow`, ID
`assistant-turn:<turnID>`. The API records the turn, starts the workflow and
returns the turn to watch. `POST /threads/:id/messages/` starts the same
workflow and waits for its result; `POST /ask/` opens the hidden thread and
starts a turn on it.

1. **Prepare** reads the thread, history, files and mentions, checks budget and
   room, and runs the scope guard. A refusal the person can act on is
   non-retryable and its message reaches the reader as written.
2. **The loop** runs in workflow code, as above.
3. **Finish** saves the turn, its proposals and artifacts, closes the turn's
   record and writes the trajectory. It claims one ledger key per attempt, after
   checking no earlier attempt saved, so a retried save never appends twice.

A partial unique index on `assistant_turns` enforces one live turn per thread;
its `origin` and `input` say what the turn answers.

**Stopping** cancels the workflow. What had happened by then is saved on a
disconnected context and the execution is recorded as cancelled. The execution
id is derived from the turn (`AssistantTurn.ExecutionID`), so Relay and Stop
reach a turn whose start was never recorded. When no execution carries the turn
— it was never handed to a worker, or its execution ended without closing the
record — Stop closes the record as `Stopped`.

Signing out stops every reply the person still has in progress in that tenant
(`authservice.Logout` → `AssistantTurnStopper`), by the same path as Stop. A
turn does not record the session that asked for it, so signing out of one
browser stops the replies started from any other. Both callers cancel through
`AssistantTurnCanceller` (`assistantjobs.NewTurnCanceller`), the one place
Temporal's `NotFound` becomes `ErrNoTurnExecution`.

The record always closes. `MarkWorkflow` only touches a live record, so a turn
that finished before its start was recorded is not put back to Running; a save
retried after a lost attempt closes the record it did not re-save; and a save
that fails on every attempt is followed by `CloseTurnActivity`, which records
the turn as Failed.

### The turn stream

A run's events go to a **Workflow Stream** its own workflow hosts
(`agentflow.HostStream`, on `go.temporal.io/sdk/contrib/workflowstreams`). The
model activity publishes the reply as it streams, batched every 100 ms; the
workflow publishes every other event. The stream exists as soon as the workflow
does, so a reader can never attach ahead of it.

Every tool call a reader sees says what it does. `tool_started`, `tool_finished`
and the tool calls on `message` carry `effect` (`lookup`, `change`, `navigate`,
`discover`, `present`, `ask` or `delegate`), read from the tool's metadata
(`serviceports.EffectOf`, which reads `Effect` from the tool's `Policy()`;
`find_tools`, `ask_user`, `publish_artifact` and `delegate_task` declare theirs in
`agentruntime/runtimepolicies.go`). `tool_finished` also carries `summary`, a one-line label worked out
where the tool ran from what it returned, or from a write's name or title. The
thread's saved messages carry the same fields: `summary` is stored on the
result, and `effect` is read from the registry when the messages are served, so
a tool that no longer exists has none. The labels are data only; the loop
decides nothing from them.

Each event's offset is its SSE event id, returned as `Last-Event-ID` to resume.
The cursor is the last event the reader **applied**, not the last it received.

A stream is read by polling the workflow, so a closed workflow cannot be read.
When the relay forwards the last event it signals `stream-drained`; the workflow
waits for that, at most 15 s, before it closes. A reader who arrives after the
workflow closed gets an ending rebuilt from the record, which tells the client
to read the conversation.

Whether anybody drained the stream is how the turn knows somebody saw it end.
One that ends **Completed, Refused or Failed with nobody drained** runs
`NotifyUnseenTurnActivity` after the stream closes: a notification to the
person who asked (`kind: assistant_reply_ready`, with `threadId`, `turnId`,
`status` and a `link` to the conversation from the record-link registry), and a
quick question's hidden thread is kept first so the stale-Ask sweep cannot
delete what the notification leads to. It is keyed on the turn, so a retry
never notifies twice. A stopped turn, and one started by
`POST /threads/:id/messages/` (whose caller waits on the result), never notify.

A turn starting (once its workflow is recorded) and its record closing, on
every path, are announced as the `assistant_turns` realtime resource,
`started` or `finished`, carrying `turnId`, `threadId`, `userId`, `status` and
`origin` and nothing the conversation said: the channel is the tenant's, not
the person's. A tab reads the person's live turns from
`GET /assistant/turns/active/`, scoped to them.

Each publish is a signal in the turn's history and each read a poll update, so a
streamed reply adds a few hundred history events. That is why chat is one
workflow per turn rather than one per thread, and why a run nobody watches
publishes nothing.

### Handing a task to another agent

The agent a person is talking to may hand a task to another agent on its
allowlist with `delegate_task`. The other agent's turn runs inline in the same
`AssistantTurnWorkflow`, through the same `Drive` and effects, as its own agent
and as the same person; its events reach the same stream tagged with
`agentId` and `delegateCallId`, its steps are saved to the thread as
`Delegated` messages the model never reads again, and its writes are recorded as
its own. One level only. **Read [agent-delegation.md](agent-delegation.md)
before changing it.**

### Decision follow-ups

A decision on a proposal or plan a conversation raised is answered in that
conversation, whoever decided it and wherever. `agentdecisionservice` and
`agentplanservice` call `DecisionFollowUps` once the change has run or failed;
`assistantfollowupservice` opens a turn with origin `DecisionFollowUp`, as the
thread's owner. A conversation already producing a reply is not interrupted,
and the reply under way read the proposal before it was decided, so it cannot
report it. Instead the follow-up waits: when a turn closes its record,
`FinishTurnActivity` asks `DecisionFollowUpResumer` to start the follow-up for
the oldest decision in the last day that the thread carries no Decision note
for. That follow-up resumes the next one when it ends, so decisions made in a
burst (several cards approved in a row, or a batch from the decisions inbox)
are each reported, in order. A follow-up turned away before it was planned
saved no note and does not resume, or it would start itself again.

Every turn also reads what became of the conversation's proposals. A replayed
tool result that recorded a proposal is swapped for its current state; a
decided proposal whose call is not in the replay (a delegate's, or one older
than the history) is told beside the question instead (`outOfViewDecisions`),
so a delegate_task result saying a card is waiting is not the last word.

## Agent runs

`AgentRunWorkflow` runs one agent definition against its subject.

1. **Prepare** loads the definition, the organization's control and the
   subject, and marks the run diagnosing.
2. **Open** builds the turn: permissions, memory, what the ledger already knows.
   A definition in **shadow mode is opened in simulation**, so an automatic
   write is previewed rather than made — shadow mode exists to withhold it.
3. **The loop** runs in workflow code under the definition's run timeout.
4. **Finish** files the proposals, the summary and the trajectory. It runs
   however the loop ended, so a write made before a failure is on the record.
5. The run then waits until **every** proposal is decided, or the decision
   window closes and the rest expire. A decision signal says something was
   decided; the run recounts what is still pending after each one, which is
   right for a plan that decides several steps under one signal and for a
   proposal decided twice in a race.

An approved proposal is executed by the decision service, not by the run: that
path carries the modification checks, the trust ledger, memory and follow-ups,
the approval UI reads its result synchronously, and a chat proposal has no run
to execute it.

What a person approves is the preview they were shown: the decision service
previews the write as the decider sees it, refuses one whose record moved on
before anything is recorded, refuses a digest that no longer matches as a
conflict, and records the preview with the decision. A plan's later step on a
record an earlier step changed runs against the version that step left. See
[proposal-previews.md](proposal-previews.md).

### Starting runs

- **Events.** A run is keyed by its subject:
  `agent-run-<definition>-subject-<subject>`. The workflow starts before the run
  is recorded, with conflict policy FAIL and
  `WorkflowExecutionErrorWhenAlreadyStarted`, so a second event about a subject
  whose run is still open is refused by Temporal, where two requests reading a
  count of open runs could both have passed. The refusal is
  `ErrAgentRunAlreadyOpen`, which the publisher treats as a skip.
  `PublishAgentEvent` defers through `ports.AfterCommit`: an event raised
  inside `WithTx` is queued on the outermost transaction and published only
  once it commits, and dropped if it rolls back, so no run starts for a record
  that was never saved. Outside a transaction it publishes at once.
- **Schedules.** Every scheduled or continuous agent has its own Temporal
  Schedule, `agent-definition/<id>`: its cron in its own timezone or its
  interval, ending at its end date, paused while it is disabled, overlap
  skipped, catch-up window five minutes. Saving an agent syncs its schedule;
  `ReconcileDefinitionSchedulesWorkflow` runs when a worker starts and every
  quarter hour, backfilling schedules and removing orphans. A firing starts
  `AgentScheduledRunWorkflow`, which checks the agent may run now and starts
  the run for that slot; the slot keys the run, so it starts once however often
  the start is retried. The static schedule registry leaves these alone: they
  carry `managedBy=agent-definitions` in their memo.

### Evaluations

`AgentEvaluationWorkflow` replays a run's input against the agent as it is now,
with simulation forced on, through the same loop, on the heavy queue. It runs the
loop itself rather than as a child `AgentRunWorkflow`: a replay files nothing,
waits on no decision and must never touch the live run's record, which is
everything a run workflow exists to do.

## One-shot calls

`completionjobs` runs the model calls a person waits on outside a conversation.
The request starts a short workflow on the chat queue and waits for its result;
the workflow's budget is the request's own deadline less the margin the handler
needs to answer, so a call never outlives the request waiting on it. A person who
stops waiting cancels a call that was theirs alone.

- `StructuredCompletionWorkflow` asks one structured question (table compose),
  retried the `modelcall` way. Formula generate and explain no longer call it;
  they are turns of the formula assistant.
- `TestAIProviderWorkflow` probes once and never retries: the administrator is
  asking whether the connection works now.
- `WriteBriefingWorkflow` rewrites a day's page.

Two people testing the same provider, or rewriting the same page, share one
execution. The caller gets the call's own error back.

## Page assistants: import and formula

The shipment import assistant and the formula assistant are ordinary agents on
the shared runtime. Each is a **system agent** (`import_assistant`,
`formula_assistant`) that `AgentDefinitionService.EnsureSystem` creates the
first time anyone in the organization opens the page, from its template, with
memory on and access for everyone. Creation is an insert that does nothing on
conflict, so two people opening the page at once end up with one agent; a
name already taken moves to "(built-in)", then "(built-in 2)", and so on.
Neither agent is offered in chat pickers, as a delegate, or on schedules.

**One conversation per page.** A page thread has origin `Import` or `Formula`
and its subject: the document being imported, or the formula template being
written (`FormulaTemplate`). `POST /documents/:documentID/import-assistant/thread/`
and `POST /formula-templates/ai/thread/` open it, requiring document or formula
template read, `assistant:create`, and use of the agent (`MayUseAgent`). A
partial unique index keeps one live thread per person and subject; a template
not yet saved has a thread of its own, removed by the stale-thread sweep once it
has been idle as long as an Ask question. The Desk lists neither origin, the
live-turn list skips them, and a turn that ends unseen does not notify, because
the answer is on the page the person is looking at. A document thread opens
tainted by the document, so every write it proposes waits for a person.

**The draft rides on the page context.** Turns go through the ordinary
`/assistant/threads/:id/turns/` route with `pageContext.draft`, a bounded copy
of what the page holds (`pagedraft.Draft`): the fields read from the document
with their confidence and status, the four required records and the stops, or
the formula's expression and variables. The prompt renders it in a
`<page_draft>` fence as data. `prepareTurn` refuses a draft on any thread that
is not a live page thread of the matching surface, and re-checks that the
subject is still readable.

**Draft tools present; they do not save.** `accept_field`,
`accept_all_confident`, `set_field_value`, `set_required_field`,
`set_stop_location`, `set_stop_schedule` and `propose_formula` are query tools
with self scope and effect `present`. Each returns a `pagedraft.Edit`; the
turn saves it as a `draft_edit` artifact and the artifact event carries it, and
the page applies it once, the way a navigation artifact is followed. Nothing
reaches the database until the person creates the shipment or saves the
formula. `set_required_field` and `set_stop_location` read the record they name
under the person's own permission and hand the page its label.
`create_location` is an internal-egress action at `ActWithApproval`, requiring
location create. The formula tools price with the formula engine:
`describe_formula_schema` lists the variables, functions and rate tables,
`test_formula_expression` prices sample loads or a saved shipment the person
may read, and `propose_formula` hands the editor an expression with its
variables, an explanation and engine-priced scenarios. It requires formula
template create or update. The model never states an amount the engine did not
produce.

**In the web app.** Both pages mount `PageAssistant`
(`components/assistant/page-assistant.tsx`), which opens the page thread and
draws the ordinary `MessageThread` bound to the page: the draft is read at send
time and always sent, and each live `draft_edit` is applied through
`useApplyDraftEdits`, claimed by artifact id for the tab (`lib/claim-once.ts`)
so a rejoined or replayed turn never reapplies a change over what the person did
since. History never applies anything. The import page applies edits as clicks
(`import-draft.ts`); the formula studio shows a proposal card and inserts it
only when asked. Without `assistant:create` the panel says so without calling
the server; a refusal from `MayUseAgent` or a disabled agent is shown in the
server's words; an archived thread (the document was re-extracted) offers a
fresh one.

**What stayed behind.** Conversations that were still active when this shipped
were carried into page threads by `20261231006560_carry_import_conversations`:
one thread per person who spoke in each, their turns in order, legacy tool calls
replayed paired with their results under fresh ids, and the organization's
import assistant created when it had none. Finished conversations remain in
`shipment_import_chat_*`, read-only through `GET
/documents/:documentID/import-assistant/history/`, for one release; the import
panel shows a finished one, collapsed and read-only, above the new thread. After that
release, drop the tables and the history endpoint together:

```sql
DROP TABLE IF EXISTS "shipment_import_chat_turns";
DROP TABLE IF EXISTS "shipment_import_chat_conversations";
```

That statement is kept here rather than in `postgres/migrations`, because the
migrator embeds every `*.sql` file in that directory and would run it at once.
Ship it as `20261231006570_drop_shipment_import_chat` when the release has
gone out, with the SQLite mirror, and delete `shipmentimportchat`, its
repository and cache, and the history route in the same change.

**Rolling it out.** Roll the workers on `agent-chat-queue` before the API. A
new API sends turns with a draft and page tools an old worker does not know;
an old API may still start `ImportAssistantTurnWorkflow`, which the new workers
keep registered. A legacy turn that finishes after the carry-over has run lands
only in the legacy tables, readable through the history endpoint.

## Batch work

- **Insights and the daily briefing** fan out: a parent lists organizations,
  in pages carried across continue-as-new, and starts one child per organization,
  four at a time, keyed for fairness by organization. One tenant's failure is
  its child's — retried on its own and named in the result — and costs no other
  tenant its run. The sweep is anchored to one instant.
- **Document extraction** submits once and then waits on a durable timer,
  polling the provider until it answers or the longest wait passes, all in one
  execution's history.
- **Inbound email** carries each attachment through the document pipeline and
  polls its extraction on a durable timer. It polls rather than waiting on a
  signal because extraction can end, or never begin, in places that would never
  send one.

## What is written down

The runtime narrates its own work — every tool it reaches for, every refusal,
every give-up — and `agent_run_events` keeps that narration.

The table is **append-only**: its repository has no update path at all.

| | `agent_run_steps` | `agent_run_events` |
|---|---|---|
| Answers | has this already run? | what happened? |
| Rows | one per operation, mutated Started→Completed | one per occurrence, never revised |
| Ordering | none needed | `sequence`, per owner |

They join on `step_key` wherever a tool is involved. Ordering is by `sequence`,
not time, and the sequence resumes from what is stored, because a retried run
opens a fresh writer.

Two rules govern the writer: **recording must never be why a run fails**, and
**recording must not slow the run down**. A run's trajectory is written by the
activity that files it, last, so a retry of the filing does not write it twice.

Read a trajectory through the `agentRunEvents` GraphQL connection, gated on
reading an agent run.

An event's `occurred_at` is the instant the workflow emitted it
(`StreamItem.At`, stamped with `workflow.Now`), not when the filing activity
wrote the account, so a day-long run's events keep their real spacing.

Each claimed step also carries the trace and span of its `execute_tool` span, the
definition and version that made the call and, for a delegate, the call id; its
outcome carries a one-line `reason` and a `verdict` (`ran`, `proposed`,
`denied`, …). A proposal carries the same trace and span, its `step_key`, when an
automatic write ran and at what version it left the record, and who it ran as.
The whole table of link columns, and who writes each, is in
[ai-tracing.md](ai-tracing.md#link-columns).

## Measuring it

`turn_first_event_seconds` is separate from `turn_duration_seconds` on purpose:
duration says how long the answer took, first-event says how long a reader who
attached as the turn began stared at nothing.

`trajectory_events_total{result}` counts dropped events. They are invisible by
design — the run carries on — so this is the only place they show up.

`gen_ai.client.token.usage` and `gen_ai.client.operation.duration` are recorded
per provider attempt, with the GenAI semantic-convention buckets.

Every run, turn, delegate's task and evaluation is one trace, named by its id and
rooted in an `invoke_agent` span its finishing activity emits; model calls, each
provider attempt, tool calls and writes hang beneath it, and a later decision
links back to the call it answers. **Read [ai-tracing.md](ai-tracing.md) before
adding a span, a trace attribute or a link column.**

## Running it in production

- **Server.** Workflow Streams uses Updates and Signals, on by default from
  server 1.29. Fairness needs `matching.enableFairness=true`. The local dev
  server (`temporalio/temporal`) sets it.
- **Workers.** Run the chat queue on workers of its own if the queue split is to
  mean anything. A worker that polls no heavy queue leaves heavy tools waiting;
  after fifteen minutes the model is told the tool could not be run.
- **Schedules.** Agent schedules are created by saves and the reconcile, not by
  the static registry; a worker's start reconciles them once.
- **Workflow Streams is Public Preview** (`contrib/workflowstreams` v0.1.1). It
  is used only through `agentflow.HostStream`/`OpenStream` and the
  `turnstream` reader, so an upgrade is local to those.

### Waiting on in-flight executions

Workflow code changed shape behind `workflow.GetVersion`, so executions started
before a change finish on the code they started on. Each old branch, and what it
alone still needs, can be deleted once Temporal shows no open execution started
before the change:

| Change id | Old branch | Kept only for it |
|---|---|---|
| `agent-loop-in-workflow` | `runInOneActivity`, `replayInOneActivity` | `RunAgentActivity`, `ReplayRunActivity`, `awaitDecision` |
| `insight-refresh-per-organization` | `refreshInOneActivity` | `RefreshInsightsActivity` |
| `daily-briefing-per-organization` | `writeInOneActivity` | `WriteDueBriefingsActivity` |
| `agent-loop-final-answer` | a turn that spends its tool budget ends on the canned `exhaustedReply` without asking the model for an answer | nothing; the check itself is the only cost |
| `assistant-turn-close-unsaved` | a turn whose save fails on every attempt leaves its record Running, and the conversation refuses every later question | nothing; the check itself is the only cost |
| `assistant-turn-notify-unseen` | a turn that ends with nobody reading its stream ends without telling the person who asked | nothing; the check itself is the only cost, and it is asked only of a turn nobody drained |
| `agent-loop-fresh-synthesized-call-ids` | a call whose id the adapter synthesized keeps it unless the replayed conversation already holds it | nothing; the check itself is the only cost, and it is asked only of a completion that carries a synthesized id |
| `document-ai-extraction-timer-poll` | `extractWithTaskToken` | `SubmitAndAwaitDocumentAIExtractionActivity`, `PollPendingDocumentAIExtractionsWorkflow` and its schedule, task tokens on `document_ai_extractions` |
| none: the workflow is retired whole | `ImportAssistantTurnWorkflow` on `agent-chat-queue`, which no route starts any more | the workflow, its activities and registry in `importassistantjobs`, `workflow_test.go`, and the turn machinery in `shipmentimportassistantservice` (the tool loop, `toolGrants`, `persistConversationTurn`); delete them once no `ImportAssistantTurnWorkflow` execution is open |

Holding writes for a person after a turn reads outside content (`TurnState.ExternalContent`,
`DispatchCall.AfterExternalContent`) took no gate either: it is optional data on the turn and the
activity input, decided from the tool's name, and adds no command. See
[agent-extensions.md](agent-extensions.md).

Taint took no gate either: see [Taint is data](#taint-is-data). Marking an
EDI-entered shipment as outside-authored took none for the same reason: the
describer sets a field on the subject in the activity that prepares the run,
`contextTaint` adds the mark in the activity that opens the turn, and workflow
code only carries the two results between them.

Hybrid tool ranking took no gate. The turn's query vector rides on `ToolSetState.Query`, and
the tools a `find_tools` call found ride on `FindToolsResult.Found` into the saved message;
both are optional data, and a history without them replays by keyword. See "Ranking" in
[ai-retrieval.md](ai-retrieval.md).

Tracing and provenance took no gate. No span is started in workflow code; the
root is emitted by the activity that files the run or turn. What rides through
workflow code is data: `StreamItem.At` is read from `workflow.Now`, which records
no command; the owner, delegate call and definition version on
`AIUsageAttribution` come from the turn's own request; `ModelReply.Usage` is an
activity result that `Drive` sums into `RunResult.Usage` (and a delegate's into
`DelegatedRun.Usage`); the proposal id, trace and span, tier source, executed
version, step key and executed time on `PendingAction`, and the reason and verdict
on `RunStepOutcome`, are all set inside the tool activity; the caller's
`traceOrigin` is a payload field. A history recorded before them replays with
none of it and is written as before: the writer stamps the event time, usage is
counted from the answering call, the proposal insert mints its id, and the root
has no origin link. The histories under `agentjobs/testdata/replay-loop` and
`assistantjobs/testdata/replay` were recorded before the change and replay
against it. See [ai-tracing.md](ai-tracing.md#determinism).

Proposal previews took no gate. When a write is held for a person, the dispatch
activity pins its target and previews it in one read-only snapshot, and keeps the
preview in `agent_proposal_baselines` keyed by the proposal id it already mints; the
preview never rides `PendingAction`, which `DispatchCall.ProposedSoFar` carries into
every later tool activity, so nothing about the action, the activity results or the
commands changed. An evaluation keeps no baseline, a retried activity's orphan is purged
by the expiry sweep inside its activity, and a settled step replayed from the ledger
never previews again. See [proposal-previews.md](proposal-previews.md).

Agent delegation (`delegate_task`) took no gate: whether a turn holds the tool
is decided when it opens, in an activity, and kept in `TurnState.Held`, so an
execution opened before it never takes the new branch. Keeping the hand-off's
account structured on the saved result (`delegateReport`) and `record` on write
results added only optional data, no command. See
[agent-delegation.md](agent-delegation.md#versioning).

Runs parked in a day-long decision wait are the slowest to drain; the recorded
histories under `agentjobs/testdata/replay` replay against the old branch and must
keep passing until it goes. The `agent-queue` drain registry goes on the same
condition: no open execution on `agent-queue`.

`workflowstarter.Enabled()` is always true now that the client connects lazily;
the branches that test it are dead and can go with the next change that touches
each of them.

## Known limits

- **Resume is at-most-once.** A crash in the execute→settle window reports
  "began, outcome unknown" rather than replaying.
- **Scheduled runs authorize as `PrincipalTypeAgent`** against the static
  `permission.IsAgentAllowed` table; "not implicitly a system administrator" is
  not yet true for unattended runs.
- **Background runs discard their transcript** beyond the summary and the event
  log.
- **Permission denials are not distinct events.** A refusal arrives as a failed
  tool result and is recorded as one.
- **Search attributes are not set.** Organization, feature, thread and
  definition are carried in workflow ids, summaries and fairness keys; typed
  search attributes need registering on the server first.
- **The provider circuit breaker is per worker.** Temporal's retries cover what
  it compensated for; sharing it across workers is a separate change.
- **A run's trace root is emitted when the run is filed.** A run parked in a
  decision wait shows its root once it is filed, not when the last proposal is
  decided, and a workflow evicted from the cache may never export its
  `RunWorkflow` span. The limits of tracing are listed in
  [ai-tracing.md](ai-tracing.md#known-limits).

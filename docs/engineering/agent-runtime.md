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
| Table compose, agent drafting and instruction tightening | `StructuredCompletionWorkflow` | `agent-chat-queue` |
| Briefing regenerate | `WriteBriefingWorkflow` | `agent-chat-queue` |
| AI provider test | `TestAIProviderWorkflow` | `agent-chat-queue` |
| Provider editor: model list and unsaved test | `ListAIProviderModelsWorkflow`, `TestAIProviderDraftWorkflow` | `agent-chat-queue` |
| Agent builder dry run | `AgentDryRunWorkflow`, one per try | `agent-chat-queue` |
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
  `plan_dispatch` run on the heavy queue whichever queue called them. Being
  dynamic, its input is opaque to the tenant interceptor, so the activity binds
  the run's tenant for row-level security itself and refuses a call that names
  none (see [row-level-security.md](row-level-security.md)).
- **Every call is held to its tool's whole schema** in the dispatch activity,
  reads and writes alike, before the tool, a preview or a card sees it. A
  misnamed required parameter is still renamed (`aliasedArguments`) and the
  owner key of a self-scoped call is set aside and stamped from the actor;
  anything else that does not fit (an undeclared key at any depth, a value
  outside an enum, a wrong type, a missing nested value) is refused with one
  `path: message` line per problem and "Fix the call and send it again". The
  refusal is a failed call: it spends the tool budget and arms the repeat
  guard. The guard answers an identical failed call with the first failure
  instead of running it; once a turn has sent failed calls again unchanged
  twice (`maxRepeatedFailures`), it stops calling tools, answers any other
  call in that batch as not run, and asks for an answer without tools under
  `stuckNote`, which says to tell the person what was refused and what they
  can do instead (`agent-loop-stop-on-repeats`; `stuckReply` when that answer
  cannot be had). Schemas are compiled once per tool (`toolschema.Validator`); every
  object in a tool's schema is closed and every enum names its source
  (`agenttoolschema.Enum`, `x-enumOf`, stripped by `toolschema.ForModel`),
  which `agenttoolpolicy/schema_contract_test.go` holds. A tool that
  implements `ToolValidator` is then asked at every tier, automatic writes and
  simulation included, and again by the executor where an approved proposal
  runs; `TestEveryActionToolValidatesBeforeFiling` lists the few that cannot,
  with the reason. All of it happens inside the activity, so no workflow
  command changes and no version gate is needed.
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

### A rested provider is rested on every worker

The completion router (`completionrouter/health.go`) rests a provider after
three unavailability failures in a row (a 429, a 5xx, a timeout, an unreachable
host) for a one-minute cooldown, so a turn stops paying for attempts on a
provider that is down. The count of failures is per process; the rest is
shared. When a worker opens the breaker it also writes `ai:breaker:{providerID}`
to Redis with a TTL equal to the cooldown, through the
`repositories.ProviderBreakerRepository` port
(`redis/repositories/providerbreaker.go`, provided in `RedisRepositoriesModule`
and injected `optional:"true"`). The key is the provider row's id, so a
tenant-owned provider is its own breaker, and the value is a marker: nothing
about the provider or its credentials is stored.

Each call reads memory first, then asks the store once, in one pipelined
`PTTL` round trip, about the providers memory does not already hold resting,
and keeps what it finds in memory until that rest ends, so a provider rested
elsewhere is not read again on this worker. The remaining TTL, not a
timestamp, is what is read, so the workers' clocks need not agree. The store is
best-effort: a read or write is bounded to 250 ms, a write outlives a caller
that has gone, and a failure (other than the caller's own cancellation) is
logged once as a warning, after which the worker carries on from memory alone
and leaves the store for 30 seconds before asking it again; it logs at info
when the store answers again. A completion is never failed or held up by the
store. Without Redis (tests, a CLI) the router is built without the port and
the breaker is exactly the in-memory one.

The kind of failure travels in the error's details (`modelcall.Failure`), so
whoever reads it afterwards — the saved turn, a request waiting on a one-shot
call — still knows whether the provider refused, was unreachable, timed out, or
no provider was configured, and a refusal keeps its own words.

An activity with a deterministic fallback — document routing, inbound email
classification — leaves a transient failure to Temporal and falls back only on
its last attempt (`modelcall.Transient`, `modelcall.FinalAttempt`).

### A reply cut off mid call

A provider's output limit is the ceiling on each reply (`aiprovider.Provider.MaxTokens`,
"Max tokens" in AI Control). A reasoning model spends most of it thinking, so two things
keep a small ceiling from ending every turn partway through a tool call:

- **Room to answer.** The OpenAI Chat and Responses adapters send at least
  `reasoningAnswerFloor` (the thinking floor plus `thinkingAnswerRoom`, the room Anthropic's
  adapter keeps after its thinking budget on a model that still takes one) when the provider is set to reason, or when the
  conversation carries this protocol's reasoning, which is how a model that thinks without
  being asked shows itself. A call that does not reason keeps the configured ceiling. Every
  adapter reports the limit it sent as `ChatCompletionResult.OutputLimit`.
- **One more try with more room.** A completion that is `Truncated` and carries no whole
  tool call, when it was cut inside a call (`CutOffCall`, set by the adapter; or a native
  call whose arguments did not parse) or spent its whole budget before any visible answer,
  is asked for again once with twice the output limit, capped at the provider maximum and
  `cutOffRetryCeiling` (32768). The raised limit rides `ChatCompletionRequest.MaxTokens` for
  the rest of the turn. The cut attempt is not kept, and the reader gets a `retrying` event
  so it drops the text it watched stream. When the retry is cut too, or there is no more room
  to give, the reply keeps whatever it said before the call and ends by saying it ran out of
  room, that the named tool was not filed, the limits it hit, and to raise **Max tokens**
  on the provider. A plain long answer cut by the limit keeps `truncationNotice`.

The retry is one more model activity, so it is behind the `agent-loop-cut-off-call-retry`
gate, asked only of a completion that meets the condition.

### Thinking on Claude

Claude models differ in how they are asked to think, and the Anthropic adapter reads which
kind it is talking to from the configured model id (`anthropicTraits` in
`modeladapter/anthropicmodels.go`). The id may be the Claude API's plain one, Bedrock's
`anthropic.`-prefixed one (with an inference profile's region in front, and its `-v1:0`
version), Vertex's `@date` one, or a dated snapshot. An id it cannot read, such as a
gateway's alias, keeps the budget behaviour every model took before, and a version newer
than any it names takes the newest known constraints.

- **Effort or budget.** Opus and Sonnet 4.6 and later, and every Fable and Mythos, think by
  effort: `thinking: {type: "adaptive", display: "summarized"}` with `output_config.effort`
  low, medium or high (Minimal asks for low, the least they have). `budget_tokens` is a 400
  on most of them. Display is asked for because these models leave the readable summary out
  by default and the thinking panel would show nothing. Older models keep the token budget
  and its raised `max_tokens`; adaptive thinking raises `max_tokens` to
  `reasoningAnswerFloor`.
- **Thinking style.** An Anthropic provider's thinking style (`aiprovider.ThinkingStyle`,
  "Thinking style" on the provider form) overrides the id for a model the adapter cannot
  read. Auto, the default, reads the id. Effort asks by effort whatever the id reads, and
  Budget asks with a token budget. Behind an alias the adapter cannot tell which effort
  model it is, so under a declared Effort None asks for adaptive thinking at low effort, the
  least every such model accepts. Only an Anthropic provider may hold anything but Auto:
  validation refuses it, and so does `ck_ai_providers_thinking_style_kind`.
- **None and Off.** None is the least thinking the model allows. Opus 5.5, Fable and Mythos
  cannot stop thinking and Sonnet 5.5 refuses `disabled`, so they get adaptive at low
  effort; Opus 5 gets `disabled`; Opus and Sonnet 4.6 to 4.8 get nothing, which is no
  thinking there. Off sends nothing and the model's own default applies, so a model that
  always thinks still does.
- **Only this turn's thinking goes back.** A thinking block is bound to the conversation
  before it, and that changes every turn: the system prompt carries the turn's page,
  memories and date, and older tool results are shortened in replay. So
  `toAnthropicMessages` replays thinking only for the assistant messages after the last
  user message, the current tool loop, which a tool result needs. Earlier turns' thinking
  is dropped on every model; removing a leading run of blocks is an edit the API accepts,
  and most models ignore earlier turns' thinking anyway.
- **A changed block is dropped, not refused.** Fable 5.1, Opus 5.5 and Sonnet 5.5 check a
  replayed block's prefix (system, tools and earlier messages), and for accounts created on
  or after 2026-08-31 a mismatch is a 400. Within a turn a tool found mid-turn still
  changes the tool list, and a failover changes the model, so those requests send
  `thinking.block_binding.prefix_mismatch_behavior: "drop_block"` with the
  `anthropic-beta: thinking-binding-controls-2026-08-01` header. The header goes only to
  those models, since a gateway in front of an older one may refuse an unknown beta, and
  Off sends `{type: "adaptive"}` there so the binding has a thinking object to ride on.
- **Drops are counted.** The API lists each dropped block in `input_transformations`
  (`message_start` when streaming). The adapter counts the `thinking_dropped` entries into
  `Response.ThinkingDropped`, and the attempt span carries it as
  `trenova.ai.thinking_dropped`. A value that keeps appearing is the harness editing a
  conversation it should hold still.

OpenAI-compatible servers that do not parse a model's tool-call template return it as text.
The chat adapter lifts it (`modeladapter.liftInlineToolCalls`): a whole GLM call
(`<tool_call>name<arg_key>k</arg_key><arg_value>v</arg_value>…</tool_call>`, each value
read as the tool's schema types it) or Hermes/Qwen call (`<tool_call>{"name":…,
"arguments":…}</tool_call>`) becomes a tool call with a synthesized id, and goes through the
same argument contract as a native one; an unfinished call, or an opening tag cut partway,
is removed from the reply and named in `CutOffCall`. Markup inside a code fence or code span
is left alone, because a model quoting the format is not calling a tool.

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
- a memory read into its prompt was written by a tainted run and nobody has
  reviewed it since (`Memory.Taints`). A person's approval of the `remember` that
  wrote it changes how it is rendered, not whether it taints; a review does both.
  Approving a tainted suggestion (in AI Control or on the Desk) is its review, and
  AI Control lists every Active memory that still taints with a "Reviewed, keep it"
  action (`reviewAgentMemory`), because one such memory holds every money,
  customer-visible and outside-recipient write of every turn that reads it;
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

`get_shipment` returns a summary unless the call passes `detail: full`: the
ids a follow-up write takes (moves, stops, assignment, holds, comments), the
customer, the rating, the commodities and charges, and each stop's window and
actuals as a local date and time in the stop location's zone (the
organization's when the location has none, UTC when neither does), with the
zone named beside it. It names the people on a move and never how to reach
them. The full record is walked before it leaves the tool, as every
`newGetTool` result is (`fieldaccess.go`, `nestedRedactor`): every nested
worker record (an object whose `id` is a worker id) keeps only the fields the
`worker` resource's sensitivities let the caller see, and the fields withheld
are listed as `worker.<field>` in `withheldByAccess`. Confidential fields are
dropped without being named. A caller below Restricted on workers therefore
never reads a driver's email, phone or address, even in the full record.
Nested people are judged by their id prefix, each object by its own rule first:
a user (an owner, a canceller, a worker's manager) and an organization or
business unit are reduced to `{id, name}`, and a carrier assignment is gated by
`shipment_move`, so `externalDriverPhone` (Restricted) is withheld as
`shipmentMove.externalDriverPhone` below Restricted.

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
| `agent_memories` | `tainted`, `taint_run_id`, for a memory `remember` wrote from a tainted or nil-taint run (`CarriesTaint`); `source_proposal_id` and `created_by_user_id` when a person approved the `remember` that wrote it; `reviewed_by_user_id` and `reviewed_at` when a person reviewed it, after which it taints no turn while `tainted` still records where it came from |
| `agent_reflections` | `tainted`, for a look back over a window that read outside content (or whose taint is unknown); every lesson it keeps is a suggestion and carries the window's taint |
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
tainted memory nobody approved or reviewed (`Memory.DrawnFromOutside`) goes under "Recorded
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
ordered by `services.MemoryRanker`: `retrievalservice.MemoryRanker` fuses
recency and use count (`agentmemoryservice.RankByRecencyAndUse`) with the
memories nearest the turn's query vector, and falls back to recency and use
alone when the organization is not indexed or the search fails. The ranker
also returns the candidates at or above the similarity floor
(`RankedMemories.Similar`).

**What bears on the turn.** `agentmemoryservice.Relevance` marks a candidate as
bearing on the turn (`agent.MemoryRelevance`, carried on
`RuntimeContext.MemoryRelevance`) when it is at or above the similarity floor, or
when its content, subject label or tool name shares a meaningful word with the
turn's query text (`agentsearch.Terms` on the query, with the operator's words
mapped to the schema's, against `agentsearch.TokenSet` on the memory; words under
three letters are ignored). The two are unioned, so a memory found either way
bears. When there is neither a usable vector nor any query words, nothing is
judged and every candidate bears, which is how the fit worked before relevance
was read.

**What the prompt carries.** `OpenTurn` fits the candidates to the agent's
`memory_token_budget` (6,000 tokens by default, 1,000–16,000; estimated with
`shared/llmtokens`) in this order, skipping whatever does not fit and filling
what is left:

1. memories about a record the turn is about;
2. memories about a record it names;
3. organization-wide Instructions and Procedures (`MemoryKind.Followed`);
4. memories about a tool loaded this turn (every held tool when the turn
   disclosed none);
5. everything else — Facts, and Corrections to tools not loaded — in the
   ranker's order, **only when it bears on the turn**.

Within a tier a memory that bears comes first, so a short budget keeps the
standing rules that apply to this message; then an Instruction comes before a
Correction, a Correction before a Procedure and a Procedure before a Fact
(`MemoryKind.Rank`). The tier-5 memories that do not bear, and anything that did
not fit, are counted (`MemoryFit.HeldBack`). When none are carried and some were
held back, the prompt still says memories exist and to call `recall_memory` when
the task may depend on one.

**What counts as used.** Carrying a memory and using it are different things.
`Definition.PlanMemories` returns the carried memories and, separately, those
that were used (`MemoryFit.Used`): memories about a record the turn is about or
names, memories about a tool picked for the turn (a disclosed turn's loaded
tools), and any carried memory that bears on the turn. A standing Instruction or
Procedure that does not bear is still carried and followed, but is not shown or
counted. A memory about a tool that was only on hand (every held tool, when the
turn disclosed none) waits in `MemoryFit.ByTool`, kept on `TurnState.ToolMemories`,
and is added to `UsedMemoryIDs` and announced when the turn calls that tool
(`Turn.noteToolMemories`). A state from before `ToolMemories` existed promotes
none, so no `GetVersion` gate is needed.

A memory over 1,200 characters is shown cut short with its id, and the prompt
tells the model to read the rest with `recall_memory`. Only what was used is counted
(`AgentMemoryService.RecordUse`), so `use_count` measures use rather than how
often a memory sat in a prompt; only what fits taints the turn. A
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

**Forgetting.** `forget_memory` retires a memory by id; it stays readable in AI
Control and can be restored. It is on no starter template: what a memory says
is what every later turn is told, an Instruction a person recorded among it, so
a person retires one in AI Control rather than an agent dropping it on its own
judgement. An organization adds the tool to an agent it builds.

**Replacing.** `remember` takes `replacesMemoryId` (an id from `recall_memory`),
and so does a lesson a look back keeps. The new memory inherits the old one's
readers, subject and tool, and records `supersedes_id`. When it is saved active,
the repository retires the old memory in the same transaction (only an Active or
Paused one) and writes an `audit_entries` row naming its replacement. An agent
replaces freely only the person's own memory or its own Agent memory
(`agentmemoryservice.ReplacedFreely`); any other replacement is held as a
suggestion, and approving it retires the old memory then. A retried write that
finds its replacement already kept returns it. `AgentMemory.supersedes` and
`AgentMemory.replacedBy` read both ends of a replacement through per-request
loaders (`AgentMemoryByID`, `AgentMemoryReplacement`, backed by `ListByIDs` and
`ListReplacements`); `replacedBy` is the newest replacement that took effect,
never a suggestion still waiting. The Desk carries the same links, with the
reason and quotes, on `DeskMemory` and on each reply's memory notes, naming the
other memory only when the person may see it.

### Learning from the work

When a conversation goes quiet or a background run settles, its agent looks back
over what it just did and keeps what the work taught: how to do a task here (a
Procedure), a Fact it had to work out, or what a person asked for from now on
(an Instruction, never from a run with nobody in it). Corrections stay with
`remember` and `RecordCorrection`. `remember` stays the tool
for what a person says; the prompt tells the agent to save only that, because
what it worked out for itself is kept for it by the look back. Both write
through `AgentMemoryService.Remember`, so dedupe, replacement, taint and the
saving mode apply to both alike. Nothing here changes `RecordCorrection` or the
nightly feedback job.

**When.** `FinishTurnActivity` cues the conversation's look back
(`AgentReflectionScheduler.AfterTurn`) once a turn is saved, and
`CompleteRunActivity` cues a run's (`AfterRun`). Both cues are activity code, so
no `GetVersion` gate. A conversation has one `AgentThreadReflectionWorkflow`
(id `agent-reflection:thread:<threadId>`), started or signalled with
`turn-finished` by signal-with-start. It waits until no turn has finished for
`QuietPeriod` (10 minutes), but never longer than `LongestWait` (an hour) from
the first turn it heard, then looks back once over the messages after the last
look back's `through_sequence`. A turn that finished while it looked starts
another round; after `RoundsPerRun` (20) rounds it continues as new. A run has
one `AgentRunReflectionWorkflow` (id `agent-reflection:run:<runId>`), started
with `ALLOW_DUPLICATE_FAILED_ONLY`. Both run on the agent background queue at
`PriorityBackground`, fair by organization, registered in the `agentjobs`
background registry.

**Whether.** `PrepareThread`/`PrepareRun` claim an `agent_reflections` row
(unique per conversation window and per run, so a retried start reads the same
row; a Failed row is claimed again) and skip without calling a model when:

| Skip reason | When |
|---|---|
| `LearningOff` | the organization's agent control or the agent's definition has `learning_off`, or the control is in shadow mode |
| `AgentUnavailable` | the agent's definition is missing or disabled |
| `NothingToRead` | the window holds no new messages (a run that has not settled is not claimed at all) |
| `NoSignal` | `ReadSignals` finds nothing worth a look |
| `OverBudget` | the organization's AI budget refuses the call |

`ReadSignals` is cheap and reads only the window: a tool that failed, or failed
then worked (refusals — `denied`, `duplicate`, `over_budget` — are not lessons);
a person correcting the agent after it answered; a person saying how they want
it done from now on; a proposal modified or rejected; a reply rated unhelpful;
and a task of six or more successful tool calls. A run with nobody in it reads
no person signals.

**What it asks.** A ready plan carries one structured completion
(`FeatureAgentReflection`, at most 2,500 output tokens, strict JSON schema of at
most five lessons of at most 1,200 characters each): the transcript (cut to the
last 60,000 characters), the decisions and feedback, the signals, and the
memories the agent already holds for those records, so it refreshes or replaces
rather than repeats. The model call is its own activity
(`ReflectionModelActivity`, `modelcall.RetryPolicy`); a failure marks the row
Failed with its message.

**What it keeps.** `Finish` reads the reply and saves each lesson through
`Remember` with source `Reflection`, `reflection_id`, the signals as evidence,
and the agent as its actor:

- a lesson is refused, with the reason on the row, when it names a record or
  a tool the work did not touch, replaces a memory the look back was not shown,
  or is an Instruction from a run with nobody in it;
- a conversation's lesson is kept for the person (`User`), the agent
  (`Agent`), or offered to the team or organization; a person without
  agent-memory create permission has shared lessons kept for themselves alone;
- team and organization lessons are always suggestions, and a run with nobody
  in it keeps its lessons at `Agent` scope and suggests organization ones;
- a window that read outside content (or whose taint is unknown) only ever
  suggests;
- otherwise the person's saving mode decides: Automatic keeps, Ask first
  suggests.

The row records each change (Saved, Suggested, Refreshed, Refused), the
model's notes and token use. A conversation's kept lessons are attached to the
reply they followed (`ConversationRepository.AddSavedMemories`) and announced on
the realtime resource `agent_memory`, so the Desk shows "learned" notes the
person can keep or undo. AI Control's Memory tab lists the look backs
(`agentReflections`) and queues the Agent and Organization suggestions for an
administrator.

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

1. **Prepare** runs as a local activity on the worker running the workflow
   (`assistant-turn-prepare-local`), so the turn's first workflow task prepares
   it and schedules the model call without a task-queue dispatch or a second
   workflow task in between. It reads the thread, history, files and mentions,
   checks budget and room, and runs the scope guard. After the thread, every check and read runs
   side by side (`checkTurn`); none writes, and a question failing several is
   told about the first in the old order (files, agent, page, budget, room,
   history), never about a cancellation. The guard's classifier runs beside the
   context build (memories, retrieval embedding), which writes nothing, so a
   refusal discards it; the classifier fails open after 3s. A refusal the
   person can act on is non-retryable and its message reaches the reader as
   written.
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
workflow publishes every other event. The first piece of each model call's
reply (text or thinking) is flushed at once rather than waiting for the batch
ticker, which starts with it (`agentflow.firstWords`). The reader rests 5 ms,
not the library's 100 ms, after each delivered batch
(`turnstream.pollCooldown`); its poll waits on the workflow until there is
something to return, so that costs no extra polls. The stream exists as soon as
the workflow does, so a reader can never attach ahead of it.

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
decides nothing from them. A `find_tools` answer's `tool_finished` also carries
`handOffAgents`, the person's other agents it named as holding what this agent
could not load, the same ids its saved tool message keeps (`hand_off_agents`),
so the Desk's hand-off menu leads with them while the turn is still running
rather than only once it is saved.

A reply that starts over and a reply that is corrected are told apart.
`retrying` (kind `restart`) means a model is being asked again: a provider died
partway, a cut-off call is asked again with more room, or a guard re-asks for a
looping or ungrounded reply. The reader withdraws the attempt's streamed text
and thinking, shows that it is retrying, and what follows is the whole reply.
`reply_replaced` (`AssistantReplyReplacedEvent`: `text`, `reason`) means the
runtime corrected the final reply after it streamed, without asking the model
again: the reply passes and the output guard below. It carries the whole reply
as recorded; the reader swaps its streamed reply text for it and shows no
retry, and the thinking before it stays. Another agent's are
`delegate_retrying` and `delegate_reply_replaced`, applied under the call that
handed it the task. Both are recorded in the run's trajectory, `reply_replaced`
after the `reply_regrounded` events saying what each pass changed.

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
allowlist with `delegate_task`, naming the records the task is about and sharing
results of its own calls as data. The other agent's turn runs inline in the same
`AssistantTurnWorkflow`, through the same `Drive` and effects, as its own agent
and as the same person; its events reach the same stream tagged with
`agentId` and `delegateCallId`, its steps are saved to the thread as
`Delegated` messages the model never reads again, and its writes are recorded as
its own, in one plan with the turn's when several wait on the person. One level only.
**Read [agent-delegation.md](agent-delegation.md)
before changing it.**

### Steering and the queue

A person can talk to a conversation while its agent is still working. What they
type goes to `assistant_queued_messages` (`conversation.QueuedMessage`), one row
per message in send order, through `assistantqueueservice` and the
`/assistant/threads/:threadID/queue/` routes. The queue lives in the database so
it survives a reload, another tab, and a restart of anything working on the
reply.

- **Steer** (`steer: true`, the composer's send gesture while a reply runs) also
  signals the turn's workflow on `agentflow.SteerSignal` with the row's id and
  words. The loop reads the signal between steps, before its next model call
  (`TurnEffects.Interject`); reading a signal records no command, so the change
  is asked about (`agent-loop-interjections`) only when one is waiting. The words
  reach the model framed by `agentruntime.SteerPrompt` and are saved as a
  `Steer` message, which history replay frames the same way and which does not
  count as a turn of its own. The reader gets a `steered` event. A message with
  files never steers: a reply under way reads words, not documents.
- **Queue** (Option+Enter, or anything with files) waits for the reply to end.
- **When a turn's record closes**, `FinishTurnActivity` (and a compaction's
  close) asks `AssistantQueueSettler.SettleQueue` to delete the messages the turn
  read (`FinishTurnInput.Steered`) and, after a turn that ended Completed or
  Refused, to start the next waiting message as a turn of its own (origin
  `Person`, as its owner, with the page, records, files and model it was queued
  with). A queued message goes ahead of a decision follow-up; with nothing
  queued, follow-ups resume as before. A stopped or failed turn holds the queue
  until the person sends it on. The started turn is announced as `next_turn`
  before the ending event, so the reader follows it without a gap.
- **A steer the turn never read** (it arrived after the last step, or no
  execution carried the turn) stays in the queue and is sent as the next message.
  The row is the source of truth; the signal is only the fast path.
- Claiming the head (`DELETE … FOR UPDATE SKIP LOCKED … RETURNING`) and the
  partial unique index on live turns keep two closers from sending one message
  twice; a claim whose turn cannot start is put back at its position.
- `assistant_queue` realtime events, addressed to the owner, move every tab's
  queue.

### The world changing under a turn

A turn watches the records it is about (`RunRequest.Records`: subject, page and
mentions), the records its `get_*` calls read and the records its executed
writes changed (`agentruntime.WorldState`, at most 64). Between steps, at most
every three seconds by `workflow.Now`, it reads what changed since it last
looked as a local activity (`agentflow.CheckWorldActivity`). The read is the
realtime bus, not the database: `realtimebroker.ChangeFeed` scans the tenant's
shard of the Redis stream from the cursor the turn opened with, at most 2,048
entries per check, keeps tenant-wide `resource.invalidation` entries whose
record is watched (a byte search before any decode), and merges several events
for one record. Changes to a record the turn itself wrote, within fifteen seconds
of the write, are its own and dropped. What is left reaches the model as a
`WorldChange` notice (system words, not the person's), clears the repeat
guard's memory of reads, is saved with the structured changes
(`assistant_messages.world_changes`) and reaches the reader as `world_changed`.
Delegated turns neither watch nor steer.

### Records in play

A conversation keeps the records it is about (`assistant_threads.working_set`,
`conversation.WorkingRecord`, newest first, at most `MaxWorkingSet` = 12): its
subject, what the person mentioned or had open, what the agent read with a
`get_*` call (`agentruntime.RecordsRead`, the same rule the world watch uses; a
failed read and another agent's delegated steps add nothing) and what its
executed writes changed. `FinishTurn` adds them through `keepWorkingSet`; the
subject always sorts first, and the kind is read off the id's prefix
(`permission.RecordKindOfIDPrefix`), so an id of no known kind is never kept.

Every turn reads them again while it is prepared, beside the subject
(`checkTurn` → `anchorRecords` → `services.RecordAnchorReader`,
`recordanchorservice`): this turn's mentions and page record first, then the
working set, leaving out the subject, which `<subject_context>` already reads
fresh. Access is checked per kind on every turn (`subjectaccess.MayReadResource`);
a record the person can no longer read, one that is gone and one that could not
be read are each named with a note and no facts. Each kind that moves has a
batch reader:

- **Shipment**: the tracking snapshot (`ShipmentTrackingReader.TrackingSnapshots`,
  the same build `EtasByShipmentIDs` uses), rendered by
  `shipmenttracking.Snapshot.Facts`: status and customer, what needs attention,
  the next stop (arrived and not departed, or overdue and by how long), who covers
  it, the last position, the driver's hours and the estimated arrival.
- **Invoice**: status, settlement and dispute, bill-to, shipment and due date.
  Amounts are left to `get_invoice`, which gates them by the reader's data access.
- **Detention occurrence**: status and notice, the clock, and the charge priced as
  of now for a running clock (`detentionservice.PriceOpenClock`).
- **Tractor**: status and last position. **Trailer**: status. **Worker**: hours of
  service, or that no ELD has reported any.

Other kinds (a customer, a location) are named but not read: nothing on them
changes under a conversation. What was read reaches the model as a
`<records_in_play>` block ahead of the question, after the `Now:` line
(`agentdefinition.DescribeAnchors`). It says the block is current and wins over
anything said earlier, and every fact taken from a report rather than the record
itself carries its age, "reported 40m ago", with "too old to plan on" past the
tracking snapshot's own staleness (`shipmenttracking.PositionStale`, `HOSStale`).
An ETA is as old as the position it was worked from.

The block is never saved: the question is kept as the person typed it, so a later
turn never replays an old snapshot as if it were current, compaction can drop the
transcript without leaving a stale status or ETA behind, and the system prompt and
the replayed history stay the same bytes from turn to turn for the provider's cache.
The anchored records are also watched for changes during the turn
(`watchedRecords`), so one that changes mid-turn reaches the model as a
`WorldChange` notice. Delegated turns get no block.

### Parking work on a wait

An agent can park its work until something happens with `wait_until`; the turn
ends, a workflow per wait holds the timer or listens for the event, and the work
comes back as a turn with origin `WaitResolved` (or a run with trigger `Wait`).
**Read [agent-waits.md](agent-waits.md) before changing it.**

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

The follow-up asks the agent to report only what the note and the proposal's
card hold. A decision that carries the person's note ("Tell the agent instead"
in the approval box) adds it, fenced as untrusted data and capped, as "The
person declined:"; the agent answers it and may propose a different change.
That turn is the rejection's own follow-up, so the note is never also sent as
a message and the agent answers once.

### A typed approval opens the approval box

Typing "approved" decides nothing: a proposal is decided in the approval box
that stands in the composer's place while it waits. The
system prompt's "Proposals awaiting a decision" section lists each waiting
proposal with its id, and a turn a person is reading, with somewhere to show
things and a proposal still waiting, holds `request_decision {proposalId}`.
The runtime answers it itself, like `ask_user` and `publish_artifact`: the loop
checks the id against the conversation's proposals as the turn opened (one of
them, still pending; one filed this turn already has its card on screen; one
card a turn), and the observer, where `publish_artifact` is kept
(`PublishArtifactActivity` in workflow code), reads the thread's proposals
again for this tenant and keeps a `decision_request` artifact over the one
still pending (and its plan, for a step of one). The client opens the
approval box on that decision (first, even one the person put off), and the
Desk's pane keeps the proposal's record with a way back to the box; the
artifact's status follows the proposal. A delegate never holds it. The tool
is held only by turns opened after it existed, so it took no gate.

Several waiting proposals of one tool share one card: `request_decision
{proposalIds}` (2 to 50, standalone, all still pending, one tool) keeps one
`decision_request` artifact anchored on the first with every id in its payload,
and the approval box decides them as one entry through `decideMyProposals`,
each with the digest of the preview it showed. A plan's steps are asked for by `{planId}`: the card is
the plan's, anchored on its first waiting step. A step of a plan among
`proposalIds` is refused with the plan's id, and a mix of tools is refused.
Exactly one of the three parameters is given.

### A proposal the agent replaced is withdrawn

A turn's proposals are filed when it ends, and two or more become one plan the person
approves whole. An agent that proposes a change, notices a mistake and proposes it again
corrected would otherwise leave both in that plan: one approval ran both, and a load was
entered twice. Once a turn has filed a proposal a person will decide, the loop offers
`withdraw_proposal {proposalId, reason?}` (`agentruntime/withdrawproposal.go`). Every
filing result names its proposal's id ("Recorded a proposal to run "create_shipment"
(proposal ap_…)"), and the runtime answers the call itself, like `request_decision`: it
marks that action `Withdrawn` on `PendingAction`, and the recorder records it
`Superseded` (never pending, never a plan step, never on the watchtower, no draft
artifact), so the approval box offers only what is still filed. Only a proposal of this
turn can be withdrawn; one from an earlier turn is refused with the instruction to have the
person reject it, since it is already in front of them, and one that already ran is refused
too. A withdrawn action is left out of `ProposedSoFar` (so the same change filed again is a
new proposal, not a duplicate), of the grounding guard's filed writes and of a delegate's
report; a delegated turn is never offered the tool. The AI audit trail records the
superseded proposal's `filed` event and the `withdraw_proposal` call like any runtime step,
with no new event kind or canonical field. It took no gate: the tool is added to the turn's
in-workflow tool set as data when a filing comes back, no command is added, and a history
recorded before it never calls it.

### What a tool receives

Every call's arguments go through `contractCall` (`agentruntime/arguments.go`,
`coerce.go`) before a tool, a preview or a card sees them. A misnamed
parameter is renamed to the one it plainly means, the runtime's own owner key
is set aside, and each value is read as the schema declares it wherever the
reading is certain: a numeral sent as text becomes the number, a number sent
for a text amount becomes its text, a lone value for a list becomes a list of
one, comma-separated text becomes its parts, `{"item": [...]}` becomes the
list, `"true"` becomes true, an enum value in another case becomes the
declared spelling, and a null for an optional parameter is dropped. An
optional parameter sent as `""` or as a list of nothing is read as not sent,
and blank entries in a list of text are set aside: several models fill every
parameter a tool declares, and a transfer that named one shipment also sent an
empty customer and search beside it, which the tool read as filters given. A
parameter whose empty value means something other than leaving it out is
marked `toolschema.KeepEmpty` and keeps it: an update's field that an empty
value clears (the master data kit's `clearable`, order amounts,
`update_invoice_draft`, `update_service_failure`, a recurring series'
blackout days), and a write's list whose absence means every record, where
an empty list read as not sent would widen the call (the pay events
`pay_worker_now` pays, the customers `assess_late_charges` charges, the sync
records `retry_accounting_sync` retries, the suggestions
`apply_carrier_intel_suggestions` applies).
`TestEveryWriteWhoseOmissionWidensKeepsAnEmptyList` in the catalog contract
holds every write whose description says leaving a parameter out means every
record to marking it. What was left out is told back on one
line ("Left out because they were empty: …"). The
readings live in `toolschema.Coerce`, and the proposal executor makes the same
ones (`proposalexecutor.CoerceParams`) on a proposal's parameters with the
approver's changes laid over them, before they are checked and run: the
approval editor sends what a person typed (`30`, `2026-10-01 08:00`), and a
proposal filed before a parameter's shape was asserted carries what the model
sent. The preview a person approves from reads them the same way
(`proposalpreviewservice.settleParams`), so what is shown is what runs.
Nothing is told back there; what runs is what was read. The id checks
below stay in the runtime.

Dates and times come in three shapes, each from one helper in
`agenttoolschema` that sets the `format`, an example and the same sentence on
every tool: `Date` (`date`, YYYY-MM-DD), `DateTime` (`date-time`, RFC 3339 with
its offset) and `LocalDateTime` (`local-date-time`, YYYY-MM-DDTHH:MM with no
zone, read in the timezone of where it happens). `pkg/toolschema` asserts all
three (an empty string passes for a parameter marked `KeepEmpty`; elsewhere
it never reaches the format, being read as not sent) and
refuses a value of another shape in one sentence naming the shape and an
example. "today", "yesterday" and "tomorrow" sent for a day become the day they
name in the organization's timezone (`toolschema.WithToday`, which dispatch
passes; the query tools' own "or today" parser and the master data kit's day
field are retired onto `agenttoolschema.Date`). Midnight UTC sent for a day
becomes the day, and a local time written
with a space or with seconds becomes its minute; a number is never read as a
date, and is refused as the Unix time it is. The shape is also said in the
description. Anthropic and OpenAI's own endpoints are sent the full schema
(`toolschema.ForModel` strips only the `x-` keywords); the OpenAI-compatible
adapter, which reaches Gemini, vLLM and the rest, and Ollama are sent
`ForPortableModel`, which also leaves out `examples` (Gemini's schema has only
a root `example`) and the `local-date-time` format, which is Trenova's own. The call
then has to fit the schema, and its record ids have to be ids of the right
kind. An id parameter built with `agenttoolschema.RecordID` (or `RecordIDs` for
a list) carries the resource it takes as `x-recordOf`, at any depth, and its
value must carry that resource's prefix from the one table in the domain
(`permission.Resource.IDPrefix`, held by a test to the prefix each entity's
insert hook mints): a carrier's id sent for a customer is refused as "… is a
carrier id; this parameter takes a customer id, from list_customers" rather
than reaching the service and coming back "not found". An id whose resource
covers several kinds of record (a qualification is an employment verification
or a clearinghouse query), a record kept inside another's (an invoice line, a
leave day), or a parameter that takes any of several kinds (a case's
`subjectId`) is built with `agenttoolschema.KindID`/`KindIDs`, or marked in
place with `OfKinds` (`OfResource` for a resource), and carries the kinds as
`x-recordKinds`; each kind's prefix is declared in `permission.RecordKind`,
held to the domain's mint by the same test, and a value of none of them is
refused naming every kind it may be. A parameter with neither mark still has
to hold something shaped like a record id when its description names where
the id comes from (`from list_customers`); the catalog contract test
(`TestEveryRecordIDParameterNamesItsKind`) fails on any id parameter left
unmarked that is not listed, with its reason, among the few that hold no
PULID. The marks are stripped from what a model is shown
(`toolschema.ForModel`). What was renamed or re-read is told back with the result
("Arguments were read as the tool declares them: … Send them that way from
now on"), on a query result, a proposal and a write that ran. A refusal names
each problem at its path with what the schema declares there (the type, the
allowed values, the parameter's first sentence), and a key the tool does not
take is answered with the parameters it does. The model's own call is never
changed; the thread records what it sent.

A proposal's result says why it waits (`heldReason`: the tool's tier, the
agent's ceiling, a change that is always a person's, what it would reach, a
condition, outside content, business hours), a permission denial says what to
tell the person and not to try another way (`deniedAdvice`), and `remember`
says whether the memory is kept or only offered on a card
(`savedMemoryContent`). A tool done as a sequence declares it
(`serviceports.RecipeTool`), and the order is shown in the prompt's tool
section and under the tool in a `find_tools` answer, with the prerequisite
reads that were loaded alongside it. A `find_tools` need that names a tool the
agent holds outright (`search_worker`) loads that tool first, ahead of the ranked
matches, since the prompt lists what can be loaded by name. That is the only place the order of a
piece of work is written: a template's starter instructions
(`agentdefinition.Template.StarterInstructions`) hold its persona, what it puts
first, its rules and what is always a person's decision, and name none of its
tools (`TestTemplates_InstructionsLeaveTheOrderOfWorkToTheTools`). Walking
through tools in prose told every turn about tools it had not loaded, went
stale when a tool was renamed, and told an agent without a tool to call it.
Every recipe and prerequisite names a registered tool, and a recipe names its
own tool (`TestEveryRecipeNamesRegisteredTools`).

The prompt's tool section groups changes only by what is certain before a
call: a change that always records a proposal (`ToolSummary.AlwaysProposes`,
from `agenttoolpolicy.AlwaysProposes`: the tool's promotable tier, its egress
or the agent's ceiling keeps it below AutoExecute), one that runs at once for
the person's own records (`PersonalRunsUnasked`), and the rest, which run at
once or are recorded as a proposal depending on what the call reaches, with
the result saying which and why. It used to call a tool whose static tier was
AutoExecute a change that "runs as soon as you call it", and the turn's taint,
a condition or the call's reach held it anyway.

### An empty list says what would have matched

A list or search result names what it applied (`searchedFor`). When the first page of a
`newListTool` list comes back empty with a text or a filter applied, the tool probes before it
answers (`listnearmiss.go`), each probe one row at most: is there anything at all, and does
dropping the text, or any one of the first four filters, find something. The note then says
which: "there are no hold reasons at all, so no other wording will find one", "dropping just
one of these finds some: status equals Approved", or "together they rule everything out". A
bare "nothing matched" sent models round a loop of near-identical calls, a status at a time,
until the tool budget ran out. `search_shipments` does the same for its status: a search that
finds nothing in the status asked for returns the rows its words match in any status, with a
note saying none was in that status.

Shipment rows from `list_shipments` and `search_shipments` carry the first pickup and the last
delivery (place, window in the stop's zone) and who drives each move, or that it needs a
driver (`ShipmentOptions.IncludeRoute`, loaded with the page as batched relations), and the
search matches customers by name or code and stops by location name or city
(`shipment_search_parties.go`) besides pro number and BOL. Without them, "which one picks up
first" and "the sunbelt load to chicago" cost one `get_shipment` per row.

### Reads bunch into one table

Every `get_*` call used to leave its own entity card, so checking five
invoices put five cards beside the conversation. The loop remembers, per turn,
the calls of each `get_*` tool that succeeded and hands them to the next call
of that tool (`DispatchCall.Earlier`, carried to the observer as
`ToolObservation.Earlier`). A first call still makes its card; a later one
folds the turn's earlier cards of that tool into one `table_view` artifact
keyed by the turn's first call of the tool (`payload.bunched`, with the calls
it covers), adds its own row, and removes the cards it replaced. The cards a
table replaced are deleted and withdrawn from the reader with `artifact_removed`, which the
client's reducer drops from the live turn, so eleven reads leave one table beside the
conversation and not eleven cards next to it. When the turn
is saved, a card a table covers is not tied back to its message. A `get_*`
result that is already rows and columns (`get_invoices`) is a table from the
start. `Earlier` is activity input, not a workflow decision, so the recorded
histories replay unchanged.

### What the reply may claim

After the final completion of a turn that filed or executed writes, the reply
is checked against what the turn actually has (`groundingguard.go`):
`numberguard.CheckNumbers` over the figures the system prompt, the question,
the tool results and the filed arguments support, and the reply's claims
against the filed tools' own schemas: each collection property and each
multi-word field (derived with `toolschema.Walk`, ids left out) must carry a
value in some filed call, or be named in its filing result, before the reply
may say it is part of the change. A sentence that says the thing was left out
is not a claim. A hit discards the reply once, the way a looping reply is (a
`retrying` event, then the draft and a correction to the model); a reply that
still drifts, or one written after the tool budget was spent, ends with a note
to check the card. Each is a `reply_regrounded` event in the run's trajectory.
The LLM judge that scores the same thing stays in the evaluations.

Two rules the prompt gives are then enforced on every final reply, a
delegate's included, before it is recorded (`replypass.go`): internal record
ids are taken out the way the web client's `withoutRecordIds` takes them out
(`shared/recordids`, the same cases; an `artifact:` link's address is left
alone), and a markdown table of six or more rows that reprints a table the
turn kept beside the conversation, and does not point to, is replaced by one
sentence pointing to it with `ArtifactRef`. A reprint is recognised when at
least half its first column is the kept table's first column
(`ShownArtifact.Labels`, read from the artifact's rows by the observer and
carried to the loop on `ToolOutcome.Shown`), or when the line above it and its
first header name at least half the table's title words. The reply has
already streamed, so a changed one is sent whole in a `reply_replaced` event,
which the reader puts in place of what streamed; it is not a `retrying`
restart, since the model was not asked again and the person would otherwise
see the answer appear to fail and start over. Each pass that changed something
is a `reply_regrounded` event (`strip_ids`, `point_to_table`) before it. The
passes sit beneath the prompt's rules, not in place of them.

The output guard (`agentguard.EvaluateOutput`) runs last, on every final
reply. A reply with code in it is no longer refused whole: a fenced block with
a programming-language tag, a fenced block holding source or markup, and a
function or markup line with its body are each replaced by "[A code block was
left out: this assistant handles transportation work, not software.]", and the
rest of the answer is kept. The decision is still the signal that something
upstream let a code request through: it is `Altered`, names the rule, and is
logged; the reply's message keeps the guard's stage, category and reason, the
run carries `OutputAltered` and `OutputRule`, and the turn's decision is the
altered one. The reply as recorded is sent in a `reply_replaced` event, which
replaces the streamed one without a retry.

### An oversized tool result

A result over 12,000 characters (`maxToolResultChars`, `agentruntime/fence.go`) is
shortened before the model reads it. A JSON object keeps every field that is not a list
whole and writes those first, then shortens its top-level lists to fit, with a note
saying how many records each list shows and to answer counts from the totals. Cutting
by bytes dropped exactly those fields: keys arrive sorted, so the dispatch board kept
its drivers and moves and lost the summary that held the uncovered and late counts.
A result that is not a JSON object, or does not fit even with its lists shortened, is
still cut at a character boundary, with a note saying records are missing and not to
count or infer from them.

### History replay

A turn replays the newest 120 messages, and their tool results are most of it.
`replayHistory` (`agentruntime/messages.go`) keeps a result whole only while it
is in the last three turns (a decision note is not a turn) and, newest first,
within 48,000 characters; the rest are shortened to their first 320 characters
inside a fence of their own, closed, with a note after it naming the tool to
call again and saying not to quote what was left out. A result a proposal
answered is replaced by its current outcome and never shortened. The figures of
every shortened result travel with the turn (`TurnState.Evidence`), so the
grounding guard above still counts them as read.

### The clock

The system prompt names today's date only, so its cached prefix is the same all
day. The question the model reads (never what the conversation keeps) opens with
`Now: Weekday YYYY-MM-DD HH:MM zone` (the weekday so "next Friday" needs no asking) in the organization's zone, UTC when the zone is
unknown, for an agent that reads the clock. `ask_user` option labels that carry a
date read "Tue Oct 6 (in 7 days)", counted from the turn's own clock
(`TurnEffects.Now`) in that zone; a past date on a scheduling question adds a note
telling the model so.

### What a provider can cache

A provider reuses the start of a prompt only while it is the same bytes, so a
request is laid out stable first:

- **Bytes.** Every request body, the tools and the replayed tool-call arguments
  included, is encoded with sorted keys (`modeladapter.requestJSON`); a map in
  Go's random order made every request different.
- **System prompt.** `BuildSystemPromptParts` writes what every turn of an agent
  shares first and the turn's own part after it. The shared part is, in order:
  the identity line (the agent's name and description), Trenova's rules
  (Boundaries, Working with records), the tools when all are offered (grouped
  by what a call does, with the plural twins and the dashboard note only for an
  agent that holds those tools; the request's schemas carry the descriptions),
  the agents it may ask, artifacts, the product guide, how to answer, then the
  organization's instructions and its Never list. The turn's own part is the
  runtime context, memories, a disclosed tool list and proposals waiting on a
  decision. Precedence is the order on the page: Trenova's sections bind the
  organization's, which bind memories, which bind the message. The turn
  carries the shared part's length (`TurnState.SystemStable`) to the adapter.
  `agentdefinition.PromptVersion` names this shape (v7) on runs, evaluation
  cases and fingerprints; the snapshots under
  `domain/agentdefinition/testdata/prompts` are regenerated with
  `go test -run TestPromptSnapshots ./internal/core/domain/agentdefinition/ -update`.
- **Anthropic** gets three of its four marks: the last tool, the end of the
  shared system part, and the last block of the conversation, which lets each
  call of a tool loop read the exchange before it back from the cache.
- **OpenAI** gets `prompt_cache_key`, a hash of organization, agent and version,
  only when the provider is reached at `api.openai.com`; a server speaking the
  same protocol elsewhere may refuse the field. Everything else (vLLM, Ollama,
  Gemini, DeepSeek and the rest) caches prefixes on its own and needs only the
  layout.

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

### Who a run acts as

A run nobody is in has two identities, kept apart on purpose.

- **Authorization is the agent's.** The actor `runRequest` builds is
  `PrincipalTypeAgent`, so the permission engine judges every call against the
  fixed agent table (`permission.IsAgentAllowed`), never against a role. Approving
  is refused outright (`guardExecute`), a self-scoped tool is refused because
  nobody is in the run, and nothing that reads a person's access (field
  sensitivity ceilings, report authorization, the user context provider, usage
  attribution) reads the system user's: they read `RequestActor.PersonUserID`
  or check `IsUser`, and both leave out anyone but a person. The database scope an agent actor binds
  (`RequestActor.DBTenant`, which the tool activity and the tenant interceptor
  read) names no user either, so row-level security shows the run nothing the
  system user's own rows or memberships would.
- **Attribution is the system user's.** The same actor carries the instance's
  system user (`UserRepository.GetSystemUser`, the `system` account) as its
  `UserID`, so a record the run creates or changes names that account in its
  created-by and updated-by columns instead of nobody, and a service that
  refused a write it had no user to attribute to (a shipment comment, a hold)
  takes one the agent table allows.
- **The agent is named alongside.** The actor's `PrincipalID` is the agent
  definition's id rather than the generic `agent`. The audit log row an
  unattended write leaves is principal `agent` with that id, `user_id` the
  system user, and a description that ends "(Ran by Dispatch Agent)":
  `auditservice` reads the definition's name in the row's tenant (cached for ten
  minutes) and appends it, or "Ran by an agent" when the name cannot be read.
  `chk_audit_entries_principal_consistency` lets an agent row carry a user, never
  an API key, and never the user as its own principal (migration
  `20261231007250_audit_agent_system_user`).
- **The system user executes the run's automatic writes.**
  `RequestActor.ExecutorUserID` is the person for a user principal and the
  carried system user for an agent, so `executed_by_user_id` on an automatic
  write of an unattended run names the system user, in the recorder and in the
  executor alike. The AI audit trail derives that execution as a `User`
  principal write by the system account, with the agent on `agent_definition_id`
  and `agent_name`, and looks for its audit log rows from the start of the run.

A run with a person in it is unchanged: its actor is the person, so its writes,
its audit log rows and its executions are theirs.

The system user is resolved once per run, when the run is opened, again when its
proposals are filed (only when it raised any), and when an evaluation replays a
background run or case. If it cannot be read, the attempt fails with a retryable
error rather than running unattributed. A scheduled run's start is recorded as
the agent, by its definition, with no user: the scheduler, not the agent, opened
it.

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
  hour, backfilling schedules and removing orphans. A firing starts
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

Every case's reply is also scored on its form (`agentscoring` check
`replyForm`, weighted beside the others and needing no rubric): it fails a
reply that shows an internal record id (`shared/recordids`) or writes out a
markdown table longer than a dozen rows (`shared/mdtable`) instead of pointing
to the table its tool kept. The runtime enforces both beneath the prompt, so a
failure here means a model, a prompt change or the reply pass let one slip.

The runtime's own contract with a model is held by scripted conversations,
`agentevalgate/evals/conversation.yaml`, run through the kit's runtime
(`agentevalgate.RunConversation`) with each case's stub reads as the only
tools: a list reprinted as a markdown table is recorded as a pointer to its
table, internal ids never reach the recorded reply, `"limit": "30"` and a lone
id for a list reach the tool as the number and the list, and a value outside an
enum is refused with the allowed values and the corrected call runs. The
model's side is scripted, so a case fails on the runtime and never on a model;
every case must pass.

### Dry runs

`AgentDryRunWorkflow` is the agent builder's "Try it": a draft agent, unsaved,
answers one message against live data with simulation forced on. The draft
travels whole in the payload, because there is nothing stored to read it back
from; `AgentDefinitionService.Draft` builds and checks it exactly as a save
would, without saving. It runs on the chat queue because a person is watching
it, hosts a Workflow Stream like a chat turn, and is read on the same request
that started it (`POST /agent-definitions/dry-run/`), so only the person who
asked can read it; a reader who leaves cancels it.

It leaves nothing behind but what it cost. Its run ID carries the evaluation
prefix, so usage is counted as an evaluation and no proposal baseline is kept;
it has no thread, no step ledger and no publishing, and no finish step files
its proposals. Before `done` it publishes `dry_run_steps`: each of the draft's
own tool calls with what it would have come to (`agentdryrun.Classify`: Runs,
AskFirst, Propose, Recorded for a shadow agent, Simulated, NotHeld, Failed).
A memory the prompt reads still has its use counted.

## One-shot calls

`completionjobs` runs the model calls a person waits on outside a conversation.
The request starts a short workflow on the chat queue and waits for its result;
the workflow's budget is the request's own deadline less the margin the handler
needs to answer, so a call never outlives the request waiting on it. A person who
stops waiting cancels a call that was theirs alone.

- `StructuredCompletionWorkflow` asks one structured question (table compose,
  and the agent builder's drafting and tightening, routed as `AssistantChat`),
  retried the `modelcall` way. Formula generate and explain no longer call it;
  they are turns of the formula assistant.
- `TestAIProviderWorkflow` probes once and never retries: the administrator is
  asking whether the connection works now.
- `ListAIProviderModelsWorkflow` and `TestAIProviderDraftWorkflow` ask an endpoint
  the editor has not saved, once each. A key typed into the editor is encrypted
  before it is handed over, so workflow history holds only ciphertext, and a
  stored key is used only for the endpoint it was entered for.
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

A background run also keeps its **transcript**: what its model said and thought,
the tools it called with their arguments, and what each returned with its
verdict, in `agent_runs.transcript` (JSONB). `settleRun` writes it from
`RunResult.Messages` on both paths that file a run (the activity and the
workflow's `FinishRunActivity`), beside the 2,000-character summary, and a
retried filing writes the same transcript again. It is bounded so the row stays
small, the same way the event log bounds a payload: a message whose encoded form
is over 64 KiB keeps only who said it, what it called and how the call was
judged, marked `omitted`, rather than a clipped body; and past 256 KiB in all
the middle of the run is left out (`sliceutils.KeepEnds` keeps the opening and
the end), with `omittedMessages` and `omittedAt` saying how many and where. A
run that produced no messages, or was filed before transcripts were kept, has
none. The column is left out of every run read (`GetByID`, lists, the AI audit
projector's source read) except `ListTranscriptsByIDs`, so filing, listing and
updating a run never carry it, and it is hidden from the run's JSON so a
realtime invalidation never ships it. Read it through `AgentRun.transcript`,
which checks the run read permission itself and loads through a per-request
dataloader; AI Control's run panel shows it behind a Transcript disclosure, with
the conversation's own tool rows.

The disclosure's **Download** saves it as a file:
`GET /api/v1/agent-runs/:runID/transcript/`, gated on reading an agent run and
read under the caller's tenant (another organization's run is not found). It is
served by `assistantservice.Service.RunTranscript` through the narrow
`AgentRunTranscriptService` port, as `text/markdown` with a server-chosen file
name and `no-store`, exactly as a conversation's download is. Both documents
come from one renderer in `assistantservice/transcript.go`
(`renderTranscriptDocument`), which takes a message list, a heading and the
proposals: a conversation passes its thread's messages, a run converts its
stored entries back with `conversation.MessageOfTranscript` and passes the
proposals it raised (`ListByRun`). The run's heading names its agent, trigger,
status, start and finish and model; a message kept without its body says it was
too long to keep; and the left-out middle is stated twice, in the heading's
message count ("40 kept, 12 left out") and as a line where it fell, and a
delegated task's steps on either side of that line are never joined across it. A golden file
(`testdata/conversation_transcript.golden.md`) holds the conversation's
download byte for byte, so a change to the shared renderer that moves it fails.

The transcript lives on the run row and goes with it: no sweep prunes runs or
their events today, and the row is deleted only with its organization, by the
existing cascade, so nothing has to keep the transcript and the event log in
step.

An event's `occurred_at` is the instant the workflow emitted it
(`StreamItem.At`, stamped with `workflow.Now`), not when the filing activity
wrote the account, so a day-long run's events keep their real spacing.

Each claimed step also carries the trace and span of its `execute_tool` span, the
definition and version that made the call and, for a delegate, the call id; its
outcome carries a one-line `reason` and a `verdict` (`ran`, `proposed`,
`denied`, …). A call the loop turns away before any step is claimed (a tool the
agent does not hold or that does not exist, arguments that did not parse, a spent
budget, a repeat, a question nobody can answer) carries its verdict on the
`tool_finished` event instead. The tool message the turn saves keeps the same
verdict (`assistant_messages.tool_verdict`, `toolVerdict` on the thread), and the
chat draws a refusal in its own words rather than as a failure: `denied` reads
"Not permitted", `invalid` "Not accepted", `over_budget` "Out of budget" and
`duplicate` "Skipped (repeat)", while `failed`, an unknown verdict, and a result
saved before the verdict was kept read as failed. A refusal is still a failed
result to everything that counts failures. `get_agent_run` lists each step's
verdict too. A proposal carries the same trace and span, its
`step_key`, when an automatic write ran and at what version it left the record,
and who it ran as.
The whole table of link columns, and who writes each, is in
[ai-tracing.md](ai-tracing.md#link-columns).

## Measuring it

`turn_first_event_seconds` is separate from `turn_duration_seconds` on purpose:
duration says how long the answer took, first-event says how long a reader who
attached as the turn began stared at nothing.

`assistant_tool_outcomes_total{tool,verdict}` counts every tool call the loop
answers, refusals included, by what became of it (the `aitrace` verdicts: ran,
proposed, simulated, denied, invalid, duplicate, over_budget, failed, unknown).
It is counted in one place, `recordToolResult`, and not while workflow code
replays its history (`ReplayAware`); a name the model invented is counted as
`unregistered`, so a confused model cannot mint a series per guess. The agent's
scorecard carries the same breakdown per agent (`AgentScorecard.toolVerdicts`):
counted from `agent_run_steps` by tool and verdict, with the three reasons given
most often, so "which tool do the models keep calling wrongly, and how" is a
read rather than an afternoon in threads. The ledger holds what reached
dispatch; the loop's own refusals before it (an unparsed call, a tool not held,
a spent budget) are only in the counter.

`trajectory_events_total{result}` counts dropped events. They are invisible by
design — the run carries on — so this is the only place they show up.

`gen_ai.client.token.usage` and `gen_ai.client.operation.duration` are recorded
per provider attempt, with the GenAI semantic-convention buckets.
`trenova.gen_ai.client.time_to_first_token` (and `first_token_ms` on the usage
row) is how long an attempt took to stream its first text or thinking. Before it,
`assistant_prepare_seconds{outcome}`, `assistant_guard_seconds{stage}` and
`assistant_model_call_wait_seconds` time preparing the turn, the scope check and
a model call's wait for a worker.

Every run, turn, delegate's task and evaluation is one trace, named by its id and
rooted in an `invoke_agent` span its finishing activity emits; model calls, each
provider attempt, tool calls and writes hang beneath it, and a later decision
links back to the call it answers. **Read [ai-tracing.md](ai-tracing.md) before
adding a span, a trace attribute or a link column.**

## Running it in production

- **Server.** Workflow Streams uses Updates and Signals, on by default from
  server 1.29. Fairness needs `matching.enableFairness=true`. The local dev
  server (`temporalio/temporal`) sets it.
- **Update limit.** Every poll a turn's reader makes is an Update on the turn's
  workflow, about ten a second while a reply streams and one more per reader
  tab. The server caps a workflow at `history.maxTotalUpdates` (default 2000),
  which one reader reaches after a little over three minutes of streaming;
  past it every poll is refused and the person is told the connection was lost
  while the turn goes on finishing on the server. Set it to `20000` on
  the namespace the workers use, as the local dev server does.
  `history.maxInFlightUpdates` (default 10) bounds readers following one turn
  at once and can stay.
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
| `weather-alert-poll-fetch-once` | `pollPerTenant` | `ListWeatherAlertTenantsActivity`, `PollNWSAlertsForTenantActivity`, and `WeatherAlertService.ListWeatherAlertTenants` / `PollNWSAlertsForTenant` |
| `agent-loop-final-answer` | a turn that spends its tool budget ends on the canned `exhaustedReply` without asking the model for an answer | nothing; the check itself is the only cost |
| `assistant-turn-close-unsaved` | a turn whose save fails on every attempt leaves its record Running, and the conversation refuses every later question | nothing; the check itself is the only cost |
| `assistant-turn-notify-unseen` | a turn that ends with nobody reading its stream ends without telling the person who asked | nothing; the check itself is the only cost, and it is asked only of a turn nobody drained |
| `assistant-turn-prepare-local` | a turn prepares as a regular activity: a task-queue dispatch and a second workflow task before the model call is scheduled | nothing; `PrepareTurnActivity` stays registered, and the old branch keeps its priority and fairness keys |
| `agent-loop-grounding-guard` | a reply that names what the filed writes do not hold is kept as written | nothing; the check itself is the only cost, and it is asked only of a reply the guard found wanting |
| `agent-loop-reply-passes` | a final reply is recorded as the model wrote it, internal ids and reprinted tables included | nothing; the check itself is the only cost, and it is asked only of a reply the passes would change |
| `agent-loop-cut-off-call-retry` | a completion cut off inside a tool call, or before any visible answer, ends the turn with `truncationNotice` (a broken native call is refused as invalid JSON and the model asked again at the same limit) | nothing; the check itself is the only cost, and it is asked only of a completion cut off that way |
| `agent-loop-interjections` | a turn never reads what the person said while it worked, nor checks its records for changes; a steer waits in the queue and is sent as the next message | nothing; the check itself is the only cost, and it is asked only when a steer is waiting or a check of the records is due |
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
both are optional data, and a history without them replays by keyword. The person's other
agents that answer named as holding what the agent could not call ride the same way, on
`FindToolsResult.HandOff` into the message's `hand_off_agents`; a history without them
replays with none, which only leaves the Desk's hand-off menu in its usual order. See "Ranking" in
[ai-retrieval.md](https://github.com/emoss08/trenova-documentation/blob/main/docs/engineering/ai-retrieval.md).

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

Resolving a criteria selection to records (`ToolSelectionResolver`, `transfer_to_billing`'s
`allTransferable`; `ToolProposalSelectionResolver`, `retry_accounting_sync`'s
`errorCategories` when proposed) took no gate: it happens inside the dispatch activity, the
ledger key is still derived from the model's own arguments, and only the activity's result
changes. See [proposal-previews.md](proposal-previews.md#record-subsets).

Agent delegation (`delegate_task`) took no gate: whether a turn holds the tool
is decided when it opens, in an activity, and kept in `TurnState.Held`, so an
execution opened before it never takes the new branch. Keeping the hand-off's
account structured on the saved result (`delegateReport`) and `record` on write
results added only optional data, no command. Handing a task its records and shared
results (`DelegateCall.Context`) is optional data on the activity input, nil for a call
made before it existed; filing a turn's and its delegates' writes as one ordered plan,
and a run for a read-only hand-off, happen in `FinishTurnActivity`. See
[agent-delegation.md](agent-delegation.md#versioning).

Runs parked in a day-long decision wait are the slowest to drain; the recorded
histories under `agentjobs/testdata/replay` replay against the old branch and must
keep passing until it goes. The `agent-queue` drain registry goes on the same
condition: no open execution on `agent-queue`.

`workflowstarter.Enabled()` is always true now that the client connects lazily;
the branches that test it are dead and can go with the next change that touches
each of them.

## Billing lifecycle tools

The tools that take a delivered shipment to a sent invoice (transfer, the billing queue
decisions, posting and sending), their tiers, and why approving, canceling, posting and
sending always stop at a proposal a person approves, are described in
[agent-billing-tools.md](https://github.com/emoss08/trenova-documentation/blob/main/docs/engineering/agent-billing-tools.md). An agent principal may move a billing
queue item into review, onto hold, into exception or back to operations; the queue refuses
it every other status. The steps over a set of items (`assign_billing_queue_billers`,
`transition_items_to_in_review`, `approve_billing_queue_items`, `post_invoices`,
`send_invoices`) are one call and one card each, and a biller nobody named is the person
asking; see [proposal-previews.md](proposal-previews.md#bulk-twins-of-single-record-tools).

Three chat templates that act as the person hold the money tools: the billing assistant (up
to an invoice in the customer's hands, and its corrections), the receivables assistant (what
the customer pays, disputes and owes late) and the settlements clerk (driver and carrier
settlements, carrier invoice matching, advances and escrow money). The settlements clerk reads
a driver's pay profile, recurring pay and escrow accounts but holds none of the tools that
change them, so the one who processes pay is not the one who sets it. None of them runs
unattended, so the agent permission ceiling does not grow for them.

## Tools no template holds

Every action tool is on a starter template unless it is deliberately withheld, and
`withheldFromEveryTemplate` (`agentdefinition/focused_templates_test.go`) keeps each withheld
tool off every template with its reason. An organization can still add any of them to an agent
it builds in AI control. The privileged accounting writes are explained in
[agent-accounting-tools.md](agent-accounting-tools.md#who-holds-them), rate agreement review in
[agent-rates-tools.md](agent-rates-tools.md#who-holds-them), and pay setup, payroll exports and driver
expenses in [agent-workforce-tools.md](agent-workforce-tools.md#who-holds-them). The rest:

| Tool | Why no template holds it |
| --- | --- |
| `cancel_shipment` | Canceling a shipment is a person's call, by owner decision: it withdraws live tenders from carriers and can tell a partner over EDI. `uncancel_shipment`, which puts a canceled shipment back to New and sends nothing, stays with the dispatch assistant. |
| `forget_memory` | A person retires a memory in AI Control; see [Forgetting](#memory-in-the-prompt). |
| `save_table_view`, `add_home_widget`, `remove_home_widget`, `arrange_home_layout` | Each changes only the caller's own screen, a person's interface state that is no desk's job; the report analyst, the one desk near dashboards, builds report dashboards and never the person's home page. |

## Known limits

- **A world change is best effort.** It is read from the realtime bus, which
  drops events under publish backpressure, trims old entries and carries no
  event addressed to one person; a check scans at most 2,048 entries and picks up
  where it stopped. A write that publishes no record id (a bulk change) is not
  seen.

- **Resume is at-most-once.** A crash in the execute→settle window reports
  "began, outcome unknown" rather than replaying.
- **Search attributes are not set.** Organization, feature, thread and
  definition are carried in workflow ids, summaries and fairness keys; typed
  search attributes need registering on the server first.
- **A run's trace root is emitted when the run is filed.** A run parked in a
  decision wait shows its root once it is filed, not when the last proposal is
  decided, and a workflow evicted from the cache may never export its
  `RunWorkflow` span. The limits of tracing are listed in
  [ai-tracing.md](ai-tracing.md#known-limits).

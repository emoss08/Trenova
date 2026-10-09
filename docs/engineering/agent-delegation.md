# Agent delegation

How the agent a person is talking to hands a task to another agent, what that
agent may do, and what comes back. Read [agent-runtime.md](agent-runtime.md)
first: delegation is the same loop, driven one level down, inside the same
workflow.

## What it is for

The Homepage Widget Builder holds dashboard tools and no report tools. Asked
"build a dashboard of this month's on-time delivery", it hands the Report
Builder the task "create a report of this month's on-time deliveries and return
its id", reads back the new report's id, and adds the tile itself.

## The decisions

These were made by the product owner and the code enforces each one.

| Decision | Where it is enforced |
|---|---|
| Who an agent may ask is a **per-agent allowlist** on its definition, set in AI Control. | `agentdefinition.Definition.DelegateIDs`, validated on save by the domain and by `agentdefinitionservice.validateDelegates` |
| A delegate keeps **its own autonomy**: its tools, tier settings, autonomy ceiling and budget. A private write it may make alone (a personal call of a tool whose policy sets `PersonalRunsUnasked`) runs, unless a person set that tool's tier on the delegate; a tier the trust ledger earned (`agent_tool_trust.earned_tier`) does not count as one (`agentruntime/tierprovenance.go`). A shared or outbound write still waits for approval. | The delegate's turn is opened from its own definition (`assistantservice.OpenDelegate`), not unattended, so `dispatch` decides every call exactly as it would for a conversation with the delegate |
| **One level only.** Only the agent the person is talking to delegates. A delegate never holds `delegate_task`, and a call to it from a delegate is refused. | `RunRequest.MayDelegate`, `OpenTurn` (the tool is held only when it is true), `Service.delegate` (refuses when `RunRequest.Delegation` is set) |
| **Taint crosses both ways.** A delegate opens with the taint of the turn that asked, and its own taint is folded back into that turn when it ends, because its reply enters that turn's context. Its proposals carry its turn's taint. | `DelegateCall.Taint` → `RunRequest.Taint` (`assistantservice.OpenDelegate`), `Turn.inheritDelegateTaint`, `DelegatedRun.Taint` → `persistDelegatedProposals` |
| **Everything runs as the same person.** Every tool call of the delegate is permission-checked against the person, so a delegate can never do more than they could. The person must also be allowed to use the delegate. | The delegate's turn carries the turn's `RequestActor`; its tool set is narrowed by `permittedTools`; `OpenDelegate` requires `assistant:create`, a delegate the person may use (open to everyone, or granted to one of their roles), and a delegate that is enabled, chat-usable and in the same tenant |

## The allowlist

`agent_definitions.delegate_ids` is a `TEXT[]` of agent definition ids in the
same organization and business unit, at most `MaxDelegates` (8). An array fits
the table's other lists (`tool_names`, `event_kinds`) and is read with the row,
so a turn never joins to learn who it may ask.

Saving checks, in order:

- **Domain** (`Definition.Validate`): at most 8, no duplicate, not itself, and
  only on an agent people talk to (`TriggerChat`) — a run nobody watches has
  nobody the other agent's work would be for.
- **Service** (`validateDelegates`): each id is an agent in the tenant (read
  with `ListByIDs` scoped to it, so another tenant's agent is simply not
  found), is chat-usable, and — when this save adds it — enabled. One already on
  the list that was disabled since stays, and is refused when asked, so
  disabling an agent does not leave every agent that delegates to it unable to
  be saved.

**Deleting an agent removes it from every allowlist in its tenant**, in the
delete's transaction (`agentdefinitionrepository.removeDelegate`). An array
cannot carry a foreign key; this is the cascade. Each agent changed gets a new
version, so a save from a copy loaded before the delete is refused rather than
putting the deleted agent back.

The REST save body (`POST`/`PUT /agent-definitions/`) takes `delegateIds`:
absent or `null` keeps the list, `[]` clears it, a list replaces it. GraphQL
reads it as `AgentDefinition.delegateIds` and resolves `AgentDefinition.delegates`
through the request's `AgentDefinitionByID` loader, so a page of agents costs
one query for all their delegates.

`AgentDefinition` is an administrator's read. The person talking to an agent
reads `MyAgent.delegates` instead (`myAgents`, needing only `assistant:read`):
the allowlist in order, as `MyAgent` again (who each is and its mark, never its
set-up), keeping only delegates that person may use — the rule the turn applies
(`ContextBuilder.delegates`): in the tenant, enabled, chat-usable, not the agent
itself, and open to them or granted to one of their roles under
`assistant:create`. It is read through the request's `UsableAgentByID` loader
(`loaders/usableagentloader.go`), keyed by delegate id and built from the
request's tenant and user: one `ListByIDs` and one `PermissionEngine.AgentsUsable`
per batch, so a page of agents costs one query and one permission read for all
their delegates. The chat client hands the thread agent's list to
`AssistantAgentProvider`, so a hand-off saved before the thread served agents'
marks still draws the delegate's icon and accent.

## What the primary agent is told

`agentruntime.ContextBuilder` resolves the delegates for a conversation turn
(`RuntimeContext.Delegates`): the allowlist, in order, keeping only agents in
the tenant that are enabled and chat-usable and that the person may use (open to
everyone, or restricted to roles and granted to one of theirs, through
`PermissionEngine.AgentsUsable`), and none at all unless the person holds
`assistant:create`. Each delegate is described by its name, id, the first
sentence of its description, and the tools it holds **that the person may
use**, so a delegate is never advertised by what the person could not have it
do.

- The system prompt gains **Agents you can ask**, fenced in
  `<delegate_agents>` because names and descriptions are typed by an
  administrator.
- The turn holds `delegate_task`, whose `agentId` is an enum of exactly those
  ids.
- `find_tools` that finds nothing the turn can call names a delegate holding
  what matched, before it falls back to "an administrator can add it". It
  names one as well when the best new match is weak (no name or description
  hit, a parameter only) and a delegate holds a tool matching by name. Both
  searches past the turn's own tools are `StrongOnly`.
- When no delegate holds what matched, the answer names the person's other
  agents that do (`agentruntime.handOffNote`, from `OtherUsableAgents`: the
  enabled chat agents they may use, the same list the Desk's case checklist
  reads to say who can take a step) and tells the model to point the person
  to Hand off to another agent; only when none holds it does it fall back to
  "an administrator can add it in AI Control". The list is read only on that
  answer, never on an ordinary turn.

The same context is what `PreviewPrompt` shows in AI Control.

## `delegate_task`

| | |
|---|---|
| Name | `delegate_task` |
| Effect | `delegate` (`agent.ToolEffectDelegate`) |
| Arguments | `agentId` (one of the turn's delegates), `task` (what to do and what to hand back, at most 4 000 characters), `records` (optional: the records the task is about, each `{entityType, id}`, at most 8), `shareResults` (optional: ids of calls this turn made whose results the delegate should work from, at most 4) |
| Cost to the primary | one tool call of its own budget |
| Limit | 3 per turn (`maxDelegationsPerTurn`); the fourth is refused and counted |

The loop (`agentruntime.Service.delegate`, in `Drive`) checks the arguments,
emits `delegate_started`, hands the task to `TurnEffects.Delegate`, folds what
came back into the turn, and emits `delegate_finished`. The call is held to its
whole schema (`toolschema.Validate`), so an undeclared key, a record kind the
registry does not have or too many entries is refused with each problem named,
before anybody is asked and without counting as a task handed out.

### What a task carries besides its words

A delegate cannot see the conversation, and a task written in prose is a copy
the delegate has to take on trust: the Dispatch desk once described a shipment
to the Shipment Desk, which read it again and retyped it wrong. So the task
names what it is about instead:

- **`records`** are `{entityType, id}` pairs. `entityType` is a key of the
  record-link registry (`agenttoolschema.RecordEntities`, the same source
  `open_page` spends, `x-enumOf: guide.entity`); an id is letters, digits, `_`
  and `-`, at most 100. Duplicates are dropped.
- **`shareResults`** are ids of tool calls this turn made. Each is read from the
  turn's own messages by `ToolCallID` (never a delegate's step), unfenced, and
  handed over as the tool returned it. A call that failed has no result and is
  refused. A result is handed over **whole or not at all**: one over 8 KiB, or
  results over 24 KiB together, are refused with the advice to hand over the
  record instead, because half a document read as data is worse than none.

What was handed over rides `DelegateCall.Context` (`Records`, `Results`) to
`OpenDelegateActivity` and `assistantservice.OpenDelegate`, which asks the
delegate `agentruntime.DelegateInput(task, context)`: the task, a line saying
what follows is data from the agent that asked, the records fenced in
`<untrusted_data>`, and each result fenced as `Result from {tool} (call {id})`.
The saved task step carries the same text; the thread shows the task as the
asking agent wrote it (the call's `task` argument).

Both prompts say how to use it. "Agents you can ask" tells the parent to hand
over records and results rather than retype them, to ask for a record to be
made with a tool rather than described (a copy with `duplicate_shipment` or the
tool that copies it, naming the record), and to name each waiting proposal by
its `proposalId`. A delegated turn's output section tells the delegate to work
from what it was handed, open each record by its id, and copy a record with
the tool that copies it. A turn holds the tool
only when `RunRequest.MayDelegate()`: a person is reading, the turn has a
conversation, it is not itself a delegate's turn, and at least one delegate
survived the checks above.

## Running the delegate

`agentflow.workflowEffects.Delegate`:

1. `OpenDelegateActivity` asks the `DelegateOpener` (the assistant, through
   `assistantjobs.delegateOpener`) to open the delegate's turn. Every check is
   made again against the records as they are now: the primary still lists the
   delegate, the delegate exists in the tenant, is enabled and chat-usable, the
   person holds `assistant:create` and may use the delegate itself, and the
   delegate's own budget
   (`AgentBudgetService.CheckRun`) is not spent. A refusal is
   `DelegateDeclined`, non-retryable, and its reason is what the primary reads.
2. The turn is opened **as the delegate** (`runtime.OpenTurn`): its tools
   narrowed to the person, its tiers, its ceiling, its model, its prompt — with
   `RuntimeContext.DelegatedBy` set, so its output section says it answers the
   agent that asked, and with `RunRequest.Delegation` set, so it holds neither
   `delegate_task` nor `ask_user` (it has nobody to ask). It is **not**
   unattended: a write that is the person's own still runs as theirs.
3. The delegate's loop is the same `Drive` with the same effects: each model
   call and each tool call is an activity of **this** workflow execution. Its
   model calls are attributed to the delegate's definition, so its spend counts
   against its own budget; its automatic writes are checked against its own
   daily tool caps.

### Inline, not a child workflow

The delegate's loop runs inline in the conversation turn's workflow rather than
as a child workflow:

- **The stream.** The model activity publishes the streamed reply to the
  Workflow Stream of the workflow it belongs to. A child's model calls would
  publish to the child's stream, which nobody reads; the reader would have to
  follow two streams, or the parent would relay every token. Inline, the
  delegate's words reach the one stream the reader already follows.
- **Stop.** Stop cancels the turn's workflow. Inline, the delegate's in-flight
  model or tool activity is cancelled by the same cancellation, and what it did
  before the stop is saved with the turn, by the same save. A child needs its
  own cancellation plumbing and its own save.
- **One record.** The turn is saved once, by one `FinishTurnActivity`, with the
  delegate's steps, writes and artifacts in it. A child would hand the same
  things back through its result, for no gain.
- **Bounded history.** At most three delegations per turn, each bounded by the
  delegate's own tool budget (at most 64), keeps the execution well inside
  Temporal's history limits.

### The step ledger

The delegate's turn shares the turn's `StepOwner`. Its step keys are
namespaced by `Delegation.StepScope`, which is itself the step key of the
`delegate_task` call (owner, tool, arguments, ordinal) — derived from what the
primary model asked for, never from a provider's call id. So a retried tool
activity of the delegate claims the same key and is answered from the ledger,
two delegations never share a key, and the turn's own keys are exactly what they
were before delegation existed (`StepKey` adds the scope only when it is set).
The delegate's turn does not seed its repeat guard from the ledger: it is opened
once per task, and the ledger's lessons are the primary's.

### Call ids

The delegate's steps are kept in the same thread as the primary's. The
delegate's turn is handed the conversation's call ids (`DelegateCall.CallIDs`)
and the primary takes the delegate's back when it ends (`Turn.ReserveCallIDs`),
so a provider that reuses an id gets a fresh one rather than two calls under one
id in the thread.

## What comes back

The primary reads one tool result for the call: the account of the task,
fenced as untrusted data (the delegate's reply may quote records), followed by a
note on how to read it. It is the same object the reader receives as
`delegate_finished`:

```json
{
  "delegateCallId": "call_…",
  "agentId": "agdef_…",
  "agentName": "Report Builder",
  "icon": "receipt",
  "accent": "teal",
  "status": "completed",
  "reply": "I saved the report \"On-time this month\".",
  "reason": "",
  "made": [{"toolName": "create_report", "callId": "call_…", "tier": "AutoExecute",
            "summary": "On-time this month",
            "result": {"action": "created", "kind": "report", "name": "On-time this month",
                       "ids": {"definitionId": "rd_…"},
                       "record": {"entityType": "report", "id": "rd_…"}}}],
  "awaiting": [{"toolName": "share_report", "callId": "call_…", "tier": "Propose",
                "summary": "On-time this month", "proposalId": "ap_…"}],
  "published": [{"id": "aart_…", "kind": "document", "title": "Report notes"}],
  "toolCallsUsed": 2
}
```

| `status` | Meaning | The call |
|---|---|---|
| `completed` | The delegate answered. | succeeded |
| `exhausted` | It spent its tool budget; `reply` is its best answer. | succeeded |
| `refused` | The output guard withheld its answer; `reason` is the refusal. Only reports from before the guard took code out of a reply instead of refusing it carry this; a delegate's reply with code in it is now `completed` with the block left out. | succeeded |
| `declined` | It could not be asked; `reason` says why. Nothing ran. | failed |
| `failed` | Its turn ended partway; `reason` says why. | failed |
| `stopped` | The person stopped the reply. | failed |

`icon` and `accent` are the delegate's mark: its own icon or the one its starter
implies (empty when it has neither, so a reader draws its initials, as
`Definition.ChosenIcon`), and its accent, chosen or derived from its id
(`ResolvedAccent`). They come from `RuntimeDelegate`, resolved when the turn's
context is built.

### Record references

A write's `result` is `agent.ToolExecutionResult`. `action`, `kind`, `name` and
`ids` are for the model: `ids` names each id by the parameter the next tool
takes it as. `record` is for a reader: the **one** record the write made or
changed, as `entityType` — a key of the record-link registry
(`client/apps/web/src/config/record-links.ts`, `productguide.Default.Record` on
the server) — and its `id`. A tool that creates or changes one identifiable
record sets it from `ExecuteWithResult`; today that is `create_report` and
`fork_report`, whose saved report definition is the registry's `report`
(`/reports/explore/{id}`). A tool that touches several records, or none a page
opens, leaves it out. `Bounded()` keeps it only when the entity is a registry
key (lower-case letters, digits and underscores) and the id is non-empty, cut to
one line. The same result is kept on the proposal (`agent_proposals.execution_result`)
and served wherever the proposal or the account is.

The client links a made write by `recordPath(record.entityType, record.id)` when
`record` names a registry entity, and only otherwise falls back to reading
`kind` as an entity and picking the id (`{entity}Id`, then `id`, then the only
one).

Each write carries `proposalId`, the proposal it was filed as (`PendingAction.ProposalID`),
and the note after the account tells the parent to name each waiting proposal by it, so
the person can find its card and the parent never searches for what was made.

A delegate that failed or was stopped is a failed call whose writes are still
named under `made`: they happened. Anything else it may have begun is
**unconfirmed** — the same rule as `unsettledToolOutcome` — and the note says
so; nothing ever claims a write did not happen.

## What is kept

- **Messages.** The delegate's steps — the task (role `User`), its model's
  messages and its tool results — are saved to the thread between the
  primary's `delegate_task` call and its result, with `kind: "Delegated"`,
  `agentId` and `delegateCallId`. They are **never replayed to a model**: the
  thread's reader for history passes `ExcludeKinds: ModelHiddenKinds()` before
  the limit is applied, and `OpenTurn` drops any that reach it anyway
  (`modelHistory`). The primary only ever sees its own call and its result.
  The messages API (and `done.messages`) returns them with `agentName`,
  `agentIcon` and `agentAccent` filled in, read for the whole page with one
  `ListByIDs` (`assistantservice.nameDelegatedSteps`). `agentIcon` is empty for
  an agent with no icon of its own or its starter's, so the client draws its
  initials; the client uses these before the conversation agent's delegate list
  and, last, a mark derived from the id.
- **The account.** The `delegate_task` call's result message (role `Tool`)
  keeps the account structured in `assistant_messages.delegate_report` (JSONB),
  served as `delegateReport` in the same shape as `delegate_finished`. It is
  written with the turn, from the `toolOutcome` the loop built, so a declined
  hand-off keeps one too; a call refused before anybody was asked (not offered,
  over the cap, from a delegate) has none. It is **bounded**
  (`conversation.DelegateReport.Bounded`): `reply` at 2 000 runes and `reason`
  at 500, each ellipsized; `made` and `awaiting` at 20 writes and `published` at
  10 documents, with what was left out counted in `moreMade`, `moreAwaiting`
  and `morePublished`; labels to one line; each write's `result` bounded as a
  proposal's. The model still reads the whole account in the result's text.
  When served, the account's `icon` and `accent` are refreshed from the agent as
  it is now, as the steps' are; an agent deleted since keeps what the turn
  recorded. The client reads `delegateReport` first and parses the fenced JSON
  in `content` only for a conversation saved before the column existed. Where
  the account cut the reply short and the delegate's own saved answer is longer,
  the client shows the saved answer.
- **Proposals.** The delegate's writes are recorded as its own:
  `RunResult.Delegations` carries each task's definition and actions, and
  `FinishTurn` records them with the turn's own in **one** `persistProposals`
  call (`proposalrecorder.RecordRequest.Delegated`), as a run of the
  delegate's definition whose subject is the same conversation. Decisions shows
  "Report Builder proposed…", trust accrues to the Report Builder, a shadow
  switch on it holds its cards, and the cards appear in the same conversation.
  The follow-up after a decision goes to the conversation, whose agent is the
  primary.
- **One plan per turn.** The recorder opens one run per agent that filed a
  write (the turn's own first), counts what waits on the person across all of
  them, and when two or more do, opens **one** plan whose steps are numbered in
  conversation order (`RecordRequest.CallOrder`, the position of each call in
  the saved messages). "Void and recreate", a void the primary proposed and a
  copy the delegate proposed, is one two-step plan run in the order asked,
  under one approval. The plan hangs on the run of the first agent with a
  waiting step and takes its name; each step keeps the run of the agent that
  filed it. Deciding it (`agentplanservice.Decide`) refuses a plan any of whose
  steps' agents is behind a shadow switch; deciding one's own (`DecideOwn`,
  `AssertOwnPlan`) also requires every step's run to be in the caller's
  conversation and access to each of those agents, the plan's own first. A
  thread's plan is served on hold when any step's agent is
  (`ListThreadPlans`). The follow-up names what each executed step made, in
  words on the line the thread shows and by id, step by step, on the agent's.
- **Runs, even for a read.** A hand-off that filed nothing still opens a
  completed run of the delegate (`RecordRequest.OpenEmptyRuns`), so AI Control
  lists every task an agent was handed. `AgentRun` serves `parentOwnerKind`,
  `parentOwnerId`, `delegateCallId` and `handedBy`, the agent of the
  conversation whose agent handed the task over (read through the request's
  `ThreadAgentByID` and `AgentDefinitionByID` loaders), and the run list reads
  "Handed by {agent}".
- **Artifacts.** What the delegate's tools produced, and the documents it
  published, are kept beside the conversation as usual; each is tied to the
  delegate's own saved message.
- **Transcript.** The delegate's steps are one section, "Handed to {agent}",
  quoted under the call. In the thread a settled hand-off is drawn open onto
  its task, steps and answer, with its chevron always shown, so after a reload
  the delegate's work is in view (#628).
- **Trajectory.** Every event, tagged, is kept with the turn's events.
- **The run's parent.** The delegate's run records where it came from:
  `turn_id` and `parent_owner_id` are the asking turn, `parent_owner_kind` is
  `AssistantTurn`, and `delegate_call_id` is the `delegate_task` call. Its
  `trace_id` is the delegate's own trace (below).

### Tracing

A delegate's task is a trace of its own, named by the turn's id and the call id
(`aitrace.ForDelegate`), not a subtree of the asking turn's. Its model calls, its
provider attempts and its tool calls run as activities of the turn's workflow
and are re-parented to the delegate's anchor, each linking the activity span it
ran under. `OpenDelegateActivity` opens a `trenova.ai.delegate.open` span in the
turn's trace, linked to the delegate's anchor, with the declined reason when it
was declined. When the turn is filed, `FinishTurnActivity` emits the delegate's
`invoke_agent` root from its `delegate_started` and `delegate_finished` events,
with its status, tool calls and the tokens and cost it spent
(`DelegatedRun.Usage`); the turn's root links to each delegate's root and each
delegate's root back to it. Its usage rows carry the call id and the delegate's
definition version, its steps the delegate's definition and version, and its
proposals the delegate's trace. See [ai-tracing.md](ai-tracing.md).

## Person-directed tasks

A person can hand a task to another agent themselves, from the conversation
they are in. The Desk does this when a case step names an agent other than
the conversation's (see "Who can take a step" in
[desk-cases.md](desk-cases.md)). The person stays in the conversation, and
the agent they chose takes the task inside it.

- **Request.** `POST /assistant/threads/:threadID/turns/` (and `…/messages/`,
  and `…/queue/`) take `directedAgentId`. The content is the task, at most
  `agentruntime.MaxDelegateTaskRunes`. It travels as
  `SendMessageRequest.DirectedAgentID` → `AssistantTurnRequest.DirectedAgentID`
  → `QueuedRequest.DirectedAgentID`. A directed message never steers a reply
  under way. If one is running it waits in the queue, and sending it now is
  refused.
- **Checks** (`assistantservice.directedTask`, run in `checkTurn` beside the
  others). The agent must not be the conversation's own. It must be enabled
  and talked to, the person must be allowed to use it (`usableAgent`, the
  same check the conversation's agent passes), and its budget must not be
  spent. The conversation's agent does **not** have to list it: the person
  chose it, not the agent. The conversation's agent must still pass its own
  checks, because the turn belongs to its conversation.
- **Turn.** `RunRequest.Directed` → `Turn.directed` → `TurnState.Directed`.
  `Drive` sees it once the turn has opened and calls `driveDirected` instead
  of the conversation's model:
  - it writes a `delegate_task` call with a fresh id (`fx.NewCallID`),
    holding `agentId` and `task`, into the assistant message;
  - it runs that call through `handTask`, the body `delegate` uses, with
    `DelegateCall.Directed` set;
  - it records the result as any delegate result is recorded;
  - it closes with the delegate's answer, or with the reason when it was
    declined, failed or stopped.

  Nothing calls the conversation's model.
- **Opening.** `OpenDelegate` uses `Definition.TaskRefusal` instead of
  `DelegateRefusal` when `Directed` is set. That is the same check with the
  allowlist left out. Every other check is made again as for any delegate:
  the agent is in the tenant, enabled, talked to and not the parent; the
  person may use it; its budget is not spent. Only runtime code builds a
  `DelegateCall`, from a task the person's own request named. A model's
  call never sets `Directed`.
- **Afterwards.** The saved turn has the same shape as one where the agent
  delegated: the person's message, the made-up call, the delegate's steps
  tagged `Delegated`, the account, and a closing reply. The thread and the
  stream show it the same way, its proposals are recorded as the delegate's,
  and taint crosses both ways. A later turn's model replays the call and the
  account, so the conversation's agent knows what was asked and what came of
  it.

## The stream

Events of the delegate's turn go to the turn's stream. Tool and message events
keep their names and gain `agentId` and `delegateCallId`. Streamed text and
restarts are renamed, because a reader applies a plain `delta` to the reply it
is showing and a plain `retrying` discards it. A refusal of the delegate's
answer is not sent on its own (it would read as a refusal of the primary's
reply); it arrives as `delegate_finished` with `status: "refused"`.

| Event | Data |
|---|---|
| `delegate_started` | `delegateCallId`, `agentId`, `agentName`, `icon`, `accent`, `task` |
| `message`, `tool_started`, `tool_finished` | as always, plus `agentId`, `delegateCallId` |
| `delegate_delta`, `delegate_reasoning` | `agentId`, `delegateCallId`, `text` |
| `delegate_retrying` | the `retrying` fields, plus `agentId`, `delegateCallId` |
| `delegate_finished` | the account above |

`artifact` events are unchanged: what a delegate shows belongs to the
conversation.

## Versioning

No `GetVersion` gate. `delegate_task` is a new tool, and whether a turn holds
it is decided when the turn opens, in an activity, and carried in
`TurnState.Held`. An execution that opened its turn before this release holds no
`delegate_task`, so a model in it that names the tool takes the unheld-tool path
it always took, which records no command; a state from before `Held` was kept
works the held set out from the definition, which never adds it. The delegate's
turn exists only in executions whose turn opened after the release. Every other
change is either data the loop only reads when delegating (`TurnState.Delegates`,
`RunContext.Delegation`, `ModelCallInput.Scope`), an activity input, or a step
key that is unchanged when no scope is set.

The structured account, the delegate's mark in it and `record` on write results
took no gate either. The account is built in workflow code, but only from data
the workflow already holds, and what changed is data: an optional field on the
tool result message and `ToolOutcome` (both activity inputs), optional fields on
the `delegate_finished` payload (published to the Workflow Stream, which records
no command) and in the text of the `delegate_task` result (an input of the next
model activity, whose inputs replay does not compare). No command is added,
removed or reordered, and no branch reads the new fields. `record` is set by a
tool inside its activity, and `RuntimeDelegate`'s resolved icon and accent are
worked out by the context activity.

Taint took no gate. `DelegateCall.Taint` is an activity input; the delegate's
taint comes back inside its `RunResult`, which the workflow already held, and is
merged into the asking turn's `RunResult.Taint` as data, with `run_tainted`
published to the Workflow Stream (no command). A turn opened before the release
has a nil taint and hands its delegate a nil, which counts as tainted for every
class that leaves the organization; see
[agent-runtime.md](agent-runtime.md#taint-is-data).

Tracing took no gate. The delegate's spans are opened in its activities and its
root by the activity that files the turn; `DelegatedRun.Usage` is summed from
model replies the workflow already holds, and the call id and definition version
on the usage attribution come from the delegate's own request. None of it adds a
command. See [agent-runtime.md](agent-runtime.md#waiting-on-in-flight-executions).

Records and shared results took no gate. `DelegateCall.Context` is optional data on
the activity input, built in workflow code only from the call's arguments and the
turn's own messages, which the workflow already holds; replay does not compare
activity inputs, and a call made before `records` and `shareResults` existed carries
neither, so its context is nil and its input is what it was. The delegate's
question is built inside `OpenDelegateActivity`. Recording the turn's and its
delegates' writes in one call, the ordered plan and the run of a read-only hand-off
all happen in `FinishTurnActivity`; no command is added, removed or reordered.

Person-directed tasks took no gate. Whether a turn is directed is decided in
`PrepareTurnActivity` and carried in `TurnState.Directed`. A turn opened before
the release has none and runs the loop it always ran. The made-up call id comes
from `fx.NewCallID`, a recorded side effect, so replay gets the same id.
`DelegateCall.Directed` is optional data on the `OpenDelegateActivity` input.
An old worker would ignore `Directed` and ask the conversation's model, so finish
rolling the chat-queue workers before the Desk sends `directedAgentId`.

A rolling deploy is the one exposure: a turn opened by a new worker and replayed
by an old one would dispatch `delegate_task` as a tool. Finish rolling the
chat-queue workers before an allowlist is configured.

## Access around a conversation

Who may use an agent (`AccessMode`, role grants, `agentaccessservice`) decides
more than who may start a conversation. These are the places it reaches, and
each is enforced on the server.

| What | Where |
|---|---|
| **Deciding a plan in your own conversation.** `decideMyPlan(id, input)` needs `assistant:update`, a plan whose run's subject is an assistant thread the caller owns (someone else's is *not found*), and access to every agent with a step in it — the one whose run the plan hangs on, and each agent a step was filed by, such as a delegate. It then goes through `agentplanservice.Decide`, so every step is decided by the decision service as the caller and each write is permission-checked against them when it runs. The in-thread plan card uses it; AI Control and the decisions queue keep `decideAgentPlan` (`agent_proposal:update`). | `agentplanservice.DecideOwn`, `resolver.DecideMyPlan`, `authzlint` (`TestAgentAccessResolversAreAuthorized`) |
| **Why a conversation cannot continue.** A thread is served with `canContinue` and, when it is false, `cannotContinueReason`: `AgentDeleted`, `NoAccess` (the reader may not use the agent, or the assistant at all), `AgentDisabled`, or `AgentNotConversational` (it now runs on a schedule, an event or continuously), in that order of precedence, so a reader is never told an agent they could not use anyway was merely turned off. Worked out when served, never stored. | `assistantservice.markContinuable`, `conversation.ContinueRefusal` |
| **Saving an agent with who may use it.** `POST`/`PUT /agent-definitions/` take `accessMode` and `accessRoleIds` together (both absent keeps access; one without the other is refused). The save and the access are one transaction (`AgentAccessService.SaveWithAccess`), so a restricted agent is created restricted and enabled in one request. Changing access needs `role:update` as well, as `setAgentAccess` does; access that already reads as asked needs nothing more, and the client sends it only when it changed. | `agentdefinitionservice.save`, `agentaccessservice.SaveWithAccess` |
| **The audience of an unsaved form.** `agentAccessPreview(input)` works each role's coverage and, while the form says Everyone, the sensitive tools out of the tools and access the form holds (for a saved agent, its other settings as saved), writing nothing. The form asks it debounced, keyed by the normalized request. | `agentaccessservice.PreviewAudience` |

## Limits

- One level. A task that needs several agents is handed to each by the primary.
- Three tasks per turn; each counts as one of the primary's tool calls.
- A delegate cannot ask the person anything; it answers with what it has.
- A delegate's steps count toward the conversation's message cap.
- Delegation is for conversations only: background runs, evaluations and
  replays never delegate.

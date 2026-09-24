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
  what matched, before it falls back to "an administrator can add it".

The same context is what `PreviewPrompt` shows in AI Control.

## `delegate_task`

| | |
|---|---|
| Name | `delegate_task` |
| Effect | `delegate` (`agent.ToolEffectDelegate`) |
| Arguments | `agentId` (one of the turn's delegates), `task` (what to do and what to hand back, at most 4 000 characters) |
| Cost to the primary | one tool call of its own budget |
| Limit | 3 per turn (`maxDelegationsPerTurn`); the fourth is refused and counted |

The loop (`agentruntime.Service.delegate`, in `Drive`) checks the arguments,
emits `delegate_started`, hands the task to `TurnEffects.Delegate`, folds what
came back into the turn, and emits `delegate_finished`. A turn holds the tool
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
                "summary": "On-time this month"}],
  "published": [{"id": "aart_…", "kind": "document", "title": "Report notes"}],
  "toolCallsUsed": 2
}
```

| `status` | Meaning | The call |
|---|---|---|
| `completed` | The delegate answered. | succeeded |
| `exhausted` | It spent its tool budget; `reply` is its best answer. | succeeded |
| `refused` | The output guard withheld its answer; `reason` is the refusal. | succeeded |
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
  `FinishTurn` records them through `persistDelegatedProposals` as a run of the
  delegate's definition whose subject is the same conversation. Decisions shows
  "Report Builder proposed…", trust accrues to the Report Builder, a shadow
  switch on it holds its cards, and the cards appear in the same conversation.
  The follow-up after a decision goes to the conversation, whose agent is the
  primary.
- **Artifacts.** What the delegate's tools produced, and the documents it
  published, are kept beside the conversation as usual; each is tied to the
  delegate's own saved message.
- **Transcript.** The delegate's steps are one section, "Handed to {agent}",
  quoted under the call.
- **Trajectory.** Every event, tagged, is kept with the turn's events.

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

A rolling deploy is the one exposure: a turn opened by a new worker and replayed
by an old one would dispatch `delegate_task` as a tool. Finish rolling the
chat-queue workers before an allowlist is configured.

## Limits

- One level. A task that needs several agents is handed to each by the primary.
- Three tasks per turn; each counts as one of the primary's tool calls.
- A delegate cannot ask the person anything; it answers with what it has.
- A delegate's steps count toward the conversation's message cap.
- Delegation is for conversations only: background runs, evaluations and
  replays never delegate.

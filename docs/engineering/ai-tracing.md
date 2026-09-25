# AI tracing

How an agent run, an assistant turn, a delegate's task and a one-shot model call
are traced: which spans exist, how a run that lasts a day and is decided a week
later still reads as one trace, and which ids are written to the database so the
AI audit trail and the screens can find the trace again.

Read [agent-runtime.md](agent-runtime.md) first. Everything here rides on its
rule that workflow code does nothing but decide commands: **no span is started
in workflow code**, and nothing below added, removed or reordered a command.

## The shape of a trace

| Unit | Trace | Root span | Emitted by |
|---|---|---|---|
| Agent run | anchored on the run id | `invoke_agent {agent}` | `agentjobs.FinishRunActivity`, from the run's created or started time |
| Assistant turn | anchored on the turn id | `invoke_agent {agent}` | `assistantjobs.FinishTurnActivity`; `CloseTurnActivity` when the save failed on every attempt |
| A delegate's task | anchored on the turn id and the `delegate_task` call id | `invoke_agent {delegate}` | `FinishTurnActivity`, from the delegate's `delegate_started` and `delegate_finished` events |
| Evaluation replay | anchored on the evaluation id | `invoke_agent`, `trenova.ai.purpose=evaluation` | `FinishReplayActivity`, `FailEvaluationActivity` |
| One-shot call | a new trace per call | `trenova.ai.job {feature}` | `completionjobs.Dispatcher.await`, synchronously |
| Other AI work (document extraction, inbound classification, accounting mapping, insights, the daily briefing) | the workflow's own trace | none of its own | router spans carry `trenova.ai.feature` |
| Deciding, running or expiring a proposal | the request's or the sweep's | `trenova.ai.proposal.decide` / `.execute` / `.expire` | `agentdecisionservice`, `proposalexecutor`, the expiry activities |

### Anchors

A run lasts from seconds to a day, runs on many workers, and is retried. Its
root cannot be a span somebody holds open. Instead the trace is **named** by the
unit of work: `aitrace.AnchorFor(kind, key)` hashes
`trenova/ai-trace/v1|{kind}|{key}` with SHA-256 and takes the trace id from the
first 16 bytes and the root span id from the hash of the same text plus `|root`.
Anything that knows the run id knows its trace.

- **Start.** The API starts the workflow with the anchor as the remote parent
  (`aitrace.ContextWithAnchor`): `agentrunservice.StartForDefinition` (events,
  schedules through `StartScheduledRunActivity`, a person pressing Run),
  `assistantjobs.StartTurnWorkflow` (the chat handler and decision follow-ups)
  and `agentevaluationservice`. Temporal's OpenTelemetry interceptor carries it
  through the workflow into every activity. The caller's own `traceparent` is
  kept on the payload as `traceOrigin` (`AgentRunPayload.Origin`,
  `AssistantTurnPayload.Origin`); the root links back to it.
- **Children.** Every AI span an activity opens takes the anchor it belongs to
  (`aitrace.ForRun(owner, delegation)`, `aitrace.ForAttribution(attribution)`)
  and `aitrace.Parent` keeps the activity's span as parent when it is already in
  that trace, and otherwise re-parents to the anchor and **links** the activity's
  span. That is how a delegate's calls, which run as activities of the turn's
  workflow, land in the delegate's own trace.
- **Root.** The finish activity emits the root with `aitrace.EmitRoot`: the
  custom ID generator forces the anchored ids for that one span, and
  `WithTimestamp` gives it the real start. Every span emitted earlier under the
  anchor hangs beneath it.

`aitrace.IDGenerator` and the anchor-aware sampler are installed on the tracer
provider (`observability/tracer.go`).

## Span catalogue

Nothing a model was sent, a tool was given or either returned is ever put on a
span: no prompt, no arguments, no result.

### `invoke_agent {agent}` (INTERNAL, root)

`gen_ai.operation.name=invoke_agent`, `gen_ai.agent.id`, `gen_ai.agent.name`,
`gen_ai.conversation.id` (the thread), `user.id`, `error.type`;
`trenova.ai.agent.version`, `trenova.ai.owner.kind`/`.id`, `trenova.ai.run.id`,
`trenova.ai.turn.id`, `trenova.ai.delegate.call_id` (a delegate's),
`trenova.ai.trigger` (a run's trigger, a turn's origin), `trenova.ai.status`,
`trenova.tenant.organization_id`/`.business_unit_id`,
`trenova.ai.tokens.input`/`.output`, `trenova.ai.cost_usd`,
`trenova.ai.tool_calls`, `trenova.ai.tainted`, `trenova.ai.taint.sources`,
`trenova.ai.simulation`, `trenova.ai.purpose` (`live` or `evaluation`),
`trenova.ai.anchor=true`.

Links: to the `traceOrigin` that started it; a turn's root to each delegate's
root and each delegate's root back to the turn's.

`error.type` is how the loop ended when it did not finish: `cancelled`,
`timeout`, `no_provider`, `providers_resting`, `refused`, `schema_invalid`,
`http_{status}` or `failed` (`agentflow.FailureKind`).

Token and cost totals come from the loop, not from a query: the model activity
returns what every attempt of the call used (`ModelReply.Usage`) and `Drive`
adds it to `RunResult.Usage`; a delegate's own sum travels on
`DelegatedRun.Usage`. A reply from a history recorded before `Usage` existed is
counted from the answering call's tokens and cost.

### `trenova.ai.completion` (INTERNAL)

One per model call (`agentruntime.Service.StreamCompletion`, which the model
activity runs). `trenova.ai.activity.attempt` (Temporal's attempt, handed over
with `aitrace.WithCallOrigin`), `trenova.ai.stream`, `trenova.ai.looped`,
`trenova.ai.feature`, and from the call's tally: `trenova.ai.attempts`,
`trenova.ai.failover`, `trenova.ai.mid_reply_restarts`, the final
`trenova.ai.provider.id`, `gen_ai.provider.name`, `gen_ai.response.model` and the
summed `trenova.ai.cost_usd`.

### `chat {model}` / `embeddings {model}` (CLIENT)

One per provider attempt in `completionrouter`, chat, structured and embedding
alike. `gen_ai.operation.name`, `gen_ai.provider.name` (`anthropic`, `openai`,
`ollama`), `gen_ai.request.model`, `gen_ai.response.model`,
`gen_ai.request.max_tokens`, `gen_ai.usage.input_tokens`,
`gen_ai.usage.output_tokens`, `gen_ai.usage.cache_read.input_tokens`,
`gen_ai.usage.cache_creation.input_tokens`, `gen_ai.response.finish_reasons`
(`stop`, `length`, `tool_calls`, `content_filter`), `server.address`,
`error.type` (the usage row's error class); `trenova.ai.provider.id`,
`trenova.ai.attempt`, `trenova.ai.failover`, `trenova.ai.cost_usd`,
`trenova.ai.reasoning_tokens`, `trenova.ai.truncated`. Events:
`provider.busy_wait{wait_s}` for each wait on a provider answering 429 or 5xx,
`provider.resting` when the attempt put the provider to rest.

Input tokens on the span are the whole prompt, as the semantic conventions ask:
Anthropic reports cached tokens beside `input_tokens`, so they are added
(`modeladapter.CacheSeparateFromInput`); OpenAI counts them inside it. The usage
row keeps the provider's own `input_tokens` so cost is unchanged, with the cache
counts in their own columns.

### `execute_tool {tool}` (INTERNAL)

One per tool call, in `agentruntime.Service.DispatchStep`, around the ledger
claim, the decision and the call. `gen_ai.operation.name=execute_tool`,
`gen_ai.tool.name`, `gen_ai.tool.call.id`, `gen_ai.tool.type=function`,
`gen_ai.agent.id`; `trenova.ai.agent.version`, `trenova.ai.tool.effect`/`.kind`,
`trenova.ai.step.key`, `trenova.ai.step.state` (`fresh`, `replayed`, `unknown`;
absent for an unguarded call), `trenova.ai.tier`, `trenova.ai.tier.source`
(`PolicyDefault`, `PersonSetting`, `TrustEarned`, `PersonalExemption`),
`trenova.ai.held_by`, `trenova.ai.egress_class`, `trenova.ai.tainted`,
`trenova.ai.after_external_content`, `trenova.ai.outcome` (`ran`, `proposed`,
`simulated`, `denied`, `invalid`, `duplicate`, `over_budget`, `failed`,
`unknown`), `trenova.ai.proposal.id`, `trenova.ai.delegate.call_id`,
`error.type` (the outcome, for one that did not run).

### `trenova.ai.write {entity}` (INTERNAL)

Around the tool's write, automatic or simulated, and around an approved write.
`trenova.ai.entity.type`/`.id`, `trenova.ai.version.before`/`.after` (the
record's version read before and after, for a tool that names its target),
`trenova.ai.proposal.id`, `trenova.ai.simulated`.

### `trenova.ai.delegate.open` (INTERNAL)

`OpenDelegateActivity`, in the asking turn's trace, linked to the delegate's
anchor. `trenova.ai.delegate.call_id`, `.agent.id`, `.agent.name`, and
`trenova.ai.delegate.declined` with the reason when it was declined.

### `trenova.ai.proposal.decide` / `.execute` / `.expire` (INTERNAL)

`decide` in the request that decided (`agentdecisionservice.DecideWithOutcome`),
`execute` inside it or wherever the executor runs, `expire` in the expiry
activities. The tenant, `trenova.ai.proposal.id`, `trenova.ai.run.id`,
`gen_ai.tool.name`, `trenova.ai.decision`, `user.id`, `trenova.ai.reason_code`,
`trenova.ai.modification_count`, and for expiry `trenova.ai.proposal.expired`,
the count closed. `decide` and `execute` link to the `execute_tool` span the
proposal was decided in; `execute` holds an `execute_tool` and a
`trenova.ai.write` span for the approved write. A run's expiry links to the
run's root; the organisation-wide sweep links to nothing.

### `trenova.ai.job {feature}` (INTERNAL, root)

A one-shot call a request waits on starts a new trace with
`trace.WithNewRoot()` and a link to the request, so a slow compose or provider
test is its own trace rather than a thread inside the request.
`trenova.ai.job.feature`, `trenova.ai.job.workflow_id`, the organization, and
`error.type` (`abandoned` when the caller stopped waiting).

### Metrics

`gen_ai.client.token.usage` (by `gen_ai.token.type`) and
`gen_ai.client.operation.duration` histograms, per provider attempt, with the
semantic-convention buckets, through the metrics registry's OpenTelemetry bridge
(`metrics.Registry.GenAI()`), so they appear on `/metrics` beside everything else.

## Link columns

What is written, by whom and when. Every id is written whether or not the trace
was sampled, so the audit trail can always name it.

| Column | Written by | When |
|---|---|---|
| `agent_runs.trace_id` | `agentrunservice.StartForDefinition` (the run's anchor); `StartInline` (the trace it ran in); the recorder for a chat run (the turn's anchor) and a delegate's run (the delegate's anchor) | when the row is created |
| `agent_runs.turn_id` | `proposalrecorder` from `persistProposals` | when a turn's or a delegate's proposals open their run |
| `agent_runs.parent_owner_kind`, `.parent_owner_id`, `.delegate_call_id` | the same, for a delegate's run: `AssistantTurn`, the turn, the `delegate_task` call | same |
| `assistant_turns.trace_id` | `assistantturnservice.Start`, which mints the turn's id before the insert | when the turn is recorded |
| `agent_run_steps.trace_id`, `.span_id` | `runstepledger.Claim`, from the `execute_tool` span | when the call is claimed |
| `agent_run_steps.agent_definition_id`, `.agent_definition_version`, `.delegate_call_id` | the same, from the turn's definition (a delegate's own) and delegation | same |
| `agent_run_steps.outcome.reason`, `.verdict` | `runstepledger.Settle` | when the call settles |
| `agent_proposals.id` | minted in the dispatch activity (`PendingAction.ProposalID`); the insert mints one for an action from an older history | when the proposal is filed |
| `agent_proposals.trace_id`, `.span_id`, `.step_key` | `proposalrecorder`, from the action | same |
| `agent_proposals.executed_at` | the recorder, from `PendingAction.ExecutedAt` (when an automatic write ran); the executor for an approved one | at filing; at execution |
| `agent_proposals.executed_by_user_id` | the recorder (the person in the conversation, for an automatic or simulated write they ran as; none for an unattended run); the executor (the approver, on success, failure and simulation) | same |
| `agent_proposals.executed_target_version` | the recorder, from `PendingAction.ExecutedVersion`; the executor, read after the write | same |
| `agent_decisions.trace_id` | `agentdecisionservice`, the `decide` span's trace | when the decision is recorded |
| `ai_usage_records.trace_id`, `.span_id` | `completionrouter.record`, the attempt's `chat` span | per attempt, off the request path |
| `ai_usage_records.owner_kind`, `.owner_id`, `.delegate_call_id`, `.agent_definition_version` | the same, from `AIUsageAttribution`, set in `Turn.completionRequest()` | same |
| `ai_usage_records.attempt`, `.failover`, `.cache_read_tokens`, `.cache_write_tokens` | the same | same |
| `agent_run_events.occurred_at` | `agentruneventservice`, from `StreamItem.At` (the workflow's clock) or now | when the account is written |

`traceId` and `traceUrl` are served on `AgentRun`, `AgentProposal` and
`AgentDecision`.

## Sampling and configuration

| Key | Default | |
|---|---|---|
| `monitoring.tracing.enabled` | `false` | nothing is exported without it; ids are still derived and stored |
| `monitoring.tracing.samplingRate` | `1.0` | ordinary traces, by trace id |
| `monitoring.tracing.aiSamplingRate` | `1.0` | runs, turns, delegates and evaluations, decided from the anchored trace id so every worker agrees |
| `monitoring.tracing.traceUrlTemplate` | empty | `{traceId}` is replaced; empty serves no `traceUrl` and the screens offer the id to copy |

The sampler is parent-based: an anchored root samples at `aiSamplingRate`, and
every child follows its parent.

Temporal's interceptor has Signal, Update and Query tracing turned off, since a
streamed turn publishes a signal per batch and every reader poll is an update.

## Determinism

No workflow-code span, no new command, no `GetVersion` gate. What changed in or
around workflow code is data:

- `StreamItem.At` is stamped in `workflowEffects.Emit` and on a turn's opening
  with `workflow.Now`, which records nothing and reads back the same instant on
  replay.
- `AIUsageAttribution.{OwnerKind, OwnerID, DelegateCallID, DefinitionVersion}`
  are set in `Turn.completionRequest()` from the turn's own request: an activity
  input.
- `ModelReply.Usage` is an activity result; `RunResult.Usage` and
  `DelegatedRun.Usage` are summed from results the workflow already holds.
- `PendingAction.{ProposalID, TraceID, SpanID, TierSource, ExecutedVersion,
  StepKey, ExecutedAt}` and `RunStepOutcome.{Reason, Verdict}` are set inside the
  tool activity.
- `traceOrigin` is a payload field.

A history recorded before any of it replays with none of it: no `At` (the
writer stamps now), no `Usage` (counted from the reply), no proposal id (the
insert mints one), no origin (no link). The histories under
`agentjobs/testdata/replay-loop` and `assistantjobs/testdata/replay` were
recorded from the code before this change and replay against it.

## Known limits

- **An evicted workflow's span may never export.** The interceptor's
  `RunWorkflow` span exports when the workflow ends on the worker that holds it;
  a run parked a day in a decision wait is usually evicted first. The trace still
  coheres, because every activity span shares the anchored trace id and hangs
  from the root.
- **A root can be emitted twice.** The finish activity emits the root last; an
  attempt lost after emitting and before reporting is retried and emits it
  again, with the same ids. Backends keep one span per id or show both.
- **A run's root ends when it is filed, not when its last proposal is
  decided.** Decisions are traces of their own, linked back.
- **Ids of unsampled traces are stored.** `traceUrl` leads nowhere for a trace
  the sampler dropped or a backend no longer holds.
- **Only a run's expiry links to the run.** The organisation-wide sweep reports
  a count and no links.
- **The tier source costs a trust read.** Telling a tier a person chose from
  one trust earned reads the trust ledger for every call on a tool the agent
  names a tier for.
- **Unattended writes name no person.** `executed_by_user_id` is empty for an
  automatic write of a background run.
- **An evaluation run by a quality suite** starts in the suite's trace; its
  spans re-parent to the evaluation's anchor and link to the suite's activity.

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
| Import assistant | `ImportAssistantTurnWorkflow`, one per document at a time | `agent-chat-queue` |
| Table compose, formula generate and explain | `StructuredCompletionWorkflow` | `agent-chat-queue` |
| Briefing regenerate | `WriteBriefingWorkflow` | `agent-chat-queue` |
| AI provider test | `TestAIProviderWorkflow` | `agent-chat-queue` |
| Event-driven and scheduled agent runs | `AgentRunWorkflow` | `agent-background-queue` |
| An agent's schedule firing | `AgentScheduledRunWorkflow` | `agent-background-queue` |
| Agent evaluations (replays) | `AgentEvaluationWorkflow` | `agent-heavy-queue` |
| Reports and dispatch planning called as tools | the tool's activity | `agent-heavy-queue` |
| Insights, daily briefing | a parent plus one child per organization | `system-queue` |
| Document extraction | `ProcessDocumentAIExtractionWorkflow` | document intelligence |
| Inbound email | `ProcessInboundMessageWorkflow` | `system-queue` |

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
- Anything that reads a clock or the database happens in an activity. The
  workflow holds only the turn and what its activities returned.

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
not dedupe: `RequiresIdempotencyKey` is checked for presence and, bar the two
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
JSON document handed back to a model is worse than none. The import assistant's
tools that create a shipment or a location are never retried at all.

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
disconnected context and the execution is recorded as cancelled. A turn recorded
but never handed to a worker has no execution to cancel; Stop closes its record
as `Stopped`.

### The turn stream

A run's events go to a **Workflow Stream** its own workflow hosts
(`agentflow.HostStream`, on `go.temporal.io/sdk/contrib/workflowstreams`). The
model activity publishes the reply as it streams, batched every 100 ms; the
workflow publishes every other event. The stream exists as soon as the workflow
does, so a reader can never attach ahead of it.

Each event's offset is its SSE event id, returned as `Last-Event-ID` to resume.
The cursor is the last event the reader **applied**, not the last it received.

A stream is read by polling the workflow, so a closed workflow cannot be read.
When the relay forwards the last event it signals `stream-drained`; the workflow
waits for that, at most 15 s, before it closes. A reader who arrives after the
workflow closed gets an ending rebuilt from the record, which tells the client
to read the conversation.

Each publish is a signal in the turn's history and each read a poll update, so a
streamed reply adds a few hundred history events. That is why chat is one
workflow per turn rather than one per thread, and why a run nobody watches
publishes nothing.

### Decision follow-ups

A decision on a proposal or plan a conversation raised is answered in that
conversation, whoever decided it and wherever. `agentdecisionservice` and
`agentplanservice` call `DecisionFollowUps` once the change has run or failed;
`assistantfollowupservice` opens a turn with origin `DecisionFollowUp`, as the
thread's owner. A conversation already producing a reply is not interrupted.

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

### Starting runs

- **Events.** A run is keyed by its subject:
  `agent-run-<definition>-subject-<subject>`. The workflow starts before the run
  is recorded, with conflict policy FAIL and
  `WorkflowExecutionErrorWhenAlreadyStarted`, so a second event about a subject
  whose run is still open is refused by Temporal, where two requests reading a
  count of open runs could both have passed. The refusal is
  `ErrAgentRunAlreadyOpen`, which the publisher treats as a skip.
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

- `StructuredCompletionWorkflow` asks one structured question (table compose,
  formula generate and explain), retried the `modelcall` way.
- `TestAIProviderWorkflow` probes once and never retries: the administrator is
  asking whether the connection works now.
- `WriteBriefingWorkflow` rewrites a day's page.

Two people testing the same provider, or rewriting the same page, share one
execution. The caller gets the call's own error back.

## The import assistant

`ImportAssistantTurnWorkflow` answers one message about one document, the loop in
workflow code: a prepare activity, one activity per model call, one per tool
call, and a finish that saves the turn. The reply streams through the turn's
Workflow Stream with the events the client has always read; the request relays it
frame for frame with the reader chat uses. A document answers one message at a
time.

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

## Measuring it

`turn_first_event_seconds` is separate from `turn_duration_seconds` on purpose:
duration says how long the answer took, first-event says how long a reader who
attached as the turn began stared at nothing.

`trajectory_events_total{result}` counts dropped events. They are invisible by
design — the run carries on — so this is the only place they show up.

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
| `document-ai-extraction-timer-poll` | `extractWithTaskToken` | `SubmitAndAwaitDocumentAIExtractionActivity`, `PollPendingDocumentAIExtractionsWorkflow` and its schedule, task tokens on `document_ai_extractions` |

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

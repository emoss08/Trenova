# The agent runtime

How an agent's work actually executes: where it runs, what makes it safe to
retry, what survives a crash, and what is written down afterwards.

Read this before changing anything under `internal/core/services/agentruntime/`,
`internal/core/services/assistantservice/`, or `internal/core/temporaljobs/agentjobs/`
and `assistantjobs/`.

## Where work runs

Agent work is Temporal work. The API server authenticates the request, persists
a record, and starts or signals a workflow; workers execute the steps.

Three queues carry agent work, and the split exists so one class of work cannot
starve another:

| Queue | Carries | Who is waiting |
|---|---|---|
| `agent-chat-queue` | interactive assistant turns | a person, right now |
| `agent-background-queue` | event-driven and scheduled runs | nobody |
| `agent-heavy-queue` | run replays | nobody, and they are slow |

A worker polls all three unless told otherwise:

```
trenova worker run                                   # every queue
trenova worker run --queues=agent-chat-queue         # only interactive turns
```

Run interactive turns on their own workers if you want the split to mean
anything. A single worker pool polling everything can still fill with replays
while somebody waits for an answer.

`agent-queue` is the queue these were split out of. A drain registry still polls
it so runs started before the split are not stranded; it has no new work and is
deleted once no in-flight run can still be on it.

Desks and the nightly sweeps run on `system-queue` and sit outside this
entirely. A desk that raises work publishes a domain event, which starts a run
on `agent-background-queue` — the correct destination, since nobody is waiting.

## What makes a retry safe

A Temporal activity can run more than once. The agent loop is one activity, so
without help a retry would re-run every tool call the first attempt made —
and the tools do not dedupe. `RequiresIdempotencyKey` is checked for presence
and, bar the two that forward it to an email provider, never looked up.

`agent_run_steps` is what makes it safe. Every operation is **claimed before it
runs** and settled after:

- The key is derived from what the model asked for — owner, tool name,
  arguments, and an ordinal — not from the provider's call id, which changes on
  every attempt and would make each retry look like new work.
- A claim that finds the key already settled returns the recorded outcome
  instead of running again.
- A claim that finds it still `Started` means the previous attempt died between
  executing and recording. The model is told the operation began and its outcome
  is unknown, rather than being silently replayed.

That last case is the honest limit: resume is **at-most-once, not lossless**. A
crash in the execute→settle window surfaces as "began, outcome unknown". The
alternative — assuming it failed and retrying — would double a write.

Outcomes over 64 KiB are dropped rather than truncated, because half a fenced
JSON document handed back to a model is worse than none.

## Chat turns

Every assistant question is answered by `AssistantTurnWorkflow` on
`agent-chat-queue`, ID `assistant-turn:<turnID>`. There is no in-request path
and no flag: the API records the turn, starts the workflow and returns the turn
to watch. `POST /threads/:id/messages/` starts the same workflow and waits for
its result; `POST /ask/` opens the hidden thread and starts a turn on it.

The workflow runs in three steps:

1. **Prepare** (`PrepareTurnActivity`) reads the thread, history, files and
   mentions, checks budget and room, and runs the scope guard. A refusal the
   person can act on — an empty message, a full thread — is non-retryable and
   its message reaches the reader as written.
2. **The loop** runs in workflow code (`agentflow`). Each model call is an
   activity that streams its reply and heartbeats on a timer. Each tool call is
   an activity named for the tool (the worker's dynamic activity), claimed in
   the step ledger under the key the workflow computed. A tool that fails after
   its retries is reported to the model as a failed call; the turn goes on.
3. **Finish** (`FinishTurnActivity`) saves the turn, its proposals and
   artifacts, closes the turn's record and writes the trajectory. It claims one
   ledger key per attempt, after checking no earlier attempt saved or began to
   save, so a retried save never appends a turn twice.

Model failures are mapped the way the Temporal AI cookbook maps HTTP responses:
a rejected request (4xx other than 408/409/429), a refusal or a missing
provider is not retried; a rate limit waits out the provider's `Retry-After`,
capped at a minute. The failure's kind travels in the error's details, so the
saved turn still says whether the provider refused or was unavailable.

`assistant_turns` gives a reply an identity while it is still being written.
A partial unique index enforces one live turn per thread. Its `origin` and
`input` say what the turn answers, so a reader who rejoins it shows the right
heading: the person's question, or the decision it reports.

### Stopping

`POST /assistant/turns/:id/stop/` cancels the workflow. The model call in
flight is cancelled; what had happened by then is saved on a disconnected
context, and the execution is recorded as cancelled. The client aborts its
reader at once so the person sees it stop without waiting for the round trip.

A turn recorded but never handed to a worker, because its start failed, has
no execution to cancel; Stop closes its record as `Stopped`, which frees the
thread's one live slot.

### The turn stream

A turn's events go to a **Workflow Stream** hosted by the turn's own workflow
(`go.temporal.io/sdk/contrib/workflowstreams`). The model activity publishes the
reply as it streams, batched every 100 ms; the workflow publishes every other
event. The stream exists as soon as the workflow does, before the start request
returns, so a reader can never attach ahead of it.

Each event's offset in the stream is its SSE event id, and a reader returns it
as `Last-Event-ID` to resume. The cursor is the last event the reader
**applied**, not the last it received: a connection that dies mid-frame delivers
something the client never folded in.

A stream is read by polling the workflow, so a closed workflow cannot be read.
When the relay forwards the last event it signals `stream-drained`; the
workflow waits for that, at most 15 s, before it closes. A reader who arrives
after the workflow closed gets an ending rebuilt from the turn's record
(`done` with `replay: true`), which tells the client to read the conversation.

Before sending, the client asks for the thread's active turn and follows it to
its end first, so a question asked while a decision's follow-up is still
answering waits for it on screen instead of failing with "already working on a
reply".

Each publish is a signal in the turn's history and each read a poll update, so
a streamed reply adds a few hundred history events. That is why chat is one
workflow per turn rather than one per thread.

### Decision follow-ups

A decision on a proposal or plan that a conversation raised is answered in that
conversation, whoever decided it and wherever: the card in the thread, the
Desk's decisions, AI Control, or a plan's approval.

`agentdecisionservice` and `agentplanservice` call `DecisionFollowUps` after the
decision is recorded **and the change has run or failed**, so the report is of
the outcome, not the click. A plan's steps are decided with `WithinPlan` and
start nothing of their own; the plan reports once, after its last step.
`assistantfollowupservice` finds the conversation through the run
(`subject_type = AssistantThread`) and opens a turn with origin
`DecisionFollowUp`, **as the thread's owner** — the decider may be someone else,
but the report is addressed to the owner with the owner's access. The turn's
input is a `DecisionNote` the assistant service writes from the decision.

The turn is a turn workflow like any other, recorded and started before the
decision returns, so the client rejoins it through `GET /assistant/threads/:id/turns/active/` — which it also does when a
thread opens, and whenever one of the thread's proposals or plans stops waiting.
A conversation already producing a reply is not interrupted: the follow-up is
skipped and the outcome reaches the agent on that turn, since every turn is told
what became of the conversation's proposals.

## What is written down

The runtime narrates its own work — every tool it reaches for, every refusal,
every give-up — and `agent_run_events` is what keeps that narration.

Before it existed, a conversation's events went to a stream that was gone
within the hour, and a background run's went to a function that used them
as a heartbeat tick and dropped them. A background run's durable record was its
final reply, cut to two thousand characters: what it concluded, never what it
did.

The table is **append-only**. Its repository has no update path at all, and that
absence is what makes it a log rather than a table that happens to be
insert-heavy.

How it relates to the ledger beside it:

| | `agent_run_steps` | `agent_run_events` |
|---|---|---|
| Answers | has this already run? | what happened? |
| Rows | one per operation, mutated Started→Completed | one per occurrence, never revised |
| Ordering | none needed | `sequence`, per owner |

They join on `step_key` wherever a tool is involved.

Ordering is by `sequence`, not time. Several events share a second easily, and a
trajectory read back in the wrong order is worse than none — it reports the agent
doing things in an order it never did. The sequence resumes from what is stored
rather than restarting at one, because a retried run opens a fresh writer that
would otherwise collide with its own earlier attempt.

Two rules govern the writer, and both matter more than completeness:

- **Recording must never be why a run fails.** A write that cannot happen is
  logged and dropped. An agent that finishes its work and then dies filing the
  paperwork is worse than one that files none.
- **Recording must not slow the run down.** Events are buffered and written in
  batches, because a run emits them far faster than a round trip to postgres.

Read a trajectory through the `agentRunEvents` GraphQL connection, gated on
reading an agent run: these events are what a run did, so anyone who may see the
run may see how it got there.

## Measuring it

`turn_first_event_seconds` is separate from `turn_duration_seconds` on purpose:
duration says how long the answer took, first-event says how long a reader who
attached as the turn began stared at nothing, which a slow worker or a
backed-up queue makes worse first.

`trajectory_events_total{result}` counts dropped events. They are invisible by
design — the run carries on — so this is the only place they show up at all.

## Known limits

- **Resume is at-most-once.** Described above. A crash in the execute→settle
  window reports "began, outcome unknown" rather than replaying.
- **Scheduled runs authorize as `PrincipalTypeAgent`** against the static
  `permission.IsAgentAllowed` table. Per-call actor context is satisfied and
  chat asserts a real actor, but "not implicitly a system administrator" is not
  yet true for unattended runs.
- **Background runs discard their transcript.** They produce the same rich
  message values chat persists — reasoning, per-call tokens, latency, cost — and
  keep only the summary. The event log now records what happened; the model's
  own words on a background run are still thrown away.
- **Permission denials are not distinct events.** A refusal arrives as a failed
  tool result whose content is prose, so it is recorded as one.
- **Temporal is on the chat path.** An outage means no question is answered;
  the API says so rather than answering some other way. The client connects
  lazily, so the API still starts, and the first call after Temporal returns
  succeeds.

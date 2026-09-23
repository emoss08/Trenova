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

## Durable chat turns

A reply can run in the request that asked for it, or on a worker.

`ai.durableTurns` decides, and it defaults to **off**. Off is the original
behaviour and is what an installation without a reachable Temporal gets. On
hands the turn to `agent-chat-queue`, so the reply survives an API restart and
somebody who closes the tab can come back to it.

The client does not assume. It asks `GET /assistant/capabilities/` once per
session and falls back to the in-request path when the answer is no — turning
the flag on without a worker polling `agent-chat-queue` means nothing is
answered durably, but nothing breaks either.

`assistant_turns` gives a reply an identity while it is still being written.
A partial unique index enforces one live turn per thread. Its `origin` and
`input` say what the turn answers, so a reader who rejoins it shows the right
heading: the person's question, or the decision it reports.

### Stopping

Stop is an operation now, not an abandonment. It used to work by aborting the
reader, because the model ran on the context that abort cancelled. With the work
on a worker, abandoning the reader stops nothing and the turn keeps billing, so
`POST /assistant/turns/:id/stop/` cancels the execution. The local abort stays,
so the person sees it stop immediately rather than waiting for a round trip to
confirm what they already decided.

The activity heartbeats on every event, because Temporal only delivers
cancellation through a heartbeat.

A turn running in an API process — the in-request path, or a decision
follow-up with durable turns off — has no execution to cancel, and may have no
request carrying it: a follow-up never had one, and a reader who reloads
rejoins through the relay rather than the request. So Stop closes the turn's
record as `Stopped`, and the turn runs on a context from
`assistantturnservice.Stoppable`, which is cancelled directly when the Stop
lands on the same instance and otherwise on its next read of the record (every
two seconds). The in-request path closes its record on a context the reader's
abort cannot reach, so a closed tab no longer leaves the thread's one live
slot held.

### The turn stream

A turn's events go to a Redis stream, one per turn, keyed `<prefix>:<org>:<turn>`.
It is a **tail buffer, never the transcript** — the transcript is in postgres and
outlives all of this. The stream is trimmed by length and dropped a quarter of an
hour after the turn ends.

A stream rather than pub/sub, because a reader needs to resume. Each event
carries the Redis entry id as its SSE event id, and a reader returns it as
`Last-Event-ID` to pick up where it stopped. The cursor is the last event the
reader **applied**, not the last it received: a connection that dies mid-frame
delivers something the client never folded in, and resuming past it would skip
it silently.

No stream is not an expired stream. A turn's stream is made by its first event,
and a reader can attach before that — a follow-up is recorded before the
decision that caused it returns. The relay follows a running turn whose stream
has not begun and waits for it; only a turn running for longer than two minutes
with no stream is reported as having lost its live view. An ending rebuilt from
the record carries `replay: true` and no result, and the client refetches the
conversation for what was said.

Before sending, the client asks for the thread's active turn and follows it to
its end first, so a question asked while a follow-up is still answering waits
for it on screen instead of failing with "already working on a reply".

Workflow history is deliberately not used for this. Sixty tokens a second is not
what a workflow history is for, and the issue this came from says so directly.

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

The turn follows `ai.durableTurns` like any other: on a worker when it is on,
off the request in this process when it is off. Either way it is recorded and
published to its stream before the decision returns, so the client rejoins it
through `GET /assistant/threads/:id/turns/active/` — which it also does when a
thread opens, and whenever one of the thread's proposals or plans stops waiting.
A conversation already producing a reply is not interrupted: the follow-up is
skipped and the outcome reaches the agent on that turn, since every turn is told
what became of the conversation's proposals.

## What is written down

The runtime narrates its own work — every tool it reaches for, every refusal,
every give-up — and `agent_run_events` is what keeps that narration.

Before it existed, a conversation's events went to the stream above and were
gone within the hour, and a background run's went to a function that used them
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

Every turn metric carries `transport="inprocess"|"durable"`, and the in-process
path is instrumented too. A comparison cannot be made from figures that do not
say which runtime produced them.

`turn_first_event_seconds` is separate from `turn_duration_seconds` on purpose:
duration says how long the answer took, first-event says how long the person
stared at nothing, and that is the one thing a durable hop plus event coalescing
could plausibly have made worse.

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
- **Stopping an in-process turn on another instance takes up to two
  seconds.** The record is closed at once, so the conversation is free, but the
  model runs until that instance next reads it.
- **A turn that dies with its process is noticed lazily.** An in-process turn
  writes `heartbeat_at` every fifteen seconds from its `Stoppable` watch. One
  that has gone a minute without a beat, and has no workflow behind it, is
  closed as Failed ("The server stopped before this reply finished.") the next
  time anybody looks: `Active` when a reader rejoins, `Start` when a new
  question finds it in the way (which then retries once), and the relay's idle
  check for a reader already following it. Nothing sweeps on a schedule, so a
  dead turn nobody looks at again stays Running in the table; it holds no
  conversation anybody is using. A durable turn is never closed this way: its
  worker, and Temporal, own its ending.
- **Redis is on the chat path.** An outage silences in-flight replies. The
  transcript still saves and the relay degrades to reading the turn record, but
  that is a fallback, not equivalence.

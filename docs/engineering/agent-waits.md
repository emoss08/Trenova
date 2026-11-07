# Durable waits

How an agent parks its work until something happens in the world and picks it
up again when it does: a truck reaching or leaving a stop, a reply about a
shipment or from a carrier or customer, an appointment coming round, detention
free time running out, a driver's drive time falling low, a record being changed
by a person or the system, a report run finishing, or a time.

Read this before changing `domain/agentwait`, `agentwaitservice`,
`temporaljobs/agentwaitjobs`, the `agent_create_wait` / `agent_cancel_wait` tools,
`realtimebroker.RecordWatchFeed`, `report_run`'s `then`, or any of the places that tell
waits what happened (below).

## The shape of it

```
agent_create_wait (tool) ──▶ agentwaitservice.Register ──▶ agent_waits row + AgentWaitWorkflow
                         reads the records first        (system-queue, id agent-wait:<id>)
                                                                 │
   agent event ─▶ agentevents.Publisher ─▶ NotifyEvent ─┐        │ timer (due / expiry)
   HOS poll ─────────────────────────────▶ NotifyHOS ───┤        │
   realtime stream ─▶ RecordWatchFeed ─▶ NotifyRecordChanges ┴─signal─┤ (settles: RecordChanged)
                                                                 ▼
                                    FinishWaitActivity ─▶ Finish: close the row, then
                                       conversation: a turn, origin WaitResolved
                                       background run: a run, trigger Wait
```

A wait never holds a conversation or a run open. The turn or run that sets one
ends as usual (the tool tells the agent to say what it is waiting for and stop).
The workflow costs nothing while it waits and survives every restart.

## Kinds

| Kind | Names | Ends when |
|---|---|---|
| `Time` | `at` (local, organization's zone) | the time comes |
| `StopArrival` / `StopDeparture` | `shipmentMoveId`, optional `stopId` | `shipment_move.arrived` / `departed` for the move (at that stop, when named) |
| `Reply` | exactly one of `shipmentId`, `carrierId`, `customerId` | `inbound_message.classified` matched to that record |
| `AppointmentNear` | `shipmentMoveId`, `stopId`, `minutesBefore` | `minutesBefore` before the stop's scheduled window start |
| `FreeTimeEnding` | `detentionOccurrenceId`, `minutesBefore` | `minutesBefore` before the occurrence's free time ends |
| `HOSDriveBelow` | `workerId`, `driveHoursBelow` | the drive clock drops below it, timed to the second while the driver drives |
| `PortalRun` | `portalRunId` | the portal task an approved `web_use_portal` started, or a `web_read_portal` that stopped for a sign-in code, ends (`NotifyPortalRun` from `PortalRunWorkflow`) |
| `DocumentRead` | `documentId` | the file's extraction is saved (`document.extracted`), or a look every minute finds its reading over: text in, and no shipment draft still `Pending` |
| `WebWatch` | `webWatchId` | the web watch finds something new (`NotifyWebWatch` from `WebWatchWorkflow`), or fails |
| `RecordChanged` | `recordId` (any record, by its id) | someone other than this conversation's own turn changes the record, then it is quiet for 30 s (at most 2 min from the first change) |
| `ReportRun` | `reportRunId` | the report run finishes (its `report-run` invalidation), or a look every two minutes finds it finished |

`PortalRun` is not one an agent asks for: `agent_create_wait` offers only `AgentKinds()`, and the
`web_use_portal` tool registers it itself (`RegisterPortalRunWait`, idempotent per run) on the
conversation or background run that filed the proposal, with a two-hour expiry. `web_read_portal`
registers one on its conversation (by thread, since a query tool has no run) when the portal
stops for a sign-in code the person can relay. While its run waits on that code, listing the
conversation's waits attaches `signInCode` to the wait for the code box, and
`AnnouncePortalRun` republishes the waits when the code is first asked for. See
[agent-portal-automation.md](agent-portal-automation.md).

`DocumentRead` is not one an agent asks for either. When `agent_read_attachment` finds a
file the person attached still being read, the turn's tool observer
(`assistantservice.WaitOnUnreadFile`) calls `WaitForDocumentRead`, which keeps one
open wait per file and conversation, with a half-hour expiry and the plan to carry
on with what the person asked of the file. `Register` refuses one on a file whose
reading has already ended, so the agent reads it instead. The timer looks at the
file every minute (`documentReadRecheckSeconds`, past `rearmSlack`) for readings
that save no extraction, such as a file that could not be read, which ends the
wait saying why. A turn that reads the file in full cancels its open wait
(`StopWaitingForDocumentRead`), and the turn that picks one up shows the file's
extraction card first. A delegated task never sets one; it reports the file to the
agent that handed it over.

`WebWatch` is not one an agent asks for either: `web_watch` registers it
(`RegisterWebWatchWait`, one open wait per watch) for the conversation or background run that
set the watch, lasting as long as the watch plus an hour (`MaxWebWatchLifetimeSeconds`, the one
kind allowed past 168 hours). Each delivery meets the waits set before its findings and sets
the next one for the same work while the watch goes on. It cannot be cancelled on its own: it
ends with its watch (`web_stop_watch`, or **Stop watching** on the Desk's row). Listing a
conversation's waits attaches `webWatch` (the watch, its cadence, when it ends and its findings
so far). See [agent-web-watching.md](agent-web-watching.md).

`RecordChanged` is the wait for "a person has to fix something before I can go on": an
invoice that will not post until the shipment has its POD, a queue item missing a rate. The
agent names the record that holds the problem (not the one it will change afterwards). A write
refused before filing (`refusedBeforeFiling`, `validateCall`) tells a conversation's agent to
do exactly that instead of ending on "tell me when it is done" (`personFixNote`); a delegated
task is never told, since it cannot set one. `Register` reads the record as the person
(`checkRecord`): its kind from its id prefix (`recordkind.OfIDPrefix`), their read grant on that
kind, and that the record is the organization's (`agentwaitports.RecordFinder`, the record
labeler's projections, which also give the label the wait keeps on `Condition.RecordLabel`). A
record they may not read is refused with the same words as a missing one. See "Hearing a
record change" below.

`ReportRun` is not one an agent asks for. `report_run` sets it (`WaitForReportRun`, one open
wait per run and conversation, two-hour expiry) when the run is still going after the tool's
few seconds and the agent said in `then` what it will do with the rows ("name the customers
over 90 days"). A report asked for on its own sets none: the card already shows it. The run is
read as the person who requested it; one already finished is refused with its outcome, so the
agent reads it now. The pick-up says how many rows and tells the agent to read them with
`report_get_run`; it never quotes them. A delegated task never sets one.

Every wait also has an expiry (`giveUpAfterHours`, default 24, at most 168; a
`Time` wait stays open at least five minutes past its time). A wait that runs
out is picked up anyway, told that it ran out.

## Registering

`Register` reads what the wait names before anything is recorded:

- a record that is not the organization's is refused;
- something already true is refused with what is true now ("the truck arrived at
  Kroger DC at …; act on that now instead of waiting"), so the agent acts;
- a timed wait computes `due_at`.

A wait belongs to a conversation (`thread_id`, `user_id`) or to a background
run's agent and record (`run_id`, `subject_type`, `subject_id`). At most ten are
open per owner (an advisory lock per owner makes the count exact). A delegated
task cannot set one. The row carries the setter's taint (`taint`) so the work
never picks up clean what it set aside after reading outside content; see
"Taint" below.

## Ending

- **Events.** `agentevents.Publisher` hands every event to `NotifyEvent` before
  it consults the plan (an organization without automated agents still has
  conversations that wait). Events carry `AgentEvent.Related` and `Detail`:
  arrivals and departures name the stop; `inbound_message.classified` names the
  matched shipment, carrier and customer, and its detail names the message and
  sender, never what it says. Open waits are found by
  `(organization, business unit, kind, watch_id) WHERE status = 'Waiting'`, one
  query per event, and their workflows are signalled `agent-wait-met`.
- **Drive time.** The HOS poll (every minute) calls `NotifyHOS` with the clocks
  it just stored. A clock below the threshold ends the wait. A driver who is
  driving runs the clock down a second a second, so the wait comes due at the
  moment it crosses, counted from the clock's own reading (`RecordedAt +
  remaining − threshold`); a changed crossing is stored and the workflow is
  signalled `agent-wait-reschedule` to sleep to the new time. When it comes due
  the check projects the clock from its last reading and ends the wait if it
  has crossed; a driver who stopped driving has no crossing until a read finds
  them driving again. The projection is only as wrong as a duty change since
  the last read, which the next read corrects within a minute.
- **Arrivals and departures, three ways.**
  - Recorded stop actuals publish `shipment_move.arrived` / `departed`, above.
  - A provider's geofence entry or exit (`telematicsservice.applyStopEvent`)
    reaches the waits on the tractor's active move whether or not the
    organization records stop actuals from it (`EnableAutoStopActuals`), for a
    visit that matches one of the move's stops.
  - Every position poll calls `NotifyPositions`. It reads the open arrival and
    departure waits of the tenant (one query; nothing more when there are none),
    the move and tractor of each once per poll, and checks the tractor's latest
    position, no older than fifteen minutes, against the watched stop's geofence
    (`geofence.Contains`: the radius of an automatic or circular one, the shape
    of a rectangle or a drawn one). An arrival ends inside it. A departure is
    being outside after being inside: the first sight inside is kept on the wait
    (`Condition.SeenInsideAt`), and a stop already recorded as arrived needs
    none. This covers a provider without geofence events and a geofence the
    provider does not hold.

  The same visit can arrive more than one way. A notice that finds the wait's
  workflow gone reads the wait first and picks nothing up for one already closed.
- **Timers.** A timed wait sleeps to `due_at`, then `CheckWaitActivity` reads the
  record again: an appointment or free time that moved puts it off, one already
  arrived or departed ends it, a record that is gone ends it saying so.
- **A lost workflow.** A signal that finds no workflow, on a wait still open,
  closes and picks it up in the notifier itself. A wait nobody signals is caught
  by `ReconcileAgentWaitsWorkflow`, on an hourly schedule
  (`agent-wait-reconcile`): every open wait fifteen minutes past its expiry, in
  every organization (`ListOverdueAcrossTenants`, under the system scope and
  listed in the RLS allowlist), is closed as run out and its work picked up.

## Hearing a record change

Every committed write already publishes a tenant-wide `resource.invalidation` to the realtime
stream ([realtime.md](realtime.md)), so a record change needs no new hook in any module. The
streamed kinds (`Kind.Streamed`: `RecordChanged`, `ReportRun`) register their watched id with
`realtimeports.RecordWatchFeed` (`realtimebroker.RecordWatchFeed`):

- **The register** is one sorted set per tenant (`trenova:realtime:watch:{org}:{bu}`), member
  the record id, score the wait's expiry. `Watch` keeps the later expiry (`ZADD GT`), trims
  expired members and keeps the key alive an hour past its last watch. When a streamed wait
  closes, `release` takes the record off only if no other open wait watches it, and `Unwatch`
  removes it only when it is not watched past that wait's expiry (a Lua check), so a later
  wait on the same record keeps it. A record whose release failed drops off at its expiry.
- **The follower** (`agentwaitjobs.WorkerModule`, worker processes only) reads every shard
  through one consumer group (`agent-waits`) with `NOACK`: each entry reaches one worker, and a
  change lost to a crash is what the wait's expiry is for. Each batch (up to 512 entries) keeps
  the tenant-wide invalidations by their headers, asks which of the batch's tenants hold a
  register (`EXISTS`, pipelined), decodes payloads only for those, and checks their records in
  one pipelined `ZMSCORE` round trip. Only watched changes are handed to `NotifyRecordChanges`, which reads
  the open waits on them, one query per tenant in the batch (`idx_agent_waits_open_watch`), and
  the names of whoever made them in one more.
- **A record change** older than the wait (`At < CreatedAt`) is what the agent already read and
  is dropped. Otherwise the wait's workflow is signalled with the change in one line
  (`worldwatch.DescribeChange`: the record, what became of it, who did it, the fields), keyed by
  record. `Kind.Settle` gives the workflow a quiet period (30 s) and a limit (2 min): it keeps
  the latest notice per record until nothing has arrived for the quiet period, then picks the
  work up once with what it gathered. A person fixing a record makes several edits; the agent
  reads it once, after the last.
- **A report run** notice reads the run; only a finished one ends the wait.

When a settled wait is picked up, `Finish` asks whether the conversation's owner started a
turn after the first change (`ConversationActivity.PersonSpokeSince`, the newest turns of the
thread by `idx_assistant_turns_thread`). If they did, they fixed the record and said so, and the
turn that answered them already read it: the wait closes as met and nothing is started.

## Picking the work up

`FinishWaitActivity` closes the row (only an open wait closes; a retry finds it
closed and picks up only what it has not) and then:

- **Conversation:** starts a turn with origin `WaitResolved` and
  `AssistantTurnRequest.ResumeWaitID`, as the conversation's owner. Its input is
  `Wait.ResumeNote()`, saved as a `WaitNote` message that the Desk draws as the
  wait it records. A met `RecordChanged` note tells the agent to read the record
  again, carry on if what held it up is resolved, and otherwise say what is still
  missing and set the wait again. The turn's model calls are kept under the
  `AgentWakeUp` usage feature (`RunRequest.UsageFeature`), so what waking up costs
  reads on its own line in AI usage. A conversation busy with another reply is
  `ErrConversationBusy`, retried with backoff for up to a day.
- **Background run:** starts a run of the same agent on the same record with
  trigger `Wait` and `AgentRunPayload.WaitID`; the run's input ends with the note
  and it opens with the wait's taint. A subject's run already open is retried
  like a busy conversation.

## Cancelling

`agent_cancel_wait` (the agent; only a wait its own conversation or, for a background
run, its own agent set) and `POST /assistant/threads/:threadID/waits/:waitID/cancel/`
(the owner) close the wait as `Cancelled` and cancel its workflow. Nothing is
picked up. A `WebWatch` wait is refused: stopping its watch closes it.

## Taint

A wait's `then` and description are the agent's own words, but a run that read
outside content may have been steered into writing them. A conversation's turns
inherit the conversation's taint already; a background run picking up a wait
opens with the taint stored on the wait. The event detail that ends a wait never
quotes outside content, with two exceptions: a `PortalRun` wait ends with what the portal
said, quoted and marked as written outside the organization (a task that strayed off its
site has its answer withheld), and a `WebWatch` wait with what the watch found, quoted and
marked the same way. `Wait.CarriesOutsideText` reports it, and the turn or run
that picks the work up opens with outside content already read
(`RunRequest.AfterExternalContent`), so every write it asks for is held for a person.

## Tools

`agent_create_wait` and `agent_cancel_wait` are core tools (every agent holds them), action
tools at `AutoExecute`, egress `internal`, gated on `agent_run:read`, and carry
taint. They change no record and reach nobody, so they run on their own; after a
web read they are capped at Propose like every other write.

## The Desk

`GET /assistant/threads/:threadID/waits/` lists a conversation's waits; the
composer shows the open ones ("Waiting on …", when each comes due or gives up,
cancel), moved by the `agent_waits` realtime event addressed to the owner. A
record change reads "Picks up when it changes" with when it gives up, and a
report run "Running": their due times are rechecks, not when the work comes back.

## Known limits

None that the code can remove. What remains is the data: a wait hears an
arrival as soon as the provider reports a geofence event, a stop actual is
recorded, or a polled position shows it, whichever is first; and drive time is
projected exactly only while the duty status the last clock read reported still
holds. A record change is heard at most once: a worker that dies holding a batch
loses it, and the wait picks the work up when it runs out, told so. A write that
publishes no invalidation is never heard; every write is meant to publish one
([realtime.md](realtime.md)).

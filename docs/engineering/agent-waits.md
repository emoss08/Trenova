# Durable waits

How an agent parks its work until something happens in the world and picks it
up again when it does: a truck reaching or leaving a stop, a reply about a
shipment or from a carrier or customer, an appointment coming round, detention
free time running out, a driver's drive time falling low, or a time.

Read this before changing `domain/agentwait`, `agentwaitservice`,
`temporaljobs/agentwaitjobs`, the `wait_until` / `cancel_wait` tools, or any of
the places that tell waits what happened (below).

## The shape of it

```
wait_until (tool) ──▶ agentwaitservice.Register ──▶ agent_waits row + AgentWaitWorkflow
                         reads the records first        (system-queue, id agent-wait:<id>)
                                                                 │
   agent event ─▶ agentevents.Publisher ─▶ NotifyEvent ─┐        │ timer (due / expiry)
   HOS poll ─────────────────────────────▶ NotifyHOS ───┴─signal─┤
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

## Picking the work up

`FinishWaitActivity` closes the row (only an open wait closes; a retry finds it
closed and picks up only what it has not) and then:

- **Conversation:** starts a turn with origin `WaitResolved` and
  `AssistantTurnRequest.ResumeWaitID`, as the conversation's owner. Its input is
  `Wait.ResumeNote()`, saved as a `WaitNote` message that the Desk draws as the
  wait it records. A conversation busy with another reply is
  `ErrConversationBusy`, retried with backoff for up to a day.
- **Background run:** starts a run of the same agent on the same record with
  trigger `Wait` and `AgentRunPayload.WaitID`; the run's input ends with the note
  and it opens with the wait's taint. A subject's run already open is retried
  like a busy conversation.

## Cancelling

`cancel_wait` (the agent; only a wait its own conversation or, for a background
run, its own agent set) and `POST /assistant/threads/:threadID/waits/:waitID/cancel/`
(the owner) close the wait as `Cancelled` and cancel its workflow. Nothing is
picked up.

## Taint

A wait's `then` and description are the agent's own words, but a run that read
outside content may have been steered into writing them. A conversation's turns
inherit the conversation's taint already; a background run picking up a wait
opens with the taint stored on the wait. The event detail that ends a wait never
quotes outside content.

## Tools

`wait_until` and `cancel_wait` are core tools (every agent holds them), action
tools at `AutoExecute`, egress `internal`, gated on `agent_run:read`, and carry
taint. They change no record and reach nobody, so they run on their own; after a
web read they are capped at Propose like every other write.

## The Desk

`GET /assistant/threads/:threadID/waits/` lists a conversation's waits; the
composer shows the open ones ("Waiting on …", when each comes due or gives up,
cancel), moved by the `agent_waits` realtime event addressed to the owner.

## Known limits

None that the code can remove. What remains is the data: a wait hears an
arrival as soon as the provider reports a geofence event, a stop actual is
recorded, or a polled position shows it, whichever is first; and drive time is
projected exactly only while the duty status the last clock read reported still
holds.

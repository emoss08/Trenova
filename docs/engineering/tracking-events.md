# Tracking events

How a stop's actual arrival and departure are settled when several sources report them, in
any order and more than once, and how each source reaches that path. Read this before
changing `trackingevent`, `trackingeventservice`, `ShipmentMoveService.ReconcileStopActuals`,
the telematics stop-visit path, the inbound EDI 214 parser, or anything else that records a
stop actual.

## The shape of it

```
Dispatcher (REST record-actual) ─┐
Agent (record_stop_actual)  ─────┤ RecordReported ── RecordStopActual (strict) ─┐
Driver (Dash recordMyStopAction) ┘   (one transaction)                          │
                                                                                ▼
Telematics (geofence, provider stop) ─┐                        tracking_events (one row per report,
EDI 214 (every AT7)  ─────────────────┴ RecordObserved ──────▶  unique on org, bu, source, source_key)
                                                                                │
                                                                                ▼
                                    ReconcileStopActuals: lock move ─▶ Project(all events) ─▶ write
                                    changed stops once ─▶ derive move status ─▶ verdict per event
```

| Piece | Where |
|---|---|
| Entity, enums, the projection | `internal/modules/operations/domain/trackingevent` |
| Applying a projection to a move | `shipment.ShipmentMove.ApplyStopActualChanges` (`domain/shipment/stopactualchange.go`) |
| Which stop a visit is about | `domain/shipment/stopvisit.go` (`StopForObservedVisit`, `StopForReportedVisit`, `MatchObservedVisit`) |
| Service | `internal/modules/operations/app/trackingeventservice` |
| Reconcile under the move lock | `app/shipmentmoveservice/stopactual_reconcile.go` |
| Repository and table | `adapters/postgres/trackingeventrepository`, migration `20261231009010_tracking_events` |
| Telematics port | `workforce/ports/telematicsports.StopVisitRecorder` |
| Driver portal port | `workforce/ports/driverportalports.StopReporter` |
| EDI 214 parser and stop targeting | `edi/app/ediinboundservice/parse.go`, `edi/app/ediservice/stopactuals214.go`, `edi/domain/edi/x12time.go` |
| GraphQL read | `shipmentTrackingEvents(shipmentId)` in `shipment.graphqls` |
| Web | "Stop reports" under move details (`routes/shipment/_components/move/stop-reports.tsx`) |

## The record

A `tracking_events` row is one report of one visit: the stop, `Arrival` or `Departure`,
the source, `event_at` (when it happened, as the source said) and `received_at` (when
Trenova got it), optional coordinates, the source's own status code, a raw reference back to
what it came from (`telematics_event:<id>`, `edi_inbound_file:<file>:<control>:<index>`), who
reported it, and how it was tied to its stop (`Direct`, `StopReference`, `Location`,
`StopRole`, `Inferred`).

`(organization, business unit, source, source_key)` is unique. A repeat of a report is the
same key and is not stored twice; the service returns the stored row with `Replayed` set and
does not reconcile again. The key per source:

| Source | Key |
|---|---|
| Dispatcher | the request's scoped `Idempotency-Key`, else a new id |
| Agent | the tool call's idempotency key |
| Driver | `<user id>:<clientEventId>` from the device, else a new id |
| Telematics | `<provider>:<event id>` |
| EDI | `edi:<tender offer>:<stop>:<status code>:<event time>` |

A report with a departure time attached (`Arrive` plus `departedAt`) is two rows; the
departure's key is the arrival's plus `:departure`. A D1 with no arrival on record adds an
`Inferred` arrival under the departure's key plus `:arrival`.

## The projection

`trackingevent.Project(move, events, opts)` is pure. Given the move as it is and every event
on it, it returns the stop changes to make and a verdict for every event. It is
order-independent by construction: the same set of events gives the same result whatever
order they arrived in, which `projection_test.go` checks over 500 shuffles and the service
test checks end to end with repeats.

For each stop in sequence, arrival then departure:

1. Candidates are ordered by source precedence (Dispatcher 50, Agent 40, Driver 30,
   Telematics 20, EDI 10), then earliest event time, then source and key.
2. A load that is invoiced or in the billing queue is frozen: its candidates are
   `Superseded` (or `Applied` when they match what is on record). A canceled load refuses
   them.
3. A time already on the stop that no candidate carries was entered some other way (the
   shipment form, a split); it stands and the candidates are `Superseded`.
4. An arrival waits (`Pending`) while an earlier stop has no departure; a departure waits
   while its own stop has no arrival. Waiting events are applied by a later reconcile, so an
   arrival that comes in before the previous stop's departure lands once that departure does.
5. The first candidate inside the bounds wins: an arrival no earlier than the previous
   stop's departure and no later than this stop's departure; a departure no earlier than
   this arrival and no later than the next stop's arrival. Others at the same time are
   `Duplicate`; the rest `Superseded`. If none fits, all are `Refused` with the bound they
   broke.

`ReconcileStopActuals` locks the move `FOR UPDATE`, loads the events inside the lock, runs the
projection, applies the changes and repeats until nothing changes (an applied departure can
release a waiting arrival). It writes each touched stop once, derives the move status
(`Completed` when every stop has departed, `InTransit` once one has arrived), refreshes the
shipment's state, and records the verdicts that changed. Post-commit effects (move events,
service-failure evaluation, detention, observers, agent waits) run once through
`kernelports.AfterCommit`, so a rolled-back reconcile announces nothing.

## The two ways in

**Interactive sources** (Dispatcher, Agent, Driver) are a person or agent saying "this
happened". `RecordReported` inserts the event and calls `RecordStopActual` in the same
transaction, so the strict checks people are used to (earlier stops first, no arrival after
departure, holds) still answer them with a 422, and nothing is stored when they refuse. The
reconcile that follows records the verdicts.

**Machine sources** (Telematics, EDI) are reports Trenova has to make sense of. `RecordObserved`
inserts the event, then reconciles. A refusal from the move (a canceled load, a rule) marks
that event `Refused` with the reason instead of failing the caller; any other error is
returned and the event stays `Pending` for the next reconcile.

Telematics declares `StopVisitRecorder` in workforce and operations implements it, so
workforce takes no new upward dependency. The stop is chosen by `StopForObservedVisit`: the
stop at the geofence's location the truck is at, else the next one it has not reached, else
the last one it visited, so a repeat or late ping is still recorded against its stop rather
than dropped. The telematics row keeps its own outcome (`Recorded`, `Duplicate`, `Pending`,
`Unmatched`, `Refused`), and the last two still raise a watchtower item.

The Dash driver portal hands a tap to `StopReporter` with the device time and a per-tap id. A
device clock running ahead is read as now, since a tap cannot be in the future.

## EDI 214

The parser reads every `AT7` in every `LX` loop. Each status keeps:

- the AT7-01 status and AT7-02 reason (or AT7-03/04 for an appointment status),
- the AT7-05 date, AT7-06 time (4, 6 or more digits) and AT7-07 time code,
- MS1 city, state, country and coordinates (DDDMMSS with an E/W/N/S direction, or decimal),
- MS2 equipment owner, number and type,
- the loop's `L11*<n>*QN` stop sequence, and the loop's N4 city and state when there is no MS1.

A time code names a zone. `ET`, `CT`, `MT`, `PT`, `AT` (Alaska), `HT`, `TT` (Atlantic) and `NT`
follow daylight saving; the `S` and `D` variants are fixed offsets; `UT` and `GM` are UTC.
`LT`, an unknown code or no code is local time at the stop, read in the stop location's time
zone once the stop is known (UTC when the location has none). `edi.X12DateTime` does the
conversion.

`X3`, `AF`, `X1`, `D1` and `CD` move stop actuals. The stop is the one whose sequence the
carrier echoed, else a stop of the status's role (pickup or delivery) in the MS1 or N4 city,
else the stop of that role the truck is at or due at next, so intermediate stops are reached
in turn. Each status writes its shipment comment and event at the corrected time, then becomes
a tracking event. Refused and waiting statuses are warnings on the inbound file.

## Adding a source or a caller

- Never call `RecordStopActual` for a new source. A person's report goes through
  `RecordReported`; anything a machine reports goes through `RecordObserved`.
- Give every report a key that is the same when the same report is sent again. A key reused
  for a different stop or action is refused.
- A module below operations declares its own narrow port and operations implements it in
  `module.go` (see the telematics and driver portal ports); do not add an upward import.
- A new source needs a precedence in `trackingevent.sourcePrecedence`, the CHECK constraint on
  `tracking_events.source`, the GraphQL enum and the web labels.

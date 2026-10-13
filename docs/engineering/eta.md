# Projected arrivals (stop ETAs)

How Trenova works out when a truck will reach each open stop, where that answer is stored, what
refreshes it, and how an at-risk load reaches a dispatcher. Read this before changing
`stopeta`, `etaprojection`, `shipmentetaservice`, `stopetarepository`, `etajobs`, `etatrigger`,
the `EtaAtRisk` watchtower source, `hosprojection.Schedule`, the places route estimator or the
arrival settings on dispatch controls and customers.

## The shape of it

```
stop event / assignment / check call ──▶ etatrigger ──▶ RefreshShipmentEtaWorkflow (per shipment)
shipment-eta-sweep (5 min, per tenant) ─────────────────────────────▶ RefreshTenant
                                                                        │
                       ShipmentEtaService.RefreshShipments ◀────────────┘
                         load shipments, board, positions, hours, check calls, exemptions
                         etaprojection.Prepare ─▶ legs ─▶ RouteEstimator ─▶ Plan.Project
                         ReplaceForShipments (stop_etas) ─▶ changed shipments
                                       │                            │
                         stop_eta_observers (EtaAtRisk)     realtime "shipmentEta"
```

| Piece | Where |
|---|---|
| Row, enums, governing stop | `internal/modules/operations/domain/stopeta` |
| Estimator (pure) | `operations/app/etaprojection` (`plan.go`, `project.go`, `reason.go`) |
| Rest for hours of service | `workforce/app/hosprojection/schedule.go` |
| Drive time | `places/ports/routeestimateports`, `places/app/routeestimateservice` |
| Refresh and reads | `operations/app/shipmentetaservice` (`refresh.go`, `read.go`, `collect.go`) |
| Table | `operations/adapters/postgres/stopetarepository`, migration `20261231009190_stop_etas` |
| Triggers | `operations/app/etatrigger` |
| Workflows and sweep | `operations/jobs/etajobs` (`etacontract` holds the requester) |
| Watchtower | `agent/app/watchtowersources/eta.go`, `DescribeEtaRisk` in `describe.go` |
| GraphQL | `Shipment.eta` (`ShipmentEta`, `StopEta`) in `shipment_board.graphqls` |
| REST | `GET /api/v1/shipments/:shipmentID/eta/` |
| Analytics | `etaSlip` in `analyticsservice/providers/shipmentprovider` |
| Web | `routes/shipment/_components/move/stop-eta.tsx`, the board's ETA cell, dispatch control and customer forms |

## The row

`stop_etas` holds one row per open stop of a tracked shipment, unique on
`(organization, business unit, stop)`. A shipment is tracked while it is `PartiallyAssigned`,
`Assigned`, `InTransit`, `Delayed` or `PartiallyCompleted` (`stopeta.TrackedShipmentStatuses`).
A stop is open until it is completed, canceled or departed, on a move that is neither completed
nor canceled. The truck being at a stop (arrived, not departed) keeps the row with
`arrived = true` and the recorded arrival.

Each row carries the estimated arrival and departure, the window end it is judged against
(`Stop.EffectiveScheduledCutoff`), slack in minutes (negative when the window will be missed),
the verdict, what it was worked from (`basis`), how the drive time was found (`drive_method`),
miles, drive and rest minutes from the starting point, when the input was recorded and whether
it is stale, the threshold that applied, a sentence saying what it rests on, and when it was
computed. Times are Unix seconds.

`ReplaceForShipments` deletes the rows of the named shipments whose stops are no longer open and
upserts the rest, keeping each stop's row id. A computation older than the stored one is ignored
(`computed_at <= EXCLUDED.computed_at`), so an overlapping sweep cannot overwrite a newer
refresh. Rows are refused when they name another organization or a shipment outside the request.

## Working it out

`etaprojection.Prepare` picks a starting point for the first open stop, in this order:

1. The truck is at the stop: `AtStop`, arrival is the recorded one.
2. A fresh tractor position (an hour or newer): `Position`.
3. A check call within four hours that states an arrival for this stop (or for no stop), not
   more than fifteen minutes in the past: `CarrierReported`, and that arrival is used.
4. A check call position newer than the tractor position: `CheckCall`, stale after four hours.
5. A stale tractor position: `Position`, marked stale.
6. The last departed stop: `LastDeparture`. If the truck should have arrived by now, the
   arrival is shown as now and the row is marked stale.
7. Nothing: `None`, no arrival.

Legs then run from the starting point through each open stop's coordinates. A stop with no
coordinates breaks the chain: it and every stop after it have no arrival. `Plan.Legs()` hands the
legs to the places `RouteEstimator`, which routes them through the organization's PC*MILER
integration when one is configured (routed miles and the mileage report's drive hours, cached
per leg for six hours and keyed by the distance profile) and otherwise uses straight-line miles
widened by the road circuity factor at the planning speed (`geoutils.PlanningRoadMiles`,
`PlanningDriveSeconds`). One refresh sends every leg of every shipment in a batch.

`Plan.Project` walks the stops in route order:

- Arrival is the previous departure plus the leg's elapsed time. Departure is
  `max(arrival, window start) + dwell`; the truck at a stop never departs before now.
- With a driver on the move whose hours are on file, fresh (12 hours) and not ELD or short-haul
  exempt, `hosprojection.Schedule` adds the 30 minute break, the 10 hour reset and the 34 hour
  restart the drive needs. Waiting for a window is off duty and dwell is on duty, so a long wait
  is a reset. A later move with a different driver stops counting rest at the handover. A
  carrier's loads have no hours.
- The verdict: past the window with no arrival is `Late`; an arrival after the window end is
  `Late`; slack under the threshold is `AtRisk`; otherwise `OnTime`; no arrival is `Unknown`. A
  shipment marked `Delayed` turns an `Unknown` stop `Late`. A stop with no window is `OnTime`.

The threshold is the customer's `eta_risk_threshold_minutes` when set, else dispatch controls'
(`DispatchControl.EtaThresholdMinutes`, 0 to 1440, default 60). Dwell is dispatch controls'
`eta_dwell_minutes` (0 to 720, default 60).

## What refreshes it

- `etatrigger` hears shipment events (created, updated, status, canceled and uncanceled, move
  status, arrived, departed, stop completed, driver and carrier assigned or unassigned, tender
  accepted) and check calls (the `check_call_observers` group), and calls
  `RefreshRequester.RequestRefresh`.
- The requester signals-with-start `RefreshShipmentEtaWorkflow` with id
  `shipment-eta-refresh-<shipment id>` on the dispatch queue. The workflow waits ten seconds,
  refreshes, and refreshes again while signals keep arriving, continuing as new after 25 rounds,
  so a burst of edits is one refresh and an edit during a refresh is never lost.
- `shipment-eta-sweep` runs every five minutes on the dispatch queue: one activity per tenant
  with tracked shipments or leftover rows (`ListActiveTenants`, under a system scope), each
  calling `RefreshTenant`. The sweep is how tractor positions and driver hours reach the
  projection, and how inputs go stale. Positions are not routed on every one-minute telematics
  poll: a moving truck misses the leg cache every time, and routing each truck each minute would
  spend the routing quota for no better answer.

A refresh announces only the shipments whose rows changed (verdict, basis, arrival to the
minute, slack, window, arrival state or staleness) to the `stop_eta_observers` group and as a
`shipmentEta` realtime invalidation, which refetches the shipment's stop ETAs and the board.

## At-risk loads

`EtaAtRisk` is a watchtower kind keyed by shipment id and read with the shipment permission. One
item per shipment describes its governing stop (`stopeta.Governing`: the worst unarrived stop,
earliest first): a warning when the slack is under the threshold, critical when the projection
is after the window, with a note when the input is stale. Its text does not carry minutes, so the
feed is not rewritten by each refresh.

- The agent module's `EtaAtRiskObserver` upserts or resolves items for the shipments a refresh
  changed. The truck reaching the stop, the load recovering, the shipment finishing and the
  organization turning `enable_eta_alerts` off all resolve it.
- `EtaAtRiskSource` lets the nightly reconcile correct the feed from `ListAttention`.

## Reads

Everything reads the stored rows; nothing estimates on read.

- `ShipmentEtaReader.EtasByShipmentIDs` (GraphQL `Shipment.eta`, the REST route, Desk cases)
  summarises a shipment by its governing stop and adds an open service failure's reason when the
  load needs attention. A shipment with no rows has no ETA (REST answers `Unknown`).
- `ShipmentTrackingReader.TrackingSnapshots`, the agent tool `get_shipment_tracking` and the
  shipment subject build the tracking snapshot with `Input.StopETAs`, so the agent quotes the same
  arrival as the board.
- `etaSlip` on the shipments analytics counts active shipments with an unarrived `AtRisk` or
  `Late` row; those shipments also count toward the at-risk total.

## Changing it

- A new starting point needs a `stopeta.Basis` value, the CHECK constraint on
  `stop_etas.basis`, the GraphQL enum, `use-eta-labels.ts`, a branch in `chooseOrigin` and a
  sentence in `reason.go`.
- Keep `etaprojection` pure: it takes the inputs and the leg estimates and never reads the clock
  or a repository, so every rule is a unit test.
- A new trigger goes through `RefreshRequester`; never refresh synchronously inside a request.
- `enable_eta_alerts` defaults to true in the table and is set where the dispatch control row is
  built; its bun tag must not carry `default:`, or a `false` would be discarded on insert.

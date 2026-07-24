# Samsara Simulator

`samsara-sim` is a local simulator for Samsara API behavior used by TMS integration flows.

## Run

```bash
cd services/samsara-sim
task run
```

Server defaults:

- `http://localhost:8091`
- bearer token: `dev-samsara-token`
- health: `GET /_sim/health`
- live map: `GET /_sim/map`

## Configure TMS

Point TMS to the simulator:

```yaml
samsara:
  token: dev-samsara-token
  baseURL: http://localhost:8091
```

## Fleet API Surface

Samsara-shaped endpoints (bearer auth required):

- `GET /fleet/vehicles/stats` — latest snapshot per vehicle (singular stat objects)
- `GET /fleet/vehicles/stats/feed` — Telematics Sync feed with plural stat arrays.
  `types` is required (comma-separated, supported: `gps`, `engineStates`,
  `fuelPercents`, `obdOdometerMeters`, `ecuSpeedMph`, `batteryMilliVolts`).
  The first call without `after` returns the latest sample per vehicle plus
  `pagination.endCursor`; subsequent calls with `after=<cursor>` return only
  samples strictly newer than the cursor at the 2-minute sample cadence.
  `hasNextPage: true` means page again immediately; `false` means caught up
  (the cursor stays stable). Supports `vehicleIds` and `limit`.
- `GET /fleet/vehicles/stats/history` — same plural shape over a required
  `startTime`/`endTime` RFC3339 window with standard pagination.
- `GET /fleet/hos/clocks`, `GET /fleet/hos/logs`
- `GET /fleet/hos/daily-logs` — per-day driver log summaries derived from the
  same deterministic duty timeline as `/fleet/hos/logs` (duty-status durations,
  drive distance, and certification metadata stay mutually consistent with
  clocks and logs). Supports `driverIds`, `startDate`/`endDate` (`YYYY-MM-DD`,
  default: last 7 sim days ending today, max 30-day range), and `after`/`limit`.
  Records are ordered newest day first per driver.
- `GET /fleet/hos/violations` — derived from deterministic HOS sim events.
  Supports `driverIds`, `types`, `startTime`/`endTime` (default: last 24h of
  sim time), and `after`/`limit`. Violation types use the real Samsara enum:
  `restbreakMissed`, `shiftHours`, `shiftDrivingHours`, `cycleHoursOn`.
- `GET /fleet/dvirs/history` — deterministic DVIRs derived from the same duty
  timeline as HOS: each driver files a `preTrip` DVIR at their sim workday
  start and a `postTrip` DVIR at workday end, tied to their assigned vehicle.
  Requires `startTime`/`endTime` (RFC3339, max 30-day range, filtered on DVIR
  `endTime`); supports `driverIds`, `vehicleIds`, and `after`/`limit`. Records
  use the real `Dvir` shape: `id`, `type` (`preTrip`/`postTrip`),
  `safetyStatus` (`safe`/`unsafe`/`resolved`, ~10% flagged), `startTime`,
  `endTime`, `odometerMeters` (odometer machinery), `location`,
  `licensePlate`, `driver{id,name}`, `vehicle{id,name}`,
  `authorSignature{signatoryUser{id,name},signedAtTime,type}`, and
  `vehicleDefects[]` (`id`, `defectType`, `comment`, `createdAtTime`,
  `isResolved`, `resolvedAtTime`, `vehicle`) on flagged DVIRs. Defects from
  DVIRs at least 2 sim-days old resolve deterministically
  (`safetyStatus: resolved` + `resolvedAtTime`).
- `GET /form-templates` — five fixture templates with real field shapes:
  Fuel Receipt (`number`/`text`), Incident Report (`text`/`multiple_choice`),
  Trip Inspection Checklist (`check_boxes`), Bill of Lading (Shipper)
  (`formCategory: routing`; `text`/`number`/`multiple_choice`/`signature`
  fields: Seal Number, Pieces Loaded, Gross Weight, Trailer Temperature, Load
  Secured, Shipper Signature, Pickup Notes) and Proof of Delivery (Consignee)
  (`formCategory: routing`; Pieces Delivered, Delivery Temperature, Condition
  on Arrival, Seal Intact, Receiver Name, Receiver Signature, Delivery Notes).
- `GET /form-submissions` — fixture records plus generated submissions from
  the trailing 24h of sim time (`ids` lookups search the trailing 7 sim days).
- `GET /form-submissions/stream` — deterministic submissions generated per
  driver/day (1-2 generic per driver per sim-day, timestamps inside the
  workday; fuel gallons/amounts hashed, checklists mostly passing). Each driver
  also files a Bill of Lading near their workday START (pickup) and a Proof of
  Delivery near their workday END (delivery), tied to that driver. Shipment
  field values are deterministic from `hashFraction`: seal numbers `SL-######`,
  pieces 4-26, gross weight 8000-44000 lbs, reefer temperature 34-38°F, Load
  Secured / Condition on Arrival biased to Yes/Good, and signatures rendered as
  media stubs (`signatureValue.media{id,processingStatus,url,urlExpiresAt}`
  where `urlExpiresAt` is sim-now + 1h). Requires `startTime` (RFC3339,
  `endTime` defaults to now, max 30-day range, filtered on `updatedAtTime`);
  supports `formTemplateIds`, `driverIds`, `userIds`, and `after`/`limit`.
  Records use the real form-submission shape (`id`, `title`, `status`,
  `isRequired`, `createdAtTime`, `updatedAtTime`, `submittedAtTime`,
  `submittedBy{id,type}`, `formTemplate{id,revisionId}`, `externalIds`,
  `location{latitude,longitude}` from the driver's route position at submit
  time, and — when the driver's route resolves — `routeId` plus `routeStopId`
  (first stop for the BOL pickup, last stop for the POD delivery), with
  `fields[]` carrying
  `numberValue`/`textValue`/`multipleChoiceValue`/`checkBoxesValue`/`signatureValue`).
- `GET /fleet/routes`, `GET /fleet/drivers`, `GET /assets`,
  `GET /assets/location-and-speed/stream`, `GET /addresses`, `GET /webhooks`

Fixture drivers carry real-shape `eldSettings`
(`{"rulesets":[{"cycle","shift","restart","break","jurisdiction"}]}`) exposed
through `GET /fleet/drivers`: most run `USA 70 hour / 8 day` /
`US Interstate Property`, two run `USA 60 hour / 7 day`, and one runs the
`Texas Intrastate` shift — all with `jurisdiction: TX`.

## Geofences

Fixture addresses carry real-shape geofence data
(`{"geofence":{"circle":{"latitude":..,"longitude":..,"radiusMeters":..}}}`),
including circles placed directly on the Texas OSM route geometry so vehicles
deterministically pass through them. Entry/exit transitions are evaluated at
the 2-minute sample cadence during stats/feed polls and emit `GeofenceEntry` /
`GeofenceExit` webhooks with the real address + vehicle payload (including
`vin` and `licensePlate`). `GET /_sim/state/summary` reports dispatched
geofence transition counts.

## Webhooks

All outbound webhooks use the real Samsara Webhooks 2.0 envelope:

```json
{
  "eventId": "<deterministic uuid>",
  "eventTime": "2026-03-01T14:00:00.000Z",
  "eventType": "GeofenceEntry",
  "orgId": 20936,
  "webhookId": "wh-1",
  "data": {}
}
```

Emitted event types are real Samsara types only: `SpeedingEventStarted` /
`SpeedingEventEnded`, `SevereSpeedingStarted` / `SevereSpeedingEnded`,
`AlertIncident` (HOS violations), `RouteStopEtaUpdated` (traffic delays),
`GeofenceEntry` / `GeofenceExit`, `RouteStopArrival` / `RouteStopDeparture`
(route-stop tracking), `DvirSubmitted` (DVIR completions:
`{driver, vehicle, dvir{...defects}}`), `FormSubmitted` / `FormUpdated`
(`{form: {...}}`), and resource CRUD events
(`AddressCreated`, `VehicleCreated`, `DriverCreated`, ...).
`RouteStopArrival` / `RouteStopDeparture`, `DvirSubmitted` and `FormSubmitted`
fire lazily during stats polls when a route-stop crossing, DVIR `endTime`, or
submission `submittedAtTime` falls inside the dispatch window, with the same
deduplication as geofence events.

`RouteStopArrival` / `RouteStopDeparture` reuse the geofence machinery: when a
vehicle that is assigned to a route crosses an address geofence, the entry
emits a `RouteStopArrival` (aligned with `GeofenceEntry`) and the exit emits a
`RouteStopDeparture` (aligned with `GeofenceExit`). The `data` payload follows
the real route-tracking shape: `{operation ("stop arrived"/"stop departed"),
type: "route tracking", time, assignedToRoute, driver{id,name,externalIds},
vehicle{id,name,assetType,licensePlate,vin,externalIds},
route{id,name,externalIds}, routeStopDetails{id, state ("arrived"/"departed"),
eta, enRouteTime, actualArrivalTime, actualDepartureTime, externalIds,
orders[]}}`.

Signing follows the real Samsara scheme:

- headers: `X-Samsara-Timestamp` (RFC3339) and `X-Samsara-Signature: v1=<hex>`
- signed string: `"v1:" + timestamp + ":" + rawBody`
- key: the webhook `secretKey` (or global `webhooks.signingSecret`) is treated
  as base64; it is base64-STD-decoded to raw key bytes before HMAC-SHA256, and
  used as raw bytes when it is not valid base64.

Webhook records expose their `secretKey` in `GET /webhooks` responses, and a
deterministic base64 `secretKey` is generated when a webhook is created
without one. The simulator also keeps its `X-Samsara-Sim-Delivery-*` headers
for duplicate/reorder/attempt introspection.

## Scenarios

Default scenario profile:

- `default`: full payload fidelity
- `sparse`: omit optional fields
- `partial`: partial collections and optional field omissions
- `degraded`: partial + omissions + deterministic 503 responses

Webhook event omission behavior:

- `default`: no event omission
- `sparse`: omits a meaningful portion of webhook events
- `partial`: omits many webhook events
- `degraded`: omits most webhook events

Override per request:

- header: `X-Samsara-Sim-Profile: sparse`

Control plane:

- `GET /_sim/scenarios`
- `GET /_sim/scenarios/active`
- `PUT /_sim/scenarios/active` with `{"profile":"degraded"}`
- `GET /_sim/time`
- `PUT /_sim/time`
- `POST /_sim/time/step`
- `GET /_sim/scripts/status`
- `GET /_sim/faults`
- `PUT /_sim/faults`
- `POST /_sim/faults/rules`
- `DELETE /_sim/faults/rules/{id}`
- `POST /_sim/faults/reset`
- `GET /_sim/state/summary`
- `POST /_sim/state/reset`
- `POST /_sim/events/trigger`
- `GET /_sim/events/active`
- `GET /_sim/events/window`

## Docker

Start simulator container from repository root:

```bash
docker compose -f docker-compose-local.yml --profile samsara-sim up -d samsara-sim
```

## Route Dataset

The simulator ships with a Texas route dataset at:

- `config/datasets/texas_osm_routes.geojson`

This dataset is derived from OpenStreetMap road geometry via the OSRM demo server and is used to seed realistic asset waypoints.
It now includes 12 long-haul corridors for multi-hour travel simulation.

Refresh the dataset:

```bash
cd services/samsara-sim
task routes-fetch
```

Disable or override route dataset loading in config:

```yaml
seed:
  routeDatasetPath: ./config/datasets/texas_osm_routes.geojson
```

## Simulation Controls

Tune long-haul behavior and event realism:

```yaml
simulation:
  fleetSize: 12
  tripHoursMin: 8  # minimum simulated loop duration for moving vehicles
  tripHoursMax: 12 # deterministic target band used for short-route stretching
  eventIntensity: balanced # balanced|compliance|driving
  violationRate: 0.08
  speedingRate: 0.14
  scriptPath: ./config/scenarios/default.yaml
  scriptMode: merge # merge|override
  scriptTimezone: UTC
```

Operational events are deterministic by seed and include:

- duty transitions (`offDuty`, `sleeperBerth`)
- stop/delay periods
- speeding bursts
- HOS violation windows

Events affect API state (`/fleet/vehicles/stats`, `/fleet/hos/clocks`, `/fleet/hos/logs`)
and are emitted as webhooks when webhooks are enabled.

Movement loop timing uses `tripHoursMin`/`tripHoursMax` to stretch short routes so
vehicles do not complete unrealistically short loops.

Simulation time is virtual and controllable via `/_sim/time`:

- pause and resume
- set explicit simulation timestamp
- step simulation forward deterministically

Scenario scripts are loaded from YAML and merged or overridden per `simulation.scriptMode`.
Rule-based fault injection for endpoints and webhook event delivery is available through `/_sim/faults`.

## Smoke Test

Run HOS and asset-location smoke checks:

```bash
cd services/samsara-sim
task smoke-sim
```

Run event and correlation checks:

```bash
cd services/samsara-sim
task smoke-events
```

Run time-control checks:

```bash
cd services/samsara-sim
task smoke-time
```

Run fault-injection checks:

```bash
cd services/samsara-sim
task smoke-faults
```

Run CI smoke suite (health, route lifecycle, moving GPS, HOS delta, webhook inbox, rate-limit headers):

```bash
cd services/samsara-sim
task smoke-ci
```

Optional overrides:

- `SIM_BASE_URL` (default: `http://localhost:8091`)
- `SIM_TOKEN` (default: `dev-samsara-token`)

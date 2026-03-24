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

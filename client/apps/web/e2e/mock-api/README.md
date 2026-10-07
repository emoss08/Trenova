# Mock API

A dependency-free Node server that stands in for the TMS API, so the shipments board
(`/shipment-management/shipments`) and AI control (`/admin/agent-control`) can be run, clicked
through and screenshotted without Postgres, Redis, Temporal or an AI provider. It answers the REST session endpoints the app
shell needs and every GraphQL operation the board sends, from a deterministic fixture.

## Run it

```bash
# terminal 1 — the mock API on :8080 (the dev client's API origin)
node e2e/mock-api/server.mjs

# terminal 2 — the web app on :5173
pnpm --filter @trenova/web dev
```

Open <http://localhost:5173/shipment-management/shipments>. You are signed in as the
fixture user with every permission.

The server answers credentialed cross-origin requests only from `http://localhost:5173` and
`http://127.0.0.1:5173`. Set `MOCK_API_ORIGINS` to a comma-separated list to serve the app from
another origin.

## Scenarios

The board is drawn for 11:30 UTC today; every timestamp in the fixture is relative to that
anchor. A scenario is set from the environment when the server starts, or at any time with
`POST /__mock/scenario` (which rebuilds the state from scratch):

| Field | Values | Default | Effect |
|---|---|---|---|
| `board` | `quiet` \| `busy` \| `high` | `high` | 2 sample loads, the 14-load prototype board, or 480 shipments |
| `ai` | `true` \| `false` | `true` | Briefing sentence, AI wording in the brief |
| `operationType` | `asset` \| `brokerage` \| `both` | `both` | Drivers, carriers or both in the strip and queue |
| `hos` | `true` \| `false` | `true` | HOS rings, sublines and HOS suggestions |
| `maps` | `true` \| `false` | `false` | Map view shows the map or "not connected" |
| `listRows` | number | `3` | Rows on the hazardous materials list, for checking a short and a long list page |
| `aiProviders` | `configured` \| `many` \| `none` | `configured` | AI control: the prototype's three providers (one failing), five cloud providers, or none |
| `aiPaused` | `true` \| `false` | `false` | Every agent paused in shadow |

```bash
curl -X POST localhost:8080/__mock/scenario \
  -H 'content-type: application/json' \
  -d '{"operationType":"brokerage","ai":false}'
```

`GET /__mock/scenario` returns the current scenario and its anchor (Unix seconds).
Environment equivalents: `MOCK_BOARD`, `MOCK_AI=0`, `MOCK_OPERATION_TYPE`, `MOCK_HOS=0`,
`MOCK_MAPS=1`, `MOCK_LIST_ROWS`, `MOCK_AI_PROVIDERS`, `MOCK_AI_PAUSED=1`.

The board briefing is stored the way the server stores it: written on the first read of a
scenario, read unchanged after that, and written again (its `generation` going up) only once
everything it flagged is cleared, so it does not move with every write.

Writes are stateful within a scenario: assigning, tendering, deciding or undoing a
suggestion, notifying a customer, approving detention and transferring to billing all change
what the next query returns, so counts move the way they would against the real API.

An operation the mock does not know is logged as `unhandled GraphQL operation <name>` (or
`unhandled GET <path>`) and answered with an empty result; add a handler in
`handlers/graphql.mjs` or `handlers/rest.mjs` when a new one appears.

## Screenshots

With both servers running:

```bash
node e2e/mock-api/screenshots.mjs [outDir] [name,name,...]
```

The script freezes the browser clock at the scenario anchor, walks through every scenario
and interaction in `SHOTS` (dark and light, expanded row, activity tab, quick filters,
capacity popover, carriers tab, timeline, map, panel open, the Filter builder, a quick
filter chip, Lane pinned while scrolled, a collapsed group, the Group by popover and the
board grouped by delivery date, customer and owner, asset without AI, brokerage,
no HOS, quiet board, 1000px and 800px widths) and writes one PNG per entry to `outDir`
(default `e2e/mock-api/screenshots`). Pass a comma-separated list of names to capture only
those. Any page error, console error or failed interaction is printed under its shot and
makes the script exit non-zero.

`CHROMIUM_PATH`, `APP_URL`, `MOCK_API_URL` and `SETTLE_MS` override the defaults.

AI control has its own walkthrough, the same way:

```bash
node e2e/mock-api/ai-control-screenshots.mjs [outDir] [name,name,...]
```

It writes to `e2e/mock-api/screenshots/ai-control` by default and covers every tab in its
scenarios and the panels each one opens. The AI control data is the design handoff's demo set
(`fixtures/aicontrol.mjs`); the provider catalog, extension catalog, agent tool catalog and event
kinds in `fixtures/ai/` are dumped from the Go service, so names and labels are the real ones.
Regenerate them after the registry changes with a throwaway test that marshals
`aiproviderhandler`'s catalog, `agentextensionservice.ListCatalog`,
`agentdefinitionservice.buildToolCatalog` (parameters stripped) and `agent.KnownEvents()`.

## Layout

```
server.mjs            HTTP entry: CORS, scenario endpoint, REST and GraphQL dispatch
state.mjs             Scenario defaults, stage mapping and the mutable board state
fixtures/board.mjs    Shipments, drivers and carriers ported from the design prototype
fixtures/session.mjs  Signed-in user and permission manifest
fixtures/aicontrol.mjs  Agents, providers, runs, usage and tune-ups for AI control
fixtures/ai/          Catalogs dumped from the Go service
handlers/rest.mjs     REST routes the shell and board call
handlers/graphql.mjs  GraphQL operations, keyed by operation name
handlers/aicontrol.mjs  AI control's GraphQL operations and REST routes
screenshots.mjs       Playwright walkthrough of every board scenario
ai-control-screenshots.mjs  Playwright walkthrough of AI control
```

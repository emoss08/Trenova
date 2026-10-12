# Mock API

A dependency-free Node server that stands in for the TMS API, so the shipments board
(`/shipment-management/shipments`), AI control (`/admin/agent-control`) and Triage (`/triage`, the
redesign of `/inbox`) can be run, clicked through and screenshotted without Postgres, Redis,
Temporal or an AI provider. It answers the REST session endpoints the app shell needs, every
GraphQL operation those pages send, and the realtime stream, from a deterministic fixture.

## Run it

```bash
# terminal 1 — the mock API on :8080 (the dev client's API origin)
node e2e/mock-api/server.mjs

# terminal 2 — the web app on :5173
pnpm --filter @trenova/web dev
```

Open <http://localhost:5173/shipment-management/shipments> (or `/triage`). You are signed in as
the fixture user, System Administrator, with every permission.

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
| `triage` | `live` \| `firstDay` \| `noMailbox` \| `deskOff` | `live` | Triage's queue: the prototype's mail with a new message every 26s, no mail yet, no source, or Nova off |
| `triageTime` | `afternoon` \| `endOfDay` | `afternoon` | Triage's clock: 13:39 (the prototype's NOW) or 17:20, after the 17:00 wrap-up |
| `triageReplyAfterMs` | number | `6500` | How long after "Wait for reply" the sender answers on the thread |
| `desk` | `full` \| `empty` \| `hold` | `full` | Desk resource cards: the fixture, every section empty, or every card request left open (the skeleton) |
| `deskSlug` | an artifact slug | `tractor-t-101` | The card the Desk cards conversation (`thr_desk_cards`) holds |

```bash
curl -X POST localhost:8080/__mock/scenario \
  -H 'content-type: application/json' \
  -d '{"operationType":"brokerage","ai":false}'
```

`GET /__mock/scenario` returns the current scenario, the board's anchor (`anchor`) and Triage's
(`triageAnchor`), both Unix seconds. A new scenario sends `reset` down every open realtime stream,
so an open page refetches. Environment equivalents: `MOCK_BOARD`, `MOCK_AI=0`,
`MOCK_OPERATION_TYPE`, `MOCK_HOS=0`, `MOCK_MAPS=1`, `MOCK_LIST_ROWS`, `MOCK_AI_PROVIDERS`,
`MOCK_AI_PAUSED=1`, `MOCK_DESK`, `MOCK_DESK_SLUG`, `MOCK_TRIAGE`, `MOCK_TRIAGE_TIME`,
`MOCK_TRIAGE_REPLY_MS`, plus `MOCK_TRIAGE_LIVE_MS` (the live arrival interval, default 26000),
`MOCK_API_PORT` (default 8080) and `MOCK_API_PUBLIC_URL` (the origin avatar URLs are built on).

The board briefing is stored the way the server stores it: written on the first read of a
scenario, read unchanged after that, and written again (its `generation` going up) only once
everything it flagged is cleared, so it does not move with every write.

Writes are stateful within a scenario: assigning, tendering, deciding or undoing a
suggestion, notifying a customer, approving detention and transferring to billing all change
what the next query returns, so counts move the way they would against the real API.

## Triage

Triage's data is the design handoff's demo set (`design_handoff_triage/design/`), ported to the
server's shapes in `fixtures/triage.mjs`: the three sources (Claims quiet for seven days), the
twelve messages with their bodies, the facts Nova underlined, attachments and what was read from
them, the two conflicts, the detention dispute's stop times, the four rules, templates, saved
views, the live pool and Nova's drafts. Around them sit the mail the briefing counts: 29 messages
Nova settled today across 6am to now (14 PODs filed, 6 invoices queued, 9 newsletters ignored), 6
closed by hand today, and the 11 Blue Ridge rate cons filed by hand this month that make the
"11 of 11" rule suggestion. The signed-in user is "System Administrator"; Dana Whitfield, Marcus
Lee (on call) and Priya Shah are the rest of the team, with avatars served from
`/__mock/avatar/<id>.svg`.

Nothing in the briefing is stored: `handlers/triage-model.mjs` works it out from the queue on every
read, so it moves as you act. Each message's `actions` come from a line-for-line port of
`triage.Actions` and `triage.GroupFor` (`services/tms/internal/modules/documents/domain/triage/actions.go`),
and the deck, the river, lanes, counts, Due/Newest order, `collapseThreads`, search and cursors
follow the server's definitions (`docs/triage-redesign-notes.md`, "Decisions"). Writes keep the
server's rules (only Actioned and Ignored are a person's to choose, an event snooze needs a
shipment, a version mismatch is refused, and so on), answer refusals as the server's error
presenter does (`extensions.code`, `type`, field `errors`), append an activity line, and publish a
`resource.invalidation` for `inbound_message` (or `inbound_mailbox`). Nova's drafts take a moment,
as they do for real: a reply draft about 1.3s, a load draft 1.2s, a send 0.6s.

Things worth knowing when clicking through:

- **Replies.** `edi@partnercarrier.example` refuses replies with a 550 and suggests
  `ops@partnercarrier.example`; a resend there goes. "Wait for reply" parks the message and the
  sender answers on its thread after `triageReplyAfterMs`.
- **Stops.** Deciding the Harborline dispute records the decision and has Nova draft the matching
  reply (waive the $185, or stand by it with `ELD_SEED-PAY-002_Oct3.pdf` attached).
- **Attachments.** `InboundReplyAttachables` lists the message's readable files, the shipments'
  BOL, POD, rate con and invoice (plus the ELD record and gate log on SEED-PAY-002) and a
  document on Harborline and Crestline, against the server's 7,852,032-byte budget. A reply or a
  draft save takes `documentIds` (unknown ids, more than 20 files or more than the budget are
  refused as the server refuses them), and the `Replied` line names the files.
  `POST /api/v1/documents/upload/` stores a multipart upload as a document on the
  `resourceType`/`resourceId` it names, and it becomes attachable.
- **Loads.** Creating a load from a draft makes `S-104260`, then `S-104261`, …
- **Sources.** "Send a test email" on Claims lands a test message after about 1.5s and the source
  turns healthy. Creating a source when nothing has arrived (`firstDay`, `noMailbox`) reads the
  backlog: 57 messages arrive over about 3s, each `Processing` for 0.7s, so `processing` and the
  per-kind totals tick up.
- **Nova off.** In `deskOff` the waiting messages are unread (no kind, match, facts or actions);
  `readUnreadInboundMessages` turns Nova on and reads them back after about 1.6s.
- **Live.** In `live` the prototype's three pool messages arrive 26s apart, each announced on the
  stream, and Marcus opens the newest for a few seconds.

**Realtime.** `GET /api/v1/realtime/stream/` is framed as `docs/engineering/realtime.md` "A
stream" describes: `ready`, a `presence.snapshot` with `?presence=users`, `resource.invalidation`,
`presence`, `typing` and a `heartbeat` every 15s. `POST /api/v1/inbox/presence/` with
`{connectionId, messageId, activity}` joins (or updates) the `inbox:messages` scope and returns its
snapshot — Dana viewing the Harborline dispute and Priya writing on the Crestline status request —
and announces the join as a `presence` event; `DELETE …?connectionId=` leaves, and
`POST /api/v1/inbox/typing/` marks the member writing. `POST /api/v1/assistant/threads/` opens a
Desk thread for a hand-off, and `MyAgents` lists Dispatch, Billing and Customer service.
`SelectOptions` answers document types, shipments, customers, carriers and users from the same
fixture.

**Checking the contract.** With the mock running:

```bash
node e2e/mock-api/check-triage.mjs     # MOCK_API_URL to point it elsewhere
```

It loads `inbox.graphql` and `triage.graphql` with the `graphql` package, sends every operation in
them with sample variables (and the refusals a write should make), and holds each response to its
selection set against `schema.graphql`: every selected field present, nothing unselected besides
`__typename`, no null on a non-null type, enums and scalars of the right kind. It then walks the
other scenarios and the realtime stream, and fails if any operation in the two files was never sent.

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

Triage has its own walkthrough:

```bash
node e2e/mock-api/triage-screenshots.mjs [outDir] [name,name,...]
```

It writes to `e2e/mock-api/screenshots/triage` by default, freezes the clock at `triageAnchor` in
UTC, and covers every state the prototype's Tweaks panel offers (live, first day, no mailbox and
desk off; afternoon and end of day) on desktop (1440×900) and phone (390×844), dark and light,
then the main interactions: the briefing, a folder with its first message open, the row context
menu, the sender card, the kind correction menu, the reply composer with templates, the 550
refusal and the resend, the load form, filing with split pages, the stops check, both conflict
banners, the rules page with its add row filled in and previewed, a rule's matches, the document
viewer, shipment and customer peeks, the Desk hand-off, connecting a source, the backlog being
read, the shortcuts sheet, the bulk selection bar, the snoozed and waiting-on-them lanes, and the
end-of-day carry-over and hand-off. Steps find controls by role and by the prototype's words
("File all", "Turn on", "Hand off to Marcus"), so they hold as the markup changes; a step that
cannot be done is printed under its shot and the rest carry on. `STEP_MS` (default 4000) bounds
each step's wait. With `MOCK_API_URL` set to a mock on another port, the browser's calls to
`localhost:8080` are sent there too, so a fresh mock can be used while another one runs.

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

The Desk's resource cards have their own walkthrough:

```bash
node e2e/mock-api/desk-cards-screenshots.mjs [outDir] [name,name,...]
```

It opens each card where people see it, beside the conversation that looked the record up
(`/desk/c/thr_desk_cards/a/<slug>`), and on its own page at full length (`…/page`), and shoots
it loaded in light and dark, with something wrong on it, empty, as its skeleton, unreadable and
in a 1000px window. It writes to `e2e/mock-api/screenshots/desk-cards` by default. The records
are in `fixtures/desk-cards.mjs`: each card has a conversation for the record loaded, one with
something to flag, one with nothing on file and one that is gone. A new card adds its records
there, its operations to `handlers/deskcards.mjs`, and an entry to `CARDS` in the walkthrough.

Settings has its own walkthrough, the same way:

```bash
node e2e/mock-api/settings-screenshots.mjs [outDir] [name,name,...]
```

It opens the fullscreen Settings dialog over the shipments board with ⌘⇧S and captures each
section in light and dark, the Away sub-rows, an expanded notification category, the Saved ·
Undo toast, a pause, the inline password change, ⌘K search and its jump, an 800px width and the
page left as it was after Escape. Writes are stateful: the profile, availability, notification
preferences and sessions the dialog saves are what it reads back. The password change accepts
`correct-horse` as the current password. It writes to `e2e/mock-api/screenshots/settings` by
default.

## Layout

```
server.mjs            HTTP entry: CORS, scenario endpoint, REST and GraphQL dispatch
state.mjs             Scenario defaults, stage mapping, and the mutable board and Triage state
realtime.mjs          The realtime stream: ready, heartbeats, invalidations and presence
fixtures/board.mjs    Shipments, drivers and carriers ported from the design prototype
fixtures/session.mjs  Signed-in user and permission manifest
fixtures/aicontrol.mjs  Agents, providers, runs, usage and tune-ups for AI control
fixtures/desk-cards.mjs  The records the Desk's resource cards draw, and their conversations
fixtures/settings.mjs   The signed-in person's settings, sessions, sign-ins and teammates
fixtures/triage.mjs   Triage's sources, messages, rules, templates and team, from the prototype
fixtures/ai/          Catalogs dumped from the Go service
handlers/rest.mjs     REST routes the shell and board call
handlers/graphql.mjs  GraphQL operations, keyed by operation name
handlers/aicontrol.mjs  AI control's GraphQL operations and REST routes
handlers/deskcards.mjs  The Desk cards conversation and each card's GraphQL operations
handlers/settings.mjs   Settings' REST routes and the user select options
handlers/triage-model.mjs  Triage's rules: actions, lanes, counts, briefing, previews and every write
handlers/triage.mjs   Triage's GraphQL operations shaped to their selections, presence and Desk routes
screenshots.mjs       Playwright walkthrough of every board scenario
ai-control-screenshots.mjs  Playwright walkthrough of AI control
desk-cards-screenshots.mjs  Playwright walkthrough of the Desk's resource cards
settings-screenshots.mjs    Playwright walkthrough of Settings
triage-screenshots.mjs  Playwright walkthrough of Triage
check-triage.mjs      Sends every Triage operation and checks each response against its selection
```

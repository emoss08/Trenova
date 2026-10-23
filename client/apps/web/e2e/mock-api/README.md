# Mock API

A dependency-free Node server that stands in for the TMS API, so the shipments board
(`/shipment-management/shipments`, with each shipment's Customer tracking tab), the customer
tracking page (`/shipment-tracking/<token>`), AI control (`/admin/agent-control`), Triage (`/triage`, the
redesign of `/inbox`), the Desk (`/desk`), the Formula Studio
(`/billing/configuration-files/formula-templates/new`, `…/<id>/edit`) and the formula libraries can be run, clicked through and screenshotted without Postgres, Redis,
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

| Field                | Values                                           | Default         | Effect                                                                                                                                                                                                                                                                                                                                                                                                            |
| -------------------- | ------------------------------------------------ | --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `board`              | `quiet` \| `busy` \| `high`                      | `high`          | 2 sample loads, the 14-load prototype board, or 480 shipments                                                                                                                                                                                                                                                                                                                                                     |
| `ai`                 | `true` \| `false`                                | `true`          | Briefing sentence, AI wording in the brief                                                                                                                                                                                                                                                                                                                                                                        |
| `operationType`      | `asset` \| `brokerage` \| `both`                 | `both`          | Drivers, carriers or both in the strip and queue                                                                                                                                                                                                                                                                                                                                                                  |
| `hos`                | `true` \| `false`                                | `true`          | HOS rings, sublines and HOS suggestions                                                                                                                                                                                                                                                                                                                                                                           |
| `maps`               | `true` \| `false`                                | `false`         | Map view shows the map or "not connected"                                                                                                                                                                                                                                                                                                                                                                         |
| `listRows`           | number                                           | `3`             | Rows on the hazardous materials list, for checking a short and a long list page                                                                                                                                                                                                                                                                                                                                   |
| `aiProviders`        | `configured` \| `many` \| `all` \| `none`        | `configured`    | AI control: the prototype's three providers (one failing), five cloud providers, both together, or none                                                                                                                                                                                                                                                                                                           |
| `aiPaused`           | `true` \| `false`                                | `false`         | Every agent paused in shadow                                                                                                                                                                                                                                                                                                                                                                                      |
| `aiAnalytics`        | `ok` \| `error` \| `hold`                        | `ok`            | AI control › Analytics: the query answers, every read fails (the error state), or every read is left open (the skeleton)                                                                                                                                                                                                                                                                                          |
| `triage`             | `live` \| `firstDay` \| `noMailbox` \| `deskOff` | `live`          | Triage's queue: the prototype's mail with a new message every 26s, no mail yet, no source, or Nova off                                                                                                                                                                                                                                                                                                            |
| `triageTime`         | `afternoon` \| `endOfDay`                        | `afternoon`     | Triage's clock: 13:39 (the prototype's NOW) or 17:20, after the 17:00 wrap-up                                                                                                                                                                                                                                                                                                                                     |
| `triageReplyAfterMs` | number                                           | `6500`          | How long after "Wait for reply" the sender answers on the thread                                                                                                                                                                                                                                                                                                                                                  |
| `desk`               | `full` \| `empty` \| `hold`                      | `full`          | Desk resource cards: the fixture, every section empty, or every card request left open (the skeleton)                                                                                                                                                                                                                                                                                                             |
| `deskSlug`           | an artifact slug                                 | `tractor-t-101` | The card the Desk cards conversation (`thr_desk_cards`) holds                                                                                                                                                                                                                                                                                                                                                     |
| `deskReply`          | `idle` \| `working`                              | `idle`          | The Desk's live conversation (`athr_desk_live`): nothing under way, or a reply under way that has been steered once, with two follow-ups queued                                                                                                                                                                                                                                                                   |
| `agentMatch`         | `match` \| `none` \| `nobody` \| `hold`          | `match`         | The Desk's agent suggestion (`MyAgentMatch`, `handlers/agentrecommendation.mjs`): the agent whose tools fit the question, no tool matching, tools matching that none of the person's agents holds, or the request left open (the "finding the agent" state). Suggestions the page records (`RecordMyAgentSuggestion`, `SettleMyAgentSuggestion`) are kept and counted in AI analytics with a month of seeded ones |

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
`MOCK_TRIAGE_REPLY_MS`, `MOCK_PALETTE=1`, plus `MOCK_TRIAGE_LIVE_MS` (the live arrival interval, default 26000),
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

Nothing in the briefing is stored: `handlers/palette.mjs   Global search, documents, pins, Nova's ask stream and the palette's previews
handlers/triage-model.mjs` works it out from the queue on every
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

## Customer tracking

`handlers/customertracking.mjs` serves the customer's tracking page and the shipment panel's
Customer tracking tab. `GET /api/v1/shipment-tracking-links/<token>/` answers one view per
token, timed against the real clock: `in-transit`, `delayed` (a weather hold and a late
delivery), `delivered`, `pod-ready` (delivered with the POD on file), `planned`, `canceled` and
`minimal` (a customer shown the route alone); any other token is a dead link.
`GET /api/v1/shipments/<id>/customer-tracking/preview/` is the in-transit view with equipment,
freight and holds hidden, as the fixture's settings say. The tab's queries keep links per
shipment for the scenario, so issuing and revoking change what the next read returns.

```bash
node e2e/mock-api/customer-tracking-screenshots.mjs [outDir] [name,name,...]
```

## Formula Studio and formula libraries

The Formula Studio (`/billing/configuration-files/formula-templates/new` and `…/<id>/edit`) and
the formula libraries (`/billing/configuration-files/formula-libraries`, `…/new`, `…/<id>/edit`)
run on Tariff. `rules.mjs` loads the client's own `packages/shared/src/rules/rules.wasm` in Node,
so every check, test run, bill, trace and scaffolded test the mock serves is the rules module's
answer, and the Studio's tests panel, the server's readiness, the backtest and Nova agree.
`SAMPLE_RECORD` is the module's sample record, read back through a trace; a shipment's record is
that with the shipment's distance, weight, flags, lane and stops over it (`recordFrom`,
`shipmentOverlay`). Every fixture source checks under the module, and the fixture's sources are
in `fixtures/formula.mjs`.

The organisation's context (`GET /formula-templates/context`) holds the approved version of each
library that is not archived (a library is left out of its own), the accessorial codes `FSC`,
`STOP`, `HAZ`, `WKND` and `DET` (the linehaul is uncoded, so it is freight), and the distance unit.

Three saved copies of "Regional per mile + fuel", each importing `org.accessorials`:

| Id                  | Status        | What it shows |
| ------------------- | ------------- | ------------- |
| `fmt_studio_draft`  | Draft, v4     | v3 is the approved version (fuel 14%); v4 raises fuel to 18%, but its first test still expects the 14% total, so it fails, the outline shows a red dot and Submit is refused until the test is fixed. Usage: 6 rate agreements, 2 rules, 214 shipments |
| `fmt_studio_review` | In review, v3 | Submitted by Dana Whitfield two days ago with the signed-in person as the requested reviewer and a note; v2 (fuel 16%) is production, so the impact and the review diff have something to show. Every test passes, so it can be approved |
| `fmt_studio_active` | Active, v2    | Approved by Priya Shah |

Three libraries:

| Id                       | Path               | Status        | What it shows |
| ------------------------ | ------------------ | ------------- | ------------- |
| `fl_studio_accessorials` | `org.accessorials` | Active, v3    | `extraStops`, `hazmat` and `HAZMAT_FEE`, imported by every template, with two review rounds |
| `fl_studio_fuel`         | `org.fuel`         | In review, v3 | the peg moved to $1.25, submitted to the signed-in person; v2 is approved |
| `fl_studio_zones`        | `org.zones`        | Draft, v1     | a test that fails (`786` is zone 3, not 2), so Submit is refused |

A created template is `fmt_studio_new_<n>` and a created library `fl_studio_new_<n>`. The
reviewers are Priya Shah, Marcus Lee and Dana Whitfield (Triage's team). The real-shipment picker
(`SelectOptions` with `SHIPMENT`) lists seven loads first (`shp_fs_240811` … `shp_fs_240719`: a
long haul, a three-stop reefer, a four-stop hazmat, a 38-mile short haul on the minimum, a
weekend five-stop, a reefer and a hazmat), then Triage's. A backtest or impact re-rates those
seven and 57 more from a seeded generator, production (the newest approved version) against the
candidate, in one `tariff.evaluate` call per shipment.

Templates, under `/api/v1/formula-templates`:

- `GET /` a limit/offset page; `POST /` creates a draft at v1; `GET|PUT /:id/` read and save (a
  change to the source or the name takes a new version and returns an active or in-review
  template to draft; a stale `version` is a 409). A template's `inputs`, `imports` (pinned to each
  library's approved version) and `testCount` are derived from its source on every read.
- `GET /context` (`{context, libraries}`: the context and each library's approved version); `POST /test` (the preview: `{source, inputs?, shipmentId?}` → the check's
  findings, the bill or the refusal or the error, the trace, the lookups and the server's run of
  the tests); `POST /scaffold-test` (`tariff.scaffoldTest` on a real shipment).
- `GET /standards`, `POST /install-standards`, `POST /duplicate`, `POST /bulk-update-status`
  (archiving one in use needs `confirmInUse`), `POST /import` (export version `2.0`).
- `GET /:id/usage`, `/versions` (and `/versions/<n>`, `/versions/scheduled`), `POST /:id/versions`,
  `/rollback`, `/fork`, `GET /:id/compare`, `/lineage`.
- `GET /:id/readiness` (keys `details`, `source`, `warnings`, `tests`, `status`, `reviewer`,
  `submissionAge`; a template with no tests only warns), `/reviews`, `/review-diff` (source and
  pinned imports against the approved version), `/reviewers`, `POST /:id/impact`, `/backtest`.
- `POST /:id/submit` (refused with 422 and field errors keyed by check when readiness fails),
  `/approve` (not by the submitter, every check passing), `/reject` and `/request-changes` (a
  comment required).

Libraries: GraphQL `FormulaLibraryTable` and `FormulaLibrary` (the library, its versions and its
reviews in one request), and REST under `/api/v1/formula-libraries`: `POST /`, `PUT /:id/`
(names are camelCase, not a keyword and unique), `POST /duplicate` (`{libraryIds}`),
`/:id/archive` (`confirmInUse` once templates import it), `/:id/rollback`, `/:id/submit`,
`/approve`, `/reject`, `/request-changes`, `GET /:id/readiness` (a library needs a test),
`/review-diff`, `/compare`, `/usages` (each template version that imports it and the library
version it pinned) and `/reviewers`.

**Nova.** The one-shots answer after about 700ms (`MOCK_FORMULA_NOVA_MS`), in the server's
shapes: `POST /ai/explain` (`{preview, items}`: the preview of the sample load or `shipmentId`
with the trace on, and a sentence for each binding, branch, charge, dropped charge and refusal,
by the trace step's index), `/ai/suggest-tests` (`{tests}`: three tests the module scaffolded
from real loads on the tariff's edges, each run against the draft, with its status and
failures), `/ai/describe-inputs` (`{descriptions}`, a line for each input with no doc comment),
`/ai/explain-test` (`{verdict, why, generated, test}` for a named test, from the values its
failing expectation saw) and `/:id/review-note` (the change since the approved version, a
backtest against production and the tests). With `aiProviders` at `none` they refuse with 422.

`POST /formula-templates/ai/thread/` and `POST /formula-libraries/ai/thread/` (`{libraryId}`) open
the page conversation (`athr_formula_<n>`, origin `Formula`, subject `FormulaTemplate` or
`FormulaLibrary`, so the Desk rail leaves it out): the same one for a saved record every time, a
fresh one for an unsaved one. A turn streams `accepted`, a `reasoning` line, the `formula_check`
and `formula_run_tests` steps and the reply, and is written to the conversation. What it does
follows the question and the editor in the turn's page draft (`formula` or `formulaLibrary`).
As on the server, a change is offered only when it checks clean and every one of its tests (at
least one) passes: it is a `draft_edit` artifact (`{surface: "formula", action:
"formula_propose", formula}`, or `formula_library`, `formula_library_propose` and
`formulaLibrary`) carrying a `SourceProposal` (`name`, `description`, `source`, `explanation`,
`inputs`, `imports`, `charges`, `warnings`, `tests: {total, passed}`); otherwise the reply says
what stops it and there is no artifact. On `fmt_studio_draft` and `fl_studio_zones`, whose tests
fail, every change is refused that way.

- an empty editor: a tariff drafted from the description (a rate per mile, a fuel percent and a
  minimum read from the question), or a library of shared fees;
- "fix": the checker's first fix applied;
- "weekend": a weekend pickup charge on `std.time`'s `weekend`, imported;
- "minimum" with an amount: the `minimum` clause set, or the default of the input it reads, and
  any test that pins the old minimum as the total moved to the new one;
- "test", "edge": three scaffolded tests appended; a library gets a test per function that takes
  one plain value, at the literals of that type the file uses, pinned to what the module says it
  returns;
- "explain": the inputs, charges and functions in a sentence, no artifact; anything else, what
  Nova can do.

The Studio applies every offered change to its editor the moment the artifact streams; the
card under the reply undoes it.

`formula-studio-screenshots.mjs` shoots both Studios in light and dark: the list, each template's
state, the editor's problems, completion, hover, signature help, rename and definition, the
preview on a real shipment and "Save as test", the tests panel failing and fixed, Nova's
suggested tests and its reading of a failing test, the reference, formatting, Nova's proposals
(on the active template) and a refused one (on the draft), the review dialogs, the backtest, the
narrow layouts, and every library page and pane.

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

The Analytics tab (`?tab=analytics`, shots named `analytics-*`) reads `agentAnalytics` and
`agentAnalyticsToolFailures` from `handlers/aianalytics.mjs`, which ports the Go service's rules
(grouping by model, provider or deployment, "few" under 30 runs or 20 tool calls, suggested
routes, the Nova sentences, health events and colour slots) over the month of work units in
`fixtures/aianalytics.mjs`. The units are deterministic: the workstation's vLLM limped and
recovered 12 days ago and stopped answering this morning, AWS Bedrock lost access a week ago,
Google Gemini was turned off six days ago, and Llama 3.3 runs on both Groq and the workstation
so the same-model card has a pair to compare. Use `aiProviders: "all"` for the analytics shots:
it connects every provider the units name. `routeAITask` changes the mock's routing, so a
suggestion disappears once it is taken. Trend buckets start at the viewer's local boundaries, as
the server's do, so a shot with a `timezoneId` (`analytics-new-york-week`) checks the chart
outside UTC.

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
A card read over REST answers its routes from the same handler; the EDI message card's X12,
built segment by segment with its inspection, is in `fixtures/desk-edi.mjs`, and a REST
fixture holds its skeleton with `{ __hold: true }` as a GraphQL one does.
The worker and carrier settlement cards read `fixtures/desk-settlements.mjs`; the loads a
settlement pays come from the board, and each one's share is filled in from them so the card's
loads carry what the settlement pays.
The detention card reads `fixtures/desk-detention.mjs` (a charge waiting on approval, a running
clock with its notice overdue, a stop that left inside its free time, a disputed and a waived
charge), each stop given a load from the board so the card's shipment and its pro number agree;
a charge the fixture does not hold answers not found.
The EDI partner card reads `fixtures/desk-edi-partners.mjs` (a partner trading cleanly, one whose
pickup fails with a dead letter, overdue acknowledgments and a file set aside, one still being set
up, and a Trenova-to-Trenova partner), its times moved to the scenario's anchor; testing Acme's line
fails at sign-in, picking up stages one file, and a partner the fixture does not hold answers not
found.
The journal entry and manual journal cards read `fixtures/desk-journals.mjs`, the same fixtures the
cards' unit tests read, moved by whole days so each booking day stays the first second of a UTC day
(an invoice's entry posted, a worker settlement's entry waiting on approval, an entry reversed and
its reversal, a posted write-off; a journal waiting on approval, a draft out of balance asked for in
a closed month, a posted one and a rejected one). Approving or posting an entry, each step of a
manual journal and asking for a reversal answer as the server does, and a record the fixture does
not hold answers not found.

The Desk home suggesting the agent with the tools for a question has its own walkthrough:

```bash
node e2e/mock-api/desk-recommendation-screenshots.mjs [outDir] [name,name,...]
```

It types a customer ETA question while Billing is chosen and shoots the suggestion of Customer
service, keeping Billing, switching, the "finding the agent" state of a question sent before
its match came back (`agentMatch: hold`) and no match (`agentMatch: none`), in light and dark
and the suggestion at phone width. It writes to `e2e/mock-api/screenshots/desk-recommendation`
by default.

A Desk case and a reply under way have their own walkthrough:

```bash
node e2e/mock-api/desk-case-screenshots.mjs [outDir] [name,name,...]
```

It opens two conversations from `handlers/deskcase.mjs`. The first, `/desk/t/athr_desk_case`, is a
case about a delivered shipment: its card with the ready-to-bill checklist (one step a person
ticks, one handed to another agent), the case menu, the parties it can wait on, the agent's two
open waits and the organization's case checklist settings (`?settings=checklists`). The second,
`/desk/t/athr_desk_live`, has a reply under way when `deskReply` is `working`: its stream sends
the turn from its start, including every steer so far, and is then held open, so a steer sent
from the page reaches it the way it reaches a live turn. The walkthrough shoots each in light and
dark, then snoozes the case, waits on the customer, ticks the added step and steers the reply,
starting each shot from a fresh scenario. It writes to `e2e/mock-api/screenshots/desk-case` by
default.

Writes keep the server's rules (`assistantcaseservice`, `agentwaitservice`,
`assistantqueueservice`): a snooze is at least a minute and at most 30 days ahead, the delivered
shipment has no stop ahead to snooze to, only a step the organization added for a person can be
ticked, a reply wait is on the record's own customer or carrier and gives up after 72 hours unless
asked otherwise (at most 168), a conversation holds at most ten open waits and twenty queued
messages, an edit at a stale version is refused, and a reorder must name every waiting message
once. Refusals carry the server's problem details (`type`, `title`, `status`, `detail`, field
`errors`). A message sent while nothing is under way is answered as a new turn. The rail's list
(`GET /assistant/threads/`) and shelf counts (`/threads/shelves/`) are read from every
conversation the mock serves, ordered and banded by when each was last touched, as
`conversationrepository` does.

An EDI connection drafted from a trading partner's implementation guide has its own walkthrough:

```bash
node e2e/mock-api/desk-edi-draft-screenshots.mjs [outDir] [name,name,...]
```

It opens `/desk/t/athr_desk_edi_guide` (`handlers/deskedidraft.mjs`), a conversation with the EDI
agent about Acme Retail Group's guide (the deskbench guide, `edi-guide.html`), at the stage the
`ediDraft` scenario field names: `held` (the first draft, its AS2 endpoint held between the guide's
production and test URLs), `ready` (drafted again with the person's answer), `proposed` (the save
waiting on its approval, with `AgentProposalPreview` answered) or `revised` (version 4.0 of the
guide, a change set against the saved partner). The drafts are in `fixtures/desk-edi-draft.mjs`,
shaped as `connectionDraftArtifact` stores them. It shoots each stage in light and dark, a value's
page quote on hover, the document types, the approval and the revised control numbers, and writes
to `e2e/mock-api/screenshots/desk-edi-draft` by default.

A plan, several of an agent's writes decided as one, has its own walkthrough:

```bash
node e2e/mock-api/desk-plan-screenshots.mjs [outDir] [name,name,...]
```

It opens `/desk/c/athr_desk_plan/a/recover-s-240201` (`handlers/deskplan.mjs`), the Dispatch
agent's four steps to recover S-240201 after the receiver's dock turned it away, at the stage
the `deskPlan` scenario field names: `awaiting`, `held` (a shadow switch), `approving` (inside
its undo window), `running`, `failed` (step 3 refused, step 4 skipped), `done`, `declined`,
`expired` or `superseded` (failed, with a later plan in the conversation). The thread's plans,
proposals and `MyPlanPreview` are answered from `fixtures/desk-plan.mjs`; `DecideMyPlan` and
`UndoMyDecision` move the stage on, and an approval with a digest other than the preview's is
refused as a conflict. It shoots every stage on the artifact's page in light and dark, beside
the conversation, in a narrow window, a folded step opened, the decline dialog and the skeleton,
and writes to `e2e/mock-api/screenshots/desk-plan` by default.

A table that pages, a list the agent read in part and the person pages through with their own
access, has its own walkthrough:

```bash
node e2e/mock-api/desk-table-screenshots.mjs [outDir] [name,name,...]
```

It opens `/desk/c/athr_desk_table/a/acme-shipments-in-transit` (`handlers/desktable.mjs`): Acme
Foods' 128 shipments in transit, of which the conversation's agent, Dispatch Desk, read two pages
of 25. The rows endpoint (`…/artifacts/:id/rows/`) pages, searches, sorts and counts them as the
server does, as the `deskTable` scenario field says: `paged`, `unnamed` (paged, but the table does
not say which agent read it), `slow` (a page past the agent's is left open, so its rows
shimmer), `failing` (pages past the first one read fail), `locked` (403: the person may no longer
read shipments), `withheld` (revenue withheld at their access), `cut` (cut to fit, with no place
to go on from), `whole` (30 rows, all held) or `short` (12 rows). It shoots each on the
artifact's page, after paging, searching and sorting where the state needs it, beside the
conversation and in a narrow window, and writes to `e2e/mock-api/screenshots/desk-table` by
default.

The Desk's extraction artifact, what document intelligence read off a file, has its own
walkthrough:

```bash
node e2e/mock-api/desk-extraction-screenshots.mjs [outDir] [name,name,...]
```

It opens `/desk/t/athr_desk_extract` (`handlers/deskextraction.mjs`), a conversation about a file
the person attached, at the card (`/desk/c/athr_desk_extract/a/northline-rate-confirmation`) and
its own page (`…/page`). The `extraction` scenario field picks the file and how it was read:
`ratecon` (a two-page rate confirmation, the second page a sideways scan on a grey bed), `image`
(a photographed bill of lading), `legacy` (a payload saved before reasons, stop parts and page
sizes were kept), `nobox` (no layout lines), `broken` (every page fails to draw) or `long` (many
fields, long values, none placed). `extractionStage` picks where the shipment stands: `idle`,
`replying`, `proposed`, `created`, `failed`, `rejected` or `attached`. The pages are drawn as SVGs
from the same text lines the payload's boxes are built from (`fixtures/desk-extraction.mjs`, a
port of `assistantservice/extraction.go`), and the walkthrough measures every drawn box against
the drawn page (inside the sheet's padding) and fails a shot whose worst box is more than 1.5px
off. Stop fields carry their `place` as the server's do, so the head reads the lane. Values are
found by their button names (`Bill to: Northline Logistics, needs a look, …`), the check button by
`Ask the agent to check {n}`. It shoots the pane, a
field on page 2, a hovered box, a zoom step, a resize, the page, and the phone, in light and dark,
then each file, the skeleton, the stored preview, drafting, checking and every stage, and writes
to `e2e/mock-api/screenshots/desk-extraction` by default.

Settings has its own walkthrough, the same way:

```bash
node e2e/mock-api/settings-screenshots.mjs [outDir] [name,name,...]
```

It opens the Settings page from the shipments board with ⌘⇧S and captures each section in
light and dark, the Away sub-rows, an expanded notification category, the Saved · Undo toast, a
pause, the inline password change, ⌘K search and its jump, an 800px width, the board again after
going back, and an old email link (`/?settings=notifications`) landing on Notifications. Writes
are stateful: the profile, availability, notification preferences and sessions the page saves
are what it reads back. The password change accepts `correct-horse` as the current password. It
writes to `e2e/mock-api/screenshots/settings` by default.

The command palette has its own walkthrough. Start the mock with `MOCK_PALETTE=1`, which also
serves the palette's records, the counts under "Needs you" and Nova's streamed answer:

```bash
MOCK_PALETTE=1 node e2e/mock-api/server.mjs
node e2e/mock-api/palette-screenshots.mjs [outDir] [name,name,...]
```

It seeds "Jump back in" in local storage, then captures Home in light and dark, the matcher on
`ai`, `acme`, `leon`, a BOL and `bq`, an `@doc` mention, `>theme` command mode, a scope browsed
with nothing typed, a page's preview, the actions menu, a copy confirmed in the footer, asking
about a record, Nova working, answering and previewing a cited record on hover, the empty state,
and the 860px and 700px layouts. Its records live in `fixtures/palette.mjs` and its routes and
operations in `handlers/palette.mjs`. It writes to `e2e/mock-api/screenshots/palette` by default.

## Layout

```
server.mjs            HTTP entry: CORS, scenario endpoint, REST and GraphQL dispatch
state.mjs             Scenario defaults, stage mapping, and the mutable board and Triage state
realtime.mjs          The realtime stream: ready, heartbeats, invalidations and presence
fixtures/board.mjs    Shipments, drivers and carriers ported from the design prototype
fixtures/session.mjs  Signed-in user and permission manifest
fixtures/aicontrol.mjs  Agents, providers, runs, usage and tune-ups for AI control
fixtures/aianalytics.mjs  A month of work units, tool calls and provider hours for AI control's Analytics
fixtures/desk-cards.mjs  The records the Desk's resource cards draw, and their conversations
fixtures/desk-edi-draft.mjs  The EDI connection drafts from Acme Retail Group's guide, version 3.2 and 4.0
fixtures/desk-edi.mjs    The EDI messages the EDI message card draws: X12, inspections, conversations
fixtures/settings.mjs   The signed-in person's settings, sessions, sign-ins and teammates
fixtures/palette.mjs  The command palette's customers, workers, shipments, documents and pins
fixtures/triage.mjs   Triage's sources, messages, rules, templates and team, from the prototype
fixtures/formula.mjs  Tariff sources for the templates, libraries and standards, the shipments and the team
rules.mjs             The client's rules.wasm in Node: every check, test, bill and scaffold
fixtures/ai/          Catalogs dumped from the Go service
handlers/rest.mjs     REST routes the shell and board call
handlers/graphql.mjs  GraphQL operations, keyed by operation name
handlers/aicontrol.mjs  AI control's GraphQL operations and REST routes
handlers/aianalytics.mjs  AI control's Analytics query and tool-failure sheet, ported from the Go service
handlers/deskcards.mjs  The Desk cards conversation and each card's GraphQL operations
handlers/deskcase.mjs   A Desk case and a reply under way: case, waits, queue, turns and checklist settings
handlers/deskedidraft.mjs  An EDI connection drafted from a partner's guide, at each stage, and its approval
handlers/deskthreads.mjs  The Desk rail: the conversation list and its shelf counts
handlers/settings.mjs   Settings' REST routes and the user select options
handlers/palette.mjs   Global search, documents, pins, Nova's ask stream and the palette's previews
handlers/triage-model.mjs  Triage's rules: actions, lanes, counts, briefing, previews and every write
handlers/triage.mjs   Triage's GraphQL operations shaped to their selections, presence and Desk routes
handlers/formula.mjs  The Formula Studio's templates, context, preview, review, backtest and GraphQL
handlers/formula-library.mjs  Formula libraries: GraphQL reads, REST writes, review, usages
handlers/formula-nova.mjs  Nova on both Studios: the one-shots and the page conversation's turns
screenshots.mjs       Playwright walkthrough of every board scenario
ai-control-screenshots.mjs  Playwright walkthrough of AI control
desk-cards-screenshots.mjs  Playwright walkthrough of the Desk's resource cards
desk-case-screenshots.mjs   Playwright walkthrough of a Desk case and a reply under way
desk-edi-draft-screenshots.mjs  Playwright walkthrough of an EDI connection drafted from a guide
settings-screenshots.mjs    Playwright walkthrough of Settings
palette-screenshots.mjs     Playwright walkthrough of the command palette
triage-screenshots.mjs  Playwright walkthrough of Triage
check-triage.mjs      Sends every Triage operation and checks each response against its selection
```

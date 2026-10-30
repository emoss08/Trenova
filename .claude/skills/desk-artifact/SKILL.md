---
name: desk-artifact
description: How to add or redesign a Desk artifact in Trenova, end to end. Use when asked to build, mock up, redesign or extend a Desk resource card (a record drawn in the Desk, under routes/desk/_components/artifacts/cards/<resource>/, such as the invoice, journal entry or fuel card cards) or an artifact kind (kinds/<kind>/, such as the plan or extraction card). Covers the mockup and approval gate, the server read and GraphQL query, the client card, skeleton, registry and record link, actions, tests, mock API fixtures and screenshots, i18n, the product guide, docs, generated snapshots, the gates to run and the PR.
---

# Adding a Desk artifact

A Desk artifact is a record drawn for what a person reads it for. There are two shapes:

- A **resource card** (`artifacts/cards/<resource>/`): one record kind (an invoice, a fuel
  card), opened in place of a table's row or as an artifact when the agent looked the record
  up. It is registered in `cards/registry.tsx`.
- An **artifact kind** (`artifacts/kinds/<kind>/`): what the agent produced (a plan, an
  extraction, an analysis), drawn from its payload plus live reads.

Both are built on the shared record-card frame (`artifacts/record-card/`). The rules live in
[docs/engineering/desk-artifacts.md](../../../docs/engineering/desk-artifacts.md); read it,
including "Adding a card", before writing anything. This skill is the order of work and the
mistakes that have already cost a CI cycle. Copyable shapes are in [recipes.md](recipes.md).
Use the `trenova-frontend` skill for every client file you touch.

**When a rule here is wrong, change the rule** in the same change, and say so in the PR.

## 1. Research what the record is read for

Before you design anything, find out what the server already serves and what the person
reading the record needs from it.

- Find the record's domain type, its repository, its GraphQL type and fragment, its page and
  table, and any existing card or payload.
- List what the card needs that nothing serves yet: aggregates, related records, who did
  what, previews. A card reads **live** data; the payload and the table row are only what the
  agent saw.
- Look at two or three recent cards for the pattern closest to yours. The journal entry, fuel
  card, invoice and detention cards are the most complete.

## 2. Mock it up and wait for approval

Unless the owner has already approved the design, **build nothing in the repo until they do.**

- Draw the mockup as static HTML on the real Desk CSS, in the scratchpad (not the repo). Load
  the app's built styles so the shared `dk-rec-*` classes render as they will for real.
- Give the card one signature part that says what the record *is*: a receipt slip, a card
  face, a ledger, a lifecycle track, a clock ruler. Design it so it isn't just a list of
  fields.
- Draw every state the record can be in (loaded, needs attention, empty or closed,
  cancelled), in light and dark, at full width and at 600px. Screenshot them with Playwright
  from `/opt/pw-browsers` and send them to the owner.
- With the screenshots, list:
  - **Server gaps:** data the design needs that the API doesn't serve yet, each with the file
    that would change.
  - **Decisions:** numbered, each with the options and a recommendation (actions on the card
    or not, what folds, what a new field is called).
- Stop and wait for the reply. "You pick" from the owner means: pick the best experience,
  build it, and say what you picked in the final report.

## 3. Serve it: one query per card

A card reads its record in **one query**, assembled on the server. Never send several
round trips, and never make the client stitch records together.

- Put the assembly in a **card service** in the record's module,
  `internal/modules/<module>/app/<name>cardservice/`. Examples are `journalcardservice`
  and `fueldeskcardservice`.
  - It depends on narrow interfaces it declares itself, not on whole repositories.
  - Aggregates and side reads go through a new **port**,
    `ports/<entity>ports/deskcard.go`, implemented in the module's postgres adapter with
    the generated keel tables, under `dbtx.Read` and `Keel(ctx)`.
  - Register both in the module's `module.go`.
- Put the date logic on the server: quarters, months and "today" are worked out in the
  organization's time zone (`orgzone.Location`). Money is decimals or minor units, never
  floats.
- **Permissions:** the resolver calls `RequirePermission` for the card's own resource. Every
  section that reads another resource is included only when `HasPermission` allows it; leave
  it out, never fail the whole card. Test both ways.
- **GraphQL:**
  - Add the types and one `<entity>DeskCard(id)` query to the module's
    `schema/*.graphqls`, using `Timestamp` and `Decimal`.
  - Bind them to the service structs in `gqlgen.yml`, add the service to
    `resolver/services.go`, and run `go run ./internal/api/graphql/gqlexec/gen`. Never run
    `go tool gqlgen generate`.
  - Load related records per row through a dataloader, never a single-ID getter.
- Write tests at each layer:
  - unit tests for the service with fakes;
  - integration tests for the repository (`go test -tags integration`, `seedtest.SetupTestDB`;
    start Docker with `nohup dockerd` if it isn't running);
  - resolver tests for the permission behaviour.

## 4. Draw it

Lay the card out in its folder as the others do, so a reader finds things where they expect:

```
cards/<resource>/
  desk-<resource>.tsx       DeskX (in place) and DeskXArtifact; reads the query, draws the skeleton
                            while loading and CardUnreadable on error; listed records via
                            useListedRecords; the bar and actions
  <resource>-card.ts        pure helpers: kicker, title, subtitle, figure, standing,
                            checks, facts; every string through t
  <resource>-parts.tsx      the card's own parts (its signature part and its rows)
  <resource>-skeleton.tsx   the card's layout in shimmer (RecordCardSkeleton + the shared parts)
  __tests__/                fixtures.ts, <resource>-card.test.ts, desk-<resource>.test.tsx
```

- **Data:**
  - Put the operation in `client/packages/graphql/src/operations/<area>/desk.graphql`, then run
    `pnpm --filter @trenova/graphql codegen`.
  - Write a wrapper in `apps/web/src/lib/graphql/<area>-desk-card(s).ts` that unmasks
    fragments with `getFragmentData` and takes `{ signal }`.
  - Add a key factory in `apps/web/src/lib/queries/<entity>.ts` (`createQueryKeys`).
- **Shared parts first:**
  - Reach for the existing parts in `record-card/` before drawing new ones: `CardHead`,
    `CardStatus`, `CardFacts`, `CardSection`, `CardChecks`, `CardStats`, `CardFactGrid`,
    `CardRecordRow`, `CardOpenLink`, `CardPath`, `CardTrack`, `CardMeter`, `CardDocuments`,
    `CardBar`, `CardActionButton`, `SettlementStub`, and the formatters in
    `record-card-values.ts`.
  - A part a second card needs moves to `record-card/` (for example `card-fuel.tsx`,
    `card-journal.tsx`).
- **Registry and links:**
  - Register the card in `cards/registry.tsx`: its `resource`, its `Skeleton`, and
    `lazyComponent` for both surfaces.
  - Add the kind to `config/record-links.ts` if it has no link, and check the page really
    opens the record from those params.
- **Listed records:**
  - A record the card names (a driver, a tractor, a purchase) opens in place through
    `useListedRecords` when its kind has a card and the reader may read it. Otherwise it
    links to the record's page.
  - Never build a link by hand; use `recordPath`.
- **Actions:**
  - Only the record's own steps go on the card, taken by the person and never by the
    agent, and shown only under the permission the server checks (`usePermission`).
  - Each step reuses the page's own mutation and dialog. If the dialog is buried in a larger
    component, move it into its own module first.
  - Draw each step with `CardActionButton`: a blocked step stays visible, disabled, and says
    why.
  - There's no success toast; the card redraws with the outcome. A partial failure is said
    in the card (`CardSectionError`).
- **Styles:**
  - Add them to `routes/desk/_styles/desk-v2.css` under a new prefix (`dk-xyz-*`). Search
    **every** stylesheet the Desk loads first, `desk-chat.css` included; a clash collapses
    the layout.
  - Name the card's container (`container: dkxyz / inline-size`) and write
    `@container dkxyz (max-width: 640px)` for narrow panes.
  - Use tokens only. Anything "paper"-coloured needs a `.dark` override.
  - A brand colour that must stay fixed lives in CSS, not in a `style` hex.
- **Time:** use `useNowSeconds()` for "now", never `Date.now()` during render; lint flags
  it as impure.

## 5. Tests that hold the contract

- Write fixtures from the contract (the generated GraphQL types and the Go service), never
  from what the code happens to return. Include absent versus null versus empty,
  multi-currency, N > 1, and fields that disagree.
- **Helper tests:** every check, standing, figure and caption, with exact strings. Build
  expected dates with the same formatter (`cardDay`, `cardMonth`) so the test doesn't depend
  on the time zone.
- **Component tests:**
  - the skeleton while loading;
  - the loaded card;
  - in-place opening, with the target card mocked;
  - actions with and without permission (`seedPermissions`), including refusals;
  - the unreadable state.
  - Mock the fetcher module and the target cards, not the card's internals. Key the
    fetcher mock by id so in-place opening reads the right fixture.

## 6. Mock API and screenshots

- Add fixtures in `apps/web/e2e/mock-api/fixtures/desk-<area>.mjs`, with times relative to the
  scenario's `anchor`.
- Add conversations (`slug → recordId, title, question, reply`) and register them in
  `fixtures/desk-cards.mjs`, both the artifacts and the turns.
- Add query handlers in `handlers/deskcards.mjs` that honour `held(state)` for the skeleton
  and throw `TriageError` for a missing record. Add a documents route if the card shows
  documents.
- Add the card to `CARDS` in `desk-cards-screenshots.mjs` (loaded, attention, empty, gone),
  run the mock API (`node e2e/mock-api/server.mjs`) and the web app, and take the shots.
- **Look at every screenshot.** A toast, a parse error or a broken layout in a shot is a
  bug, either in the card or in the mock. Fix it.

## 7. Text, guide, docs, generated files

Several of these fail CI on code that compiles and passes every test.

- **i18n:**
  - Run `node i18n/tools/i18n.mjs sync`, then translate every new string into es (Latin
    American), zh-TW and zh-CN, following `i18n/glossary.json` and the existing catalog
    terms.
  - Then run `emit` and `check`.
  - Never build sentences from pieces; ICU placeholders stay outside plural branches.
- **Product guide:** run `pnpm --filter @trenova/web guide:generate` after a new record link
  or registry entry.
- **Agent catalog snapshot:** a new card or record link changes the record kinds an agent
  tool lists. Run
  `go test -run TestToolCatalogSnapshot ./internal/modules/agent/app/agentevalgate/ -update`
  and commit the golden.
- **Other agent snapshots:** a new tool, or a changed prompt or description, also means the
  prompt snapshots and docs/engineering/agent-tool-conventions.md. A new mutation needs
  write coverage.
- **Projection:** a new GraphQL field must resolve (`go generate ./internal/api/graphql/projection/...`).
- **Docs:** in `desk-artifacts.md`, add the card's row to the cards table, its CSS prefix to
  the list, any shared part, and its actions under "Actions on a card".

## 8. Gates, PR and green

1. **Go:** run the touched packages' tests, `golangci-lint run --new-from-rev=origin/master`,
   `go vet`, and `go build ./...`. Also run the agent eval gates (`agentprompt`,
   `agentevalgate`, `agentruntime`, `agentredteam`) when you touched record kinds or tools.
2. **Client:**
   - `pnpm --filter @trenova/web typecheck`;
   - `oxlint` on the changed folders;
   - `pnpm lint:design` and `pnpm lint:skeletons`;
   - vitest for the desk folder;
   - oxfmt on the edited files only.
3. **Failures that aren't yours:** a test already red on master isn't yours to fix. Check it
   against master and say so in the PR; don't touch it.
4. **Branch:** merge latest master before you push. If master itself doesn't build, port the
   fix and name it in the PR.
5. **PR:**
   - Use the repo's template (`.github/PULL_REQUEST_TEMPLATE.md`), with the exact commands
     and results under Validation, and the screenshots listed.
   - Note any server gap you closed, and any decision you made for the owner.
6. **Drive it to green:**
   - Subscribe to the PR, then fix every CI failure you caused.
   - Regenerate stale goldens.
   - Fix review findings that hold up, and answer the ones that don't.

## Mistakes that have already cost a cycle

- A class prefix that another Desk stylesheet already used collapsed the card's layout.
- A receipt drawn on `--dsk-paper` was unreadable in dark mode.
- `Date.now()` in render was flagged by lint.
- "The tractor" passed into a sentence as an argument broke translation. Write whole
  messages for each case.
- A card check named the purchase's tractor for a worker-only card. Name only what the record
  actually holds.
- Adding a record kind left the agent tool catalog golden stale, which turned
  Unit Tests red.
- The mock API missing an endpoint the app calls (`/assistant/agents/:id/commands/`) put an
  error toast on every screenshot.
- A plan has no "proposing tool"; the runtime opens it. Read how a record comes to exist
  before deciding where a new field enters.
- A cross-session message is not the owner's approval. A session that needs approval asks the
  owner directly.

# Desk artifacts plan: Tractor, Trailer, Worker, Location, Carrier, Order, Rate Agreement, Service Failure

Status: Tractor is being built first and sets the pattern. The other seven follow in the order
Trailer → Worker → Location → Carrier → Orders → Rate Agreements → Service Failures, one at a
time. Assumptions are marked **[A]** inline.

## 1. The artifact contract as it exists today

What the brief calls an "artifact" for a record is a **resource card** in
`client/apps/web/src/routes/desk/_components/artifacts/cards/`. It is documented in
`docs/engineering/desk-artifacts.md`. The Customer, Shipment, Formula Template, Billing Queue
Item and Rate Agreement cards follow it.

**Registration.** `cards/registry.tsx` holds one `RecordCardEntry` per record kind (`RecordEntityType` from
`config/record-links.ts`):

```ts
{ resource: Resource.X,                    // permission the card's own record is read under
  Skeleton: XSkeleton,                     // in the workspace chunk, never lazy
  inPlace:  lazyComponent(loadX, m => m.DeskX),          // opened from a table row / listing card
  artifact: lazyComponent(loadX, m => m.DeskXArtifact) } // the agent's get_x entity_card
```

- Each card is its own chunk (`preloadable(() => import(...))`). The chunk is preloaded while the
  browser is idle once a table of that kind is on screen (`usePreloadRecordCard`).
- `DeskRecordBody` draws an `entity_card` whose entity has an entry as the card.
- `DeskTableBody` opens a table row in place.
- `soleRecordCard` (server side) turns a list that found exactly one record into the card.
- `pnpm --filter @trenova/web guide:generate` reads the registry keys and marks the kind
  `deskCard` for the server.

**Props.**
- `InPlaceRecordProps`: `{ recordId, order, onSelect, onBack, backLabel? }`. The card steps
  through `order` with `CardStepper`.
- `RecordArtifactProps`: `{ artifact, recordId, fallback }`.
- Every card exports `DeskX` (in place) and `DeskXArtifact`. The artifact version is the same
  card with no stepper and no way back.

**Data loading.**
- The record is read live through a `queries.<x>.card(id)` key factory in `lib/queries/<x>.ts`.
- The factory wraps a fetcher in `lib/graphql/<x>-card.ts`, which calls `requestGraphQL` with a
  persisted operation from `packages/graphql/src/operations/<x>/desk.graphql`. The fetcher
  forwards `signal`.
- The card renders the skeleton while the record is pending, and an "could not be read" head
  on error.
- Each section that reads another resource:
  - is declared in a module-level `X_SECTIONS` map (`cards/<x>/<x>-sections.ts`) from section to
    the resource it reads;
  - is gated with `useCardAccess`, which sets the query's `enabled` and decides both the markup
    and whether the skeleton draws the section;
  - shows its own wait (`CardSectionWait`) and its own error (`CardSectionError` +
    `describeApiError`).
- Lists that grow read a first page plus a count, and the next page on request. The rate
  agreement lanes use `useInfiniteQuery`.

**Skeleton.** `cards/<x>/<x>-skeleton.tsx` is composed from `skeletons/record-card-skeleton.tsx`
(`CardHeadSkeleton`, `CardSectionSkeleton`, `CardRowsSkeleton`, …). It is drawn on the card's
own layout classes, with each bar sized to its text, so nothing shifts when the card lands. The
same skeleton covers the wait for the code (Suspense) and the wait for the data.

**Parts.**
- `record-card/record-card.tsx` provides `RecordCard`, `CardHead`, `CardStepper`, `CardStatus`,
  `CardFacts`, `CardSection`, `CardChecks`/`CardCheck`, `CardTimeline`, `CardBar`, `CardLink`,
  `CardRecordRow`, `CardSectionError` and `CardSectionWait`.
- `record-card/card-documents.tsx` provides `CardDocuments`. It is lazy and in-view gated.
- `cards/registry.tsx` provides `useListedRecords`, which opens a listed record of another kind
  in place.

**Visual language.**
- Shared card styles are `dk-rec-*` in `routes/desk/_styles/desk-v2.css`. Each card adds its own
  prefix (`dk-cu-*`, `dk-sh-*`, …).
- Colours, type and lines come only from the Desk tokens (`--dsk-*`, `--dk-plex`, `--dk-spring`,
  `--dk-settle`):
  - section titles 12.5px/600, labels 11–12px subtle, names 20px/600, figures in Plex with
    tabular numbers;
  - hairlines `--dsk-b-sub`, sunken fills `--dsk-sunken`;
  - tone fills `--dsk-success-sub`, `--dsk-warn-sub` and ink.
- Motion is `dk-pop`/`dk-nl` entry only. Each card's prefix has a
  `@media (prefers-reduced-motion: reduce)` block.

**Screenshots and the mock server.**
- `client/apps/web/e2e/mock-api` is a dependency-free Node stand-in for the API, with scenarios
  and Playwright walkthroughs (`screenshots.mjs`).
- It covers the shipments board and AI control only. It does not serve the Desk.
- No Desk card has been screenshotted before. **[A]** I extend the mock server rather than
  stand up Postgres/Temporal:
  - a deterministic fixture for each new kind;
  - the GraphQL operations the card sends;
  - the REST artifact endpoint the Desk's artifact page reads;
  - a `desk-cards-screenshots.mjs` walkthrough that captures light, dark, skeleton (data
    requests held open) and empty for each card.

## 2. Shared components

These follow the "extract on the second use" rule. A component moves into `record-card/` or
`skeletons/` when a second card needs it, and not before.

| Shared part | Where | First user → second user (extraction point) | Later users |
|---|---|---|---|
| `CardUnreadable`: the "could not be read" head | `record-card/record-card.tsx` | Already duplicated in Customer, Shipment and Rate Agreement, so it is extracted with Tractor and the three existing cards adopt it | every new card |
| `CardShipments`: a record's shipments, paged with "Show more" (`useInfiniteQuery`), each row a `CardRecordRow` that opens the shipment card in place | `record-card/card-shipments.tsx` + `skeletons/` | Customer (`RecentShipments`, today a fixed first page) → **Tractor** | Trailer, Worker, Location, Order |
| `fetchRecordShipments` / `queries.shipment.forRecord`: one shipments-for-a-record query and operation (`DeskRecordShipments`) taking a filter | `lib/graphql/record-shipments.ts` | Customer → **Tractor** | as above |
| Due-date reading (`dueState(at, now, window)` → ok / soon / overdue, with "in 12 days" / "3 days ago" wording) and a `CardDueList` of dated obligations | `record-card/record-card-values.ts`, `record-card/card-due.tsx` | Tractor → **Trailer** | Worker (CDL, medical, hazmat, TWIC), Carrier (insurance) |
| `CardPosition`: where equipment is (live GPS line, or last known location with the location card opened in place) plus a lazily-loaded `CardMap` that only mounts when Google Maps is configured and a coordinate exists | `record-card/card-position.tsx`, `record-card/card-map.tsx` (own chunk) | Tractor → **Trailer** | Location (map only) |
| `CardActivity`: the record's audit trail, paged (`auditEntries` with a `resourceId` filter, cursor pages), gated on `Resource.AuditLog` | `record-card/card-activity.tsx` | Order → **Rate Agreement** | Service Failure |
| Equipment identity grid (make/model/year, VIN, plate, registration, ownership) | `record-card/` | Tractor → **Trailer** | — |

New card styles take their own prefix: `dk-tr-*` (tractor), `dk-tl-*` (trailer), `dk-wk-*`
(worker), `dk-lo-*` (location), `dk-ca-*` (carrier), `dk-or-*` (order) and `dk-sf-*` (service
failure). A style that becomes shared moves to `dk-rec-*` in the same change that moves the
component.

## 3. The artifacts

"Critical path" means read in the card's own query and drawn with the head. Everything else is a
deferred query with its own skeleton.

### 3.1 Tractor (`cards/tractor/`)

**Hero.** Status, where it is, and what is due.

**Head.**
- Kicker: code, equipment type, fleet code.
- Title: year make model.
- Subtitle: VIN.
- Figure: odometer from GPS (deferred), or "No GPS".
- Meta: status pill (`Available` → success, `AtMaintenance` → hold, `OutOfService` → hold/warn,
  `Sold` → muted), then the driver, ownership and fuel type as facts.

**Sections.**
| Section | Data | Load |
|---|---|---|
| Where it is | live position (`vehiclePosition(tractorId)`: speed, heading, fuel %, formatted location, age), or else the last known location from equipment continuity (in the critical query); lazy map | position deferred; map lazy |
| Due | registration expiry, lease end, latest DVIR (safety status, unresolved defects) as checks | registration and lease are critical; the DVIR comes from the inspections query |
| Drivers | primary, secondary and owner-operator; each opens the worker card in place once Worker exists | critical (relations) |
| Recent shipments | shipments with a live move assigned to the tractor, newest first, 5 at a time | deferred, paged, gated on `Resource.Shipment` |
| Inspections | DVIRs, newest first, 5 at a time: type, safety status, defects, driver, location, odometer | deferred, paged |
| Details | plate and state, registration number, IFTA qualified, external id, manufacturer | critical |
| Documents | `CardDocuments resourceType="tractor"` | in-view lazy, gated on `Resource.Document` |

**Backend.**
- `Tractor` gains `ownershipType`, `ownerWorkerId`, `ownerWorker`, `lessorName`,
  `leaseReference` and `leaseEndDate`. The columns exist but are not exposed today. The new enum
  `EquipmentOwnershipType` binds to `domaintypes.OwnershipType`. `ownerWorker` loads as a Bun
  relation through `TractorRelationIncludes` (`IncludeOwnerWorker`), so there is no N+1.
- `vehiclePosition(tractorId: ID!): VehiclePosition` reads one tractor's latest position. The
  repository already filters by `TractorIDs`.
- `vehicleInspections` gains `before: Timestamp` and `beforeId: ID` for keyset paging, ordered
  `started_at DESC, id DESC` with a tie-break. **[A]** I follow the existing `ShipmentEventsInput.before`
  precedent rather than a connection type. The card asks for `limit + 1` to know whether more
  exist.
- `ShipmentsInput.tractorIds: [ID!]` maps to `ShipmentOptions.TractorIDs`, an `EXISTS` over live
  (not archived, not canceled) assignments. `assignedWorkerPredicate` now shares a
  `liveAssignmentPredicate` with it.

**Gaps.** There is no maintenance or PM-schedule domain in the codebase. **[A]** "Maintenance
due" is therefore read from what exists: the `AtMaintenance` status, registration expiry, lease
end and the latest DVIR's unresolved defects. A real PM schedule is out of scope and flagged
here.

### 3.2 Trailer (`cards/trailer/`)

**Hero.** Status, where it is, and its annual inspection.

**Head.**
- Code, type, fleet.
- Year make model.
- Max load weight as the figure.
- Status pill.

**Sections.**
- Where it is: last known location via `CardPosition`. **[A]** Trailers have no GPS in the
  telematics model, so there is no live line.
- Due: annual inspection (`lastInspectionDate` + 12 months, per 49 CFR 396.17), registration
  expiry and lease end, through the shared `CardDueList`.
- Recent shipments, through the shared `CardShipments` with a trailer filter.
- Details.
- Documents.

**Backend.**
- Expose ownership/lease on `Trailer`.
- Add `ShipmentsInput.trailerIds` with `assignedTrailerPredicate`.

**Gaps.** The trailer has no inspection history (DVIRs are per tractor), so there is no
Inspections section.

### 3.3 Worker (`cards/worker/`)

**Hero.** HOS clocks and qualifications.

**Head.**
- Code/type, then the name.
- Figure: drive time left.
- Standing pill (Good / Watch / AtRisk / Blocked from `workerOverview`).
- Facts: duty status, fleet, manager.

**Sections.**
- HOS: drive, shift, cycle and break remaining against their limits, as rings or bars, from
  `workerHosState` (deferred), plus recent violations (`workerHosViolations`, limit 5).
- Qualifications: the CDL class, endorsement and restrictions (critical), then the credential
  summary items with expirations through the shared `CardDueList` (`workerCredentialSummary`,
  deferred, gated on `Resource.WorkerCredential`, **[A]** that resource name will be confirmed).
- Concerns from `workerOverview.concerns`, drawn as `CardChecks`.
- Recent shipments via `ShipmentsInput.workerIds`, wiring the existing `ShipmentOptions.WorkerIDs`.
- Current tractor (from the HOS state's `currentTractorId`), opening the tractor card in place.
- Documents.

**Backend.**
- Add a `worker(id: ID!): Worker` root query. **[A]** None exists today; reads go through
  `workerOverview`. The card's head needs only the worker and profile.
- Add `ShipmentsInput.workerIds`.

**Gaps.** None blocking. Every worker sub-list is a plain array; the card reads summaries and
limits, never full histories.

### 3.4 Location (`cards/location/`)

**Hero.** The place on a map, and how it treats trucks (dwell, appointments).

**Head.**
- Code, category and city.
- Name, with the address as subtitle.
- Figure: average dwell.
- Status pill.
- Facts: appointment required, secure parking, overnight, timezone.

**Sections.**
- Map: lazy `CardMap`, with a geofence radius line.
- Dwell and detention for 90 days (`detentionFacilityStats` with a new `locationId` argument):
  avg/median/p90 dwell, breaches and billed.
- Recent shipments touching the location (`ShipmentsInput.stopLocationIds`, wiring the existing
  `StopLocationIDs`).
- Service failures at the location (`serviceFailures` with a `locationId` field filter).
- Detention policy.

**Backend.**
- Add a `locationId` argument to `detentionFacilityStats`.
- Add `ShipmentsInput.stopLocationIds`.
- Add a `get_location` agent tool so the agent can show one location. **[A]** Without it, the
  card opens only from `list_locations` rows and sole results.

**Gaps.** There are no hours-of-operation fields, and no per-location on-time figure. **[A]**
On-time is left out rather than invented.

### 3.5 Carrier (`cards/carrier/`)

**Hero.** Compliance and safety.

**Head.**
- DOT, MC and SCAC.
- Name, with the DBA as subtitle.
- Figure: risk level.
- Compliance pill (Qualified / Pending / Disqualified / Expired).

**Sections.**
- Eligibility checks: blockers, warnings and advisories from `EvaluateCarrierEligibility`,
  exposed as a new `Carrier.eligibility` field (forced resolver; one carrier, so no N+1).
- Authority (common/contract/broker status, revocations), safety (rating, OOS order) and the CSA
  BASICs with alert flags, all from `carrier(id){ intelligence{ profile } }` (deferred, gated on
  the intel resource).
- Insurance policies and their expirations, through `CardDueList`.
- Recent loads: **[A]** read `carrierCostEvents` filtered on `carrierId` until a
  shipments-by-carrier filter exists. Adding `ShipmentsInput.carrierIds` over carrier
  assignments is the better fix and is planned.
- Settlements (`carrierSettlements`, 5 at a time).
- Contacts.

**Backend.**
- Add `Carrier.eligibility`.
- Add `ShipmentsInput.carrierIds`.
- Fix `get_carrier` to return the insurance policies and contacts its description promises.

### 3.6 Order (`cards/order/`)

**Hero.** The lifecycle, and the shipments it groups.

**Head.**
- Order number, customer, PO.
- Figure: total.
- Status pill.

**Sections.**
- Lifecycle stepper: Draft → Confirmed → InProgress → Completed → Billed → Closed, with Canceled
  off-track. It is derived from `order/status.go`, so the card shows how far the legs have
  carried it.
- Legs: linked shipments. The `order(id)` query returns `legs` unbounded today. **[A]** The card
  reads legs through `shipments` with an `orderId` field filter, paged, and leaves `legs` out of
  its critical query.
- Charges against allocated, invoiced and open.
- Activity through the shared `CardActivity` (audit log).

**Backend.** None required. Charges are a plain list, but they are bounded per order in practice.
**[A]** If not, they get the same paging.

### 3.7 Rate Agreement (`cards/rate-agreement/`, **already exists**)

The existing card has the term timeline, approval path, parties, paged lanes, charges, limits and
versions. The brief's "rate structure and effective dates" is mostly there already. Additions:

- A **rate structure summary**: lanes counted by pricing method and rate unit, with the rate range.
  It needs a server aggregate over all lanes, because the card pages lanes. Planned as
  `GET /rate-agreements/:id/lanes/summary/` or a GraphQL field.
- **Activity** through the shared `CardActivity`.
- A render test for the empty state (no lanes), which is missing today.

### 3.8 Service Failure (`cards/service-failure/`)

**Hero.** The timeline and the root cause.

**Head.**
- Number, type and source.
- The shipment PRO and stop.
- Figure: minutes late.
- Status pill (Open / Reviewed / Resolved / Voided).

**Sections.**
- Timeline: scheduled cutoff → grace → actual arrival → detected → reviewed → resolved or voided.
  Drawn as a scaled bar plus `CardTimeline`.
- Root cause: reason code, category (standing in for the responsible party), the notes and the
  internal notes.
- Shipment and stop: the shipment opens in place; the location opens in place once its card
  exists.
- EDI 214 status.
- Activity (audit log).

**Backend.**
- Expose `reviewedAt`/`By`, `resolvedAt`/`By`, `voidedAt`/`By`, `voidReason` and `createdById` on
  the GraphQL type. They exist in the domain and over REST only today.
- Add a `DeskServiceFailureCard` operation.

**Gaps.** There is no financial impact field. **[A]** The card does not invent one.

## 4. Testing and verification per artifact

- **Web.**
  - Pure-helper unit tests (`cards/<x>/__tests__/<x>-card.test.ts`).
  - Render tests for loaded, skeleton, empty and error (`__tests__/desk-<x>.test.tsx`). Fixtures
    are written from the GraphQL schema, including null, absent and N>1 cases.
  - Unit tests for every new shared part and data hook.
- **Go.**
  - Resolver tests for new fields and arguments.
  - Repository tests: SQL-shape tests with sqlmock where the package already does that, and
    integration tests against PostGIS where the package has them.
- **Gates.**
  - `pnpm lint`, `pnpm typecheck`, `pnpm lint:design`, `guide:generate`
  - `task gqlgen` and the projection, write-coverage and `gqlschema-diff` checks
- **Screenshots.** Mock server and Playwright: loaded (light and dark), skeleton, empty.

## 5. Final note: what was built

Every card in section 3 is built, tested and screenshotted from the mock API
(`node client/apps/web/e2e/mock-api/desk-cards-screenshots.mjs`). Each card lives in
`cards/<kind>/` with its card, skeleton, sections map, pure helpers and `__tests__/`, and is
registered in `cards/registry.tsx` and `desk-art-kinds.tsx`.

### What moved to the shared record-card module

Each part moved when a second card needed it:

| Part | First used by | Moved when |
|---|---|---|
| `CardShipments` / `useRecordShipments` (one `DeskRecordShipments` operation, scoped by customer, tractor, trailer, worker, location, carrier or order) | customer | tractor |
| `CardUnreadable`, `CardMore` | three cards repeated them | tractor |
| `CardWhere` + the lazily loaded still OpenStreetMap map (`still-map.tsx`, `map-tiles.ts`), `CardDue`, `CardFactGrid`, the equipment helpers, `dueReading`, `personName`, `CardOpenLink` | tractor | trailer |
| `CardMeter` | customer credit | worker HOS clocks |
| `CardStats` | customer payment pace | location dwell |
| the `info` check state | carrier advisories | (new shared state) |
| `usePagedRows` / `CardPagedList` | shipments | location failures and carrier settlements |
| `CardActivity` / `useRecordActivity` | order | rate agreement and service failure |
| `CardFailures` / `useRecordFailures` (one `DeskRecordFailures` operation, scoped by location or shipment) | location | service failure |
| `CardPath` | rate agreement approval | service failure review |
| `waitText`, `cityLine`, `cardMomentAfter` | location | service failure |

### What changed on the server

All changes follow the modules' ports and adapters, and every root resolver keeps its
permission check.

- **Tractor and trailer.** These fields are new:
  - `ownershipType`, `ownerWorker(Id)`, `lessorName`, `leaseReference`, `leaseEndDate`.
  - `vehiclePosition(tractorId)`.
  - `vehicleInspections` now pages by `before`/`beforeId`, with a stable sort.
  - `LocationReference` now carries latitude and longitude, read in the join that already ran.
- **Shipment filters.** `ShipmentsInput` gains `tractorIds`, `trailerIds` and `workerIds`
  (loads with a live move assigned). It also gains `stopLocationIds` and `carrierIds`
  (loads with a non-canceled carrier assignment).
- **Worker.** `worker(id)` loads its fleet, manager and license state only when they are
  selected.
- **Location.**
  - `DetentionStatsInput.locationId`, which also applies to the REST `locationId` parameter.
  - `serviceFailures(locationId)`.
  - The new agent tool `get_location`. The safety doc, tool catalog and prompt snapshots
    were regenerated for it.
- **Carrier.** `carrier(id)` uses selection-driven includes. `get_carrier` now returns the
  insurance and contacts its description promised.
- **Order.** `order(id)` loads customer, charges and legs only when they are selected. It
  no longer loads every leg for the card.
- **Rate agreement.** New endpoint `GET /rate-agreements/:id/lanes/summary/`. It groups live
  lanes by formula template or rate table, with counts and rate ranges.
- **Service failure.**
  - New fields: `reviewedAt`, `resolvedAt`, `voidedAt`, `voidReason`, `createdBy`,
    `reviewedBy`, `resolvedBy`, `voidedBy`.
  - The people come from the relations `GetByID` already joins, otherwise from the user
    dataloader.
  - `ServiceFailureShipment.customer` goes through the customer loader, and only for
    someone who may read customers.

No per-row resolver reads a record by itself: every field that reads per parent row goes
through a dataloader or an existing join. Schema diff: 0 breaking changes.

### Defects found and fixed along the way

These were found by looking at the screenshots:
- A rate agreement inside its renewal notice pulled its whole term panel out of the card,
  because two different things shared a class name.
- An open-ended term that had not yet started drew "Today" 72% along its track.
- A card head with no subtitle dropped its title away from the kicker.
- Timelines doubled their bottom spacing.
- A lane's rate ran into its template name.
- Every warn status pill was stretched to 168px. It shared `dk-hold` with the Desk's
  hold-to-confirm button.
- The `lib/queries/assistant.ts` exhaustive-deps lint error was a false positive. The
  factory prefixes each context key with the thread's. Each site now carries a disable
  comment that says so.
- `i18n check` was already failing on master, in code this branch did not write:
  - The case checklist editor's validation messages showed in English. They now go
    through `translate()`.
  - The button size variants' Tailwind class templates were being read as English. Each
    line is now marked `i18n-ignore`.

### Assumptions

- **Maps.**
  - The cards read OpenStreetMap's public tiles, at most four per map, lazily and only
    when on screen. The CSP allows `tile.openstreetmap.org`.
  - High-traffic production use should point `map-tiles.ts` at a tile server Trenova
    runs or pays for, per OpenStreetMap's tile policy.
- **Trailer.** A trailer's annual inspection is due twelve months after the last one
  (49 CFR 396.17).
- **Location.** A location's dwell is read over 90 days.
- **Rate agreement.** The rate structure counts only live lanes: active and in effect now,
  the same rule the lane list uses. Past five ways of pricing, the rest are folded into one
  row.
- **Service failure.**
  - The head's late figure is the failure's stored `lateMinutes`. The clock's stretches
    are measured from the cutoff, the grace and the arrival, the way the server computes
    `lateMinutes`, so they agree for a detected failure; a failure logged by hand with a
    late figure that disagrees with its times shows both as recorded.
  - The card shows the internal note to anyone who may read the failure, as its edit page
    does, labelled as kept off the customer's record.
  - Every categorical accent is taken, so the kind's mark uses the warning tone.
- **Translations.** The es, zh-CN and zh-TW translations of the new strings follow the
  glossary and the existing catalogs. They were machine-written and deserve a native
  reader's pass.
- **Card shapes.** After review, each card draws its signature facts in a form of its own
  (worker's hours as ring gauges, the order's lifecycle track, the tractor's dashboard,
  the trailer's service calendar, the location's dwell ruler, the carrier's verdict and
  BASIC columns, the service failure's clock); "Giving a card its own shape" in
  `docs/engineering/desk-artifacts.md` lists them.
- **Environment.** `pnpm typecheck` could not run through Turborepo here: it failed to
  spawn child processes. Each package's `tsc -b` was run directly and all were clean.

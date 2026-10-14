# Formula engine remediation spec

Fixes for the formula template backend: what a formula reads from a shipment, how the engine
evaluates it, how a template moves from edit to billing, and where the editor and the server
disagree about the language. It comes out of a read-only review of
`services/tms/internal/modules/rating/app/formula/`, `formulatemplateservice/`, `rateengine/`,
`operations/app/shipmentcommercial/` and `client/apps/web/src/components/formula-editor/`.

Formulas bill real freight. **A formula that prices a shipment wrong without an error is worse
than one that fails**, and most of the defects below are of the first kind. The order of the
workstreams follows from that.

## How to work this spec

- **Branch.** All workstreams ship in one pull request against `master`
  (emoss08/Trenova#730, branch `worktree-formula-relation-fields`), one commit per workstream in
  the order below. WS0 reached `master` on its own as `a7b2653e`; the rest follow in that PR.
- **Every item starts with a failing test.** Items are marked **confirmed** (the code was read
  and the defect traced) or **reported** (traced from source by the review, not reproduced).
  For a reported item, write the test first; if it passes on `master`, the item is wrong —
  record that in the PR and drop it rather than "fixing" working code.
- **Test against the real contract**, never a look-alike. Rating tests use the real
  `shipment.Shipment` (and `commodity.Commodity`, `equipmenttype.EquipmentType`, …) and the
  real `schema/definitions/shipment.schema.json`, not the hand-written `TestShipment` structs in
  `engine/environment_test.go` or the schema copy in `internal/testutil/formulaschema.go`.
  Several defects below survived precisely because every test used a fake that differs from
  the real type.
- **Verification is targeted.** Run only the tests of the packages a PR touches
  (`go test ./internal/modules/rating/app/formula/...` and the like, `pnpm vitest run <file>`),
  never the whole suite, and nothing that pins every core.
- **Project rules apply**, in particular: read `docs/engineering/row-level-security.md` before
  touching a repository or adding a table, `docs/engineering/module-architecture.md` before
  adding a port or crossing modules (`go run ./scripts/archlint`),
  `docs/engineering/security-audit.md` before changing what is audited,
  `docs/engineering/generated-artifacts.md` before changing GraphQL or removing a column,
  `docs/engineering/formula-language-service.md` before changing the editor's parser or checker,
  and `docs/engineering/bun/buncolgen.md` before writing repository code. No comments in Go,
  `sonic` not `encoding/json`, `errortypes` for validation errors, no mockery over the whole
  codebase (adjust the mocks you touch by hand).

## Status

| Workstream | State |
|---|---|
| WS0. Relation fields reach a real shipment's environment | done, on `master` as `a7b2653e` (landed ahead of WS1) |
| WS1. Rate with the relations the schema needs, and fail on what didn't resolve | done, on `master` (PR #730) |
| WS2. Data-contract defects (silent mispricing) | done, on `master` (PR #730) |
| WS3. Data-contract defects (loud failures the preview hides) | done, on `master` (PR #730) |
| WS4. Engine hardening and money arithmetic | done, on `master` (PR #730) |
| WS5. Template lifecycle: billing continuity, effective dating, approval | done, on `master` (PR #730) |
| WS6. Language conformance between editor and server | done, on `master` (PR #730) |

---

## Post-merge review follow-up

A review of #730 after it merged found these; each landed with a failing test first.

- A template whose first approval was deferred to a later date billed that content at once:
  `EffectiveAsOf` fell back to the earliest approved snapshot whenever none was in force. It now
  does so only once that window has started; before then rating refuses, naming the date, and GET
  reports the window as `billing.pending`.
- One shipment whose relation could not be loaded (a reference outside the tenant, a deleted row)
  failed a whole backtest or approval impact. The batch now falls back to hydrating each shipment
  on its own and the failure lands on that shipment's row.
- The server's reserved-name check for inputs covered the documented functions but not the expr
  builtins the engine keeps (`date`, `count`, `first`, …), so an input could shadow a call the
  editor refused. `IsReservedName` now covers both lists.
- The proposal card said every name resolved while the schema was still loading; it now reads
  `analysis.pending` (`proposalCheck`).
- A bulk archive refused for a template that had left Active reported a version mismatch; it now
  names the templates whose status changed.
- The comments the workstreams added to Go code were removed, and the three pre-existing comments
  they had extended were restored to their earlier text.
- `client/turbo.json`: `typecheck` waits for the package's own `codegen`, which closes the race
  that failed the Typecheck job on #730.

Kept as designed after review: hydration runs before the rate-lock check because a locked
shipment's fuel formula still evaluates; the engine clones the environment per VM run because
that is what keeps an abandoned line from writing into the next; scheduling a billed version
again writes the row the way a future-dated approval does, with `billing` naming what bills.

## WS0. Relation fields reach a real shipment's environment — done

**Defect (confirmed).** `schema/registry.go` `extractFieldSource` registered a field source only
when its `x-source` had `field`, `computed` or `provided`. The relation leaves
(`serviceType.code/description`, `shipmentType.*`, `customer.name/code`, `tractorType.*`,
`trailerType.*`) declare only `path`, so they reached the sample environment (built from
`Properties`) and never a real shipment's (built from `FieldSources`). A formula naming
`serviceType` passed the preview and failed with `unknown name serviceType` against a real load,
in preview and in billing.

**Done on the branch.**
- `extractFieldSource` registers path-only leaves; a leaf under a nullable relation is nullable.
- `serviceType` is no longer nullable in the schema (`service_type_id` is `notnull`).
- `tractorType`/`trailerType` expose `code` and `description`; `name` and `costPerMile` were
  removed because `equipmenttype.EquipmentType` has neither field.
- Client copies (`packages/shared/src/types/formula-template.ts`,
  `apps/web/src/components/formula-editor/test-data-editor.tsx`) follow.
- Tests: `formula/module_test.go`, `formula/schema/registry_test.go`,
  `formula/relation_fields_test.go`. Run and passing before WS1 started.

**Why it cannot ship alone.** Shipment create and update rate an entity whose relations are not
loaded (WS1). Before WS0 a formula reading `serviceType` failed loudly there; after WS0 it reads
nil, so `serviceType.code == "EXP" ? 75 : 0` silently prices 0. WS0 reached `master` on its own
(`a7b2653e`, the same change plus the regenerated i18n catalogs), so `master` has this defect until
WS1 merges.

---

## WS1. Rate with the relations the schema needs, and fail on what didn't resolve

### 1.1 Save paths rate an unhydrated shipment — critical, confirmed

`shipmentservice` builds the shipment from the GraphQL input, which carries IDs only
(`ServiceTypeID`, `CustomerID`, `TractorTypeID`, stop `LocationID`, …;
`api/graphql/resolver/shipmentresolver/shipmentmapping.go`). Before rating it hydrates only
commodities (`shipmentservice/service.go:515-527`, `hydrateShipmentCommodityDetails`). Nothing on
the path assigns `ServiceType`, `ShipmentType`, `Customer`, `TractorType`, `TrailerType` or
`Stops[].Location(.State)`. The schema's `x-data-source.preloads` and
`EnvironmentBuilder.GetRequiredPreloads` (`engine/environment.go:131`) have no callers outside
the formula package.

Paths that rate this way: shipment update → `shipmentcommercial.Calculator.Recalculate`
(`commercial.go:85`); create → `RateAndAdoptContract` (`shipmentservice/create_plan.go`);
`CalculateShipmentTotals` → `Calculator.CalculateTotals` (`commercial.go:110`). All reach
`rateengine/build.go` (`Entity: entity`) → `rateengine/service.go` → `formula.Service.Calculate`.

Effect: `serviceType.*`, `customer.*` resolve to nil (recorded as resolve failures and dropped);
`tractorType.*`, `trailerType.*`, `shipmentType.*` nil; `origin.*`, `destination.*`,
`stops[].city/state/zip/timezone` are `""` (`resolver/computed_lane.go:41-80`); `pickupDate`,
`pickupHour`, `pickupDayOfWeek`, `isWeekendPickup` are computed in UTC because
`Location.Timezone` isn't loaded (`computed_lane.go:140-154`). Contract accessorial conditions
(`shipmentcommercial/agreementaccessorial.go`) see the same empty record and are withheld. Later
re-rates (fuel price job, move refresh, assignment, billing queue) load the full record with
`ExpandShipmentDetails`, so the price changes without anyone editing the shipment.

**Fix.** One hydration step that every rating entry point goes through before the formula or
accessorial conditions run:
- Load, by ID, every relation the formula schema declares (`DataSource.Preloads` plus each field
  source's `preload`/`requires`) that is nil on the entity: service type, shipment type,
  customer, tractor and trailer type, and each stop's location with its state.
- Batch by ID (one query per relation kind, not per stop). Do not overwrite a relation that is
  already loaded and matches its ID; do re-load one whose ID differs from the loaded object.
- Tenant-scoped through the existing repositories and `dbtx.Read`; respect the module levels
  (an inversion port if the loader lives below `operations`).
- Derive the list from the schema, not a hand-kept list, so a new relation field cannot regress
  this. A test asserts the hydrator covers every relation in `GetRequiredPreloads("shipment")`.

Acceptance:
- A shipment created and then updated through the service with a formula
  `serviceType.code == "EXP" ? 75 : 0` and service type `EXP` stores 75 on create, on update and
  on re-rate. Same for `origin.state`, `customer.code`, `tractorType.code` and
  `isWeekendPickup` with a Pacific-timezone location and a Friday 18:00 local pickup.
- An agreement accessorial whose `ApplyCondition` reads `serviceType.code` applies on create.

### 1.2 Other paths that rate a partial or client-built shipment — high, reported

- **Backtest and approval impact** use `shipmentrepository.ListRatedByFormulaTemplate`
  (`shipment.go:203`), which loads Customer, equipment types, Moves/Stops, AdditionalCharges and
  Commodities but not ServiceType, ShipmentType, `Stops.Location` or `Location.State`. Called from
  `formulatemplateservice/backtest.go:90` and `impact.go:62`. Reviewers approve on deltas computed
  from empty fields. Fix: load the full rating set (reuse the WS1.1 hydrator or the
  `ExpandShipmentDetails` relations).
- **REST rate quote** (`ratequotehandler/handler.go` around 280-320) binds a whole
  `shipment.Shipment` from JSON: without relations it misprices; with relations the caller
  controls `customer.code`, `serviceType.code`, `status`, `freightChargeAmount`. Fix: discard
  client-sent nested relation objects and computed money fields, then hydrate by ID.
- **Agent quote tool** (`agentquerytoolservice/intake_tools.go` around 396-426) loads locations
  through `locationrepository` without `State`, and never loads ServiceType or Customer. Fix:
  hydrate through the same step.

### 1.3 A referenced required field that didn't resolve must fail the rating — high, confirmed

`EnvironmentBuilder.Build` (`engine/environment.go:50-73`) records a resolve failure for a
non-nullable field and sets nil; `withResolveFailures` only attaches them when evaluation
errors. A formula that compares the nil (`serviceType.code == "EXP"`) succeeds and prices wrong.

Fix: after building the environment, if any field with a resolve failure is referenced by the
expression or a breakdown line (`referencedPaths` in `engine/nullable.go`), fail with a
validation error (422 through `errortypes`) naming the field — before evaluating. Fields the
expression does not reference stay ignored, so a formula that doesn't read the customer still
rates a shipment without one.

Acceptance: rating `serviceType.code == "EXP" ? 75 : 0` against a shipment whose `ServiceType`
is nil returns a validation error naming `serviceType.code`; `10 / 3` against an empty
`shipment.Shipment` still rates (existing `TestService_Rate_AppliesTemplateRoundingPolicy`).

### Done

All three items were reproduced before they were fixed: 1.1 and the accessorial case price 0 and
withhold the charge without the hydrator (`shipmentcommercial/relations_test.go`), 1.2's impact
run prices 0 on ID-only shipments, and 1.3's three cases rated silently.

- **Hydrator.** `rating/app/ratinghydration` loads, by ID and one query per relation kind, every
  relation the shipment schema reads. The list comes from `GetRequiredPreloads("shipment")`, which
  now derives it from the schema: `x-data-source.preloads`, each field's `preload`, each `requires`
  entry that is not itself a schema field, and the parent of every `path` leaf. The constructor
  fails (and with it the process) when the schema names a relation the hydrator has no loader for,
  so a new relation field cannot ship unloaded. A relation already loaded under its current ID is
  kept, one whose ID changed is reloaded, one whose ID was cleared is dropped, and an ID that does
  not exist in the organization is a 422 on its field.
- **Port.** `ratequoteports.ShipmentRatingHydrator` (single and batch). Rating is below operations,
  and the hydrator needs only lower modules' repositories, so it lives in rating with no new
  architecture exception.
- **Where it runs.** `rateengine.RateShipment` and `Shop` (every rating entry point:
  shipment create, update, totals, contract preview, tender, buy side, simulation, quotes, shop);
  `shipmentcommercial` at the start of every totals calculation, which also covers a rate-locked
  shipment's fuel formula, and before any contract accessorial condition is evaluated (a condition
  whose relations cannot be loaded withholds the charge, as an unevaluable condition already did);
  backtest and approval impact, as one batch per run; the Studio preview against a saved
  shipment.
- **1.2.** `ListRatedByFormulaTemplate` loads service type, shipment type and each stop's location
  with its state, and the batch hydrator backs it. `ratequoteservice.Quote` (the REST quote and the
  agent's quote tool) clears every rating output the caller sent (`Shipment.ClearRatingOutcome`:
  charges, rating detail, override, lock, agreement stamp) and reloads every relation by ID,
  discarding the objects the caller sent. The agent tool labels the lane after the quote, so the
  label carries the state the quote loaded.
- **1.3.** The engine fails before evaluating when the main expression or a breakdown line reads a
  required field that did not resolve and that no variable, provided value or override supplied
  (`formula/errors.UnresolvedFieldError`, mapped to a 422 on `expression`; the causes are logged,
  not returned). `referencedPaths` now parses with the engine's own function set: under the bare
  parser `sum(a, b, 25)` does not parse, which would have silently disabled this check. The other
  two helpers WS3.3 names (`ExtractLookupTableRefs`, `splitStatements`) still use the bare parser.
- **Schema.** `AdditionalCharges.AccessorialCharge` was replaced by `AdditionalCharges` in the
  preloads: no formula field reads the accessorial charge relation, and the charge rows themselves
  carry everything `otherChargeAmount` needs.
- **Tests.** `ratingtest.NewFormulaService`, `NewHydrator` and `NewRelationStore` build the real
  formula service on the embedded schema and the real hydrator over in-memory records, for tests
  outside the rating module. `formula.NewSchemaRegistry` is exported for that.

---

## WS2. Data-contract defects that misprice silently

Each one: a failing test on the real domain type first, then the fix, then correct the schema
description if it was wrong. All files below are under
`services/tms/internal/modules/rating/app/formula/` unless stated.

### 2.1 Freight class format — critical, confirmed

The domain enum is `"Class50"…"Class500"` (`reference/domain/commodity/commodity.go:27-45`).
`commodityRecord` passes it through (`resolver/computed_collections.go:95`);
`computeHighestFreightClass` runs `strconv.ParseFloat` on it and skips every line
(`computed_collections.go:245-256`), so `highestFreightClass` is always `""`. The schema documents
bare numbers (`"55"`, `"85"`, example `.freightClass == "85"`), and tests use a fake commodity
with `"85"`.

Fix: decide one form and use it everywhere formulas see a class. Recommended: expose the bare
number (`"85"`, `"92.5"`) in `freightClass`, `primaryFreightClass` and `highestFreightClass`,
since that is how rates are written and how the schema documents it. Check that rate-matrix keys
for freight class use the same form (`rateengine/matrix.go` dimensions) and align them.

Acceptance: with commodities of class `Class85` and `Class100`, `highestFreightClass == "100"`,
`primaryFreightClass` is the bare number of the heaviest line, and
`any(commodities, .freightClass == "85")` is true.

### 2.2 Canceled stops and load order — high, reported

`computeTotalStops` and `orderedStops` count every stop in slice order, canceled ones included
(`resolver/computed.go:57-66`, `resolver/walk_memo.go:40-73`); the rate engine's own `countStops`
excludes `StopStatusCanceled` (`rateengine/build.go:276-292`). `origin`/`destination` take the
first and last stop of any type (`computed_lane.go:25-39`), while the schema says "first pickup
stop" / "final delivery stop" and the domain has `FirstShipperStop`/`LastConsigneeStop` that pick
by type and sequence (`operations/domain/shipment/shipment.go:369-435`). On the save path stops
arrive in the order the client sent them.

Fix: one ordered, non-canceled stop walk (sorted by move sequence, then stop sequence) used by
every stop-derived field; origin/destination and pickup/delivery dates from the domain's
origin/destination rules. `stops[]` keeps canceled stops (with their status) so a formula can
still see them, but in sequence order.

Acceptance: a shipment with stops P1, D1(canceled), D2 sent in order D2, P1, D1 gives
`totalStops == 2`, `origin` = P1, `destination` = D2, and the standard "Additional Stop Fee"
template does not charge for D1.

### 2.3 `pieces` reaches the VM as `*int64` — high, reported

`Shipment.Pieces` is `*int64`; the schema gives it no transform and `resolvePath` returns non-nil
pointers as is (`resolver/resolver.go:163-167`). The compile shape uses `int64(0)`
(`engine/engine.go:760-785`), so no deref is compiled. `pieces * rate` errors, `max(pieces, 1)`
is rejected by `toFloat64`, and `pieces == 10` is silently false.

Fix: `resolvePath` dereferences non-nil pointers to scalars (covers every pointer field, not just
`pieces`); keep transforms for the ones that convert type.

### 2.4 Nullable relation objects are never nil — medium, reported

`Build` writes `tractorType = {code: nil, description: nil}` when the shipment has no tractor
type, so `tractorType == nil ? 0 : fee` always charges the fee.

Fix: when a nullable relation is absent, the environment holds `nil` for the object. The compile
shape keeps the object's map of declared zeros (so `tractorType?.code ?? ""` compiles), and the
runtime value is nil. `describe.go` and the client mark the relation's fields nullable.
`UnguardedNullableFields` warns for nested paths too: probe by setting the parent object to nil
(nested members are typed "unknown" by expr, so the current per-leaf nil probe can never fail
compilation — `conf/env.go` types only top-level keys).

Acceptance: `tractorType == nil ? 0 : 50` is 0 without a tractor type and 50 with one;
`hasPrefix(tractorType.code, "DC") ? 1 : 0` on save raises an unguarded-nullable warning
suggesting `tractorType?.code ?? ""`.

### 2.5 Formula aggregates disagree with the rate engine — medium, reported

`totalWeight`/`totalPieces` prefer the shipment header then fall back to commodities
(`resolver/computed.go:110-150`); the rate engine prefers the commodity sum
(`rateengine/build.go:231-274`). `totalCubicFeet`/`density` ignore the load envelope the engine
prefers (`build.go:316-335`). `hasHazmat` needs the `HazardousMaterial` relation
(`computed.go:68-87`); the engine checks the ID (`build.go:410-421`). A rule can be matched on
one weight and priced on another.

Fix: one shared implementation of each aggregate (weight, pieces, cube, density, hazmat, stop
count), used by both the rate engine's context and the formula's computed fields. Put it where
both modules may import it.

### 2.6 Contract accessorial conditions never see provided values — medium, reported

`formula.Service.EvaluatePredicate` (`service.go:497-512`) passes no `Provided` and no
`TenantInfo`, so `fuelPrice`, `fuelPriceDate`, `fuelIndexCode` are nil; `fuelPrice > 4` errors and
the accessorial is withheld with a warning log (`shipmentcommercial/agreementaccessorial.go`).

Fix: pass the tenant and `ContextVariables(ctx, tenant)` exactly as `EvaluateExpression` does.

### 2.7 `currentTotalCharge` is stale while rating — low, reported

It reads the stored `TotalChargeAmount` (`resolver/computed.go:392-394`), which `Recalculate`
writes only after the formula runs: 0 on create, the previous save's total afterwards. Fix:
describe it accurately in the schema ("the total stored before this rating"), and have
template validation warn when a freight-charge template references it.

### 2.8 Smaller contract fixes — low, reported

- `deliveryDate` is empty whenever the last stop has neither actuals nor `ScheduledWindowEnd`
  (`computed_lane.go:126-138`); the schema says "only when no stops". Fix the description or
  fall back to `ScheduledWindowStart`.
- `fuelPriceDate` is documented as `YYYY-MM-DD` but scanned from a DATE column into a string
  (`fuelsurcharge/price.go:31`). Pin the format with a test and normalise it.
- "Percent of Sell Rate" gets a real `sellTotal` only through `rateengine/pricing.go:125-129`; a
  matrix rule (`matrix.go:230-236`) or the fallback template gets 0 and bills $0. Supply it on
  every path or refuse that template where it can't be known.
- `internal/testutil/formulaschema.go` claims it "cannot drift" and has: it lacks `baseRate`,
  `status`, `serviceType`, the lane fields and more. Embed the real `shipment.schema.json`.
- Client parity: `SHIPMENT_VARIABLES` and `DEFAULT_TEST_VALUES` lack `fuelPrice`,
  `fuelPriceDate`, `fuelIndexCode`, `serviceType.*`, `shipmentType.*`, `totalCubicFeet`,
  `density`, `primaryFreightClass`, `highestFreightClass`, `stops`, `commodities`,
  `pickupDate`, `deliveryDate`; the sample sets temperatures 34/40 with
  `requiresTemperatureControl: false` and `temperatureDifferential: 0`. Better than patching the
  lists: have the client read variables from the server's `DescribeSchema` and delete the
  hand-kept list, keeping only sample values.

### Done

Every item was reproduced on a real `shipment.Shipment` first
(`formula/data_contract_test.go`); all reported items held.

- **One implementation of every aggregate (2.5).** The shipment domain carries the rating
  facts (`operations/domain/shipment/ratingfacts.go`): ordered and rated stops, stop count,
  distance, weight, pieces, linear feet, cube, density, hazmat, freight classes. The rate
  engine's context and every formula computed field read them, so a rule is matched and
  priced on the same numbers. Where the two sides disagreed the engine's meaning won, because
  it is what contracts are matched on: weight and pieces prefer the commodity lines over the
  header, cube prefers the load envelope over the lines, hazmat is a hazardous material on a
  commodity by ID or by record. A formula that read `totalWeight` from the header on a load
  whose lines disagree now prices from the lines.
- **Typed computed fields.** The formula's computed fields read `*shipment.Shipment` directly
  instead of walking an `any` by reflection, and refuse any other entity
  (`resolver.ErrUnsupportedEntity`). The reflection helpers and the tests built on look-alike
  structs are gone; `resolver/computed_test.go` covers every computed field on the real type.
- **Freight class (2.1).** `commodity.FreightClass.Number()` and `ParseFreightClass`: formulas
  see `"85"`, `"92.5"` in `freightClass`, `primaryFreightClass` and `highestFreightClass`. Rate
  matrices keep their stored `Class85` keys; a formula lookup on a freight-class axis reads
  either form, so `lookup(t, highestFreightClass)` finds the cell without migrating data.
- **Stops (2.2).** Stops are ordered by move then stop sequence, whatever order the client sent.
  `totalStops` and `totalHours` skip canceled stops; `stops[]` lists them with their status.
  `origin`, `destination`, `pickupDate` and `deliveryDate` come from the domain's
  `ShipperStop`/`DeliveryStop`.
- **Pointers and named types (2.3, and 3.1's conversion).** A path value is dereferenced and
  converted to its plain Go kind (`*int64` → `int64`, `shipment.Status` → `string`); transforms
  still receive the dereferenced raw value.
- **Absent optional relations (2.4).** An absent nullable relation is `nil` in the environment
  (`tractorType == nil`), the compile shape keeps the object's declared zeros so
  `tractorType?.code ?? ""` compiles, and reading a member of the absent object fails at run
  time rather than reading empty. A leaf under a nullable object is nullable in the compile
  shape, the unguarded-field check and `describe.go`. The check probes a nested member by
  running the expression with its parent nil, because expr types nested members as unknown and
  a compile probe can never fail; its suggestion is `tractorType?.code ?? ""`.
- **Predicates (2.6).** `EvaluatePredicateRequest` carries the tenant, and conditions see the
  tenant's provided values exactly as rating does.
- **`currentTotalCharge` (2.7).** The schema says what it is. The preview warns wherever it is
  read, and approval readiness warns when a freight charge template reads it (under "Safe on
  every shipment", so the client needed no new check).
- **2.8.** `deliveryDate` falls back to the delivery's scheduled window start, and both date
  descriptions name the pickup and delivery stops. `fuelPriceDate` is normalized to `YYYY-MM-DD`
  (`fuelsurcharge.NormalizePriceDate`) whatever the DATE column scans to. `sellTotal` is bound
  on every path that knows it (agreement rules and matrix cells on the buy side); every other
  path declares it unavailable (`CalculateRequest.UnavailableInputs`), and a template that reads
  it there is refused with a validation error instead of billing $0. `testutil`'s schema and
  the copies in the template service, the template handler and the shipment service tests now
  register the embedded schema (`formula.ShipmentSchemaJSON`); the drift hid a template variable
  named `baseRate` colliding with the schema field, which those tests now avoid (the collision
  itself is WS5.6). The client's sample values cover every schema variable and agree with each
  other, pinned by a test that reads `shipment.schema.json`. Replacing the client's hand-kept
  variable list with the server's is done with WS6.3, which needs the same loading state.

---

## WS3. Data-contract defects that fail loudly where the preview passes

### 3.1 `status` is a named Go type — high, reported

`resolvePath` returns `field.Interface()`, i.e. `shipment.Status`, and `status` is a top-level
key, so expr types it as `shipment.Status`. `status == "New"` and `status in ["A","B"]` fail to
compile ("mismatched types") on real shipments — `nature.ComparableTo` falls through to
`AssignableTo(shipment.Status → string)`, which is false — while the preview's `""` compiles.
The schema's enum also lists `Billed` and `Cancelled`; the domain has `ReadyToInvoice`,
`Invoiced`, `Canceled` (`operations/domain/shipment/enums.go:5-16`), and `describe.go` hands the
wrong list to the editor and the agents.

Fix: `resolvePath` converts any value whose kind is string (bool, int, float likewise) to the
plain Go type; correct the enum from the domain constants, and add a test that fails when the
schema enum and the domain constants differ.

### 3.2 `temperatureDifferential` is nil on every dry-van load — high, reported

`computeTemperatureDifferential` errors when either temperature is missing
(`resolver/computed.go:99-108`); the schema declares it a non-null number; as a top-level nil it
fails compilation of any formula that uses it, even
`requiresTemperatureControl ? temperatureDifferential * x : 0`. Fix: return `0.0` without an error
when either temperature is missing (the schema already describes it as a number).

### 3.3 Helper parses use a different parser configuration — high, confirmed

`ExtractLookupTableRefs` (`engine/lookup.go:170`), `referencedPaths` (`engine/nullable.go:157`) and
`splitStatements` (`engine/statements.go:106`) call `parser.Parse(expression)` with no config.
The compile path uses `BuiltinFunctions()` (`engine/engine.go:97-101`), which overrides `sum` and
other builtins. Under the bare parser `sum` is expr's predicate builtin, so expressions the
compiler accepts may not parse in the helpers. Reported effects: save-time lookup-table
validation passes silently (`formula/service.go:572-579`), the preview loads no rate tables
(`testexpression.go:208-219`), and statement traces, nullable warnings and missing-field
explanations go blank.

Fix: every parse goes through one function that applies the engine's parser configuration.
Acceptance: `sum(lookup("fsc", fuelPrice) * linehaul, stopFee, 25)` (with a real `fsc` table)
validates its table reference on save, previews with the table loaded, and traces its
statements.

### Done

`formula/loud_failures_test.go` and `formulatemplateservice/lookup_preview_test.go`.

- **3.1.** `status` reaches the VM as a plain `string` (the WS2 conversion of named kinds);
  without it `status == "InTransit"` fails to compile with "mismatched types shipment.Status and
  string", which the test reproduces when the conversion is taken out. The schema's enum is the
  domain's ten statuses, pinned by a test against `shipment.Stages()` (whose own test already
  requires every status to have a stage).
- **3.2.** `temperatureDifferential` is `0.0` when either temperature is missing; done with
  WS2's typed computed fields, pinned here by
  `requiresTemperatureControl ? temperatureDifferential * 2 : 0` on a dry van and a reefer.
- **3.3.** Every parse goes through `engine.parseExpression`, configured from `languageOptions()`,
  the same function set and node limit the compiler uses (`NewEngine` builds its options from
  it). `ExtractLookupTableRefs`, `ExtractLookupTables`, `SplitStatements` and `referencedPaths`
  all use it: save-time table validation, the preview's table loading, statement traces and the
  missing-field and unresolved-field checks no longer go blank on `sum(a, b, 25)`.

---

## WS4. Engine hardening and money arithmetic

### 4.1 Lookup recorder data race — high, confirmed

`lookupRecorder` appends to `entries` with no lock (`engine/receipt.go:33-111`). `run` abandons
the VM goroutine on timeout (`engine/engine.go:420-471`) and that goroutine can keep calling the
injected lookup closures, while the same recorder is reused for the next breakdown line
(`engine.go:383-418`). `run` also shallow-copies the environment, so nested maps are shared with
the abandoned goroutine.

Fix: a recorder per evaluation (main expression and each breakdown line), merged into the
receipt only after that evaluation returned normally; a timed-out evaluation's recorder is
discarded. Deep-copy nested maps into the VM's environment, or make the environment immutable
after build. Acceptance: a `go test -race` test with a breakdown that times out while making
lookups and a second breakdown that makes lookups passes.

### 4.2 Resource limits — medium, confirmed

`NewEngine` sets only `BuiltinFunctions()`, `MaxNodes(1000)` and `WithContext`
(`engine/engine.go:97-101`). There is no explicit memory budget, no builtin is disabled, and
nothing bounds concurrent evaluations; a timed-out `vm.Run` cannot be interrupted, so abandoned
goroutines accumulate under load. expr does not count string concatenation or accumulator loops
against its memory budget.

Fix:
- Set `expr.MemoryBudget` explicitly.
- Bound concurrent evaluations process-wide with a weighted semaphore sized from configuration;
  an evaluation that cannot acquire a slot within its deadline fails with a timeout error. Count
  abandoned goroutines against the bound until they actually finish, so a burst of timeouts
  cannot grow past it.
- Remove or cap string-growing builtins (`repeat`, and string concatenation if a patch can
  bound it) so a single evaluation's allocation is bounded.
- Metrics: evaluations in flight, abandoned goroutines, timeouts.

Formula evaluation stays in-process. The review found no escape surface a separate process,
WASM or a Temporal activity would close (no I/O, no syscalls, the environment holds only plain
values), and those would cost the per-keystroke preview and per-shipment rating their latency.

### 4.3 Non-deterministic builtins — medium, confirmed

All expr builtins are enabled, including `now()`, `date()`, `duration()`, `toJSON`, `fromJSON`;
`guardFor` even suggests `coalesce(field, now())` for a nullable date (`engine/nullable.go:91`).
Re-rating, backtests and audits can't reproduce a bill.

Fix: disable `now` and the JSON builtins with `expr.DisableBuiltin`. Before disabling `date` or
`duration`, query stored templates, versions and accessorial conditions for them; if any use
them with constant arguments, keep those two. If a formula needs "now", supply a fixed
`ratingTime` provided variable (the rating date the engine already uses). Change the datetime
guard suggestion to a field that exists (`pickupDate`) or `ratingTime`. Save-time validation
rejects a disabled builtin with a message that names the replacement.

### 4.4 Money arithmetic in float — high, reported

expr computes in float64. Decimals become float on entry (`resolver/transforms.go:39-65`),
lookups return float64, `round` converts float → decimal → float (`engine/functions.go:30-95`),
`sum`/`avg` accumulate in float (`functions.go:149-174`), and the result becomes a decimal only
at the end (`engine/engine.go:557-605`). `round(0.145 * 100)` gives 14 because the product is
14.499999999999998.

Fix (within expr; the long-term answer is a decimal evaluator, out of scope here):
- Wherever a float becomes a decimal — the `round` family, `roundTo`, the final `toDecimal` —
  convert through 15 significant digits (`strconv.FormatFloat(v, 'g', 15, 64)`), which removes
  binary noise without changing any value a person could have meant.
- `sum` and `avg` accumulate in `decimal.Decimal`.
- Breakdown lines are rounded with the template's rounding policy, and the response carries a
  rounding-remainder line so the lines sum to the billed total.

Acceptance: `round(0.145 * 100)` = 15, `round(1.005, 2)` = 1.01, `0.1 + 0.2` bills 0.30,
`sum([0.1, 0.2, 0.3])` = 0.6, and breakdown lines plus remainder equal the total for a
three-line template.

### 4.5 Smaller engine items — low, reported

- Panics during environment build (reflection in the resolver) are outside any `recover`
  (`engine/environment.go:50-73`, `resolver/resolver.go`). Recover there and return a validation
  error.
- Client-facing errors include Go type names (`*shipment.Shipment`) and internal paths
  (`resolver/resolver.go:76,107`, `engine.go:662-679`). Strip them from what reaches the API;
  keep them in logs.
- The receipt records each lookup's value but not the rate-table stamp
  (`ratematrixrepository/stamp.go`). Stamp the receipt.
- Breakdown lines skip the boolean-result rejection the main expression has
  (`engine.go:383-418` vs `engine.go:217-223`). Apply it.
- Decide whether a negative amount with no minimum charge is allowed; if not, reject it in
  `ApplyChargePolicy`.

### Done

Tests: `engine/abandoned_lookup_test.go`, `engine/resource_limits_test.go`,
`formula/determinism_test.go`, `formula/money_test.go`, `formula/hardening_test.go`,
`metrics/formula_test.go`, `decimalutils/float_test.go`, the readiness remainder case in
`formulatemplateservice/studio_test.go`, and the `disabled-function` cases in the editor's
`expr-check.test.ts`.

- **4.1.** `Engine.run` makes a recorder per evaluation (the main expression and each breakdown
  line) and returns its traces only when the VM returned; a timed-out run's recorder is dropped
  with its goroutine. The VM gets a deep copy of the environment (`maputils.CloneDeep`), so an
  abandoned goroutine shares nothing with the next line. The test reproduced the race (lost
  and foreign traces in the receipt) before the change and passes under `-race` after it.
- **4.2.** Every run executes on its own `vm.VM` with `MemoryBudget` 100,000 (expr 1.17 sets the
  budget on the VM, not through an option). A process-wide weighted semaphore, sized by
  `formula.maxConcurrentEvaluations` (default 64), admits evaluations; one that cannot get a
  slot before its deadline fails with `ErrEvaluationTimeout`, and an abandoned goroutine keeps
  its slot until it really finishes, so a burst of runaway formulas queues callers rather than
  piling up goroutines. String growth is metered per evaluation (1 MiB): `+` on text is patched
  to a metered concatenation (only where both sides may be text, so `"lane" + 5` stays a compile
  error), and `replace`, `join`, `string`, `toBase64`, `upper` and `lower` wrap expr's own
  implementations with the meter. `repeat` is disabled (4.3). Metrics:
  `trenova_formula_evaluations_in_flight`, `trenova_formula_evaluations_abandoned` and
  `trenova_formula_evaluation_timeouts_total{stage}`, documented in the metrics README.
- **4.3.** `now`, `toJSON`, `fromJSON` and `repeat` are disabled. A formula that calls one fails
  to compile with a message naming the replacement (`now()` → `ratingTime`), and the editor
  flags the same calls (`disabled-function`, with a one-click `ratingTime` fix for `now()`).
  `date()` and `duration()` stay: `date()` without a zone already parses in UTC (the test pins
  it), so neither depends on the server or the clock. `ratingTime` is a provided variable on
  every evaluation: the rating date the request carries, else the shipment's actual ship date,
  else when it was entered (`Shipment.RatingAsOf`, which the rate engine now uses too), and the
  receipt records it. The nullable-date guard suggests `coalesce(pickupDate, ratingTime)`.
  Stored templates could not be queried from this branch for `date`/`duration` with constant
  arguments; since both stay enabled, nothing stored can break.
- **4.4.** Floats become decimals through 15 significant digits (`decimalutils.FromFloat`) in the
  round family, `roundTo` and the final conversion, and `sum`/`avg` accumulate in decimal: all
  the acceptance cases hold, plus `1.15 * 100` = 115. Breakdown lines are rounded with the
  template's policy (`formula.RoundBreakdown`, on the billing path and both previews), and when
  the unrounded lines reconcile with the formula a `roundingRemainder` line ("Rounding") makes
  them sum to the bill. Lines that do not reconcile get no remainder: a missing line is not
  rounding drift. Readiness compares the lines with the rounded total.
- **4.5.**
  - A resolver panic during environment build is recovered per field and recorded as that
    field's failure even when it is nullable. A formula that reads the field gets the
    unresolved-field validation error (422), and one that does not still prices.
  - Messages no longer carry Go types or struct paths. `ResolveError`/`ComputeError` drop the
    entity type from `Error()` but keep it on the struct, and the unresolved-field note lists
    formula fields only. Values are described in words ("got a list"), and expr's own type
    tokens (`[]interface {}`, `float64`, `time.Time`) are humanized. Panics carry a
    `PanicError` with the value and stack. `formulaerrors.LogFields` puts the entity type, panic
    value and stack back into the logs.
  - The receipt carries `rateTableStamp`, the stamp the lookup was built under, whenever the
    formula read a table.
  - Breakdown lines already refused true/false since the 4.1 refactor, and a test now pins it.
  - Negative amounts: refused. Both template types price charges, and credits are separate,
    positive adjustments, so a negative charge with no minimum is a formula bug.
    `ApplyChargePolicy` returns a validation error naming the amount and the fix (`max(..., 0)`
    or a minimum charge). With a minimum charge the clamp stands, and zero is allowed.
- The i18n extractor did not know `NewUnresolvedFieldError` (WS1) or `NewPanicError`, which made
  `task i18n` fail on this branch. Both are listed under `noMessage`, and the catalogs carry the
  new editor messages and WS1's untranslated strings.

---

## WS5. Template lifecycle

All under `services/tms/internal/modules/rating/app/formulatemplateservice/` unless stated.

### 5.1 Editing a live template stops billing — high, confirmed

A material edit (or rollback) of an Active template sets it to Draft (`service.go:198-205`,
`versions.go:195-201`); `formula.Service.Calculate` refuses any non-Active template
(`formula/service.go:120-126`); the fallback rating path returns that error on purpose
(`rateengine/service.go:262-272`), and `Recalculate` runs on every shipment save, move update and
assignment. Every save on a shipment rated by that template fails with a 422 until someone else
approves; a reject moves it to Inactive and billing stays broken. Archiving through
`BulkUpdateStatus` (`service.go:345-408`) needs only update permission and checks no usage.

Fix — the approved content keeps billing while a revision is pending:
- `Calculate` rates a Draft or InReview template with its latest approved snapshot
  (`GetLatestByStatus(Active)` on the version repository — `backtest.go:242-247` already resolves
  that baseline) when one exists; a template that was never approved still refuses.
- Reject and RequestChanges on a template that has an approved snapshot return it to Draft, not
  Inactive. Inactive means archived and never bills.
- Archiving a template that is in use (see 5.6 for counting usage) requires approve permission
  and an explicit confirmation listing what uses it.
- The UI and API say which content is billing ("Billing v5 while v6 is in review").

Acceptance: editing an Active template's expression leaves `Recalculate` pricing with the
previous expression until the edit is approved, then with the new one; rejecting the edit keeps
the previous expression billing.

### 5.2 Effective dating does not protect history — high, reported

Rating for a date with no scheduled version uses the current row (`formula/service.go:189-217`,
`effectiveversioncache.go:122-141`, version repository `formulatemplateversion.go:476-511`).
Approve never stamps `EffectiveFrom` on the new snapshot (`approval.go:127-152`), so approving v6
re-prices every unlocked historical shipment the next time anything touches it.
`ClearScheduled` nulls every `effective_from`, including windows already in force, and
unscheduling a version that already took effect is unchecked (`effectivedate.go:46`).

Fix:
- Approval closes a window: the new snapshot is effective from approval time (or a chosen future
  date); earlier snapshots keep the past.
- Rating resolves the snapshot effective at the rating date; a shipment dated before the first
  window takes the earliest approved snapshot once that window has started. Until a template's
  first approved window starts (an approval deferred to a later date), nothing bills and rating
  refuses, naming the date.
- `ClearScheduled` clears only future dates (`effective_from > now`); unscheduling a window that
  already started is refused.
- `RateQuote` records the formula template version alongside the agreement version
  (`rating/domain/ratequote/quote.go:55-56`).
- **Decision (product owner, while WS5 was in progress): lock when the invoice posts.** Nothing
  in this repository set `RateLocked` to true, and nothing elsewhere protects an invoiced
  shipment. Posting an invoice now sets `RateLocked` on every leg it bills, in the same
  transaction; a credit-and-rebill or a void that reopens the shipment clears it, and a leg
  another posted invoice still bills stays locked.

### 5.3 Stale schedules override approved content — high, reported

Update clears schedules only when the original was Active or InReview (`service.go:199-201`);
`BulkUpdateStatus` (`service.go:390`) and Rollback (`versions.go:195-201`) never clear — the
comment at `versions.go:21-24` warns about exactly this. A version scheduled long ago keeps
winning `EffectiveAsOf` after newer content is approved.

Fix: every transition out of Active clears future schedules; ReviewDiff and Readiness list
pending schedules.

### 5.4 Approval controls — medium, reported

- Four-eyes compares submitter and approver, not author and approver (`approval.go:108-115`,
  `reviews.go:73`): the author of a material version can approve it if someone else submitted.
  Refuse approval by anyone who created a material version in the round.
- Scheduling an old Active snapshot (`effectivedate.go:46-73`) re-validates nothing. Re-validate
  its expression and lookup tables and run its scenarios, as Rollback does
  (`versions.go:183-187`).
- Reject, RequestChanges (`approval.go:191`, `reviews.go:62`) and stale-submission expiry
  (`reviews.go:192-202`) call `approvals().Apply` outside a transaction, unlike Submit and
  Approve. Wrap each in one transaction with its schedule clearing and review row.
- `BulkUpdateStatus` checks status then updates without a status predicate or version bump
  (`service.go:362-390`, repository `formulatemplate.go:411-451`). Add both.
- Standard catalog installs land Active with no scenarios and need only create permission
  (`standards.go:102-112`), skipping the one-scenario rule (`testcases.go:560-566`). Install as
  Draft, or install with the catalog's scenarios and require approve permission to activate.

### 5.5 Audit — medium, reported

`service.go:466-497`: entries are never `WithCritical()` or `WithRequest(ctx)`; the audit write
happens after commit and its failure is only logged; scenario update and delete are not audited
(`testcases.go:106-182`); the effective-date audit records only the new value
(`effectivedate.go:81-92`). Make approve, reject, schedule, rollback, archive and material edits
critical with request metadata, audit scenario changes, and record old and new values. Follow
`docs/engineering/security-audit.md`.

### 5.6 Validation and cost bounds — medium, reported

- Variable definitions (`rating/domain/formulatemplate/formulatemplate.go:110-116`): validate
  name pattern, uniqueness, reserved names, schema-field shadowing, the type enum, and that the
  default matches the type. `applyVariableDefaults` (`engine/engine.go:543-555`) silently ignores
  a variable that collides with a schema field; make that a save-time error.
- Fork (`versions.go:242-314`) validates nothing: run the same validation as create.
- The preview endpoint (`testexpression.go:90-142`, `formulatemplatehandler/expression.go`) does
  not enforce the expression length or breakdown count caps. Enforce them.
- Description and Metadata have no size limit; bound them.
- `CountUsages` (`formulatemplaterepository/formulatemplate.go:569-637`) ignores
  `BillingControl.FallbackFormulaTemplateID` (`rateengine/service.go:302-308`). Count it.
- `/impact` needs only read and runs up to 500×2 evaluations with no overall deadline
  (`impact.go:39-81`); backtest evaluations don't get the preview timeout. Give both an overall
  deadline and the WS4.2 concurrency bound.

### 5.7 Smaller lifecycle items — low, reported

- `shipment.formulaTemplate` and `ratingDetail.expression` expose expression and variables to
  anyone with shipment read (`api/graphql/resolver/base/shipmentmapping.go:82`). Require
  formula-template read for those fields.
- `ListReviewers` returns approvers' emails to anyone with template read (`reviewers.go:68-77`).
  Return names only.
- ListConnection, CountUsages and SelectOptions run inside `dbtx.Write`
  (`formulatemplate.go:165,573,644`). Use `dbtx.Read`.
- `DeleteTestCase` hand-writes `ftc.*` columns
  (`formulatemplatetestcaserepository/formulatemplatetestcase.go:112-117`). Use buncolgen.
- Backtest's "current" side recomputes with today's tables (`backtest.go:198-211`); compare
  against the stored billed amount as its comment says.

### Done

Tests: `formula/lifecycle_test.go`, `effectiveversioncache_test.go`, and in
`formulatemplateservice/` `approval_rounds_test.go`, `audit_test.go`, `effectivedate_test.go`,
`impact_test.go`, `notifications_test.go`, `service_test.go`, `studio_test.go`,
`testcases_test.go` and `standards_test.go`. Also the domain's `formulatemplate_test.go`,
`formulatypes/variable_validation_test.go`, `rateagreementservice/template_reference_test.go`,
`rateengine_test.go`, billing's `invoicevoid/release_test.go` and
`invoiceservice/mark_invoiced_test.go`, the formula template handler's `handler_test.go`, and
GraphQL's `shipmentresolver/formulavisibility_test.go`. Integration tests
(`formulatemplaterepository/usage_integration_test.go`,
`formulatemplateversionrepository/billing_integration_test.go`,
`formulatemplatetestcaserepository/delete_integration_test.go`) were vetted with
`-tags integration` but not run here, as this environment has no Docker. Client tests:
`archive-refusal.test.ts`, `studio/model/__tests__/billing.test.ts`,
`version/__tests__/schedule-state.test.ts` and `backtest-export.test.ts`.

The model behind all of it: an approved snapshot (version status Active) bills from its start,
`EffectiveFrom` if it was scheduled and its approval time otherwise
(`FormulaTemplateVersion.EffectiveStart`). Rating resolves the snapshot in force at the rating
date (`formula.Service.ResolveBillingVersion`, `effectiveversioncache.EffectiveAsOf`); before
the first window it uses the earliest approved snapshot once that window has started; while a
template's only approved windows lie ahead, rating refuses and names the start date
(`BillingResolution.Scheduled`; GET `billing.pending`). The template row is what is being
edited, never what bills, unless the template is Active and has no approved snapshot at all
(templates approved before snapshots existed).

- **5.1.**
  - A Draft or InReview template bills with its approved snapshot. One that was never approved
    refuses ("has never been approved"), and an archived one refuses as archived.
  - Reject returns a template that has an approved snapshot to Draft, and archives only one that
    never had any. The notification and the dialog say which happened.
  - Archiving an Active template that something uses (5.6 counts usage) needs approve
    permission. The handler checks it and the service refuses without it. It also needs
    `confirmInUse`: without it the 422 names each template and what uses it, and the list page
    shows that message and offers "Archive anyway".
  - `GET /formula-templates/:id` carries `billing` (the version and its start), and the studio
    shows "Billing v5 while v6 is in review", "… while v7 is a draft" or "… until v6 takes
    effect".
  - Rate agreements accept a template that is Active or has an approved snapshot.
- **5.2.**
  - Approval closes the previous window: the new snapshot bills from approval time, or from a
    chosen future start (`effectiveFrom` on approve, at most an hour in the past). The approve
    dialog offers the date.
  - `ClearScheduled` touches only pending windows (`effective_from > now`) and retires them to
    Inactive instead of nulling dates.
  - On an approved snapshot, `UpdateVersionEffectiveDate` works as follows:
    - Clearing a pending window starts it now.
    - Unscheduling a window that already started is refused ("already billed shipments").
    - A new date on a started window approves the same content again as a new snapshot that
      starts then, after validating it and running its scenarios, so history keeps its prices.
  - `RateQuote.formulaTemplateVersionNumber` records the snapshot that priced the linehaul
    (migration `20261231009250`).
  - The lock follows the decision recorded under 5.2.
- **5.3.** Every transition out of Active clears pending windows: archive (`BulkUpdateStatus`),
  rollback to Draft, and material edits, which already did. ReviewDiff and Readiness list
  `pendingSchedules`.
- **5.4.**
  - Approval is refused to anyone who authored a version created since the last approved one
    (`ListAuthorIDsSince`), not only to the submitter.
  - Moving a started window re-validates and re-runs scenarios, as above. A pending window's
    content was validated when it was approved and has billed nothing, so moving it does not
    re-run them.
  - Reject, RequestChanges and stale-submission expiry each run in one transaction with their
    review row and schedule clearing.
  - `BulkUpdateStatus` updates only rows still Active or Inactive and bumps `version`.
  - Standard templates install as Draft. The catalog's "Per Hour" declared an input named
    `totalHours`, which shadows the shipment value (5.6), so it is now `billedHours` and falls
    back to `totalHours`.
- **5.5.**
  - Approve, reject, schedule, rollback, archive and material edits are audited as critical
    entries with request metadata.
  - Scenario update and delete are audited with the scenario before and after.
  - The effective-date entry records `previousEffectiveFrom`.
  - Audit writes still follow the commit, as everywhere else in the service. A critical entry
    is inserted directly and, if that fails, buffered and retried, so a failure no longer only
    shows up in the log.
- **5.6.**
  - Variable definitions are checked on save (`formulatypes.ValidateVariableDefinitions`):
    name pattern, duplicates, type (case-insensitive, stored canonical) and a default of the
    declared type. A name that is a formula function, or that shadows a schema field or
    provided value, is refused.
  - Fork runs create's validation.
  - The preview enforces the expression length (10,000) and breakdown count (20) caps.
  - Description is capped at 2,000 characters and metadata at 16 KiB.
  - `CountUsages` counts the billing-control fallback.
  - `/impact` needs create, update or approve.
  - Backtest and impact share one runner: a 60-second overall deadline, the preview timeout
    per evaluation, `truncated` when the deadline cut the run short, and the WS4.2 slots
    through the engine. The UI says when a run was truncated.
- **5.7.**
  - `Shipment.formulaTemplate` and `ShipmentRatingDetail.expression` resolve only for a
    reader with formula template read; anyone else gets null and an empty string.
  - Found while fixing this and not changed here: REST `GET /shipments/:id?expandShipmentDetails=true`
    (and the list, create and update responses that load details) still return the template
    relation and the rating detail's expression to anyone with shipment read. That needs the
    same check on about ten handlers and is left for a follow-up.
  - Reviewers are listed by name and username, without email.
  - ListConnection, CountUsages and SelectOptions run in `dbtx.Read`.
  - `DeleteTestCase` uses the generated tenant scope and columns. This is a refactor with no
    behaviour to fail first; the integration test pins the template and tenant scoping.
  - Backtest's current side is the amount the shipment was billed (`currentSource: Billed`):
    its stored freight charge, when its rating detail names this template and no hand-set rate
    replaced it. Otherwise the billed content is rated again (`Recomputed`). Approval impact
    always recomputes, so a reviewer sees what the content change does, separate from table
    drift since billing.

---

## WS6. Language conformance between editor and server

The editor re-implements the language in TypeScript (`client/apps/web/src/components/
formula-editor/expr-parser.ts`, `expr-check.ts`, `expr-format.ts`, `expr-language.ts`,
`known-identifiers.ts`, …). It and the server disagree on the formulas below (all reported).

### 6.1 Shared conformance corpus

A JSON corpus of cases — expression, expected parse result, expected compile result against a
fixed schema environment, expected value or error — read by a Go test in the formula engine and
a vitest test in the formula editor. Every divergence below becomes a case. CI runs both, so a
change to either side that breaks agreement fails.

### 6.2 Divergences to fix

The editor rejects what the server accepts (fix in TS):
1. Chained comparison `0 < totalWeight <= 500` (expr rewrites to `&&`).
2. Digit separators and hex/octal/binary literals (`10_000`, `0x1F`).
3. Named pointers `#acc`, `#index` in `reduce`/`map`.
4. `let` inside parentheses, `if` blocks and predicates.
5. Optional index `stops?.[0]`.
6. Trailing comma in a call `max(a, b,)`.
7. Unicode identifiers (`tarifa_año`).
8. Builtins missing from the editor's list: `reverse`, `toPairs`, `fromPairs`, `bitnot`.

The editor accepts what the server rejects (raise to errors in TS):
9. `let` that shadows a field or input (server: "cannot redeclare").
10. `let` named after a builtin or function (`let count = …`, `let max = …`).
11. `%` on a float (`totalDistance % 100`).
12. `==` between different types (`origin.zip == 75201`).
13. Predicate-form `sum(stops, .weight)` (the server's `sum` override turns predicate parsing off).
14. `.field` outside a predicate.
15. `if` as an operand (`base + if hazmat { 50 } else { 0 }`).
16. Pipe without parentheses (`x | round`).
17. Date plus a number (`pickupDate + 1`).

Tooling:
18. Error underline is one column off: expr reports a 1-based rune column (`file/error.go:82`),
    `engineErrorRange` treats it as 0-based (`expr-check.ts:1371-1384`); the fixture at
    `__tests__/expr-check.test.ts:154,163` encodes the wrong column.
19. Raw strings (doubled backticks, no escapes), unescaping string values, and `1.` as a number.

Where a case reveals that the server behaviour is the wrong one for authors (for example 13,
where `sum(stops, .weight)` is natural), change the server instead and record which side moved.

### 6.3 One source for functions

The editor's builtin list, arities and return types (`known-identifiers.ts:60-120`,
`expr-check.ts:178-288`, `989-1009`) come from the server's function specs
(`engine/functionmeta.go`) through `DescribeSchema` or a generated file, not a hand-kept copy.

### 6.4 Language version

Add `language_version` (default 1) to formula templates, their versions, and agreement
accessorial conditions, carried through import/export and the agent tools. Nothing reads it yet;
it is what lets a later decimal evaluator or typed front end change semantics per template
without changing what existing templates bill.

### Done

Tests:
- **Corpus.** `formula/testdata/conformance.json`, read by `formula/conformance_test.go` and by the
  editor's `__tests__/conformance.test.ts`.
- **Schema fixture.** `formula/describe_golden_test.go` keeps the editor's copy of the schema
  description (`testdata/shipment.describe.json`) equal to what the engine publishes.
- **Server shapes.** `engine/functionshape_test.go`.
- **Editor.** `expr-grammar.test.ts` and the updated `expr-check`, `known-identifiers`, `expr-lint`,
  export and starter-value tests.
- **Language version.**
  - `formulatypes/language_test.go`.
  - The language-version cases in the domain's `formulatemplate_test.go`, in
    `rateagreement/accessorial_language_test.go` and in `import_test.go`.
  - The accessorial row case in the agent's `rating_tools_test.go`.

- **6.1.** The corpus has 73 cases. Each states whether the engine saves an expression, with the
  inputs the corpus declares; most also give what it comes to on a fixed real shipment.
  - The engine test runs each case through the save-time validation and, where a value is
    given, through `EvaluateExpression`.
  - The editor test runs the same expressions through `checkExpression` against the engine's
    published schema. It requires an error exactly where the engine refuses.
  - Cases that pin an error position also check that `engineErrorRange` lands where the
    engine's message points.
  - The editor reads the engine's `DescribeSchema("shipment")` from a golden file, which a Go
    test holds equal to the live output. Regenerate it with `-update-description`.
- **6.2.** Every reported divergence was probed against the engine first (expr 1.17.8, the real
  schema). Each is a corpus case, and the editor now agrees with all of them:
  1. Comparisons chain the way the engine chains them, and a parenthesized comparison does not.
  2. The tokenizer reads the engine's number syntax:
     - `_` separators;
     - hex, octal and binary literals;
     - `1.` (as long as it isn't the start of a `..` range).
  3. `#acc` and `#index` parse. `#acc` outside `reduce()` and an unknown named pointer are errors.
  4. `let` parses inside parentheses, `if` blocks and predicates, with its own scope.
     - Redeclaring a name from an enclosing scope is an error.
     - Sibling scopes may reuse a name.
  5. `?.[` parses.
  6. A trailing comma parses.
  7. Identifiers take any Unicode letter, as Go's `unicode.IsLetter` does.
  8. The editor knows every builtin from the engine (6.3).
  9. A `let` that shadows a field or an input is an error (the engine: "cannot redeclare"). It
     used to be a warning.
  10. A `let` named after a function or a builtin is an error.
  11. `%` takes whole numbers only, as the engine does. These count as whole: integer literals,
      `integer` schema fields, whole `let`s, `int()`/`len()`/`count()`-style results, and `+ - * %`
      of whole numbers. A fraction is an error, with an `int(…)` fix.
  12. `==` between kinds the engine types differently is an error. Nested relation fields
      (`origin.zip == 75201`) reach the engine untyped, so it compiles them and they are never
      true. The editor keeps a warning there, and the corpus records why.
  13. **The server stays.** `sum` is overridden to accumulate in decimal and to take several values
      (`sum(a, b, c)`), and expr turns predicate parsing off for an overridden builtin. Giving
      `sum` a predicate would break every stored `sum(a, b, c)`. The editor refuses
      `sum(stops, .weight)` and offers `sum(map(stops, .weight))`.
  14. `.field` outside a predicate is an error.
  15. A bare `if` operand is an error; a parenthesized one, a call argument or a ternary branch
      is fine, as in the engine.
  16. A pipe without parentheses is an error, with an "Add ()" fix.
  17. A date plus or minus a number is an error. A date plus a duration, and a date minus a date,
      are not.
  18. The error position bug is confirmed. The engine's column is 1-based and counts runes;
      `engineErrorRange` converts it, and the fixture that encoded the wrong column is fixed.
  19. Backtick strings take no escapes, and a doubled backtick is a literal one. Quoted strings
      are unescaped (`\n`, `\u…`, `\x…`, octal). `1.` is a number.
  - **The server moved once.** A true/false result passed save-time validation, and then rating
    refused it on every shipment. `validateResultType` now refuses it at save with the rating
    error. Only template validation uses that path; accessorial conditions go through
    `EvaluatePredicate`.
  - The editor previously refused `sum(map(…))` because its hand-kept list marked `sum` as
    numbers-only. It is accepted now.
- **6.3.** `GET /formula-templates/schema` publishes every function's shape: `minArgs`, `maxArgs`,
  `returns`, `numericArgs` and `predicate`.
  - The documented functions derive their shape from their Go signatures when they are
    executable, and declare it otherwise.
  - The expr builtins the engine keeps come in `builtins` (`engine/functionshape.go`). A test
    holds that list equal to expr's own builtins, minus the disabled ones and the ones Trenova
    overrides.
  - The disabled builtins come in `disabled`, with a reason code and a replacement.
  - The editor's hand-kept lists are gone: `SHIPMENT_VARIABLES`, `AVAILABLE_FUNCTIONS`,
    `FALLBACK_SCHEMA`, `EXPR_BUILTIN_FUNCTIONS`, `DISABLED_FUNCTIONS`, and the return-type,
    numeric-parameter, arity and predicate tables. The keyword and operator-word sets now come
    from the parser.
  - Until the schema loads, `ready` is false. The checks report no unknown name, arity or
    disabled-function finding, and the problems panel says names are not checked yet.
- **6.4.** The formula language version is stored and checked; nothing reads it yet.
  - `language_version` (default 1) is on `formula_templates` and `formula_template_versions`, and
    `condition_language_version` on `rate_agreement_accessorials` (migration
    `20261231009260`).
  - Snapshots copy it both ways, forks and imports keep it, and a change of language is a
    material change.
  - An unset version reads as the current one, and a version this server cannot read is refused
    on save and on import.
  - Exports carry it (export format 1.4), and older exports import as version 1.
  - `get_formula_template` returns it with the template. The rate agreement tool names it beside
    an accessorial's condition.

---

## Out of scope

Recorded so they aren't attempted under this spec:
- A Go front end (parser, type checker) compiled to WASM for the editor, replacing the TS parser.
- An owned decimal-only evaluator as language v2, and units of measure as types.
- Moving formula evaluation out of process (separate service, WASM sandbox, Temporal
  activities). WS4.2 covers the resource risks in process.
- Changing `RatingDetail`/`RatingGuardrail` money fields from float64 to decimal and
  `ShipmentRatingDetail.result` from `Float` to `Decimal` in GraphQL; it is a schema change with
  its own migration and client work.

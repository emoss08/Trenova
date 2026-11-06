# Dispatch optimizer

How Trenova decides which driver takes which move: the cost model, the solve, the prices it
publishes and where each piece lives. Read this before changing anything under
`operations/app/dispatch*`, `operations/domain/dispatch*`, `operations/*/demandforecast*`,
`shared/assignmentsolver`, `shared/dispatchplanner` or the dispatch economics settings.

## The model

Dispatch is economics, so every number the optimizer works with is in cents.

- **Pair cost.** Each driver-move pairing has a cost (`dispatchcost.Model`): empty miles,
  pickup and delivery lateness, idle wait beyond the early grace, rest a driver must take
  first, and the end-market value of where the move ends. Those are *hard* components and
  decide coverage. Home time, fairness and the preferences (customer experience, on-time
  history, safety, acceptance, trailer continuity, haul fit, time-off buffer) are *soft*:
  they only choose between drivers. They are normalized per move, so the cheapest soft cost
  among a move's candidates is zero. Missing data is neutral (quality 0.5), never free.
- **Only avoidable lateness counts.** Delivery lateness is measured against the later of the
  window's close and the earliest delivery any driver could make (pickup no earlier than now
  or the window opening, plus dwell and the loaded drive). A load days past its window is not
  priced out of coverage for lateness no driver choice could avoid.
- **The plan's cost.** A pairing costs the plan its own cost less the move's preference floor
  (what its cheapest candidate pays in preferences). That is the number weighed against the
  coverage price, stated in the rationale, stored as `dispatch_plan_prices.cost_cents` with
  `preference_floor_cents`, and shown for every candidate of a planned move, whose breakdown
  ends with "less what the best driver pays in preferences".
- **Coverage price.** Each move has a price for leaving it uncovered:
  `max(revenue × margin %, floor) + urgency premium` (`dispatchcost.CoveragePrice`; Late and
  Now buckets from `dispatchurgency`). A move is covered only when a driver costs less than
  leaving it.
- **Home time.** `HomeOverdueDays` is the days from when the driver was last home to when the
  move frees them, past `HomeTimeTargetDays`; `HomewardShare` is the share of the way home the
  move closes. Last home is the latest of `workers.last_home_at` (set on the worker form, or
  stamped by the telematics poll when a tractor is within `worker.HomeRadiusMiles` of home),
  the end of approved time off in the last 60 days, and the current position when it is at
  home. Home is `workers.home_latitude/longitude`, filled when the worker form picks the
  address from the place lookup and cleared by `KeepHomeWithAddress` when the address changes
  without a new lookup. Unknown home or last home prices nothing.
- **Economics** are per-organization settings (`dispatchcontrol.Economics`, defaults in
  `dispatcheconomics.Default`). Nothing else weighs a pairing: ranking, pre-flight, the plan
  and the agent tool all use the same model.

## The solve

1. **Snapshot** (`dispatchcandidateservice.BuildSnapshot`): sequential, bounded reads of every
   dispatchable driver (`DriverLimit`, dispatchable first), commitments by overlap with ETA
   ends and remaining drive from `stop_etas`, HOS, positions, history. A driver's trip is
   scheduled after the work they already hold; time off and overlap are checked against that
   projected trip (departure to free), not the move's scheduled window. `Prepare` builds one
   profile per driver: origin (commitment, then GPS, then tractor last known), HOS timeline
   carried through committed drive, history signals.
   One move's ranking (the inspector's candidates) reads the `DriverLimit` drivers nearest
   its pickup (`DispatchBoardFilter.Near`), each placed in SQL by the same precedence (the
   drop of their open work, a position under four hours old, the tractor's last drop), so
   the drivers the plan chooses among, its pick included, are the ones the list ranks.
2. **Graph** (`BuildGraph`): a quadtree picks the nearest drivers per move, the top
   `CandidatesPerMove` by cost are evaluated in parallel (`conc` pool), and the top 8 per move
   get road miles from the route estimator. A pair past the deadhead cap, blocked by a rule
   or without a known location is no arc; the nearest such driver is kept in `Rejected` so
   the plan can say why a move stayed uncovered.
3. **Assignment** (`shared/assignmentsolver`): sparse successive shortest paths with
   potentials. Each row has a dummy "uncovered" column priced at its coverage price; free
   columns keep potential 0, so the potentials are proper duals. 2,000 × 3,000 at K=40 solves
   in about 6 ms; the dense Hungarian it replaced took 16.6 s.
4. **Prices** from the same solution:
   - `ColumnValues()`: how much more the plan costs without each driver, for every driver in
     one reverse Dijkstra from the exits (free drivers and uncovered columns); `ColumnValue`
     is the single-driver form and the tests hold the two equal.
   - `OverrideCost(row, col)`: the exact extra cost of forcing a different driver onto a move
     with the rest re-planned around it, checked against brute force on random boards. Each
     search stops once nothing left can beat its best candidate and settles at most 4,096
     nodes: an answer is exact or reported unknown, never approximate.
   - The plan prices alternatives for the six nearest arcs of each covered move in parallel,
     soonest pickup first, within 1.5 s. With spare drivers 5,000 moves price in ~80 ms; a
     5,000 × 5,000 board in full shortage hits the budget and later moves carry no
     alternatives.
5. **Stability.** A move whose last plan chose a driver (`dispatch_plan_prices`) costs
   `ReplanChangeCents` more on any other driver or uncovered, in both modes, so proposals do
   not churn for a few dollars.
6. **Horizon mode** chains several moves per driver with `shared/dispatchplanner`
   (`dispatchplanservice/horizon.go`). The planner seeds from the exact assignment, inserts
   the rest in pickup order, then anneals with relocate, swap, 2-opt* (tail exchange),
   reorder, leave and cover-in-place moves under a wall-clock budget and an iteration cap,
   keeping the best plan, so it is never worse than the seed. A driver is only tried on a
   move through the move's arcs, a chain predecessor (`successorsOf`: moves whose pickup is
   reachable from another's delivery) or the standing proposal; a chain only starts on an
   arc. Chains are priced exactly by a `Timeline` (`horizon_timeline.go`) that installs the
   driver's planned commitments and calls `ScoreCandidate`, so HOS, windows and position
   carry through. Every step is memoised in a prefix trie keyed by (driver, chain prefix,
   move), so a re-evaluated chain costs nothing and only a new suffix reaches the oracle. A
   standing pairing moved to another driver costs `ReplanChangeCents`. Alternatives and driver
   value come from one-step insertion costs against the final plan.

`dispatchplanservice` implements `dispatchplanports.Planner`. The agent
(`dispatchautoassignservice`) only calls the port: it turns pairings into proposals, with a
confidence of `1 − hard/coverage − 0.1 × warnings` and a rationale in dollars.

## Stored plan prices

Every plan rewrites `dispatch_plan_prices`, one row per planned move
(`dispatchplanpricerepository.Replace`). A row holds the planned driver (or none, with the
uncovered reason), cost, coverage price, driver value and alternatives. The board attaches
the rows in one batched read (`dispatchconsoleservice.attachPlanPrices`) as
`DispatchBoardMove.planPrice`. Each driver gets `plannedMove`.

- **Prices only move forward.** `Replace` deletes rows for the planned moves whose
  `generated_at` is not newer than the plan's, then inserts with `ON CONFLICT DO NOTHING`.
  An older plan that commits after a newer one never overwrites it.
- Rows follow their move (`ON DELETE CASCADE`). A covered move shows no price, and an old
  price says how old it is (`generatedAt`).

## Learned dispatch style

With `learnDispatchStyle` on, the soft components carry per-organization multipliers
(`dispatch_styles`, `operations/domain/dispatchlearning`), bounded to 0.25-4x and applied by
`dispatchcost.Multipliers` to soft components only. `BuildSnapshot` reads them (one cached
row) for every surface, so the inspector, pre-flight and plan agree.

- **The signal is operations' own data.** `dispatchlearningjobs` (every 15 minutes, a tenant
  fan-out over organizations with learning on) reads `DriverAssigned` shipment events a
  person made (`actor_type = 'user'`) after the style's cursor, with the move's plan price
  row (planned driver, cost lines, alternatives with their cost and lines) when it was
  written in the day before. Planned driver taken: approved; a priced alternative taken:
  overridden; anyone else: declined. Agent auto-executions are system events and teach
  nothing.
- **The update** (`DispatchStyle.Learn`) is a passive-aggressive step in log-multiplier
  space: the soft line differences between the preferred and rejected pairing set the
  direction, the gap to close is the alternative's override cost (or the closest
  alternative's shortfall from a $25 margin for an approval, the shortlist's mean for a
  decline) plus the margin. Each component's rate decays with its samples (0.5 down to
  0.05) and one decision moves a weight at most ±10%. Approvals weigh a quarter and declines
  a half of an override.
- **Idempotent.** The cursor is saved with the weights in one optimistic-locked write, so a
  retried activity never learns a decision twice; a reset (`resetDispatchStyle`, audited)
  clears the weights and keeps the cursor.

## End-market values

With `planForEndMarket` on, each plan folds its drivers' exact duals (`ColumnValue`, an idle
driver at zero) into `dispatch_market_values` by where and when each driver becomes free:
web mercator tiles at zoom 8 (`orb/maptile`) by four-hour slot of the week (UTC, Monday
00:00). A slot keeps the mean of the current week and an exponential moving average of past
weekly means (weight `max(1/(weeks+1), 0.2)`), so how often the plan runs does not change
the estimate, and a replayed plan (same `observed_at`) folds nothing. `BuildSnapshot` reads
the rows of the moves' destination cells once (`SnapshotRequest.Destinations`) and the
valuer credits a pairing `0.8 × value × n/(n+4)` for `n` observations, falling back to the
cell's average at half weight for an unseen slot. The fold is best-effort: a failed write is
logged and the plan stands.

## Anticipated freight

End-market values look backward: what drivers free in a place were worth to past plans. With
`anticipateFreight` on, the plan also looks forward. A per-organization freight forecast says
where loads are likely to appear before they are booked, and the "where the driver ends up"
component (`endMarket`) prices a move by the chance a load will need a driver nobody else
covers where the move leaves them, against the same chance where they are now. The model is
`operations/domain/demandforecast`; the service, repository and nightly job are
`demandforecastservice`, `demandforecastrepository` and `demandforecastjobs`.

Network-level demand forecasting and backward-looking driver values exist elsewhere; TMS
dispatch screens rank drivers for booked loads only. What is new is the combination, inside
the one cost model:

1. **A nowcast from the booking curve.** What is already booked for a future window updates
   the forecast for it, exactly (Poisson–Gamma), so a heavier week than usual is believed.
2. **Supply-aware value.** A driver is worth the chance that demand exceeds the other drivers
   expected there, so the first extra driver in an area is worth much more than the fifth.
3. **Relative credit.** A move earns what the driver is worth where it leaves them less what
   they were worth staying put, so pulling a truck out of an area about to get busy is a charge.
4. **A two-pass solve** so two drivers are not both credited for the same likely load.
5. **One currency, one component.** The credit is `endMarket`, in cents, explained in the
   cost breakdown. The forecast replaces the history value where it covers the drop's area.
6. **Proven before it acts.** A nightly backtest on the organization's own history decides
   whether plans may use it, and Dispatch controls shows the evidence.

### The model

- **Load, area, slot.** A load is a non-canceled shipment move. Its origin is the first live
  stop's location, its time that stop's scheduled window start, its booking time the
  shipment's `created_at`. Carrier-covered moves count: they are freight that appeared. An
  area is a zoom-8 map tile (`dispatchmarket.CellOf`, about 120 km across in the US) and a
  slot one of the 42 four-hour blocks of the week (`SlotOf`, UTC, Monday 00:00 first), the
  same keys as `dispatch_market_values`.
- **Rate per area and slot** (`Fit`): empirical Bayes Poisson–Gamma over the last 26 weeks
  (`HistoryWeeks`), each week weighted `0.5^(age / 8)` (`HalfLifeWeeks`). The prior mean is
  the area's weekly total × the tenant's share of volume in that slot (its weekly rhythm), at
  a strength of 2 pseudo-weeks (`PriorWeeks`): `α = 2·m0 + Σ r_w·y_w`, `β = 2 + Σ r_w`. A
  sparse slot shrinks toward the area's rhythm instead of reading as zero.
- **Customer series** `(customer, area, slot)` are fitted the same way, with the customer's
  area total × the area's slot share as the prior. They do not change the area rate; they
  name who drives a forecast ("Acme Foods about 3") and weight the booking curve.
- **Booking curves** (`fitCurves`): `F_c(L)`, the share of a customer's loads booked at least
  `L` hours before pickup, on the buckets `LeadHours` (0, 2, 4, 8, 12, 24, 36, 48, 72, 96,
  120, 168, 336), shrunk to the tenant's curve with 20 pseudo-loads (`CurvePriorLoads`). A
  slot's curve is its customers' curves weighted by their rates.
- **Nowcast** (`Nowcast`): for a slot starting `L` hours ahead, the share still unbooked is
  `u = 1 − F(L)`. With `b` loads already booked for it, `λ | b ~ Gamma(α + b, β + 1 − u)` and
  the unbooked remainder is negative binomial with mean `(α + b)·u / (β + 1 − u)`. A fully
  booked slot (`u = 0`) expects nothing more.
- **Window.** A driver free at `t` can take loads in the slot containing `t` and the next two
  (`WindowSlots`, twelve hours). Window demand is the booked loads still open (pickup in the
  window, not covered, not in the current solve) plus the unbooked remainder of the three
  slots, summed by moments.
- **Supply** `k`: dispatchable drivers whose plan origin is in the area and who are free
  before the window ends, never counting the driver being valued.
- **Value** (`Forecaster.ValueAt`, one window):
  `round(0.8 × median revenue × margin × P(open booked + unbooked > k))`, where median
  revenue is the area's (`demand_forecast_areas`), margin is the organization's coverage
  margin, and the result is capped by the economics setting `anticipationMaxCents` (default
  $400). Worked example: unbooked mean 3.4, $360 of value, one other driver expected gives
  `0.8 × 360 × 0.86 = $248`; with four others it falls to $72.
- **Credit** (`endMarketOf` in `dispatchcandidateservice`): drop value − stay value. The drop
  window is in the move's destination area from the driver's projected free time; the stay
  window is in the area of the driver's current start (`FleetSnapshot.DriverStart`) from when
  they are available, and counts as zero where the forecast does not cover that area. The net
  can be negative: a move out of a busier area is a charge. The arc's `endMarket` line is the
  negative of the net.
- **Explanation** (`Explain`): "Ending near Dallas, TX is worth $248: 3–5 loads usually
  appear there Tue 06:00–18:00 (Acme Foods about 3), 1 already booked, 1 other driver
  expected." The candidate's `endMarket` reason (`DispatchEndMarket`, source `Forecast`)
  carries the drop's fields plus `dropValueCents`, `stayValueCents` and `stayAreaLabel`;
  `valueCents` is the net. The cost breakdown shows the drop sentence and, when staying was
  worth something, what the move adds or costs against it.

### The solve

- **Forecast or history, never both.** Where the forecast is Ready and covers the drop's area
  (it has a fitted area with a median revenue), `endMarket` comes from the forecast; elsewhere,
  when `planForEndMarket` is on, it falls back to `dispatch_market_values` (drop value only).
  A pairing is never credited twice.
- **Two passes, immediate mode only** (`reanticipate` in `dispatchplanservice`, before
  `planImmediate`). Pass 1 solves with arcs priced from the snapshot's supply. Supply is then
  recounted from where that solve leaves every driver: an assigned driver at the move's
  destination from their projected free time, an unassigned one at their start
  (`Forecaster.WithSupply`). Every forecast arc's drop and stay values are repriced
  (`dispatchcandidateservice.Reprice`) and the plan solves again with the new prices. A likely
  single load in Dallas then credits one of two drivers sent there, not both. The extra solve
  costs milliseconds. Horizon mode prices from the snapshot's supply and runs no second pass.
- **Bounded reads.** The snapshot reads rates, areas and curves for the moves' destination
  cells in one statement (`IN` on cell, index on organization, business unit and cell) and the
  booked-but-unplanned counts per window in one grouped statement. Only those cells are
  loaded, so a stay window in an area no move ends in values at zero.

### Readiness

Every night, per organization, a rolling-origin backtest (`Backtest`) refits on the weeks
before each of the last 8 weeks (`BacktestWeeks`), predicts every area-slot count of that
week and compares:

- **Interval coverage**: the share of actual counts inside the forecast's 80% range (10th to
  90th percentile), ideally near 0.8.
- **Skill**: `1 − MAE / naive MAE`, where naive repeats the same slot of the week before.
- **Status**: `InsufficientHistory` (fewer than 8 weeks or 200 loads, or nothing testable),
  `NoSkill` (skill at most 0.05) or `Ready`. Plans use the forecast only when Ready; Dispatch
  controls shows the status, the coverage, the skill and when it was last fitted.

### Where it lives

| Table | What |
|---|---|
| `demand_forecast_rates` | `alpha`, `beta` per area and slot; with `customer_id` for customer series |
| `demand_forecast_areas` | per area: label (busiest origin city), median revenue, loads |
| `demand_booking_curves` | booking shares per lead bucket; no `customer_id` is the tenant curve |
| `demand_forecast_quality` | one row per organization: status and the backtest numbers |

The Temporal schedule `demand-forecast-fit` (cron `15 3 * * *`, overlap skipped) fans out
with a bounded concurrency over every tenant that has a dispatch controls row. Each tenant
aggregates its history in one statement, fits, backtests and replaces its rows in one
transaction. Turning `anticipateFreight` on starts `FitDemandForecastWorkflow` for that tenant
(one workflow ID per tenant, an existing run reused); a failed start only logs a warning and
the nightly run fits it. Nothing is fitted on a request path. `anticipateFreight` lives on
dispatch controls and is off by default. The console's **Outlook** tab
(`dispatchFreightOutlook`), the quality card (`dispatchForecastQuality`) and Nova's
`dispatch_forecast_demand` read the same service (`demandforecastports.Service`).

Where a driver starts, read in SQL, is one join in `adapters/postgres/driverstart`: the drop
of their last open (assigned or in-transit) move, else a GPS position under 4 hours old, else
the tractor's last drop. The Outlook's supply (`demandforecastrepository.ListDriverStarts`)
and the console's nearest-driver order use the join; the console's filters and summary take
its open move statuses and freshness from it. A new reader joins it instead of writing its
own precedence.

### Evidence

`TestSimulation_AnticipationEarnsMoreWithoutLosingCoverage` (`simulation_test.go`) fits 16
weeks of a steady rhythm in four Texas and Oklahoma cities, then plans 4 weeks with 10
drivers and the real solver:

| Credit | Covered | Contribution |
|---|---|---|
| Off | 378 of 488 (77.5%) | $276,261 |
| Drop only | 392 (80.3%) | $283,273 |
| Drop minus stay | 391 (80.1%) | $284,460 (+3.0%) |

The win is covering loads a myopic plan leaves, which costs some empty driving (65.7 empty
miles per load against 64.9 off). Measure contribution (revenue covered less empty miles at
the organization's rate), not empty miles per load.

## Rules that bite

- **One objective.** Don't add a score, a weight or a second ranking. A new consideration is
  a cost component in `dispatcheconomics.Component` with its economics setting, and it is
  either hard or soft.
- **No fake zeros.** An unknown position, HOS or history must not price as free. Unknown
  location ranks after known ones (`dispatchplanports.CompareRank`) and never auto-executes.
- **Bounded on the request path.** The graph is top-K per move. Never evaluate every
  driver-move pair, and never call the route estimator outside the top 8 per move.
- **Snapshot reads stay sequential** ([scale.md](backend/scale.md): no goroutine fan-out of
  database reads). Parallelism is for pure evaluation.
- **The forecast credits only when Ready.** `Forecaster.Covers` is false for any other
  status, whatever `anticipateFreight` says. Every path that prices from the forecast goes
  through it.
- **Forecast or history, never both.** A drop's `endMarket` comes from one source. Adding the
  forecast credit on top of `dispatch_market_values` counts the same future twice.
- **The credit is relative.** A forecast `endMarket` is drop value minus stay value, and a
  negative net is a charge. Crediting the drop alone pays a plan for pulling a driver out of
  an area about to get busy; the simulation shows it earns less.
- **Supply excludes the driver being valued.** Counting the driver among the others expected
  in the area makes them compete with themselves and lowers their own credit. In immediate
  mode the second pass recounts supply from where the first solve leaves every driver and
  reprices both windows of every forecast arc; skipping it credits two drivers for one likely
  load.
- **Judge anticipation by contribution.** It covers more loads at a little more empty driving
  each; empty miles per load alone reads a win as a loss.
- **Change the solver only with its exactness tests.** `sparse_test.go` compares the optimum,
  `OverrideCost` and `ColumnValue` against brute force. A change that breaks them is wrong
  even if the plan looks fine.

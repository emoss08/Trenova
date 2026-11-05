# Dispatch optimizer

How Trenova decides which driver takes which move: the cost model, the solve, the prices it
publishes and where each piece lives. Read this before changing anything under
`operations/app/dispatch*`, `operations/domain/dispatch*`, `shared/assignmentsolver`,
`shared/dispatchplanner` or the dispatch economics settings.

## The model

Dispatch is economics, so every number the optimizer works with is in cents.

- **Pair cost.** Each driver-move pairing has a cost (`dispatchcost.Model`): empty miles,
  pickup and delivery lateness, idle wait beyond the early grace, rest a driver must take
  first, and the end-market value of where the move ends. Those are *hard* components and
  decide coverage. Home time, fairness and the preferences (customer experience, on-time
  history, safety, acceptance, trailer continuity, haul fit, time-off buffer) are *soft*:
  they only choose between drivers. They are normalized per move, so the cheapest soft cost
  among a move's candidates is zero. Missing data is neutral (quality 0.5), never free.
- **Coverage price.** Each move has a price for leaving it uncovered:
  `max(revenue × margin %, floor) + urgency premium` (`dispatchcost.CoveragePrice`; Late and
  Now buckets from `dispatchurgency`). A move is covered only when a driver costs less than
  leaving it.
- **Economics** are per-organization settings (`dispatchcontrol.Economics`, defaults in
  `dispatcheconomics.Default`). Nothing else weighs a pairing: ranking, pre-flight, the plan
  and the agent tool all use the same model.

## The solve

1. **Snapshot** (`dispatchcandidateservice.BuildSnapshot`): sequential, bounded reads of every
   dispatchable driver (`DriverLimit`, dispatchable first), commitments by overlap with ETA
   ends and remaining drive from `stop_etas`, HOS, positions, history. `Prepare` builds one
   profile per driver: origin (commitment, then GPS, then tractor last known), HOS timeline
   carried through committed drive, history signals.
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
   - `ColumnValue(col)`: how much more the plan costs without that driver (exact reroute,
     not the degenerate dual).
   - `OverrideCost(row, col)`: the exact extra cost of forcing a different driver onto a move
     with the rest re-planned around it. It is checked against brute force on random boards.
5. **Horizon mode** chains several moves per driver with `shared/dispatchplanner`
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
   standing proposal moved to another driver costs `StabilityCents`. Alternatives and driver
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
- **Change the solver only with its exactness tests.** `sparse_test.go` compares the optimum,
  `OverrideCost` and `ColumnValue` against brute force. A change that breaks them is wrong
  even if the plan looks fine.

# Distance and mileage

How Trenova measures a move, which distance profile each kind of mile uses, and which code reads
which mile. The settings live on the distance control (`places/domain/distancecontrol`, the
Distance Controls admin page); the measuring is `places/app/distancecalculationservice`, the
route estimator is `places/app/routeestimateservice`, and both reach PC*MILER through
`places/app/pcmilerruntime`.

## Purposes

A distance profile is a set of PC*MILER routing options (routing type, data version, units,
location granularity, tolls, borders). The distance control assigns one to each purpose, because
trucking measures the same trip differently for different jobs:

| Purpose | Control field | What it measures |
|---|---|---|
| Loaded move | `loadedMoveDistanceProfileId` | the move's own distance (`shipment_moves.distance`) when it carries freight |
| Empty move | `emptyMoveDistanceProfileId` | the move's own distance when it repositions empty |
| Arrival estimates | `etaOutOfRouteDistanceProfileId` | the legs ETAs and dispatch suggestions are estimated over |
| Pay | `payDistanceProfileId` | the miles drivers and carriers paid by the mile are paid on, commonly practical miles |
| Billing | `billingDistanceProfileId` | the miles rating, tariffs, fuel surcharge and invoices bill on, often shortest or household-goods miles |
| Fuel | `fuelDistanceProfileId` | the miles fuel tax returns are filed from, and the route the jurisdiction breakdown comes from |

`DistanceControl.ProfileIDForPurpose` is the one place a purpose becomes a profile id. Every
assignment is required; an assignment that names no usable profile (deleted, inactive, not
PC*MILER) leaves that purpose unrouted, exactly as an unroutable loaded move keeps its
hand-entered distance. There is no second fallback rule.

`ratingDistanceUnit` is the unit tariffs and rate tables read distance in
(`rating/app/tariff/settings.go`). It is not a stored-mileage setting.

## Purpose distances on a move

`ResolveForShipment` routes each move with its loaded or empty profile and stores the result on
the move. For each of pay, billing and fuel whose profile differs from the profile the move was
routed with, it measures the move again with that profile and stores the result in
`pay_distance`, `billing_distance` or `fuel_distance`, converted to the move's distance units.
Purposes that share a profile share one route. When a purpose shares the move's profile, or the
move's distance is manual or an override, the purpose column is null and the move's distance
stands for it. Readers never read the columns directly: they call
`ShipmentMove.PaidDistance()`, `BilledDistance()` or `FuelTaxDistance()`, which fall back to
`Distance`; SQL readers use `coalesce(<purpose>_distance, distance)`
(`movecompletion.FuelTaxMiles`).

Scale rules that bite:

- Every route of a shipment, every purpose included, goes in the one `Mileage` call the shipment
  already makes; a purpose never adds a call per move. With the defaults (every purpose on the
  practical profile) no extra route is requested at all.
- Stored mileage is keyed by the distance profile (`RouteHash` includes the profile id, and the
  lookup filters on `DistanceProfileID`), so a billing route never answers a pay lookup. A purpose
  route checks stored mileage of its own profile first and buffers its result for that profile.
- When the fuel profile differs from the move's, the move's own route asks for no state report;
  the fuel route carries it, and the jurisdiction rows, their profile id and the mismatch check
  all come from the fuel route. `RecalculateMoveJurisdictionMiles` (and the IFTA backfill that
  calls it) always routes with the fuel profile and stores `fuel_distance` when it differs.
- A purpose route that fails leaves its column null and adds a warning to the move's result and
  metadata (`distanceMetadata.purposeDistances` records each purpose's profile and source).

## Consumers

| Consumer | Code | Reads |
|---|---|---|
| Driver pay accrual and estimate | `settlement/app/driversettlementaccrualservice` (`moveDistance`) | paid |
| Carrier assignment per-mile pay and its distance check | `rating/app/carrierassignmentservice` (`buildAssignment`, `validatePerMileDistance`) | paid |
| Tender screening, counter-offer margin, escalation ceiling | `operations/app/tendercreate`, `tendernegotiation`, `tenderescalation` | paid |
| Carrier capacity quotes on the board | `operations/app/shipmentcapacityservice` (`rankCarrierMatches`, board row `PaidDistance`) | paid |
| Rating facts: rate engine, rate agreement distance ranges, tariffs, fuel surcharge | `operations/domain/shipmentrating` (`Distance`, `Miles`) | billed |
| Tariff record leg distances | `rating/app/tariff/record` (`singleLegDistance`) | billed |
| Billing queue freight basis and review prompt | `billing/app/billingqueuereadiness`, `billingqueueassist/reviewprompt` | billed |
| IFTA unattributed miles and mismatches | `settlement/adapters/postgres/iftarepository`, `operations/adapters/postgres/shipmentmovejurisdictionmilerepository` | fuel tax |
| IFTA jurisdiction rows | `distancecalculationservice` (fuel route) | fuel route |
| ETA projection, dispatch candidate legs | `routeestimateservice` (ETA profile), `etaprojection`, `dispatchcandidateservice` | ETA route |
| Dispatch board, dispatch cost and economics, costing estimates and fleet miles, driver portal, tracking, quick filters, telematics feasibility | read `distance` | the move's routed distance |

Operational cost and dispatch read the move's own distance on purpose: it is the route the truck
is planned to drive, not a contractual measure. Costing is a cost of driving, so it reads it too.

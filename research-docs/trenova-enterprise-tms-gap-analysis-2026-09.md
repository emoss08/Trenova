# Trenova Enterprise TMS Gap Analysis

Date: 2026-09-04

Supersedes `trenova-enterprise-tms-gap-analysis.md` (2026-05-26).

## Scope and Method

Trenova today is roughly 790k lines of Go across `services/tms` plus the `gtc`
CDC service, `samsara-sim`, and `edi-partner-sim`; 108 domain packages, ~180
services, ~110 REST handler packages, 71 GraphQL schema files, 34 Temporal job
packages, ~600 migrations covering ~300 tables, a 1,072-file React back-office,
and a 33-file driver PWA.

Every claim below was verified by reading domain structs, service
implementations, migrations, the route registry (`internal/api/router.go`), the
GraphQL schema directory, the report catalog
(`internal/infrastructure/database/reportcatalog/reportcatalog.yml`), the
integration enum (`domain/integration/enums.go`), and the client router
(`client/apps/web/src/router.tsx`). File paths are cited so any specific finding
can be re-checked.

The May 2026 analysis is materially stale. The Order/Shipment split, carrier
procurement, routing guides, driver and carrier settlements, rate agreements,
fuel surcharge programs, AS2 transport, and SCIM have all landed since it was
written.

The question this document answers: **what does Trenova still need to be a
complete asset-based enterprise TMS, and where can it be genuinely ahead of the
market rather than merely at parity?**

## 1. What Is Already Strong

These areas are at or above commercial-TMS parity and should not be rebuilt.

**Rating.** `services/rateengine/` applies a traced pricing order — base charge,
discount, absolute minimum, deficit distance, rule guardrails, rounding, then FX
conversion last. Every rating persists a `ratequote` row including failures and
fallbacks. One `RateAgreement` prices both the customer sell side and the carrier
buy side via `PartyType` (`domain/rateagreement/enums.go`), so margin cannot
drift between the two. Supporting depth: four-dimension rate matrices, zones,
General Rate Increase handling, amendment and approval workflow, rate import with
a mandatory dry-run stage, rate simulation with historical replay, deficit
rating, and 2025 NMFC density scales. The formula engine is `expr-lang`,
sandboxed at 1,000 expression nodes and a 5-second evaluation timeout, and
emits evaluation receipts.

**Fuel surcharge.** `domain/fuelsurcharge/` ingests live EIA/DOE diesel prices by
PADD region with a 21-day staleness guard, and supports five program methods
(per-mile step/peg, per-mile MPG, table per-mile, table percent, table flat).
Contract-level overrides for waiver, peg, increment, and cap live in
`domain/rateagreement/fuelbinding.go`.

**Dispatch.** `services/dispatchcandidateservice/score.go` ranks candidates on 13
tunable, individually explained factors: deadhead, HOS margin, on-time, on-time
history, trailer continuity, fleet match, driver-type fit, home time, load
balance, lane experience, safety history, acceptance, and PTO proximity — with
strategy presets in `domain/dispatchcontrol/scoringweights.go`. Underneath sit an
ALNS planner (`shared/dispatchplanner/alns.go`, with random/worst-cost/resource
ruin operators and greedy/regret repair) and an optimal Hungarian solver
(`shared/assignmentsolver/hungarian.go`). Horizon planning runs 48 to 336 hours
ahead with scheduled re-planning.

**Hours of service.** `services/hosprojection/` implements the 10-hour reset, the
34-hour restart, and split sleeper-berth pairing, with rulesets derived for US
Interstate property and passenger, Texas intrastate, and Canada CS/CN. This is
ahead of most incumbents.

**Detention.** The deepest module in the repository. Policy tiers, clock-start
basis, late-arrival rules, per-stop-type free minutes, separate pay-vs-bill free
time, rounding modes, per-stop/day/shipment caps, layover conversion, notice PDFs,
and a backtest harness (`services/detentionservice/backtest.go`).

**Accounting.** Real double-entry with subledger-to-GL posting, maintained period
balances, fiscal period lock and reopen, 21 configurable default GL account
pointers, a 2,551-line invoice adjustment engine, bank-receipt matching with
tolerance policy, and AR analytics covering DSO, CEI, average days to pay,
write-off ratio, dispute rate, short-pay rate, cash-flow forecast, and a
severity-ranked collections worklist.

**Settlements.** Driver settlements include instant pay, escrow with 49 CFR
376.12(k) interest accrual, advances with a recovery lifecycle, recurring
deductions and earnings, and a driver-facing dispute flow. Carrier settlements
include two-way invoice matching against inbound EDI 210 with variance
acceptance and adjustment events.

**EDI.** Transaction sets 204, 990, 214, 210, 997, and 999 in both directions,
with control-number acknowledgement matching. Transports: AS2 with S/MIME signing,
encryption, MIC computation, and synchronous or asynchronous MDN
(`shared/as2/`), plus SFTP. A real mapping designer with Starlark transforms, a
certified template lifecycle (Draft to Certified to Active to Deprecated to
Archived to Superseded), a test-case verdict harness, and a dedicated partner
simulator service.

**Costing.** `services/costingservice/` produces fixed and variable cost per mile,
break-even revenue per mile, and margin, sourced from benchmark rates, overrides,
or actual GL postings. Margin-floor enforcement runs at carrier assignment
(`services/carrierassignmentservice/buyside.go`).

**Permitting.** Per-state oversize/overweight requirement derivation with
exceedances, escort thresholds, superload flags, daylight/rush-hour/weekend
restrictions, lead time, validity, fees, and a waiver workflow
(`domain/permit/`, `domain/jurisdictionrule/`).

**Platform.** An RBAC engine with privilege-escalation guards across 403
registered resources; the Temporal job fabric; the GTC CDC service with DLQ and
replay; AES-GCM envelope encryption backed by GCP KMS; document intelligence with
real OCR and LLM extraction; SCIM; and OIDC SSO.

**AI agents.** A genuine governance model — Propose, ActWithApproval, and
AutoExecute autonomy tiers, shadow mode, evidence capture, prompt versioning,
input-context hashing, and tenant-scope enforcement (`domain/agent/`).

## 2. Missing Departments

These are whole modules, not features.

### 2.1 Maintenance and Asset Lifecycle

Entirely absent. No work orders, preventive maintenance schedules, defects,
parts inventory, warranty, tires, or repair vendors.

Three consequences matter operationally:

- `EquipmentStatus.AtMaintenance` (`pkg/domaintypes/enums.go`) is a status with
  nothing behind it — it points at no maintenance record.
- DVIRs are ingested read-only from Samsara into
  `domain/telematics/vehicleinspection.go`, carrying `unresolved_defect_count`,
  and then dead-end. There is no defect to work order to repair to resolution
  loop inside the TMS.
- `services/dispatcheligibility/evaluate.go` blocks dispatch on driver
  credentials and shipment holds but never on equipment. There is no
  PM-overdue, out-of-service, expired-registration, or failed-inspection gate.

`domain/tractor/tractor.go` also carries no odometer, engine hours, or fuel
economy, even though `telematics_vehicle_positions` already ingests
`odometerMeters` and `fuelPercent`.

### 2.2 Safety Department

Entirely absent. No accidents, incidents, cargo claims, OS&D, roadside or annual
inspections, DataQs, or CSA/FMCSA BASIC scores.

`domain/servicefailure/` covers late and missed pickup/delivery events with fault
attribution (Carrier, Driver, Shipper, Consignee, Facility, Equipment, Weather,
Integration) but explicitly not damage, shortage, or overage.
`carrier.SafetyRating` is a manually-set four-value enum, not an FMCSA feed. The
organization's own tractors, trailers, and drivers have no insurance-policy
tracking — only carriers do (`domain/carrier/insurancepolicy.go`).

### 2.3 Drug and Alcohol Program, FMCSA Clearinghouse

Effectively absent, and this is a DOT audit failure.

The entire program is one boolean control
(`DispatchControl.EnforceDrugAndAlcoholCompliance`) and one date field
(`WorkerProfile.LastDrugTest`). The whole enforcement is
`services/dispatcheligibility/worker.go:111-127`, a single check that
`LastDrugTest > HireDate`.

There are no test records, no test types (pre-employment, random,
post-accident, reasonable-suspicion, return-to-duty, follow-up), no random
selection pool management, and no Clearinghouse query or registration. The
string `clearinghouse` does not appear anywhere in the repository.

### 2.4 Fuel Spend, IFTA, and IRP

Entirely absent. The existing fuel code is a billing construct: `fuel_indices`,
`fuel_index_prices`, and `fuel_surcharge_programs` compute the customer fuel
surcharge from EIA prices. That is revenue, not spend.

There is no fuel purchase table, no fuel card integration — Comdata, EFS, and WEX
exist only as `AdvanceSource` enum labels on a manually-entered advance
(`domain/driverpay/enums.go`) — no MPG or idle tracking, no jurisdictional
mileage accumulation, no quarterly IFTA return, and no IRP apportioned
registration or cab cards.

Every input already exists: `stored_mileages`, `distance_calculations`, EIA
prices, and telematics odometer readings. They are simply never joined into a tax
computation.

### 2.5 Accounts Payable

Absent, with live configuration pointing at it. There is no vendor, no bill, no
purchase order, and no three-way match.

Meanwhile `domain/tenant/accountingcontrol.go` declares
`ExpenseRecognitionOnVendorBillPost`, emits
`JournalSourceEventVendorBillPosted`, and points at `DefaultAPAccountID`. All of
this is dead configuration for an entity that does not exist — nothing ever
credits AP.

## 3. Structural Gaps

| Gap | Evidence | Why it matters |
|---|---|---|
| No customer portal | Only token-scoped public pages: `/tender-offer/:token`, `/rate-confirmation/:token` | Non-EDI shippers cannot place an order, track a load, pull a POD, or view an invoice |
| No carrier portal | Same | Carriers cannot self-onboard, upload COIs, or see settlement status |
| No outbound webhooks | No webhook domain, service, or subscription; only inbound Samsara and Postmark receivers | No external system can subscribe to Trenova events; blocks any partner ecosystem |
| No live ETA | `CandidateScore.ProjectedArrival` is dispatch-time only; nothing on shipment or stop | Cannot answer "where is my load and when will it arrive"; no ETA-vs-appointment slip detection |
| No dock or appointment scheduling | `domain/location/location.go` has no operating hours, doors, capacity, or slots | Appointment windows exist on the stop, but nothing schedules against facility capacity |
| Fiscal close does no accounting | `fiscalyearservice.Close` flips status and audits; `DefaultRetainedEarningsAccountID` is validated but never written to | No closing entries, no retained-earnings roll-up, no opening-balance carryforward |
| No consolidated invoicing | `InvoiceLine` carries per-line shipment references, but there is zero grouping logic | Blocks LTL and any statement-billed customer |
| No tax on invoices | Invoice has Subtotal, Other, and Total only; no tax code, rate, jurisdiction, or line | Blocks Canada GST/HST and any taxable accessorial |
| Native MFA unimplemented | `iam.MFAAuthenticator` models TOTP and WebAuthn; only `ListMFAAuthenticators` exists. `authservice/service.go:883` only reads `amr` claims from an IdP | A customer without SSO cannot enforce 2FA; SOC2 blocker |
| SAML modeled but disabled | `iamservice/service.go:124` — "SAML providers cannot be managed until SAML sign-in is available" | Large shippers and 3PLs still require SAML |
| Rate limiting is in-process and IP-keyed | `middleware/ratelimit.go` uses `x/time/rate` on `c.ClientIP()` | Does not survive horizontal scaling; not per-tenant or per-API-key |
| Public API is a 13-resource allowlist | `apikeyservice/policy.go`, read/create/update only | No billing, invoice, settlement, document, or EDI access via API |
| Report catalog excludes all financial tables | `reportcatalog.yml` has 27 datasets, none of `journal_entries`, `gl_accounts`, `gl_balances`, `driver_settlements`, `customer_payments`, `bank_receipts`, `rate_agreements` | The self-service report builder cannot report on the ledger, cash, settlements, or rates at all |
| US-only geography | `domain/usstate/` is the only jurisdiction entity; no provinces | Cross-border Canada is impossible; no PARS/PAPS, ACE eManifest, in-bond, or customs broker |
| Telematics is Samsara-only | `domain/integration/telematics.go:9` returns true only for Samsara; Motive is a commented-out enum member | The 12-method provider interface is ready; implementations are not |
| Realtime hard-coupled to one vendor | `realtimeservice` mints JWTs for Foony (`wss://realtime.foony.io`); no self-hosted WebSocket or SSE fallback | A vendor outage means no realtime and no degraded mode |
| Deployment is single-node | `deploy/` is a Caddyfile, three Dockerfiles, and Prometheus/Grafana. No Kubernetes, Helm, or Terraform | Corroborated by the in-process rate limiter — multi-replica was not designed for |
| Web app is desktop-only | Four responsive utility classes across roughly 600 `.tsx` files | Operations managers cannot work from a phone |

### Orphaned Schema

Three artifacts imply features that do not exist. Each needs a decision: finish
or delete.

- **Freight consolidation.** `consolidation_groups` and `consolidation_settings`
  exist with tuned matching parameters — `max_pickup_distance`,
  `max_route_detour`, `max_time_window_gap`, `max_shipments_per_group`
  (`20250628004846_add_consolidations`). No Go code reads or writes them.
  `Shipment.ConsolidationGroupID` is a write-through field only. Note that
  `customer.AllowInvoiceConsolidation` is a separate, unrelated billing concept.
- **Dedicated lanes.** `domain/dedicatedlane/` (pattern, patternconfig,
  suggestion, errors) plus three migrations. The only live reference in the
  application is a permission-mapping constant at
  `handlers/analyticshandler/handler.go:34`. No service, repository, resolver, or
  route.
- **`hazmat_expirations`.** Created in `20241228042029_compliance`, never
  referenced by any Go code.

## 4. Shallow Where Depth Is Expected

- **MVR** is two date fields (`LastMVRCheck`, `MVRDueDate`). No record entity, no
  violations, no points, no vendor pull.
- **Driver qualification file** is flags plus generic documents. No structured
  checklist with per-document required and expiry state, no PSP or prior-employer
  inquiry records.
- **Driver scorecards** are computed at candidate-ranking time and never
  persisted. No trend, no driver-facing view. The only `Scorecard` type in the
  repository is `EdiPartnerScorecard`.
- **PTO** has request windows but no accrual, balances, or carryover.
- **Team driving** has primary and secondary worker fields, but `hosprojection`
  models a single driver's clocks — there is no co-driver handoff.
- **Equipment renewals** have expiry reports but no workflow or sweep job.
  Drivers get `compliancejobs/credential-expiry-sweep`; equipment gets nothing.
- **Home time** is scored as `daysOut` only. There is no domicile or home-terminal
  FK on `worker.Worker`, and no home-time commitment or target policy.
- **Financial statements** — `glbalanceservice` is 161 lines: single-period trial
  balance, income statement, balance sheet. No YTD, comparatives, cash flow
  statement, account hierarchy, drill-through, or export.
- **Segment accounting** — journal entry lines carry only `CustomerID` and
  `LocationID`. `GLAccount.RequireProject` references a project entity that does
  not exist.
- **Multi-currency** is real at the rating and reference-data level but cosmetic
  in accounting. `RealizedFXGainAccountID` and `RealizedFXLossAccountID` are
  configured and never posted to; there is no revaluation and no realized gain on
  payment.
- **Document retention** is three knobs (audit, EDI inbound, EDI message). No
  document retention schedule, no legal hold, no purge job.
- **Document parsing rules** support exactly one kind,
  `DocumentKindRateConfirmation`. No BOL, POD, invoice, or W-9.
- **No e-signature.** The rate-confirmation token link is click-to-accept, not a
  signature with an audit certificate. No driver signature pad in `apps/dash`.
- **AI agents** run one live workflow, `BillingExceptionAgentWorkflow`, with five
  billing-scoped tools. `DispatchAssignment` is a declared-but-dead enum value.
- **Analytics providers** number two: `shipmentprovider` and `apikeyprovider`.
  There is no financial provider.
- **Missing standard trucking KPIs**: operating ratio, driver turnover, revenue
  per truck per day, maintenance cost per mile, fuel economy, claims ratio.
- **No user-configurable automation.** No rules engine, no workflow designer, no
  feature flags. All automation is compiled Go across 34 Temporal packages. A
  dispatcher cannot express "if X then Y" without a developer.
- **Driver PWA gaps**: no offline queue, no signature capture, no guided document
  scan, no DVIR submission, no fuel-purchase entry, no navigation.
- **SMS has exactly one caller** — `workerptoservice` PTO approve and deny —
  despite a working Twilio client and Temporal workflow. No voice or IVR, no
  escalation policies.
- **No bid/RFP management and no market rate benchmarking** (DAT, Greenscreens).
  Notable because the rate engine underneath would support a bid module cheaply.
- **No load boards, visibility providers, factoring, payroll export, ERP or GL
  export, or carrier vetting feeds.** The GL is a closed island — there is no
  IIF, QBO, or CSV journal export of any kind.

## 5. Where Trenova Can Be Genuinely Ahead

Each of these is buildable because of what already exists. They are compositions,
not greenfield bets.

1. **The living plan.** Today the horizon planner runs on a schedule and the
   score is computed on demand. Turn it into a continuously re-optimizing plan
   that reacts to telematics drift, HOS burn-down, detention clocks, and weather
   polygons, surfacing swap proposals with a quantified dollar delta from
   `costingservice`. Every ingredient already exists — ALNS, the 13-factor score,
   HOS projection, the cost engine, and agent autonomy tiers. Nobody in the
   market has this.
2. **Margin-aware autonomous tender response.** Combine break-even RPM
   (`costingservice`), dispatch feasibility (`dispatcheligibility`), and the
   routing guide into an auto-accept, auto-counter, or auto-decline policy on
   inbound 204s and spot offers, with the reason chain shown. The agent
   framework's `ActWithApproval` tier is exactly the right governance.
3. **Driver retention engine.** Turnover runs near 90% industry-wide and no TMS
   models it. Trenova already has home-time scoring, PTO, pay events,
   settlements, refusal history, and HOS. Add a domicile FK and a home-time
   commitment, and attrition risk becomes forecastable per driver — letting the
   planner trade a few dollars of deadhead against a promised home-time date.
   This is a category-defining capability, not a report.
4. **Predictive maintenance from the telematics feed.** Samsara already delivers
   odometer, fuel level, and DVIR defects that currently dead-end. Adding the
   maintenance domain plus fault-code ingest converts the largest Tier-1 gap into
   a differentiator: predict failures, pre-position the unit, and block dispatch
   automatically through the eligibility engine that already exists.
5. **Explainability ledger.** Rate receipts, agent evidence, and dispatch score
   breakdowns already exist in isolation. Unify them into a single "why" surface
   — why this rate, why this driver, why this charge, why this exception —
   queryable and exportable. Legacy systems cannot explain anything; this is a
   sales weapon.
6. **Network digital twin.** `ratesimulationservice` replays history against
   candidate rules, and `detentionservice/backtest.go` replays detention policy.
   Generalize that into whole-network what-if analysis: add five trucks in
   Laredo, drop this customer, move a domicile — replayed against real history.
7. **Self-serve quote-to-cash for shippers.** Expose the rate engine through a
   customer portal for instant quoting off the customer's own agreements, then
   book, track, POD, and pay. Rare in the market, and it closes the customer
   portal gap at the same time.
8. **Operations copilot with a real tool surface.** Expand `agenttoolservice`
   from five billing tools to plan, tender, re-rate, notify, and quote, keeping
   the existing autonomy tiers and shadow mode.
9. **Carbon and CARB reporting.** Zero presence in the codebase, increasingly
   demanded by shippers in RFPs, and CARB Clean Truck Check is mandatory in
   California. Cheap to add once fuel purchases and jurisdictional mileage exist.

## 6. Suggested Sequence

### P0 — Compliance and Correctness

A carrier can be fined or fail an audit without these.

1. Drug and alcohol program plus FMCSA Clearinghouse (2.3)
2. Maintenance, work orders, PM schedules, the DVIR defect loop, and equipment
   dispatch eligibility (2.1)
3. IFTA, fuel purchases, and fuel card ingest; IRP after (2.4)
4. Fiscal close accounting — closing entries and retained earnings (3)
5. Native MFA, TOTP and WebAuthn — the model already exists (3)

### P1 — Commercial Blockers

6. Customer portal (track, POD, invoice, quote) plus live ETA
7. Outbound webhooks, a widened public API, and distributed rate limiting
8. Safety module: accidents, incidents, cargo claims and OS&D, roadside
   inspections
9. Consolidated invoicing and invoice tax
10. Point the report catalog at the financial tables — small change, large payoff

### P2 — Depth and Reach

11. Carrier portal; carrier vetting feed (FMCSA and COI monitoring)
12. AP: vendors, bills, three-way match; ERP/GL export; ACH/NACHA payout; 1099-NEC
13. A second telematics provider (Motive) against the existing interface
14. Dock and appointment scheduling; yard and trailer pools
15. Finish or delete the orphaned consolidation and dedicated-lane schema
16. Cross-border: provinces, ACE eManifest, customs
17. Kubernetes/Helm, a realtime fallback, and a responsive web shell

### P3 — Differentiators

The capabilities in section 5, once the data foundation above is in place.

## Bottom Line

Trenova is no longer an early-stage platform. Rating, dispatch optimization, HOS
projection, detention, settlements, and EDI are at or beyond commercial parity,
and several of them are ahead of what incumbents ship.

What is missing is not sophistication — it is whole departments. Maintenance,
safety, the drug and alcohol program, fuel tax, and accounts payable are absent
rather than shallow, and three of those five carry direct regulatory exposure for
an asset-based carrier. The external surface is the second theme: no customer
portal, no carrier portal, no outbound webhooks, and no live ETA means the system
is excellent at running a fleet and poor at letting anyone outside the company
see it.

The strategic read is that the compliance gaps must close first because they are
liabilities, the external surface second because it is what customers buy, and
the differentiators in section 5 third — but those differentiators are unusually
cheap to reach, because the planner, cost engine, HOS projection, and agent
governance model they depend on are already built.

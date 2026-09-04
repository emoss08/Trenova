# Enterprise Gap-Closure Plan

Companion to `research-docs/trenova-enterprise-tms-gap-analysis-2026-09.md`, which
contains the evidence for every gap referenced here. That document says *what is
missing*. This one says *what to build*, at the level of structs, columns,
endpoints, and acceptance criteria.

**Audience:** implementation agents working one workstream at a time.

---

## 0. How to use this document

### 0.1 Working protocol

1. Claim exactly one workstream. Do not start a second until the first meets its
   acceptance criteria.
2. Read the referenced existing files before writing anything. Every workstream
   names a **reference implementation** — an existing module that already solves
   the same shape of problem. Match it rather than inventing a new pattern.
3. Work the sub-steps in the order given. They are ordered so the code compiles
   at each step.
4. A workstream is done when its acceptance criteria pass, `task lint` is clean,
   and `task test` is green.
5. Update the status table in section 6 as part of your final commit.

### 0.2 Conventions (enforced — read `CLAUDE.md` in full before starting)

Backend:
- Hexagonal layering. Domain in `internal/core/domain/`, ports in
  `internal/core/ports/`, business logic in `internal/core/services/`, adapters
  in `internal/infrastructure/`. A domain package never imports infrastructure.
- Bun ORM. Always use generated column helpers from `services/tms/pkg/buncolgen/`
  — never hand-write column references. Read `docs/bun/buncolgen.md` first.
- `sonic` for JSON. `encoding/json` is forbidden by lint.
- Ozzo validation for struct validation; `pkg/errortypes` for structured errors.
- Uber FX for DI. No comments in Go code.
- Utilities go in `shared/` (e.g. `shared/stringutils`), never inline in domain
  or service files. Reuse before writing — check `shared/` first.
- Group function parameters into a named struct once you exceed 3–4.
- `t.Context()` in tests, not `context.Background()`.

Frontend:
- Named exports. `useWatch`, never `watch()`. Autocomplete fields for entity
  references. Badges from the shared status-badge component.
- GraphQL for list reads, REST for detail reads and mutations.
- Shared code (used by both `apps/web` and `apps/dash`) goes in
  `client/packages/shared/src/` and imports as `@trenova/shared/...`.

### 0.3 Codegen — run these or CI fails

Any change to the listed inputs requires the matching command, run from
`services/tms/`:

| You changed | Run | Regenerates |
|---|---|---|
| A domain struct | `task generate-columns` | `pkg/buncolgen/*_gen.go`, `domain/*/fieldmap_gen.go` |
| `internal/api/graphql/schema/*.graphqls` | `task gqlgen` | `graphql/generated`, `gqlmodel`, `resolver` |
| A repository interface in `ports/` | `task generate-mocks` | `internal/testutil/mocks/` — **regenerate only the affected mock, never the whole codebase** |
| A handler's swagger annotations | `task docs-generate` | `docs/openapi-3.json`, `swagger.json` |
| A Postgres migration | `task sqlite-convert` | `internal/infrastructure/sqlite/migrations/` |
| A seed file | `task generate-seeds` | `pkg/seedhelpers/seed_ids_gen.go` |
| `reportcatalog.yml` | `task generate-reportcatalog` | report catalog generated code |

After GraphQL schema changes also run client codegen:
`pnpm --filter @trenova/graphql codegen`.

### 0.4 Shared files — merge-conflict hotspots

These files are touched by *every* workstream that adds a domain. If multiple
agents run in parallel, expect conflicts here. Append at the end of the relevant
block; never reorder existing entries.

- `internal/bootstrap/modules/repositories.go` — repository constructors
- `internal/bootstrap/modules/api/services.go` — service constructors
- `internal/bootstrap/modules/api/handlers.go` — handler constructors
- `internal/bootstrap/modules/validators.go` — validator constructors
- `internal/api/router.go` — handler struct field, fx param, `RegisterRoutes` call
- `internal/core/domain/permission/resource_gen.go` — `Resource` constants
- `internal/core/domain/permission/registry.go` — `ResourceDefinition` registration
- `client/apps/web/src/router.tsx` — lazy route
- `client/apps/web/src/config/navigation.config.ts` — sidebar entry

---

## 1. The domain recipe

Every workstream that introduces a new entity follows this. **Reference
implementation: `holdreason`** — small, complete, and current. Read all of it
before starting:

```
internal/core/domain/holdreason/holdreason.go
internal/core/ports/repositories/holdreason.go
internal/infrastructure/postgres/repositories/holdreasonrepository/holdreason.go
internal/core/services/holdreasonservice/{service,validator}.go
internal/api/handlers/holdreasonhandler/handler.go
```

For a domain with sub-entities and lifecycle transitions, read `detention` or
`permit` instead — they show multi-file domains, derivation, and Temporal wiring.

### 1.1 Migration

Create `internal/infrastructure/postgres/migrations/<UTC timestamp>_<name>.tx.up.sql`
and a matching `.tx.down.sql`. Timestamp format `YYYYMMDDHHMMSS`; use a value
later than every existing migration.

Rules observed across the existing set:
- Enums are Postgres types: `CREATE TYPE "x_status_enum" AS ENUM(...)`.
- Separate every statement with `--bun:split`.
- Every tenanted table carries `id varchar(100)`, `business_unit_id varchar(100)`,
  `organization_id varchar(100)`, all `NOT NULL`, composite PK
  `(id, business_unit_id, organization_id)`.
- Timestamps are `bigint` epoch seconds with
  `DEFAULT extract(epoch from current_timestamp)::bigint`.
- Money is `NUMERIC(19,4)` plus a `bigint` minor-unit column where the value
  posts to the GL. Follow `domain/invoice/invoice.go`.
- Add a `search_vector tsvector` column and a GIN index if the entity is
  searchable.
- The `.down.sql` must drop tables **and** enum types, in reverse order.

Then run `task sqlite-convert` and commit the generated SQLite migrations.

### 1.2 Domain struct

In `internal/core/domain/<name>/<name>.go`. Copy the shape from
`holdreason.go` exactly:

- Compile-time interface assertions: `bun.BeforeAppendModelHook`,
  `validationframework.TenantedEntity`, `domaintypes.PostgresSearchable`,
  `pagination.CursorEntity`. Add `customfield.CustomFieldsSupporter` if the
  entity should accept custom fields.
- Embed `bun.BaseModel` with `bun:"table:<table>,alias:<short>"` and
  `pagination.CursorValueSet` with `bun:",embed"`.
- Implement `Validate(multiErr *errortypes.MultiError)` using
  `validation.ValidateStruct` with human-readable messages — these surface
  directly in the UI form.
- Implement `GetID`, `GetCreatedAt`, `GetTableName`, `GetOrganizationID`,
  `GetBusinessUnitID`, `GetPostgresSearchConfig`.
- Implement `BeforeAppendModel` to assign a prefixed PULID on insert
  (`pulid.MustNew("prefix_")`) and stamp `CreatedAt` / `UpdatedAt`.
- Put enums in `enums.go` in the same package, as string constants with an
  `IsValid()` method.

Run `task generate-columns`, which produces `fieldmap_gen.go` in the domain
package and `pkg/buncolgen/<name>_gen.go`.

### 1.3 Repository port and adapter

Port in `internal/core/ports/repositories/<name>.go`: request structs plus an
interface with `List`, `ListConnection`, `GetByID`, `Create`, `Update`, and
`SelectOptions` as applicable.

Adapter in `internal/infrastructure/postgres/repositories/<name>repository/`.
Use `querybuilder.ApplyFilters` / `ApplyCursorFilters` with the buncolgen table
alias. Log with a named logger (`postgres.<name>-repository`). Handle optimistic
locking through the `Version` column — follow the existing `Update` method.

Run `task generate-mocks` for the new interface only.

### 1.4 Service and validator

`internal/core/services/<name>service/service.go` — `Params` struct with
`fx.In`, constructor `New(p Params) *Service`, and methods delegating to the
repository. Mutating methods take `userID pulid.ID` and emit an audit entry via
`services.AuditService`.

`validator.go` — build a
`validationframework.NewTenantedValidatorBuilder[*T]()` with `WithModelName`,
`WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(...))`,
`WithUniqueField(...)` per unique constraint, and `WithCustomRule(...)` for
cross-field rules. Expose `ValidateCreate` and `ValidateUpdate`.

### 1.5 Permissions

Add a constant to `internal/core/domain/permission/resource_gen.go` under the
correct category comment, then register it in `registry.go`:

```go
_ = r.Register(&ResourceDefinition{
    Resource:           ResourceX.String(),
    DisplayName:        "X",
    Description:        "...",
    Category:           "Operations",   // or Administration, Accounting, Compliance
    Operations:         standardOps,    // or standardOpsWithDelete, readOnlyOps
    DefaultSensitivity: SensitivityInternal,
})
```

Pick `SensitivityConfidential` for anything containing medical, test-result, or
personally identifying data — this drives audit masking automatically.

### 1.6 Handler and route

`internal/api/handlers/<name>handler/handler.go`. Register a route group, guard
every route with
`h.pm.RequirePermission(permission.ResourceX.String(), permission.OpRead|OpCreate|OpUpdate)`,
and write full swagger annotations on each method (they generate the OpenAPI
spec). Add `select-options` sub-routes if the entity appears in a dropdown.

Wire into `internal/api/router.go`: import, struct field, fx params field,
assignment in the constructor, and a `RegisterRoutes` call in the authenticated
group.

### 1.7 GraphQL (list reads)

Add `internal/api/graphql/schema/<name>.graphqls` with a type, a connection, and
a `extend type Query`. Add a `<name>mapping.go` in `resolver/` converting the
domain type to the gqlmodel. Run `task gqlgen`, then
`pnpm --filter @trenova/graphql codegen`.

### 1.8 Client

- `client/apps/web/src/services/<name>.ts` — REST client
- `client/apps/web/src/lib/graphql/<name>-table.ts` — list query
- `client/apps/web/src/routes/<name>/page.tsx` plus
  `_components/{<name>-table,<name>-form,<name>-panel}.tsx`
- Register in `router.tsx` (lazy import) and `config/navigation.config.ts`
- Add enum label maps to `client/apps/web/src/lib/choices.ts`

### 1.9 Tests

- Domain: table-driven validation tests covering each rule and boundary.
- Service: mock the repository; cover the validator paths.
- Repository: integration test against the test database (`task test-integration`).
- Handler: permission enforcement — assert 403 without the permission.
- Tenant isolation: assert an entity from org A is never returned to org B.

---

## 2. P0 workstreams — compliance and correctness

These close gaps that create regulatory or financial exposure. Do them first.

### WS1 — Drug & alcohol program and FMCSA Clearinghouse

**Gap:** analysis §2.3. The entire program today is
`DispatchControl.EnforceDrugAndAlcoholCompliance` plus
`WorkerProfile.LastDrugTest`, enforced by `evaluateDrugAndAlcohol` at
`internal/core/services/dispatcheligibility/worker.go:111`. Read that function
before starting — its condition is
`profile.LastDrugTest > 0 && profile.LastDrugTest <= profile.HireDate`, so it
only fires when a test exists *and* predates hire. **A driver with no
pre-employment test recorded at all passes silently.** There are no test records
and no Clearinghouse concept anywhere in the repository.

**Reference implementation:** `domain/permit/` for the requirement/satisfaction
pattern; `temporaljobs/compliancejobs/` for the sweep.

#### 1.1 Domain — `internal/core/domain/drugalcohol/`

`test.go` — `DrugAlcoholTest`:

| Field | Type | Notes |
|---|---|---|
| `ID`, `BusinessUnitID`, `OrganizationID` | `pulid.ID` | prefix `dat_` |
| `WorkerID` | `pulid.ID` | notnull |
| `TestType` | `TestType` | enum below |
| `Substance` | `Substance` | `Drug`, `Alcohol`, `Both` |
| `Reason` | `string` | free text, required for `ReasonableSuspicion` |
| `CollectionSiteName` | `string` | |
| `CollectedAt` | `int64` | notnull |
| `ResultReceivedAt` | `*int64` | |
| `Result` | `Result` | enum below |
| `MRODecision` | `string` | Medical Review Officer note |
| `MROVerifiedAt` | `*int64` | |
| `ClearinghouseReported` | `bool` | violations must be reported within 3 days |
| `ClearinghouseReportedAt` | `*int64` | |
| `DocumentID` | `*pulid.ID` | chain-of-custody form |
| `Notes` | `string` | |
| `Version`, `CreatedAt`, `UpdatedAt` | | standard |

`enums.go`:
- `TestType`: `PreEmployment`, `Random`, `PostAccident`, `ReasonableSuspicion`,
  `ReturnToDuty`, `FollowUp`
- `Result`: `Pending`, `Negative`, `Positive`, `Refusal`, `Adulterated`,
  `Substituted`, `Cancelled`, `NegativeDilute`
- `Substance`: `Drug`, `Alcohol`, `Both`

Add `func (r Result) IsViolation() bool` returning true for `Positive`,
`Refusal`, `Adulterated`, `Substituted`. This single method drives eligibility,
Clearinghouse reporting, and the return-to-duty requirement — do not duplicate
the logic.

`pool.go` — `RandomTestingPool`: `Name`, `Year`, `DrugTestRatePercent`,
`AlcoholTestRatePercent`, `Active`. `pool_member.go` — `PoolMember`:
`PoolID`, `WorkerID`, `AddedAt`, `RemovedAt`.

`selection.go` — `RandomSelection`: `PoolID`, `WorkerID`, `PeriodStart`,
`PeriodEnd`, `SelectedAt`, `Substance`, `Status`
(`Selected`, `Completed`, `Excused`, `Missed`), `TestID *pulid.ID`.

`clearinghouse.go` — `ClearinghouseQuery`: `WorkerID`, `QueryType`
(`FullPreEmployment`, `LimitedAnnual`), `ConsentObtainedAt`, `QueriedAt`,
`Result` (`Clear`, `ProhibitedStatus`, `Pending`), `ExpiresAt`,
`DocumentID *pulid.ID`.

#### 1.2 Eligibility integration — the point of the workstream

In `internal/core/services/dispatcheligibility/`, add to `finding.go`:

```go
CodeDrugTestPositive          = "driver.drug_test_positive"
CodeDrugTestRefusal           = "driver.drug_test_refusal"
CodeReturnToDutyIncomplete    = "driver.return_to_duty_incomplete"
CodeRandomSelectionOverdue    = "driver.random_selection_overdue"
CodeClearinghouseNotQueried   = "driver.clearinghouse_not_queried"
CodeClearinghouseProhibited   = "driver.clearinghouse_prohibited"
```

Extend `worker.go` to **replace** the `LastDrugTest > HireDate` check with:

1. Latest `PreEmployment` test exists with `Result == Negative` before the first
   dispatch — else `CodeDrugTestPositive` / block. Cite 49 CFR 382.301.
2. No unresolved violation: if the most recent violating test has no subsequent
   `ReturnToDuty` negative, block with `CodeReturnToDutyIncomplete`. Cite
   49 CFR 382.503.
3. Clearinghouse full query on hire and limited query within 12 months — else
   `CodeClearinghouseNotQueried`. Cite 49 CFR 382.701.
4. `ProhibitedStatus` from the most recent query blocks unconditionally with
   `CodeClearinghouseProhibited`.
5. An overdue `RandomSelection` (past `PeriodEnd`, status `Selected`) warns —
   severity from `dispatchcontrol.ComplianceEnforcementLevel` via
   `SeverityForEnforcement`, matching how the existing MVR check behaves.

Every finding must set `Regulation` — the field exists on `Finding` and is
already surfaced in the dispatch UI.

#### 1.3 Random selection job

`internal/core/temporaljobs/compliancejobs/` — add a
`RandomSelectionWorkflow` scheduled quarterly. For each active pool, compute the
required count from the rate percentage against average pool size, select
uniformly at random without replacement, persist `RandomSelection` rows, and
notify the compliance role via `notificationservice`.

Extend the existing daily `CredentialExpirySweepWorkflow` to also flag
Clearinghouse annual queries coming due within 30 days.

#### 1.4 Surfaces

REST `/api/v1/drug-alcohol-tests`, `/random-testing-pools`,
`/clearinghouse-queries`. GraphQL list schema. New permission resources
`drug_alcohol_test`, `random_testing_pool`, `clearinghouse_query` — all
`SensitivityConfidential`.

Client routes `drug-alcohol-test`, `random-testing-pool`, plus a
"Drug & Alcohol" tab on the worker detail panel.

Add a canned report `drug-alcohol-program-status` to
`internal/core/services/reporting/canned/library_compliance.go` following the
existing `driverQualificationStatus` builder.

**Acceptance:**
- A worker with a positive test and no completed return-to-duty cannot be
  assigned; the API returns a 422 whose message cites 49 CFR 382.503.
- A worker with no negative pre-employment test cannot be assigned — including
  the case where **no test is recorded at all**, which the current check misses
  entirely.
- A worker with a Clearinghouse `ProhibitedStatus` cannot be assigned regardless
  of enforcement level.
- The quarterly workflow produces selections matching the configured rate ±1.
- Test results never appear in audit logs in plaintext — verify the
  `SensitivityConfidential` masking with a test.
- The old `evaluateDrugAndAlcohol` body is replaced, not left running alongside
  the new rules.

**Out of scope:** consortium/TPA integration, actual Clearinghouse API calls
(record the query and its evidence; the query itself stays manual for now).

---

### WS2 — Maintenance, DVIR defect loop, equipment condition gating

**Gap:** analysis §2.1. No maintenance domain exists.
`EquipmentStatus.AtMaintenance` (`pkg/domaintypes/enums.go`) points at nothing.
DVIRs land in `domain/telematics/vehicleinspection.go` carrying
`unresolved_defect_count` and dead-end.

Note precisely what *does* exist so you do not rebuild it: `dispatcheligibility`
already gates on equipment **availability and type** (`equipment.tractor_unavailable`,
`equipment.tractor_type_mismatch`, `equipment.tractor_in_use`, and the continuity
codes) via `services/equipmentavailabilityhelper/helper.go`. The gap is
**condition-based** gating — PM overdue, out of service, expired registration,
failed inspection.

`SequenceTypeWorkOrder` already exists in `domain/tenant/enums.go`. Use it;
do not add a new sequence type.

#### 2.1 Domain — `internal/core/domain/maintenance/`

`workorder.go` — `WorkOrder`: prefix `wo_`.
`Number` (from the existing work-order sequence), `EquipmentType`
(`Tractor` | `Trailer`), `TractorID *pulid.ID`, `TrailerID *pulid.ID` (exactly
one non-nil — enforce in `Validate`), `Status`, `Priority`, `Source`,
`OpenedAt`, `ScheduledFor *int64`, `StartedAt *int64`, `CompletedAt *int64`,
`OdometerMiles *int64`, `EngineHours *decimal.NullDecimal`, `VendorName`,
`VendorInvoiceNumber`, `TotalPartsCost`, `TotalLaborCost`, `TotalCost` (plus
minor-unit columns), `TakesOutOfService bool`, `Notes`.

- `Status`: `Open`, `Scheduled`, `InProgress`, `AwaitingParts`, `Completed`,
  `Canceled`
- `Priority`: `Low`, `Routine`, `High`, `Critical`
- `Source`: `PreventiveMaintenance`, `DVIRDefect`, `RoadsideInspection`,
  `DriverReport`, `Breakdown`, `Recall`, `Manual`

`workorderline.go` — `WorkOrderLine`: `WorkOrderID`, `LineType`
(`Part` | `Labor` | `Fee`), `Description`, `PartNumber`, `Quantity`,
`UnitCost`, `TotalCost`, `LaborHours`.

`defect.go` — `Defect`: `EquipmentType`, `TractorID`/`TrailerID`,
`Source` (`DVIR` | `RoadsideInspection` | `AnnualInspection` | `Manual`),
`SourceInspectionID *pulid.ID` (FK to `vehicle_inspections`), `Description`,
`Severity` (`Minor` | `Major` | `OutOfService`), `ReportedAt`,
`ReportedByWorkerID`, `Status` (`Open` | `InWorkOrder` | `Repaired` | `Deferred`),
`WorkOrderID *pulid.ID`, `RepairedAt *int64`, `RepairedNote`.

`pmschedule.go` — `PMSchedule`: `Name`, `EquipmentTypeID *pulid.ID`,
`TractorID`/`TrailerID` (nullable — a schedule applies to a type or one unit),
`IntervalKind` (`Miles` | `EngineHours` | `Days`), `IntervalValue`,
`WarnThreshold`, `LastPerformedAt *int64`, `LastPerformedOdometer *int64`,
`NextDueAt *int64`, `NextDueOdometer *int64`, `Active`.

#### 2.2 Odometer and engine hours on equipment

Add to `domain/tractor/tractor.go` and `domain/trailer/trailer.go`:
`OdometerMiles *int64`, `EngineHours *decimal.NullDecimal`,
`OdometerUpdatedAt *int64`, `OdometerSource` (`Telematics` | `Manual` |
`WorkOrder`).

Populate from telematics: `telematics_vehicle_positions` already ingests
`odometerMeters`. Extend `internal/core/services/telematicsservice/sweep.go` to
write the latest reading onto the tractor, converting metres to miles through a
helper in `shared/` — do not inline the conversion.

#### 2.3 DVIR defect ingestion — closes the dead-end

In `internal/core/services/telematicsservice/`, add `defectingest.go`: when a
`VehicleInspection` arrives with `unresolved_defect_count > 0`, create a
`Defect` row per JSONB defect entry, idempotently keyed on
`(source_inspection_id, description)` so re-polling does not duplicate.
A defect whose provider payload marks it out-of-service creates the `Defect`
with `SeverityOutOfService` and sets the equipment's `Status` to `OutOfService`.

#### 2.4 Condition gating — the point of the workstream

Add to `dispatcheligibility/finding.go`:

```go
CodeEquipmentOutOfService       = "equipment.out_of_service"
CodeEquipmentOpenOOSDefect      = "equipment.open_oos_defect"
CodeEquipmentPMOverdue          = "equipment.pm_overdue"
CodeEquipmentPMDueSoon          = "equipment.pm_due_soon"
CodeEquipmentRegistrationExpired = "equipment.registration_expired"
CodeEquipmentInspectionExpired  = "equipment.inspection_expired"
```

Create `dispatcheligibility/equipment.go` and call it from `Evaluate` in
`evaluate.go`, which today evaluates only worker, HOS, availability, and holds.
Rules:
- Equipment `Status == OutOfService` → block.
- Any open `Defect` with `SeverityOutOfService` → block.
- PM past due → severity from `ComplianceEnforcementLevel`; within warn
  threshold → `SeverityWarn`.
- `RegistrationExpiry` in the past → block. Cite 49 CFR 396.
- Annual inspection older than 12 months (`Trailer.LastInspectionDate`) → block.

#### 2.5 PM due sweep

New `internal/core/temporaljobs/maintenancejobs/` following
`compliancejobs/`. Daily: recompute `NextDueAt` / `NextDueOdometer` from the
latest odometer, flag due and overdue units, notify the maintenance role.

Separately, extend `compliancejobs` to sweep **equipment** registration and
inspection expiry — today it sweeps drivers only. This is the fix for the
"equipment renewals have reports but no workflow" finding.

#### 2.6 Surfaces

REST `/api/v1/work-orders`, `/defects`, `/pm-schedules`. Permission resources
`work_order`, `defect`, `pm_schedule`, category `Operations`. Client routes
`work-order`, `pm-schedule`, plus a Maintenance tab on tractor and trailer
detail panels.

Add canned reports `maintenance-cost-by-unit` and `pm-compliance` to
`reporting/canned/library_fleet.go`. Register `work_orders` and `defects` in
`reportcatalog.yml`, then `task generate-reportcatalog`.

**Acceptance:**
- A tractor with an open out-of-service defect cannot be assigned.
- A trailer with an expired annual inspection cannot be assigned, citing 49 CFR 396.
- Completing a work order that resolves the last OOS defect returns the unit to
  `Available` and it becomes assignable again in the same request cycle.
- A Samsara DVIR with two defects creates exactly two `Defect` rows, and
  re-polling the same DVIR creates none.
- PM due dates recompute from telematics odometer without manual entry.
- Maintenance cost per unit is reportable through the report builder.

**Out of scope:** parts inventory with stock levels, warranty claim tracking,
tire-position management. Model `WorkOrderLine` so these can be layered later.

---

### WS3 — Fuel purchases and IFTA

**Gap:** analysis §2.4. Existing fuel code (`fuel_indices`,
`fuel_surcharge_programs`) is **revenue**, not spend — do not modify it. Every
input for IFTA already exists (`stored_mileages`, `distance_calculations`,
telematics odometer, `usstate`); nothing joins them.

**Dependency:** WS2 §2.2 (odometer on equipment) should land first. If running in
parallel, coordinate on the tractor struct change.

#### 3.1 Domain — `internal/core/domain/fuelpurchase/`

`purchase.go` — `FuelPurchase`: prefix `fp_`.
`TractorID`, `WorkerID *pulid.ID`, `StateID` (jurisdiction of purchase — this is
what makes IFTA possible), `PurchasedAt`, `VendorName`, `LocationCity`,
`FuelType` (`Diesel` | `DEF` | `Gasoline` | `Reefer`), `Gallons decimal`,
`PricePerGallon decimal`, `TotalAmount decimal` + minor, `OdometerMiles *int64`,
`CardLastFour`, `TransactionReference` (unique per org — the idempotency key for
card imports), `Source` (`Manual` | `CardImport` | `DriverPortal`),
`IsTaxPaid bool` (bulk purchases are not), `DocumentID *pulid.ID`.

`card.go` — `FuelCard`: `CardNumberLastFour`, `Provider`
(`Comdata` | `EFS` | `WEX` | `Other`), `AssignedWorkerID`, `AssignedTractorID`,
`Status` (`Active` | `Suspended` | `Cancelled`).

#### 3.2 Domain — `internal/core/domain/ifta/`

`jurisdictionmileage.go` — `JurisdictionMileage`: `TractorID`, `StateID`,
`PeriodYear`, `PeriodQuarter`, `TotalMiles decimal`, `TaxableMiles decimal`,
`Source` (`Telematics` | `RouteCalculation` | `Manual`), `RecalculatedAt`.

`return.go` — `IFTAReturn`: `PeriodYear`, `PeriodQuarter`, `Status`
(`Draft` | `Finalized` | `Filed`), `TotalMiles`, `TotalGallons`,
`FleetMPG decimal`, `FinalizedAt`, `FinalizedByID`, `FiledAt`.

`returnline.go` — `IFTAReturnLine`: one row per jurisdiction —
`ReturnID`, `StateID`, `TotalMiles`, `TaxableMiles`, `TaxPaidGallons`,
`NetTaxableGallons`, `TaxRate decimal`, `TaxDue decimal` + minor,
`SurchargeDue decimal`.

`taxrate.go` — `IFTATaxRate`: `StateID`, `PeriodYear`, `PeriodQuarter`,
`FuelType`, `RatePerGallon decimal`, `SurchargeRatePerGallon decimal`.
Seed known rates; they change quarterly, so the entity must be
effective-dated rather than a constant.

#### 3.3 Mileage attribution — the hard part

`internal/core/services/iftaservice/mileage.go`. Two strategies behind one port
so the second can be added without touching callers:

1. **Route-based (implement first).** For each completed `ShipmentMove` in the
   period, take the resolved route and apportion distance across the states it
   crosses. `shared/pcmiler` already returns state-by-state breakdowns for
   routed distance — check `shared/pcmiler/payload.go` before writing anything;
   if the breakdown is already requested, consume it rather than re-deriving.
2. **Telematics-based (stub the interface, implement later).** Attribute from
   `telematics_vehicle_positions` by point-in-polygon against state geometry.
   PostGIS is available and `domain/geofence` already does spatial work.

Include empty/deadhead moves — IFTA counts all miles, not just loaded. This is
the most common correctness error in the domain; add an explicit test for it.

#### 3.4 Return computation

`internal/core/services/iftaservice/compute.go`:
```
FleetMPG          = TotalMiles / TotalGallons          (fleet-wide, not per-unit)
NetTaxableGallons = TaxableMiles / FleetMPG − TaxPaidGallons
TaxDue            = NetTaxableGallons × TaxRate
```
A negative `NetTaxableGallons` is a credit, not an error — carry the sign.
Use `shared/decimalutils`; never float arithmetic on tax.

#### 3.5 Fuel card import

`internal/core/services/fuelpurchaseservice/import.go` — CSV import with a
dry-run stage. Follow `services/rateimportservice/` exactly: upload → read →
dry-run review → commit, with a 32 MB cap. Deduplicate on
`TransactionReference`.

Do **not** build live Comdata/EFS/WEX API clients in this workstream. CSV import
covers every provider and is what carriers actually use.

#### 3.6 Surfaces

REST `/api/v1/fuel-purchases`, `/fuel-cards`, `/ifta/returns`,
`/ifta/tax-rates`. Permission resources `fuel_purchase`, `fuel_card`,
`ifta_return`. Client routes `fuel-purchase`, `ifta-return` (quarterly worksheet
with per-jurisdiction lines and a CSV export). Add fuel-purchase entry to the
driver portal (`apps/dash`) behind a new
`allowFuelPurchaseEntry` flag in `driverportalservice/features.go`.

Register `fuel_purchases` and `ifta_jurisdiction_mileages` in
`reportcatalog.yml`.

**Acceptance:**
- A quarter's return computes per-jurisdiction net taxable gallons and tax due,
  and the jurisdiction miles sum to total fleet miles for the period.
- Deadhead miles are included in taxable miles — covered by an explicit test.
- Re-importing the same card CSV creates zero duplicate purchases.
- Finalizing a return locks its lines against edit; reopening requires a reason
  and writes an audit entry.
- Fleet MPG matches `TotalMiles / TotalGallons` to four decimal places.

**Out of scope:** IRP apportioned registration, e-filing to any state portal.

---

### WS4 — Fiscal close accounting

**Gap:** analysis §3. `fiscalyearservice.Close`
(`internal/core/services/fiscalyearservice/service.go:490-538`) validates, calls
`repo.Close`, writes an audit record, and stops. It generates **no closing
entries**. `DefaultRetainedEarningsAccountID` is defined in
`domain/tenant/accountingcontrol.go` and validated in
`accountingcontrolservice/validator.go:102-103` but is never written to.

This is the smallest P0 workstream and the highest-confidence one — the posting
machinery already exists and works.

**Reference implementation:**
`internal/infrastructure/postgres/repositories/journalpostingrepository/journalposting.go`
(`CreatePosting` → `aggregateLinesByAccount` → `upsertPeriodBalance` →
`updateGLAccountRunningBalance`), and `services/journalreversalservice/` for a
system-generated entry with provenance.

#### 4.1 Closing entry generation

New `internal/core/services/fiscalyearservice/closing.go`:

1. Verify every period in the year is `Closed`. If not, return blockers through
   the existing `fiscalclose.Result` / `fiscalcloseblockers` helper rather than a
   bare error — the UI already renders blockers.
2. Sum ending balances for every account whose `AccountType.Category` is
   `Revenue`, `CostOfRevenue`, or `Expense`.
3. Build one balanced closing journal entry:
   - Debit each revenue account by its credit balance
   - Credit each expense and cost-of-revenue account by its debit balance
   - Post the net (income or loss) to `DefaultRetainedEarningsAccountID`
4. Post it through the existing posting repository with a new
   `JournalSourceEventYearEndClose`, dated the last day of the fiscal year.
5. Generate opening balances for the next fiscal year: carry forward ending
   balances for `Asset`, `Liability`, and `Equity` accounts only. Nominal
   accounts open at zero.
6. Store the closing entry ID on the fiscal year so reopening can reverse it.

#### 4.2 Reopen

`ReopenYear` must reverse the closing entry through the existing
`journalreversalservice` — never delete it — and require a reason that lands in
the audit trail.

#### 4.3 Idempotency

Closing twice must be a no-op, not a double post. Guard on the stored closing
entry ID inside the same transaction as the status flip.

**Acceptance:**
- Closing a year with revenue 1,000,000 and expenses 900,000 produces a single
  balanced entry crediting retained earnings 100,000.
- After close, every revenue and expense account has a zero ending balance and
  every balance-sheet account carries forward unchanged.
- A loss year debits retained earnings.
- Closing a year with an open period returns blockers naming the period, and
  posts nothing.
- Closing twice posts once.
- Reopening reverses the closing entry and restores prior balances exactly.
- The trial balance for the first period of the new year balances to zero.

**Out of scope:** cash flow statement, comparative statements, consolidation.

---

### WS5 — Native MFA

**Gap:** analysis §3. `domain/iam/models.go:141` already defines
`MFAAuthenticator` with TOTP and WebAuthn kinds, an encrypted secret, and a
verified-before-enabled invariant. A permission resource is registered
(`registry.go:499`). Only `ListMFAAuthenticators` exists
(`iamservice/service.go:494`) — there is no enrollment, challenge, or verify
flow. `authservice/service.go:883` only *reads* `amr` claims from an upstream
IdP, so a customer without SSO cannot enforce 2FA at all.

The data model is done. This workstream is flows and UI.

#### 5.1 TOTP enrollment and verification

Extend `internal/core/services/iamservice/`, new file `mfa.go`:

- `BeginTOTPEnrollment(ctx, userID)` — generate a secret, encrypt it through the
  existing `encryptionservice` with a new `PurposeMFASecret`, persist the
  authenticator as unverified, return the otpauth URI and a QR payload.
- `CompleteTOTPEnrollment(ctx, userID, authenticatorID, code)` — verify, mark
  verified and enabled, generate ten single-use recovery codes stored as hashes
  (reuse `shared/hashutils`), return them once.
- `VerifyTOTP(ctx, userID, code)` — accept a ±1 step window; reject a code
  already consumed within the window to prevent replay.
- `DisableAuthenticator` — require a fresh verification first.
- `ConsumeRecoveryCode(ctx, userID, code)`.

Use an established TOTP library; do not hand-roll RFC 6238.

#### 5.2 Login integration

In `internal/core/services/authservice/service.go`, `Login` currently returns a
session directly. Add an intermediate state: when the user has a verified
authenticator or the organization enforces MFA, return an MFA challenge with a
short-lived (5 minute) token instead of a session. Add
`CompleteMFAChallenge(ctx, challengeToken, code)` which mints the real session
and sets `AuthenticatorAAL = 2` plus `MFAAuthenticatedAt`, matching what
`assuranceFromOIDCClaims` already sets for the IdP path.

Rate-limit verification attempts per user and lock after repeated failures.

#### 5.3 Org enforcement

Add `EnforceMFA bool` and `MFAGracePeriodDays int` to the organization auth
settings alongside the existing `EnforceSso` in `domain/tenant/ssoconfig.go`.
Users past the grace period must enrol before any other action.

#### 5.4 Surfaces

REST under the existing auth group: `POST /auth/mfa/enroll`,
`POST /auth/mfa/enroll/verify`, `POST /auth/mfa/challenge`,
`DELETE /auth/mfa/:id`, `POST /auth/mfa/recovery-codes`.

Client: an MFA section in user profile settings (QR, code entry, recovery-code
display and download) and an MFA step in the login flow.

**Acceptance:**
- A user can enrol a TOTP authenticator, scan the QR in a standard app, and
  verify.
- Login with MFA enabled requires a valid code; an invalid code does not mint a
  session.
- A recovery code works exactly once.
- With `EnforceMFA` on, a user without an authenticator is routed to enrolment
  and cannot reach any other route.
- Secrets are encrypted at rest — assert the raw column is not the plaintext
  secret.
- A replayed code within the same time step is rejected.
- The existing OIDC `amr` path still sets AAL 2 and is unaffected.

**Out of scope:** WebAuthn ceremonies (the model already supports the kind —
leave the enum value and implement TOTP only), SAML.

---

## 3. P1 workstreams — commercial blockers

Specified at the same level of detail once a P0 workstream frees an agent. Scope
summaries and the decisions already made:

### WS6 — Customer portal and live ETA
Two halves, buildable independently.

**Live ETA.** Add `EstimatedArrival *int64`, `ETASource`, `ETACalculatedAt` to
`domain/shipment/stop.go`. New `internal/core/services/etaservice/` recomputing
from the latest telematics position, remaining route distance, and the HOS
projection that already exists in `services/hosprojection/` — an ETA that
ignores a driver's remaining drive time is wrong, and the projection is already
there. Recompute on each position update and on stop actuals. Emit a
`shipmentevent` when the ETA crosses the appointment window so service failures
can be predicted rather than recorded after the fact.

**Portal.** New `apps/portal` in the client workspace, mirroring how
`apps/dash` is structured. Backend `customerportalservice` modelled directly on
`driverportalservice` — including its `features.go` per-tenant flag pattern and
`invitation.go` token flow. Scope: shipment list and detail, live ETA and
milestone timeline, POD download, invoice list and download, and a quote request
against the existing rate engine. Reuse the existing `CommentVisibilityCustomer`
and `ShipmentHold.VisibleToCustomer` flags, which currently have no consumer.

### WS7 — Outbound webhooks
New `domain/webhook/` — `WebhookSubscription` (URL, secret, event types, active,
retry policy) and `WebhookDelivery` (attempt log with status, response code,
next retry). Publish from the existing `shipmentevent` taxonomy — do not invent
a second event vocabulary. HMAC-SHA256 signature over the body with a timestamp
header. Delivery through a Temporal workflow with exponential backoff and a DLQ,
following `edijobs`. Add subscription management under admin.

### WS8 — Safety module
`domain/safety/` — `Accident` (DOT-recordable flag, preventability, injuries,
fatalities, tow-away, citations, drug-test linkage into WS1), `Incident`,
`CargoClaim` (OS&D: overage, shortage, damage, refusal, concealed damage; with
reserve, recovery, and links to shipment, invoice adjustment, and carrier),
`RoadsideInspection` (level 1–6, violations, OOS flags — feeding WS2 defects).
Links to `servicefailure` where an accident caused one.

### WS9 — Consolidated invoicing and invoice tax
**Consolidation:** the schema exists and is unused (`consolidation_groups`,
`consolidation_settings`, `SequenceTypeConsolidation`). Decide first — either
implement grouping in `billingqueueservice` producing one invoice with lines
across shipments, or delete the tables. Do not leave it as is.
**Tax:** add `TaxAmount` + minor and an `InvoiceTaxLine` child to
`domain/invoice/`, plus a `TaxRate` entity keyed on jurisdiction and effective
date. Required before any Canadian operation.

---

## 4. Standalone cleanups

Small, independent, safe to hand to any agent. Each is a single commit.

### C1 — Report catalog financial datasets
`internal/infrastructure/database/reportcatalog/reportcatalog.yml` has 27
datasets and none of `journal_entries`, `gl_accounts`, `gl_balances`,
`driver_settlements`, `customer_payments`, `bank_receipts`, `rate_agreements`.
The report compiler already works; it is simply not pointed at these tables.
Add them following the existing dataset shape, run
`task generate-reportcatalog`, and confirm row-level authorization in
`reporting/compiler/authorize.go` covers the new datasets — financial data must
not become readable to anyone who could not already read it through the domain
API. Add a test asserting that.

### C2 — Orphaned schema decision
Three artifacts imply features that do not exist. Each needs an explicit
decision recorded in the commit message:
- `consolidation_groups` / `consolidation_settings` — owned by WS9; leave until
  WS9 decides.
- `domain/dedicatedlane/` plus its three migrations — the only live reference is
  a permission constant at `handlers/analyticshandler/handler.go:34`. Delete the
  domain package, the migrations (with a new down migration; do not edit shipped
  migrations), and the permission constant, unless dedicated lanes are on the
  near-term roadmap.
- `hazmat_expirations` — created 2024-12-28, referenced by nothing. Delete.

### C3 — AP configuration honesty
`domain/tenant/accountingcontrol.go` declares
`ExpenseRecognitionOnVendorBillPost`, emits
`JournalSourceEventVendorBillPosted`, and points at `DefaultAPAccountID` for an
entity that does not exist. Until AP is built, either mark these clearly as
reserved in the validator's error messages or remove them. Do not leave
configuration that silently does nothing.

---

## 5. Cross-cutting requirements

Every workstream must satisfy these; they are not optional extras.

**Tenant isolation.** Every new query filters on `organization_id` and
`business_unit_id`. Add a test that an entity created in org A is invisible to
org B through the repository, the REST handler, and the GraphQL resolver.

**Audit.** Every mutating service method writes an audit entry through
`services.AuditService`. Anything containing medical, test-result, or PII must
register at `SensitivityConfidential` so the existing masking in `auditservice`
applies. Verify with a test that the plaintext does not appear in the audit row.

**Permissions.** No route without `RequirePermission`. Add a handler test
asserting 403 for a user lacking the permission.

**Errors.** Field-level validation through `errortypes.MultiError` so the
frontend form binds errors to inputs. Nested paths and array indices are
supported — use them for child collections.

**Money.** `NUMERIC(19,4)` plus a `bigint` minor-unit column wherever the value
posts to the GL. Use `shared/money` and `shared/decimalutils`. Never float.

**Time.** `bigint` epoch seconds via `shared/timeutils`. Never `time.Time` in a
Bun column.

**Migrations.** Postgres first, then `task sqlite-convert`, and commit both.
Never edit a migration that has shipped — add a new one.

---

## 6. Status

Update as part of your final commit for a workstream.

| ID | Workstream | Priority | Depends on | Status | Owner |
|---|---|---|---|---|---|
| WS1 | Drug & alcohol + Clearinghouse | P0 | — | Not started | |
| WS2 | Maintenance + DVIR + condition gating | P0 | — | Not started | |
| WS3 | Fuel purchases + IFTA | P0 | WS2 §2.2 | Not started | |
| WS4 | Fiscal close accounting | P0 | — | Not started | |
| WS5 | Native MFA | P0 | — | Not started | |
| WS6 | Customer portal + live ETA | P1 | — | Not started | |
| WS7 | Outbound webhooks | P1 | — | Not started | |
| WS8 | Safety module | P1 | WS1, WS2 | Not started | |
| WS9 | Consolidated invoicing + tax | P1 | — | Not started | |
| C1 | Report catalog financial datasets | — | — | Not started | |
| C2 | Orphaned schema decision | — | WS9 (partial) | Not started | |
| C3 | AP configuration honesty | — | — | Not started | |

WS1, WS2, WS4, and WS5 are fully independent and can run in parallel by four
agents, subject to the shared-file conflict policy in §0.4.

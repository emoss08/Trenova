# Worker file

The worker record page (`/hr/workers/$id`) draws its header, tab marks, section strips,
attention list, clearance runway and file grid from one read: `workerOverview(workerId)`. Each
tab keeps its own read for its lists. Paths are under `services/tms/`.

| Piece | Where |
|---|---|
| Schema | `internal/api/graphql/schema/worker_overview.graphqls` |
| Resolver and section gates | `internal/api/graphql/resolver/workeroverviewresolver` (`OverviewSections`) |
| Composition | `internal/modules/workforce/app/workeroverviewservice` (`Get`, `overview.go`) |
| Standing, concerns, clearance | `internal/modules/workforce/domain/workercompliance` (`standing.go`, `clearance.go`, `dispatch.go`) |
| Destinations and URL slugs | `internal/modules/workforce/domain/workerpage` |
| Dispatch verdicts | `internal/modules/workforce/ports/workerclearanceports`, implemented in `internal/modules/operations/app/dispatcheligibility/clearance.go` |

## The one read

- **A withheld section is null.** `OverviewSections` turns each section's own read permission
  into a flag (the permission engine also refuses a resource off the plan); the service never
  reads a section whose flag is off, and the standing, concerns and clearance are built only from
  what the reader may see. HOS and benefits also ask the plan gate
  (`integrations.samsara`, `workforce.benefits`). The DQ file reads credentials and the drug and
  alcohol standing for its own items without exposing them as sections.
- **The worker row is read once**, with its profile, state and (for the portal) its portal user,
  and handed to every summarizer that needs it (`SummaryFor`, `EntitlementFor`, `FileFor`).
  Time comes from `timeutils.Clock`; the standing, concerns and clearance share one `asOf`.
- **Bounded fan-out.** The section reads run in one `errgroup` limited to six at a time, so a
  single overview never holds more of the tenant's connection slots than that; the DQ file and the
  dispatch verdicts run after, because they need the credential summary and the testing standing.
  Every read is a single-worker read on an index that leads with the worker (credentials,
  training, safety events, tests, leave, shifts, pay assignment); documents and memories are
  count queries.
- **Right now.** `hos` maps the stored HOS state (`telematicsservice.WorkerHOSLink`, never a
  provider call): the drive, shift, cycle and break clocks with their limits, and `tractorCode`,
  read by primary key from the tractor the state names, only for a reader with `tractor` read.
  `nextTimeOff` is the earliest Requested or Approved time off whose end is on or after `asOf`
  (one row from the PTO list, `idx_worker_pto_tenant_worker_dates`), behind `worker_pto` read.
- **Concerns** are capped at 30 (`ConcernCap`); `concernCount` is the total before the cap.

## Concerns

One concern per record when the concern is about a record (a credential, a course, a safety
event, an employer, a checklist), carrying the record's name as `subject`, the dates the row
names (`occurredAt`, `dueAt`, `daysUntil`) and a destination that opens it. The server sends no
sentences; the client builds the words from the code (`client/apps/web/src/lib/worker-concerns.ts`).

Ranked by severity, then concerns that stop assignment, then soonest due. The standing is
unchanged by the new codes except where an existing rule makes one Critical:

| Code | Severity | Stops assignment |
|---|---|---|
| `NotEmployed` | Info (standing Blocked) | yes |
| `NotAssignable` | Critical (standing Blocked) | yes |
| `CredentialExpired`, `CredentialMissing` | Critical | by the clearance rules |
| `CredentialExpiring` | Warning | at its date, by the clearance rules |
| `TrainingExpired`, `TrainingMissing`, `TrainingOverdue` / `TrainingDue` | Critical / Warning | never |
| `SafetyRating` | Critical at risk, Warning on watch | never |
| `OutOfService` (one per out-of-service order in the scorecard window) | Critical | never |
| `ActiveDiscipline` | Warning | never |
| `OpenSafetyEvent` | Info | never |
| `ChecklistOpen` / `ChecklistOverdue` | Info / Warning | never |
| `ReviewOverdue` / `ReviewDue` (within 30 days) | Warning / Info | never |
| `DrugAlcoholProhibited` | Critical | when the dispatch check refuses it today |
| `ClearinghouseQueryOverdue` | Critical when the dispatch check stops on it, Warning otherwise | by the dispatch check |
| `ClearinghouseQueryDue` (within `ClearinghouseDueSoonDays`, the sweep's horizon) | Info | never |
| `EmploymentVerificationOutstanding` (one per employer, 49 CFR 391.23) | Warning past the 30-day deadline, Info before | never |

## Clearance

`BuildClearance` lays out every dated requirement: credential expiries, training due or expiry
dates (with `worker_training` read), the next Clearinghouse query (with `worker_dot_test` read)
and the next review (with `performance_review` read). It is null without `worker_credential`
read, because no verdict can be given without that evidence.

An item **stops assignment** only by the two existing verdicts:

1. **RequiredCredential**: `CredentialSummaryItem.BlocksWhenLapsed()`, the same predicate that
   makes the credential summary NonCompliant and so the standing Blocked.
2. **DispatchCompliance**: `EvaluateWorkerCompliance` run at the item's date. The service asks
   the operations module through `workerclearanceports.DispatchCheck`; for each probe (a
   credential whose type mirrors a profile date, and the Clearinghouse query) it sets that date
   on a copy of the profile and evaluates one second after it, with and without hazmat, at the
   tenant's enforcement level. A Block finding on that requirement stops; one that appears only
   with hazmat is `hazmatOnly`, drawn solid and never ends the cleared band. At the default
   `Warning` level no dated item stops by this rule.

Training and review items are always `Due`. Lanes: Medical credentials to Medical, Background to
Reviews and queries, every other credential category to License.

**Drug and alcohol stay with their readers.** For a reader without `worker_dot_test` read the
service strips the prohibition and Clearinghouse requirements from the dispatch verdicts
(`DispatchVerdicts.Without`). A refusal that came only from them still makes the worker not
cleared, reported as `DispatchHold` (the reader would see the driver refused at assignment
anyway); the `DrugAlcoholProhibited` and Clearinghouse concerns and the Clearinghouse runway item
are never built for that reader.

`cleared` is false when any block holds today: not employed, a dispatch hold (`!canBeAssigned`,
`!availableForDispatch` or `assignmentBlocked`), a required credential lapsed or missing, or the
dispatch check refusing today. A Blocked standing therefore always means not cleared.
`clearedThrough` is the first future stop (hazmat-only excluded), `notClearedSince` the earliest
lapsed stop. Items are ascending and capped at 60 (`itemsTruncated`).

## Activity

The Activity tab reads `auditEntriesBySubject(workerId)`: every entry whose subject is the worker,
its sub-records' included, limited to the kinds the reader may read ("A record's file" in
[security-audit.md](security-audit.md#a-records-file)).

## Destinations

`workerpage.Destination` is a tab, an optional view and an optional record. `Query()` turns it
into the page's search params and `workerpage.Path(id, dest)` into the address through
`productguide.RecordPath("worker", …)`; every server link to a worker (notifications, portal
alerts, watchtower items) is built this way.

| Tab | Views (slug) |
|---|---|
| `overview` (no `tab` param) | |
| `profile` | |
| `employment` | `history`, `checklists` |
| `qualification` | `credentials`, `dq-file`, `drug-and-alcohol` |
| `development` | `training`, `reviews` |
| `safety` | |
| `time` | `hours-of-service`, `schedule`, `time-off`, `leave` |
| `pay` | `pay-profile`, `benefits` |
| `files` | `documents`, `driver-portal`, `agent-memory` |
| `activity` | |

Record params mirror the client's `WORKER_RECORD_PARAMS`: `credential`, `training`, `review`,
`checklist`, `leave`, `test`, `employer`, `injury`; `event` on Employment and `safetyEvent` on
Safety; `timeOff` on Time and `request` (a profile change request) on Files. A record whose id is
unknown (a credential never recorded) opens the list instead.

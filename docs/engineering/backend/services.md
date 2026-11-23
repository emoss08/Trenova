# Application services

Services in `app/<name>service/` own a use case: they validate, decide, persist through ports,
and announce what happened. The reference is
`internal/modules/reference/app/commodityservice/`; paths are under `services/tms/`.

## Shape

- fx `Params` struct (`fx.In`) holding ports only: repositories, validators, `AuditService`,
  `kernelports.WorkflowStarter`, other services' ports, and the config sections it reads
  (`StorageConfig *storageconfig.Config`), never the whole `*config.Config`
  ([Configuration](README.md#configuration)). `New(p) *Service`, wired in the module's
  `module.go`.
- Depend on the narrowest port that does the job. A service that needs one method of a wide
  interface gets a narrower port declared next to its consumer.
- **fx builds every dependency, once.** A service never calls another service's, repository's
  or client's constructor, never keeps a `Deps` or `Dependencies` twin of its `Params` with a
  `NewWithDependencies`, and never builds a fallback when a field is nil. A dependency is
  required, so a nil check on it is noise ([go.md](go.md)), and nothing is `optional:"true"`: a
  port a process lacks gets a contract implementation or a split caller
  ([module-architecture.md](../module-architecture.md#inversion-ports)).
  One type serving several ports is one constructor bound with `fx.As` to each
  (`workflowstarter.New`, `agentqualitycontract.NewQualitySchedules`), never one constructor
  per port.
- **A default a bench replaces** is provided in the base graph and swapped with
  `fx.Decorate` (`temporaljobs.NewSDKLogger`, `toolaccess.NewIgnoredDisclosures`,
  `temporaljobs.NewQueueFilter`), not left optional.
- **Time** comes from `timeutils.Clock` (provided by the kernel module), not a `Now` field a test
  sets.
- **An immutable catalog** (`permissioncatalog.Default`, `templateregistry.Default`) is built once
  and provided by fx; code outside the graph (a domain table, a seed) reads the same instance
  through `Default()`.
- **Plan, then persist, then announce.** `PlanCreate`/`PlanUpdate` (`plan.go`) transform,
  validate and load the original as a `kernelports.RecordChange{Before, After}`; the service then
  writes through the repository and records the audit entry (`LogAction` with `WithDiff`). A pure
  plan step is what lets agents preview the same write ([proposal-previews.md](../proposal-previews.md)).
- **A write with consequences plans them as typed steps.** When a write sets off effects in
  other services (an employment event ends pay, settles time off, starts checklists), the plan
  is a list of steps, each carrying the effects it would cause and the function that causes
  them, tagged before, inside or after the transaction; the preview returns the steps' effects
  and the write runs exactly those steps, so the two cannot drift
  (`workeremploymentservice/plan.go`). Collaborators expose a read-only planning function that
  their write takes as input (`ptoledgerservice.PreviewTerminationSettlement`, then
  `SettleOnTermination(plan)`), never a second copy of the rule.
- Bulk changes are one plan over many records (`statuschange.Plan`,
  `auditservice.BuildBulkLogEntries`), one statement, one batch of audit entries.
- A use case that spans several repositories runs in one transaction on the context; the
  repositories join it through `Keel(ctx)`. A service that only needs the transaction depends on
  `kernelports.Transactor` (`Transact(ctx, opts, fn)`), not `kernelports.DBConnection`
  (`app/tractorservice`).

## Validation

- Each entity has a validator built once with `tenantedvalidation.NewBuilder[T]()`
  (`commodityservice/validator.go`). A run goes: the entity's own `Validate` and the built-in
  checks, then custom rules by stage (`StageBasic`, `StageDataIntegrity`, `StageBusinessRules`,
  `StageCompliance`) in the order they were added, with every uniqueness and reference check
  batched into one round trip at the start of `StageDataIntegrity`
  ([data-access.md](data-access.md#validators)). Hold it as
  `tenantedvalidation.EntityValidator[T]` (returns `error`) or `EntityReporter[T]` (returns the
  report, for a caller that adds its own checks or reads advisories), never the concrete
  generic type ([go.md](go.md#generics-and-build-time)).
- A rule that needs a read others need too (shipment control, the original record) loads it
  through `validationframework.Remember` with a `MemoKey`, once per run; `ValidateUpdateWithOriginal`
  hands rules the original through `validationframework.OriginalOf`.
- A rule reports refusals on the `MultiError` and returns any other failure as its error: the
  run stops and the caller gets a 500. Never turn a failed query into a field error.
- Field errors carry structured paths (`["items", 0, "name"]`, `pkg/fieldpath`) so the client
  puts each message on its field; `Err()` is the only way out of a report.
- Validate everything that crosses a trust boundary: request input, webhook payloads, files, model
  output an agent produced. Model output is input.
- Entered codes, names, emails and cities follow the tenant's data entry control. The entity says
  which fields are which in `ApplyTextFormat(dataentrycontrol.TextFormat)`, and its service calls
  `dataentrycontrolports.TextFormatter.Format` before validating, on create and update
  (`carrierservice/plan.go`, `workerservice`). A failed read stops the save. Sign-in emails are
  never formatted.

## Errors

`pkg/errortypes` maps errors to HTTP status codes and form fields:

```go
multiErr := errortypes.NewMultiError()
multiErr.Add("email", errortypes.ErrRequired, "Email is required")
if multiErr.HasErrors() {
    return multiErr
}
```

`NewBusinessError`, `NewNotFoundError`, `NewConflictError`, `NewAuthorizationError` and
`NewEditConflictError` cover the rest. Anything else is an internal error: wrap it with `%w` and
let the transport turn it into a 500 without leaking its text.

## Authorization

Permission checks happen at the edge (resolvers through `r.RequirePermission`, REST through
`pm.RequirePermission`); services receive a `RequestActor` and trust it. A service called from
somewhere without an edge check (a job, an agent tool, another service acting for a person) takes
the `permissionports.PermissionEngine` and checks itself. Data scopes ("own", "team") are applied
by RLS and `pkg/readscope`, not by `WHERE` clauses a service adds. Read
[permissions.md](../permissions.md) before adding a check, a resource or an operation.

A service that writes a base record a client sent back calls
`permissionports.PlanFieldKeeper.Keep` with the stored record before it validates, so a field the
plan withholds (`FieldFeatures`) keeps its stored value; see "What the plan withholds" in
[reference-views.md](../reference-views.md#what-the-plan-withholds)
(`operations/app/shipmenteffects/planfields.go`).

## Add-ons a base flow sets in motion

An add-on that is off does no work and changes nothing, even when a base action would set it in
motion ([permissions.md](../permissions.md#what-the-plan-includes)).

- **Base code asks the gate before calling into an add-on**, once, where it crosses: a
  `platformcatalogports.FeatureGate.Enabled` check that skips the add-on step and leaves the
  base flow whole (`shipmentcommercial.Calculator.syncDetentionCharge`: no detention charge is
  added or removed without Detention). An add-on service that a base flow or a base permission
  reaches also calls `platformcatalogports.RequireFeature` at its entry, so a direct caller gets
  `PLAN_RESTRICTED` (`freightoutlookservice.Service.Outlook`). A gate error is an error, handled
  as the step's own failure is at that site: never an allow, never a new hard failure.
- **The add-on's entry refuses before any work.** When the only refusal is deep inside (the
  model call, after records were read or a workflow started), add `RequireFeature` at the
  service's entry, once, where every caller passes (`dispatchautoassignservice.Plan`,
  `customertrackingservice`'s link and preview methods, `telematicsservice`'s reads). A base
  flow that would start add-on work asks the gate before it starts it
  (`paperworkcontract.Requester.RequestRead`, `dispatchcoveragewatch`). Work that an add-on's
  output feeds into another add-on (the planner records Trenova AI proposals) requires both.
- **An integration names its feature.** `integrationservice.planFeature` maps an integration
  type to the add-on feature it belongs to (Samsara, Google Maps); enabling, testing or reading
  its runtime configuration is refused or not ready without it.
- **An observer names its feature.** Every observer port embeds
  `platformcatalogports.FeatureObserver`, and each implementation returns its catalogue constant
  from `ObserverFeature()` in its own file (a base observer names its base feature). The code that
  notifies a list filters it with `platformcatalogports.Observing`, which asks the gate once per
  distinct feature and returns the gate errors to log as an observer failure
  (`shipmenteventservice.notifyObservers`). `TestEveryObserverNamesAKnownFeature` reads every
  declaration in the tree.

## After the write

- **Audit**: `auditService.LogAction(ctx, …)` with the request's `ctx`, inside the write's
  transaction when there is one, so the entry commits or rolls back with the change; never pass
  request metadata by hand, the service reads it from `ctx`. It also publishes the record-change
  invalidation after commit, so a CRUD service usually needs no separate realtime call.
- **Realtime** beyond that: `realtimeinvalidation.Publish`, or `PublishAfterCommit` inside a
  transaction ([realtime.md](../realtime.md)).
- **Side effects that must not happen on rollback** (emails, workflow starts, webhooks, cache
  bumps) go through `kernelports.AfterCommit(ctx, fn)`.
- **Slow or fan-out work** starts a workflow through `kernelports.WorkflowStarter` and returns;
  a request never waits on a third party it does not have to ([jobs.md](jobs.md)).
- **Access changes** call `SecurityAuditor.RecordChange` after commit
  ([security-audit.md](../security-audit.md)).
- **Retried writes** are safe through `Idempotency-Key` ([idempotency.md](../idempotency.md));
  a write that creates something a client may resend stores the key under a unique index.

## Every write is reachable by an agent

A new mutation or write route needs an agent tool, a reasoned exemption or a `pending` entry in
`internal/api/writecoverage/writecoverage.yml` ([agent-write-coverage.md](../agent-write-coverage.md)).
Design the service so the tool and the screen call the same method.

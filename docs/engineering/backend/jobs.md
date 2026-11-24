# Background work: Temporal

Anything slow, retried, scheduled, fanned out across tenants or waiting on the world runs on
Temporal, not in a request and not in a goroutine. The reference is
`internal/modules/operations/jobs/etajobs/`; layout and why it is separate from `module.go` are
in [module-architecture.md](../module-architecture.md). Agent work has its own runtime
([agent-runtime.md](../agent-runtime.md)).

## Layout

- `jobs/<name>jobs/module.go`: a `DomainConfig` (task queue from `pkg/temporaltype/queues.go`),
  the `[]registry.WorkflowDefinition`, `NewRegistry` and an `fx.Module` that adds the registry to
  `group:"worker_registries"`. Add the module to the module's `jobs/jobs.go` `WorkerOptions()`.
- `jobs/<name>jobs/<name>contract/`: workflow and signal names, payloads (which carry the
  organization and business unit), workflow ID helpers and a requester. Other modules and the API
  import only the contract, never the jobs package.
- Services start work through `kernelports.WorkflowStarter` / `WorkflowSignalStarter`, after
  commit (`kernelports.AfterCommit`).

## Starting and outages

- **A process waits for what is under it, within a budget.** The API and worker do not exit
  because Redis, Meilisearch or Temporal is not answering yet: each check goes through
  `startwait.Wait` (`internal/infrastructure/startwait`), run from an `fx.Invoke` so fx's
  15-second start timeout does not cut it short. It retries with backoff until the dependency
  answers or `app.startupWait` (default 5m, measured from process start, so the waits share it)
  is spent, then fails with the last error. It writes one line when the wait begins, one a minute
  while it lasts and one when it ends. Readiness still checks only the database: a cache restart
  must not take a working API out of rotation.
- The worker waits for Temporal (`temporaljobs.AwaitEngine`) because `worker.Start` refuses to
  start without it and cannot be retried. Schedule reconciliation does not gate start: the
  scheduler validates the declarations at start (a bad one fails loudly) and reconciles in the
  background with backoff until Temporal answers; nothing depends on schedules existing when the
  worker reports ready.
- **One line per outage.** The SDK writes "Failed to poll for task." once a minute per poller,
  about seventy lines a round. `pollwatch.Watch` turns them into one warning when the outage
  starts, one a minute while it lasts (`rate.Sometimes`) and one info line when it ends, with how
  long the engine was away and how many failed polls were reported. The end is observed by a
  `DescribeTaskQueue` probe every five seconds during the outage only: it goes to the matching
  service, as polls do, so a frontend that answers while matching does not is still an outage.

## Workflows

- Workflow code is deterministic: no I/O, clock, randomness, map iteration order or goroutines
  outside the Temporal SDK. Every database or network call is an activity.
- **No version gates.** We have not launched; change workflow code directly and let running
  executions be reset or terminated. Don't add `workflow.GetVersion`.
- **One workflow per tenant, not per event or per entity.** Repeated triggers collapse into a
  debounced drain keyed by the tenant: a workflow per entity is still one per row, and a tenant
  moving thousands of shipments keeps thousands cycling, each a start, a timer and several
  activities written to the Temporal database. The request is a row in `work_queue_items`
  (`workqueueports.Queue`, kind per job), written in the requester's transaction and moved
  strictly forward on every request (`GREATEST(due + 1, now)`), so nothing is lost between
  commit and signal. `temporaljobs.DrainRequester` enqueues and, after commit, sends one
  payload-free `SignalWithStart` to `temporaljobs.DrainWorkflowID(kind, tenant)`;
  `temporaljobs.RunTenantDrain` is the workflow loop (debounce, pages per round,
  continue-as-new) and `temporaljobs.DrainDue` the activity page: it claims due rows, works
  them, and settles each only while it still holds the claimed due time, pushing a failed one
  back; a handler that returns `temporaljobs.ErrDeferred` (no room under a cap) leaves the
  page blocked, so the drain looks again each debounce instead of ending. A row's `detail`
  carries what the work needs beyond its subject (the event kind of an agent's queued run). The job's sweep calls `DrainRequester.WakeDue` as the backstop for a lost wake
  (`etajobs.ShipmentEtaDrainWorkflow`). A queue that is a property of the row itself keeps it
  on the row instead (`billing_queue_items.recheck_due_at`, `BillingQueueTenantWorkflow`).
  The fairness interceptor keys each start by its tenant; never set a `FairnessKey` by hand.
- **A projection fed by many writers queues from the database.** When the inputs of a stored
  projection change through many services (an invoice's open amount moves on posting, payment,
  credit memo, adjustment and void), an `AFTER` row trigger `WHEN` the read columns changed
  inserts the drain's `work_queue_items` row in the writer's transaction, guarded by one index
  probe so a tenant that does not use the feature pays nothing more. No writer can forget it.
  The services a person waits on wake the drain after commit; everything else is woken by a
  one-minute `WakeDue` schedule (`bankreceiptcandidatejobs`, migration
  `20261231024220_bank_receipt_candidates`). The backfill is the same queue: the migration
  inserts a row per subject, the nightly sweep inserts them again to repair drift.
- **Work for a tenant reaches only an entitled, active tenant.** A fan-out lists through a helper
  that takes an `entitlement.Need` or passes its page through `temporaljobs.KeepEntitled`; a
  cross-tenant statement composes `entitlementports.Filter.ForWork`; work that belongs to no tenant
  says why. See [shared-tenancy.md](../shared-tenancy.md#background-work).
- **Heavy work takes a slot.** PDF rendering, extraction, exports, imports and previews hold a
  `kernelports.HeavyJobSlots` slot for their run; a new kind of heavy work adds a kind there.
- **A timer over a time column is a sweep, not a workflow per row.** A snoozed message, an EDI
  flow's next wake and a web watch's next check are columns with a partial index; a one-minute
  schedule claims what is due (`WakeDueTriageWorkflow`, `DispatchDueFlowTimersWorkflow`,
  `CheckDueWebWatchesWorkflow`). Keep a workflow per entity only when it orchestrates more than
  a timer, as an agent wait does (signals, settle windows, retries while a conversation is busy).
- **A workflow ends when its work does.** Waiting on a person is a record's status, not a
  parked workflow: close the record from the action that settles it and from the sweep that
  expires it, with one conditional statement both can race on
  (`agentrunclose.Closer`). A uniqueness a live workflow id used to hold moves to a partial
  unique index.
- **Deterministic workflow IDs** with a reuse or conflict policy
  (`WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE`, `CONFLICT_POLICY_USE_EXISTING`) make a duplicate
  start a no-op.
- Long-running loops continue-as-new before history grows large; carry only the state needed.
- **A long wait on someone else is its own workflow.** A step that waits hours on a provider
  (a Message Batch) inside a serial parent hands the wait to a child started with
  `PARENT_CLOSE_POLICY_ABANDON` and returns, carrying only the handle; the child polls on
  `workflow.Sleep` with backoff under a deadline and finishes the work
  (`agentqualityjobs.AgentSuiteJudgeWorkflow`).

## Activities

- Explicit `workflow.ActivityOptions` with a `RetryPolicy` and a `StartToCloseTimeout`; a long
  activity sets `HeartbeatTimeout` and heartbeats (`temporaltype.HeartbeatEvery`).
- **Every activity is idempotent**: it will be retried after it has partly succeeded. Use
  natural keys and unique indexes, `INSERT … ON CONFLICT`, conditional updates on `version` or
  status, and idempotency keys on outbound calls. A row recorded from something read again on
  retry takes an id derived from what it records (`pulid.Derived`) and is written with
  `CreateOrSkip`. An outbound call with no idempotency key that must not run twice (a batch
  submission) runs with `MaximumAttempts: 1` and a fallback path.
- An error that will never succeed on retry fails once. The worker's refusal interceptor
  (`temporaljobs/interceptors/refusal.go`, `temporaltype.ToActivityFailure`) makes a validation
  failure, a not-found, a refused permission, a plan refusal and a statement keel refused for how
  it was written (`dberror.IsStatementMisuse`: no limit, no `AllTenants`, no scope)
  non-retryable for every activity, so raise the right `errortypes` error at the owner (a
  missing stored object is a not-found, not a database error) rather than adding
  `NonRetryableErrorTypes`. A business error stays
  retryable because the domain also raises one for an outage worded for a person; return
  `ToDomainRefusal` when yours is final, or a Temporal `ApplicationError` to choose either way.
- Something wrong with the record itself (its file is gone or unreadable, its format refused) is
  recorded on the record as a failure and the activity succeeds; it is not an error to retry
  (`documentextractionjobs.readFailure`).
- Every retry policy sets `MaximumAttempts`. Waiting for someone else's state (a conversation
  still busy) is not a retry: the activity answers at once with a non-retryable error in the
  benign category, and the workflow waits on a timer with backoff under a deadline
  (`agentwaitjobs.finish`).
- The worker's logging interceptor reports each failed attempt once, with the activity, the
  workflow and the attempt: a Warn while a retry follows, an Error when the failure is final, a
  Debug for a benign answer. The SDK's own per-attempt line is kept at Debug (`sdkLogger`).
- Activities take IDs and load current state; never pass a whole entity through history.
- Batch: an activity handles a page of records, not one.

## Schedules and sweeps

- Schedules come from a `ScheduleProvider.GetSchedules()` with `schedule.Every` and
  `SCHEDULE_OVERLAP_POLICY_SKIP`, registered with `schedule.AsProvider`.
- Every schedule names its `Feature` (see below); `Validate` refuses one without a feature the
  catalogue knows.
- A sweep across tenants lists candidates under `dbscope.WithSystem(ctx, "<reason>")` in the
  repository (reviewed by `rlslint`), then does each tenant's work under `dbscope.WithTenant`.
  `temporaljobs.RunTenantFanOut` is the helper.
- Prefer an event that starts the debounced workflow over a sweep that polls; keep the sweep only
  as the catch-up for what events miss, and bound it with an index on its predicate.
- A sweep that re-checks rows against a value only the process knows (a catalog hash) stamps
  each row with the value it checked and selects `col < v` then `col > v`: two range scans on an
  index on the column, empty once the sweep has caught up. Its write is conditional on the
  row's `version` and on the stamp still being stale, so a person's save that lands meanwhile
  wins (`reportcatalogjobs/catalogcheck.go`).

## A schedule names its feature

- `schedule.Schedule.Feature` is the `platformcatalog` feature the schedule's work belongs to,
  written as the catalogue constant. Where the work is on records of a permission resource, it is
  that resource's feature in the permission registry; base work names its base feature
  (`FeatureCoreTMS`, `FeatureBilling`, `FeatureAdministration`, …), never nothing. Work that
  serves base and add-on features alike (the rollup refresh) is base.
  `TestEveryDeclaredScheduleNamesAKnownFeature` reads every declaration in the tree.
- The reconciler asks `FeatureGate.EnabledAnywhere` once per feature: a schedule belongs to no
  tenant, so it runs while any scope on the install has its feature (the instance on a dedicated
  install, any active organization on a shared server; [shared-tenancy.md](../shared-tenancy.md)).
  Work it fans out to tenants still asks each tenant's plan. A schedule whose feature
  is off everywhere is still created, but paused, with a note saying the plan does not include the feature;
  when the feature is on it runs unless its declaration says `Paused: true`. The paused state and
  its reason are part of the hash in the note, so a plan change is an update and an unchanged plan
  is not. With every feature on (self-hosted) the hash is the declaration's own.
- A pause or unpause made by hand in Temporal replaces the note; from then on the paused state is
  the person's and the reconciler keeps it, until the plan turns the feature off, which always
  pauses. A gate error fails the reconcile pass before Temporal is touched.
- The scheduler checks the gate every minute after the start-up reconcile and re-reconciles only
  when the set of off features among those its schedules name changes, so buying or dropping an
  add-on takes effect within about a minute and a half with no restart. A failed pass is retried
  on the next check. Every worker replica runs the check; the reconcile writes the same desired
  state from the same inputs, so replicas racing converge.
- Schedules a service creates at run time (an agent definition's, a conversation's) do not go
  through the registry; the service gates them with `RequireFeature` itself.

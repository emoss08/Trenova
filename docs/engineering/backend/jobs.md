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

## Workflows

- Workflow code is deterministic: no I/O, clock, randomness, map iteration order or goroutines
  outside the Temporal SDK. Every database or network call is an activity.
- **No version gates.** We have not launched; change workflow code directly and let running
  executions be reset or terminated. Don't add `workflow.GetVersion`.
- **One workflow per thing, not per event.** Repeated triggers for the same entity
  `SignalWithStartWorkflow` a debounced workflow keyed by the entity
  (`etacontract.RefreshWorkflowID(id)`), which sleeps for the debounce, drains its signals, does
  the work once and continues-as-new after N iterations (`etajobs/workflows.go`). Pass a tenant
  `FairnessKey` so one tenant's burst does not starve the rest.
- **Key the debounce by tenant when the entity count is large.** A workflow per entity is
  still one per row: a tenant moving thousands of items keeps thousands cycling, each a
  start, a timer and several activities, all written to the Temporal database. Keep the work
  queue in the row instead: a `<work>_due_at` column set in the requesting transaction, moved
  strictly forward on every request (`GREATEST(coalesce(due, 0) + 1, now)`), so nothing is lost
  between commit and signal. One payload-free `SignalWithStart` per tenant wakes a drain that
  claims a page of due rows, works each, and clears the marker only while it still holds the
  claimed value (a request that landed meanwhile keeps the row due); a failed row is pushed
  back (`due_at = now + backoff`). The sweep marks stale rows due and signals the tenants with
  due rows, never row by row (`billingqueuejobs.BillingQueueTenantWorkflow`).
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

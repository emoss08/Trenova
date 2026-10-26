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
- **Deterministic workflow IDs** with a reuse or conflict policy
  (`WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE`, `CONFLICT_POLICY_USE_EXISTING`) make a duplicate
  start a no-op.
- Long-running loops continue-as-new before history grows large; carry only the state needed.

## Activities

- Explicit `workflow.ActivityOptions` with a `RetryPolicy` and a `StartToCloseTimeout`; a long
  activity sets `HeartbeatTimeout` and heartbeats (`temporaltype.HeartbeatEvery`).
- **Every activity is idempotent**: it will be retried after it has partly succeeded. Use
  natural keys and unique indexes, `INSERT … ON CONFLICT`, conditional updates on `version` or
  status, and idempotency keys on outbound calls.
- An error that will never succeed on retry (validation, not found, refused) is
  `temporaltype.NewNonRetryableError` or `ToDomainRefusal`.
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
  wins (`reportjobs/catalogcheck.go`).

# Application services

Services in `app/<name>service/` own a use case: they validate, decide, persist through ports,
and announce what happened. The reference is
`internal/modules/reference/app/commodityservice/`; paths are under `services/tms/`.

## Shape

- fx `Params` struct (`fx.In`) holding ports only: repositories, validators, `AuditService`,
  `kernelports.WorkflowStarter`, other services' ports. `New(p) *Service`, wired in the module's
  `module.go`.
- Depend on the narrowest port that does the job. A service that needs one method of a wide
  interface gets a narrower port declared next to its consumer.
- **Plan, then persist, then announce.** `PlanCreate`/`PlanUpdate` (`plan.go`) transform,
  validate and load the original as a `kernelports.RecordChange{Before, After}`; the service then
  writes through the repository and records the audit entry (`LogAction` with `WithDiff`). A pure
  plan step is what lets agents preview the same write ([proposal-previews.md](../proposal-previews.md)).
- Bulk changes are one plan over many records (`statuschange.Plan`,
  `auditservice.BuildBulkLogEntries`), one statement, one batch of audit entries.
- A use case that spans several repositories runs in one transaction on the context; the
  repositories join it through `Keel(ctx)`. A service that only needs the transaction depends on
  `kernelports.Transactor` (`Transact(ctx, opts, fn)`), not `kernelports.DBConnection`
  (`app/tractorservice`).

## Validation

- Each entity has a validator built with `validationframework.NewTenantedValidatorBuilder[T]()`
  (`commodityservice/validator.go`): uniqueness and reference checks, then the entity's own
  `Validate`. Hold it as `validationframework.TenantedEntityValidator[T]`, never the concrete
  generic type ([go.md](go.md#generics-and-build-time)).
- Validation returns `*errortypes.MultiError` with field paths (`user.address.street`,
  `items[0].name`) so the client puts each message on its field.
- Validate everything that crosses a trust boundary: request input, webhook payloads, files, model
  output an agent produced. Model output is input.

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

## After the write

- **Audit**: `auditService.LogAction` after a successful write. It also publishes the
  record-change invalidation, so a CRUD service usually needs no separate realtime call.
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

# Shared tenancy

A managed install is either **dedicated** (`managed.tenancy: dedicated`, the default: one customer,
whose plan is the instance's) or **shared** (`managed.tenancy: shared`: a server Trenova runs for
many companies, one organization each, whose plans differ). Shared requires `managed.enabled`.
`EditionInfo.SharedTenancy()` answers from the same setting. Self-hosted installs are neither: every
feature is on (`entitlementservice.AllowAll`) and nothing here is read.

## The scope

A **scope** names whose plan a question is about (`kernel/domain/managedscope`):

| Tenancy | Scope | Key |
|---|---|---|
| dedicated | the instance, for every context | `instance` |
| shared | the organization of the tenant in `dbscope` | the organization's id |

`managedscope.Resolver` (provided by the kernel module from `managed.tenancy`) turns a context into a
scope (`Scope(ctx)`) or a tenant it was handed into one (`ForTenant`). On a shared server a context
with no tenant, a system scope included, is `ErrNoTenantScope`. `Accepts` refuses a scope from the
other tenancy, so the instance scope is never written on a shared server and an organization scope
never on a dedicated one.

Every piece of managed state is keyed by scope: the feature list, its history, the plan URL, the
plan's end and the add-on endings (`managed_feature_states`, `managed_feature_changes`), the scope's
lifecycle (`active` or `suspended`), and the usage state (`managed_usage_states`). The tables are
global tables read and written only by the managed services under system scope: they are the
control plane's entitlement metadata, never customer data. Caches are otter caches keyed by scope,
30 seconds, bounded by entry count, and a write drops only its own scope. Details are in
[managed-mode.md](managed-mode.md).

## The gate's two questions

```go
Enabled(ctx, feature)         // for the scope the context carries
EnabledAnywhere(ctx, feature) // does any scope on this install have it
```

- **Ask `Enabled` with the tenant the work is for in `dbscope`.** A request, a tenant-scoped
  activity and a fan-out loop that binds each tenant already carry it. A public route or webhook
  resolves its tenant first (token, slug, connection, directory, mailbox, profile), binds it with
  `dbscope.WithTenant`, then asks, and answers a tenant without the feature exactly as it answers an
  unknown link, so nothing is learned by guessing (`tenderresponse.resolveToken`,
  `inboundintake.ReceiveWebhook`, `accountingconnectionkit.Kit.Entitled`). Never bind a tenant from
  a request body.
- **A no-tenant question on a shared server is an error**, never a quiet yes. That is the point:
  a call site that forgot its tenant fails loudly in shared mode instead of answering for the whole
  server.
- **`EnabledAnywhere` is only for work that belongs to no tenant**: schedule pausing
  ([jobs.md](backend/jobs.md#a-schedule-names-its-feature)), start-up. On a dedicated install it is
  the instance's answer; on a shared server, whether any active organization has the feature. Work
  for one tenant never asks it.
- **A scope with no plan, or a suspended one, has no features at all**, base included. On a
  shared server an organization the control plane has sent no plan for gets nothing until its
  first list arrives; on a dedicated install the instance always has the base plan and is always
  active.
- The permission engine asks the plan for the organization it is checking (`permission.planScope`),
  so `onplan` refusals and the manifest's `unavailableFeatures` are per organization and the client
  needs no change.

## Quotas and notices

`managedusageports.Gate` decides per scope. `ManagedQuotaGuard` and `ManagedUsageProvider` turn the
tenant they are given into a scope. The meters measure one scope: the instance across organizations,
an organization under its own tenant scope, so the dedicated readings stay instance totals. An
allowance's `ceiling` stops an action whatever the overage setting. Plan and storage notices go to the
administrators of the scope they concern.

On a shared server an email address is still unique across the server, so the refusal of a taken
address or username is generic (`tenant.EmailAddressTakenError`) and never says it belongs to
another company.

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

## Background work

A schedule names the feature its work belongs to
([jobs.md](backend/jobs.md#a-schedule-names-its-feature)). On a shared server that is not enough:
the schedule runs while any organization has the feature, and its work must reach only the
organizations that do. **Work for a tenant runs only for a tenant whose plan has the feature and
whose scope is active.** A suspended organization, and one the control plane has sent no plan for,
get no background work at all. On a dedicated install every check answers for the instance, and on
a self-hosted one everything is on, so both behave as before.

`entitlement.Need` (`kernel/domain/entitlement`) is what a piece of work needs: `entitlement.For(feature)`
or `entitlement.AnyActive()` (an active plan, no feature: plan notices, storage warnings, permission
changes). Its zero value is invalid, so a call site cannot forget it. `entitlementports.Filter` is the
one place the answer is computed:

| Method | For |
|---|---|
| `Organizations(ctx, col, need)` | a predicate on an organization id column, composed into a cross-tenant statement |
| `ForWork(ctx, col)` | the same, with the need the schedule's header carried into the activity |
| `Keep(ctx, need, ids)` | the entitled subset of a page of ids, one read |

All three read `managed_feature_states` through one partial index
(`idx_managed_feature_states_entitled`: active scopes with a plan), so the filter is one index probe
per statement, not a read per tenant. `managedfeaturesrepository.ActivePlan` writes the lifecycle
and `received_at` as constants (`keel.ConstText`, `keel.ConstInt64`): a bound value would keep a
generic plan off the partial indexes.

What each answer means:

- `entitlement.For(feature)` of a base feature asks for an active plan alone; of an add-on, that
  the plan lists the add-on. `AnyActive` is work for the platform as a whole: every active plan,
  whatever it holds.
- On a shared server `Organizations` is the column `IN` the entitled scopes; on a dedicated or
  self-hosted install it is `TRUE` or `FALSE` with no join, the instance's answer for every
  organization. `Keep` returns the entitled ids in the order given.
- `ForWork` is `Organizations` with the need `entitlement.WithWork` put in the context. Only the
  `workfeature` propagator puts it there, from the schedule's header, so `ForWork` belongs in a
  statement a schedule's activity runs. On a shared server a context with no work need is
  `entitlement.ErrNoWork`, never every organization; a sweep called from anywhere else passes its
  need to `Organizations`.
- `KeepEntitledRequest` names no tenant, so the activity behind `KeepEntitled` runs in no
  tenant's scope. Every task queue's worker registers it (`WorkerManager.SetQueueActivities`), so a
  workflow calls it on its own queue.
- A drain (`temporaljobs.DrainDue`) reads the plan again before every page: work queued before the
  plan dropped the feature, or before the tenant was suspended, stays queued, untouched, and the
  kind's sweep wakes the drain when the plan covers it again. The accounting sync drain claims only
  the record kinds the plan covers now (`ClaimAccountingSyncRecordsRequest.ObjectTypes`; empty
  claims nothing): the base sync needs accounting integrations, the ledger side needs Accounting.
- `platformcatalogports.Observing` asks the plan of the tenant the change belongs to, not the scope
  of the code that made it, so a sweep under system scope still tells that tenant's observers; a
  change with no tenant reaches no observer on a shared server.
- A plan that gains an add-on, or a suspended scope that resumes, is `Replacement.Gained`. The
  search index sweep then runs for that one organization (`retrievalservice.SweepWorkflowIDFor`,
  so a change that lands twice starts it once), reading the plan again before it works.
- The plan-end reminder pages 500 scopes at a time, at most 40 pages an hour, in plan end then
  scope order; a scope whose reminder cannot be recorded is stepped over and is first again on the
  next run.

There are three shapes of background work, and each filters in one place:

- **Fans out.** A workflow lists tenants and starts work for each. Every tenant-enumerating helper
  takes a need: `tenantports.EachTenant`, `TenantSyncRepository.ListTenantsPage`,
  `temporaljobs.OrganizationPage` and `temporaljobs.RunTenantFanOut` filter each page in the
  same statement. A workflow that lists its tenants some other way (its own query, the tenants
  with rows due) passes the page through `temporaljobs.KeepEntitled` before it starts anything:
  one activity, one read per page.
- **Sweeps in SQL.** One system-scoped statement works rows of every tenant. It composes
  `Filter.ForWork` (or `Organizations` with an explicit need) on its organization column, so an
  unentitled tenant's rows are never read. A sweep with no need in its context on a shared server
  fails (`entitlement.ErrNoWork`): it is a sweep nobody classified.
- **No tenant.** Work that belongs to the instance (the audit trail's tamper evidence and
  retention, partition upkeep, rollup refresh, plan-end reminders) runs for every tenant on
  purpose; each says why below.

Retention follows the same rule: a tenant that lost a feature keeps the feature's data until the
feature returns or the organization is removed, rather than having it purged by a sweep it no
longer pays for.

### Every schedule

The harness in `internal/testutil/entitlementtest` runs every schedule's workflow on a shared
server with an entitled, a lacking and a suspended organization, and fails when work reaches the
latter two (`TestEverySchedulingPackageRunsItsSchedulesOnASharedServer` lists the packages; each
package's `entitlement_test.go` declares its schedules' shapes). The lacking organization has a
plan without the feature, or no plan at all when the feature is base. Every activity of the
package is replaced by a fake of the same signature: the first call made in no tenant's scope
answers every organization in each list of items that carry an organization and a business unit,
as a listing query would before any filter; every other call answers zero values.
`temporaljobs.KeepEntitled` runs for real. A package whose schedules are run by another package's
test is listed, with the reason, in `scheduleHarnessExceptions`; keep that list short. Repository
tests that are not about entitlement build with `entitlementtest.Instance()`, a dedicated
install's filter with every feature. The harness cannot run a sweep's
SQL, so a sweep's filter is proven by its repository's integration test, and the harness checks
only that the sweep keeps its shape.

| Schedule | Feature | Shape | Where it filters |
|---|---|---|---|
| `fiscal-auto-close-periods` | `Accounting` | fans out | KeepEntitled over the tenants with scheduled period close |
| `fiscal-auto-create-next-year` | `Accounting` | fans out | KeepEntitled over the fiscal-calendar tenants |
| `fiscal-year-close-readiness` | `Accounting` | fans out | KeepEntitled over the fiscal-calendar tenants |
| `ai-correction-retention` | `AIDesk` | fans out | RunTenantFanOut over OrganizationPage |
| `extraction-accuracy-drift` | `DocumentIntelligence` | fans out | RunTenantFanOut over OrganizationPage |
| `ai-feedback-maintenance` | `AgentAutomation` | fans out | RunTenantFanOut over OrganizationPage |
| `ai-tune-ups` | `AgentAutomation` | fans out | RunTenantFanOut over OrganizationPage |
| `insight-refresh` | `AIDesk` | fans out | RunTenantFanOut over OrganizationPage |
| `agent-profiles` | `AgentAutomation` | fans out | RunTenantFanOut over OrganizationPage, child per tenant |
| `retrieval-index-sweep` | `AIDesk` | fans out | RunTenantFanOut over OrganizationPage |
| `ai-analytics-refresh` | `AgentAutomation` | fans out | RunTenantFanOut over the tenants with AI activity |
| `daily-briefing` | `AIDesk` | fans out | EachTenant lists due tenants; RunTenantFanOut, child per tenant |
| `briefing-retention` | `AIDesk` | sweeps in SQL | briefings, shipment briefs and accounting briefs DeleteBefore compose ForWork in the limited subquery |
| `agent-schedule-reconcile` | `AgentAutomation` | sweeps in SQL | ListScheduledAcrossTenants composes ForWork; an unentitled tenant's agent schedules are removed and come back with the plan |
| `expire-stale-proposals` | `AIDesk` | sweeps in SQL | proposal and plan ExpirePending, ListPendingForReminder, run FailAbandoned/CloseSettled, baseline PurgeOrphans compose ForWork |
| `assistant-retention` | `AIDesk` | sweeps in SQL | conversation DeleteStaleThreads and palette question DeleteBefore compose ForWork |
| `agent-quality-schedules` | `AgentAutomation` | sweeps in SQL | ListScheduleTargets composes ForWork on organizations |
| `agent-wait-reconcile` | `AgentAutomation` | sweeps in SQL | ListOverdueAcrossTenants composes ForWork |
| `ai-audit-projector` | `AIAuditTrail` | sweeps in SQL | every source keyset read composes ForWork |
| `ai-audit-verify` | `AIAuditTrail` | fans out | ListChainHeads composes ForWork; KeepEntitled before per-tenant verify |
| `ai-audit-prune` | `AIAuditTrail` | sweeps in SQL | ListChainHeads composes ForWork |
| `ai-audit-export-cleanup` | `AIAuditTrail` | sweeps in SQL | export ListExpired/ListStale compose ForWork |
| `conversation-schedule-reconcile` | `AIDesk` | sweeps in SQL | ListAcrossTenants composes ForWork |
| `eval-case-capture` | `AgentAutomation` | sweeps in SQL | ListCaptureCandidates/AdoptCandidates/RetireStaleAutomatic compose ForWork |
| `eval-case-purge` | `AgentAutomation` | sweeps in SQL | PurgeExpired(AllTenants)/PurgeOrphaned compose ForWork |
| `watchtower sweeps (reconcile, shipment-exception, decision, factoring)` | `AIWatchtower` | sweeps in SQL | sweepAll pages through EachTenant(AIWatchtower) |
| `no-check-call-sweep` | `AIWatchtower` | sweeps in SQL | dispatch control listTenantsWhere composes ForWork |
| `exception-attention` | `AIWatchtower` | sweeps in SQL | ListTenantsNeedingAttention composes ForWork |
| `watchtower-retention` | `AIWatchtower` | sweeps in SQL | DeleteResolvedBefore composes ForWork in the limited subquery |
| `web-watch-sweep` | `AIDesk` | sweeps in SQL | ClaimDueChecks composes ForWork in the claim |
| `horizon-plan-sweep` | `DispatchIntelligence` | sweeps in SQL | dispatch control listTenantsWhere composes ForWork |
| `billing-queue-recheck` | `Billing` | sweeps in SQL | MarkStaleRecheckDue/ListRecheckDueTenants compose ForWork |
| `billing-queue-nova` | `AgentAutomation` | sweeps in SQL | ListNovaBacklogTenants composes ForWork |
| `billing-transfer-automatic` | `Billing` | sweeps in SQL | billing control listAll composes ForWork |
| `billing-transfer-zombie` | `Billing` | sweeps in SQL | transfer run ListStale composes ForWork |
| `customer-credit-sweep` | `Accounting` | sweeps in SQL | ListCreditReviewCustomers composes ForWork |
| `customer-statement-sweep` | `Accounting` | sweeps in SQL | billing control listAll composes ForWork |
| `payment-reminder-sweep` | `Accounting` | sweeps in SQL | billing control listAll composes ForWork |
| `factoring-readiness-sweep` | `Factoring` | fans out | KeepEntitled before per-tenant sweep |
| `consolidated-invoice-run` | `Billing` | sweeps in SQL | ListDueBillingSchedules(all tenants) composes ForWork |
| `accounting-connection-check` | `AccountingIntegration` | sweeps in SQL | ListDueForHealthCheck composes ForWork |
| `accounting-webhook-subscriptions` | `Accounting` | sweeps in SQL | subscription ListConnections composes ForWork |
| `accounting-reference-refresh` | `AccountingIntegration` | fans out | connection ListActive composes ForWork; KeepEntitled before the child per connection |
| `accounting-changes-read` | `Accounting` | sweeps in SQL | connection ListActive composes ForWork; drains are woken only for entitled connections |
| `accounting-drift-reconcile` | `Accounting` | sweeps in SQL | connection ListActive composes ForWork |
| `accounting-drift-weekly` | `Accounting` | sweeps in SQL | connection ListActive composes ForWork |
| `accounting-sync-dispatch` | `AccountingIntegration` | sweeps in SQL | sync ListDueConnections composes ForWork |
| `accounting-sync-safety-net` | `AccountingIntegration` | sweeps in SQL | connection ListActive composes ForWork |
| `accounting-sync-retention` | `AccountingIntegration` | sweeps in SQL | PurgeHistory composes ForWork in both limited subqueries |
| `capture-maintenance` | `DocumentManagement` | sweeps in SQL | capture batch ListStale/ListRetentionDue/ListRetentionReminders and request ListExpired compose ForWork; abandoned pairings belong to no organization yet and expire for everyone |
| `document-extraction-reconciliation` | `DocumentIntelligence` | fans out | KeepEntitled before the per-tenant reconcile |
| `document-upload-reconciliation` | `DocumentManagement` | fans out | KeepEntitled before the per-tenant reconcile |
| `inbound-message-retention` | `AITriage` | sweeps in SQL | EachTenant(AITriage) instead of every organization |
| `triage-wake` | `AITriage` | sweeps in SQL | ListOverdueParked composes ForWork in both arms of the union |
| `edi-inbound-poll` | `EDIIntegration` | fans out | KeepEntitled over the pollable profiles before polling |
| `edi-raw-retention-purge` | `EDIIntegration` | fans out | KeepEntitled over the retention tenants |
| `edi-flow-timers` | `EDIIntegration` | sweeps in SQL | ListDueInstances composes ForWork |
| `orphaned-access-sweep` | `Administration` | sweeps in SQL | ListOrphanedAssignments composes ForWork (skips suspended tenants and tenants with no plan) |
| `support-access-expiries` | `Administration` | sweeps in SQL | ListUnrecordedExpiries composes ForWork |
| `support-log-timeouts` | `Administration` | sweeps in SQL | support log Overdue composes ForWork |
| `late-charge-assessment` | `Accounting` | fans out | KeepEntitled before the per-tenant assessment |
| `scim-grant-sweep` | `Identity` | sweeps in SQL | MarkExpiredSaversPending and ListPendingDirectories compose ForWork; reconciles are requested only for entitled tenants |
| `carrier-tracking-link-retention` | `Dispatch` | sweeps in SQL | tracking link PurgeDead composes ForWork in the limited subquery |
| `customer-status-sweep` | `CustomerTracking` | fans out | work queue ListDueTenants composes ForWork for the drain wake; KeepEntitled before the per-tenant sweep |
| `customer-tracking-link-purge` | `CustomerTracking` | sweeps in SQL | link PurgeDead composes ForWork |
| `shipment-eta-sweep` | `CustomerTracking` | fans out | work queue ListDueTenants composes ForWork; KeepEntitled before the per-tenant refresh |
| `demand-forecast-fit` | `DispatchIntelligence` | fans out | RunTenantFanOut |
| `dispatch-style-learning` | `DispatchIntelligence` | fans out | RunTenantFanOut |
| `detention-notice-sweep` | `Detention` | fans out | shipment control ListDetentionEngineTenants composes ForWork; KeepEntitled before the per-tenant sweep |
| `recurring-shipment-dispatch` | `CoreTMS` | sweeps in SQL | recurring shipment ListDue composes ForWork (skips suspended tenants and tenants with no plan) |
| `shipment-auto-delay` | `CoreTMS` | fans out | KeepEntitled before the per-tenant run |
| `shipment-auto-cancel` | `CoreTMS` | fans out | KeepEntitled before the per-tenant run |
| `tender-stalled-sweep` | `CarrierBrokerage` | sweeps in SQL | ListForSweep(across tenants) composes ForWork |
| `rate-confirmation-issue-sweep` | `CarrierBrokerage` | sweeps in SQL | ListAcceptedMissingRateConfirmation composes ForWork |
| `tender-token-retention` | `CarrierBrokerage` | sweeps in SQL | PurgeDeadOfferTokens and PurgeDeadSignTokens compose ForWork in the limited subquery |
| `weather-alert-poll` | `CustomerTracking` | sweeps in SQL | weather ListTenants and ExpireStaleAlerts compose ForWork; the NWS feed itself belongs to no tenant |
| `carrier-intelligence-sweep` | `CarrierBrokerage` | fans out | RunTenantFanOut |
| `carrier-intelligence-reconcile` | `CarrierBrokerage` | fans out | RunTenantFanOut |
| `carrier-intelligence-digest` | `CarrierBrokerage` | fans out | RunTenantFanOut |
| `carrier-intelligence-maintenance` | `CarrierBrokerage` | fans out | raw payload PurgeExpired and usage DeleteOlderThan compose ForWork; the daily usage rollup summarizes usage that already happened and is never filtered; RunTenantFanOut for the per-tenant prune |
| `stored-mileage-buffer-flush` | `Dispatch` | sweeps in SQL | finishes the mileage writes tenants' own requests buffered; a suspended tenant buffers none, and dropping the buffer would lose them |
| `formula-template-expire-stale-submissions` | `Rating` | sweeps in SQL | library, template and rate matrix ListStaleSubmissions compose ForWork |
| `eia-fuel-price-refresh` | `Billing` | fans out | KeepEntitled before the per-tenant refresh; the EIA fetch itself belongs to no tenant |
| `exchange-rate-refresh` | `ExchangeRateIntegration` | fans out | KeepEntitled before the per-tenant refresh |
| `carrier-settlement-batch-generation` | `CarrierBrokerage` | sweeps in SQL | carrier settlement control ListAutoGenerate composes ForWork |
| `settlement-batch-generation` | `Settlement` | sweeps in SQL | settlement control ListAutoGenerate composes ForWork |
| `escrow-interest-accrual` | `Settlement` | sweeps in SQL | settlement control ListAll composes ForWork; each tenant's accounts are read in its own scope |
| `credential-expiry-sweep` | `WorkforceCompliance` | sweeps in SQL | credential ListExpiring, training ListDue/ListExpiring, ListWorkersWithClearinghouseDue and ListWorkersWithLapsedPoints compose ForWork |
| `pto-accrual` | `WorkforceTimeOff` | fans out | KeepEntitled before each tenant's accrual pages |
| `samsara-telematics-poll` | `SamsaraIntegration` | fans out | KeepEntitled before the per-tenant poll |
| `telematics-retention-sweep` | `SamsaraIntegration` | sweeps in SQL | CleanupExpired composes ForWork on all three tables |
| `samsara-telematics-sweep` | `SamsaraIntegration` | fans out | KeepEntitled before the per-tenant sweep |
| `audit-partition-check` | `Administration` | no tenant | the shared monthly partitions belong to the instance |
| `audit-retention-purge` | `Administration` | no tenant | the audit trail's retention and legal holds are the instance's duty to every tenant, plan or not |
| `audit-seal` | `Administration` | no tenant | tamper evidence of every tenant's audit trail, suspended or not; a skipped window would never be sealed |
| `audit-verify` | `Administration` | no tenant | chain verification of every tenant's audit trail |
| `audit-export-cleanup` | `Administration` | no tenant | ends the download window of audit exports, a security bound that holds for every tenant |
| `fuel-card-feed-sync` | `FuelTax` | fans out | KeepEntitled before the per-tenant sync |
| `notification-daily-digest` | `RealtimeNotifications` | sweeps in SQL | ListDigestSubscribers composes ForWork |
| `managed-plan-end-reminder` | `Administration` | no tenant | works over the scoped plan table itself, active scopes only, paged with no cap |
| `report-schedule-dispatch` | `Analytics` | sweeps in SQL | report schedule ListDue composes ForWork |
| `report-artifact-cleanup` | `Analytics` | sweeps in SQL | report run ListExpired composes ForWork |
| `report-zombie-reconciliation` | `Analytics` | sweeps in SQL | report run ListStale composes ForWork |
| `report-catalog-check` | `Analytics` | sweeps in SQL | ListCatalogStale composes ForWork in both range scans |
| `rollup-refresh` | `Administration` | no tenant | keel's refresher works the dirty marks writes left, across tenants; a suspended tenant writes none |

### The backstop

A filter that missed must not become work done. The schedule reconciler creates and updates each
schedule under `workfeature.Starting`, so every run carries its feature; a workflow started any
other way through the client carries none, and an activity that starts another workflow does not
pass its own feature on, since that work may belong to another feature. An activity with no tenant,
or not run for a schedule, is not checked. The worker's activity interceptor chain runs
`interceptors.Entitlement` after the tenant scope: when the schedule's feature reached the
activity (the `workfeature` propagator carries it from the schedule through children, activities
and continue-as-new) and the activity has a tenant, it asks the gate for that tenant and refuses
with a non-retryable `TenantNotEntitled` application error. Each refusal is counted in
`trenova_tenancy_entitlement_refusals_total{activity_type,feature}` and logged at most once a
minute per activity type. Any refusal is a filter that missed: fix the filter, never silence the
backstop.

### Fairness

`interceptors.Fairness` sets the Temporal fairness key of every workflow, child workflow, signal
with start and activity from the tenant it is for (the arguments' `dbscope` tenant or organization
id, then the context's), and `system` when there is none. A payload that names its organization
without its business unit is still keyed by the organization. It is a client interceptor, which
the SDK applies to the workers built from that client. Nothing sets a fairness key by hand; a
schedule's own action carries `system`, since schedule updates are not intercepted.

## Limits

Each limit is bounded per organization with an instance cap behind it, and each switch is `auto`
(on for a shared server, off on a dedicated or self-hosted one), `on` or `off`
(`tenantLimits` in the config). Refusals are counted in
`trenova_tenancy_limit_refusals_total{limit,scope}` and waits in
`trenova_tenancy_slot_waits_total{kind,scope}`.

| Limit | Setting (default) | Enforced in | What the caller sees |
|---|---|---|---|
| Requests | `tenantLimits.requests` (auto; instance 30000/min, burst 3000) with `security.rateLimit.tenant` (6000/min, burst 600) | `middleware.RateLimiter.ByPrincipal` | sessions, capture devices and agents count in their organization's bucket as API keys do; 429 with `Retry-After` and `X-RateLimit-Scope: tenant` or `instance` |
| GraphQL cost | `security.graphql.costBudget` (person 6M points/min; organization 24M, burst 6M) | `gqlext.CostBudgetExtension`, in Redis | 429 and `OPERATION_COST_BUDGET_EXCEEDED` with `Retry-After` |
| Heavy jobs | `tenantLimits.heavyJobs` (auto; per kind below) | `heavyslots.Gate` through `kernelports.HeavyJobSlots` | a request waits up to `wait`, then 429 "the server is busy" with `Retry-After`; an activity waits, heartbeating, until near its deadline, then its retry policy runs it again |
| Built-in embedding | `tenantLimits.aiSlots` (auto; 2 per organization) | `completionengine.ClaimSlot` | the provider is busy for that organization (`ErrProviderBusy`): the router falls through to the next provider, or the call waits as any provider slot does |
| Model provider breaker | always per organization for the built-in provider | `completionengine.BreakerKey` | one organization's failures rest the built-in provider for that organization alone |
| Trenova sender | `tenantLimits.email` (auto; 300 an hour) inside the instance's 3000 | `emailservice.admitTrenovaRate` | `RateLimitError` naming the organization's hourly limit and when to try again |
| Database | `database.statementTimeout` (10s), `database.idleInTransactionSessionTimeout` (30s) | session parameters of the tenant and system pools | the statement fails; an idle transaction is ended and its connection freed |

Heavy job slots (`tenantLimits.heavyJobs`), each with a `hold` after which a dead holder's slot
lapses:

| Kind | Per organization | Instance | Where |
|---|---|---|---|
| `pdf` | 2 | 4 | `gotenberg.Client.Render` (every PDF the renderer draws) |
| `extraction` | 1 | 1 | document conversion in `documentextractionjobs` |
| `export` | 1 | 4 | report runs, audit exports, AI audit exports |
| `import` | 1 | 4 | rate sheet upload, fuel card statement staging |
| `preview` | 1 | 4 | invoice packet page previews; warming ahead takes only a free slot |
| `removal` | 1 | 1 | a removed shared-server tenant's batched deletes, cascade and store deletes (`tenantjobs`) |

The defaults fit one 24 GB server running the whole stack: the PDF renderer (0.5 GB) renders four
pages at once, extraction (3 GB) one job, exports and imports in the worker (2.5 GB) four each,
previews in the imaging worker (1.5 GB) four. The instance request bucket keeps a flood from every
organization at once below the point where the database pool queues, a few hundred requests a
second.

A heavy job's organization is the tenant in its context; work with none counts against the
instance bound only. A request waits up to the kind's `wait`; an activity waits until ten seconds
before its deadline (at most the kind's `hold`), heartbeating, so Temporal's retry policy is the
queue. `TryHold` is for work that may be skipped, such as warming previews; a viewer whose load
joined a warm that gave up loads again. The PDF renderer also keeps its in-process queue behind
the shared slot.

The GraphQL budget keys carry the organization as a Redis hash tag (`{org}:gqlcost:<principal>`,
`{org}:gqlcost`), so both buckets of one operation sit on one shard. A provider slot of the
built-in embedding service counts in the provider's bound and in the calling organization's share;
a provider an organization owns has its own bound only.

`slotstore.Store` is the one Redis slot mechanism: a sorted set per bound scored by lapse time,
claimed in every bound or none by one script. Provider slots and heavy-job slots both use it; a
new bounded resource claims from it rather than adding another. When Redis cannot be reached a
heavy job runs without a slot rather than failing every job.

## Tenant lifecycle (`internal/cloud`)

Everything below lives in `services/tms/internal/cloud`, under its own license
(`internal/cloud/LICENSE`, the Trenova Commercial License). Public code never imports it:
`scripts/archlint` fails on any importer outside `internal/cloud` and `internal/bootstrap`, and
`.go-arch-lint.yml` gives it the component `cloud`, which may use every module's domain, ports and
contracts, `infrastructure`, `api` (only `managedroutes`) and `pkg`. Core reaches it through ports:
the managed listener's route group (`managedroutes.Contributor`, fx group
`managed_route_contributors`) and the email service's `emailports.InstanceTags`, which
`cloud.SharedOptions` decorates on a shared server. Its keel models generate into
`internal/cloud/cloudkinds` and `internal/cloud/cloudtables` (`task keel-gen` runs both), so no
public package lists a commercial table. Its system scopes and repositories are checked by its own
`internal/cloud/lint_test.go` (the same `rlslint` and `tenantbindlint` rules core runs).

A **tenant** is one company on a shared server: one business unit and one organization, known to
the control plane by its instance id. `cloud_tenants` holds the server's bookkeeping about each:

| Column | What |
|---|---|
| `instance_id` | the control plane instance (primary key) |
| `tenant_organization_id`, `tenant_business_unit_id`, `admin_user_id` | what the tenant became; null until it is made |
| `state` | `applying`, `active`, `suspended`, `removing`, `removed` |
| `applied_revision` | the newest revision the tenant holds everything of; 0 before its first |
| `desired_revision`, `desired_digest`, `desired` | the entry last read (its SHA-256 and JSON); `desired` is the entry last accepted once the tenant is made |
| `admin_invite_sha256` | the first administrator's invite last installed |
| `last_archive_*` | the newest archive taken for an `archive` command (the next step writes it) |
| `refused_revision`, `refused_code`, `refused_reason` | the newest revision the server will not apply as written, and why, in a line that names nobody |
| `removal_started_at`, `removed_at`, `created_at`, `updated_at` | Unix seconds |

It is a **global table** (`trenova_rls.global_tables`), read and written only by the lifecycle under
system scope, for the same reason as the scoped managed state beside it: it is the control plane's
metadata about tenants, not a tenant's data, and it must outlive a removed tenant so the server can
keep answering "removed". Its columns are named `tenant_organization_id` so the purge, which deletes
by `organization_id`, never touches it (and `TestGlobalTablesHoldNoTenantColumns` holds). A unique
partial index on `tenant_organization_id` serves the email tag lookup.
`cloud_tenant_events` is the ledger kept after removal: instance id, kind (`created`, `applied`,
`refused`, `suspended`, `resumed`, `removing`, `removed`), revision, counts and reasons, never a
name, address or record of the tenant.

### Applying an entry

`tenantlifecycle.Service.Apply(instanceID, entry)` is the one write. It runs in one system-scope
transaction holding a transaction-scoped advisory lock per instance id (`trenova:cloud-tenant:<id>`),
so two concurrent applies of the same tenant serialise and the second finds the first's work:

- **Unchanged** (an older revision, or the same revision with the same SHA-256 of the entry) does
  nothing and answers the held state. The same revision with a different digest is applied: the
  control plane composes `account` (plan URL, plan end) when it serves, so a plan end can move
  without a new revision.
- **Refused** entries are recorded, never a 500: a malformed field (`cloudtenant.Desired.Refusal`,
  every problem in one line), a plan URL the feature list refuses, a changed organization name or
  first administrator on a made tenant, or an administrator address already taken anywhere on the
  server ("the first administrator's email address cannot be used on this server": it names nobody).
  The refusal carries the revision and code `invalid`; the agent reports it; the next revision that
  can be applied clears it.
- **First time**: in the same transaction, `tenantbootstraprepository.Bootstrap` makes the business
  unit, organization (placeholders the organization setup gate asks the administrator to finish,
  exactly as the dedicated bootstrap does: `instancebootstrap.Inputs.Organization`), the first
  administrator with an unusable password (`Inputs.Administrator`), the Organization Administrator
  role and its grant, and the reference data. The organization quota is admitted under the new
  organization's own scope (its ids are chosen before the insert), not the server's. The singleton
  `instance_bootstraps` record is never written on a shared server.
- **Every applied revision** writes the scope's plan through `managedfeaturesports.Writer`
  (`SetFeatures`: features, plan URL, add-on endings, plan end, final backup days) and its lifecycle
  (`SetLifecycle`), installs the first administrator's invite when its hash is new and not expired
  (`FirstAdministratorInviteService.InstallFor`, the same idempotent install the dedicated
  `/first-administrator/invite` route uses, for a named administrator), then records the revision
  as applied. Allowances and the AI provider are held in `desired`; what a tenant may meter comes
  from the usage answers (`PUT /managed/v1/tenants/usage`).
- **Suspended**: the scope's lifecycle becomes `suspended`, so the gate answers no feature at all;
  the tenant's sessions are ended by the workflow `CloudTenantSessionsWorkflow`
  (`cloud-tenant-sessions/<instance>/<revision>`); and every request of the organization is refused
  at `planservice.Admission` (the HTTP `ReadOnlyGuard` on every authenticated and device route, and
  the GraphQL mutation guard) and at sign-in (`authservice.createLoginResponse`, before a session is
  made) with `PLAN_RESTRICTED`, reason `organization_suspended`, and `manageUrl`, the console plan
  page from the entry's account. `internal/api/suspension_guard_test.go` fails if a route group
  authenticates without that guard. Public and webhook routes resolve the tenant and ask its plan,
  which a suspended organization does not have. Data is untouched. Resuming sets `active` again.
- **Archive commands** are not carried out by this release: an entry with one is applied, and the
  revision is reported refused ("this release does not take tenant archives…") so the control
  plane's wait for the archive is visible to staff.
- **Removed**: only a tenant with a recorded *final* archive is removed (`HasFinalArchive`); without
  one the revision is refused ("…holds no final archive of the tenant and does not remove it").
  With one, the tenant is suspended, marked `removing` (reported as `applying`) and
  `CloudTenantRemovalWorkflow` (`cloud-tenant-removal/<instance>`, reuse policy "allow duplicate
  failed only") starts; every later apply of a removing tenant starts it again, which is a no-op
  while it runs. `Service.Remove` is the same step for the archive work to call.

Every lifecycle change is audited through `permissionports.SecurityAuditor` after the transaction
commits (`user.created`, `role.created`, `role.assigned` when the tenant is made;
`organization.suspended`, `organization.resumed`), with the system actor and
`metadata.actor = trenova_control_plane` and the control plane instance id; these rows are the
tenant's and go with it. The ledger row survives.

On a shared server the system email sender tags each message with its tenant's instance id
(`tenantlifecycle.InstanceTags`, an otter cache of `cloud_tenants` by organization), so the control
plane lists each bounce to the right tenant; `directory.TenantOfSystemMessage` still finds the
organization from the message itself, never from the event.

### What removal deletes

The workflow's activities are idempotent and resumable; a failed run is started again by the next
apply. Each activity's payload names the tenant, so it runs in that tenant's scope and fairness key.
The removal and the session sign-out are started through the client, never by a schedule, so they
carry no work feature and the entitlement backstop lets them run for a suspended tenant (proven in
`tenantjobs/backstop_test.go` and the removal integration test, which run behind the real
interceptors). The batched row deletes, the organization's cascade and the object, search and cache
deletes each hold a `removal` heavy-job slot (one per organization, one on the instance), so two
removals never delete at once; the purge heartbeats after every batch.

| Store | How it is found | What happens |
|---|---|---|
| Sessions | the organization's memberships, paged by user | a member of no other organization loses every session (`DeleteAllForUser`); a member of another organization loses that organization's sessions and is moved onto the other (`Reassign`) |
| Rows of every table with `organization_id` | `pg_catalog`: every `public` table, partition parents only, then an `EXISTS` per table for this tenant | deleted children first (foreign-key order from the catalog), 500 rows a statement, each statement its own transaction with `statement_timeout 30s` and `lock_timeout 5s`; restricting references from tables outside the plan are deleted first, in batches |
| Rows of tables with only `business_unit_id` (`users`, …) | the same, while no other organization shares the business unit (always on a shared server) | the same |
| Append-only audit rows (`audit_entries`, the AI audit ledger) | their triggers refuse a direct delete (insufficient privilege, raise or integrity error) | left to the organization's `ON DELETE CASCADE`, which `audit_entries`' trigger allows from inside another trigger; a table whose rows survive the cascade fails the verification below rather than being skipped |
| The scope's managed state | `scope_key` = the organization id | `managed_feature_states`, `managed_feature_changes`, `managed_usage_states` rows deleted with the organization |
| The organization, then the business unit | by id | deleted once nothing restricts them |
| Documents and other objects | key prefix `<organization id>/` (documents, thumbnails, renders, extraction output, message files) and `<reporting.artifactPrefix>/<organization id>/` (report runs) | `kernelports.PrefixDeleter.DeletePrefix` |
| Search documents | every Meilisearch index whose filterable attributes include `organization_id` | `DeleteDocumentsByFilter organization_id = "<id>"` (GTC also deletes them as the rows go) |
| Redis keys | `SCAN MATCH *<organization id>*` and `*<business unit id>*` (the GTC entity caches `cache:<entity>:<org>:<bu>:<id>` and `cache:organization:<org>`, and any other key that names the tenant) | `UNLINK`; pulids are unique, so no other tenant's key matches; sessions are keyed by session and user and are ended in the first step |
| Temporal | per-record schedules (agents, conversation schedules, quality runs) | their modules' hourly reconcilers remove a schedule whose record is gone; work already started for the tenant fails to find its rows, and the entitlement backstop refuses scheduled activities of a tenant with no plan |

Then `Remaining` counts every catalog table for the tenant (one `UNION ALL` statement); any row
left fails the activity (retried, then the workflow fails and staff see the tenant applying), and
only an empty count marks the tenant `removed`, records the ledger's `removed` row with the counts,
and reports it. Kept: the `cloud_tenants` row (ids only) and the ledger.

## The listener's tenant routes

The managed listener (`internal/api/managedserver.go`, bearer `managed.token`, constant-time
compare, 30-second timeouts) starts on a shared server too. `internal/cloud/api/tenanthandler`
contributes four routes through `managedroutes.Contributor`; they answer 409 on a dedicated
instance, and the dedicated routes answer 409 on a shared server:

| Route | Body / query | Answer |
|---|---|---|
| `PUT /managed/v1/tenants/{instanceId}` | the control plane's `SharedTenant`, field for field (at most 256 KiB) | `200` the tenant's `SharedTenantStatus` (`instanceId`, `appliedRevision?`, `state`, `administrator?`, `lastArchive?`, `refused?`); `400` an entry for another instance or a revision below 1 |
| `GET /managed/v1/tenants?after=&limit=` | cursor is the last `instanceId`; `limit` 1–200, default 100 | `200 {tenants: [SharedTenantStatus], next?}`; administrator facts are read for the page in one statement |
| `GET /managed/v1/tenants/readings?from=&to=&after=&limit=` | `from`, `to` RFC 3339 on the hour, `from` within 35 days | `200 {tenants: [{instanceId, readings: [{meter, hour, quantity}]}], next?}` for every made tenant that is active or suspended (storage is still kept for a suspended one); each meter is one statement for the whole page (`HourlyCounter.CountByHourForOrganizations`, `LevelReader.LevelsForOrganizations`), so a page costs the same however many tenants it holds |
| `PUT /managed/v1/tenants/usage` | `{tenants: [{instanceId, window?, overage, meters}]}`, at most 1,000 (4 MiB) | `200 {applied, refused: [{instanceId, code, reason?}]}`; each entry becomes that organization scope's usage state |

What the existing routes do on a shared server:

| Route | Shared server | Why |
|---|---|---|
| `GET /readings`, `PUT /state`, `GET|PUT /features`, `PUT /first-administrator/invite` | 409 | they answer the instance scope, which a shared server does not have; the tenant routes replace them |
| `GET /log-requests`, `PUT /log-requests/{id}`, `POST /support/log-pulls` | 409 | log bundles hold every tenant's lines; the instance API refuses them from a shared server |
| `GET|PUT|DELETE /system-email`, `POST /system-email/events` | kept | one sender serves the server; each event is resolved to its tenant by its message |
| `GET|PUT /support-access`, `GET /support-access/grants` | kept | instance-wide key set and grants; the control plane does not yet send a shared server support keys, so nothing sets them |
| `GET /data-usage` | kept | the server's database, documents and search totals, which the agent reports as its telemetry |

`managed.instanceId` is not required on a shared server (it may name the server); `managed.token`
is.

## The tenant agent

The shared server's link to the control plane is `services/operator/cmd/tenant-agent`, a second
binary of the operator module with no Kubernetes or Helm import (see
[operator.md](operator.md#the-tenant-agent)). It reads the tenants (`GET /v1/shared/tenants`, ETag),
puts each tenant the TMS does not hold at that revision, reports the server and its tenants
(`PUT /v1/shared/status`), relays usage both ways every 15 minutes, and sends host telemetry, the
escrowed secrets, the system email credential and its events as the operator does.

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

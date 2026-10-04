# Cloud free tier

How Trenova Cloud (`platform.mode: cloud`) lets a stranger sign up, verify an
email, get a working organization, walk through onboarding, and try the product
inside hard usage limits — and why none of it exists on a self-hosted or local
install unless the operator turns cloud mode on.

Read this before changing anything under `internal/core/domain/platformplan`,
`internal/core/domain/subscription`, `internal/core/services/planservice`,
`internal/core/services/quotaservice`, `internal/core/services/cloudsignupservice`,
`internal/core/services/onboardingservice`,
`internal/infrastructure/postgres/tenantbootstrap`, or the cloud-only routes.

## The switch

Everything here is keyed off one value:

```yaml
platform:
  mode: cloud            # self_hosted (default) | development | cloud
```

| Mode | Plans and quotas | Signup routes | Onboarding wizard |
|---|---|---|---|
| `self_hosted`, `community`, `enterprise`, `development` | unlimited (`UnlimitedPlanService`, `UnlimitedQuotaGuard`) | not registered | never shown |
| `cloud` | the organization's plan from `organization_subscriptions` | registered when `platform.cloud.signup.enabled` | shown until the organization completes it |

To work on the cloud flow locally, set `platform.mode: cloud` in your own
`config/config.yaml` (Turnstile ships Cloudflare's always-pass test keys in
`config.example.yaml`; email goes to the log when `platform.cloud.systemEmail.apiKey`
is empty and `app.env` is not `production`).

An organization with no subscription row in cloud mode is treated as an
**unlimited internal** organization. That keeps the seeded operator organizations
working on cloud.trenova.app; only organizations created by signup get the free plan.

Cloud mode does not need the control plane. Provider selection in
`internal/bootstrap/modules/api/platform.go`:

| Configuration | `EntitlementProvider` | `BillingProvider` | `UsageProvider` |
|---|---|---|---|
| `platform.controlPlane.enabled` | control plane | control plane | control plane |
| `platform.mode: cloud`, no control plane | `LocalEntitlementProvider` (every feature; capabilities are the plan's job) | `platformbillingservice.LocalPlanBillingProvider` | `usageservice.LocalPlanUsageProvider` |
| anything else | `LocalEntitlementProvider` | `LocalBillingProvider` | `NoopUsageProvider` |

Cloud mode requires the postgres driver (the quota guard takes advisory locks).

## Configuration

```yaml
platform:
  mode: cloud
  cloud:
    signup:
      enabled: true
      maxActiveTenants: 500        # trialing + read_only organizations; 0 = no cap
      maxSignupsPerDay: 100        # verified signups per UTC day; 0 = no cap
      perIpPerHour: 3              # signup + resend requests per client IP
      verificationTokenTtl: 24h    # 5m..168h
      blockDisposableEmail: true
      allowedEmailDomains: []      # non-empty = only these domains may sign up
      termsUrl: https://trenova.app/terms
      privacyUrl: https://trenova.app/privacy
    turnstile:
      enabled: true
      siteKey: ""
      secretKey: ""
      verifyUrl: https://challenges.cloudflare.com/turnstile/v0/siteverify
      timeout: 5s
    systemEmail:
      provider: resend
      apiKey: ""
      fromAddress: noreply@trenova.app
      fromName: Trenova
      replyTo: ""
      timeout: 10s
    trial:
      lifetime: 720h               # 30 days of full use
      readOnlyGrace: 336h          # then 14 days read-only, then purge
    freePlan:                      # every key optional; defaults in platformplan
      limits:
        shipments.total: 12        # or nested: shipments: {total: 12}
```

`TRENOVA_PLATFORM_CLOUD_*` environment variables override these like any other key
(`TRENOVA_PLATFORM_CLOUD_SIGNUP_MAXSIGNUPSPERDAY`, `TRENOVA_PLATFORM_CLOUD_SYSTEMEMAIL_APIKEY`).

Go: `config.PlatformConfig.Cloud` (`PlatformCloudConfig`) with `Signup`
(`CloudSignupConfig`), `Turnstile` (`CloudTurnstileConfig`), `SystemEmail`
(`CloudSystemEmailConfig`), `Trial` (`CloudTrialConfig`) and `FreePlan`
(`CloudFreePlanConfig`); `PlatformConfig.IsCloud()`. Read values through the getters,
which supply the defaults above: `GetMaxActiveTenants`, `GetMaxSignupsPerDay`,
`GetPerIPPerHour`, `GetVerificationTokenTTL`, `GetAllowedEmailDomains` (trimmed, lower
case), `GetTermsURL`, `GetPrivacyURL`; `GetVerifyURL`, `GetTimeout`; `GetProvider`,
`GetFromAddress`, `GetFromName`, `GetReplyTo`, `GetTimeout`, `HasAPIKey`; `GetLifetime`,
`GetReadOnlyGrace`; `FreePlan.GetLimitOverrides()` (meter key → value). Viper splits
dotted keys, so `limits` decodes as `map[group]map[name]int64` and the getter joins them
back into meter keys.

The loader refuses, in cloud mode: a non-postgres driver; signup with Turnstile enabled
but no `siteKey` or `secretKey`; a negative free-plan limit. In production and staging it
also refuses signup with Turnstile disabled or with no `systemEmail.apiKey`. An unknown
meter in `freePlan.limits` fails startup when `planservice.NewCatalog` builds the plans.

## Plans

`internal/core/domain/platformplan` defines plans in code. A plan is a key, a name,
a set of **limits** keyed by meter, and a set of **restricted capabilities**.

| Go | Meaning |
|---|---|
| `PlanKey` — `PlanKeyFreeDemo` (`free_demo`), `PlanKeyUnlimited` (`unlimited`) | plan identity, stored in `organization_subscriptions.plan_key` |
| `Limit{Max int64, Window Window}`, `Window` — `WindowLifetime`, `WindowMonthly`, `WindowPerItem` | one meter's limit |
| `Capability` — `CapabilityEmailOutbound`, `CapabilityIntegrations`, `CapabilityAPIKeys`, `CapabilityAgentAutomation`, `CapabilityAgentWebSearch`, `CapabilityCarrierIntelligencePaid`, `CapabilityDocumentIntelligence`, `CapabilitySMS`, `CapabilitySSO` | what a plan can restrict |
| `Plan{Key, Name, Limits, RestrictedCapabilities}` with `Limit(meter)`, `Restricts(cap)`, `Allows(cap)`, `IsUnlimited()`, `MeterKeys()` | a plan |
| `FreeDemo(overrides map[MeterKey]int64) (*Plan, error)`, `Unlimited() *Plan`, `DefaultFreeDemoLimits()`, `ParseLimitOverrides(map[string]int64)` | builders |
| `Catalog` — `NewCatalog(overrides)`, `Get(key)`, `FreeDemo()`, `Unlimited()`; `planservice.NewCatalog` provides it to fx | every plan the instance knows; an unknown key is `ErrUnknownPlan` |
| `ResolvedPlan{Plan, Origin, OrganizationID, BusinessUnitID, Subscription, Status, ResolvedAt}` with `IsManaged()`, `Key()`, `Limit()`, `Allows()`, `AllowsWrites()`, `AllowsLogin()` | what `PlanService.Resolve` returns. `Origin` is `self_hosted`, `internal` (cloud, no row) or `subscription`; `Status` is the subscription's **effective** status at resolution (`Subscription.EffectiveStatus(now)`), so a trial past `trial_ends_at` reads as `read_only` before the sweep has run |
| `ReasonPlanRestricted`, `ReasonSubscriptionReadOnly`, `ReasonSubscriptionExpired`, `ReasonSignupsPaused` | `PLAN_RESTRICTED` reasons (aliases of `errortypes.PlanRestrictionReason*`) |

### The free demo plan (`free_demo`)

| Meter key | Limit | Window | Counted from |
|---|---|---|---|
| `shipments.total` | 12 | lifetime | `shipments` rows (every create path, including bulk duplicate and recurring generation) |
| `recurring_shipments.series` | 1 | lifetime | `recurring_shipments` rows |
| `customers.total` | 8 | lifetime | `customers` rows |
| `locations.total` | 25 | lifetime | `locations` rows |
| `workers.total` | 3 | lifetime | `workers` rows |
| `tractors.total` | 3 | lifetime | `tractors` rows |
| `trailers.total` | 3 | lifetime | `trailers` rows |
| `users.seats` | 1 | lifetime | `user_organization_memberships` rows |
| `documents.uploads` | 25 | lifetime | `documents` rows |
| `documents.storage_bytes` | 104857600 (100 MB) | lifetime | `SUM(documents.file_size)` |
| `documents.file_bytes` | 10485760 (10 MB) | per item | the upload's declared size |
| `ai.assistant_messages` | 25 | calendar month (org timezone) | `assistant_turns` with `origin = 'Person'` created in the month |
| `ai.spend_cents` | 150 ($1.50) | calendar month (org timezone) | `CEIL(SUM(ai_usage_records.cost_usd) * 100)` for rows created in the month |

`users.seats` counts memberships whose user is not the organization's `system` user and
whose `expires_at` is empty or in the future. Every meter key is a `platformcatalog.Meter*`
constant (`MeterShipmentsTotal`, `MeterRecurringShipmentSeries`, `MeterCustomersTotal`,
`MeterLocationsTotal`, `MeterWorkersTotal`, `MeterTractorsTotal`, `MeterTrailersTotal`,
`MeterUserSeats`, `MeterDocumentUploads`, `MeterDocumentStorageBytes`,
`MeterDocumentFileBytes`, `MeterAIAssistantMessages`, `MeterAISpendCents`) and is in the
platform catalog.

"Lifetime" counts the rows that exist now, so deleting a shipment frees a slot.
That is deliberate: the limit is on what a demo organization holds, not on how
many times it clicked Create.

Restricted capabilities on `free_demo` (each returns `PLAN_RESTRICTED`):

| Capability | Why |
|---|---|
| `email.outbound` | Invoices, rate confirmations, tenders, detention notices and driver invites would turn the tier into a spam relay. Authentication email (verification, password reset) uses the platform sender instead. |
| `integrations` | Samsara, QuickBooks, Xero, EDI, Google Maps keys and every other third-party connection. |
| `api_keys` | No programmatic access from a demo tenant. |
| `agent.automation` | Scheduled and event-triggered agents, the morning briefing, background agent runs. Interactive assistant turns remain, inside the AI limits. |
| `agent.web_search` | Paid vendor calls. |
| `carrier_intelligence.paid` | Paid lookups. |
| `document_intelligence` | Extraction and classification model calls. |
| `sms` | Twilio. |
| `sso` | Organization SSO and SCIM configuration. |

## Subscriptions

`organization_subscriptions` (one row per organization — unique on `organization_id` —
tenant table, RLS; domain `internal/core/domain/subscription`, `Subscription`, statuses
`StatusTrialing`/`StatusActive`/`StatusReadOnly`/`StatusExpired` held in a `varchar` with
a CHECK constraint like every other recent enum):

| Column | Meaning |
|---|---|
| `plan_key` | `free_demo` today; Stripe will write paid plan keys here |
| `status` | `trialing` → `read_only` → `expired` (`active` is reserved for paid plans) |
| `trial_ends_at` | when writes stop |
| `read_only_until` | when the organization is purged |
| `stripe_customer_id`, `stripe_subscription_id` | reserved, nullable |

`services.PlanService` (`internal/core/ports/services/plan.go`):

```go
type PlanService interface {
    Resolve(ctx context.Context, orgID, buID pulid.ID) (*platformplan.ResolvedPlan, error)
    RequireCapability(ctx context.Context, tenantInfo pagination.TenantInfo, capability platformplan.Capability) error
    RequireWritable(ctx context.Context, tenantInfo pagination.TenantInfo) error
    IsCloud() bool
    Invalidate(orgID pulid.ID)
}
```

`planservice.CloudService` resolves the plan from the subscription row, cached
in-process for 30 s per organization; `planservice.UnlimitedPlanService` (every non-cloud
mode) returns the unlimited plan without touching the database. `planservice.Module`
picks one from `platform.mode` and is part of the base fx options, so repositories in the
API and the worker can depend on it.

- Resolve reads under the organization's own tenant scope (`dbscope.EnsureTenant`): it
  keeps a caller's scope that already covers the organization (so it joins the caller's
  transaction) and binds the organization otherwise. A missing row means "unlimited", so
  reading it under another tenant's scope would fail open.
- `RequireCapability` returns `PLAN_RESTRICTED` (`plan_restricted`, or
  `subscription_expired` once expired) for a restricted capability; unmanaged
  organizations always pass.
- `RequireWritable` returns `PLAN_RESTRICTED` `subscription_read_only` /
  `subscription_expired` when the subscription no longer allows writes. `ReadOnlyGuard`
  builds on it.
- **Call `Invalidate(orgID)` after any subscription write commits** (provisioning, the
  sweep, billing), from `ports.AfterCommit`. A resolve inside the provisioning
  transaction before the row exists would otherwise cache "unlimited internal" for 30 s.

### Lifecycle

`cloudlifecyclejobs.CloudSubscriptionSweepWorkflow` runs on the `cloud-subscription-sweep`
Temporal schedule (`7 * * * *`, system queue, overlap skipped). `ScheduleProvider` returns
it only when `platform.mode: cloud`, so the reconciler removes it from any other
deployment. `cloudlifecycleservice.Service.Sweep`:

1. `ListDue` (system scope) returns subscriptions due a transition. Each is bound to its
   own tenant and moved to its `EffectiveStatus`: `trialing` past `trial_ends_at` →
   `read_only` (every member but the `system` user gets
   `PlatformEmailService.SendTrialEnded`); `read_only` (or a trial that is also past its
   grace) past `read_only_until` → `expired` (members get `SendAccountPurged`, the
   sessions of members who belong to no other organization are revoked through
   `SessionRepository.DeleteAllForUser`). `UpdateStatus` is optimistic on `version`; each
   write is followed by `PlanService.Invalidate`. A failed transition is counted and the
   sweep carries on.
2. `ListExpired` (system scope) adds every `expired` subscription whose organization still
   exists, so a purge that could not finish is retried.
3. The workflow starts `CloudTenantPurgeWorkflow` as an abandoned child per organization
   with ID `cloud-tenant-purge/<org>/<yyyymmdd>` and `REJECT_DUPLICATE`, so a purge runs at
   most once a day per organization however often the sweep fires.

`CloudTenantPurgeWorkflow` is idempotent and resumable; every step can run again:

| Step | Activity | What it does |
|---|---|---|
| check | `CheckCloudTenantPurgeActivity` | refuses (`Skipped`) unless a subscription row exists and is `expired` — an organization with no row is "unlimited internal" and is never purged — and lists the members before their memberships go |
| rows | `PurgeCloudTenantRowsActivity` | calls `TenantPurgeRepository.PurgeRows` until it reports `Complete` or a pass deletes nothing |
| storage | `PurgeCloudTenantStorageActivity` | deletes every object (all versions) under `<orgID>/` through `storage.PrefixDeleter` (`minio.Client.DeletePrefix`); a backend without it is skipped and logged |
| users | `PurgeCloudTenantUsersActivity` | per member: moves a user who belongs elsewhere to that organization; deletes one who belongs nowhere else, or marks them `Inactive` when a retained row (an audit entry) still references them; revokes their sessions |
| finalize | `FinalizeCloudTenantPurgeActivity` | deletes the `cloud_signups` row, then the organization (cascading the subscription and onboarding rows) and, when no other organization uses it, the business unit; `PlanService.Invalidate` |

`tenantpurgerepository.PurgeRows` discovers the tenant tables from `pg_catalog` (every
`public` table with `organization_id`, plus tables keyed only by `business_unit_id` when
the business unit belongs to this organization alone) and the foreign keys between them,
orders children before parents, clears non-tenant rows that restrict a parent's delete,
and deletes in batches (`(tableoid, ctid) IN (… LIMIT n)`) inside savepoints under one
system-scoped transaction per call (budget: 40 batches of 500 rows). A foreign-key
violation on a self-referencing table deletes that table in one statement; one from a
non-tenant child deletes the referencing rows; one from another tenant table is retried on
the next pass. Search documents follow through the GTC CDC pipeline, which sees the
deletes.

**What a purge keeps.** `audit_entries`, `auth_events`, `ai_audit_events`,
`ai_audit_chain_heads`, `ai_audit_seals`, `ai_audit_exports` and `ai_logs` are never
deleted by the purge: their triggers allow deletes only to the retention sweeps
(`security-audit.md`, `ai-audit-trail.md`). A table whose delete is refused with
`insufficient_privilege` or a raised exception is reported as retained. Because
`audit_entries` cascades from `organizations` and its trigger refuses critical rows younger
than 365 days, deleting the organization row usually fails until those rows age out; the
finalize step then records `RetainedReason`, the organization row, its `expired`
subscription and the retained audit rows stay (every other tenant row, stored object and
signup record is already gone), and the daily re-run completes the purge once retention
allows. Users still referenced by retained audit rows are deactivated, not deleted.

While `read_only`, `ReadOnlyGuard` refuses unsafe requests and GraphQL mutations (see
below) with `PLAN_RESTRICTED` reason `subscription_read_only`. While `expired`, every
request but the account shell is refused with `subscription_expired`.

### Read-only and expired organizations

`middleware.ReadOnlyGuard` runs on the protected and capture-device groups after
authentication (and the control-plane check), and is inert outside cloud mode. It calls
`planservice.Admit`, which resolves the plan once (cached) and refuses:

- every request from an `expired` organization except `GET /users/me`, `/users/me/organizations`,
  `/me/billing`, `/me/entitlements`, `/me/platform-catalog`, `POST /users/me/switch-organization`
  and the auth logout/CSRF routes;
- every API-key request when the plan restricts `api_keys`;
- every unsafe method (`POST`, `PUT`, `PATCH`, `DELETE`) from a `read_only` organization,
  except `/graphql` (left to the GraphQL guard), `POST /users/me/change-password`,
  `/users/me/switch-organization`, `PATCH /users/me/settings`,
  `POST /assistant/turns/:turnID/stop`, comment typing, and routes whose last segment is a
  read action (`preview`, `bulk-preview`, `simulate`, `validate`, `inspect`, `explain`,
  `shop`, `match`, `export`, `backtest`, `compose`, `presence`, `calculate-totals`,
  `calculate-distance`, `check-for-duplicate-bols`, `check-hazmat-segregation`,
  `check-worker-compliance`, `loading-optimization`, `previous-rates`, `edi-214-payload`,
  `preview-prompt`, `inspect-certificate`). A new `POST` route that only reads must be added
  there.

`graphql.ReadOnlyExtension` (registered after `FeatureAccessExtension`) refuses every
mutation from a read-only or expired organization, and any API-key mutation the plan
restricts, with the same error and a 403. No mutation is allowlisted. Background writes
from a read-only organization fail at `QuotaGuard.Enforce`; the recurring-shipment
dispatcher skips such series instead of recording a failure.

## Enforcing a limit
## Enforcing a limit

```go
err := r.db.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
    if err := r.quota.Enforce(ctx, &services.QuotaRequest{
        TenantInfo: req.TenantInfo,
        Meter:      platformcatalog.MeterShipmentsTotal,
        Quantity:   int64(len(rows)),
    }); err != nil {
        return err
    }
    // insert
})
```

Use `WithTx`, not `dbtx.Write`: `dbtx.Write` opens a transaction only when row-level
security is on, and outside one the advisory lock would be released before the insert.
`WithTx` reuses the caller's transaction when there is one, and `rlslint` accepts it.

`services.QuotaGuard` (`internal/core/ports/services/plan.go`):

```go
type QuotaGuard interface {
    Enforce(ctx context.Context, req *QuotaRequest) error
    Check(ctx context.Context, req *QuotaRequest) (*QuotaDecision, error)
    Usage(ctx context.Context, tenantInfo pagination.TenantInfo) (*QuotaUsageSummary, error)
}
```

`quotaservice.CloudGuard.Enforce` must run inside the write transaction that performs the
insert (otherwise `quotaservice.ErrTransactionRequired`). It resolves the plan, refuses a
read-only or expired organization with `PLAN_RESTRICTED`, takes
`pg_advisory_xact_lock(hashtextextended(org || ':' || meter, 0))`, counts with the
meter's counter on the same transaction, and refuses when `used + quantity > limit` with
`QUOTA_EXCEEDED`. Two concurrent creates cannot both take the last slot. A `per_item`
meter compares the quantity alone, with no lock or count. A meter the plan does not
limit, an unmanaged organization and a zero quantity pass; a negative quantity is
`ErrInvalidQuantity`.

`Check` evaluates the same rule without the lock or transaction and returns a
`QuotaDecision` (for pre-flight checks such as an upload session); `Usage` returns every
limited meter with `limit`/`used`/`remaining` and the subscription dates.
`quotaservice.UnlimitedQuotaGuard` is used outside cloud mode; `quotaservice.Module`
selects.

The counters live in `quotacounterrepository` behind `repositories.QuotaCounterRepository`
(`Supports`, `Lock`, `Count`, `OrganizationTimezone`). Monthly windows run from the first
of the month to the first of the next in the organization's timezone
(`timeutils.MonthStart` / `NextMonthStart`). A new limited meter needs a counter there;
`TestEveryCountedFreeDemoMeterHasACounter` fails until it has one.

Where the guard sits, per meter. Repositories take `Quota services.QuotaGuard`
(`optional:"true"`, nil-safe). `quotatx.Run(ctx, db, guard, fn, reqs...)` opens the
transaction only when the guard enforces (`quotaservice.Enforcing`), so outside cloud mode
a create costs nothing extra; `quotaservice.EnforceAll` is the in-transaction form and
`quotaservice.Preflight` the non-locking `Check` that turns a refusal into
`QUOTA_EXCEEDED`.

| Meter | Choke point(s) |
|---|---|
| `shipments.total` | `shipmentrepository.Create` (preflight before the pro number is minted, enforce in the insert transaction), `shipmentrepository.BulkDuplicate` (quantity = copies; also preflighted by `shipmentservice.Duplicate` before the workflow starts), `recurringshipmentrepository.Generate` (one per generated shipment) |
| `recurring_shipments.series` | `recurringshipmentrepository.Create` |
| `customers.total`, `locations.total`, `workers.total`, `tractors.total`, `trailers.total` | each repository's `Create` — the only insert path for each table (service, GraphQL, REST, agent tools, import assistant and onboarding sample data all go through it) |
| `users.seats` | `userrepository.ReplaceOrganizationMemberships` (one per newly added organization) and `driverportalrepository.ActivatePortalAccess` |
| `documents.*` | hard limit in `documentrepository.Create` (`file_bytes`, `uploads`, `storage_bytes`, in that lock order) — covers upload, bulk upload, every new version, the upload-session finalizer and generated documents; preflight in `documentservice.Upload`, `documentuploadservice.CreateSession` and `Complete` through `usageservice.CheckDocumentUploadLimit` and `CheckDocumentBytesLimit` |
| `ai.assistant_messages` | `assistantturnservice.Start` for a person's turn (quantity 1, before the turn row exists) and `assistantservice` turn checks (quantity 0, the turn row already counted) |
| `ai.spend_cents` | `completionrouter` `CompleteStructured`, `CompleteChat`/`StreamChat` and `SubmitBackground` (quantity 1, so a spent month refuses); embeddings are not metered |

Temporal paths: `temporaltype.ToPlanRefusal` turns `QUOTA_EXCEEDED` / `PLAN_RESTRICTED`
into non-retryable application errors of type `quota_exceeded` / `plan_restricted`
(bulk duplicate, SMS). An upload finalizer refused by a limit marks the session failed
with `PLAN_LIMIT_REACHED` and removes the uploaded object. `modelcall.Classify` makes a
plan refusal non-retryable and carries it as `Failure.Plan`, so an agent turn refused by
the spend limit ends with an error event `{code: "usage_limit", limit: {kind:
"plan_limit", meter, used, limit, plan}}`; a turn refused while preparing carries the same
`limit` (rejection params `code: quota_exceeded`).

### Capabilities

`planservice.RequireCapability(ctx, plans, tenant, capability)` and `planservice.Allows`
are the nil-safe helpers every guard uses; services take `Plans services.PlanService`
(`optional:"true"`).

| Capability | Guarded at |
|---|---|
| `email.outbound` | `emailservice.Send` and `SendPersisted` for every purpose except `Authentication` (a queued message is failed, non-retryable) |
| `integrations` | `integrationservice.UpdateConfig` when enabling, `TestConnection`, runtime config of an enabled integration (client runtime config reports not ready) — covers Samsara, Google Maps, fuel cards, email providers; `ediservice.CreateConnection`/`CreatePartner`; `accountingconnectionservice.StartAuthorization`/`CompleteAuthorization`/`ChooseCompany`/`SaveApp` |
| `api_keys` | `apikeyservice.CreateAPIKey`/`RotateAPIKey`, and every API-key request (`ReadOnlyGuard`, GraphQL guard) |
| `agent.automation` | `agentrunservice.StartForDefinition` (every background run); enabling or switching an agent to a scheduled, event or continuous trigger (`agentdefinitionservice` Create/Update/Patch); `conversationscheduleservice.Create`/enabling; scheduled conversation turns (`assistantturnservice.Start`); `briefingservice.Regenerate`. Sweeps skip restricted organizations: agent events (`agentevents.Publisher`), scheduled runs (`plan_restricted`), the daily briefing (`isDue`) |
| `agent.web_search` | `agentextensionservice.UpdateConfig` when enabling, `TestConnection`, tool dispatch (a `ToolError` the agent reads), and `ActiveExtensions` (no tools offered) |
| `carrier_intelligence.paid` | `carrierintelservice.Fetch` (after a cache hit), `SearchCarriers`, `VerifyEquipment`; `AutocompleteCarriers` returns nothing |
| `document_intelligence` | every model call in `aidocumentservice` (routing, extraction, background extraction — the pipeline's deterministic fallback takes over) and `documentintelligenceservice.Reextract` |
| `sms` | `smsjobs.SendSMSActivity` (non-retryable `plan_restricted`) and `workerptoservice` before starting the SMS workflow |
| `sso` | `iamservice` identity provider create/update, SCIM directory and group-mapping create; `organizationservice` Microsoft and Okta SSO config |

## Errors

| Type | Code | HTTP | GraphQL `extensions` |
|---|---|---|---|
| `errortypes.QuotaExceededError` | `QUOTA_EXCEEDED` | 402 | `code`, `type: quota-exceeded`, `params: {meter, limit, used, plan}` |
| `errortypes.PlanRestrictionError` | `PLAN_RESTRICTED` | 403 | `code`, `type: plan-restricted`, `params: {capability, reason, plan}` |

Constructors: `errortypes.NewQuotaExceededError(meter string, limit, used int64, plan string)`
and `errortypes.NewPlanRestrictionError(capability, reason, plan string)` (an empty reason
is `plan_restricted`; the message follows the reason). Helpers: `IsQuotaExceededError`,
`IsPlanRestrictionError`, `Params()`. `params` values are strings on both transports
(`"limit": "12"`), the same map REST puts in the problem document's `params`. For a
`per_item` meter `used` is the size of the item that was refused. The problem titles are
"Plan Limit Reached" and "Not Available on Your Plan".

The web client turns both into the upgrade dialog (`PlanLimitDialog`).

`usageservice.LocalPlanUsageProvider` answers `UsageProvider.CheckLimit` for
`documents.uploads`, `documents.storage_bytes` and `documents.file_bytes` from
`QuotaGuard.Check` (other meters are allowed, reason `not_plan_metered`); a refusal has
reason `quota_exceeded` and carries `plan`, and `usageservice.LimitError` turns it into a
`QuotaExceededError`. `platformbillingservice.LocalPlanBillingProvider` makes
`GET /me/billing` report the real plan, subscription status, `trialEndsAt` /
`readOnlyUntil`, every limited meter's usage and window, and the plan's `restrictions`.
The subscription summary also carries `planKey` (omitted by the control plane), matching
`billingSubscriptionSummarySchema` in the web client.

## Signup

Public routes (`internal/api/handlers/cloudsignuphandler`), behind the global CSRF browser
guard. The signup routes are always mounted so the route table and the write-coverage
ledger see them, but they answer `404` unless `platform.mode: cloud` and
`platform.cloud.signup.enabled` (`CloudSignupService.Enabled`). `signups` and
`signups/resend` also pass an extra per-IP limiter (`perIpPerHour` on the Redis GCRA
`RateLimitStore`, key `cloud_signup:ip:<ip>`, `429` with `Retry-After`):

| Route | Body | Result |
|---|---|---|
| `POST /api/v1/cloud/signups` | `{name, emailAddress, password, companyName, acceptTerms, turnstileToken, website}` | `202 {status: "pending"}` — always the same response whether the email is new, pending or already a user |
| `POST /api/v1/cloud/signups/resend` | `{emailAddress, turnstileToken}` | `202` |
| `POST /api/v1/cloud/signups/verify` | `{token}` | `200` login response + session cookie + `csrfToken`, exactly like `POST /auth/login`; an unknown, expired or used token is a validation error on `token` |
| `GET /api/v1/system/public-config` | — | `{platformMode, signupEnabled, turnstileSiteKey, termsUrl, privacyUrl, freePlan: {limits}}` — always registered |

Bot and abuse defences, in order:

1. **Honeypot** — a non-empty `website` is accepted with the generic 202 and dropped.
2. **Turnstile** — verified server-side against Cloudflare with the client IP; a failed
   or replayed token is a 422 on `turnstileToken`.
3. **Per-IP limit** — `perIpPerHour` on the existing Redis GCRA store.
4. **Email checks** — syntax, MX-shaped domain, disposable-domain blocklist
   (`shared/emailutils`), optional allowlist, and normalization (lower case; Gmail dots
   and `+tags` stripped) for uniqueness.
5. **Password policy** — 12+ characters, at most 72 bytes (bcrypt's limit), not the
   address and not containing its local part, not in the common-password list or a run of
   one character or a sequence (`shared/passwordutils`).
6. **Nothing is created until the email is verified.** `cloud_signups` holds the request
   with a **bcrypt** password hash — the same hash `users.password` uses, so the hash is
   copied onto the owner and the chosen password signs in — and a SHA-256 token hash (64
   lowercase hex characters, enforced by a CHECK) of a 32-byte `crypto/rand` token
   (`shared/tokenutils`). The token is mailed through the platform sender and expires
   after `verificationTokenTtl`. One pending row per normalized address (partial unique
   index); a repeat signup replaces its token with `Refresh` rather than adding a row, and
   `resend` reissues it with `Touch`. Each send counts as an attempt; after 5 the request
   still answers `202` but nothing more is mailed. An address that already belongs to a
   user gets an "you already have an account" email instead, and nothing is stored.

Validation failures use the standard validation problem (the same status every REST
validation error uses), with `errors[].field` one of `name`, `emailAddress`,
`companyName`, `password`, `acceptTerms`, `turnstileToken`. A failed or replayed
Turnstile token and an unreachable Cloudflare both answer on `turnstileToken`; Turnstile
fails closed.
7. **Caps** — `maxSignupsPerDay` and `maxActiveTenants` are checked at verify time; when
   either is hit, verification returns `PLAN_RESTRICTED` reason `signups_paused` and the
   page tells the person they are on the wait list.

Each step writes an `auth_events` row (`provider` = `signup_requested`,
`signup_rejected`, `signup_verified`, `signup_provisioned`) with IP, user agent and the
reason in `error_code` (`honeypot`, `turnstile_rejected`, `turnstile_unavailable`,
`invalid_input`, `existing_account`, `send_limit_reached`, `unknown_signup`,
`invalid_token`, `expired_token`, `signups_paused`, `email_in_use`,
`provisioning_failed`). The provisioned rows carry the new organization and owner.

`cloud_signups` (domain `internal/core/domain/cloudsignup`) holds no tenant data until it
is provisioned, but it is **not** a global table: it carries password hashes, so it is
under forced row-level security with a policy that shows a tenant only the request that
provisioned it (`provisioned_organization_id` / `provisioned_business_unit_id`). Every
access goes through `repositories.CloudSignupRepository`, whose methods each declare
`dbscope.WithSystem` with a reason listed in `rlslint`: `Create`, `GetPendingByTokenHash`
(`FOR UPDATE`; call it inside the provisioning transaction), `GetPendingByEmail`,
`Refresh`, `Touch`, `IncrementAttempts`, `MarkProvisioned`, `Reject` (stores
`rejection_reason`), `Expire(now)` and `CountProvisionedSince(since)`.
`repositories.SubscriptionRepository.CountByStatus` counts active tenants for
`maxActiveTenants`.

### Provisioning

`cloudsignupservice.Verify` runs, under one system-scoped transaction
(`dbscope.WithSystem(ctx, "cloud signup provisioning")` in `Service.provision`, listed in
`rlslint`'s allowlist):

1. `TenantBootstrapRepository.LockProvisioning` takes a transaction advisory lock so the
   caps below hold under concurrent verifications.
2. Lock and re-check the signup row (`GetPendingByTokenHash`, `FOR UPDATE`; single use).
   An expired row is rejected (`Reject`, reason `expired_token`) after the transaction.
3. Caps: `maxSignupsPerDay` against `CountProvisionedSince(start of the UTC day)` and
   `maxActiveTenants` against `CountByStatus(trialing, read_only)`. Either refuses with
   `PLAN_RESTRICTED` `signups_paused` and leaves the row pending.
4. An address that became a user meanwhile (or a unique violation on
   `idx_users_email_address`) rejects the row (`email_in_use`).
5. `TenantBootstrapRepository.Bootstrap` → `tenantbootstrap.Bootstrap` — the same code the
   base seeds run (`internal/infrastructure/postgres/tenantbootstrap`): business unit (a
   random unique `CL…` code), organization, every control
   (`CreateControls`), sequences (`CreateSequences`), the owner user (status Active,
   `MustChangePassword` false, the signup's bcrypt hash, a unique username of at most 20
   characters from the address's local part) and default membership, the Organization
   Administrator role with every permission (`CreateAdminRole`) assigned to the owner
   (`AssignRole`), chart of accounts and accounting defaults (`CreateChartOfAccounts`),
   document types, service-failure reason codes, document template starters and the
   system agent definitions. The instance-wide `system` user (seed 05) belongs to the
   default organization and is shared by every tenant, so no tenant gets its own.
6. `organization_subscriptions`: `free_demo`, `trialing`, `trial_ends_at = now +
   trial.lifetime`, `read_only_until = trial_ends_at + trial.readOnlyGrace`.
7. `organization_onboarding`: `pending`.
8. `MarkProvisioned`; `PlanService.Invalidate(org)` is registered with
   `ports.AfterCommit`.
9. After commit: `signup_verified` / `signup_provisioned` auth events, the welcome email,
   and `AuthService.CreateSessionForUser` — the same session, login response and
   `password` auth event as `POST /auth/login`. The handler sets the cookie and CSRF token.

The organization is created with placeholder profile values the onboarding wizard
replaces (`onboarding.Placeholder*`): name = company name, SCAC `TBDX`, DOT `0`, city
`Pending`, ZIP `00000`, state `NY`, timezone `America/New_York`, both brokerage and asset
operations enabled. The login slug is the company name as an ASCII slug, made unique with
`-2`…`-10` and then a random suffix (`tenantbootstrap.AvailableLoginSlug`); `bucket_name`
is the slug (storage keys are prefixed by organization, so no bucket is created).

`GET /api/v1/auth/csrf` answers `200 {csrfToken: "", headerName}` without a session (or
with a stale cookie), so the signup pages can bootstrap it before anyone is signed in.

### Platform email

`services.PlatformEmailService` (`internal/core/services/platformemailservice`, provided
in the base fx options so the worker can use it) sends platform, non-tenant email through
Resend with `platform.cloud.systemEmail`, reusing `emailservice.ResendSender`. Typed
messages: `SendSignupVerification` (link `{app.webBaseUrl}/signup/verify?token=…`),
`SendSignupExistingAccount`, `SendWelcome`, `SendTrialEnded` (the read-only notice),
`SendAccountPurged`, and `SendRendered` for content rendered elsewhere. Templates are
embedded `templates/<kind>.{subject,html,txt}` with a shared layout; they are platform
content, so they are not tenant-authorable document template kinds. With no
`systemEmail.apiKey` outside production the message (link included) is logged instead; in
production it is `ErrPlatformEmailNotConfigured`.

`passwordresetservice` renders the reset email through the tenant's template as before,
but sends it through `SendRendered` when the organization's plan is managed (a cloud
subscription), because a free organization has no email profile and outbound tenant
email is restricted.

### Turnstile

`services.TurnstileVerifier` (`internal/infrastructure/turnstile`) posts `secret`,
`response`, `remoteip` and `idempotency_key` (the request ID) to `turnstile.verifyUrl`
within `turnstile.timeout`, and accepts only `success` with `hostname` equal to
`app.webBaseUrl`'s host (when set) and `action` equal to the expected `signup` /
`signup_resend`. Cloudflare's documented test secrets skip the hostname check and accept an
empty action. A disabled verifier accepts everything.

### Public config

`GET /api/v1/system/public-config` (always mounted, public, cached 60 s):
`platformMode`, `signupEnabled`, `turnstileSiteKey` (only when signup and Turnstile are
on), `termsUrl` / `privacyUrl` and `freePlan.limits` (meter key → max of the free plan,
overrides applied) in cloud mode; outside cloud mode only the mode, `signupEnabled: false`
and empty limits.

## Onboarding

`organization_onboarding` (tenant table, one row per organization; domain
`internal/core/domain/onboarding`, `Onboarding`, `NewPending`, `Complete(CompleteParams)`)
holds `status` (`pending` | `completed`), `operation_type` (`asset` | `brokerage` |
`both`), `sample_data_loaded`, `completed_at`, `completed_by_id`. A CHECK keeps
`completed_at` set exactly when the status is `completed`.
`repositories.OnboardingRepository`: `Get`, `Create`, `Complete` (optimistic on
`version` and only from `pending`).

| Route | Purpose |
|---|---|
| `GET /api/v1/onboarding/` | state + whether the wizard applies (`required` is false outside cloud mode) |
| `POST /api/v1/onboarding/complete/` | `{organization: {name, timezone, addressLine1, city, stateId, postalCode, scacCode?, dotNumber?}, operationType: "asset"\|"brokerage"\|"both", loadSampleData}` |

`internal/core/services/onboardingservice` and `internal/api/handlers/onboardinghandler`.
`GET` is open to any signed-in user of the organization and returns `required: false,
status: "completed"` outside cloud mode or when the organization has no onboarding row;
while `pending` it returns the organization's name and timezone with the placeholder
fields blank, and once completed the full profile (placeholder SCAC/DOT stay blank).
`POST complete/` needs `organization:update`, runs in one transaction and answers the new
state. It validates the body (field errors under `organization.<field>`, plus
`operationType`), updates the organization through `OrganizationService.Update` (its
validation errors are re-keyed to `organization.<field>`; a blank SCAC or DOT keeps the
placeholder), sets `brokerage_enabled` / `asset_operations_enabled` from
`operationType`, and when `loadSampleData` is true creates the sample set through the
normal services, so it counts against the limits and any `QUOTA_EXCEEDED` rolls the whole
completion back: the reference data a shipment needs (a "Customer Facility" location
category, `TRACTOR` and `DRYVAN` equipment types, a manufacturer, service type `STD`,
shipment type `FTL`, and the standard formula template library via
`InstallStandards`, using "Flat Rate"), then 2 customers, 4 geocoded locations in Texas,
1 worker at the organization's address, 1 tractor driven by that worker, 1 trailer and
2 shipments with a pickup and a delivery each. A second completion is a `409`.

The web app redirects any signed-in user whose organization's onboarding is
`pending` to `/onboarding` while `platformMode` is `cloud`.

## Login hardening (all modes)

- Failed logins are counted per account and per IP in Redis
  (`repositories.LoginThrottleStore`, `internal/infrastructure/redis/repositories/loginthrottle.go`;
  account keys hold a SHA-256 of the address, never the address). After 5 failures for an
  account within a sliding 15 minutes it is locked for 30 s, doubling per further failure
  up to 15 minutes; after 20 failures from one IP in an hour that IP is refused until the
  hour ends. A throttled attempt is a `429` with `Retry-After`
  (`errortypes.RateLimitError.WithRetryAfter`, written by the error handler) and is checked
  before the password, so it reveals nothing. A successful login clears the account
  counter. Tripping and refusing both write `auth_events` rows (`provider`
  `login_throttled`, `error_code` `login_throttled_account` / `login_throttled_ip`). A
  Redis failure is logged and the attempt allowed.
- In cloud mode, login is refused for an organization whose subscription no longer allows
  it (`ResolvedPlan.AllowsLogin`, i.e. `expired`). Without an organization slug, a user
  whose current organization is expired is moved to another membership that still allows
  login; with none left, the refusal says the trial has ended
  (`error_code` `subscription_expired`). `read_only` organizations still sign in.
- `security.encryption.allowLocalKeyManagerInProduction` lets a single-host production
  deploy run `app.env: production` with `keyManager: local`; the key must then be at least
  32 characters with no placeholder text. Without it production still requires a cloud
  KMS. Cloud should run production mode so introspection is off, the
  persisted-operation allowlist is enforced and HSTS is sent.

## Repositories and services at a glance

| Port | Implementation | Scope |
|---|---|---|
| `repositories.SubscriptionRepository` — `GetByOrganization`, `Create`, `UpdateStatus` (optimistic), `ListDue`, `ListExpired`, `CountByStatus` | `subscriptionrepository` | tenant; `ListDue`, `ListExpired` and `CountByStatus` system |
| `repositories.TenantPurgeRepository` — `ListMembers`, `OrganizationProfile`, `PurgeRows`, `PurgeUser`, `DeleteTenant` | `tenantpurgerepository` | system |
| `storage.PrefixDeleter` | `minio.Client.DeletePrefix` | |
| `cloudlifecycleservice.Service` | sweep and purge steps | |
| `repositories.OnboardingRepository` | `onboardingrepository` | tenant |
| `repositories.CloudSignupRepository` | `cloudsignuprepository` | system |
| `repositories.TenantBootstrapRepository` — `LockProvisioning`, `Bootstrap` | `tenantbootstraprepository` (wraps `tenantbootstrap`) | system |
| `repositories.LoginThrottleStore` | Redis `loginThrottleStore` | |
| `services.CloudSignupService` | `cloudsignupservice` | |
| `services.OnboardingService` | `onboardingservice` | tenant |
| `services.PlatformEmailService` | `platformemailservice` | |
| `services.TurnstileVerifier` | `turnstile.Verifier` | |
| `repositories.QuotaCounterRepository` | `quotacounterrepository` | the caller's transaction |
| `services.PlanService` | `planservice.CloudService` / `UnlimitedPlanService` | |
| `services.QuotaGuard` | `quotaservice.CloudGuard` / `UnlimitedQuotaGuard` | |

Mocks for each live in `internal/testutil/mocks` (`MockPlanService`, `MockQuotaGuard`,
`MockSubscriptionRepository`, `MockOnboardingRepository`, `MockCloudSignupRepository`,
`MockQuotaCounterRepository`, `MockTenantBootstrapRepository`, `MockLoginThrottleStore`,
`MockCloudSignupService`, `MockOnboardingService`, `MockPlatformEmailService`,
`MockTurnstileVerifier`). `internal/testutil/plantest` builds resolved plans and
restricting `MockPlanService`s for guard tests; `dbtest.NewSQLMock` gives a
`*postgres.Connection` over sqlmock for repository quota tests. The tables arrive in migration
`20261231008150_cloud_free_tier`.

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

`CloudSubscriptionSweepWorkflow` runs hourly on a Temporal schedule (cloud mode only):

1. `trialing` past `trial_ends_at` → `read_only`; the owner gets a platform email.
2. `read_only` past `read_only_until` → `expired`; sessions for the organization's
   users are revoked and the tenant purge job is started.
3. The purge deletes the business unit, organization and every tenant row under them,
   then the users who belonged to no other organization, then the storage bucket prefix.

While `read_only`, `ReadOnlyGuard` refuses every unsafe REST method and every GraphQL
mutation except sign-out, password change and the onboarding/subscription reads, with
`PLAN_RESTRICTED` and reason `subscription_read_only`. While `expired`, login is refused.

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

Where the guard sits, per meter:

| Meter | Choke point(s) |
|---|---|
| `shipments.total` | `shipmentrepository.Create`, `shipmentrepository.BulkDuplicate`, `recurringshipmentrepository.Generate` — the three places that mint a pro number |
| `recurring_shipments.series` | `recurringshipmentrepository.Create` |
| `customers.total`, `locations.total`, `workers.total`, `tractors.total`, `trailers.total` | each repository's `Create` (covers service, GraphQL, REST, agent tools, import assistant) |
| `users.seats` | membership insert in `userrepository` and `driverportalrepository.ActivatePortalAccess` |
| `documents.*` | `documentservice.Upload`, `documentuploadservice.CreateSession`/`Complete` via `LocalPlanUsageProvider`: `usageservice.CheckDocumentUploadLimit` (count) and `usageservice.CheckDocumentBytesLimit` (file size and total storage) |
| `ai.*` | `completionrouter` before each model call, and `assistantservice` before a turn |

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

## Signup

Public routes, registered only when `platform.mode: cloud` and
`platform.cloud.signup.enabled`, all behind the global CSRF browser guard and an extra
per-IP limiter (`perIpPerHour`):

| Route | Body | Result |
|---|---|---|
| `POST /api/v1/cloud/signups` | `{name, emailAddress, password, companyName, acceptTerms, turnstileToken, website}` | `202 {status: "pending"}` — always the same response whether the email is new, pending or already a user |
| `POST /api/v1/cloud/signups/resend` | `{emailAddress, turnstileToken}` | `202` |
| `POST /api/v1/cloud/signups/verify` | `{token}` | `200` login response + session cookie, exactly like `POST /auth/login` |
| `GET /api/v1/system/public-config` | — | `{platformMode, signupEnabled, turnstileSiteKey, termsUrl, privacyUrl, freePlan: {limits}}` — always registered |

Bot and abuse defences, in order:

1. **Honeypot** — a non-empty `website` is accepted with the generic 202 and dropped.
2. **Turnstile** — verified server-side against Cloudflare with the client IP; a failed
   or replayed token is a 422 on `turnstileToken`.
3. **Per-IP limit** — `perIpPerHour` on the existing Redis GCRA store.
4. **Email checks** — syntax, MX-shaped domain, disposable-domain blocklist
   (`shared/emailutils`), optional allowlist, and normalization (lower case; Gmail dots
   and `+tags` stripped) for uniqueness.
5. **Password policy** — 12+ characters, not the email, not in the breached-password
   top list.
6. **Nothing is created until the email is verified.** `cloud_signups` holds the request
   with an argon2id password hash and a SHA-256 token hash (64 lowercase hex characters,
   enforced by a CHECK); the token is mailed through the platform sender and expires after
   `verificationTokenTtl`. One pending row per normalized address (partial unique index);
   a repeat signup replaces its token with `Refresh` rather than adding a row.
7. **Caps** — `maxSignupsPerDay` and `maxActiveTenants` are checked at verify time; when
   either is hit, verification returns `PLAN_RESTRICTED` reason `signups_paused` and the
   page tells the person they are on the wait list.

Each step writes an `auth_events` row (`signup_requested`, `signup_rejected`,
`signup_verified`, `signup_provisioned`) with IP, user agent and the reason.

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
(`dbscope.WithSystem(ctx, "cloud signup provisioning")`, listed in `systemscopelint`):

1. Lock and re-check the signup row (single use).
2. `tenantbootstrap.Bootstrap` — the same code the base seeds run for every seeded
   organization: business unit, organization, every control, sequences, the
   Organization Administrator role with every permission, chart of accounts and
   accounting defaults, document types, service-failure reason codes, document
   template starters, system user, system agent definitions.
3. The owner user (status Active, `MustChangePassword` false), membership, role assignment.
4. `organization_subscriptions` row: `free_demo`, `trialing`, timestamps from `trial`.
5. `organization_onboarding` row: `pending`.
6. After commit: a session is created exactly as login does, and a welcome email is sent.

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

Completing sets the organization profile, the brokerage/asset capability flags from
`operationType`, and when `loadSampleData` is true creates a small sample set through
the normal services (so it counts against the limits): 2 customers, 4 locations,
1 worker, 1 tractor, 1 trailer, 2 shipments.

The web app redirects any signed-in user whose organization's onboarding is
`pending` to `/onboarding` while `platformMode` is `cloud`.

## Login hardening (all modes)

- Failed logins are counted per account and per IP in Redis; after 5 failures for an
  account in 15 minutes it is throttled with exponential backoff (429 with
  `Retry-After`), and after 20 failures from one IP in an hour that IP is throttled.
  A successful login clears the account counter. Each throttle writes an `auth_events` row.
- `security.encryption.allowLocalKeyManagerInProduction` lets a single-host production
  deploy run `app.env: production` with `keyManager: local`; the key must then be at least
  32 characters with no placeholder text. Without it production still requires a cloud
  KMS. Cloud should run production mode so introspection is off, the
  persisted-operation allowlist is enforced and HSTS is sent.

## Repositories and services at a glance

| Port | Implementation | Scope |
|---|---|---|
| `repositories.SubscriptionRepository` — `GetByOrganization`, `Create`, `UpdateStatus` (optimistic), `ListDue`, `CountByStatus` | `subscriptionrepository` | tenant; `ListDue` and `CountByStatus` system |
| `repositories.OnboardingRepository` | `onboardingrepository` | tenant |
| `repositories.CloudSignupRepository` | `cloudsignuprepository` | system |
| `repositories.QuotaCounterRepository` | `quotacounterrepository` | the caller's transaction |
| `services.PlanService` | `planservice.CloudService` / `UnlimitedPlanService` | |
| `services.QuotaGuard` | `quotaservice.CloudGuard` / `UnlimitedQuotaGuard` | |

Mocks for each live in `internal/testutil/mocks` (`MockPlanService`, `MockQuotaGuard`,
`MockSubscriptionRepository`, `MockOnboardingRepository`, `MockCloudSignupRepository`,
`MockQuotaCounterRepository`). The tables arrive in migration
`20261231008150_cloud_free_tier`.

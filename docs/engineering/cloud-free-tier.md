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
      verificationTokenTtl: 24h
      blockDisposableEmail: true
      allowedEmailDomains: []      # non-empty = only these domains may sign up
    turnstile:
      enabled: true
      siteKey: ""
      secretKey: ""
      verifyUrl: https://challenges.cloudflare.com/turnstile/v0/siteverify
    systemEmail:
      provider: resend
      apiKey: ""
      fromAddress: noreply@trenova.app
      fromName: Trenova
      replyTo: ""
    trial:
      lifetime: 720h               # 30 days of full use
      readOnlyGrace: 336h          # then 14 days read-only, then purge
    freePlan:                      # every key optional; defaults in platformplan
      limits:
        shipments.total: 12
```

`TRENOVA_PLATFORM_CLOUD_*` environment variables override these like any other key.

## Plans

`internal/core/domain/platformplan` defines plans in code. A plan is a key, a name,
a set of **limits** keyed by meter, and a set of **restricted capabilities**.

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
| `ai.assistant_messages` | 25 | calendar month (org timezone) | user turns in `ai_usage_records` / assistant messages |
| `ai.spend_cents` | 150 ($1.50) | calendar month (org timezone) | `SUM(ai_usage_records.cost_usd)` |

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

`organization_subscriptions` (one row per organization, tenant table, RLS):

| Column | Meaning |
|---|---|
| `plan_key` | `free_demo` today; Stripe will write paid plan keys here |
| `status` | `trialing` → `read_only` → `expired` (`active` is reserved for paid plans) |
| `trial_ends_at` | when writes stop |
| `read_only_until` | when the organization is purged |
| `stripe_customer_id`, `stripe_subscription_id` | reserved, nullable |

`planservice.Service.Resolve(ctx, orgID, buID)` returns the effective plan (cached
in-process for 30 s, invalidated on write). In non-cloud modes it always returns
the unlimited plan without touching the database.

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
err := dbtx.Write(ctx, r.db, func(ctx context.Context) error {
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

`QuotaGuard.Enforce` must run inside the write transaction that performs the
insert. It takes `pg_advisory_xact_lock` on `(organization, meter)`, counts with the
meter's registered counter on the same transaction, and refuses when
`used + quantity > limit`. Two concurrent creates cannot both take the last slot.

Where the guard sits, per meter:

| Meter | Choke point(s) |
|---|---|
| `shipments.total` | `shipmentrepository.Create`, `shipmentrepository.BulkDuplicate`, `recurringshipmentrepository.Generate` — the three places that mint a pro number |
| `recurring_shipments.series` | `recurringshipmentrepository.Create` |
| `customers.total`, `locations.total`, `workers.total`, `tractors.total`, `trailers.total` | each repository's `Create` (covers service, GraphQL, REST, agent tools, import assistant) |
| `users.seats` | membership insert in `userrepository` and `driverportalrepository.ActivatePortalAccess` |
| `documents.*` | `documentservice.Upload`, `documentuploadservice.CreateSession`/`Complete` via `LocalPlanUsageProvider` |
| `ai.*` | `completionrouter` before each model call, and `assistantservice` before a turn |

## Errors

| Type | Code | HTTP | GraphQL `extensions` |
|---|---|---|---|
| `errortypes.QuotaExceededError` | `QUOTA_EXCEEDED` | 402 | `code`, `type: quota-exceeded`, `params: {meter, limit, used, plan}` |
| `errortypes.PlanRestrictionError` | `PLAN_RESTRICTED` | 403 | `code`, `type: plan-restricted`, `params: {capability, reason, plan}` |

The web client turns both into the upgrade dialog (`PlanLimitDialog`).

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
   with an argon2id password hash and a SHA-256 token hash; the token is mailed through
   the platform sender and expires after `verificationTokenTtl`.
7. **Caps** — `maxSignupsPerDay` and `maxActiveTenants` are checked at verify time; when
   either is hit, verification returns `PLAN_RESTRICTED` reason `signups_paused` and the
   page tells the person they are on the wait list.

Each step writes an `auth_events` row (`signup_requested`, `signup_rejected`,
`signup_verified`, `signup_provisioned`) with IP, user agent and the reason.

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

`organization_onboarding` (tenant table) holds `status` (`pending` | `completed`),
`operation_type`, `sample_data_loaded`, `completed_at`.

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
  deploy run `app.env: production` with `keyManager: local`; without it production still
  requires a cloud KMS. Cloud should run production mode so introspection is off, the
  persisted-operation allowlist is enforced and HSTS is sent.

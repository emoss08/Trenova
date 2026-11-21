# Managed mode

A dedicated instance run by Trenova Cloud is the public TMS with `managed.enabled: true`. The
control plane sells it allowances (shipments, Document Intelligence pages, document storage,
organizations) and bills overage; the TMS counts what was used and stops a metered action when the
control plane says so. AI runs on the customer's own model provider, so there are no AI usage
credits: the TMS records every model call for the customer's own analytics and the AI audit
trail, and meters none of them. The operator carries both ways ([operator.md](operator.md#usage-relay));
this doc is the TMS's side. Self-hosted installs never turn it on, and nothing here runs there.

## What turns on

| Piece | Where | Self-hosted | Managed |
|---|---|---|---|
| `QuotaGuard` | `identity/app/quotaservice` (`New` picks by `managed.enabled`) | `UnlimitedQuotaGuard` | `ManagedQuotaGuard` |
| `UsageProvider` | `kernel/app/usageservice`, chosen in `kernel/module.go` | `NoopUsageProvider` | `ManagedUsageProvider` |
| `FeatureGate` | chosen in `kernel/module.go` (`featureGate`) | `entitlementservice.AllowAll` | `managedfeaturesservice.Service` |
| The operator's listener | `internal/api/managedserver.go`, API process only | off | `managed.host:managed.port` (8090), bearer `managed.token`; on a shared server also the tenant agent's routes, contributed by `internal/cloud` through `managedroutes.Contributor` |
| The first administrator | `trenova db bootstrap --admin-setup`, `identity/app/firstadmininviteservice` | `password` | `link`, then an invite from the operator |
| Trenova support access | `identity/app/supportaccessservice`, `POST /api/v1/support/enter` | not routed, nothing read | grants, keys from the operator, staff entry ([Support access](#support-access)) |
| The system email sender | `workspace/app/systememailservice` (reads only with `managed.enabled`), chosen per message in `emailservice` | none: every email needs the organization's profile | the sender Trenova supplies, for system email with no profile |
| The built-in search service | `agent/app/builtinembeddingservice` (only with `managed.enabled` and `managed.embedding`), chosen in `completionengine.UsableFor` | none: search by meaning needs the organization's Embedding provider | the instance's `embedding` component, behind the organization's own route ([Search by meaning](#search-by-meaning)) |

Both providers ask one gate, `managedusageports.Gate` (`kernel/app/managedusageservice`), so every
existing hook enforces the control plane's answer with no new call site: shipments
(`shipmentrepository`, `Preflight` then `EnforceAll` in the insert transaction), document storage
(`CheckDocumentBytesLimit` and the document insert's `quotatx`), Document Intelligence pages
(`aidocumentservice.admit`, before a call or a deferred submission, asking for the document's page
count, or one page when it is not known; routing is not an extraction) and organizations
(`tenantbootstraprepository.Bootstrap`, the one place an organization is created, under the
provisioning lock).

Everything here is held per **scope** ([shared-tenancy.md](shared-tenancy.md)). A dedicated
instance (`managed.tenancy: dedicated`, the default) has one scope, the instance: its organizations
share its features, its shipments, its pages and its storage, none of them has an allowance of its
own, and every reader counts across all organizations (`dbscope.WithSystem`). A shared server
(`managed.tenancy: shared`) has one scope per organization, and the same tables, services and rules
answer for each organization alone. The listener's routes below write the instance scope and
answer 409 on a shared server, where the same listener serves the tenant agent's
`/managed/v1/tenants` routes instead; each existing route's decision is in
[shared-tenancy.md](shared-tenancy.md#the-listeners-tenant-routes).

## The state

`managed_usage_states` holds one row per scope (`scope_key`: `instance`, or an organization's id on
a shared server; a global table): the open window, whether overage is billed, and each app meter's
quantity, allowance, hard cap, ceiling and `stopped`, exactly as the control plane last answered.
The operator replaces the instance's row every relay (`PUT /managed/v1/state`). Every API and worker
process reads it through a 30-second in-process cache (otter, keyed by scope, at most 10,000
entries); the process that takes the write drops that scope's entry at once, the others within the
TTL. `managedusageports.Gate.Decide` and `State` take the scope; `ManagedQuotaGuard` and
`ManagedUsageProvider` resolve it from the tenant they are given (`managedscope.Resolver.ForTenant`).

`managedusage.Decide` is the whole rule:

- no state yet, or a meter the state does not name: not limited (a fresh instance works before
  its first relay, and the control plane decides which meters it sells);
- no open window: every named meter stops (unpaid, or the subscription ended);
- `stopped`: the meter stops;
- a ceiling (`ceiling` > 0), when the requested quantity would cross it: the action stops whatever
  the overage setting (`ReasonCeiling`, refused with `managedusage.CeilingRefusal`, which names the
  ceiling and no price). A ceiling is 0 for none or at least the allowance;
- a hard cap, or overage not billed: the action stops when the requested quantity would cross the
  allowance, so one large batch cannot overrun it between two reports.

A level meter (storage, organizations) is decided on its live level, read through its
`LevelReader` (`Service.Decide`, `managedusage.DecideAt`), not on the quantity the control plane
echoed: deleting documents frees room at once, not at the next relay. A counted meter is decided on
the state's quantity, which is what the control plane has billed.

A refusal is `errortypes.QuotaExceededError` with the meter, allowance and quantity, the same
error the Cloud edition's plans raise. When the allowance is what stopped it, a managed refusal
carries the meter's own words (`managedusage.Refusal`): storage offers turning on overage "from your
Trenova account" or deleting documents, pages offers overage or the next period, and organizations
says how many the plan includes. The TMS never names a price. The client's plan-limit dialog says
the same through its copy table (`apps/web/src/lib/plan-limit-copy.ts`, plan `managed`).

## Storage

- **Warning.** At `managedusage.StorageWarnRatio` (80%) of the included storage, the organization
  administrators (`organization:update`) of the scope's organizations (every organization for the
  instance, the one organization on a shared server) get one notice
  (`storagequotaservice.Warnings`, a `managed_state_observers` member that
  `managedusageservice.Receiver` tells after every state the operator sets). It compares the
  previous state with the new one, so it fires once per crossing; dropping back below resolves the
  notice (`ResolutionCleared`), and the next crossing sends a new one. The documents tab shows
  administrators a banner from `GET /documents/storage/` (used, included, overage, warning,
  stopped), read from `QuotaGuard.Usage`.
- **At the limit.** With overage off, an upload that would cross the allowance is refused with the
  storage wording; with overage on, uploads continue and the control plane bills the excess.
- **The level is cheap.** `document_storage_totals` holds the bytes per organization, kept by
  statement triggers on `documents` (insert, update, delete, cascades included) in the same
  transaction as the write. The reading and every upload check add up one row per organization,
  never the documents. The last state applies until the next one: when the
control plane or the operator is down, the instance keeps working on what it was last told.

## The feature list

Trenova is sold as a base plan plus add-ons. The spec's `features` names the feature keys the
customer bought, and the operator sends that list on every reconcile tick (about every 15 seconds):
`PUT /managed/v1/features` with `{"features": ["tms.accounting", "ai.desk"], "planUrl": "https://…",
"addonEndings": [{"addon": "ACC", "endsAt": "<RFC 3339>"}]}`, answered 204. The list is the whole
set: it replaces what the instance holds, and an empty list means the base plan alone.
`GET /managed/v1/features` answers what the instance holds, `{"features": [...],
"receivedAt": "<RFC 3339>"}`, with an empty list and no `receivedAt` before the first one.

The same request carries two more things from the spec, both optional and both kept when a
request leaves them out:

- **`planUrl`**, the spec's `account.planUrl`: the instance's plan page in the Trenova console
  (`{console origin}/instances/{instanceId}/plan`), where add-ons are bought. The client renders it
  as a link, so the listener refuses anything but an absolute `http` or `https` URL with a host and
  no user information, at most 2,048 characters (`managedfeatures.Received.Validate`, a 400
  naming `planUrl`). The control plane adds `account` to every spec it serves, from where its
  console is served; no stored revision holds it, so every instance has it whenever it was
  ordered, and a console that moves changes the spec's `ETag` and reaches every instance on the
  next poll. A spec served without one makes the operator send none.
- **`addonEndings`**, the spec's `addonEndings`: add-ons whose removal is scheduled, with when each
  ends; they stay on the plan until then. The operator sends the list on every tick, an empty one
  when the spec has none, so the instance's list is replaced each time. At most 50; each `addon`
  is shaped like a catalogue add-on key (`^[A-Z][A-Z0-9]{1,9}$`) and `endsAt` is required, or the
  request is refused with a 400. A well-shaped key the catalogue does not know is dropped and
  logged (at warn when the state changes, then at most hourly while it is still sent), not refused,
  for the same reason as an unknown feature. An add-on listed twice keeps its earliest end.
- **`planEndsAt`**, the spec's `account.endsAt`: when the plan ends, the close of the period
  already paid for, while an owner has set it to end. Unlike the two above it is a statement of
  fact on every request: the operator always sends the key, an RFC 3339 time when the plan is
  ending and `null` when it is not, so keeping the plan clears the end at once. A request without
  the key is read as not ending. The control plane composes `account.endsAt` when it serves the
  spec, and the spec's `ETag` changes with it, so the operator's next tick carries it with no new
  revision. Once the plan has ended the control plane keeps serving the same end while the
  instance is taken down, so the end may be in the past: the listener stores it and announces
  nothing (withdrawing it would read as a plan that was kept). It refuses only an end before 1970
  or more than 400 days ahead (a yearly period is the longest), with a 400 naming `planEndsAt`.
- **`finalBackupDays`**, the spec's `account.finalBackupDays`: how many days the final backup is
  kept after the instance is taken down. It is the control plane's retention setting (30 by
  default), composed when the spec is served like the rest of `account`, and the operator sends it
  whenever the spec has it. A request without it, or with 0, has not said, and the instance keeps
  what it holds (`final_backup_days`, 30 until told). The listener refuses a value below 1 or above
  3650 with a 400 naming `finalBackupDays`. A changed number is stored and announced to nobody.

### The plan's end

An owner never destroys a running instance: cancelling in the console sets the plan to end at the
close of the period already paid for. The instance keeps running, add-ons included, until then;
at that time the control plane decommissions it (a final backup kept for the control plane's
retention, `finalBackupDays`, then teardown). Until then an owner or admin can keep the plan, which
clears the end. Every sentence that says how long the final backup is kept takes the number from
the stored state when it is written (the notifications' `PlanEndNotice.FinalBackupDays`, the page's
`managedPlan.finalBackupDays`), never fixed text.

The state row holds the end (`plan_ends_at`), the end the administrators were told about
(`plan_end_announced`) and the end they were reminded of (`plan_end_reminded`), all Unix seconds
like the rest of the row. `managedfeatures.Replace` decides the notice in the transaction that
stores the list, so a repeated list or a retried request announces nothing twice:

- an end that differs from the announced one is announced ("Your plan ends on {date}"); an end
  learned less than three days ahead is marked reminded at once, since the announcement is enough;
- no end while one was announced is a kept plan ("Your plan continues"), and clears both marks;
- the same end again, or no end while none was announced, changes nothing.

The three-day reminder ("Your plan ends in 3 days") runs on the `managed-plan-end-reminder`
schedule (`workspace/jobs/planendjobs`, hourly, created only when `managed.enabled`): one
activity calls `managedfeaturesports.PlanEndReminder`, which finds the active scopes whose end is
less than three days away, not yet past and not yet reminded (`DuePlanEndReminders`, at most 500 a
run, `idx_managed_feature_states_plan_ends_at`), then for each locks and reads its row, sends the
reminder and stores `plan_end_reminded` in the same transaction.

Every notice goes through the same observer as the add-on notices (`workspace/app/plannotice`),
to the administrators of the scope it concerns (every organization for the instance, the one
organization on a shared server) (`organization:update`, at most 25, `NotifyPermitted`),
keyed `managed-plan:plan-end` so a later notice about the end replaces an open earlier one, and
linking to `/admin/plan`. The receiver tells the observers after every stored change, and the
observer publishes `managed_plan`/`updated` to each organization, so an open Your plan page
refetches (`RESOURCE_QUERY_KEY_MAP.managed_plan` in `realtime-patching.ts`). The page shows the
end as a warning notice above the usage, with Keep plan opening `manageUrl` for whoever sees it.

`managed_feature_states` holds one row per scope (`scope_key`, with `scope_business_unit_id` for an
organization; a global table) with the list, the plan URL (`plan_url`, null until one arrives), the
add-on endings (`addon_endings`) and the scope's `lifecycle` (`active`, or `suspended`: a suspended
organization has no features, base included; the instance is always active, and
`managedfeaturesservice.SetLifecycle` refuses it). `received_at` is 0 for a scope suspended before
its first list. The list is stored normalised:
`managedfeatures.Normalize` drops base features (always on), duplicates and keys the catalogue
(`platformcatalog`) does not know, and sorts the rest, so the same set is always the same value.
An unknown key is ignored rather than refused because a newer control plane may sell a feature
this release does not have yet; refusing would stop the whole list from reaching the instance.
`managedfeaturesservice.SetFeatures` validates the request, compares the list, the plan URL and the
endings with the stored ones and writes nothing when all are equal; a change is stored with the
receive time, drops the cache and is logged with the count, the features added and removed, the
endings set and called off, and any ignored keys (at warn).

`managedfeaturesservice.Service` is the `FeatureGate` in managed mode, read through a 30-second otter
cache keyed by scope, like the usage state. `Enabled` answers for the context's scope (an error on
a shared server when the context carries no tenant); `EnabledAnywhere` answers the instance's list
on a dedicated install and, on a shared server, whether any active organization has the feature
(`AnyActiveScope`, cached per feature for the same 30 seconds and dropped on any write):

- a base feature is always on, with no read;
- an add-on feature is on only when the stored list holds it;
- before any list arrives, only base features are on for the instance; on a shared server an
  organization the control plane has sent no plan for has nothing at all, base included, and a
  suspended one has nothing either;
- a key the catalogue does not know is off.

The background schedules follow the list too. Every schedule names the feature its work belongs
to, and the worker's schedule reconciler keeps one whose feature is off created but paused, with a
note naming the feature the plan does not include. The worker asks the gate once a minute for the
features its schedules name that are off, and re-reconciles when that set changes, so buying or
dropping an add-on starts or stops its schedules within about a minute and a half (the cache TTL
plus the check interval), with no restart ([jobs.md](backend/jobs.md#a-schedule-names-its-feature)).

Self-hosted installs keep `AllowAll`: every feature is on and the table is never read.

### The change record, the notice and the page

- **The record.** `managedfeaturesrepository.Replace` takes the received list under a
  transaction-scoped advisory lock per scope (`keel.LockKey`, so two puts never both record a change), reads
  the held list, and lets `managedfeatures.Replace` decide. A list equal to the held one writes
  nothing. A different one replaces `managed_feature_states` and inserts one row into
  `managed_feature_changes` (a global table: when, the keys `added` and `removed`, and `starting`)
  in the same transaction, then deletes every row of that scope but its newest 200. The first list an instance
  receives is recorded with `starting = true`: it is the starting plan, not a change. A change row
  also records `ending` (an add-on on the plan newly set to end, or whose end moved, with its new
  end) and `kept` (an add-on whose scheduled end was called off while it is still on the plan; one
  that left on its day is `removed`, not kept). A new plan URL, or an ending for an add-on not on
  the plan, updates the state row and records no change (`Replacement.Updated` against
  `Replacement.Changed`).
- **The notice.** `managedfeaturesservice.Receiver` (what the listener calls) tells the
  `managed_plan_observers` fx group after a change to the feature list
  (`Replacement.FeaturesChanged`; an ending, a call-off or a new plan URL tells nobody, so it never
  makes every organization re-read its permissions), so the gate service that every module depends
  on never depends on the notification code (no fx cycle). `workspace/app/plannotice` is the
  member: it groups the change by add-on (`managedfeatures.Shift`) and, for every add-on that left
  (all its features off, at least one on before) or joined, notifies each organization's
  administrators (`organization:update`, at most 25, `NotifyPermitted`, like the storage warning):
  "<Add-on> is no longer on your plan" with "Everything you entered is kept…", or "<Add-on> is now
  on your plan". The starting plan and a change inside an add-on that stays on notify nobody. The
  notice links to `/admin/plan` (`data.link`) and is keyed per add-on
  (`managed-plan:<add-on>`), so a second change to the same add-on while the first notice is open
  updates it. Observers run after the write with a 10-second bound and a context that outlives the
  request; a failure is logged and the listener still answers 204, because the row is the durable
  record and the page shows it. The text is written with `i18n.T`, in the default language, as
  every stored notification is.
- **The page.** `managedPlan` (GraphQL, `organization:update`) answers the base plan's features,
  the add-ons on and off the plan with their features in catalogue order (an add-on on the plan
  whose removal is scheduled carries `endsAt`), and the 20 newest changes that moved an add-on,
  set one to end or kept one, worked out by walking back from the held list through the change
  rows (`managedfeatures.History`). It also answers `manageUrl`, the stored plan URL (null until
  one arrives), and `usage` for the current billing period (null without a usage state or with
  one that has no window): the period, and per meter the catalogue's name for it in the reader's
  language (`platformcatalog.Meters`, a meter the catalogue cannot name is left out), its unit,
  `used`, `included`, `hardCap` and `stopped`. `used` is the state's quantity, except for a level
  meter, which is read live the way `Decide` reads it (`managedusageservice.Usage`: one read per
  level meter the state names, a handful). No price, SKU or machine size is ever answered.
  Self-hosted it answers `managed: false` with nothing else and reads no table. The client knows
  the instance is managed from the public config's `managed` (`GET /system/public-config`); the
  page (`/admin/plan`, "Your plan") and its admin link (`managedOnly`) exist only then. The page
  shows usage first, then the add-ons on the plan ("On", or "Ends {day}") and those that can be
  added, what every plan includes, and recent changes. Its header's **Manage plan** opens
  `manageUrl` in a new tab, and each add-on that can be added links to `manageUrl` with
  `?add=<add-on key>` (any query the URL carries is kept), asking the console's plan page to open
  with that add-on switched on, ready to confirm. Neither is drawn while `manageUrl` is null.

### A base flow that needs an add-on's record: the standard rating templates

Pricing a shipment is base; writing rating logic is the Rating add-on. The rule has one owner per
layer and is the model for any base flow that has to choose a record an add-on owns:

- **The record kind stays the add-on's.** `formula_template` keeps `FeatureRating`, so the
  permission engine refuses every template route, GraphQL field, studio, AI and test endpoint, the
  agent tools and the client routes (`createPermissionLoader` shows "not on your plan") with no
  check of their own.
- **A base set is marked on the row.** `formula_templates.standard_key` names the catalogue entry
  (`formulatemplateservice/standardcatalog`) a standard template was installed from. Standard rows
  are never changed on any plan (`FormulaTemplate.RefuseStandardChange` at every write entry:
  update, version, rollback, schedule, archive); duplicating or forking one makes the
  organization's own.
- **Every organization has them, installed on first use.** `formulatemplateservice.EnsureStandards`
  runs at the entry of the picker and the template list: one indexed read of the organization's
  keys (`uk_formula_templates_standard_key`), then an in-process note (otter, single-flight per
  tenant) so later calls read nothing. Missing entries are checked, tested, approved and inserted
  in one transaction under the requesting person; a name the organization already uses gets
  " (Standard)". No organization-creation hook and no backfill job: a new catalogue entry reaches
  every organization the same way.
- **The base read paths filter by plan.** The `FORMULA_TEMPLATE` select option lists standard rows
  only without Rating (`StandardOnly`); `GET /pricing-templates/:templateID/inputs` (shipment,
  rate agreement or billing control read) answers a standard template's billing inputs on every
  plan and the organization's own only with Rating and `formula_template` read.
- **Base writes refuse a new choice the plan withholds.** `permissionports.PricingTemplatePlan`
  (implemented by `formulatemplateservice`) answers `PLAN_RESTRICTED` when a shipment, a new or
  changed rate agreement lane, or the billing fallback picks one of the organization's own
  templates without Rating. An unchanged choice is kept, and a shipment that keeps one keeps its
  stored tariff inputs.
- **Evaluation is the engine's.** `rateengine.price` refuses a non-standard template without
  Rating (`tariffports.Template.Standard`, read with the template it already loads, so no extra
  query). A shipment's own rating then holds its bill (`shipmentcommercial.rateOwnOnPlan`: the
  stored charges stand, nothing is deleted); a contract lane records an error quote and the shipment
  falls back to manual pricing. Everything returns with the add-on.
- The shipment's rating method, tariff inputs and rating detail, and the invoice's template name
  and tariff notes, are base fields; only the nested `Shipment.formulaTemplate` record stays
  Rating's.

## The readings

`GET /managed/v1/readings?since=<hour>` answers from the source rows, never from a counter the
TMS keeps, so a resent hour is always the truth and a missed relay is made good by the next:

| App meter | Unit | Kind | Source | Index |
|---|---|---|---|---|
| `shipments.total` | count | counted | shipments created in the hour (`shipmentmeter`) | `idx_shipments_created_at` |
| `document_intelligence.pages` | count | counted | `sum(pages)` of the hour's successful Document Intelligence extraction calls (`aiusagemeter`); an extraction records its document's page count, or one when none is known | `idx_ai_usage_records_feature_created_at` |
| `documents.storage_bytes` | bytes | level | the bytes stored now, from `document_storage_totals`, reported for the current hour (`documentmeter`) | the table's primary key |
| `organizations.total` | count | level | the organizations on the instance now, reported for the current hour (`organizationmeter`) | `organizations`' primary key |

Each reader lives in the module that owns its table and joins the fx groups
`managed_hourly_counters` or `managed_level_readers`; the kernel service fills the hours with
no rows with zeros. A new meter is a reader in its module plus the control plane's `appMeter`.
Each reader takes the scope: the instance is read across organizations (`dbscope.WithSystem`,
listed in `rlslint`) because a dedicated instance is one customer, and an organization scope is read
under its own tenant scope through the tenant-first indexes (`idx_shipments_org_bu_created_at`,
`idx_ai_usage_records_tenant_feature_time`, `document_storage_totals`' key, and
`idx_organizations_business_unit` for the organizations of its business unit). The listener asks for
the instance, so it keeps answering instance totals. `since` must be on the hour, within 35 days and not after the current
hour. Each reader also answers many organizations of a shared server in one statement
(`CountByHourForOrganizations`, `LevelsForOrganizations`, grouped by organization under system
scope and listed in `rlslint`), which `managedusageports.OrganizationReadings` serves to the tenant
readings route, so a page of tenants costs one statement per meter.

Storage is billed at its highest point in the cycle. An hourly level is enough for that: storage
moves with uploads and deletions over hours, not seconds, and the control plane takes the maximum
of the hourly levels it received in the window. A resent hour replaces that hour's level.

## Its own address

A customer may give an instance one address of their own (`tms.acme.com`). The spec then carries
`customHostname`, which the operator renders as the chart's `aliases: [customHostname]` and
`primaryDomain: customHostname` ([operator.md](operator.md#from-spec-to-values)). What that means
for the instance:

- **Two addresses are served.** The first address (`hostname`) keeps working for ever and is never
  redirected; the custom one is served beside it by the same Ingress, the same client and the same
  API. The files host stays under the first address.
- **Origins are lists.** `server.cors.allowedOrigins` and `security.csrf.trustedOrigins` name both
  origins, and so do the bundled MinIO's CORS and the documents bucket's rules. The client's
  Content-Security-Policy names the page's own origin as `'self'`, so it holds on either address.
- **Links are built on the primary.** `app.webBaseUrl`, `app.problemTypeBaseUri`,
  `security.passwordReset.baseUrl`, `portal.baseUrl` and `reporting.deliveryLinkBaseUrl` use the
  custom address while the spec carries it, and the first address when it does not (the control
  plane leaves it out while the customer's address has fallen back). Everything that derives an
  absolute URL from `webBaseUrl` follows: notification and record links, invoices, the driver
  portal, Trenova Capture's pairing page and discovery answer, SCIM resource locations, and the
  default accounting callbacks.
- **Sessions are per address.** The session cookie is host-only (`__Host-`), so a session started
  on one address is not carried to the other: people sign in again on the custom address.
- **Single sign-on works from either address.** An SSO configuration's redirect URL is the
  organization's own setting and does not move. When a person starts SSO on the other address
  and the redirect URL's origin is one the API admits (`server.cors.allowedOrigins`), the start
  endpoint first sends the browser to the same start endpoint on the redirect URL's address,
  carrying only the checked `returnTo`, moved to that address (or to its home when it is not a
  plain path). The whole flow then runs on one address and the person ends signed in there, which
  may be the other address from the one they started on. A redirect URL on any other origin is
  left as it was (`authservice.ssoStartHandoff`). An accounting connection's callback follows the
  primary unless the configuration names one, and finishes on that address.
- **Trenova is Trenova on either address.** An agent's portal site may cover neither address: the
  extension portal service refuses a site that covers the address links use or any origin the API
  admits.
- **Staff and the control plane stay on the first address**: the support entry, the first
  administrator's invite and the readiness probes.

## What the instance's data takes

`GET /managed/v1/data-usage` (bearer `managed.token`, the managed listener only, so it does not
exist with `managed.enabled` off) answers what the whole instance's data takes, for the operator
to report beside its own disk readings:

```json
{"measuredAt": "2026-11-16T12:00:00Z", "databaseBytes": 123, "documentsBytes": 456, "searchBytes": 789}
```

| Field | What it is | Source |
|---|---|---|
| `databaseBytes` | the instance's database | `pg_database_size(current_database())` (`kernel/adapters/postgres/databasesize`, system scope, listed in `rlslint`) |
| `documentsBytes` | the bytes of every document stored, across all organizations | the `documents.storage_bytes` meter's own reader (`documentmeter`, `document_storage_totals`): one value serves both, so the figure and the billed level never disagree |
| `searchBytes` | the search index | Meilisearch's `GET /stats` `databaseSize`, through the TMS's search client (`meilisearch.NewIndexSize`) |

Every figure is optional: one Trenova could not measure is left out, never sent as zero, and a
measured empty index is `0`. `searchBytes` is absent when search is not configured. Only these
three instance totals are ever answered, never a figure per organization. `measuredAt` is when the
oldest figure in the answer was measured, or the time of the answer when it carries none. `200`
always, `401` for a wrong token; the operator reads a `404` as a release that does not measure.

**Never measured on the request.** `kernel/app/datausageservice` answers from an in-process otter
cache, one entry per figure, each kept 30 minutes after it was last measured (otter's write
expiry, on the service's clock). A request starts a measuring round in the background when the
last one began 5 minutes ago or more, and never waits for it, except the first request after the
process starts, which waits at most 2 seconds and then answers what is known (possibly only
`measuredAt`). A round measures every source at once, each under its own timeout (database and
documents 10 seconds, search 5), so a slow or failing source never blanks the others: its last
good value is kept until it expires 30 minutes after it was measured, and then the figure is
absent. A round runs only when the operator asks (about once a minute), so a self-hosted install,
whose handler is never mounted, measures nothing. A new figure is a `datausageports.Source` in the
`managed_data_sources` fx group, a field in the answer and the operator's `DataUsage`.

## What the control plane must send

The state (`PUT /managed/v1/state`) names each app meter the subscription sells, with:

| `meter` | `unit` | `allowance` | `hardCap` | `quantity` |
|---|---|---|---|---|
| `shipments.total` | `count` | shipments included in the window | `false` (overage billed when `overage` is on) | shipments counted in the window |
| `document_intelligence.pages` | `count` | pages included in the window (1,000 with the Document Intelligence add-on) | `false` | pages counted in the window |
| `documents.storage_bytes` | `bytes` | bytes included | `false` | the latest hourly level received, not the cycle's maximum (the warning compares one state with the next) |
| `organizations.total` | `count` | organizations the plan includes (Production 1, Production Plus 5, Enterprise as negotiated) | `true`, always: never overage | the latest level received |

- `overage` is whether the customer turned overage on; it applies to every meter without
  `hardCap`.
- `stopped` is a kill switch (unpaid, suspended), never the allowance: the instance decides the
  allowance itself. For a level meter the instance uses its live level, so `stopped` set because
  the cycle's maximum crossed the allowance would keep refusing after the customer deleted
  documents.
- The units are `count` and `bytes`; `cents` is refused (no app meter is priced in money).
- A meter the state does not name is not limited; no state at all limits nothing.

The readings (`GET /managed/v1/readings`) list exactly these four meters: the two counted meters
for every hour from `since` to the current hour, the two levels for the current hour.

The spec carries, beside the plan, what the operator passes to the TMS on every tick:

- `account` (`planUrl`, `finalBackupDays`, `endsAt`, `supportUrl`, `newSupportRequestUrl`),
  composed at serve time ([The feature list](#the-feature-list), [Trenova support from the account
  menu](#trenova-support-from-the-account-menu)).
- `support` (`instanceId`, `staffDomain`, one to four Ed25519 `keys`), composed at serve time; the
  `ETag` changes with it ([Support access](#support-access)). Without it the instance can never
  admit staff, so the control plane sends it for every managed instance.

In return each status report carries `supportGrants`, the complete list of the instance's live
grants, or leaves it out when the TMS could not be read ([The live
grants](#the-live-grants-instance-to-operator-to-control-plane)).

## The first administrator

The person who ordered the instance becomes its first administrator, and chooses their password
from a link Trenova Cloud emails them. The instance never holds that link's token, only its
SHA-256, so a dump of the instance yields nothing that can be redeemed, and nothing depends on the
instance having an email provider.

1. **Bootstrap.** The operator renders `bootstrap.{orgName, adminName, adminEmail}` from the spec's
   `firstAdministrator` with `bootstrap.adminSetup: link`. The release Job runs
   `trenova db bootstrap --admin-setup link`, which creates the organization and an administrator
   the way an invited user is created: an unusable random password, no sign-in and no password
   change (`User.AwaitingAccountSetup`). No password is passed or read.
2. **Invite.** Once the instance first runs, the control plane mints a token, keeps it sealed, and
   puts its SHA-256 and expiry in the spec as `firstAdministrator.invite`. The operator calls
   `PUT /managed/v1/first-administrator/invite` with them. `firstadmininviteservice.Install` finds
   the administrator through the `instance_bootstraps` record and stores the hash as their
   account invite (`password_reset_tokens`, purpose `invite`), expiring at the earlier of the
   given time and `security.passwordReset.inviteTokenTtl`. It is idempotent by hash:
   - a live invite with that hash is reported as `inviteTokenSha256`;
   - a used, expired or replaced one is reported as nothing and never made live again;
   - a new hash retires the outstanding invites and stores the new one, audited through the
     `SecurityAuditor`;
   - an administrator who has chosen a password or signed in is reported as `setUp`.

   Before the bootstrap it answers 409.
3. **Delivery.** The operator reports the answer as the status report's `administrator`. When the
   instance holds the current hash, the control plane emails
   `https://<hostname>/auth/reset?token=<token>&purpose=invite`, the TMS's own account setup page
   and `POST /api/v1/auth/reset-password`.

A password set at bootstrap (`--admin-setup password`, self-hosted) records `password_changed_at`,
so that administrator is not mistaken for one awaiting setup.

**The link stays good until it is used.** Nothing an upgrade or a reconcile runs retires it: the
release Job's bootstrap answers "completed" without touching the administrator or their tokens,
and the operator puts the same hash again, which `Install` reports as live. Redeeming it is one
transaction (`passwordresetservice.ResetPassword`): the token is marked used, the password
written and sibling links retired together, so a write that fails leaves the link working. Only
a lookup that finds no live token answers "This link is no longer valid"; any other failure is a
server error the person can retry. A new link asked for from the console replaces the old one.

4. **Organization setup.** The bootstrap gives the organization only its name, the default time
   zone and state, and stand-ins the database accepts for the rest (`tenant.Placeholder*`: SCAC
   `TBDX`, DOT `0`, city `Pending`, ZIP `00000`, no street address). `Organization.SetupMissing`
   names every required field still blank or a stand-in; `Organization.Validate` refuses to save
   one, so the organization update is the only way out of it. `GET /users/me` carries
   `organizationSetup {complete, missing, canComplete}` (`canComplete`: the session may update
   the organization), read on every session check. While it is incomplete, `AppSession` shows
   `OrganizationSetupGate` in front of every page, after the password and role gates: an
   administrator fills the missing details in through the same `updateOrganization` mutation
   the settings page uses, and anyone else is told an administrator has to finish it and may
   sign out. It applies the same way to a self-hosted install whose bootstrap left a field out.

## System email

Every managed instance can send its **system email** with no setup, through a sender Trenova
supplies; email the organization sends to anyone else still needs its own email profile and
domain.

**Two kinds.** Every email names its type (`emailports.SendEmailRequest.Type`), and the type
declares its kind where it is defined (`workspace/domain/email/types.go`, `defineType` or
`defineInvitation`): no type exists without one, and `TestEveryEmailTypeDeclaresAKind` fails
for one that has none.

| Type | Kind | Sent by |
|---|---|---|
| `PasswordReset` | system | `passwordresetservice` |
| `AccountInvitation` | system, invitation | `passwordresetservice` (a person invited to the organization) |
| `DriverPortalInvitation` | system, invitation | `driverportalinvitationservice` |
| `Notification` | system | `notificationservice` (a notification's email) |
| `NotificationDigest` | system | `notificationservice` (the daily digest) |
| `ProposalReminder` | system | `agentproposalnotifier` |
| `RecordShare` | system | `recordshareservice` (a record shared with a colleague) |
| `ReportDelivery` | system | `reportjobs` (a scheduled report) |
| `Invoice` | customer-facing | `invoicesendservice` |
| `CustomerStatement` | customer-facing | `customerstatementservice` |
| `PaymentReminder` | customer-facing | `paymentreminderservice` |
| `FactoringNoticeOfAssignment` | customer-facing | `factoringnoaservice`, the factoring agent tools |
| `FactoringDocumentRequest` | customer-facing | `factoringreadinessservice`, the factoring agent tools |
| `FactoringSubmission` | customer-facing | `factoringtransport` |
| `CustomerRequest` | customer-facing | `requestemail` (an agent asks a customer for documents or a status) |
| `CustomerUpdate` | customer-facing | `customerupdateservice`, the agent's customer email tool |
| `CustomerTrackingLink` | customer-facing | `customertrackingservice` |
| `CustomerStatusUpdate` | customer-facing | `customertrackingservice` |
| `CarrierTrackingLink` | customer-facing | `checkcallservice` |
| `TenderOffer` | customer-facing | `tenderdispatch` |
| `RateConfirmation` | customer-facing | `rateconfirmationservice` |
| `DetentionNotice` | customer-facing | `detentionnoticeservice` |
| `InboundReply` | customer-facing | `inboundreplyservice`, the inbox reply tool |
| `MailboxTest` | customer-facing | `inboundmailbox` (a test of the organization's own mail routing) |
| `TemplateTest` | customer-facing | the document template test send |
| `ProfileTest` | customer-facing | an email profile's test send |

**Who sends** (`emailservice.route`, the one place a sender is chosen, for `Send` and for
`ResolveSender`'s previews alike):

1. the organization's own profile, exactly as before: the one the request names, else the one
   assigned to its purpose, else the one assigned to General;
2. with none, and only for a **system** type, the Trenova sender when the instance holds one;
3. otherwise the "no email profile" refusal (`emailports.ErrNoEmailProfile`, remedy
   `SetUpEmailProfile`), which is all a self-hosted install or a customer-facing email ever gets.

A customer-facing type never reaches step 2, and `SendPersisted` refuses a message marked as
Trenova's whose type is not a system type, before it reads the credential. The database holds it
too: `ck_email_messages_sender` allows `sender = 'Trenova'` only with no profile and
`kind = 'System'`. Every message records `type`, `kind` and `sender` in the email log.

**What the Trenova sender may carry** (`admitTrenovaSend`, after the suppression list):

- only addresses of the organization's own people (a membership in it, `RecipientDirectory`);
  a recipient outside it refuses the whole send with the "add an email profile" remedy. The one
  exception is an invitation type, whose address is not a user yet;
- at most 3,000 recipients an hour across the instance (burst 300), and 30 invitations an hour
  per organization, in the shared rate-limit store (`kernelports.RateLimitStore`, Redis), so
  every API and worker process counts against one budget;
- the suppression list and bounce handling as for any message, through the relay below.

It sends as `<organization name> via <fromName>` from Trenova's `fromAddress`, with no reply-to
(an organization has no reply address of its own outside a profile), and tags every message with
the instance's ID (`trenova_instance=<managed.instanceId>`: a Resend tag, Postmark `Metadata`;
`SendProviderMessage.Metadata`, set only in `trenovaProvider`, never on an organization's own
profile). The API key goes to the provider under its own config key (Resend `apiKey`, Postmark
`serverToken`). A provider failure is kept in the email log as "Trenova's system sender could not
deliver this email", with its retry class; the provider's own words are logged on the instance
only, with the API key removed.

**The instance's ID.** `managed.instanceId` (a lowercase UUID, the chart's `managed.instanceId`,
which the operator sets from the spec) is required while `managed.enabled`: the configuration is
refused at start without it, and the chart refuses to render without it, so a managed instance
never sends through Trenova's sender untagged. `trenovaProvider` also refuses, without retrying,
a message it would have to send untagged.

**Bounces and complaints** come back through the control plane. The provider account is
Trenova's, so its webhook is account-wide: it posts every instance's events to the control plane,
which verifies each delivery and keeps the bounces and spam complaints for the instance the
`trenova_instance` tag names. The operator fetches them and relays them to
`POST /managed/v1/system-email/events` ([the contract](#the-contract)). The TMS feeds each
payload to the same parsing as an organization's own webhooks (`emailservice.ParseProviderEvent`)
and handles it like one (`HandleRelayedSystemEvent`, then `HandleProviderEvent`), minus the
signature check: the message, found by provider and provider message ID among the ones Trenova
sent across tenants (`TenantOfSystemMessage`, `dbscope.WithSystem`), names the organization whose
suppression list it updates. The instance holds no webhook secret and has no public endpoint for
Trenova's sender. `HandleProviderEvent` records the event, the message's status and the
suppression in one transaction, and the event is unique per provider and provider event ID (a
relayed Resend event is keyed by the relayed event's ID, which the control plane keeps once per
Svix delivery; a Postmark one by its record ID), so a relayed event delivered twice changes
nothing the second time, and one that failed halfway is applied whole when it comes again.

**The sender on the instance.** `system_email_senders` holds one row (`id = 'current'`, a global
table): provider, from address and name, the API key sealed by the TMS's own encryption (`kernelports.SecretCipher`, purpose `system_email_credential`), the fingerprint and
when it arrived. `systememailservice` reads it through a 30-second one-entry otter cache; the
process that takes a write drops its entry at once, the others within the TTL. A row that does
not open (the encryption key that sealed it is gone) or does not match its fingerprint counts as
unset, so the operator's next tick sends it again. No API or page returns it; the email settings
page tells an administrator "System email is sent by Trenova on your behalf", with the from
address, when the instance holds a sender and the organization has no General profile (the case
in which it is used). Self-hosted installs never read the table.

### The contract

The operator delivers the sender and relays its bounces on the managed listener:

| Call | Body | Answer |
|---|---|---|
| `PUT /managed/v1/system-email` | `{"provider": "Resend" \| "Postmark", "apiKey", "fromAddress", "fromName"}`, at most 8 KiB, no other field | 204; 400 for a malformed body or a value of the wrong shape, naming the field and never the value; 413 past 8 KiB |
| `GET /managed/v1/system-email` | | `{"configured": true, "provider", "fromAddress", "fingerprint"}`, or `{"configured": false}` |
| `DELETE /managed/v1/system-email` | | 204, also when there was none |
| `POST /managed/v1/system-email/events` | `{"events": [{"id": "<uuid>", "provider": "Resend" \| "Postmark", "payload": "<the provider's webhook body, verbatim>"}]}`: 1 to 50 events, each `id` a distinct lowercase UUID, no other field, at most 8 MiB (50 payloads of 65,536 bytes is 3.2 MB; the cap leaves room for JSON escaping) | 200 `{"handled": ["<id>", …]}`; 400 for a malformed batch; 413 past 8 MiB |

Shapes: `apiKey` 20 to 512 printable ASCII characters with no space; `fromAddress` a bare
address of at most 254 characters on a domain Trenova verified with the provider; `fromName` 1
to 40 printable characters with no leading or trailing space.

An event is **handled** when it was applied, when it is for a message this instance did not
send, or when it cannot be read, is not a bounce or complaint, or carries an empty payload or one
past 65,536 bytes; each of those is logged with its ID, provider and reason, never its payload,
so a bad event never blocks the queue. It is left out only when applying it failed in a way worth
retrying (a database or dependency error), and the operator relays it again on its next tick.

The **fingerprint** is the lowercase hex SHA-256 of the UTF-8 text `trenova.system-email.v1`,
`provider`, `fromAddress`, `fromName` and `apiKey`, joined by a single line feed, in that order.
`systememail.Fingerprint` (TMS) and `internal/systememail.Fingerprint` (operator) compute it; both
tests compute the expected value from this formula for a Postmark and a Resend credential. A
`PUT` whose fingerprint equals the stored one writes nothing.

From the control plane (`api/instance.yaml`):

- the spec carries `systemEmail: {"fingerprint": "<hex>"}` while Trenova supplies a sender, and
  nothing when it does not;
- `GET /v1/secrets` carries `systemEmail: {"provider", "apiKey", "fromAddress", "fromName"}`
  while the spec names one. It is one sender for the whole fleet, set by Trenova staff and sealed
  by the control plane's keyring; unlike the other secrets, it may change. The webhook's secret
  stays on the control plane and never reaches an instance;
- `listCommands` answers `emailEvents: true` while bounces or complaints wait for the instance;
  the operator then fetches them (`GET /v1/email-events`, at most 50), relays them here and
  acknowledges exactly the IDs the TMS answered as handled (`POST /v1/email-events/ack`). An
  unacknowledged event is listed again, and deleted after 7 days;
- the operator reports the fingerprint the instance holds with each status report
  (`systemEmailFingerprint`), so staff can see when every running instance holds a new sender
  before they revoke the old key.

**Rotation**: the control plane stores the new credential and serves its fingerprint with the
spec from then on (it is added when the spec is served and is part of its ETag; no spec revision
is written). The operator fetches it, puts it, and the instance replaces the old row in one
write; until then it keeps sending with the old credential, and a message queued before the swap
is sent with whichever credential is stored when its workflow runs. The old credential must stay
valid at the provider until the instance reports the new fingerprint (the operator's `GET` shows
it within a tick). **Removal**: the control plane drops `systemEmail` from the spec; the operator
calls `DELETE`, and system email with no profile is refused as on a self-hosted install. A queued
message whose sender was removed fails without retrying.

## Search by meaning

Every managed instance with Trenova AI searches by meaning with no setup, through a search
service on the instance: the chart's `embedding` component (Ollama serving `nomic-embed-text`
v1.5 from its own image, `deploy/Dockerfile.embedding`). An organization may still route the
Embedding task to a model of its own. Trenova tunes for `nomic-embed-text` only (the similarity
floor, the eval gate and the ranking fixtures are recorded against it), and AI control says so.

**Who embeds** (`completionengine.Engine.UsableFor` for `aitask.TaskEmbedding`, the one place an
embedding provider is chosen, for indexing, searches, the catalog and the configured model alike):

1. the organization's own enabled providers that serve Embedding, in their routing order, exactly
   as before;
2. after them, the built-in service, when `builtinembeddingservice` offers it: only with
   `managed.enabled`, `managed.embedding.url` and `managed.embedding.model` set, and `ai.desk` on
   in the feature gate. A self-hosted install never reaches it, whatever its config says.

No row is written for an organization. The built-in provider is built per call
(`aiprovider.BuiltInEmbedding`): one fixed ID (`aiprv_trenova_search`) for every organization,
so its health, its two concurrent calls and its usage rows are the instance's, OpenAI-compatible
at `<url>/v1`, nomic task prefixes, 768 dimensions, priced at zero (so its calls are neither
charged to the indexing budget nor counted as unpriced), batches of 8 and a 120-second timeout
for two CPU threads. The configured model is the first candidate's, so an organization's own
route wins; the service stays behind it, so searches pinned to the old model keep working
through a model change either way. Its address is instance configuration, not an
organization's provider: the engine gives it the client that may reach a private address
(`reachesPrivateNetwork`), while an organization's provider still needs
`ai.privateNetworkProviders`, which the operator forces off.

**Recording the model.** Vectors are stored under the provider's model key
(`<host>/<model>@<dimensions>`), as for any provider, so the existing model change carries every
switch: an organization that routes its own model gets it as the pending model, indexes beside
the service's vectors and swaps; one that removes its route falls back to the service and does
the same in reverse.

**Indexing starts when routing changes** (each is a `SignalWithStart` on the organization's index
workflow, so repeats join the run in flight):

- a provider write that touches the Embedding task (create, update, patch, delete, assign,
  route, reorder; `aiproviderservice.providersChanged`) wakes that organization's indexer;
- Trenova AI turning on (`retrievalplanobserver`, a `managed_plan_observers` member) starts one
  retrieval sweep, `retrieval-index-sweep-plan`, 45 seconds later (past the feature cache's 30)
  and joined by a second change while it waits;
- the hourly sweep visits every organization and wakes one whose routed model is neither the
  active nor the pending one (`RetrievalSweepResult.RoutingChanged`), which also covers a new
  organization and an instance that gained the service with a release.

The config (`managed.embedding`) is written by the chart from `embedding.enabled` (the service
URL `http://<release>-trenova-embedding:11434` and `embedding.model`); `managedconfig.Validate`
refuses a URL with a path, credentials or another scheme, and a model without a URL. The
retrieval status (`aiRetrievalStatus`) answers `embeddingSource` (`BuiltIn`, `Organization`, or
absent), `tunedModel` and `configuredModelTuned`, and the AI control Retrieval tab says "Search by
meaning runs on your instance. There is nothing to set up." or "Trenova tunes search for
nomic-embed-text. With another model, search may find less."

## Trenova support from the account menu

Support requests live in the Trenova console, so the instance has to know where it is. The
control plane adds two addresses to every spec's `account`: `supportUrl` (the console's support
page, the organization's requests) and `newSupportRequestUrl` (its new request form with this
instance chosen and marked as opened from Trenova, `{console}/support/new?from=trenova&instance=<id>`).
The operator renders each into the chart's `managed.supportUrl` and `managed.newSupportRequestUrl`
when it is an absolute `http`/`https` address (otherwise it leaves it out), the chart passes them
to the TMS as `managed.supportUrl` and `managed.newSupportRequestUrl`, and `managedconfig.Validate`
refuses at start anything but an absolute `http`/`https` URL with a host, no user information,
fragment or space, at most 2,048 characters. A changed address is a values change, so it reaches
the instance with a restart, like the rest of its configuration.

`GET /system/public-config` serves them, only with `managed.enabled`, as
`trenovaSupport: {requestsUrl, newRequestUrl}` (left out when both are empty). The addresses name
the console and the instance's ID in Trenova Cloud, nothing about the instance's data, so they sit
with the rest of the public config. The client (`publicConfigSchema.trenovaSupport`) drops a link
that is not an `http`/`https` URL, and the account menu (`TrenovaSupportItems`) shows, on a
managed instance only, "Contact Trenova support" (the form, in a new tab) and "Your support
requests" (the list, in a new tab), each only when its address arrived. The form's link adds
`page`: the matched route's pattern (`/shipments/$shipmentId`, `supportRequestPage`), never the
address itself, so no record's ID, name or email leaves the instance. The console writes it into
the request as a line staff read ("Opened from Trenova, on /shipments/$shipmentId"); see the
control plane's decision 0028.

## Support access

Trenova staff enter a customer's instance only under a grant an administrator of that
organization made (Admin > Trenova support access), and only with a single-use assertion the
control plane signs for that staff member. The instance never holds a staff password, the
control plane never calls the instance, and with `managed.enabled` off none of this is routed,
scheduled or read. The instance's side is `identity/app/supportaccessservice`; the rules a support
session is held to are in `identity/domain/supportaccess/policy.go`; who can reach what is in
[trust-boundaries.md](trust-boundaries.md).

### The key set (operator to instance)

The control plane delivers its support-signing public keys through the operator, beside the
feature list. The operator calls, with the bearer `managed.token`:

`PUT /managed/v1/support-access`, body at most 16 KiB, unknown fields refused:

```json
{
  "instanceId": "inst_01J…",
  "staffDomain": "trenova.app",
  "keys": [
    {"kid": "cp-2026-11", "kty": "OKP", "crv": "Ed25519", "alg": "EdDSA", "use": "sig",
     "x": "<32-byte public key, unpadded base64url>"}
  ]
}
```

- `instanceId` is the instance's id in the control plane, the audience every assertion must name.
  A shared server leaves it out (one is refused): each assertion names its tenant's instance, which
  the server checks against its tenants ([shared-tenancy.md](shared-tenancy.md#support-access-for-a-shared-tenant)).
- `staffDomain` is the one email domain grants may name; lower-cased, no subdomain is accepted.
- `keys` holds one to four Ed25519 JSON Web Keys with distinct `kid`s. To rotate, deliver the
  old and the new key together, start signing with the new one, then deliver the set without the
  old. A set with no key is refused rather than stored, so a broken delivery cannot replace
  working keys.

Answers: `204` stored (an unchanged set writes nothing), `400` with a problem detail naming what is
wrong, `401` for a wrong token, `500` with no detail. `GET /managed/v1/support-access` answers
`{"instanceId", "staffDomain", "keyIds": [...], "receivedAt"}` (never a key), with an empty
`keyIds` before the first delivery, so the operator can deliver only when what it holds differs.

`DELETE /managed/v1/support-access` removes the set (`204`, also when none is held); the tenant
agent sends it when the tenants document carries no `support`, so the server refuses every
assertion. The operator of a dedicated instance never sends it.

The set is one row, `managed_support_access_states` (`id = 'current'`, a global table; `instance_id`
null on a shared server), read through a 30-second one-entry cache like the feature list. Until a set arrives every entry is
refused and the grants page says support access is not available.

The control plane puts the set in the spec as `support` (`instanceId`, `staffDomain`, `keys`),
composed when it serves the spec like `account`, so a rotation changes the `ETag` without a new
revision. The operator reads `GET /managed/v1/support-access` on every tick while the instance is
ready and active, and sends `PUT` with exactly `{instanceId, staffDomain, keys}` when the instance
id, the domain or the set of key ids (in any order) differs. A spec without `support` sends
nothing and the instance keeps what it holds.

### The live grants (instance to operator to control plane)

`GET /managed/v1/support-access/grants`, bearer `managed.token`, exists only on the managed
listener, which a self-hosted instance never opens, and answers 409 on a shared server, which
reports each tenant's grants in that tenant's status instead. It answers every live grant on the instance,
across organizations, soonest to expire first, at most 100 (`supportaccessports.MaxReportedGrants`;
an organization holds at most 5):

```json
{"grants": [
  {"id": "sag_01J…", "organizationId": "org_01J…", "organizationName": "Acme Freight",
   "staffEmail": "dana@trenova.app", "scope": "read_only", "expiresAt": "2027-01-15T08:00:00Z"}
]}
```

- Live means not revoked, not past its expiry and its expiry not yet recorded. `scope` is
  `read_only` or `full`; `expiresAt` is RFC 3339 in UTC; `organizationName` is the name the
  organization has now.
- It never carries who made the grant, the reason they gave, or anything done under it: those
  stay in the organization's audit log.
- `200` with an empty list when none is live, `401` for a wrong token, `500` with no detail.
- One read under `dbscope.WithSystem` (listed in rlslint's allowlist): the grants through the
  partial index `idx_support_access_grants_unrecorded_expiry`, then the organizations' names in one
  batched lookup by id.

The operator reads it while building each status report (the report is built every tick and sent
when it changed or the minute is up, the way the held features are) and puts the list in the
report's `supportGrants`. A changed list is reported at once. When the TMS is not up or the read
fails the field is left out and the control plane keeps the last list; an empty list says no grant
is live. The control plane keeps it for its staff, who see which organizations granted them
access before they ask to enter.

### The assertion (control plane)

A compact JWS, EdDSA (Ed25519) only, protected header
`{"alg": "EdDSA", "kid": "<a delivered kid>", "typ": "support-assertion+jwt"}`, no `jwk`, `x5c`
or `crit` header. Claims, exactly these (an unknown claim is refused):

| Claim | Value |
|---|---|
| `iss` | `trenova-control-plane` |
| `aud` | the instance id, a single string (an array is refused) |
| `sub` | the staff member's verified Google Workspace address |
| `name` | the staff member's display name, at most 100 characters, no control or format characters |
| `org` | the organization's id on the instance (`org_` and 26 Crockford base32 characters) |
| `jti` | a fresh id, 16 to 64 of `[A-Za-z0-9_-]` |
| `iat` | issue time, Unix seconds |
| `exp` | at most 60 seconds after `iat` |
| `nbf` | optional; not before |

The instance accepts it only when the signature verifies with the key its `kid` names, the
audience is this instance, `iat` is no more than 15 seconds ahead of the instance's clock, `exp`
plus 15 seconds has not passed, `exp - iat` is at most 60, the `sub` is plain ASCII at exactly the
staff domain (compared after lower-casing ASCII letters only; any non-ASCII byte is refused, so no
look-alike passes), the `jti` has not been used (kept in `support_assertion_uses` until it could no
longer verify; the primary key decides two uses at once), and a grant for exactly that address
and organization is live at that moment.

The control plane mints an assertion only for a staff member signed in to the admin with Google
Workspace who confirmed a passkey within its recent-authentication window, and only for an
instance and organization they chose in "Enter instance". It records on its side, in its staff
audit log: the staff member, the instance, the organization, the `jti`, the time, and the reason
they gave; the instance records the entry in the customer's audit log under the same `jti`.

### Entering (browser)

The control plane renders, on its own origin, a page that posts the assertion and nothing else:

```html
<form method="post" action="https://<instance host>/api/v1/support/enter">
  <input type="hidden" name="assertion" value="<the JWS>">
  <noscript><button type="submit">Enter</button></noscript>
</form>
<script nonce="…">document.forms[0].submit()</script>
```

served with `Cache-Control: no-store` and `Referrer-Policy: no-referrer`. The assertion is never
put in a URL: a query string reaches logs and history, and the route refuses one.

`POST /api/v1/support/enter` (the edge passes only `/api` and `/graphql` to the API, so the route
lives there): `application/x-www-form-urlencoded`, exactly one field `assertion`, body at most
8 KiB, no query string. It is limited to 10 attempts a minute per address and 60 a minute for the
instance, and a limiter that cannot answer refuses. The browser provenance guard and the
cross-origin check both let this one post through (`middleware.SupportEntryPath`,
`middleware.ExceptSupportEntry`): it is cross-site by design, the browser sends it with the
admin's origin or `null`, and it carries no cookie either check protects. Any other method on
that path, and every other path, is still checked. A new early middleware that judges a
request by its origin must make the same exception, or every entry is refused with a bare 403
before the handler reads the assertion (found live on 2026-11-16).

- Accepted: `200`, the session cookie set as for any sign-in, and a page that moves the browser to
  `/` itself, so the strict same-site cookie rides on a same-site navigation.
- Refused, for any reason (a bad signature, wrong audience, expired, replayed, no live grant,
  revoked between mint and use, keys not delivered): `403` and the same page, which says only that
  support could not open a session. Over the limit `429` with `Retry-After`, an oversized body
  `413`, a server fault `503`; none says why. The reason goes to the instance's logs, and when the
  assertion verified it also goes to the named organization's security audit log
  (`support_access.entry_refused`, with the staff address and `jti`) and its sign-in activity. An
  assertion that did not verify reaches no customer's log.

### The session

A support session acts as the grant's support principal: a users row (`sup_…` id) the grant
creates, inactive, with an unusable password, an address under `support.trenova.invalid`, named
"Trenova support: <staff address>", which a database trigger keeps from ever being renamed,
activated or given a password. It is no person of the organization: people lists leave it out,
it holds no membership or role, and sign-in refuses it. Every foreign key that needs a user names
it, so nothing done under a grant is credited to a customer's person.

The session (`session.Support`, sealed) names the grant, the staff address and name and the
scope. It expires with the grant at the latest, and every request reads the grant: a revoked or
expired grant ends the session there and then. Revoking also deletes every session of the
principal at once. The permission engine sends a check for a support principal to its grant
(`supportaccessports.Principals`), never to role assignments: read-only allows `read` alone;
full allows an administrator's reach less the deny list, and an operation refused on a child
resource is taken off its parent too, so no parent grant reaches it. Every audit entry written in
the session is credited to principal type `support` with the grant as principal and carries
`supportGrantId` and `supportStaffEmail`, so the grants page lists a grant's actions by filtering
the audit log by principal.

A grant's end ends its sessions: revoking deletes them at once and every request reads the grant
(`requireLiveSupport`), so a revoked or expired grant refuses the next request on any process and
an open event stream ends at the session watch's next check; managed mode turned off answers every
support request with no grant at all. Support turned off by the control plane (the support keys
removed with `DELETE`) ends a live session the same way: `RequireLive` reads the server's key set
uncached with the grant on every request, so the next request on any process is refused. A
suspended tenant's requests are refused at the plan guard on the next request and its sessions
are deleted by `CloudTenantSessionsWorkflow`; a removed tenant's sessions are deleted by the
removal's first step. The hourly expiry sweep (`RecordExpiries`) records each grant on its own: a
grant it cannot record is logged and tried again on the next run while the others are recorded,
and the run fails (for the schedule's retry) only when every grant of the page failed. The sweep
only records an end that already took effect: sessions never outlive their grant. A grant whose organization's plan has ended is refused like
every other session by the plan guard. Extending a grant does not lengthen a session already open:
it ends at the end it was opened with, and staff enter again (the grants page says so beside an
extended grant). A presigned download link handed out during a session lives out its own short
expiry after the grant ends; that is accepted, since it names one file the session could already
read.

### What a session may do

- **Read-only writes nothing.** The plan's read-only guards also hold support: `ReadOnlyGuard`
  refuses every unsafe REST method from a read-only support session
  (`middleware.SupportWriteRefusal`) except the previews that write nothing, presence (which shows
  the customer who is looking) and the assistant's own routes; `ReadOnlyExtension` refuses every
  GraphQL mutation. Both answer 403 with "This Trenova support session is read-only…". Personal
  writes (settings, favourites, saved views) are refused, not ignored, so the interface never
  claims a change it did not make. The permission engine refuses the same writes again, since a
  read-only grant holds only `read`.
- **Every session**: no password change, no organization switch, no assistant schedule (a
  schedule keeps running after the grant ends); a full session keeps its own preferences, which
  go with the principal.
- **Where money is sent is never support's.** `supportaccess.PaymentDestinations` names the
  columns (invoice remittance instructions and factoring override, the billing control's default
  terms that print as statement and reminder remittance text, the organization's address printed
  as the remit-to on invoices, carrier remit-to fields, the factor a customer pays, a factoring
  company's address, remittance text and notice of assignment). Migration
  `20261231023400_support_access_completion.sql` puts `support_payment_destination_guard` on each
  table with the same columns; it reads the transaction's signed scope and refuses a support
  principal (`sup_` user) a change, or a new row carrying a value where `OnInsert`, with SQLSTATE
  `TRS01`, which `helpers.NormalizeDatabaseError` turns into one sentence. It holds for every
  writer: REST, GraphQL, bulk, imports, agent tools, an approved proposal and a workflow run in the
  session's scope. `TestPaymentDestinations_TheDatabaseGuardsExactlyWhatThePolicyNames` keeps the
  triggers and the list equal.
- **Nothing leaves the organization at support's hand, whatever the scope.** A support principal
  cannot send or schedule a message to anyone outside the organization (every email, test sends
  included; outbound EDI; detention notices; replies to inbound mail; factoring submissions and
  their transports), create or change a share, tracking or tender link or a scheduled report,
  or change text and recipients outside readers see (invoice memo and email snapshot, invoice
  line descriptions, a new invoice line whose words a person typed, order charge descriptions,
  the billing control's invoice footer, document template versions, customer
  status update recipients, the customer's email profile). `supportaccess.OutboundGuards` names
  the tables and columns; migration `20261231023401_support_outbound_guard.sql` puts
  `support_outbound_guard` on each (SQLSTATE `TRS02`, turned into one sentence by
  `helpers.NormalizeDatabaseError`), which holds for every writer.
  `TestOutbound_TheDatabaseGuardsExactlyWhatThePolicyNames` keeps the two equal. A typed
  invoice line is told apart by `invoice_lines.typed_description`, which the memo service sets
  (the only path that puts a caller's words on a line); lines rating, posting and adjustments
  derive leave it false, so support still re-rates and posts. A new path that writes a caller's
  text onto a line must set it. Above it,
  `emailservice.Send` refuses a send whose request or transaction scope names a support principal
  before anything is written, `customer_communication` is write-denied in the policy, and the
  agent runtime and proposal executor refuse support any tool whose egress reaches outsiders
  (`ToolPolicy.ReachesOutsiders`; withheld from its tool set too). The guard judges who
  initiated the write, from the transaction's scope: a send the system started (a customer's own
  scheduled statement run) or a person started earlier still goes out while a support session is
  open. A send that a support action sets off in the same session (an invoice support posts that
  would e-mail itself) is support's, and is refused with the sentence.
- **The assistant is the grant's choice.** A grant says whether staff may use the assistant
  (`allows_assistant`, off by default), because every answer is paid for by the organization's AI
  providers. When it does, the principal also holds `assistant` read and create, whatever its
  scope; tools still run under the grant, so a read-only session's assistant only reads. Every
  model call is admitted by `supportaccessports.ModelCalls` in the completion engine (all but
  embeddings): a call whose tenant, database scope or session names a support principal runs only
  while that principal's grant is live and allows the assistant, otherwise it is refused with a
  sentence and nothing is spent. Usage rows carry the support principal as their user.
- **Approvals.** The agent runtime and the proposal executor check a support principal through the
  engine like anyone, so it approves only what a full grant holds and the deny list leaves; it is
  never the person a tainted change is held for (`ErrTaintedNeedsPerson`) and has no own records
  for a self-scoped tool (withheld from its tool set and refused at dispatch).

### Notifications

The administrators holding `support_access` read are notified (`supportaccessnotice`, in-app) when
a grant is made (`support_access.granted`), extended (`support_access.extended`), first entered
(`support_access.first_entry`, high priority) and ended, by revocation or on its own
(`support_access.ended`). The administrator who made the change is not told of their own act.

### The grants page

Admin > Trenova support access lists live and past grants with who granted, extended or ended
each and why, whether staff may use the assistant, and "In your organization now" while a session
of the grant was used in the last five minutes (`SupportAccessService.ActiveNow`, at most a page of
session reads). An administrator who may grant may extend a live grant by 1, 4, 24 or 72 hours
counted from now (never shorter than it already lasts; `ck_support_access_grants_expiry` keeps the
end within 72 hours of the last extension) and anyone who may revoke may end it, whoever granted
it. Each grant opens its activity, paged from the audit log by principal id, and exports it as CSV
through the audit log's own export (`filters.principalId`), which needs `audit_log` export. The
page says what each scope allows and lists what support can never do from
`supportAccessSettings.limits`, the labels of the policy's own limits
(`supportaccess.Limits`), so the page cannot promise less or more than the engine and the
database enforce.

## Logs for Trenova support

The instance keeps 7 days of its own logs, separated by source (Loki and Alloy in the chart; the
operator collects and uploads them). Logs reach Trenova in two ways, and the TMS is the gate for
both: an administrator sends them, or staff pull them under a live support grant. The cross-layer
contract is in the control plane's `design-plans/managed-trenova/BUILD.md` ("Logs: design and
contract"); this section is the TMS's half. Code: `identity/domain/supportlog`,
`identity/app/supportlogservice`, `identity/adapters/postgres/supportlogrepository`,
`identity/app/supportaccessservice/logpull.go`, `api/managedserver_logs.go`.

### An administrator sends them

Admin > Trenova support access > "Send logs to Trenova support" (GraphQL `supportLogRequests`,
`requestSupportLogs`, both behind `organization:update` like Your plan, refused to an API key, an
agent and a support session). The person picks the last hour, 24 hours or 7 days; the TMS stores a
`support_log_requests` row with who asked (their user, and their name and email as they were then,
since the operator and Trenova staff are shown them), the window (`since`, `until`), the status
(`requested` → `collecting` → `sent` | `failed`), the bundle id the control plane gave, a failure
sentence and its times. The bundle is kept 30 days in the instance's own object storage.

- **Tenancy.** The row belongs to the asking organization (who asked is that organization's
  data); the page lists that organization's 10 newest. The bounds are the instance's, because the
  logs and the operator are shared: one request in flight for the whole instance (the partial
  unique index `uq_support_log_requests_in_flight` on a constant, which row-level security does not
  narrow, so it holds across organizations and against a race), and at most 5 requests in any 24
  hours. The validator's business rule counts both under a system scope (`InstanceLoad`) to give
  the person a sentence; the index is what decides a race. Because only one can be in flight, two
  requests at once cannot both pass the daily count either.
- **Timeouts.** `FailOverdueSupportLogRequestsWorkflow` (schedule
  `support-log-request-timeout-sweep`, every 5 minutes, managed instances only) fails a request
  still `requested` 30 minutes after it was made, or still `collecting` 2 hours after collecting
  started, with a sentence saying so.
- **Afterwards.** Every change publishes `support_log_request`/`created|updated` to the
  organization (the page refetches), the request and its end are written to the security audit log
  (`support_logs.requested`, `support_logs.sent`, `support_logs.failed`), and the person who asked
  is notified when the logs are sent or the request fails (`supportlognotice`).

### The listener contract

| Call | Body | Answer |
|---|---|---|
| `GET /managed/v1/log-requests` | | `{"requests": [{"id", "status", "bundleId"?, "since", "until", "requestedBy": {"email", "name"}, "requestedAt"}]}`, times RFC 3339 UTC: the requests still `requested` or `collecting`, oldest first, at most 10; `bundleId` only once the request has one |
| `PUT /managed/v1/log-requests/{id}` | `{"status": "collecting" \| "sent" \| "failed", "bundleId"?, "message"?}`, at most 16 KiB, no other field | 204; 400 for a malformed body, an unknown status, a bundle id that is not 1 to 100 of `[A-Za-z0-9_-]`, a message over 500 characters or `sent` with no bundle id held or given; 404 for an unknown id; 409 when the request already finished with another status, names another bundle, or changed meanwhile |
| `POST /managed/v1/support/log-pulls` | `{"email", "kind": "lines" \| "bundle"}`, at most 4 KiB, no other field | 200 and an audit entry when that address holds a live grant; 403 otherwise; 400 for a malformed body |

A `PUT` that repeats the request's current status (`collecting` twice, `sent` with the same
bundle) changes nothing and answers 204, so the operator may retry.

The list carries `collecting` requests too, so an operator that restarts in the middle of a
collection picks the request up again rather than leaving the administrator waiting for the
timeout. The operator keeps what it is working on in memory and skips those; after a restart it
finds the `collecting` request, registers it with the control plane again (idempotent on the
request id, so it gets the same bundle), reports `collecting` (a no-op) and collects again. The
2-hour timeout stays as the backstop for an operator that never comes back. A `failed` with no message is
stored as "The instance could not collect the logs." The message is shown to the administrators
as written, so the operator sends one plain sentence with no secret or internal detail in it.

### Staff pull them

Before the operator carries out a staff command (`collect_logs` or `collect_log_bundle`) it posts
the staff address to `/managed/v1/support/log-pulls`. The TMS normalizes the address exactly as an
entry does (`supportaccess.NormalizeStaffEmail`, against the delivered staff domain), lists the
live grants for it in every organization (one system-scoped read, `idx_support_access_grants_live_staff`),
and writes `support_access.logs_pulled` into each of those organizations' audit logs, credited to
the grant's support principal, saying Trenova support pulled log lines or a log bundle and who.
Refused for any reason (keys not delivered, an address outside the staff domain, no grant, an
expired or revoked one): `403` with the same empty problem, and the reason only in the
instance's logs. No secret is compared on this path: the bearer token is, in constant time, by
the listener. A customer's own request needs no grant.

## Rules that bite

- The listener is reachable only from the operator: the chart's NetworkPolicy admits
  `managed.from` alone and refuses to render managed mode without it. The token is compared in
  constant time; problem bodies never carry an internal error.
- The routes are not on the API router, so `authzlint` and `writecoverage` do not see them: the
  bearer token is their authorization. They write only what the control plane decides: the usage
  state, the feature list, the first administrator's invite, which grants nothing until someone holding the
  emailed token redeems it, the system email sender, which only ever carries system email, and the
  bounces and complaints for that email, which only ever add a suppression to the organization
  whose Trenova-sent message they name.
- The system email credential is a secret: it is sealed before it is stored, never returned,
  never logged (a failure logs the field and the rule, never the value), and the `PUT` body is
  never echoed in a problem or a log line.
- A relayed system email event names the customer's recipients: it is logged by ID, provider and
  reason only, never its payload, and a bad one is answered as handled so it never blocks the
  operator's queue. Only a failure worth retrying leaves it out of `handled`.
- A managed instance never sends through Trenova's sender untagged: `managed.instanceId` is
  required at start, and without it the sender refuses the message.
- The invite endpoint never sends email and never returns a token; it takes a hash and answers
  with the same hash or nothing.
- The support key route stores public keys only and answers key ids, never a key. It cannot grant
  access by itself: entry also needs a grant only the organization's administrator can make.
- The log pull check grants nothing: it answers whether an administrator already let that staff
  address in, and records the pull where they will see it. Every refusal looks the same.
- Logs hold every organization's activity on the instance, so sending them is limited to one
  request in flight and 5 a day for the whole instance, never per organization.
- A support session is never a person: a new resource that holds a secret or controls sign-in
  must be classified in `supportaccess/policy.go`, and
  `TestEverySensitiveResourceIsClassifiedForSupport` fails until it is. A new column that decides
  where money is sent goes in `PaymentDestinations` and the guard's trigger in the same change.
- Shipments are never hard-deleted, so counting them by `created_at` is a ledger. A meter whose
  source rows can be deleted needs its own append-only record before it is billed. Pages are read
  from `ai_usage_records`, which outlives the document; the extraction keeps its page count on
  `document_ai_extractions.page_count` so a deferred answer is attributed the same.
- A base flow that must choose a record an add-on owns never moves the kind to base: it marks the
  base set on the row, filters its picker and its write by that mark through one port, and lets
  the engine that uses the record refuse the rest
  ([standard rating templates](#a-base-flow-that-needs-an-add-ons-record-the-standard-rating-templates)).
  Data entered under the add-on is kept and comes back with it.

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
| The operator's listener | `internal/api/managedserver.go`, API process only | off | `managed.host:managed.port` (8090), bearer `managed.token` |
| The first administrator | `trenova db bootstrap --admin-setup`, `identity/app/firstadmininviteservice` | `password` | `link`, then an invite from the operator |
| Trenova support access | `identity/app/supportaccessservice`, `POST /api/v1/support/enter` | not routed, nothing read | grants, keys from the operator, staff entry ([Support access](#support-access)) |
| The system email sender | `workspace/app/systememailservice` (reads only with `managed.enabled`), chosen per message in `emailservice` | none: every email needs the organization's profile | the sender Trenova supplies, for system email with no profile |

Both providers ask one gate, `managedusageports.Gate` (`kernel/app/managedusageservice`), so every
existing hook enforces the control plane's answer with no new call site: shipments
(`shipmentrepository`, `Preflight` then `EnforceAll` in the insert transaction), document storage
(`CheckDocumentBytesLimit` and the document insert's `quotatx`), Document Intelligence pages
(`aidocumentservice.admit`, before a call or a deferred submission, asking for the document's page
count, or one page when it is not known; routing is not an extraction) and organizations
(`tenantbootstraprepository.Bootstrap`, the one place an organization is created, under the
provisioning lock).

Everything here is instance-wide. The feature gate and the usage state are one row each with no
tenant in them, and every reader counts across all organizations (`dbscope.WithSystem`): the
organizations on an instance share its features, its shipments, its pages and its storage, and
none of them has an allowance of its own.

## The state

`managed_usage_states` holds one row (`id = 'current'`, a global table): the open window, whether
overage is billed, and each app meter's quantity, allowance, hard cap and `stopped`, exactly as the
control plane last answered. The operator replaces it every relay (`PUT /managed/v1/state`). Every
API and worker process reads it through a 30-second in-process cache (otter, one entry); the
process that takes the write drops its entry at once, the others within the TTL.

`managedusage.Decide` is the whole rule:

- no state yet, or a meter the state does not name: not limited (a fresh instance works before
  its first relay, and the control plane decides which meters it sells);
- no open window: every named meter stops (unpaid, or the subscription ended);
- `stopped`: the meter stops;
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
  administrators (`organization:update`) of every organization get one notice
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
activity calls `managedfeaturesports.PlanEndReminder`, which locks and reads the one state row,
sends the reminder only when the end is less than three days away, not yet past and not yet
reminded, and stores `plan_end_reminded` in the same transaction. It is bounded by construction:
one row, one read, nothing while the plan is not ending.

Every notice goes through the same observer as the add-on notices (`workspace/app/plannotice`),
to each organization's administrators (`organization:update`, at most 25, `NotifyPermitted`),
keyed `managed-plan:plan-end` so a later notice about the end replaces an open earlier one, and
linking to `/admin/plan`. The receiver tells the observers after every stored change, and the
observer publishes `managed_plan`/`updated` to each organization, so an open Your plan page
refetches (`RESOURCE_QUERY_KEY_MAP.managed_plan` in `realtime-patching.ts`). The page shows the
end as a warning notice above the usage, with Keep plan opening `manageUrl` for whoever sees it.

`managed_feature_states` holds one row (`id = 'current'`, a global table) with the list, the plan
URL (`plan_url`, null until one arrives) and the add-on endings (`addon_endings`), with the list as
stored:
`managedfeatures.Normalize` drops base features (always on), duplicates and keys the catalogue
(`platformcatalog`) does not know, and sorts the rest, so the same set is always the same value.
An unknown key is ignored rather than refused because a newer control plane may sell a feature
this release does not have yet; refusing would stop the whole list from reaching the instance.
`managedfeaturesservice.SetFeatures` validates the request, compares the list, the plan URL and the
endings with the stored ones and writes nothing when all are equal; a change is stored with the
receive time, drops the cache and is logged with the count, the features added and removed, the
endings set and called off, and any ignored keys (at warn).

`managedfeaturesservice.Service` is the `FeatureGate` in managed mode, read through the same
30-second one-entry otter cache as the usage state:

- a base feature is always on, with no read;
- an add-on feature is on only when the stored list holds it;
- before any list arrives, only base features are on;
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
  transaction-scoped advisory lock (`keel.LockKey`, so two puts never both record a change), reads
  the held list, and lets `managedfeatures.Replace` decide. A list equal to the held one writes
  nothing. A different one replaces `managed_feature_states` and inserts one row into
  `managed_feature_changes` (a global table: when, the keys `added` and `removed`, and `starting`)
  in the same transaction, then deletes every row but the newest 200. The first list an instance
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
The readers are instance-wide (`dbscope.WithSystem`, listed in `rlslint`) because the whole
instance is one customer. `since` must be on the hour, within 35 days and not after the current
hour.

Storage is billed at its highest point in the cycle. An hourly level is enough for that: storage
moves with uploads and deletions over hours, not seconds, and the control plane takes the maximum
of the hourly levels it received in the window. A resent hour replaces that hour's level.

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

- `account` (`planUrl`, `finalBackupDays`, `endsAt`), composed at serve time ([The feature
  list](#the-feature-list)).
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
- the suppression list and bounce handling as for any message: bounces and complaints arrive at
  `POST /api/v1/webhooks/email/system/{resend|postmark}/`, verified with the sender's
  `webhookSecret`, and the message (found by provider and provider message ID among the ones
  Trenova sent, across tenants) names the organization whose suppression list it updates. A
  webhook for a message this instance did not send is answered 204 and ignored.

It sends as `<organization name> via <fromName>` from Trenova's `fromAddress`, with no reply-to
(an organization has no reply address of its own outside a profile). A provider failure is kept
in the email log as "Trenova's system sender could not deliver this email", with its retry class;
the provider's own words are logged on the instance only, with the API key removed.

**The sender on the instance.** `system_email_senders` holds one row (`id = 'current'`, a global
table): provider, from address and name, the API key and webhook secret sealed by the TMS's own
encryption (`kernelports.SecretCipher`, purpose `system_email_credential`), the fingerprint and
when it arrived. `systememailservice` reads it through a 30-second one-entry otter cache; the
process that takes a write drops its entry at once, the others within the TTL. A row that does
not open (the encryption key that sealed it is gone) or does not match its fingerprint counts as
unset, so the operator's next tick sends it again. No API or page returns it; the email settings
page tells an administrator "System email is sent by Trenova on your behalf", with the from
address, when the instance holds a sender and the organization has no General profile (the case
in which it is used). Self-hosted installs never read the table.

### The contract

The operator delivers the sender on the managed listener:

| Call | Body | Answer |
|---|---|---|
| `PUT /managed/v1/system-email` | `{"provider": "Resend" \| "Postmark", "apiKey", "fromAddress", "fromName", "webhookSecret"}`, at most 8 KiB, no other field | 204; 400 for a malformed body or a value of the wrong shape, naming the field and never the value; 413 past 8 KiB |
| `GET /managed/v1/system-email` | | `{"configured": true, "provider", "fromAddress", "fingerprint"}`, or `{"configured": false}` |
| `DELETE /managed/v1/system-email` | | 204, also when there was none |

Shapes: `apiKey` 20 to 512 printable ASCII characters with no space; `webhookSecret` empty or up
to 512 of the same (Resend: the Svix signing secret `whsec_…`; Postmark: `user:password` of the
webhook's basic authentication); `fromAddress` a bare address of at most 254 characters on a
domain Trenova verified with the provider; `fromName` 1 to 40 printable characters with no
leading or trailing space.

The **fingerprint** is the lowercase hex SHA-256 of the UTF-8 text `trenova.system-email.v1`,
`provider`, `fromAddress`, `fromName`, `apiKey` and `webhookSecret` (empty when absent), joined by
a single line feed, in that order. `systememail.Fingerprint` (TMS) and
`internal/systememail.Fingerprint` (operator) compute it; both test the vectors
`Postmark / no-reply@mail.trenova.app / Trenova / pm-0123456789abcdef0123456789 / (none)` →
`97bc40de…d604d01f` and `Resend / no-reply@mail.trenova.app / Trenova /
re_0123456789abcdef0123456789 / whsec_abcdef` → `1ca13b2b…0c9736fc`. A `PUT` whose fingerprint
equals the stored one writes nothing.

From the control plane (`api/instance.yaml`):

- the spec carries `systemEmail: {"fingerprint": "<hex>"}` while Trenova supplies a sender, and
  nothing when it does not;
- `GET /v1/secrets` carries `systemEmail: {"provider", "apiKey", "fromAddress", "fromName",
  "webhookSecret"?}` while the spec names one. It is escrowed like the other secrets (sealed by
  the keyring for that instance) but, unlike them, it may change.

**Rotation**: the control plane stores the new credential, then publishes a spec revision with
its fingerprint. The operator fetches it, puts it, and the instance replaces the old row in one
write; until then it keeps sending with the old credential, and a message queued before the swap
is sent with whichever credential is stored when its workflow runs. The old credential must stay
valid at the provider until the instance reports the new fingerprint (the operator's `GET` shows
it within a tick). **Removal**: the control plane drops `systemEmail` from the spec; the operator
calls `DELETE`, and system email with no profile is refused as on a self-hosted install. A queued
message whose sender was removed fails without retrying.

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
- `staffDomain` is the one email domain grants may name; lower-cased, no subdomain is accepted.
- `keys` holds one to four Ed25519 JSON Web Keys with distinct `kid`s. To rotate, deliver the
  old and the new key together, start signing with the new one, then deliver the set without the
  old. A set with no key is refused rather than stored, so a broken delivery cannot replace
  working keys.

Answers: `204` stored (an unchanged set writes nothing), `400` with a problem detail naming what is
wrong, `401` for a wrong token, `500` with no detail. `GET /managed/v1/support-access` answers
`{"instanceId", "staffDomain", "keyIds": [...], "receivedAt"}` (never a key), with an empty
`keyIds` before the first delivery, so the operator can deliver only when what it holds differs.

The set is one row, `managed_support_access_states` (`id = 'current'`, a global table), read
through a 30-second one-entry cache like the feature list. Until a set arrives every entry is
refused and the grants page says support access is not available.

The control plane puts the set in the spec as `support` (`instanceId`, `staffDomain`, `keys`),
composed when it serves the spec like `account`, so a rotation changes the `ETag` without a new
revision. The operator reads `GET /managed/v1/support-access` on every tick while the instance is
ready and active, and sends `PUT` with exactly `{instanceId, staffDomain, keys}` when the instance
id, the domain or the set of key ids (in any order) differs. A spec without `support` sends
nothing and the instance keeps what it holds.

### The live grants (instance to operator to control plane)

`GET /managed/v1/support-access/grants`, bearer `managed.token`, exists only on the managed
listener, which a self-hosted instance never opens. It answers every live grant on the instance,
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
instance, and a limiter that cannot answer refuses. The browser provenance guard lets this one
path through (`middleware.SupportEntryPath`): the post is cross-site by design and carries no
cookie the guard protects.

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
the audit log by principal. The administrators who hold `support_access` read are notified when a
session starts.

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
  emailed token redeems it, and the system email sender, which only ever carries system email.
- The system email credential is a secret: it is sealed before it is stored, never returned,
  never logged (a failure logs the field and the rule, never the value), and the `PUT` body is
  never echoed in a problem or a log line.
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
  `TestEverySensitiveResourceIsClassifiedForSupport` fails until it is.
- Shipments are never hard-deleted, so counting them by `created_at` is a ledger. A meter whose
  source rows can be deleted needs its own append-only record before it is billed. Pages are read
  from `ai_usage_records`, which outlives the document; the extraction keeps its page count on
  `document_ai_extractions.page_count` so a deferred answer is attributed the same.
- A base flow that must choose a record an add-on owns never moves the kind to base: it marks the
  base set on the row, filters its picker and its write by that mark through one port, and lets
  the engine that uses the record refuse the rest
  ([standard rating templates](#a-base-flow-that-needs-an-add-ons-record-the-standard-rating-templates)).
  Data entered under the add-on is kept and comes back with it.

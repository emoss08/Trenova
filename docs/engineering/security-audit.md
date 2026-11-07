# Security audit trail

What Trenova records about sign-ins, administrative changes, document access and
refused cross-tenant requests, where each record lives, how it survives an outage,
and how long it is kept. This is the trail an organization's administrators and
an incident responder read. The AI audit chain, which covers agent runs, is a
separate system: see [ai-audit-trail.md](ai-audit-trail.md).

## What is recorded

| Event | Table | Written by | Critical |
|---|---|---|---|
| Password sign-in, success or failure | `auth_events` | `authservice.Login` | — |
| SSO callback, success or failure | `auth_events` | `authservice.HandleSSOCallback` | — |
| Sign-out | `auth_events` | `authservice.Logout` | — |
| Password reset requested or redeemed | `auth_events` | `passwordresetservice` | — |
| Account invitation redeemed (a new user chose their first password) | `auth_events` | `passwordresetservice` | — |
| Sign-in throttled (account or IP failure limit) | `auth_events` | `authservice.Login` | — |
| Cloud signup requested, rejected, verified, provisioned (Cloud edition) | `auth_events` | `internal/cloud/signup/cloudsignupservice` | — |
| Cloud signup owner granted Organization Administrator (Cloud edition) | `audit_entries` | `internal/cloud/signup/cloudsignupservice` | yes |
| Instance bootstrap (`trenova db bootstrap`) created the first administrator, the Organization Administrator role and its grant, and the system account | `audit_entries` | `instancebootstrapservice` | yes |
| API key created, updated, rotated, revoked | `audit_entries` | `apikeyservice` | yes |
| Role created or updated, assigned or unassigned, inheritance, separation of duty constraints, resource permissions | `audit_entries` | `roleservice` | yes |
| Identity providers, SCIM directories, tokens and group mappings, access policies | `audit_entries` | `iamservice` | yes |
| Organization memberships replaced | `audit_entries` | `userservice` | yes |
| User created by an administrator, with the roles granted and whether the invitation went out | `audit_entries` | `userservice` | yes |
| User status, lock, email address or forced password change changed by an administrator | `audit_entries` | `userservice` | yes |
| Password changed, reset link or account invitation sent by an administrator | `audit_entries` | `userservice`, `passwordresetservice` | yes |
| Two-factor code at sign-in, success or failure | `auth_events` | `authservice.VerifyMFAChallenge` | — |
| Authenticator app enrolled, confirmed or turned off; recovery codes regenerated; two-factor reset by an administrator | `audit_entries` | `mfaservice` | yes |
| Trenova support access allowed, changed or revoked (Cloud edition) | `audit_entries` | `internal/cloud/supportaccess/supportaccessservice` | yes |
| Trenova support session started, elevated to write, returned to read-only, ended (Cloud edition) | `audit_entries` | `internal/cloud/supportaccess/supportaccessservice` | yes |
| Request refused inside a Trenova support session (Cloud edition) | `audit_entries` | `internal/cloud/supportaccess/supportaccessservice` | yes |
| Platform staff member added or removed (`trenova cloud staff`, Cloud edition) | `audit_entries` | `internal/cloud/supportaccess/supportaccessservice.StaffManager` | yes |
| Agent tool rule held lower or returned to its declared rule, with the reason and the agents whose answer moved | `audit_entries` | `agenttoolruleservice` | yes |
| Money limits for agents changed in AI Control (one change's ceiling, a run's and a day's count and total) | `audit_entries` | `agentcontrolservice` | yes |
| Agent memory written after outside content reviewed (directly, or by approving it as a suggestion), so it no longer holds the writes of the turns that read it | `audit_entries` | `agentmemoryservice` | yes |
| Document downloaded or viewed (content, presigned URL or page image) | `audit_entries` | `documentservice` | — |
| Document attached to an outbound email | `audit_entries` | `documentmailservice` | — |
| Cross-tenant request refused | `audit_entries` | `middleware.TenantBoundaryMiddleware` | yes |
| File refused by the malware scanner (upload, inbound email attachment, capture page, inbound EDI file, fuel card export) | `audit_entries` | `contentscanservice` | yes |

Every row carries the request ID, client IP and user agent of the request that
caused it. The request ID is the `X-Request-ID` the API returns, so a support
ticket quoting it finds the row. Each `audit_entries` row also carries an `action`
(`shipment.update`, `api_key.rotated`, `organization.cross_tenant_refused`), a
`category` (`User`, `System` or `Security`), a `retention_class`, the `channel` it
came through, the OpenTelemetry `trace_id`, the `ai_run_id` of the agent run that
caused it, an `entity_ref` (the record's number or name when it was written) and an
`actor_name` the insert trigger snapshots from the user, API key or agent, so the row
still names its actor after a rename or deletion.

## Request metadata

`middleware.NewRequestMetaMiddleware` runs right after `requestid.New()` and puts a
`requestmeta.Meta` (request ID, `ClientIP()`, user agent, truncated to the audit
columns) on both the request context and the gin context. `LogAction` and `LogActions`
take the caller's `ctx` and fill `ip_address`, `user_agent` and `correlation_id` (the
request ID) from it, `trace_id` from the active span, and `channel` and `ai_run_id`
from `pkg/auditctx`. Nothing passes request metadata by hand. A context with no
metadata (a Temporal activity, a CLI command) leaves those fields empty rather than
failing.

`channel` is `web`, `api`, `portal`, `agent_conversation`, `agent_run`, `support`, `scim`,
`job`, `cli` or `system` (`pkg/auditctx`). `auditctx.WithChannel` sets it where the principal
alone does not say it (the SCIM handler); otherwise it follows the principal: an API key is
`api`, an agent `agent_run`, the system principal `system`, a person `web`. The agent runtime puts the run on the context for every step
(`auditctx.WithAIRun`), so every write a run makes links back to it.

`ClientIP()` is only as trustworthy as `server.trustedProxies`. Behind a load
balancer, list it there, or every row records the balancer's address.

## Authentication events

`iamports.AuthEventRecorder` (`autheventservice`) writes one `auth_events` row per
attempt. The caller fills an `AuthEventRecord` (provider, outcome, user,
organization, assurance levels, an error code) and the recorder adds the request
metadata. `provider` names the method: `password`, `sso.<provider>`,
`session.logout`, `mfa` (the second step of a sign-in), `mfa.step_up` (a signed-in
person's second factor for a sensitive change), `password_reset.request`, `password_reset.confirm`,
`invite.accept` (a new user redeeming the invitation link and choosing their first password), `login_throttled`,
and for cloud signup `signup_requested`, `signup_rejected`, `signup_verified`,
`signup_provisioned`. Signup rows carry no user until provisioning; a rejection names
its reason in `error_code` (`honeypot`, `turnstile_rejected`, `turnstile_unavailable`,
`invalid_input`, `existing_account`, `send_limit_reached`, `invalid_token`,
`expired_token`, `signups_paused`, `email_in_use`, `provisioning_failed`).
`login_throttled` rows name the scope in `error_code` (`login_throttled_account`,
`login_throttled_ip`). Every attempt is counted before its password is checked, in one
Redis script that refuses an account under lock or an address past its threshold and
otherwise increments both counters and sets the lock on the attempt that reaches the
threshold, so concurrent guesses cannot all slip in under it. A successful sign-in clears
the account's counter and returns its attempt to the address. An unknown address still
costs a bcrypt comparison against a dummy hash, and a locked or inactive account answers
"invalid email or password" until the right password is given, so neither timing nor the
message tells an attacker which addresses exist or what state they are in. The throttle fails open: when Redis cannot be reached the attempt
is allowed and a warning is logged, so a cache outage never locks every user out; the
password check, the per-account bcrypt cost and the signup Turnstile check still apply.

- **Outcome.** `success`; `denied` when policy refused a known account (SSO
  enforced, account locked, no access to the organization, reset rate limit);
  `failed` otherwise. `error_code` says which check, for example
  `unknown_account`, `invalid_credentials`, `sso_nonce_mismatch`, `invalid_token`.
- **Tenant.** The organization being signed into when it is known, otherwise the
  user's current organization, so that organization's administrators see failed
  attempts against their accounts. An address that matches no account has no
  tenant; the row is written under the system scope and only operators can read it.
  The submitted email address is never stored.
- **Never blocks sign-in.** The write is detached from the request's cancellation,
  bounded by a short timeout, and a failure is logged at error with every field of
  the event and counted in `trenova_audit_security_event_total{kind="auth_event",status="failure"}`.
- **No oracle.** Recording happens after the response is decided. Password reset
  still answers an unknown, inactive and rate-limited address identically; only the
  row differs.

A password sign-in for an account with an active authenticator app records its
`password` row with `mfa_state = challenged` and issues no session; the code step
records an `mfa` row at assurance level 2 on success, or `mfa_state = rejected` with
`error_code` `mfa_invalid_code`, `mfa_attempts_exhausted` (five wrong codes end the
challenge) or `mfa_challenge_invalid` (unknown or expired challenge). Secrets, codes and
recovery codes never reach a row. Recovery codes are stored as HMAC-SHA256 digests keyed by
`security.mfa.recoveryCodePepper` (required in production; never change it while codes are
issued under it), with `hash_version` 2; codes issued before that are unsalted SHA-256
(`hash_version` 1), keep working until their owner generates new ones, and make the two-factor
settings ask for that (`recoveryCodesNeedReplacing`).

An SSO identity is linked to a Trenova user once, on its first sign-in, and only through an
`email` claim the provider vouches for: `email_verified` true, or for Microsoft Entra the
`xms_edov` (domain owner verified) optional claim. `preferred_username` and `upn` are never
used to find a user. Every later sign-in resolves the user by the link's issuer and subject,
whatever the email has become. Entra tenants that link new people therefore need the `email`
and `xms_edov` optional claims on the app registration.

An SSO sign-in counts as two-factor only when its identity provider has "Trust federated
MFA" on and the ID token names an exact multi-factor `amr` value (`mfa`, `otp`, `hwk`, `swk`,
`fpt`, `face`, `iris`, `retina`, `sc`, `fido`, `ngcmfa`) or an `acr` value the administrator
listed on the provider (`federatedMfaAcrValues`); no substring of an `acr` counts. The SSO row
records `mfa_state` `federated` or `federated_not_trusted`. When the provider's MFA is not
trusted and the person has an authenticator app, the callback issues no session: it sends the
browser back to the sign-in page with a challenge in the URL fragment, and the `mfa` step
finishes the sign-in with the SSO provider, subject and federation level. Without an app the
session stays at assurance level 1, so an access policy with `mfaRequired` refuses it (see
"Access policies" in [permissions.md](permissions.md)).

Turning the authenticator off, generating new recovery codes and re-authenticating with
password and code are step-ups of a signed-in person, recorded as `mfa.step_up` rows
(`step_up_rejected`, or `step_up_throttled` when refused or when the attempt reached the
limit). Each is counted against the same per-account and per-address throttle as sign-in,
so a stolen session cannot guess codes faster than a sign-in could.

API key authentication is not recorded per request; key creation, rotation and
revocation are, and `api_keys.last_used_*` tracks use.

## Administrative changes

Services that change who can do what take a `permissionports.SecurityAuditor` and call
`RecordChange` after the change commits, with the resource, operation, actor,
tenant, before and after state, a comment and metadata. The auditor:

- writes a **critical** entry with category `User` and the request metadata;
- serializes any state, including slices, as a JSON document (a list becomes
  `{"value": [...]}`), and computes the diff from those documents, so a state it
  cannot diff never drops the entry;
- fills a missing tenant or actor from the request's database scope, and falls back
  to the system actor;
- never fails the change. If the entry cannot be stored, it logs the change in full
  at error and counts `trenova_audit_security_event_total{kind="security_change",status="failure"}`.
- marks the entry `Security` with `SecurityChange.Action` as its action (`role.assigned`,
  `data_retention.updated`), which keeps it for the security retention class, hides it
  from readers without `security_audit_log:read`, and lets the alert policies match it.

Secrets never reach an entry: OIDC client secrets, SCIM token hashes and API key
hashes are `json:"-"`, API keys are recorded through their response shape, and
tokens are identified by prefix only.

Adding a security-relevant mutation means calling the service's `recordChange`
helper after the write succeeds, never before, and never for a refused change. A
method that can run inside a caller's transaction records through
`kernelports.AfterCommit`, so the entry is written only once that transaction
commits: `roleservice.AssignRole` does this, which is what lets user creation grant
its roles in the same transaction as the user and leave no "Role assigned" entry when
a later grant is refused and the whole creation rolls back.

### When access ends

Losing access ends it everywhere it is held, and each step is recorded:

- Deactivating or locking a user, one at a time or in bulk, ends every session they
  have open and revokes their Capture devices in every organization, through the
  `userAccessRevokers` fx group (`tenantports.UserAccessRevoker`). Saving the same
  deactivation again ends any session or device a failed first attempt left behind. A
  device's refresh also checks its person's standing, so a revocation that failed stops
  at the next refresh, within the fifteen-minute life of its access token.
- Revoking a driver's portal access ends the portal user's sessions.
- A SCIM directory that deactivates or deletes a person ends their access in the directory's
  organization through the same exit path as removing a membership, or deactivates the account
  when that organization is the only one they belong to; it never touches access elsewhere
  ([scim.md](scim.md#users)). Each change is a `provisioning_audit_records` row and a security
  entry attributed to the system actor with the directory and token in its metadata.
- An API key never grants more than its creator holds. Creating or editing a key with a
  resource, operation or scope the person saving it does not hold is refused, and at
  request time the key's grants are intersected with what its creator holds now: a
  creator who is demoted narrows their keys, and one who is deactivated, locked or no
  longer a member of the organization leaves their keys granting nothing.
- Removing a user from an organization deletes their role assignments there, one
  critical `Role unassigned` entry each, and ends their sessions bound to it. The
  user's current organization moves to their default one, and a user cannot be left
  with no organization at all (deactivate them instead).
- The `orphaned-organization-access-sweep` schedule (`identity/jobs/accessjobs`, hourly)
  finds role assignments whose holder has no live membership in that organization (and
  it is not their current one), revokes them in that organization's own scope with the
  same entries, attributed to the system actor, and ends the matching sessions. It
  catches a removal whose revocation failed partway, and grants left behind by an
  expired membership.
- A membership with an expiry stops granting at that moment, in the user's current
  organization too: the permission engine caps a cached set at the expiry and computes
  nothing from a membership that has ended. A person with no membership row is
  unaffected.

- A session issued by sign-in (`Session.IssuedBySignIn`, set only by
  `authservice.createSession`) is checked again each time it is touched, at most every
  five minutes: if the user is no longer a member of the session's organization (and it
  is not their current one) or can no longer sign in, the session is deleted and the
  request is refused. This catches whatever a failed revocation above left open. A
  database error leaves the session alone and checks again on its next request.

Support sessions draw their permissions from the support access grant, not from role
assignments, and are not issued by sign-in, so none of this touches them; their own
lifecycle (grant revoked or expired, staff removed) ends them.

## Trenova support sessions

Trenova Cloud staff enter a customer organization only through a grant its own
administrators made (Admin > Support access). Everything about that access is written
to the customer's own `audit_entries`, under the organization resource, with
`metadata.supportAccess` naming the event, so the organization's administrators read it
in their audit log:

- grant created, changed or revoked, by the administrator who did it;
- session started (staff member, reason, ticket), elevated to write (reason, ticket),
  returned to read-only and ended (with the reason it ended: staff exit, expiry, grant
  revoked or expired, staff removed, replaced by a newer session, sign-out, two-factor
  sign-in lapsed);
- every request the session was refused (read-only or deny list), with the method and
  path or GraphQL field.

Writes made in a write-elevated session are ordinary writes by the session's
principal, a synthetic, inactive user named "Trenova Support (staff name)", so each
domain audit entry names support as the actor. Admins who can update the organization
are notified when a session starts. Platform staff membership changes are recorded by
`trenova cloud staff add|remove` against the staff member's own organization.

## Document access

`documentservice` records every download and view, whether it streams the content
or hands out a presigned URL, as a non-critical entry with operation `read` and
`metadata.access` (`download`/`view`) and `metadata.channel`
(`content`/`presigned_url`/`page_image`). Because it is in the service, invoice share links, the
driver portal and attachment reads are covered as well as the documents API.
A page drawn as an image to read (`GET /documents/:id/pages/:page/image/`, which the
extraction card shows) is a view on the `page_image` channel, recorded once per page the
browser fetches; a repeat answered `304 Not Modified` from its ETag delivers nothing and is
not recorded. Preview thumbnails are not recorded. A read with no user (a share link) is
attributed to the system principal; the request metadata identifies the requester.

### Documents sent by email

A feature that mails stored documents asks the documents module for them through
`documentports.DocumentMailAttachments` (`documents/app/documentmailservice`), never by
reading the files itself:

- `Resolve` turns document ids into `emailports.EmailAttachment`s that carry the storage
  object key, not the bytes. It reads the documents under the caller's tenant, refuses a
  missing one or one that cannot be sent (archived, expired, rejected, or owned by an
  assistant conversation) with a field error on `Field[i]`, checks that the person holds
  `document:read` and read on each document's owning resource (`document.OwnerResource`),
  and refuses more than `MaxMailAttachments` files or more than
  `emailports.MaxAttachmentBytes` in all. That budget is what fits in the smallest
  provider's message (Postmark's 10 MiB, base64-encoded, less `BodyAllowanceBytes` for the
  body), defined once beside the provider ceilings in `emailports/limits.go`; billing's
  invoice delivery reads the same constants. `System: true` is for Nova and other
  unattended writers: it skips the person's access check, so what it resolves must be
  resolved again, with the person, before anything is sent. With `Recipients` set,
  `Resolve` writes one `read` entry per document with `metadata.access` `email`,
  `metadata.channel` `attachment` and the recipients.
- `Mailable` lists the sendable current documents on a set of records (`resourceType`,
  `resourceId`), skipping the records the person cannot read and returning nothing without
  `document:read`. It is what a picker of "files you could attach" reads.

Documents are stored as sealed streaming blobs bound to their tenant and storage path, so an
object key alone would mail ciphertext. Each attachment the capability returns is
`Seal: Document`; the email service keeps the seal on `email_message_attachments.seal` and,
in the send activity, decrypts a sealed object with the document AAD of the message's own
tenant. A key from another tenant fails to decrypt, and an unsealing failure fails the
send without a retry. A file rendered for the occasion (a detention record, a report) is
stored first with `documentports.DocumentStore.Store`, which writes it synchronously as a
document on its record, so it is on file and its id can be attached at once.

## Writes an agent made

A write made in a conversation is the person's: their row, principal
`session_user`. A write an unattended agent run made is principal `agent` with the
agent definition's id as `principal_id`, `user_id` the instance's system user, and a
description ending "(Ran by Dispatch Agent)", the name `auditservice` reads for the
definition. `chk_audit_entries_principal_consistency` allows a user on an `agent`
row, never an API key, and never the user as the row's own principal; a `system`
row still names no user. Rows written before
`20261231007250_audit_agent_system_user` name the generic `agent` principal and no
user. See [agent-runtime.md](agent-runtime.md#who-a-run-acts-as).

## Files refused by the malware scanner

Every file Trenova receives passes `contentscanports.Gate` before it is stored or parsed
(`documentservice`, `documentuploadservice`, `captureintake`, `ediinboundservice`,
`rateimportservice`, the fuel card file connector, and the profile picture and logo uploads). When ClamAV flags one,
`contentscanservice` writes a critical `Security` entry with action
`document.malware_refused` against resource `document` with operation `reject`, the file's SHA-256 as `resource_id` (the file is never stored, so it has no ID),
and the file name, source, signature, digest and size in `current_state`. A new way of
receiving a file calls `contentscanports.Check` on its bytes before anything opens them and
answers a refusal with `contentscanports.RefusalError`. Configuration and operation are in
"Malware scanning" in [docs/self-hosting/README.md](../self-hosting/README.md).

## Refused cross-tenant requests

Three places refuse a request that reaches for another tenant, and each marks the
attempt with `tenantboundary.Report`:

| Source | Where | Detail |
|---|---|---|
| `request_body` | `authctx.BindJSON` | the field and the foreign organization or business unit |
| `path` | organization and IAM routes | the organization ID in the path |
| `database_policy` | `helpers.ErrorHandler` | PostgreSQL rejected a write under row-level security |

`middleware.TenantBoundaryMiddleware.Track` gives every request a tracker and,
after the handler returns, writes each marked attempt as a critical entry against
the caller's organization (a `Security` entry with action
`organization.cross_tenant_refused`, resource `organization`, the attempted organization as
`resource_id`, the operation from the HTTP method) with the route and response
status. A request with no session is only logged. A row-level security rejection
is answered 403 with a generic message instead of surfacing as a server error.

To refuse another cross-tenant shape, call `tenantboundary.Report(ctx, …)` where
the request is refused; the middleware does the rest.

## How an entry is written

`auditservice.LogAction(ctx, params, opts...)` builds the entry, applies the field policy
(below), normalizes it (action, retention class, channel, record reference, truncation)
and inserts it with `auditrepository.Insert`:

- inside the caller's transaction when `ctx` carries one for the same tenant, so the entry
  commits or rolls back with the change it describes;
- otherwise in its own write under the context's tenant scope, or a system scope for an
  entry about another tenant.

The write is detached from the request's cancellation (`context.WithoutCancel`, 10 second
timeout), so a client that hangs up after the change committed does not lose the entry.
Realtime publication and alert evaluation run through `kernelports.AfterCommit`, so a
rolled-back change raises nothing. An update that carries a diff stores only the diff;
creates keep `current_state` and deletes `previous_state`. `LogActions` inserts a batch in
one statement (bulk status changes).

There is no buffer and no dead-letter queue. A failed insert returns an error to the caller
and the service logs the entry in full at error, so the log pipeline is the last copy; the
security auditor and the auth event recorder do the same.

## What an entry may hold

`auditservice.Redactor` applies a field policy to the states, the diff and the metadata, by
field name at any depth:

| Field | Stored as |
|---|---|
| A secret: a name containing `password`/`passphrase`, or ending in `secret`, `token`, `api key`, `private key`, `signing key`, `access key`, `encryption key`, `recovery code`, `key hash`, `token hash` | left out |
| Tax ID, EIN, SSN, account, routing and card numbers, IBAN | `[REDACTED] …1234 #a1b2c3d4`: the last four and an HMAC-SHA256 fingerprint keyed by `audit.fingerprintPepper` |
| A private key, JWT or URL with a password inside the comment | replaced with `[REDACTED]` |

A redacted entry has `sensitive_data` set, and the client marks those values **Sensitive**.
The fingerprint lets an investigator see that two entries changed the same number without
storing it. Without a pepper only the last four is kept. Anything else, including record IDs,
is stored as written: the policy is by name, not by guessing at values.

## Retention classes

Each entry's `retention_class` is set when it is written (`audit.ClassifyRetention`):

| Class | What | Floor | Default |
|---|---|---|---|
| `Access` | reads, downloads, exports | 90 days | 90 days |
| `Operational` | every other change | 730 days | 730 days |
| `Financial` | changes to billing, accounting, payroll, fuel and tax, settlements, detention and rate confirmations (`audit.IsFinancialResource`, from the permission catalog's categories) | 2,555 days | 2,555 days |
| `Security` | `Security` entries | 730 days | 730 days |

An organization chooses each on Data retention (`data_retention.audit_*_retention_days`),
between the floor and 3,650 days. The floor is enforced three times: the domain validation,
a `CHECK` on `data_retention`, and `audit_retention_floor_days()` inside the delete trigger.

## Legal holds

`audit_legal_holds` stops the sweep deleting an organization's entries: all of them, one
resource's, or one record's (`auditlegalholdservice`, permission `audit_legal_hold`). A hold
is released, never deleted, and a released hold cannot change
(`trg_audit_legal_holds_guard`). Placing and releasing are `Security` entries. The delete
trigger checks for a covering active hold on every row, so a sweep cannot remove a held
entry even if the service has a bug, and `drop_expired_partitions` skips any month that still
holds a held entry.

## Integrity seals

Every hour (`AuditSealWorkflow`, minute 12) `auditintegrity.Sealer` seals each tenant's
entries up to the last whole hour at least 10 minutes old, looking back 72 hours and at most
24 hours per seal. A seal (`audit_seals`, append-only) records, per retention class, the row
count and a SHA-256 digest of the entries computed in SQL, and an HMAC-SHA256 chained to the
previous seal (`trenova.audit-seal/v1`), signed with the active `aiAudit.chain` key.
`audit_chain_heads` holds where each tenant's chain ends.

`AuditVerifyWorkflow` (daily, 04:27) re-verifies every seal since the last run, then 200 older
ones per tenant on a rolling cursor, so the whole chain is re-read over time. It reports
`Verified`, `Broken` (an entry changed, an entry added to a sealed hour, or an altered seal,
with the sequence where it broke) or `KeyMissing` (a seal signed with a key no longer
configured, which is not tampering). The audit log page shows the status to readers with
`security_audit_log:read`. Retention deletes past a sealed hour are expected: seals of hours
whose entries have all aged out are removed with them.

## Exports

`auditexportservice` writes a date range (at most 400 days) of the trail to CSV or JSONL on
Temporal (`AuditExportWorkflow`), streamed to object storage at up to two million rows.
Requesting needs `audit_log:export`; security entries need `security_audit_log:read` as
well, and the fields the requester could not see on the page are withheld (the visible
groups are stored with the export's filters). Only the requester can download the file, until
`audit.exportTtl` (a week by default), and the request and each download are `Access`
entries. `AuditExportCleanupWorkflow` deletes expired files hourly.

## Security alerts

`audit_alert_policies` holds each organization's choice for each alert kind:

| Kind | Raised by |
|---|---|
| `RoleChanged`, `APIKeyChanged`, `MFAReset`, `IdentityConfigChanged`, `LegalHoldChanged` | an entry with that action |
| `CrossTenantRefused`, `MalwareRefused` | a refused request or file |
| `BulkDelete` | 50 deletes by one principal in 10 minutes (configurable) |
| `BulkExport` | 25 reads, downloads or exports by one principal in 10 minutes (configurable) |

`auditservice.Alerts` matches each committed entry against the cached policies; counted kinds
increment a Redis counter per principal and window, and raise once when the threshold is
crossed. A raised alert starts `AuditAlertWorkflow`, which notifies everyone holding
`security_audit_log:read` and writes an `audit.alert_raised` entry. Redis being down drops
counted alerts, never entries. Changing a policy needs `security_audit_log:update` and is
itself a `Security` entry.

## Reading the trail

The audit log's GraphQL queries check `audit_log:read`, then list only entries about
resources the reader can read across the organization (`auditlogresolver.readableResources`,
one batched permission check over the registry). A reader with only own or team scope on a
resource does not see its history. `Security` entries need `security_audit_log:read`. The
list reads a summary projection without the states; opening an entry reads its detail, the
other entries of the same request and a field's history (`changedField`).

Sign-ins stay in `auth_events` and have their own page (Sign-in activity), and agent runs in the
AI audit trail; the audit log links to both rather than merging them.

## Storage

`audit_entries` is range-partitioned by month on `timestamp`, with partitions in the
`audit_partitions` schema, which the application role has no privileges on. Creating or dropping
a partition needs the table's owner, and no function here runs as its owner (`SECURITY DEFINER`
is refused by `schemalint`), so `trenova-db` does both on every deploy as the migrator: its
Ensure step runs `ensure_partitions` 24 months ahead and `drop_expired_partitions` past ten
years. A daily check (`AuditPartitionCheckWorkflow`, 01:17) reads how far the partitions reach,
records `trenova_audit_partition_months_ahead`, and logs an error when fewer than three months
remain, long before an insert could find no partition.
The tenant indexes cover time, record, principal, resource, correlation ID and security
entries; trigram indexes back search over the record reference and comment; a BRIN index covers
time. States and metadata use lz4 compression.

## Retention

`audit_entries` and `auth_events` are append-only. `prevent_audit_modification()` refuses
every `UPDATE` and `TRUNCATE`, and every `DELETE` except one by a member of
`trenova_rls_bypass` in a transaction that has run `SET LOCAL trenova.audit_retention = 'on'`,
of a row older than its retention class's floor and covered by no active legal hold.
`prevent_auth_event_modification()` (migration `20261231007050_audit_retention`) allows a
`DELETE` of an authentication event older than 365 days (`iam.AuthEventRetentionDays`) under
the same opt-in, and the foreign key setting `user_id` or `identity_provider_id` to NULL when
the user or identity provider is deleted.

Both tables are listed in `trenova_rls.append_only_tables`, so `reconcile()` keeps
`UPDATE` and `DELETE` revoked from `trenova_tenant`: the application role cannot
issue either even inside its own tenant.

The retention sweep (`AuditRetentionWorkflow`, daily 02:43, `auditretention.Sweep`) deletes
each organization's entries past each class's window in batches of 5,000, deletes
authentication events past 365 days, and removes seals whose hours have aged out. Deletes go
through `postgres.DeleteUnderAuditRetention`, which opens a transaction on the system pool and
opts in. Past ten years whole monthly partitions are dropped on deploy instead. Nothing else may delete
from either table; a sweep that asks for a row the trigger protects fails as a whole.

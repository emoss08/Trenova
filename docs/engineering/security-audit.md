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
| Sign-in throttled (account or IP failure limit) | `auth_events` | `authservice.Login` | — |
| Cloud signup requested, rejected, verified, provisioned (Cloud edition) | `auth_events` | `internal/cloud/signup/cloudsignupservice` | — |
| Cloud signup owner granted Organization Administrator (Cloud edition) | `audit_entries` | `internal/cloud/signup/cloudsignupservice` | yes |
| Instance bootstrap (`trenova db bootstrap`) created the first administrator, the Organization Administrator role and its grant, and the system account | `audit_entries` | `instancebootstrapservice` | yes |
| API key created, updated, rotated, revoked | `audit_entries` | `apikeyservice` | yes |
| Role created or updated, assigned or unassigned, inheritance, separation of duty constraints, resource permissions | `audit_entries` | `roleservice` | yes |
| Identity providers, SCIM directories, tokens and group mappings, access policies | `audit_entries` | `iamservice` | yes |
| Organization memberships replaced | `audit_entries` | `userservice` | yes |
| Password changed, reset link sent by an administrator | `audit_entries` | `userservice`, `passwordresetservice` | yes |
| Two-factor code at sign-in, success or failure | `auth_events` | `authservice.VerifyMFAChallenge` | — |
| Authenticator app enrolled, confirmed or turned off; recovery codes regenerated; two-factor reset by an administrator | `audit_entries` | `mfaservice` | yes |
| Trenova support access allowed, changed or revoked (Cloud edition) | `audit_entries` | `internal/cloud/supportaccess/supportaccessservice` | yes |
| Trenova support session started, elevated to write, returned to read-only, ended (Cloud edition) | `audit_entries` | `internal/cloud/supportaccess/supportaccessservice` | yes |
| Request refused inside a Trenova support session (Cloud edition) | `audit_entries` | `internal/cloud/supportaccess/supportaccessservice` | yes |
| Platform staff member added or removed (`trenova cloud staff`, Cloud edition) | `audit_entries` | `internal/cloud/supportaccess/supportaccessservice.StaffManager` | yes |
| Document downloaded or viewed (content or presigned URL) | `audit_entries` | `documentservice` | — |
| Cross-tenant request refused | `audit_entries` | `middleware.TenantBoundaryMiddleware` | yes |

Every row carries the request ID, client IP and user agent of the request that
caused it. The request ID is the `X-Request-ID` the API returns, so a support
ticket quoting it finds the row.

## Request metadata

`middleware.NewRequestMetaMiddleware` runs right after `requestid.New()` and puts a
`requestmeta.Meta` (request ID, `ClientIP()`, user agent, truncated to the audit
columns) on both the request context and the gin context. Anything that writes an
audit row adds `auditservice.WithRequest(ctx)`: it fills `ip_address`, `user_agent`
and `correlation_id` when the entry has none, and copies the request ID into
`metadata.requestId`. A context with no metadata (a Temporal activity, a CLI
command) leaves those fields empty rather than failing.

`ClientIP()` is only as trustworthy as `server.trustedProxies`. Behind a load
balancer, list it there, or every row records the balancer's address.

## Authentication events

`services.AuthEventRecorder` (`autheventservice`) writes one `auth_events` row per
attempt. The caller fills an `AuthEventRecord` (provider, outcome, user,
organization, assurance levels, an error code) and the recorder adds the request
metadata. `provider` names the method: `password`, `sso.<provider>`,
`session.logout`, `mfa` (the second step of a sign-in), `password_reset.request`, `password_reset.confirm`, `login_throttled`,
and for cloud signup `signup_requested`, `signup_rejected`, `signup_verified`,
`signup_provisioned`. Signup rows carry no user until provisioning; a rejection names
its reason in `error_code` (`honeypot`, `turnstile_rejected`, `turnstile_unavailable`,
`invalid_input`, `existing_account`, `send_limit_reached`, `invalid_token`,
`expired_token`, `signups_paused`, `email_in_use`, `provisioning_failed`).
`login_throttled` rows name the scope in `error_code` (`login_throttled_account`,
`login_throttled_ip`). The throttle fails open: when Redis cannot be reached the attempt
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
recovery codes never reach a row; recovery codes are stored hashed.

API key authentication is not recorded per request; key creation, rotation and
revocation are, and `api_keys.last_used_*` tracks use.

## Administrative changes

Services that change who can do what take a `services.SecurityAuditor` and call
`RecordChange` after the change commits, with the resource, operation, actor,
tenant, before and after state, a comment and metadata. The auditor:

- writes a **critical** entry with category `User` and the request metadata;
- serializes any state, including slices, as a JSON document (a list becomes
  `{"value": [...]}`), and computes the diff from those documents, so a state it
  cannot diff never drops the entry;
- fills a missing tenant or actor from the request's database scope, and falls back
  to the system actor;
- never fails the change. If the entry cannot be stored or buffered, it logs the
  change in full at error and counts `trenova_audit_security_event_total{kind="security_change",status="failure"}`.

Secrets never reach an entry: OIDC client secrets, SCIM token hashes and API key
hashes are `json:"-"`, API keys are recorded through their response shape, and
tokens are identified by prefix only.

Adding a security-relevant mutation means calling the service's `recordChange`
helper after the write succeeds, never before, and never for a refused change.

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
(`content`/`presigned_url`). Because it is in the service, invoice share links, the
driver portal and attachment reads are covered as well as the documents API.
Preview thumbnails are not recorded. A read with no user (a share link) is
attributed to the system principal; the request metadata identifies the requester.

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
the caller's organization (resource `organization`, the attempted organization as
`resource_id`, the operation from the HTTP method) with the route and response
status. A request with no session is only logged. A row-level security rejection
is answered 403 with a generic message instead of surfacing as a server error.

To refuse another cross-tenant shape, call `tenantboundary.Report(ctx, …)` where
the request is refused; the middleware does the rest.

## Surviving an outage

Non-critical entries go to the Redis buffer and fall back to a direct insert. A
critical entry is inserted directly; if that fails it is pushed to the Redis buffer,
whose flush and dead-letter queue retry it, and `trenova_audit_critical_buffered_total`
counts it. Only when both PostgreSQL and Redis refuse does `LogAction` return an
error, and the service logs the entry IDs. The security auditor and the auth event
recorder then log the full event, so the log pipeline is the last copy.

## Retention

`audit_entries` and `auth_events` are append-only. `prevent_audit_modification()`
and `prevent_auth_event_modification()` (migration `20261231007050_audit_retention`)
refuse every `UPDATE` and every `DELETE` except:

- a `DELETE` by a member of `trenova_rls_bypass` in a transaction that has run
  `SET LOCAL trenova.audit_retention = 'on'`, and then
  - for `audit_entries`, of a non-critical row, or of a critical row older than
    365 days (`audit.CriticalRetentionDays`);
  - for `auth_events`, of a row older than 365 days (`iam.AuthEventRetentionDays`);
- for `auth_events`, the foreign key setting `user_id` or `identity_provider_id` to
  NULL when the user or identity provider is deleted.

Both tables are listed in `trenova_rls.append_only_tables`, so `reconcile()` keeps
`UPDATE` and `DELETE` revoked from `trenova_tenant`: the application role cannot
issue either even inside its own tenant.

The retention sweep (`DeleteAuditEntriesActivity`) deletes each organization's
entries past its configured retention, keeping critical entries for at least a
year whatever the organization chose, and deletes authentication events past 365
days. Both deletes go through `postgres.DeleteUnderAuditRetention`, which opens a
transaction on the system pool and opts in. Nothing else may delete from either
table; a sweep that asks for a row the trigger protects fails as a whole.

# Managed mode

A dedicated instance run by Trenova Cloud is the public TMS with `managed.enabled: true`. The
control plane sells it allowances (shipments, AI usage credits, Document Intelligence, document
storage) and bills overage; the TMS counts what was used and stops a metered action when the
control plane says so. The operator carries both ways ([operator.md](operator.md#usage-relay));
this doc is the TMS's side. Self-hosted installs never turn it on, and nothing here runs there.

## What turns on

| Piece | Where | Self-hosted | Managed |
|---|---|---|---|
| `QuotaGuard` | `identity/app/quotaservice` (`New` picks by `managed.enabled`) | `UnlimitedQuotaGuard` | `ManagedQuotaGuard` |
| `UsageProvider` | `kernel/app/usageservice`, chosen in `kernel/module.go` | `NoopUsageProvider` | `ManagedUsageProvider` |
| The operator's listener | `internal/api/managedserver.go`, API process only | off | `managed.host:managed.port` (8090), bearer `managed.token` |
| The first administrator | `trenova db bootstrap --admin-setup`, `identity/app/firstadmininviteservice` | `password` | `link`, then an invite from the operator |

Both providers ask one gate, `managedusageports.Gate` (`kernel/app/managedusageservice`), so every
existing hook enforces the control plane's answer with no new call site: shipments
(`shipmentrepository`, `Preflight` then `EnforceAll` in the insert transaction), AI spend (the
completion router and the assistant's turn limits), document storage (`CheckDocumentBytesLimit`)
and Document Intelligence extractions (`aidocumentservice.admit`, before a call or a deferred
submission; routing is not an extraction).

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

A refusal is `errortypes.QuotaExceededError` with the meter, allowance and quantity, the same
error the Cloud edition's plans raise. The last state applies until the next one: when the
control plane or the operator is down, the instance keeps working on what it was last told.

## The readings

`GET /managed/v1/readings?since=<hour>` answers from the source rows, never from a counter the
TMS keeps, so a resent hour is always the truth and a missed relay is made good by the next:

| App meter | Kind | Source | Index |
|---|---|---|---|
| `shipments.total` | counted | shipments created in the hour (`shipmentmeter`) | `idx_shipments_created_at` |
| `ai.spend_cents` | counted | `ceil(sum(cost_usd) × 100)` of the hour's AI calls that carry a cost (`aiusagemeter`) | `idx_ai_usage_records_created_at` |
| `document_intelligence.extractions` | counted | the hour's successful Document Intelligence extraction calls (`aiusagemeter`) | the same |
| `documents.storage_bytes` | level | the bytes of every stored document now, reported for the current hour (`documentmeter`) | `idx_documents_file_size` |

Each reader lives in the module that owns its table and joins the fx groups
`managed_hourly_counters` or `managed_level_readers`; the kernel service fills the hours with
no rows with zeros. A new meter is a reader in its module plus the control plane's `appMeter`.
The readers are instance-wide (`dbscope.WithSystem`, listed in `rlslint`) because the whole
instance is one customer. `since` must be on the hour, within 35 days and not after the current
hour.

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

## Rules that bite

- The listener is reachable only from the operator: the chart's NetworkPolicy admits
  `managed.from` alone and refuses to render managed mode without it. The token is compared in
  constant time; problem bodies never carry an internal error.
- The routes are not on the API router, so `authzlint` and `writecoverage` do not see them: the
  bearer token is their authorization. They write only what the control plane decides: the usage
  state, and the first administrator's invite, which grants nothing until someone holding the
  emailed token redeems it.
- The invite endpoint never sends email and never returns a token; it takes a hash and answers
  with the same hash or nothing.
- Shipments are never hard-deleted, so counting them by `created_at` is a ledger. A meter whose
  source rows can be deleted needs its own append-only record before it is billed.

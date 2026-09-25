# G01 plan: accounting sync, QuickBooks Online first

Brief: [Gap Program G01](trenova-gap-program.md#g01-accounting-sync-the-ledger-that-stays-in-step-on-its-own).
Status: **approved 2026-09-24** with every recommendation in §12 accepted. Building M1.
Verified against the code on 2026-09-24. `S/` is `services/tms/internal/`, `C/` is `client/apps/web/src/`.

---

## 1. What the code gives us, and what it does not

Every fact here was read in the code, not taken from the brief. The ones that change the design come first.

### 1.1 Facts that shape the design

| # | Fact | Where | Consequence for G01 |
|---|---|---|---|
| F1 | In `JournalPostingMode = Manual` (the default), subledger journals are written as `Pending` or `Approved` and **nothing ever moves them to `Posted`**. The journal repositories expose only `CreatePosting` and `MarkReversed`; balances are updated only when `IsPosted` is true. | `invoiceservice/accounting_helpers.go` `invoicePostingWorkflow`; `journalentryrepository` `MarkReversed`; `journalpostingrepository.CreatePosting` | Ledger mode (push journals) would push nothing for a Manual-mode tenant. Document mode must not depend on journals at all. |
| F2 | Credit memos created by invoice adjustments (void of a posted invoice, CreditOnly, CreditAndRebill, FullReversal) are inserted directly as `Posted` with no journal and no `journal_sources` row. Only write-offs journal. | `invoiceadjustmentservice/service.go` `createCreditMemoInvoice` | `journal_sources` cannot be the sync trigger; it misses real documents. |
| F3 | `journal_sources` rows exist only when a journal was created (`journal_batch_id NOT NULL`), keyed by an idempotency key unique per org and business unit. | migration `20260410203000_add_journal_sources_and_balances` | Useful as a cross-reference in ledger mode, not as the outbox. |
| F4 | Invoices post two header lines to the tenant default AR and revenue accounts. Invoice lines carry `Type` (Freight, Accessorial, Memo) and `ChargeCode` but no GL account. The billing profile's `RevenueAccountID`/`ARAccountID` are never read. `accessorial_charges` has no GL account. There is no revenue-code model. | `accounting_helpers.go:89-120`; `invoice/invoice.go` `InvoiceLine` | Document mode maps line type and charge code to QuickBooks **Items**, which carry the income account. This gives QuickBooks line detail that Trenova's own GL does not have. |
| F5 | Posting services run in `db.WithTx`; the transaction rides the context and nested calls reuse it. There is **no after-commit hook**. Invoice posting enqueues EDI best-effort *after* `WithTx` returns. | `infrastructure/postgres/connection.go` `WithTx`; `invoiceservice/edi.go` `enqueueEDIAfterPost` | Best-effort after-commit enqueue loses work if the process dies between commit and enqueue. G01 writes its outbox row **inside** the posting transaction. |
| F6 | No accounting table has an external-ID column. Customers and carriers have `external_id`, which no service reads. | domain packages | All external identity lives in G01's own mapping and sync tables. We do not reuse `customers.external_id`. |
| F7 | Customer payments, driver settlements and carrier settlements resolve a fiscal period by date and do not check its status. Invoices and manual journals do. | `customerpaymentservice/validator.go`; settlement posting | Inbound payments from QuickBooks must check Trenova's period status explicitly (G01 does not rely on the payment service to refuse a closed period). |
| F8 | `AutoPostSourceEvents` is validated and stored but nothing reads it (`CanUseAutomaticSourcePosting` has no caller). `VendorBillPosted`, `VendorPaymentPosted`, `EscrowInterestAccrued` have no producer. | `accountingcontrolpolicyservice/service.go:62` | G01 does not build on those settings. |
| F9 | Customers have no email or phone column; the email profile (`customer_email_profiles`) holds recipients. | `customer/customer.go`, `emailprofile.go` | The QuickBooks customer's email comes from the email profile's first `To` recipient. |
| F10 | The carrier settlement is the only AP document: posting writes Dr expense / Cr AP and a carrier ledger `Bill`; `MarkPaid` writes Dr AP / Cr Cash and a ledger `Payment`; void writes an `Adjustment`. Carrier invoice numbers live on `carrier_invoice_matches`. | `carriersettlementservice/posting.go`, `lifecycle.go` | Carrier settlements sync as QuickBooks **Bills** to a **Vendor** (the carrier) and `MarkPaid` as a **BillPayment**. |

### 1.2 Integration framework facts

- `integrations` has one row per org, business unit and **type**, a JSONB `configuration`, and an optimistic `version`. It has no status, health, last-error or token-expiry column. `secretconfig.Codec.Merge` drops keys not in the spec, and only `integrationservice.UpdateConfig` writes the row. **Rotating OAuth tokens therefore cannot live in `integrations.configuration`**: a background refresh would race admin saves on `version`, and the next save would drop the keys. They get their own table (§3).
- `TestConnection` auto-enables an integration on success. The catalog status is derived only from `Enabled` and required fields.
- There is no third-party OAuth connect flow. User SSO (`authservice`) already uses `golang.org/x/oauth2` (a direct dependency) with state, nonce and PKCE in Redis (`ssologinstaterepository`), 10-minute TTL, get-then-delete (not atomic).
- **Production session cookies are `SameSite=Strict` with the `__Host-` prefix** (enforced by `config/loader.go`). A redirect from Intuit to an API callback would arrive without the session. This dictates the OAuth completion design in §5.
- `shared/restx` retries on transport errors, 429 and 5xx **for every method, POST included**, with backoff and `Retry-After`. Safe for QuickBooks only because every write carries a stable `requestid` (§4.3).
- `outboundlimit.Limiter` implements `restx.Limiter` over Redis. The carrier-intel service has a per-tenant circuit breaker and a per-call usage recorder. Telematics and carrier intel persist feed state (`LastSuccessAt`, `FailureCount`, `LastError`); neither is a Watchtower source.
- No integration has a multi-step wizard. Bespoke modals exist (Samsara, carrier intelligence with tabs). Dialog sizes: `xs`, `sm` (default), `md`, `lg` (`max-w-2xl`), `xl` (`max-w-4xl`), `2xl`.
- Integrations are REST only (`integrationhandler`); there are no GraphQL resolvers for them.
- `TaskQueueIntegration` (`integration-queue`) is in use (fuel prices, rate simulation). It is the queue for G01.
- No string-similarity helper exists in `shared/`. `shared/stringutils` has normalization helpers only.
- `redishelpers` has `GetJSON`/`SetJSON` but no atomic get-and-delete.

### 1.3 QuickBooks Online facts (checked 2026-09-24)

- Writes accept a `requestid` query parameter: the same id replays the original response instead of creating a second transaction ([Intuit best practices](https://help.developer.intuit.com/s/article/QuickBooks-Online-API-Best-Practices)).
- Throttling is 500 requests per minute per realm, batch requests included ([API call limits](https://help.developer.intuit.com/s/article/API-call-limits-and-throttling)).
- Change Data Capture returns changed entities since a timestamp, looking back at most 30 days, up to 1000 objects per call ([CDC](https://developer.intuit.com/app/developer/qbo/docs/learn/explore-the-quickbooks-online-api/change-data-capture)).
- Access tokens are short-lived; each refresh returns a new refresh token valid 100 days, and refresh tokens now have a five-year maximum validity ([refresh token policy](https://help.developer.intuit.com/s/article/Validity-of-Refresh-Token), [policy change](https://blogs.intuit.com/2025/11/12/important-changes-to-refresh-token-policy)). Always store the latest one.
- Webhooks moved to the **CloudEvents** format; migration was required by May 15, 2026 ([upcoming changes](https://blogs.intuit.com/2025/12/01/upcoming-changes-to-apis-and-tools-that-may-impact-your-application/)).

To confirm against Intuit's current reference in M1 before coding against them (Intuit's doc pages did not render for automated reading): the CloudEvents envelope and signature header, whether PKCE is supported on the authorize endpoint, `DocNumber` maximum length, which entities expose `MetaData.LastModifiedByRef`, and the Preferences field for the books closing date. The plan does not depend on any of these being a particular value; each has a handling path below.

---

## 2. Design in one page

1. **Documents, not journals, drive sync.** Every posting path that creates or changes an accounting document writes an `accounting_sync_records` row **inside its own transaction** through a new `AccountingSyncEnqueuer` port. The row is the outbox and the ledger at once. No active connection means the enqueuer is a no-op.
2. **A safety net proves nothing is missed.** A sweep finds documents posted since the sync start date that have no sync record and enqueues them. The same query drives backfill. A test fails if a posting path is added without enqueueing.
3. **One dispatcher per connection.** A Temporal workflow per connection (`accounting-sync:<connectionID>`) drains the outbox in dependency order (customer, item, vendor → invoice/credit memo/bill → payment/bill payment). It is kicked right after commit through `workflowstarter.SignalWithStartWorkflow` for latency, and swept every minute for durability. One workflow per realm respects the 500-per-minute and concurrency limits by construction, and `restx.Limiter` adds a per-realm bucket.
4. **Every write is idempotent twice.** Trenova side: one record per `(connection, idempotency_key)`, where the key is `<objectType>:<objectID>:<operation>:<revision>`. QuickBooks side: the record's stable `request_id` is sent as `requestid`, so a restx retry or a Temporal activity retry replays instead of duplicating.
5. **Inbound is pull, triggered by push.** A webhook only says "this realm changed". It enqueues a CDC poll for that connection. CDC is the source of truth, so ordering, duplication and missed webhooks do not matter. A scheduled poll every 5 minutes covers silence.
6. **Drift is computed, not guessed.** A nightly job compares each synced document's total and status with QuickBooks, and AR by customer for the period, and raises specific findings with both values, the time and (where QuickBooks supplies it) the editor.
7. **Tokens live in their own row.** `accounting_connections` stores the encrypted access and refresh tokens, their expiries, and the connection's health. Refresh takes a row lock so two workers never refresh at once.
8. **Connectors are adapters.** A capability-split `AccountingConnector` port; QuickBooks Online is the first adapter. Xero, Business Central and NetSuite follow as their own milestones.

---

## 3. Data model

All tables: composite PK `(id, organization_id, business_unit_id)`, tenant FKs, `version` where rows are edited, Unix-second `BIGINT` timestamps, money as `NUMERIC(19,4)` plus `*_minor BIGINT` where Trenova's own tables do. New enums use varchar with named `CHECK` constraints (`ck_<table>_<col>`), registered in `enum_constraints_test.go`, so later values never need a non-transactional `ALTER TYPE`. SQLite mirrors regenerate with `task sqlite-convert`.

### 3.1 `accounting_connections`

One per tenant per provider. Columns:

- `integration_type` (FK semantics to `integration.Type`), `status`: `Connecting`, `Connected`, `Degraded`, `Failing`, `Disconnected`, `Revoked`.
- `external_realm_id`, `external_company_name`, `external_country`, `external_home_currency`, `external_multicurrency_enabled`, `external_books_closed_through` (Unix date, from Preferences; nullable).
- Tokens: `access_token_ciphertext`, `access_token_expires_at`, `refresh_token_ciphertext`, `refresh_token_expires_at`, `refresh_token_absolute_expires_at`, `last_refreshed_at`. Encrypted with `encryptionservice` envelope and AAD bound to tenant, connection id and field (new `Purpose` `AccountingConnectionToken`). Never selected by any read query except the token repository method.
- Settings: `mode` (`Document` now; `Ledger` added in M6 by a new CHECK value), `ledger_granularity` (`Detailed`, `DailySummary`, M6), `sync_start_date`, `auto_sync` (sync on post, or hold records as `AwaitingApproval` until a person releases them), `inbound_payment_policy` (`Off`, `Propose`, `Apply`; M5), `default_class_external_id`, `default_location_external_id`, `doc_number_policy` (`UseTrenovaNumber`, `LetProviderNumber`).
- Setup: `setup_step` (`Connect`, `Mappings`, `StartDate`, `Complete`) so the wizard resumes where it was left.
- Health: `last_success_at`, `last_failure_at`, `consecutive_failures`, `last_error_category`, `last_error_message`, `paused_at`, `paused_by_id`, `paused_reason`, `cdc_cursor_at`, `last_webhook_at`, `last_reconciled_at`.
- `connected_by_id`, `connected_at`, `disconnected_by_id`, `disconnected_at`.
- Constraints: unique `(organization_id, business_unit_id, integration_type)`; **unique `(integration_type, external_realm_id)` where status is not `Disconnected`**, so one QuickBooks company can be connected to one Trenova tenant only (webhooks route by realm, and a company connected twice would receive every document twice).

The `integrations` row for `QuickBooksOnline` still exists: it carries the catalog card, `Enabled`, and nothing sensitive. Connecting sets `Enabled`; disconnecting clears it.

### 3.2 `accounting_reference_objects`

A cache of the provider's reference data, used for mapping search, proposals and validation: `connection_id`, `kind` (`Account`, `Item`, `Customer`, `Vendor`, `Class`, `Location`, `TaxCode`, `Term`, `PaymentMethod`), `external_id`, `name`, `fully_qualified_name`, `account_type`, `account_sub_type`, `parent_external_id`, `active`, `currency_code`, `sync_token`, `last_modified_at`, `attributes` (bounded JSONB of the fields the matcher reads), `last_seen_at`. Unique `(connection_id, kind, external_id)`. Index on `(connection_id, kind, lower(name))`. Refreshed on connect, daily, and from CDC.

### 3.3 `accounting_mappings`

`connection_id`, `kind` (as above), `trenova_object_type` (`GLAccount`, `Customer`, `Carrier`, `AccessorialCharge`, `InvoiceLineType`, `PaymentTerm`, `PaymentMethod`, `AccountRole`), `trenova_object_id` (nullable), `trenova_key` (for non-entity rows: a line type like `Freight`, a payment term like `Net30`, a role like `ShortPayWriteOff`, `UndepositedFunds`, `ARAccount`, `APAccount`), `external_id`, `external_name` (denormalized for display), `state` (`Proposed`, `Confirmed`, `Rejected`), `source` (`Suggested`, `Manual`, `CreatedInProvider`, `Agent`), `confidence NUMERIC(5,4)`, `reason` (plain text shown in the UI), `signals` (JSONB: which matchers fired and their scores), `confirmed_by_id`, `confirmed_at`. Unique `(connection_id, kind, trenova_object_type, coalesce(trenova_object_id, ''), coalesce(trenova_key, ''))`. Only `Confirmed` rows are used by payload builders.

### 3.4 `accounting_sync_records` (outbox and ledger)

`connection_id`, `object_type` (`Customer`, `Vendor`, `Item`, `Invoice`, `CreditMemo`, `DebitMemo`, `CustomerPayment`, `CarrierBill`, `CarrierBillPayment`; M6 adds `JournalEntry`), `object_id`, `object_number` (for display and search), `operation` (`Create`, `Update`, `Void`, `Delete`), `source_event` (what enqueued it, for example `InvoicePosted`, `InvoiceAdjustmentCreditMemo`, `CustomerPaymentReversed`, `Backfill`, `SafetyNet`, `DependencyOf`), `idempotency_key`, `request_id`, `revision`, `depends_on_record_id`, `status` (`Queued`, `AwaitingApproval`, `InFlight`, `Synced`, `Blocked`, `Failed`, `DeadLettered`, `Skipped`, `Superseded`), `attempt_count`, `next_attempt_at`, `lease_expires_at`, `external_id`, `external_doc_number`, `external_url`, `external_sync_token`, `payload_hash`, `payload` (JSONB request body, retained per §8.4), `error_category` (`Transient`, `RateLimited`, `Auth`, `Validation`, `Mapping`, `ClosedPeriod`, `Currency`, `Duplicate`, `NotFound`, `Conflict`), `error_code`, `error_message` (provider text), `resolution` (plain-language fix shown to the user), `blocked_reason`, `queued_at`, `started_at`, `synced_at`, `skipped_by_id`, `skipped_reason`.

Constraints and indexes: unique `(connection_id, idempotency_key)`; index `(organization_id, business_unit_id, connection_id, status, next_attempt_at)`; index `(organization_id, business_unit_id, object_type, object_id)` for "sync state of this record".

### 3.5 `accounting_sync_attempts`

One row per attempt, append-only: `sync_record_id`, `attempt_number`, `started_at`, `finished_at`, `duration_ms`, `http_status`, `outcome`, `error_category`, `error_code`, `error_message`, `response_excerpt` (redacted, 4 KB cap), `provider_request_id`. This is the brief's "a row per object per attempt".

### 3.6 `accounting_drift_findings`

`connection_id`, `sync_record_id` (nullable for account-level findings), `object_type`, `object_id`, `external_id`, `fiscal_period_id`, `kind` (`AmountMismatch`, `StatusMismatch`, `DeletedInProvider`, `VoidedInProvider`, `MissingInProvider`, `UnknownInProvider`, `CustomerBalanceMismatch`), `trenova_value`, `provider_value`, `difference` (money columns where the kind is money), `provider_modified_at`, `provider_modified_by` (when the provider exposes it), `status` (`Open`, `Resolved`, `Dismissed`), `resolution` (`PushedTrenovaValue`, `AdjustedTrenova`, `Dismissed`), `resolved_by_id`, `resolved_at`, `resolution_note`, `detected_at`, `last_seen_at`. Unique open finding per `(connection_id, object_type, object_id, kind)` (partial unique index on `status = 'Open'`).

### 3.7 `accounting_backfills`

`connection_id`, `range_start`, `range_end`, `object_types` (array), `cursor` (last object id and posted-at per type), `status` (`Queued`, `Running`, `Paused`, `Completed`, `Failed`, `Cancelled`), `total_estimated`, `enqueued_count`, `synced_count`, `failed_count`, `requested_by_id`, `started_at`, `completed_at`, `last_error`. Resumable: the workflow reads and advances `cursor` in each activity.

### 3.8 Existing tables touched

- `integration_type` enum: add `QuickBooksOnline` (M1), `Xero`, `BusinessCentral`, `NetSuite` (their milestones). `integration_category`: add `Accounting`. Non-transactional migration with `--bun:split` per value.
- `agent_subject_type_enum`: add `AccountingConnection`, `AccountingSyncRecord`, `AccountingDriftFinding` (non-transactional).
- No change to invoices, payments or settlements. External identity lives in §3.3 and §3.4.

---

## 4. Backend

### 4.1 Ports (`S/core/ports/services/accountingconnector.go`)

Capability interfaces, so each provider implements what it supports and the UI shows capability badges:

- `AccountingOAuth`: `AuthorizeURL(state, codeChallenge)`, `ExchangeCode(code, verifier, realmID)`, `Refresh(refreshToken)`, `Revoke(token)`.
- `AccountingCompanyReader`: `CompanyInfo`, `Preferences` (home currency, multicurrency, books closed through).
- `AccountingReferenceReader`: `ListReference(kind, since)` for accounts, items, customers, vendors, classes, locations, tax codes, terms, payment methods.
- `AccountingDocumentWriter`: `UpsertCustomer`, `UpsertVendor`, `UpsertItem`, `CreateInvoice`, `UpdateInvoice`, `VoidInvoice`, `CreateCreditMemo`, `CreatePayment`, `VoidPayment`, `CreateBill`, `VoidBill`, `CreateBillPayment`, `ReadDocuments(kind, ids)` (for drift).
- `AccountingJournalWriter` (M6): `CreateJournalEntry`, `VoidJournalEntry`.
- `AccountingChangeReader`: `ChangesSince(kinds, since)`.
- `AccountingWebhookVerifier`: `Verify(headers, body)`, `ParseRealms(body)` (returns the realm ids that changed; nothing else is trusted).
- `AccountingConnectorFactory`: `For(ctx, connection)` returns a bound client with fresh tokens and the per-realm limiter, patterned on `infrastructure/telematics/factory.go` but keyed by connection, not "first match".

Payloads cross the port as provider-neutral structs (`AccountingInvoice`, `AccountingLine`, `AccountingPayment`, `AccountingBill`), built by `S/core/services/accountingsyncservice/payloads.go` from Trenova documents and confirmed mappings. The adapter translates them to provider JSON. This keeps mapping rules in one place for every provider.

### 4.2 Provider client (`shared/quickbooks`)

A client package like `shared/carrierok` and `shared/fmcsa`, on `shared/restx` with:

- Fixed hosts only (production and sandbox accounting API, the OAuth endpoints), enforced by an allowlist (the `hostguard` pattern), no user-supplied URLs.
- `requestid` on every write, set from the sync record's `request_id`; `minorversion` pinned in one constant.
- A `restx.Limiter` bucket per realm (`outboundlimit`), sized under 500 per minute with headroom for inbound polling.
- An `ErrorDecoder` that maps provider faults to the error categories in §3.4 (auth, validation with the field, stale `SyncToken` as `Conflict`, duplicate document number as `Duplicate`, closed books as `ClosedPeriod`, 429 as `RateLimited`).
- Secrets redacted from errors (restx already redacts configured headers and query keys; the `Authorization` header is added to the redaction list).
- Contract tests against recorded fixtures served by `httptest`, one per entity and fault.

Adapter: `S/infrastructure/accounting/qboconnector/` implements the ports on top of the client. Factory and FX module: `S/infrastructure/accounting/`.

### 4.3 Services

- **`accountingconnectionservice`**: start connect (state, PKCE), complete connect, disconnect (revoke at provider, wipe tokens, set `Disconnected`, keep history), settings update, pause and resume, health transitions, token access with refresh under `SELECT ... FOR UPDATE` on the connection row. A refresh that returns `invalid_grant` sets `Revoked` and raises the Watchtower item. A refresh token within 14 days of absolute expiry raises a "reconnect soon" item.
- **`accountingmappingservice`**: reference refresh, proposals, confirm, reject, bulk confirm, create-in-provider (Items, Customers, Vendors), usage counts (how many synced records use a mapping), and the remap guard: remapping an account or item that synced records reference requires `acknowledgeHistory: true` and is audited.
- **`accountingsyncservice`**: the enqueuer (`AccountingSyncEnqueuer` port implementation), payload builders, dependency resolution (ensure the customer, items and terms are mapped or creatable before an invoice; block with a `Mapping` reason and the exact missing mapping otherwise), dispatcher activities, error classification, retry scheduling (exponential from 30 s to 6 h, then `DeadLettered` after 8 attempts for transient errors; validation, mapping and closed-period errors go straight to `Blocked` and never retry blindly), replay and bulk replay, skip, safety-net sweep, backfill, and the sync-state reader used by GraphQL dataloaders.
- **`accountingdriftservice`**: nightly reconcile, finding upsert and auto-resolve (a finding that no longer reproduces resolves itself with `resolution_note = "No longer differs"`), resolve actions with previews.
- **`accountinginboundservice`** (M5): CDC apply, inbound payment matching and application per `inbound_payment_policy`, reference-cache updates.

### 4.4 Enqueue points (inside the existing transactions)

| Service and function | Record | Notes |
|---|---|---|
| `invoiceservice.Post` | `Invoice`, `CreditMemo` or `DebitMemo` / `Create` | Debit memos sync as QuickBooks invoices (QuickBooks has no debit memo); the memo reason and reference invoice go in the private note. |
| `invoiceservice.CreateMemo` with `AutoPost` | same | Runs inside the memo's transaction, which `WithTx` reuses. |
| `invoiceservice` void of a Draft | none | Drafts never synced. |
| `invoiceadjustmentservice.createCreditMemoInvoice` | `CreditMemo` / `Create` | Covers F2: these memos have no journal but must reach the books. |
| `invoiceadjustmentservice` full reversal of a synced invoice | `CreditMemo` / `Create` (never an in-place void) | A posted Trenova invoice is reversed by a credit memo, so QuickBooks mirrors the same audit trail. |
| `invoiceadjustmentservice` write-off | `CreditMemo` / `Create` to the `ShortPayWriteOff` role item | |
| `customerpaymentservice.PostAndApply`, `ApplyUnapplied` | `CustomerPayment` / `Create` or `Update` | Short-pay amounts become a credit memo to the write-off item linked in the same QuickBooks payment. |
| `customerpaymentservice.Reverse` | `CustomerPayment` / `Void` | |
| `carriersettlementservice.Post` | `CarrierBill` / `Create` (M4) | Vendor is the carrier; lines use mapped expense accounts; the carrier invoice number from `carrier_invoice_matches` becomes the bill's reference. |
| `carriersettlementservice.Void` | `CarrierBill` / `Void` (M4) | |
| `carriersettlementservice.MarkPaid` | `CarrierBillPayment` / `Create` (M4) | |
| `customerservice` / `carrierservice` update of a mapped record | `Customer` / `Vendor` `Update` | Only when a mapping exists; unmapped ones are created on demand as dependencies. |

The enqueuer reads the active connection through a per-request cached lookup and returns immediately when there is none, so tenants without an accounting integration pay one indexed read per posting. After the transaction commits, each service calls `accountingsyncservice.Kick(connectionID)` best-effort. Losing the kick costs at most one minute; losing the record is impossible because it committed with the document.

`TestEveryPostingPathEnqueues` (in `accountingsyncservice`) lists every function in the table and fails when a new `CreatePosting` caller or invoice-status transition appears without a matching entry, so the set cannot drift silently.

### 4.5 Temporal (`S/core/temporaljobs/accountingsyncjobs/`)

On `IntegrationTaskQueue`, package layout as `detentionjobs` (`activities.go`, `workflow.go`, `registry.go`, `schedules.go`, `types.go`, `module.go`), wired in `bootstrap/app.go`, activity names added to `activitynames_test.go`:

| Workflow | Trigger | Does |
|---|---|---|
| `DrainAccountingOutboxWorkflow` | signal-with-start per connection; sweep schedule every minute via `RunTenantFanOut` | Claims due records in dependency order with a lease (`FOR UPDATE SKIP LOCKED` then `lease_expires_at`, the pattern `airetrievalrepository/indexentry.go` already uses), pushes, records attempts, schedules retries. A lease that expires (worker died mid-push) returns the record to the queue; the stable `requestid` makes the re-push a replay. Continues-as-new after N records. |
| `RefreshAccountingTokensWorkflow` | every 10 minutes | Refreshes access tokens expiring within 15 minutes; flags refresh tokens near absolute expiry. |
| `PollAccountingChangesWorkflow` (M5) | every 5 minutes, and signalled by the webhook | CDC since cursor, reference cache updates, inbound payments, drift candidates. Falls back to a full compare when the cursor is older than 30 days. |
| `ReconcileAccountingDriftWorkflow` (M5) | nightly per connection | Document totals and statuses in batches, AR by customer per period. |
| `AccountingSafetyNetWorkflow` | hourly | Finds posted documents since `sync_start_date` with no record; enqueues with `source_event = SafetyNet`; counts appear on the status page. A non-zero count is itself a Watchtower item, because it means an enqueue point is missing. |
| `BackfillAccountingWorkflow` | on request | Pages through documents by type and posted date, enqueues, advances `accounting_backfills.cursor`; pause, resume, cancel by signal. |
| `RefreshAccountingReferenceWorkflow` | on connect, daily, and on demand | Pulls reference data into §3.2 and re-scores open proposals. |
| `AccountingSyncRetentionWorkflow` | daily | Clears `payload` on synced records older than the retention window; attempts older than it are deleted. |

### 4.6 Webhook

`POST /webhooks/accounting/quickbooks/` on the public group, body capped at 1 MiB, rate-limited by client IP. It verifies the signature with the app-level verifier token (constant-time compare), parses realm ids only, looks up connections by `(integration_type, external_realm_id)`, updates `last_webhook_at`, and signals `PollAccountingChangesWorkflow`. It never applies payload contents. Unknown realms return 200 and are logged at debug, so a stale subscription does not retry forever. Until M5 lands, the handler records `last_webhook_at` and returns; M5 adds the signal.

### 4.7 Configuration

App-level Intuit credentials live in server config, not per tenant (one Intuit app serves every tenant): `integrations.quickbooks.clientId`, `clientSecret`, `webhookVerifierToken`, `environment` (`sandbox` or `production`), `redirectPath`. Validated in `config.go`; the catalog card shows "Not available" with the reason when unset, rather than a broken connect button.

### 4.8 Mapping assistant

A deterministic scorer, then an optional model pass, each proposal carrying a confidence and a reason sentence:

1. **Deterministic** (`shared/stringutils/similarity.go`, new: normalized Jaro-Winkler and token-set ratio, no dependency):
   - Accounts: exact account number, then normalized name, then fully-qualified name, weighted by account-type compatibility (Trenova `accounttype.Category` against the provider account type) and by usage (Trenova accounts with journal activity rank above dormant ones).
   - Customers and vendors: exact name or DOT/MC number, then normalized name with legal suffixes stripped (Inc, LLC), then address and postal code as a tie-breaker.
   - Items: charge code equal to item SKU or name, then description similarity; income account must match the mapped revenue account role.
   - Terms and payment methods: fixed synonym table (`Net30` ↔ "Net 30").
2. **Model pass** for the remaining low-confidence items only, as AI task kind `AccountingMapping` (§6.7), with the candidates' names and types, never balances. Its answer is accepted only if it names a candidate the deterministic step supplied. With no provider configured, the deterministic result stands.
3. Confidence bands: `≥ 0.95` pre-checked in the review, `0.70–0.95` shown with the reason, `< 0.70` shown as "no confident match" with the top three candidates and a "Create in QuickBooks" action.

---

## 5. OAuth connect flow

Designed around `SameSite=Strict` session cookies (§1.2):

1. The user clicks **Connect QuickBooks** in the wizard. The GraphQL mutation `startAccountingAuthorization` (authenticated, `accounting_integration:manage`, a signed-in person only) mints a 32-byte state, stores only its SHA-256 hash with `{userID, orgID, buID, integrationType}` in Redis for 10 minutes, and returns the Intuit authorize URL. Trenova is a confidential client (the client secret never leaves the server), so the state plus the secret-authenticated code exchange protect the flow; PKCE is added if Intuit's reference confirms support.
2. The browser opens it in the same tab. Intuit redirects to the **web app** route `C/routes/admin/integrations/quickbooks/callback` with `code`, `state` and `realmId`.
3. That page (a top-level navigation, so no API call has happened yet) immediately calls the GraphQL mutation `completeAccountingAuthorization` with the three values. This is a same-site request, so the `Strict` session cookie is sent.
4. The server atomically takes the state (`GETDEL`, new `redishelpers.GetDelJSON`), requires that the session user and tenant equal the ones that started it, exchanges the code, reads company info and preferences, refuses a realm already connected to another tenant, stores encrypted tokens, sets `setup_step = Mappings`, audits, and returns.
5. The page routes back to the integrations catalog with the wizard open at the mapping step.

A state used twice, expired, or presented by a different user fails with a specific message and no side effect. As built (M1), every authenticated step is GraphQL, so persisted operations, the resolver permission lint and generated client types cover it; only the public webhook is REST. The realm id in the redirect is not trusted on its own: the connection is saved only after the new access token successfully reads that company's info.

---

## 6. AI surface (Part 2 contract)

### 6.1 Permissions

- `accounting_integration` (Category "Accounting", `SensitivityRestricted`): `read`, `update` (settings, mappings), `manage` (connect, disconnect, pause, resume, backfill).
- `accounting_sync` (Category "Accounting", `SensitivityInternal`): `read`, `update` (retry, skip, resolve drift), `export`.
- Both hand-added to `resource_gen.go` and `registry.go`, mirrored with `pnpm --filter @trenova/graphql resources`, with routes in `routeregistry.go` and a route test.
- `agentAllowedPermissions`: `accounting_sync` read and update; `accounting_integration` read and update. `manage` is **not** granted to agents: connect, disconnect and backfill stay with people (backfill is offered as a tool at `update` scope, see below).

### 6.2 Tools

Egress: a push to the tenant's own books is the tenant's own ledger, so sync tools declare `EgressInternal` with that rationale. Creating reference records in the provider also declares `EgressInternal`. **Decision D2 asks Eric to confirm this classification.** No tool moves money.

Read (query) tools, `readPolicy`:

| Tool | Resource | Returns |
|---|---|---|
| `get_accounting_sync_status` | `accounting_sync` | Connection health, queue depth by status, last success per object type, safety-net count, token expiry |
| `list_accounting_sync_records` | `accounting_sync` | `listSpec`: filter by status, object type, error category, date; the Desk table |
| `get_accounting_sync_record` | `accounting_sync` | One record with attempts, error, resolution and external link; the Desk entity card |
| `get_record_accounting_sync_state` | `accounting_sync` | Sync state for a Trenova invoice, payment or settlement by its id (what the header line shows) |
| `list_accounting_drift_findings` | `accounting_sync` | `listSpec`: open findings with both values |
| `get_accounting_mapping` | `accounting_integration` | Current mapping for an account, item, customer, vendor or role, with usage count |
| `list_accounting_mapping_gaps` | `accounting_integration` | Unmapped or low-confidence items blocking sync, with candidate matches and reasons |

Action tools (all implement `ToolSimulator` and `TargetedTool`; all that touch the provider set `Idempotent: true`):

| Tool | Resource / op | Default tier | Max tier | Reversible | Validator | Purpose |
|---|---|---|---|---|---|---|
| `set_accounting_mapping` | integration / update | Propose | ActWithApproval | yes, `clear_accounting_mapping` | mapping service | Confirm or change one mapping (the brief's `propose_accounting_mapping`: a proposal an agent raises becomes this call once approved) |
| `clear_accounting_mapping` | integration / update | Propose | ActWithApproval | no | same | Remove a mapping; refused while queued records depend on it |
| `create_accounting_reference_record` | integration / update | Propose | ActWithApproval | no | same | Create a missing Item, Customer or Vendor in the provider and map it |
| `refresh_accounting_reference_data` | integration / update | AutoExecute | AutoExecute | n/a (no state change a person sees) | none | Re-pull reference data and re-score proposals |
| `retry_accounting_sync` | sync / update | AutoExecute | AutoExecute | no | sync service (only `Failed`, `DeadLettered`, `Blocked` whose cause is fixed) | Re-queue records; idempotent by `requestid` |
| `skip_accounting_sync` | sync / update | ActWithApproval | ActWithApproval | yes, `retry_accounting_sync` | same | Mark records as intentionally not synced, with a reason |
| `resolve_accounting_drift` | sync / update | ActWithApproval | ActWithApproval | no | drift service | Push Trenova's value, or adjust Trenova through an invoice adjustment; Simulate shows both sides and the resulting entries |
| `dismiss_accounting_drift` | sync / update | ActWithApproval | ActWithApproval | no | drift service | Dismiss with a required note |
| `request_accounting_backfill` | integration / update | ActWithApproval | ActWithApproval | no | backfill validator | Queue a backfill for a range; a person with `manage` releases it (the queued backfill waits in `Queued` until then) |
| `pause_accounting_sync` | integration / update | AutoExecute | AutoExecute | yes, `resume_accounting_sync` | none | Stop pushing (for example during a provider outage) |
| `resume_accounting_sync` | integration / update | ActWithApproval | ActWithApproval | no | none | Resume pushing |

Connect, disconnect and settings changes have no tools by design: they change credentials or where the books live (Part 2: credentials never auto-execute; here they are simply not offered). This is stated in the tool catalog's family description so tenants see why.

### 6.3 Events and subjects

`knownEvents` additions: `accounting.sync_failed` (a record reached `DeadLettered`), `accounting.sync_blocked` (mapping, currency or closed period), `accounting.drift_detected`, `accounting.connection_degraded` (health left `Connected`, including `Revoked` and token near expiry). Subject types `AccountingConnection`, `AccountingSyncRecord`, `AccountingDriftFinding`, each with an `agentsubjectservice` arm, `subjectrecord.go` prefix entry and repository mapping.

### 6.4 Watchtower

`SourceAccountingSync` (varchar column, no migration), mapped to `accounting_sync` read, added to the GraphQL enum. Snapshot items: records `DeadLettered` or `Blocked` older than 15 minutes (grouped per cause so 400 records blocked by one missing item are one item that names the item), open drift findings, safety-net count above zero, connection not `Connected`, refresh token within 14 days of absolute expiry, provider books closed through a date that blocks queued records. Live upsert and resolve from the services as states change.

### 6.5 Template

`TemplateBooksKeeper` ("Books Keeper"): wakes on the four events and a weekly schedule; tools are the read tools plus `retry_accounting_sync`, `refresh_accounting_reference_data`, `set_accounting_mapping`, `create_accounting_reference_record`, `resolve_accounting_drift`, `dismiss_accounting_drift`, `pause_accounting_sync`. Starts with shadow mode on and ceiling `ActWithApproval`. Instructions: retry only transient failures, propose mapping fixes with the reason, never dismiss drift above the tenant's reconciliation tolerance, write a weekly reconciliation note in Desk. All `Starter*` switches, `identity.go` icon, `starters.go`, GraphQL `AgentTemplate`, the client template picker, and the prompt snapshot.

### 6.6 Desk

`get_*` tools map to entity cards and `list_*` to tables through the existing `artifactFromObservation` prefixes. No new artifact kind: the drift comparison fits the entity card's field list, and Simulate already renders `FieldChange`s in the decision queue. Revisit only if the card cannot show both values legibly.

### 6.7 AI task kind

`TaskAccountingMapping` in `aiprovider/enums.go`, `aiproviderhandler/catalog.go`, `modeladapter/sampling.go`, `aiprovider.graphqls` (enforced by `task_wiring_test.go`). `CompleteStructured` with a JSON schema; deterministic fallback per §4.8.

### 6.8 Reporting

`reportcatalog.yml`: `accounting_sync_record` and `accounting_drift_finding` (category Accounting, curated fields, edges to connection). Canned reports: **Sync exceptions by week** (records that failed, blocked or were skipped, by week, object type and cause) and **Open drift by customer**.

### 6.9 Presentation and catalog

`TOOL_TITLES` entries for every tool; a tool family `accounting_sync` in `agenttoolcatalog/families.go`; `agentevalgate` tool-selection cases (for example "why didn't INV-1042 reach QuickBooks" → `get_record_accounting_sync_state`); golden catalog and `ai-tool-safety.md` regenerated.

---

## 7. UI

All text through `t()`; tokens only; `pnpm lint:design` clean.

### 7.1 Integrations catalog and setup wizard

- Catalog card "QuickBooks Online" under a new **Accounting** category. Its modal is a wizard (`Dialog size="lg"`), the first multi-step setup in the catalog, built as a reusable `IntegrationSetupWizard` in `C/routes/admin/integrations/_components/shared/` so Xero and the others reuse it.
- Steps: **Connect** (what will be shared and why; the connect button; after return, the company name and home currency) → **Map** (grouped: accounts by role, items, customers, vendors, terms; low-confidence first; `AssistMark` on proposals with a "why" disclosure listing the signals; `j`/`k`, `x`, `a`, `r`, `Shift+A`; "Create in QuickBooks" for no-match rows; a counter of what still blocks sync) → **Start date and backfill** (date picker, estimated document counts per type, backfill on or off, auto-sync on post or hold for approval) → **Done** (what happens next, link to the sync page).
- The wizard resumes at `setup_step`. It can be closed at any step without loss.
- In M6 a **Mode** step is inserted after Connect, explaining the two modes in plain language with an example of what each sends.

### 7.2 Accounting sync pages (Accounting module navigation)

- `/accounting/sync` (added to the existing `accountingModule` in `C/config/navigation.config.ts`, beside manual journals and reversals): `PageLayout` with header actions (Pause/Resume, Retry failed, Backfill), `KpiStrip` (Synced today, Pending, Needs attention, Drift findings, Last success), then `DataTableLazyComponent` > the sync ledger table (object, number, operation, status with phase tone, attempts, last error in plain language, external link), filters by status, object type, error cause and date, bulk Retry and Skip. Row opens a side sheet: status timeline of attempts, the plain-language cause and the one action that fixes it (for example "Map charge code DET to a QuickBooks item" opening the mapping row), the request payload (collapsed), and the QuickBooks link.
- `/accounting/sync/drift`: findings table with both values side by side, difference, when and by whom, resolve and dismiss actions with a preview dialog.
- `/accounting/sync/mappings`: the mapping editor outside the wizard, search on both sides, usage counts, confidence, history warning before remapping.
- Empty states: not connected ("Connect QuickBooks", and a line that the Books Keeper agent will watch the sync once connected); connected with nothing yet ("Posted invoices will appear here within a minute"); all clear ("Everything posted since {date} is in QuickBooks").
- Keyboard: every list supports `j`/`k` and the decision-queue keys; pages registered in the command palette ("Accounting sync", "Drift findings", "Accounting mappings").

### 7.3 Sync state on records

A shared `AccountingSyncStateLine` in the header of invoice, credit memo, customer payment and carrier settlement detail views, and on customer and carrier detail: "Synced to QuickBooks 2 min ago" with the external link, "Waiting to sync", "Blocked: charge code DET has no QuickBooks item" with a Fix action, or "Failed: {cause}" with Retry. Fed by an `accountingSync: AccountingSyncState` field on the GraphQL `Invoice`, `CustomerPayment`, `CarrierSettlement`, `Customer` and `Carrier` types through a per-request dataloader (`batchByIDFunc` on `object_id`), declared in `projection.yml` `virtuals`. Hidden entirely when the tenant has no accounting connection.

### 7.4 Product guide

`docs/product-guide/accounting/sync.md`, `drift.md`, `mappings.md`, and an integrations page for QuickBooks; `pnpm --filter @trenova/web guide:generate`.

---

## 8. Cross-cutting

### 8.1 Security

- Tokens encrypted with AAD bound to tenant, connection and field; never returned by GraphQL, REST, logs, audit entries or tools. Audit entries for connections use a purpose-built struct without token fields (the existing `integrationservice.UpdateConfig` audit serializes ciphertexts; G01 does not copy that).
- OAuth state single-use (`GETDEL`), 10-minute TTL, bound to user and tenant; realm uniqueness across tenants.
- Webhook: signature verified with constant-time compare before parsing; realm ids only; body cap; IP rate limit.
- Outbound: fixed vendor hosts only, allowlisted.
- Every GraphQL resolver reaches `requirePermission` (authzlint); every repository query tenant-scoped; integration tests prove cross-tenant reads and writes fail for each new repository.
- Payload JSONB contains customer names and addresses already visible to the same tenant; it is retained 90 days after `Synced`, then cleared (hash kept).

### 8.2 Performance

- Enqueue is one indexed insert inside an existing transaction plus one cached connection lookup.
- Dispatcher claims with `FOR UPDATE SKIP LOCKED`, batches reference lookups, and uses QuickBooks batch requests where the entity allows.
- Sync ledger list uses connection pagination and gates `COUNT` on `IncludeTotalCount`; the status line uses a dataloader, never a per-row query.

### 8.3 Money and currency

Decimal end to end (`shared/decimalutils`, `shared/money`). A document whose currency differs from the provider's home currency is `Blocked` with category `Currency` unless the provider has multicurrency enabled, in which case the document carries its currency and the exchange rate Trenova used.

### 8.4 Periods

- Outbound: a document dated on or before the provider's books-closed date is `Blocked(ClosedPeriod)`. The Watchtower item offers two fixes: re-date to the first open day (only when Trenova's `ClosedPeriodPostingPolicy` is `PostToNextOpen`, so both sides follow one rule) or ask a QuickBooks admin to move the closing date, then retry.
- Inbound: a payment from QuickBooks dated in a Trenova period that is not `Open` or `Locked` is not applied; it becomes a proposal naming the period (F7).

---

## 9. Milestones

Each milestone is a vertical slice with schema, services, API, UI, tools, events, tests and docs, passes the full gate, and ships as its own PR.

| # | Milestone | Usable result |
|---|---|---|
| M1 | **Connect and health** | Connect and disconnect a QuickBooks company; see its health on the integrations page; Watchtower raises revoked or expiring tokens. Includes `shared/quickbooks` auth and company endpoints, `accounting_connections`, OAuth flow, token refresh job, webhook endpoint (verify and record), permissions, `get_accounting_sync_status`, `pause`/`resume`, event `connection_degraded`, Watchtower source, subject type `AccountingConnection`. Confirms the §1.3 open facts against Intuit's reference. |
| M2 | **Reference data and mapping** | Pull the chart of accounts, items, customers, vendors, terms; review confident proposals in the wizard; edit mappings; create missing items in QuickBooks. `accounting_reference_objects`, `accounting_mappings`, scorer, `TaskAccountingMapping`, mapping tools, mappings page. |
| M3 | **Receivables out** | Posted invoices, credit memos (including adjustment memos), debit memos, customer payments with short pay, payment reversals and customers reach QuickBooks within a minute, with a link back. Outbox, enqueue points, dispatcher, safety net, backfill with resumable cursor, sync ledger page, record status lines, retry/skip tools, `sync_failed`/`sync_blocked` events, Watchtower items, report catalog and **Sync exceptions by week**, **Books Keeper** template (sync tools). The brief's first three acceptance criteria and the idempotency replay test are met here. |
| M4 | **Payables out** | Carrier settlements as Bills to carrier vendors, voids, and `MarkPaid` as BillPayments. Vendor mapping and creation. Driver settlements per decision D5. |
| M5 | **Changes in and drift** | CDC poller and webhook trigger, inbound payments per policy, nightly drift reconcile, drift page, `resolve_accounting_drift`/`dismiss_accounting_drift` with Simulate, `drift_detected` event, **Open drift by customer**, Books Keeper weekly note. Meets the brief's drift acceptance criterion. |
| M6 | **Ledger mode** | "Trenova is the ledger": detailed or daily-summary journal entries for every posted journal, account mapping for all accounts with activity, mode step in the wizard, trial-balance drift per account. Depends on decision D4 for Manual-mode tenants. |
| M7 | **Xero** | Adapter on the same ports (OAuth2 with PKCE, tenant id instead of realm, Xero webhooks with intent-to-receive), catalog card, wizard reuse, fixture contract tests. |
| M8 | **Business Central** | Adapter (Entra ID OAuth, company selection step, API v2.0, webhooks subscriptions with renewal job). |
| M9 | **NetSuite** | Adapter (token-based auth with per-tenant consumer and token keys stored encrypted, SuiteTalk REST, saved-search-based change polling since NetSuite has no CDC equivalent). |

Brief acceptance criteria map: connect and push within a minute (M3), drift with editor and one-click fix both ways (M5), revoke shows Disconnected within one poll with nothing lost (M1 health plus M3 outbox holding records while `Revoked`), duplicate pushes impossible (M3 test), whole flow drivable from Desk (M3 onward, complete at M5).

### 9.1 As built: M1

M1 shipped the connect-and-health slice with four departures from the table above, each moved to the milestone where it first has something to act on:

- **`pause_accounting_sync` / `resume_accounting_sync` move to M3.** M1 sends nothing to QuickBooks, so a pause would hold nothing. They arrive with the first outbound push, together with `paused_at`, `paused_by_id` and `paused_reason`.
- **`setup_step` moves to M2.** M1's wizard has two steps, Connect and Review company. Its position comes from the connection's own status and from the `setup=connected` flag the callback page adds on return, so nothing needs storing. M2 adds the Map step, and with it the column and the resume.
- **`check_accounting_connection` was added.** It is an `AutoExecute`, `EgressInternal` tool that lets an agent re-test the link right away rather than wait for the fifteen-minute check.
- **The §1.3 open facts are handled, not confirmed.** The webhook verifier accepts both the CloudEvents envelope and the legacy `eventNotifications` shape, under the `intuit-signature` HMAC. PKCE is not used: Trenova is a confidential client, and the state is single-use, hashed and bound to the person. Both need checking against a live Intuit sandbox app, which depends on D6.

### 9.2 M2 design, pinned to the code

The survey before M2 found three facts that change §3.3 and §4.8:

- **Trenova posts with default accounts, not per-line accounts.** An invoice posts one entry, from `AccountingControl.DefaultARAccountID` to `DefaultRevenueAccountID`. Invoice lines (`Freight`, `Accessorial`, `Memo`), accessorial charges and fuel surcharges carry no GL account. Customer payments use `DefaultCashAccountID`, `DefaultUnappliedCashAccountID` and `DefaultWriteOffAccountID`. So QuickBooks accounts map to these **roles**, and QuickBooks items map to line types and accessorial charges. Mapping individual GL accounts belongs to ledger mode (M6).
- **Customers have no email or legal-name field.** Matching uses name, DOT/MC number, address and postal code. Carriers have name, DBA, DOT, MC and SCAC.
- **Payment terms are a fixed string enum** (`Net10` to `Net90`, `DueOnReceipt`), and so are payment methods (`ACH`, `Check`, `Wire`, `Card`, `Cash`, `Other`).

**Mapping targets.** `accounting_mappings.target_type` and the column that identifies the Trenova side:

| `target_type` | Trenova side | QuickBooks kind | First used |
|---|---|---|---|
| `AccountRole` | `trenova_key`: `ARAccount`, `RevenueAccount`, `DepositAccount`, `WriteOffAccount`, `APAccount`, `PurchasedTransportationAccount` | Account | M3, M4 |
| `LineType` | `trenova_key`: `Freight`, `Memo` | Item | M3 |
| `AccessorialCharge` | `trenova_object_id` | Item | M3 |
| `ItemRole` | `trenova_key`: `ShortPayWriteOff` | Item | M3 |
| `Customer` | `trenova_object_id` | Customer | M3 |
| `Carrier` | `trenova_object_id` | Vendor | M4 |
| `PaymentTerm` | `trenova_key`: the enum value | Term | M3 |
| `PaymentMethod` | `trenova_key`: the enum value | PaymentMethod | M3 |

**States.** Every target has a row, so gaps are visible:
- `Unmatched`: no candidate.
- `Proposed`: the scorer or the model chose one.
- `Confirmed`: a person or an approved agent action chose it. Only `Confirmed` rows reach payload builders.

Rejecting a proposal returns the row to `Unmatched` and remembers the rejected id in `signals`, so it is never proposed again for that target. Re-scoring never touches `Confirmed` rows, or rows a person set by hand. A confirmed mapping whose QuickBooks record has gone inactive or been removed is flagged as needing attention; it is never changed on its own.

**Eligibility.** Only active records are proposed. Category and Group items are never proposed, because they cannot appear on a document. Account roles filter by type:
- AR: `Accounts Receivable`.
- Revenue: `Income`.
- Deposit: `Bank` or Undeposited Funds.
- Write-off: `Expense` or `Other Expense`.
- AP: `Accounts Payable`.
- Purchased transportation: `Cost of Goods Sold` or `Expense`.

**Scorer.** This is deterministic, uses `shared/stringutils` similarity, and records the reason sentence and signals for every proposal.
- **Account role:** the exact account number (Trenova `AccountCode` against `AcctNum`) scores 0.99. Next is the name of the Trenova account behind the role against the QuickBooks name and fully qualified name. With no Trenova account set, the role's synonyms are used instead (for example "Undeposited Funds" for the deposit role). The only eligible account of its type scores 0.96, but only for the roles whose account type is their own (accounts receivable, accounts payable, and the deposit account's bank type). An expense or income account is never proposed just because it is the only one.
- **Items:** the accessorial code against `Sku` or name scores 0.99 when equal. Otherwise the description is compared with the name. Line types and the item role use synonym lists; `Freight`, for example, uses "Freight", "Linehaul" and "Line haul".
- **Customers and vendors:** equal company names, legal suffixes stripped, score 0.97. Otherwise company-name similarity is used, and an equal postal code adds 0.02. For vendors, an MC or DOT number found in the vendor's account number scores 0.99. Candidates are narrowed by a token index first, so thousands of customers do not become millions of comparisons.
- **Terms:** equal due days score 0.98. `DueOnReceipt` matches zero days.
- **Payment methods:** a synonym table, for example `Card` matches "Credit Card", "Visa" and "Mastercard".
- **Bands:** 0.95 and above is pre-checked in the review. 0.70 to 0.95 is shown with its reason. Below 0.70, the proposal is shown as "no confident match" with the top three candidates.

**Model pass.** `TaskAccountingMapping` runs inside the reference refresh activity on Temporal, calling `CompletionService` directly.
- It is only for targets below 0.70 that have candidates, at most 25 targets per call and 100 per refresh. It sends names and types only, never balances or amounts.
- Its answer is accepted only if it names a supplied candidate. Accepted answers are capped at 0.90, so a model's pick is never pre-checked.
- With no provider configured, or on the last failed attempt, the deterministic result stands.
- Usage is attributed to feature `AccountingMapping` and subject `AccountingConnection`.

**Setup step.** `setup_step` is `Mappings` after connecting and `Complete` once a person finishes the review. M3 inserts `StartDate` between them.
- The review cannot finish while any required mapping is unconfirmed. The required set is the AR, revenue and deposit roles and the `Freight` line item.
- Customers, vendors and accessorial items can be confirmed later, or created on demand when M3 first needs them.

**Create in QuickBooks.** Items, customers and vendors can be created.
- Every item is created with the confirmed revenue role's account as its income account, so an item cannot be created before that role is mapped.
- The `requestid` is derived from the connection, the mapping and its version, so a retry replays rather than duplicating. A duplicate-name answer offers the existing record instead.
- Accounts, terms and payment methods are never created from Trenova: they are the bookkeeper's.

**Refresh.** `RefreshAccountingReferenceWorkflow` runs:
- on connect;
- daily, for every active connection;
- on demand.

One workflow per connection (`accounting-reference:<connectionID>`) deduplicates concurrent requests. A record not returned by a complete pull is marked removed.

**Deferred to M3, with the records they depend on.** Usage counts and the remap guard (`acknowledgeHistory`) need `accounting_sync_records`. So do the sync start date and backfill.


---

## 10. Testing

- **Unit**: payload builders with golden JSON per document type and edge case (negative credit memo totals, short pay, multi-shipment consolidated invoice, split bill), error classifier per provider fault, mapping scorer with a fixture chart of accounts, retry schedule, token refresh race (two refreshers, one wins, the other reads the new token).
- **Contract**: `shared/quickbooks` against recorded fixtures served by `httptest`, including 401 then refresh, 429 with `Retry-After`, stale `SyncToken`, duplicate `DocNumber`, closed books.
- **Idempotency**: replay the same posting event twice → one record; replay the same record push twice → one provider call observed and the second answered by `requestid` replay in the fixture server.
- **Integration** (`//go:build integration`, `seedtest`): every repository, cross-tenant isolation per table, enqueue inside a rolled-back transaction leaves no record, `SKIP LOCKED` claiming under concurrency, realm uniqueness constraint.
- **Workflow**: Temporal test environment for drain, backfill resume after a simulated crash, and continue-as-new.
- **Coverage guards**: `TestEveryPostingPathEnqueues`; existing agent tool contract, permission, tenant-scope, target, catalog, eval and snapshot tests; authzlint; route registry; `enum_constraints_test.go`.
- **Client**: Vitest for the wizard (resume at step, OAuth callback page errors), mapping review keyboard flow, status line states; each written red first per `client/apps/web/CLAUDE.md`.
- **Sandbox**: `task test-qbo-sandbox`, skipped unless `TRENOVA_QBO_SANDBOX_*` credentials are set, runs the brief's acceptance criteria end to end against an Intuit sandbox company. Run manually before each milestone PR and recorded in the PR.

## 11. Gates per PR

`task lint`, `task test`, `task test-integration`, `task gqlgen`, `task gqlschema-diff`, `task generate-columns`, `task generate-reportcatalog`, `task sqlite-convert`, `go generate ./internal/api/graphql/projection/...`, `go generate ./internal/core/services/agenttoolpolicy/safetydoc/...`, `task i18n-check` (repo root); from `client/`: `pnpm --filter @trenova/graphql codegen`, `pnpm lint`, `pnpm typecheck`, `pnpm lint:design`, `pnpm test`, `pnpm --filter @trenova/web guide:generate`. `docs/engineering/generated-artifacts.md` read before each schema change.

---

## 12. Decisions for Eric

All seven were decided on 2026-09-24: Eric accepted every recommendation below.

| # | Decision | Recommendation |
|---|---|---|
| D1 | (From the brief) Does ledger mode ship in milestone one or after document mode? | After: M6. Document mode is what QuickBooks shops expect, and ledger mode depends on D4. |
| D2 | Pushes to the tenant's own books are classified `EgressInternal` (runtime ceiling AutoExecute), so `retry_accounting_sync` can run on its own as the brief asks. The alternative, `EgressExternalRecipient`, caps every sync tool at Ask first. | Internal. The books belong to the tenant; the tool re-sends a record Trenova already decided to send, and it is idempotent. |
| D3 | Payments recorded in QuickBooks (a customer pays in QuickBooks) flow back into Trenova AR by `inbound_payment_policy`. Default? | `Propose` for new connections: a person confirms the first weeks, then switches to `Apply`. |
| D4 | Ledger mode for tenants in `Manual` posting mode (F1): their subledger journals never reach `Posted`, so there is nothing to push. | Ledger mode requires `Automatic` posting and says so in the mode step. Separately, file the Manual-mode posting gap (and F2, adjustment credit memos without journals) as its own fix outside G01. |
| D5 | Driver settlements: owner-operators as Bills to a vendor per driver (1099 flow), company drivers excluded because payroll belongs to the payroll system. | Yes, in M4, behind a per-connection toggle. |
| D6 | Intuit app: production keys need an Intuit developer account and Intuit's app assessment before any customer can connect. | Start the assessment now; it has lead time and does not block M1 to M3 on sandbox. |
| D7 | One QuickBooks company per Trenova tenant, enforced across tenants. | Yes. The alternative lets one company receive a document twice. |

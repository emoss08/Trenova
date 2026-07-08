# EDI Gap-Closure Plan — Phase 2

**Phase 1 (shipped):** AS2 transport (crypto core, outbound, inbound + MDN, UI), 210 inbound parser with carrier-invoice reconciliation, EDI ops dashboard + dead-letter/quarantine alerting, and full test-case CRUD + preview all landed on branch `add-graphql-tractor-filter-support`. This document supersedes the Phase 1 plan with the follow-up work found in the post-implementation review.

Phase 2 is organized by severity. **WS0 is mandatory before this work is considered done** — it fixes three verified defects in the just-shipped code. WS1–WS6 are the remaining functional, UX, and enterprise-standard gaps.

Conventions (unchanged, enforce throughout): hexagonal layering (domain in `core/`, adapters in `infrastructure/`), Bun ORM, `sonic` for JSON, Ozzo validation, fx DI, no comments in Go, utilities in `shared/`, `errortypes` for structured errors. Frontend: named exports, `useWatch` never `watch()`, autocomplete fields for entity refs, no colored left-border accents on cards, badges from `client/src/components/status-badge.tsx`, GraphQL for list reads + REST for detail/mutations. Regenerate GraphQL (gqlgen + client codegen) whenever `internal/api/graphql/schema/*.graphqls` changes.

---

## WS0 — Critical fixes (verified defects, do first)

### 0.1 — BLOCKER: outbound AS2 is never wired into the dispatcher
**Verified.** `services/tms/internal/core/services/editransport/module.go:8-24` registers only `NewSFTPTransport` and `NewVANTransport` into the `group:"edi_transports"` fx group. `NewAS2Transport` exists (`editransport/as2.go:128`) with a valid `Method()` returning `ConnectionMethodAS2`, but is never provided — so the dispatcher's method map (`dispatcher.go:22-31`) has no AS2 entry. Yet `ediservice/delivery.go:26` lists `ConnectionMethodAS2` in `deliverableMethods`. Every outbound AS2 send therefore fails at `dispatcher.go:39-41` with *"EDI delivery is not supported for connection method AS2."*

**Fix:** add the AS2 transport to the fx group in `module.go`:
```go
fx.Annotate(
    NewAS2Transport,
    fx.As(new(services.EDITransport)),
    fx.ResultTags(`group:"edi_transports"`),
),
```
If `NewAS2Transport` needs deps (HTTP client, logger, secrets decryptor) that SFTP/VAN don't, confirm its constructor signature and provide them.

**Test (the gap that let this through):** add a DI-level test that builds the transport module and asserts the dispatcher resolves a transport for every method in `deliverableMethods`. The existing `as2_test.go` only tests the transport in isolation.

### 0.2 — SECURITY: inbound AS2 receiver accepts unsigned/unencrypted payloads
**Verified.** `ediinboundservice/as2.go:103-109` calls `as2.ParseMessage` with `DecryptionCertificate`/`DecryptionKey`/`PartnerCertificate`/`MICAlgorithm`/`TransferEncoding` but **never sets `RequireSignature` or `RequireEncryption`** (the library supports both — `shared/as2/message.go:191-192,314-319`). The receiver is a public route (`/api/v1/edi/as2/inbound/`, mounted under `setupPublicRoutes`, `router.go:485`) authenticated only by matching `AS2-From`/`AS2-To` against active profiles — and AS2 IDs are not secret. Anyone who knows the two IDs can POST an unsigned, unencrypted X12 payload and have it staged and processed into transfers/invoices.

**Fix:**
1. Add per-partner inbound-security config to the AS2 profile (e.g. `RequireSignedInbound`, `RequireEncryptedInbound` on the communication-profile `Config`), defaulting to **true** when a partner signing/encryption cert is present.
2. Set `RequireSignature`/`RequireEncryption` in `ParseMessageOptions` (`ediinboundservice/as2.go:103`) from that config. Fail closed: when certs are configured, reject unsigned/unencrypted with a negative MDN rather than accepting.
3. Surface the toggles in the AS2 transport tab (`client/src/routes/edi/_components/panel/edi-communication-profile-fields.tsx`).

**Test:** post an unsigned body to the receiver against a profile with a partner cert → expect rejection + negative MDN; signed+encrypted → accepted.

### 0.3 — CORRECTNESS: AES-192 silently downgraded to AES-256
**Verified.** `shared/as2/smime.go:163` collapses `EncryptionAlgorithmAES192CBC` and `AES256CBC` into the same `pkcs7.EncryptionAlgorithmAES256CBC` branch. The validator advertises `aes192-cbc` as a valid negotiable algorithm (`ediservice/validator.go:488`), so a partner negotiating AES-192 receives AES-256-encrypted data it cannot decrypt.

**Fix:** either implement true AES-192 (if `smallstep/pkcs7` exposes it) or **remove `aes192-cbc`** from the validator's allowed set and the UI algorithm options (`edi-schemas.ts` encryption options) so it can't be selected/negotiated. Removal is the safer v1.

### 0.4 — HARDENING: plaintext-secret fallback
`ediservice/profiles.go:284-286` returns the secret value **unencrypted** when `s.encryption == nil`. This is a silent-plaintext-storage foot-gun. **Fix:** fail closed — return an error if the encryption service is unwired rather than persisting plaintext.

### 0.5 — FEATURE DEBT: dangling validation profile
`EDIPartner.DefaultValidationProfileID` (`domain/edi/partner.go:41`) is persisted (repo `edipartner.go:238`) and accepted at the handler (`handler.go:52,812`) but **no `EDIValidationProfile` type/repo/service exists and nothing reads it**. Decide: (a) implement `EDIValidationProfile` (validation rule set applied during generation/parse), or (b) remove the column, handler field, and repo mapping. Pick (b) unless validation profiles are on the near-term roadmap — do not leave a field that collects an ID nothing consumes.

**WS0 acceptance:** outbound AS2 delivers through the DI graph; the inbound receiver rejects spoofed unsigned payloads when certs are configured; no algorithm can be negotiated that we can't honor; secrets never persist in plaintext; no dangling validation-profile field.

---

## WS1 — Backend hardening (observability, retry config, retention)

### 1.1 — EDI observability (high)
No Prometheus/OTel anywhere in `ediservice`/`editransport`/`ediinboundservice`/`edijobs` — only zap logging. The metrics framework exists at `internal/infrastructure/observability/metrics/` (has `http.go`, `database.go`, `temporal.go`, etc.) but **no `edi.go`**.

- Add `observability/metrics/edi.go` with: delivery latency histogram, MDN round-trip latency, ack latency, per-partner + per-transaction-set counters (sent/failed/dead-lettered/received/quarantined), inbound parse duration.
- Instrument the send path (`ediservice/delivery.go`), the AS2 transport (`editransport/as2.go`), the inbound pipeline (`ediinboundservice/service.go`), and the Temporal activities (`edijobs`).
- Add OTel spans around deliver / receive / parse so a single document's lifecycle is traceable. Follow the `golang-observability` skill.
- Add an EDI health check (mailbox reachability, dead-letter backlog threshold).

### 1.2 — Per-partner retry/backoff config (med)
Retry is hardcoded in `edijobs/workflow.go:24-33` (6 attempts, 30s initial, ×2, 15m max) for all partners. Add optional retry-policy fields to the partner or communication profile (max attempts, initial interval, max interval, dead-letter-after), fall back to the current defaults, and thread them into the Temporal `RetryPolicy` when the delivery workflow is started from `ediservice/delivery.go`.

### 1.3 — Message/file retention & purge (med-high)
`edi_inbound_files.RawContent` and `edi_messages.raw_x12` store full payloads (with PII) indefinitely; the existing `datarententionrepository` has zero EDI coverage.
- Add EDI resources to the retention system with configurable windows (per org).
- Add a purge/archival job (Temporal scheduled) that offloads or deletes raw payloads past the window while retaining metadata + audit rows.
- Add a PII note to the operations guide (raw X12 contains addresses/names).

---

## WS2 — Bulk actions (frontend, highest operator win)

The `DataTable` already supports `enableRowSelection` + dock actions (`client/src/components/data-table/data-table.tsx:543`), but no EDI workspace opts in (`edi-table.tsx:50-187`). All remediation is one-record-at-a-time. After an outage (e.g. 300 dead-lettered messages) there is no bulk path.

- **Messages:** bulk **Retry Delivery** for selected Queued/Failed/DeadLettered rows.
- **Inbound files:** bulk **Reprocess** for selected Quarantined/PartiallyProcessed rows.
- **Inbound transfers:** bulk **Approve/Reject** (respecting the per-row actionable gate).

Implementation: enable `enableRowSelection` + `dockActions` in `MessagesWorkspace`, `InboundFilesWorkspace`, and `TransfersWorkspace` (`edi-table.tsx`). Add backend batch endpoints (or loop the existing single-item endpoints server-side behind one call) — prefer a real batch endpoint per entity to avoid N round-trips. Reuse the existing single-item mutations' invalidation (`edi-panel-invalidation.ts`). Confirm permission gating on the batch endpoints matches the single-item ops.

**Acceptance:** an operator can select N rows and retry/reprocess/approve in one action, with a progress/result toast reporting successes and failures.

---

## WS3 — Frontend UX completeness

### 3.1 — Test-case pass/fail verdict (high value, cheap)
The panel captures `expectedWarnings`/`expectedErrors` (`panel/edi-test-case-panel.tsx:337-354`) but preview only shows raw inspector diagnostics (`:166-181`) — nothing compares actual vs expected. Add a green/red **verdict badge** after preview: compare the inspector's actual diagnostics against the expected sets and render Pass/Fail with the diff (unexpected + missing). This turns the feature from "eyeball it" into real certification.

### 3.2 — Surface alerts outside the dashboard (high value, cheap)
Dead-letter/quarantine alerts exist only on `/edi/overview`. Add a **nav badge** (unread/attention count) on the EDI nav item in `client/src/config/navigation.config.ts`, fed by the existing `EdiSummary` query (dead-lettered + quarantined + overdue-ack counts). Optionally push a toast/notification-center entry when the count increases. No new backend needed.

### 3.3 — Live queues (med)
Only the dashboard polls (30s, `overview/use-edi-summary.ts:9`). Add a light `refetchInterval` to the transfer and message tables so queues update themselves, or a manual auto-refresh toggle — the module currently promises "live" on the landing page but is static everywhere else.

### 3.4 — AS2 cert UX polish (med)
In `panel/edi-communication-profile-fields.tsx`: replace the paste-only cert `TextareaField`s (`:128-153`) with file upload (`.pem/.crt/.p12`) reusing the app's upload component, parse and show **fingerprint + expiry** ("expires in N days"), and switch the AS2 private key input (`:411-416`) from a plaintext `TextareaField` to the masked `SensitiveField` used for the AS2 password. Add a **"Test connection / send test MDN"** button for AS2 (and SFTP/VAN) so partners aren't configured blind.

---

## WS4 — Per-partner SLA & dashboard analytics (full-stack, high enterprise value)

The summary is org-wide aggregate only (`ediservice/summary.go`); the dashboard is instantaneous counts with no trends, no time range, no partner breakdown. This is a headline feature in McLeod/MercuryGate/SPS.

### Backend
- Extend the summary (or add `ediPartnerScorecards` query) with **per-partner** rollups: sent/failed/dead-lettered/received counts, delivery success rate, ack turnaround (avg/p95), overdue-ack count, oldest-pending age. Keep queries grouped/efficient (add `GroupBy(partner)` variants to the count methods in `edimessage.go`, mirroring the existing grouped `COUNT(*)`).
- Wire the already-existing but unused `sinceHours` variable end-to-end (`client/src/lib/queries/edi.ts:81` hard-codes `null`) so a time range actually filters.
- Add a time-bucketed volume/success-rate series query for trend charts.

### Frontend
- Add a **time-range selector** to the dashboard (feeds `sinceHours`).
- Add **trend charts** (volume over time, delivery success rate, ack-turnaround distribution) — follow the `dataviz` skill for palette/accessibility, theme-aware.
- Add a **per-partner scorecard table/section** with SLA status and aging buckets (>4h / >24h), each row drilling into that partner's filtered messages.
- Bump `InfoTile` value typography (`panel/edi-panel-primitives.tsx:74-76`) so KPI numbers are scannable (value large, label secondary), and add severity emphasis when `value > 0` (numeral color/weight — not a left-border accent) so a dead-letter count of 500 doesn't look identical to 0.
- Give the attention feed a "view all" affordance + count when server-capped.

---

## WS5 — Partner onboarding wizard / readiness (frontend, high enterprise value)

Today onboarding is disjointed: create partner → separately navigate to Communication Profiles → Mapping Profiles → Designer/templates → Test Cases, with nothing sequencing the steps or tracking completeness. Build a guided flow:

- A **partner readiness view** on the partner edit panel (or a dedicated wizard route) showing a checklist: partner details ✓, communication profile configured ✓, mappings defined ✓, active template/document profile per direction ✓, at least one passing test case ✓.
- Each step deep-links to the relevant creator (reuse existing panels; don't rebuild).
- Show a partner **readiness state** column/badge on the partners table and optionally gate "activation" (enabling inbound/outbound) until the checklist passes.
- Reuse the nice existing internal-pair auto-fill (`edi-partner-panel.tsx:101-127`) and the `PendingConnectionsPanel` handshake inbox as the first step of the internal flow.

This is the single highest-value net-new UX for a complex domain; it's assembly of existing panels + a completeness model, not new primitives.

---

## WS6 — Standards completeness (backend roadmap, sequence by demand)

Independent items; pull in as partner requirements dictate.

- **TA1 interchange acknowledgment (med):** no TA1 anywhere (`grep TA1` = 0). VANs/large partners often require envelope-level accept/reject. Add TA1 generation on inbound interchange validation + parsing of inbound TA1 for our outbound.
- **Detailed 999 IK/CTX reporting (med):** current 999 looks limited to accept/reject; add AK3/AK4/IK3/IK4/CTX-level error reporting for certification.
- **ISA05/07 qualifiers as first-class envelope fields (med):** `X12EnvelopeSettings` has sender/receiver IDs but no qualifier fields; today they can only be hardcoded as template constants (`edix12/renderer.go:164-166`). Add `InterchangeSenderQualifier`/`ReceiverQualifier` to the envelope config + runtime values, and surface in the document-profile envelope editor.
- **Outbound batching (med):** renderer builds one interchange per message, send-immediately. Add optional per-partner send windows, functional-group batching (multiple STs under one GS/ISA), and per-partner throttle.
- **Replay & control-number reset (med):** allow resend of a delivered historical message and a control-number sequence reset (the `edicontrolnumber` repo exposes only `AllocateControlNumbers` — add an audited reset).
- **Transaction sets 810 & 856 (med-high):** 810 outbound customer freight invoice (pairs with billing — auto-invoice-on-delivery) and 856 ASN are the notable absent sets. 820 remittance matching pairs with the existing 210 reconciliation. 850/855/940/945/404/410 are lower priority for an asset/brokerage TMS.
- **Transport breadth (med-high):** add HTTP/REST webhook, FTPS, and email/mailbox transports (each implements `services.EDITransport`, registered in the same `group:"edi_transports"` fx group — see WS0.1 for the pattern that must be correct this time).

---

## Cross-cutting

- **DI wiring tests:** the AS2 blocker (WS0.1) proves transports need a graph-level test. Add one assertion that every `deliverableMethods` entry resolves in the dispatcher — it will also protect the WS6 transports.
- **GraphQL codegen:** WS4 (scorecards/trends) and any WS6 config additions touch `internal/api/graphql/schema/edi.graphqls` → regenerate Go resolvers + client documents (`client/src/graphql/generated/*`, `persisted-documents.json`). Keep list reads on GraphQL, writes on REST.
- **Docs:** update `docs/operations-guides/edi-module-feature-inventory.md` as each item flips from missing to implemented, and add a retention/PII section (WS1.3) and an AS2 inbound-security section (WS0.2).
- **Do not regress the strong areas:** ack reconciliation + overdue workflow, mapping cross-refs + Starlark, race-safe control numbers (`SELECT FOR UPDATE`), dual-layer inbound dedup, envelope encryption (KMS + AAD), audit coverage, and the dashboard's one-click deep-link into the exact edit drawer.

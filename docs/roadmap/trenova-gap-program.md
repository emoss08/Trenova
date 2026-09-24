# Trenova Gap Program: work briefs for Claude Code agents

Source analysis: [Trenova vs DataTruck, Rose Rocket, Alvys and McLeod](https://claude.ai/artifact/Kzgmuxeri1agVefxDezQyD) (2026-09-22).
Owner: Eric (emoss08). Repository: `emoss08/Trenova`.

This document turns the competitive gaps into work that Claude Code agents can pick up one brief at a time. It has four shared parts that every agent must read, and then twenty briefs in priority order. The briefs are deliberately opinionated: each names what the market ships today and then a higher bar that Trenova ships instead.

> **Trenova does not meet the standard. Trenova sets it.** Innovation and creativity are the core principles. If a brief's result would look at home in DataTruck, Rose Rocket, Alvys or McLeod, it is not done.

---

## How to use this document

### For Eric: launching an agent

Launch one agent per brief. Briefs in the same wave can run in parallel except where Part 4 lists a dependency. Paste this prompt, filling in the brief ID:

```text
You are implementing brief {G0X} of the Trenova Gap Program.

Read, in order and in full before writing any code:
1. docs/roadmap/trenova-gap-program.md, the Errata, Parts 1-4 and brief {G0X}
   (or the copy attached to this session). Where the Errata and a brief
   disagree, the Errata wins.
2. CLAUDE.md, AGENTS.md and client/apps/web/CLAUDE.md
3. docs/engineering/design-system.md
4. docs/engineering/generated-artifacts.md
5. Every file the brief lists under "Today in code"

Then write a plan to docs/roadmap/{G0X}-plan.md covering data model, migrations,
services, GraphQL/REST surface, AI surface (every tool, event, Watchtower source,
template and artifact from the brief), UI routes and components, and the milestone
order. Stop and surface the plan and the brief's "Decisions for Eric" before
building. After approval, build milestone by milestone, each one a vertical slice
that passes the full gate (Part 3, "Gates") and ships as its own PR.
```

### For agents: rules of engagement

1. **Read Parts 1 to 4 and your brief in full.** Your brief is the scope. Parts 1 and 2 are the bar every brief is judged against; they are not optional extras.
2. **Verify before you trust.** Paths and facts in this document were read from the code on 2026-09-22. The code moves fast. Open every file the brief names and confirm it before building on it. If the code contradicts the brief, the code wins; note the difference in your plan.
3. **Plan first, then build in vertical milestones.** Each milestone ships schema, service, API, UI, AI tools and tests together and is usable on its own. Never ship a backend without its UI, or a UI without its AI tools. No stubs, no "phase 2" placeholders inside a milestone.
4. **Ask only what the brief marks as a decision.** For anything else, pick the option that best serves Part 1, say which you picked in the PR, and keep going.
5. **Mind the merge hotspots.** Several briefs touch the same files. Keep edits to these small, append-only and rebased often:
   - `services/tms/internal/core/domain/permission/resource_gen.go` and `registry.go`
   - `services/tms/internal/core/domain/permission/agent.go` (agent allowlist)
   - `services/tms/internal/bootstrap/app.go` and the FX module lists
   - `services/tms/internal/core/domain/agent/events.go` (`knownEvents`)
   - Enum migrations (`ALTER TYPE ... ADD VALUE`): always a new migration file, never an edit to an old one
   - `client/apps/web/src/components/assistant/tool-presentation.ts` (`TOOL_TITLES`)
   - `services/tms/internal/infrastructure/database/reportcatalog/reportcatalog.yml`
6. **Never edit** generated files by hand except where Part 3 says the repo expects it (`resource_gen.go` is hand-edited; `seed_ids_gen.go`, `models_gen.go`, GraphQL client output and `buncolgen` are regenerated).

---

## Errata (verified against the code on 2026-09-24)

Every claim in this document was re-read against the repository on 2026-09-24. This section records what changed or was wrong. It overrides anything below it. Paths use `S/` for `services/tms/internal/` and `C/` for `client/apps/web/src/`.

### E1. Tools declare a policy, not individual methods

`AgentTool` and `AgentQueryTool` no longer have `PermissionResource`, `PermissionOperation`, `Reversible`, `RequiresIdempotencyKey` or `DefaultAutonomyTier` methods. Each tool has one `Policy() ToolPolicy` (`S/core/ports/services/toolpolicy.go`) whose fields carry all of it: `Kind`, `Resource`, `Operation`, `Scope`, `DefaultTier`, `MaxTier`, `Egress`, `Classify`, `Condition`, `TaintHold`, `Effect`, `Artifact`, `Reversible`, `Idempotent`, `ReadsExternal`, `Source`, `CarriesTaint`, `Rationale`.

- `AgentTool`: `Name`, `Description`, `ParamSchema`, `Policy`, `Execute(ctx, ToolExecuteParams) error`. Optional: `ToolSimulator`, `ToolValidator`, `TargetedTool`, `ToolResultReporter` (`ExecuteWithResult`, for a write whose caller needs the id it made).
- `AgentQueryTool`: `Name`, `Description`, `ParamSchema`, `Policy`, `Query(ctx, *QueryToolParams) (any, error)` (a pointer). Query tools build their policy with `readPolicy(name, readSpec{resource: ...})` in `agentquerytoolservice/policy.go`.
- Part 2 rows A3, A5, A6 and A8 therefore mean: set `DefaultTier`/`MaxTier`, `Idempotent`, `Reversible`, `Resource`/`Operation` in the policy. `guardExecute` enforces `Idempotent` by refusing a call with no key.
- Recipes 3.1 and 3.2 below are rewritten to match.

### E2. Egress classes set ceilings; money is the exception you must handle

Every policy declares `Egress` from `none`, `personal`, `internal`, `customer_visible`, `driver_visible`, `external_recipient`, `money` (`S/core/domain/agent/toolsafety.go`). `EgressClass.Ceiling()` caps `customer_visible`, `driver_visible` and `external_recipient` at `ActWithApproval`, whatever tier the tenant or earned trust sets. So Part 2's "anything sent outside the tenant defaults to Propose" is the default tier; the ceiling is already enforced by the runtime.

`money` has **no** ceiling today: it is `AutoExecute`, and the existing money tools (`match_bank_receipt`, `post_customer_payment`) declare `MaxTier: TierAutoExecute`. `TestAssess_MoneyRunsOnItsOwnUntilTheRunIsTainted` pins that behaviour: money runs on its own until the run reads outside text. Part 2's rule that money movement and payment instructions never reach `AutoExecute` is therefore **not** enforced by the runtime. Until Eric decides whether to change the global ceiling, every tool this program adds that moves money or changes payment instructions sets `MaxTier` to `agent.TierPropose` or `agent.TierActWithApproval` explicitly and has a test proving it. Credentials, security and permission tools do the same.

### E3. Gates a new tool must pass, missing from Part 3

- `agenttoolpolicy/contract_test.go`: every registered tool validates, sits in the egress class it was given (explicit map), query tools change and send nothing, tools that read outside text say where.
- `agenttoolcatalog`: `TestEveryToolDescribesItselfWellEnoughToBeChosen`, `TestEveryToolFamilyNamesRegisteredTools` (families in `families.go`).
- `agentevalgate/testdata/catalog.golden.json`: `go test ./internal/core/services/agentevalgate -run TestToolCatalogSnapshot -update`.
- Prompt snapshots: `go test ./internal/core/domain/agentdefinition -run TestPromptSnapshots -update` when a template's starter tools or instructions change.
- `docs/engineering/ai-tool-safety.md` is generated: `go generate ./internal/core/services/agenttoolpolicy/safetydoc/...`; CI fails when it drifts.
- `agentpermission_test.go`, `tenant_scope_test.go`, `target_test.go` (as Part 3 says).

### E4. Paths, names and counts

| Document says | Code says |
|---|---|
| 114 compiled tools | 134: 78 query, 52 action, 4 runtime (`find_tools`, `ask_user`, `publish_artifact`, `delegate_task`) |
| 179 permission resources | 183 |
| The agent allowlist is a single-literal return the test regex-parses | A `map[Resource]map[Operation]struct{}` read through `IsAgentAllowed`; the coverage test fails on missing and on unclaimed entries |
| `listSpec` in `listcatalog.go` | Defined in `listtools.go`; `listcatalog.go` holds the spec instances |
| `draftSpecs` in the tool service | `S/core/services/assistantservice/artifacts.go` |
| `task i18n-check` in `services/tms` | Root `Taskfile.yml`; run from the repo root |
| `pnpm lint:design` in the web app | `client/package.json` only; run from `client/` |
| Watchtower jobs fan out with `RunTenantFanOut` | `watchtowerjobs` loops tenants inside its activity; `RunTenantFanOut` (`S/core/temporaljobs/fanout.go`) is used by `aifeedbackjobs`, `briefingjobs`, `carrierintelligencejobs`, `insightjobs`, `retrievaljobs`. Use it for new per-tenant jobs. |
| `ARTIFACT_KINDS` in `artifacts-pane.tsx` | Defined in `C/components/assistant/voice/artifact-chrome.tsx`; the pane imports it. `artifactKindSchema` is in `C/types/assistant.ts`. |
| `AGENTS.md` paths under `client/src` | `client/apps/web/src` |
| GTC runs in `docker-compose-local.yml` (CLAUDE.md) | Not in the compose file |
| GTC publishes `reporting:cdc` | Only `services/gtc/config/gtc.example.yaml` (5 tables). Neither live config (`config/gtc.yaml`, `services/gtc/config/gtc.yaml`) publishes it, so the report result cache consumer receives nothing today. |

### E5. Internationalization is already built

Translations are a custom system, not a React package: catalogs in `i18n/` (en, es, zh-TW, zh-CN; `messages.en.json` is generated), the runtime in `@trenova/shared/i18n` (`useT`, `translate`, `format`), and Go extraction in `shared/cmd/i18n-extract`. The English string is the key. Write English inside `t("...")` (or an `errortypes`/ozzo message in Go) and run `task i18n`; `task i18n-check` fails on missing or orphaned entries. Adding a language is one line in `i18n/locales.json` plus `task i18n`, so G19's fr-CA needs no framework work. Part 1.3 should read "English, Spanish, Traditional and Simplified Chinese".

### E6. Per-brief corrections

- **G01.** The 14 event types are `tenant.JournalSourceEventType` (`S/core/domain/tenant/enums.go:145`) and a Postgres enum `journal_source_event_enum`; `journal_sources.source_event_type` itself is `VARCHAR(100)`. `journal_sources.idempotency_key` is unique per organization and business unit. `DefaultAPAccountID` is used by carrier settlement posting. `VendorBillPosted` and `VendorPaymentPosted` have no producer; the accounting-control validator still requires one of them in `AutoPostSourceEvents` when expense recognition is on bill post or cash disbursement. Journal lines already carry `CustomerID`, `VendorID`, `DepartmentID`, `ProjectID`, `LocationID`, `TaxCode`, `TaxAmount`.
- **G02.** `DocumentPacketRule` accepts Shipment, Trailer, Tractor and Worker but billing does not use it (it drives driver qualification files). Packet readiness must build on `S/core/services/shipmentservice/billing_readiness.go` (`GetBillingReadiness`, per-payer document requirements from the billing profile's document types, rate validation, auto mark-ready). Bank receipt matching links receipts to customer payments only, one receipt per import. The cash-flow forecast (`accountsreceivableservice.GetCashFlowForecast`) covers inflows only.
- **G03.** The integrations unique constraint is per organization, business unit and **type**, so Samsara and Motive can already coexist. The blockers are `Factory.ProviderFor` returning the first match (`S/infrastructure/telematics/factory.go`) and the single `external_id` column on `tractors`, `trailers` and `workers`. Tractors and trailers already auto-match by VIN then unit code (`telematicsservice/sweep.go`); workers are pushed to Samsara, not matched. Samsara live shares, routes, messages and addresses clients are unused.
- **G04.** `AIDocumentService` also has `SubmitRateConfirmationBackgroundExtraction`/`PollRateConfirmationBackgroundExtraction`. Email intake exists (`inboundmessage`, `inboundmessageservice`, `POST /webhooks/inbound-mail/:mailboxToken/`, Postmark and Resend). Per-field confidence, evidence excerpt and page number are stored; bounding boxes are not (Tesseract TSV coordinates are discarded in `contentbuilding.go` `parseTesseractTSV`). The `document.extracted` agent event already exists.
- **G05.** The portal has 61 methods, 32 prefixed `My`; `DashControl` has 19 settings. No portal mutation accepts a client key; `RecordStopActual` accepts `OccurredAt` but the portal does not pass it (`driverportalservice/portal.go`), so a retried arrival fails with "You've already arrived at this stop". Dash already ships `dash.webmanifest`. Signatures today are typed names on policy acknowledgements.
- **G06.** `tenderservice.MarkExhausted` records a shipment event, notifies dispatch, audits and invalidates; it publishes no agent event and offers no extension hook. The setter-injected ports on the tender service are the pattern to follow. Scoring lives in `dispatchcandidateservice/score.go`; `shared/dispatchplanner` is an assignment solver (greedy/regret plus ALNS).
- **G07.** `shared/webhooksig` verifies Svix and basic-auth signatures and has `SignSvix`. The OpenAPI spec is generated by swag plus `cmd/openapi-postprocess` (`task docs-generate`) and checked in CI.
- **G08.** Agent extensions (`docs/engineering/agent-extensions.md`, Exa web search) are the model for tenant-credentialed tools: activation gate, per-tool availability, a taint mark that caps later writes at Propose, daily limits and cost recording. `agentdefinition.Definition.Version` is an optimistic-lock counter, not a revision history.
- **G09.** `invoicesharehandler` is an authenticated internal share, not public; the public invoice download is `invoicehandler` `/billing/invoices/shared-documents/:token/download/`.
- **G10.** Tenders use `PreviewByToken`/`RespondByToken`; rate confirmations use `PreviewByToken`/`ConfirmByToken`. The rate confirmation signature is a typed name and title with no IP or user agent. Automatic carrier invoice matching is called only from EDI 210 (`ediinboundservice/tenderedcarrier.go`); emailed invoices are classified but never matched.
- **G12.** The `mfa_authenticators` table (webauthn, totp) exists with a list endpoint only.
- **G13.** Safety events belong to a worker (with an optional shipment), not a tractor or trailer. `fmcsa.Basics` data reaches carrier vetting through `Composite`, not the tenant's own fleet.
- **G14.** `AtMaintenance` is set by hand or by the `fleet_tools` agent tool and blocks dispatch through the generic "not Available" check; nothing sets or clears it automatically. `vehicle_inspections` are listed in GraphQL but drive nothing.
- **G15.** Dashboard cross-filtering is client-side (`cross-filter.ts`); saved dashboard filters are server-side.
- **G17.** `tablechangealert` (CDC triggers with conditions that only notify) is the nearest existing trigger layer.
- **G18.** LTL rating has NMFC density scales, class-from-density and deficit weight; no tariff integration.

---

## Part 1. The bar

Every brief contains a **Market bar** (the best any of the four competitors ships) and a **Trenova bar** (what we ship). The Trenova bar is the acceptance target. These principles apply to every brief even when the brief does not repeat them.

### 1.1 Be the standard

- Build the version a competitor would copy. For each feature, ask: what is the tedious part a dispatcher, biller, driver or safety manager does today in every TMS? Remove it, do not just digitize it.
- Prefer **zero-entry** over data entry. If the information exists in a document, an email, a telematics feed, a prior load or a public registry, Trenova reads it and proposes it. The human confirms; they do not type.
- Prefer **proactive** over reactive. Trenova tells the user about the problem before they go looking: through Watchtower, a notification, a Desk briefing, or an agent proposal.
- Prefer **explained** over opaque. Every computed number, score, match, ETA or recommendation shows *why* in one click: the inputs, the rule or model that produced it, and the confidence.
- Prefer **reversible** over final. Destructive or external actions show a preview (`Simulate`) and, where the domain allows, an undo or reversal path.

### 1.2 AI-native by construction

Trenova's governed agent runtime (tenant-defined agents, BYO or self-hosted model providers, autonomy tiers, shadow mode, earned trust, budgets, the decision queue and Desk) is the lead no competitor matches. Every feature in this program must extend it. The concrete contract is Part 2. The spirit:

- **Anything a person can do in the UI, an agent can do through a tool**, under the same permissions, with the same validation, and leaving the same audit trail.
- **Anything an agent needs to know, it can read through a tool**, with field sensitivity respected.
- **Anything that happens emits an event** an agent can wake on.
- **Anything that goes wrong surfaces in Watchtower** where an agent can propose the remedy.
- **Tenants can build their own agents** on top of the new tools without Trenova shipping code. Our starter templates are examples, not the ceiling.

### 1.3 The UI is the product

The design system is not decoration; it is the reason Trenova reads as a different class of product. Before writing a line of UI, read `docs/engineering/design-system.md`. Non-negotiables:

- **Page anatomy.** Always `PageLayout` with `pageHeaderProps`. A list page is `PageLayout` > `DataTableLazyComponent` > table with no wrapper `div`. Create buttons read "New {thing}". Figures use `KpiStrip`/`KpiStripItem`, titled blocks `SectionPanel`, read-only label/value `DescriptionList`, callouts `<Alert size="sm">`, dialogs take `size`. Forms use `FormSection` and `FormSaveDock`.
- **Tokens only.** Colour is a tone or a categorical accent. Statuses declare a lifecycle phase and the tone follows. No raw palette classes, hex, arbitrary font sizes, hand-rolled focus rings, or shadows on anything that sits in the page. `pnpm lint:design` must pass.
- **Machine suggestions** are marked with `AssistMark` from `@trenova/shared/components/ui/assist-mark`. Never `Sparkles`, `WandSparkles` or `Wand2`. Agent presence uses the existing `AgentGutter`, `AgentTile` and `WorkingDot` primitives.
- **Keyboard first.** Every list supports `j`/`k` navigation and the actions the decision queue already established (`x` select, `a` approve, `r` reject, `m` more, `Shift+A` bulk). Every page is reachable from the command palette. Every primary action has a shortcut shown in its tooltip.
- **Every state is designed.** Empty (with the one action that fills it, and an agent that can fill it for you), loading (`ui-shimmer` skeletons in the real layout), partial, error (what failed, why, and the fix), offline, and permission-denied. A blank table is a defect.
- **Explainability UI.** Scores, matches, ETAs, confidences and AI extractions have an inline "why" affordance that opens the inputs and the reasoning.
- **Bulk and undo.** Lists support multi-select with bulk actions. Reversible actions offer undo in the toast.
- **Accessibility.** WCAG 2.2 AA: labels, focus order, `aria-live` for async results, 44px touch targets on the driver app, reduced-motion respected, contrast enforced by the token linter.
- **Internationalization.** All user-facing text is written in English inside `t("...")` from `@trenova/shared/i18n/use-t` (Go: `errortypes`/ozzo messages), then `task i18n` extracts it and `task i18n-check` (repo root) gates CI. Trenova ships en, es, zh-TW and zh-CN; the Canada brief adds fr-CA. See `i18n/README.md` and Errata E5.
- **Performance budgets.** List pages render their first row under 1 s on a cold load against seed data; interactions respond under 100 ms; no N+1 GraphQL resolvers (use the dataloaders); connection queries gate `COUNT` on `IncludeTotalCount`.

### 1.4 Integrations heal themselves

Competitors win by listing logos. Trenova wins by having integrations that do not break silently.

- Every integration has a **health model** (connected, degraded, failing, disconnected) visible on the integrations page and as a Watchtower source.
- Every inbound and outbound sync writes a **sync ledger** row (what, when, result, external ID, payload hash) so any record can show "last synced" and "why not synced".
- Failures retry with backoff, then land in a **dead-letter queue** with one-click replay and bulk replay. Backfills are resumable.
- Credential expiry, rate-limit exhaustion and schema drift are detected and **raised before they cause data loss**, with an agent proposal to fix what can be fixed (re-map a field, refresh a token, re-run a window).
- Secrets are encrypted through `integrationservice/secrets.go` and never logged, echoed to the client or returned by a read tool.

### 1.5 Honest data

- Never show a number the system cannot defend. If a value is estimated, say so and show the basis (the existing `ArrivalEstimate.Verdict`/`Basis` pattern is the model).
- Every AI-extracted field carries a confidence and a source region. Low-confidence fields are visually distinct and block auto-apply.
- Money is `decimal` end to end (`shared/decimalutils`, `shared/money`, GraphQL `Decimal`). Timestamps are Unix seconds (GraphQL `Timestamp`).

### 1.6 Production-grade, secure, tenant-safe

- Tenant isolation on every query (`tenantOf(params)`/`tenantFrom` in tools, org and business-unit scoping in repositories). Tests prove cross-tenant reads fail.
- Least privilege: a new permission resource for every new domain; field sensitivity classified in the permission registry.
- Every external call goes through `shared/httpsafe` (SSRF protection) unless it targets a fixed vendor host.
- Every public token endpoint is single-purpose, expiring, revocable, rate-limited and audited, following `tenderpublichandler` and `rateconfirmationpublichandler`.
- Follow CLAUDE.md in full: Uber Go style, no code comments, `sonic` not `encoding/json`, buncolgen column helpers, Ozzo validation, `errortypes.MultiError`, parameter structs past three or four arguments, utilities in `shared/`.

---

## Part 2. The AI-manageability contract (Definition of Done)

A brief is not done until everything below that applies to it exists, is tested, and is listed in the PR description under "AI surface". Reviewers check this list first.

| # | Requirement | Rule |
|---|---|---|
| A1 | **Read tools** | Every new entity has `get_<entity>` and `list_<entity>s` (or a `listSpec` in `listcatalog.go`), plus `search_*` where users search. Summaries and health have `insight`-style tools. Field sensitivity is enforced (`fieldaccess.go`). |
| A2 | **Action tools** | Every user-facing mutation that changes state has an `AgentTool`. One tool per intent (`approve_vendor_bill`, not `update_vendor_bill` with a status flag). |
| A3 | **Tier defaults** | `Policy().DefaultTier` follows this table, and `MaxTier` caps it (Errata E2). Tenants may raise a tier through earned trust; the default never starts higher. |
| A4 | **Simulate** | Every new action tool implements `ToolSimulator` and returns a `ToolSimulation` with a human summary and `FieldChange`s. The decision queue must render it. |
| A5 | **Target and idempotency** | Tools that act on one record implement `TargetedTool.Target` so stale proposals are rejected. Tools with external side effects set `Policy().Idempotent`. |
| A6 | **Reversible** | `Policy().Reversible` is true only when a reverse tool exists and is registered. If true, name the reverse tool in the description. |
| A7 | **Validation** | Tools that can fail business rules implement `ToolValidator` using the same validator the service uses. No second copy of rules. |
| A8 | **Allowlist** | Every tool's `Policy().Resource`/`Policy().Operation` pair is an entry in the `agentAllowedPermissions` map in `core/domain/permission/agent.go` (`IsAgentAllowed`). The coverage tests in `agenttoolservice/agentpermission_test.go` fail both ways: a tool whose pair is missing, and an entry no tool claims. `OpApprove` is never granted to an agent. |
| A9 | **Events** | Every meaningful state transition publishes an agent event (`services.PublishAgentEvent`). New kinds are added to `knownEvents`. New subject types get the enum, the migration and a resolver in `agentsubjectservice`. |
| A10 | **Watchtower** | Every condition a person should act on (a failing sync, an expiring document, an overdue item, an at-risk load) is a `WatchtowerSource` with snapshot, upsert and resolve. |
| A11 | **Agent template** | Each brief ships at least one starter template in `agentdefinition` that uses its tools. Anything that talks to people outside the tenant or moves money starts at `TierPropose` with shadow mode on. |
| A12 | **Desk** | Read tools map to Desk artifacts (`get_*` to an entity card, `list_*`/`search_*` to a table). If the domain needs a new artifact kind, add it (enum, `ck_assistant_artifacts_kind` migration, client schema, pane, payload, component, and the kinds guard test). |
| A13 | **Reporting** | Every new table the business will ask questions about is in `reportcatalog.yml` with curated fields, and each brief ships at least one canned report. |
| A14 | **AI task kinds** | Any new LLM call is a named task kind in `aiprovider` (so tenants route it to their chosen provider) and has a deterministic fallback when no provider is configured. |
| A15 | **Presentation** | Every tool has a `TOOL_TITLES` entry and is in the `agenttoolcatalog` vocabulary so tenants can find it when building agents. |
| A16 | **MCP-ready** | Tool names, descriptions and `ParamSchema` are written for an external model to use without context: precise descriptions, enums not free strings, units in field names, examples in descriptions. G07 exposes them over MCP. |

**Default tiers**

| Kind of action | Default tier |
|---|---|
| Reads | n/a (always allowed within permission) |
| Internal, reversible state change (tag, assign, snooze, draft) | `AutoExecute` allowed |
| Internal, hard to reverse (post, void, close period, delete) | `ActWithApproval` |
| Anything sent outside the tenant (email, SMS, EDI, portal publish, load board post, webhook config) | `Propose` |
| Money movement or payment instructions (payouts, factoring submission, bank details) | `Propose`, and never `AutoExecute` even through earned trust |
| Changes to credentials, security or permissions | `Propose`, never `AutoExecute` |

---

## Part 3. Engineering recipes

Paths below are relative to `services/tms/internal/` (backend, `S/`) and `client/apps/web/src/` (web client, `C/`) unless written in full. Read the pattern file before copying it.

### 3.1 Read tool (`AgentQueryTool`)

- Port: `S/core/ports/services/agentquerytool.go`: `Name`, `Description`, `ParamSchema() map[string]any`, `Policy() ToolPolicy`, `Query(ctx, *QueryToolParams) (any, error)`. The policy comes from `readPolicy(t.Name(), readSpec{resource: permission.ResourceX})` (`agentquerytoolservice/policy.go`).
- Pattern: `S/core/services/agentquerytoolservice/insight_tools.go`. Start `Query` with `guardQuery(params)` and scope with `tenantOf(params)`.
- Generic list tools: add a `listSpec` (type in `listtools.go`) instance in `listcatalog.go` rather than a hand-written tool.
- Register in `agentquerytoolservice/module.go` with `fx.ResultTags(`group:"agent_query_tools"`)`. Renamed tools go in `legacyToolNames` in `registry.go`.
- Field sensitivity: `fieldaccess.go`, classified in `S/core/domain/permission/registry.go`.
- Desk mapping: `S/core/services/assistantservice/artifacts.go` (`artifactFromObservation`).
- Vocabulary: `agenttoolcatalog/catalog.go`. Client label: `C/components/assistant/tool-presentation.ts`.

### 3.2 Action tool (`AgentTool`)

- Port: `S/core/ports/services/agenttool.go`: `Name`, `Description`, `ParamSchema`, `Policy() ToolPolicy`, `Execute(ctx, ToolExecuteParams) error`. Optional: `ToolSimulator`, `ToolValidator`, `TargetedTool`, `ToolResultReporter`.
- The policy is a literal `serviceports.ToolPolicy{Name, Kind: agent.ToolKindAction, Resource, Operation, DefaultTier, MaxTier, Egress, Reversible, Idempotent, Rationale, ...}`. Choose `Egress` honestly (Errata E2); it sets the ceiling.
- Pattern: `placeShipmentHoldTool` in `S/core/services/agenttoolservice/shipment_tools.go`. Start with `guardExecute(t, params)` (`base.go`); scope with `tenantFrom`.
- Register in `agenttoolservice/module.go` `ToolProviders()` (grouped as `group:"agent_tools"`). Tiers are in `S/core/domain/agent/enums.go`; egress classes and kinds in `toolsafety.go`.
- `proposalrecorder` and `proposalexecutor` already handle idempotency, target staleness, budgets and audit; do not reimplement them.
- Tools that send messages outward also add a `draftSpecs` entry (`S/core/services/assistantservice/artifacts.go`) so the draft is reviewable.
- Tests and generated files to update: Errata E3.

### 3.3 Agent event and subject

- `S/core/domain/agent/events.go`: add the kind constant and a `knownEvents` entry.
- Publish: `services.PublishAgentEvent(ctx, s.events, services.AgentEvent{Kind, SubjectID, TenantInfo})`.
- New subject type: enum constant, migration `ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS '<value>'`, and a resolver in `agentsubjectservice`.

### 3.4 Watchtower source

- Add a `SourceKind` in `S/core/domain/watchtower/enums.go` and to the GraphQL enum.
- Implement `WatchtowerSource{Kind(); Snapshot(ctx, tenant)}` in `S/core/services/watchtowersources/` and register it with `asSource` in its `module.go`.
- The projector upserts and resolves; reconciliation runs in `S/core/temporaljobs/watchtowerjobs`.

### 3.5 Desk artifact kind

- `S/core/domain/assistantartifact/enums.go` plus a migration updating the `ck_assistant_artifacts_kind` check.
- Client: `artifactKindSchema`, `ARTIFACT_KINDS`, `artifacts-pane.tsx`, `artifact-payloads.ts`, a new `*-artifact.tsx`, and the guard test `assistant-artifact-kinds.test.ts`.

### 3.6 Agent template

- `S/core/domain/agentdefinition/enums.go` (the `Template` constant and every `Starter*` switch), `identity.go`, the GraphQL `AgentTemplate` enum and the client template picker.

### 3.7 Permission resource

- Hand-edit `S/core/domain/permission/resource_gen.go` and register the resource in `registry.go` (`registry_coverage_test.go` enforces it). Mirror to the client with `pnpm --filter @trenova/graphql resources`.
- Add the route in `routeregistry.go` and its test. GraphQL resolvers must reach a permission check or be listed with a reason in `S/api/graphql/authzlint` (`lint_test.go`). REST handlers use `h.pm.RequirePermission`.

### 3.8 Temporal jobs

- New package `S/core/temporaljobs/<x>jobs/` with `activities.go`, `workflow.go`, `registry.go`, `schedules.go`, `types.go`, `module.go`, templated on `watchtowerjobs`. Wire into `S/bootstrap/app.go`.
- Fan out per tenant with `RunTenantFanOut`. Start ad-hoc workflows through `workflowstarter`. Extend `activitynames_test.go`.

### 3.9 Integration type

- `S/core/domain/integration/enums.go` plus a migration; `config_spec.go` `ConfigSpecs` drives the settings form; secrets through `S/core/services/integrationservice/secrets.go`; connection tests in `testers.go`; catalog card in `S/core/ports/services/integration.go`. The UI at `C/routes/admin/integrations/page.tsx` renders from the spec, so a new integration needs no bespoke settings page unless its setup is a wizard (OAuth, mapping).

### 3.10 Notifications

- In-app and web push: `notificationservice.Create(&notification.Notification{...})`.
- Drivers: `drivernotificationservice.Notify`.
- Email: `EmailService.Send`. SMS: `SendSMSWorkflow`.

### 3.11 AI task kind

- `S/core/domain/aiprovider/enums.go`, `aiproviderhandler/catalog.go`, `modeladapter/sampling.go` and `aiprovider.graphqls`; `task_wiring_test.go` enforces the wiring. Call `CompleteStructured` and implement a deterministic fallback.

### 3.12 Reporting catalog

- Edit `S/infrastructure/database/reportcatalog/reportcatalog.yml`, run `task generate-reportcatalog` and its check task. Read `docs/engineering/reporting-catalog-curation.md` and `reporting-canned-authoring.md`.

### 3.13 Gates (run all before every PR)

- Backend: `task lint`, `task test`, `task gqlgen`, `task gqlschema-diff`, `task i18n-check`; integration tests for repositories (`task test-integration`); `go test -tags nofitz ./internal/api/graphql/...` if `libmupdf` is absent.
- GraphQL projections: every new field resolves to a column, relation or a `projection.yml` override/virtual.
- Client: `pnpm --filter @trenova/graphql codegen`, `pnpm lint`, `pnpm typecheck`, `pnpm lint:design`, `pnpm test`.
- Read `docs/engineering/generated-artifacts.md` before touching schema, columns or user-facing text.

---

## Part 4. Priority and dependencies

Ordered by revenue impact and how often a prospect disqualifies Trenova for lacking it. Wave 1 closes the gaps that lose deals today. Wave 2 opens the external surfaces. Wave 3 extends depth where Trenova already leads or reaches McLeod's enterprise tier.

| Wave | ID | Brief | Depends on | Parallel-safe with |
|---|---|---|---|---|
| 1 | G01 | Accounting sync (QuickBooks Online first) | none | all of wave 1 |
| 1 | G02 | Factoring autopilot | G04 for packet extraction (soft) | G01, G03, G05, G06, G07 |
| 1 | G03 | Telematics network (Motive, Geotab, mixed fleets) | none | all |
| 1 | G04 | Document AI everywhere | none | all |
| 1 | G05 | Driver app (native shell, offline) | G04 for guided scan (soft) | all |
| 1 | G06 | Load boards and market rates | none | all |
| 1 | G07 | Open platform: webhooks, full API, MCP server | none | all; G08 reuses its MCP work |
| 2 | G08 | Agent extensibility (HTTP and MCP tools, custom events) | G07 (MCP transport) | G09-G14 |
| 2 | G09 | Routed ETA and customer portal | G03 (live positions) | G10-G14 |
| 2 | G10 | Carrier portal and onboarding | G04 (COI/W-9 extraction) | G09, G11-G14 |
| 2 | G11 | Payouts (bank accounts, NACHA, quick pay) | G10 for carrier bank capture (soft) | G09, G12-G14 |
| 2 | G12 | Identity (passkeys, TOTP, SAML, SCIM) | none | all |
| 2 | G13 | Safety and claims | G04 (inspection and claim docs, soft) | all |
| 2 | G14 | Maintenance | G03 (odometer and fault codes) | all |
| 3 | G15 | Reporting to Power BI class | none | all |
| 3 | G16 | AP, budgets and bids/RFP | G01, G04, G06 | G17-G20 |
| 3 | G17 | Workflow builder | G08 | G15, G18-G20 |
| 3 | G18 | LTL consolidation and cross-dock | none | all |
| 3 | G19 | Canada and tax | G01 (tax codes sync) | all |
| 3 | G20 | Responsive web | none | all |

"Soft" dependencies mean the brief can start and integrate the other work when it lands; the plan must say how.

---
## Wave 1: close the gaps that lose deals today

Each brief below follows the same shape: Why, Today in code, Market bar, Trenova bar, Scope, UX and UI, AI surface, Acceptance criteria, Dependencies, Decisions for Eric. Tool tables list the minimum set; add any tool Part 2 implies.

---

### G01. Accounting sync: the ledger that stays in step on its own

**Why.** Trenova's own double-entry GL is deeper than DataTruck's, Rose Rocket's or Alvys's, but it is a closed island. Every competitor connects to the books a carrier already uses: DataTruck to QuickBooks Online ([integrations](https://www.datatruck.io/integrations)), Rose Rocket to QBO, Xero two-way and Business Central ([Xero](https://help.roserocket.com/xero-how-does-xero-work-with-rose-rocket)), Alvys to QBO, QB Desktop, Business Central, NetSuite and Sage Intacct ([NetSuite](https://docs.alvys.com/en/help/integrations/netsuite-integration-collection)), and McLeod added QBO in 25.1. Small and mid carriers will not leave QuickBooks to adopt Trenova; this is a disqualifier in the first demo.

**Today in code.**
- `S/core/domain/journalsource/`: the `journal_sources` table with an `IdempotencyKey`, and 14 `JournalSourceEventTypes`.
- `JournalPostingRepository.CreatePosting`; `S/core/services/invoiceservice/accounting_helpers.go`.
- `AccountingControl` has no external-sync settings. No QBO, Xero or ERP integration type exists in `S/core/domain/integration/enums.go`.

**Market bar.** A one-way or two-way push of invoices, payments and bills to QBO with a manual account-mapping screen; sync errors appear as a failed badge on the record and someone re-pushes it.

**Trenova bar.**
- **Two operating modes per tenant**: *Trenova is the ledger* (push summarized or detailed journals out) or *the external system is the ledger* (push documents: invoices, payments, bills, credit memos, customers, vendors, and pull payments and chart of accounts back). The mode is explicit and explained in setup.
- **Mapping assistant.** On connect, Trenova pulls the external chart of accounts, classes, locations, tax codes, items, customers and vendors, and proposes a full mapping using names, account types, historical usage and fuzzy matching, each with a confidence and reason. The user reviews a diff, not an empty form.
- **Sync ledger.** A row per object per attempt keyed off `journal_sources` (external ID, payload hash, status, error, attempt count). Every invoice, payment and journal in Trenova shows "Synced to QuickBooks 2 min ago" with a link, or exactly why it did not.
- **Drift reconciliation.** A scheduled job compares balances and document totals on both sides per period and raises specific drift ("Invoice INV-1042 is $1,250 in Trenova and $1,200 in QuickBooks; edited in QuickBooks on Sep 20 by j.doe"), with proposals to fix either side.
- **Period awareness.** Respect closed periods on both sides; a sync into a closed period becomes a Watchtower item with a proposal, never a silent failure.
- **Connectors in order:** QuickBooks Online (OAuth2, webhooks and CDC), then Xero, then Microsoft Dynamics 365 Business Central, then NetSuite (token-based auth, SuiteTalk REST). Build a provider interface first so each connector is an adapter.

**Scope.**
1. `AccountingConnector` port in `S/core/ports/services/` (connect, refresh, pull reference data, push document, push journal, pull changes, webhook verify) and a factory like `S/infrastructure/telematics/factory.go`.
2. Integration types, `ConfigSpecs`, encrypted OAuth tokens with refresh and expiry detection, connection tester.
3. Tables: `accounting_connections`, `accounting_mappings` (Trenova entity to external ID by kind), `accounting_sync_records`, `accounting_drift_findings`.
4. Temporal `accountingsyncjobs`: outbound queue worker driven by `journal_sources` and domain events, inbound change poller and webhook handler, nightly drift reconcile, backfill with resumable cursor.
5. `AccountingControl` settings: mode, sync granularity (per document or daily summary journal), start date, auto-sync on post versus manual approval.

**UX and UI.**
- Setup is a four-step wizard in a dialog (`size="lg"`): connect, choose mode, review proposed mappings (grouped, confidence-sorted, low-confidence first, `AssistMark` on proposals, bulk accept with `Shift+A`), choose start date and backfill. It can be abandoned and resumed.
- An **Accounting sync** page: `KpiStrip` (synced today, pending, failed, drift findings), then the sync ledger table with filters by status and object kind, row click opens a side sheet with the payload, the external record link and the error in plain language plus the fix.
- Every synced record shows a small sync state line in its header; failed shows the reason and a Retry action.
- Mapping editor supports search on both sides, shows usage counts, and warns before remapping an account with history.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `get_accounting_sync_status` | read | | Health, queue depth, last success per object kind |
| `list_accounting_sync_records` | read | | Filterable sync ledger |
| `list_accounting_drift_findings` | read | | Open drift with both sides' values |
| `get_accounting_mapping` | read | | Current mapping for an entity or account |
| `propose_accounting_mapping` | action | Propose | Suggest mappings for unmapped items with reasons |
| `retry_accounting_sync` | action | AutoExecute | Re-queue failed records (idempotent) |
| `resolve_accounting_drift` | action | ActWithApproval | Push Trenova's value or accept the external value, with Simulate showing both |
| `trigger_accounting_backfill` | action | ActWithApproval | Backfill a date range |

- Events: `accounting.sync_failed`, `accounting.drift_detected`, `accounting.connection_degraded`.
- Watchtower source `AccountingSync` (failed records older than N minutes, drift, expiring or revoked tokens, closed-period rejections).
- Template: **Books Keeper** agent: watches sync failures and drift, retries transient errors, proposes mapping fixes, and writes a weekly reconciliation note in Desk.
- Report catalog: sync records and drift findings; canned report "Sync exceptions by week".

**Acceptance criteria.**
- Connect a QBO sandbox, accept proposed mappings, and see invoices, payments and customers appear in QBO within one minute of posting, each with an external link.
- Edit an invoice amount in QBO; drift is detected on the next reconcile with the editor and time, and a one-click fix exists in each direction.
- Revoke the QBO app; the integration shows Disconnected within one poll, a Watchtower item appears, and nothing is lost: queued records sync after reconnect.
- Duplicate pushes are impossible (idempotency by `journal_sources.IdempotencyKey`), proven by a test that replays the same event twice.
- The whole flow can be driven from Desk by an agent with the tools above.

**Dependencies.** None. G16 (AP) and G19 (tax) extend the object kinds.

**Decisions for Eric.** Whether detailed-journal mode (Trenova is the ledger) ships in milestone one or after the document mode. Recommended: document mode first, because that is what QBO shops expect.

---

### G02. Factoring autopilot

**Why.** Most small carriers factor. DataTruck lists 21 factoring partners and sends packets from the load once the POD is verified ([factoring](https://www.datatruck.io/blog/best-tms-software-with-built-in-factoring-for-carriers)); Alvys has 11 connections including RTS, TAFS, Triumph and OTR ([setup](https://docs.alvys.com/en/help/integrations/how-to-set-up-and-use-factoring-in-alvys)); Rose Rocket and McLeod connect to TriumphPay. Trenova has nothing: the factoring fields on the customer billing profile are commented out.

**Today in code.**
- `S/core/domain/customer/billingprofile.go` (around lines 91-106): `UseFactoring` and `FactoringCompanyID` commented out.
- `DocumentPacketRule` and `PacketSummary` already assemble document packets for billing; reuse them.
- Invoices, payments and bank receipt matching exist in AR (`bankreceipt` tools in `agentquerytoolservice`).

**Market bar.** Pick a factor per customer, click "Send to factor" on an invoice, a packet goes by email or API, and the user marks it funded by hand.

**Trenova bar.**
- **Factoring as a ledger, not a button.** A `FactoringCompany` entity with terms (advance rate, fee schedule by days outstanding, reserve percentage, recourse or non-recourse, chargeback window), a Notice of Assignment per customer with signed NOA document and status, and a **reserve ledger** that tracks advanced, fees, reserve held, reserve released and chargebacks per invoice, posting correctly to the GL.
- **Packet readiness scoring.** Every invoice shows whether it is ready to factor: required documents present and legible (POD signed, BOL, rate con), amounts match the rate con, no open disputes. Missing items are listed with the one action that fixes each. G04's extraction verifies signatures and amounts.
- **Autopilot.** When a load delivers and the packet is ready, the Factoring agent assembles the schedule and proposes the submission. Submitting to a factor is a money-movement action, so under Part 2 it stays at Propose and never reaches AutoExecute; the human's job shrinks to one approval per batch.
- **Schedule of accounts** generated per factor's format (PDF plus CSV) with the packet, submitted through the factor's transport: API where available (Triumph, RTS, OTR), SFTP, or email with a structured subject.
- **Remittance auto-match.** Factor remittance files and emails are parsed and matched to invoices; advances, fees and reserve releases post automatically; mismatches go to an exception queue.
- **Non-factored customers** stay on normal AR; per-customer and per-load overrides exist.
- **Cash forecast integration.** The existing cash-flow forecast includes expected advances and reserve releases.

**Scope.**
1. Domain and tables: `factoring_companies`, `factoring_agreements`, `factoring_noas`, `factoring_submissions` (schedule of accounts), `factoring_submission_items`, `factoring_reserve_entries`. Uncomment and complete the billing profile fields with a migration.
2. `FactoringTransport` port with adapters: email, SFTP (`shared/sftp`), and API adapters for the first two factors Eric picks.
3. Journal source events for advance, fee, reserve hold, reserve release and chargeback, wired through `journal_sources`.
4. Remittance parsers (CSV and PDF via G04 when present) and a matcher reusing bank receipt matching patterns.
5. Temporal `factoringjobs`: readiness evaluation on document and invoice events, submission, remittance ingestion, NOA expiry and chargeback window sweep.

**UX and UI.**
- **Factoring** page: `KpiStrip` (ready to submit, submitted awaiting funds, reserve held, chargebacks due), then a readiness queue table: invoice, customer, factor, readiness badge with phase tone, missing items inline, bulk "Submit selected".
- Invoice detail gets a Factoring `SectionPanel`: readiness checklist with document thumbnails, submission history, reserve ledger lines.
- Factor detail page: terms in `DescriptionList`, NOAs by customer with status, reserve balance trend.
- Remittance import: drop a file, see matched and unmatched lines side by side, fix unmatched with suggestions.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `get_factoring_readiness` | read | | Readiness and missing items for an invoice |
| `list_factoring_ready_invoices` | read | | Queue of ready invoices by factor |
| `get_factoring_reserve_balance` | read | | Reserve by factor, customer or invoice |
| `list_factoring_submissions` | read | | Schedules and their funding state |
| `submit_invoices_to_factor` | action | Propose (never Auto) | Build and send a schedule of accounts |
| `request_missing_factoring_document` | action | Propose | Ask the driver or carrier for a missing doc |
| `apply_factoring_remittance` | action | ActWithApproval | Post a parsed remittance |
| `record_factoring_chargeback` | action | ActWithApproval | Record a chargeback and reopen the invoice for collection |

- Events: `factoring.invoice_ready`, `factoring.submission_funded`, `factoring.chargeback_due`, `factoring.noa_expiring`.
- Watchtower source `Factoring` (ready but unsubmitted over 24 h, submitted but unfunded past terms, chargeback windows closing, NOA missing for a factored customer).
- Template: **Factoring Clerk** agent (Propose, shadow on).
- Artifact: reuse table and entity cards; add a `factoring_schedule` artifact only if the schedule preview needs its own layout.
- Report catalog: submissions, reserve entries; canned reports "Factoring fees by customer" and "Days to fund by factor".

**Acceptance criteria.**
- A delivered load with a signed POD produces a ready invoice within one minute; a missing POD shows a single "Request from driver" action.
- Submitting three invoices produces a schedule of accounts PDF and CSV and sends it via the chosen transport; the GL shows advance, fee and reserve entries that balance.
- Importing a remittance matches at least the exact-amount lines automatically and lists the rest with ranked suggestions.

**Dependencies.** Soft on G04 for signature and amount verification. G11 reuses the bank details model for reserve releases.

**Decisions for Eric.** Which two factors get API adapters first. Recommended: Triumph (TriumphPay is also used by Rose Rocket and McLeod) and RTS or OTR.

---

### G03. Telematics network: every ELD, mixed fleets, one model

**Why.** Samsara-only disqualifies most prospects. DataTruck integrates about 30 ELDs ([integrations](https://www.datatruck.io/integrations)), Alvys about 20 including Motive, Geotab, Verizon and Orbcomm ([Samsara setup](https://docs.alvys.com/en/help/integrations/connecting-samsara-to-alvys)), McLeod about 22, and Rose Rocket Samsara, Geotab plus the Terminal aggregator ([integrations](https://www.roserocket.com/integrations)).

**Today in code.**
- `TelematicsProvider` interface (`S/core/ports/services/telematicsprovider.go`, around line 195) with 13 methods; factory at `S/infrastructure/telematics/factory.go`; Samsara adapter plus `shared/samsara` (including live shares).
- `Type.SupportsTelematics()` in the integration domain is the single gate. `telematicsservice` handles polling, webhooks, stop actuals and sweeps; `telematicsjobs` exists.
- Motive is a commented-out enum value.

**Market bar.** A list of ELD logos. Each maps vehicles by hand, one provider per company, and silently stops updating when a token expires.

**Trenova bar.**
- **Provider-agnostic core.** Positions, HOS clocks, duty status, DVIRs, fault codes, odometer and engine hours, driver-vehicle assignments and geofence events all land in one normalized model regardless of source.
- **Mixed fleets.** Multiple providers per tenant at once (owner-operators on Motive, company trucks on Samsara), with per-asset source of truth and conflict rules.
- **Auto-matching.** Vehicles match by VIN, then plate, then unit number; drivers by license number, then email, then name, each with confidence. Unmatched items are a review queue, not a blank.
- **Coverage map.** An integrations view that shows which assets report, their last ping, stale assets, and the provider each uses.
- **Adapters in order:** Motive (KeepTruckin), Geotab (MyGeotab SDK), then an aggregator (Terminal) to cover the long tail, then Verizon Connect and Omnitracs direct if customers need it.
- **Self-healing** per Part 1.4: token refresh, rate-limit aware polling, webhook signature verification, replay of missed windows after an outage.

**Scope.**
1. Review the 13-method `TelematicsProvider` interface and split it into capability interfaces (`PositionSource`, `HOSSource`, `DVIRSource`, `FaultCodeSource`, `OdometerSource`, `AssignmentSource`) so providers implement what they support and the UI shows capability badges.
2. Add Motive, Geotab and Terminal integration types with `ConfigSpecs`, OAuth or key auth, testers, and adapters; keep Samsara working unchanged.
3. `telematics_asset_links` (asset to provider external ID, match method, confidence, confirmed by) and a matching service.
4. Persist odometer and engine hours on tractors (needed by G14) and fault codes (new table) for providers that supply them.
5. Replace the single `SupportsTelematics()` gate with capability checks; support more than one active telematics integration per tenant.

**UX and UI.**
- **Telematics** admin page: `KpiStrip` (assets reporting, stale over 30 min, unmatched, providers healthy), a coverage table (asset, provider, last ping age, capabilities), and a matching queue with `AssistMark` suggestions and bulk accept.
- Asset detail shows its data source and freshness in the header.
- Setup for each provider is a guided dialog with a live "we can see 142 vehicles and 97 drivers" preview before saving.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `get_telematics_coverage` | read | | Reporting, stale and unmatched counts by provider |
| `list_unmatched_telematics_assets` | read | | Queue with suggested matches |
| `get_vehicle_live_state` | read | | Position, speed, odometer, faults, HOS of assigned driver |
| `link_telematics_asset` | action | AutoExecute when confidence is exact VIN, else Propose | Confirm a match |
| `resync_telematics_window` | action | AutoExecute | Replay a time window from a provider |

- Events: `telematics.asset_stale`, `telematics.provider_degraded`, `telematics.fault_code_raised`.
- Watchtower source `TelematicsHealth` (stale assets on active loads, disconnected providers, unmatched assets on active loads).
- Extend the existing monitoring and dispatch agents' tool lists; add a **Fleet Connectivity** template that keeps matching and health clean.

**Acceptance criteria.**
- With Samsara and Motive sandboxes connected at once, a dispatch board shows positions from both, each asset from its assigned source.
- 90% or more of seed vehicles auto-match by VIN; the rest appear in the queue with ranked suggestions.
- Disconnecting Motive raises a Watchtower item within one poll and shows which active loads lose tracking.
- Existing Samsara features (HOS, DVIR, geofence auto-arrive, live shares) pass their tests unchanged.

**Dependencies.** None. G09 (ETA) and G14 (maintenance) consume its data.

**Decisions for Eric.** Whether to buy Terminal (aggregator) access to cover the long tail quickly. Recommended: yes, after Motive and Geotab direct.

---

### G04. Document AI everywhere

**Why.** Trenova classifies rate cons, BOLs and PODs but extracts fields only from rate confirmations. DataTruck's TruckGPT extracts rate cons, BOLs, PODs, CDLs, receipts, work orders and inspections ([TruckGPT](https://www.datatruck.io/truckgpt)); Rose Rocket has DataBot OCR and TED email-to-order ([TED](https://help.roserocket.com/platform/using-ted-email-ai-to-automate-order-and-quote-entry)); Alvys Foundry covers BOLs and PODs. Every later brief (factoring, carrier portal, AP, safety) depends on reading documents well.

**Today in code.**
- `AIDocumentService` (`S/core/services/aidocumentservice/`) exposes only `RouteDocument` and `ExtractRateConfirmation`; prompts live in `prompts.go`.
- `documentparsingruleservice`: the `DocumentKind` parsing rule supports only rate confirmations.
- Pipeline: `S/core/temporaljobs/documentintelligencejobs/activities.go` (fitz text, Tesseract OCR, AI) feeding `DocumentShipmentDraft`. `DocumentControl` holds the toggles. `shipmentimportassistantservice` handles the import flow.

**Market bar.** Upload a document, fields appear in a form, a person fixes the wrong ones. No confidence, no source highlighting, no learning.

**Trenova bar.**
- **A document kind registry** where each kind declares its schema, validators, the entity it attaches to, and what happens after extraction. Kinds in order: BOL, POD (with signature, date, piece count and exception notation detection), lumper and fuel receipts, carrier invoice, certificate of insurance, W-9, CDL, medical card, DVIR, inspection report, work order and repair invoice, accident report, scale ticket.
- **Per-field confidence and provenance.** Every extracted value carries confidence and a bounding box. The review viewer shows the page beside the form; hovering a field highlights its source region; low-confidence fields are flagged and block auto-apply.
- **OS&D detection.** On PODs and BOLs, detect handwritten exceptions ("2 cartons damaged", "short 3", "refused") and raise them as a service failure and claim candidate (feeds G13).
- **Cross-document checks.** POD consignee matches the stop, piece count matches the BOL, carrier invoice amount matches the rate con, COI named insured matches the carrier. Mismatches are findings, not surprises at billing.
- **Learn from corrections.** Corrections are stored per tenant, per kind and per sender (by layout fingerprint) and used as few-shot examples and deterministic overrides next time. Show the accuracy trend per kind.
- **Multi-document splitting.** A 12-page scan of a packet is split and classified into its documents automatically.
- **Email intake.** A per-tenant intake address routes attachments through the same pipeline and attaches results to the right load by reference numbers.

**Scope.**
1. Generalize `ExtractRateConfirmation` into `Extract(kind, document)` driven by the registry; one AI task kind per document family so tenants can route them (Part 3.11), each with a deterministic OCR-and-regex fallback for the key fields.
2. Tables: `document_extractions` (kind, model, fields with confidence and boxes as JSONB, status), `document_extraction_corrections`, `document_layout_fingerprints`.
3. Extend `DocumentControl` with per-kind auto-apply thresholds.
4. Post-extraction handlers per kind: attach POD to stop and mark delivered-with-POD, create lumper accessorial, feed factoring readiness, update carrier insurance, create DQ file items, create maintenance and safety records.
5. Packet splitting activity in `documentintelligencejobs`.

**UX and UI.**
- A **Review** workspace: queue on the left (`j`/`k`), document viewer centre with zoom and region highlights, extracted fields right with confidence, `AssistMark` on AI values, keyboard to accept a field (`Enter`) or jump to the next low-confidence field (`Tab`). Accepting all is `a`.
- Load detail documents tab shows per-document status: extracted, needs review, applied, mismatch.
- An accuracy panel per kind on the Document settings page: volume, auto-applied rate, correction rate trend.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `get_document_extraction` | read | | Fields, confidence and provenance for a document |
| `list_documents_needing_review` | read | | Review queue with reasons |
| `find_document_mismatches` | read | | Cross-document check findings for a load |
| `extract_document` | action | AutoExecute | Run or re-run extraction for a kind |
| `apply_document_extraction` | action | AutoExecute above threshold, else ActWithApproval | Apply fields to the target entity |
| `split_document_packet` | action | AutoExecute | Split a multi-document scan |
| `raise_osd_from_document` | action | Propose | Create a service failure or claim from a detected exception |

- Events: `document.extracted`, `document.needs_review`, `document.mismatch_found`, `document.osd_detected`.
- Watchtower source `DocumentReview` (review backlog age, mismatches on loads ready to bill).
- Template: **Document Clerk** agent that clears the review queue within thresholds and escalates the rest.
- Artifact: a `document_extraction` artifact in Desk showing the page with highlighted fields.
- Report catalog: extractions and corrections; canned report "Extraction accuracy by kind and sender".

**Acceptance criteria.**
- For a seed set of 20 documents per kind, key fields reach the accuracy target Eric sets (propose 95% on printed fields) and every field has a confidence and box.
- A POD with "2 damaged" handwritten produces an OS&D finding linked to the stop.
- Correcting a field once for a sender's layout changes the next extraction from that layout.
- With no AI provider configured, the fallback extracts the reference numbers and dates and marks everything for review.

**Dependencies.** None. G02, G10, G13, G14 and G16 consume it.

**Decisions for Eric.** The accuracy target and the review threshold defaults.

---

### G05. Driver app: native, offline, one-thumb

**Why.** Every competitor ships native iOS and Android driver apps: DataTruck ([mobile](https://www.datatruck.io/mobile-app), rated 3.5 on iOS with upload complaints), Rose Rocket with ePOD signature and scanning ([driver app](https://help.roserocket.com/platform/rose-rocket-driver-mobile-app)), Alvys ([Play](https://play.google.com/store/apps/details?id=io.alvys.alvys)), McLeod Driver Sidekick. Trenova's Dash is an installable web app without offline, signature, DVIR or real scanning. Drivers in dead zones lose work; recruiters lose drivers.

**Today in code.**
- `client/apps/dash` (Vite, React 19, deployed on Cloudflare Workers). `dash-sw.js` handles push only: no offline cache, no IndexedDB queue. The camera is a plain `<input capture>`.
- Backend: `driverportalservice` with about 60 `My*` methods; `driver_portal.graphqls`; `DashControl` with about 20 feature flags; `drivernotificationservice`.

**Market bar.** A native app with a load list, status buttons, a camera for documents, and a signature pad. Uploads fail in poor coverage and drivers retry by hand.

**Trenova bar.**
- **Offline first.** Every driver action (arrive, depart, document capture, signature, DVIR, messages, expenses) writes to a local queue and syncs when the network returns, with visible per-item state and conflict handling. The app is fully usable in a dead zone.
- **Guided scan.** Edge detection, auto-capture when steady, perspective correction, multi-page, glare warning, and on-device quality checks before upload; the document kind is suggested and extraction runs (G04), so the driver sees "POD accepted" or "Signature missing, retake page 2".
- **Electronic signature and ePOD** with consignee name, timestamp and location, producing a PDF attached to the stop.
- **DVIR** pre-trip and post-trip with component checklist, photos of defects, and a certified-repair flow (feeds G14). Samsara DVIRs remain supported for fleets that use them.
- **Background location** with battery-aware sampling for fleets without ELD coverage (owner-operators), and geofence auto-arrive.
- **Voice** for hands-free status updates and messages, with a **safety lockout** that disables typing and non-essential screens while the vehicle moves.
- **Spanish** from day one; one-thumb layout; 44 px targets; readable in sunlight (high-contrast mode).
- **Earnings you can trust**: settlement previews per load as soon as it delivers, not at payday.

**Scope.**
1. Wrap `client/apps/dash` in a Capacitor native shell for iOS and Android (keeps one codebase and shares `@trenova/shared`), with native plugins for camera and document scanning, background geolocation, secure storage, push (APNs/FCM) and biometrics.
2. An offline data layer: IndexedDB (or SQLite through Capacitor) cache for the driver's active loads and a durable mutation queue with idempotency keys the server honours.
3. Server: idempotent `My*` mutations (accept a client-generated key), a batch sync endpoint, signed upload URLs with resumable uploads.
4. DVIR domain for Trenova-native DVIRs if G14 has not landed yet (coordinate with G14 on one model).
5. Store listings, build pipeline, and release channels (internal, beta, production).

**UX and UI.**
- Home is "Your next stop": one card with the address, appointment, the one action that is due next, and navigation. Everything else is one tap away.
- A sync pill in the header shows queued items; tapping it lists each with state.
- Capture flow: kind picker is pre-filled; guided scan; review with extraction result; done. Three taps for a POD.
- Follows the design system tokens; dark mode default at night.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `get_driver_app_status` | read | | Last seen, app version, queued items, location permission for a driver |
| `list_drivers_with_sync_issues` | read | | Drivers whose queues are stuck or apps are outdated |
| `request_driver_document` | action | Propose | Push a request for a document on a stop |
| `send_driver_message` | action | Propose | Message a driver through the app (respects safety lockout) |
| `request_driver_location_ping` | action | AutoExecute | Ask the app for a fresh position |

- Events: `driver.document_captured`, `driver.dvir_submitted`, `driver.defect_reported`, `driver.app_offline_long`.
- Watchtower source `DriverApp` (stuck queues, missing PODs after delivery, drivers on active loads without location).
- Extend the existing driver-facing agents; the **Driver Assistant** template answers drivers' questions in the app (pay, next stop, documents) with read tools only.

**Acceptance criteria.**
- In airplane mode, a driver can arrive, capture a signed POD and depart; on reconnect, all three sync in order with no duplicates.
- Guided scan produces a cropped, flattened PDF and G04 reports the kind and signature presence.
- The app passes App Store and Play review in internal testing tracks.
- Safety lockout engages above a speed threshold and is testable in a simulator.

**Dependencies.** Soft on G04 (scan verification) and G14 (DVIR model).

**Decisions for Eric.** Native technology. Recommended: Capacitor around the existing Dash app (fastest, one codebase, reuses `@trenova/shared`). The alternative is React Native, which gives smoother native feel at the cost of a second UI codebase.

---

### G06. Load boards and market rates

**Why.** DataTruck connects DAT, Truckstop, 123Loadboard, Sylectus, Uber Freight, RXO and Relay ([integrations](https://www.datatruck.io/integrations)); Rose Rocket DAT and Truckstop; Alvys posts and books on DAT, Truckstop and Uber Freight ([marketplace](https://docs.alvys.com/en/help/integrations/alvys-carrier-marketplace)); McLeod DAT, Truckstop and 123Loadboard. Trenova has none, yet it has the best pieces to do this better: a real rating engine, dispatch scoring with HOS, and a routing guide.

**Today in code.**
- Rating: `RateEngine` with `RateShipment`/`Shop`; rate simulation; EIA fuel.
- Tendering: `tenderservice` `CreateWaterfall`/`CreateSpot`, stalled-tender sweep, public accept page.
- Carrier sourcing in `carrierintelservice` and the `CarrierIntelConnector` pattern.
- Dispatch scoring in `dispatchcandidateservice` and `shared/dispatchplanner`.

**Market bar.** Post a load to DAT and Truckstop from the load screen; search boards in a pop-up; see a DAT rate lookup in a side panel.

**Trenova bar.**
- **Market rate everywhere it matters**: quotes, rate agreements, tenders and carrier pay show the market rate (DAT RateView, Truckstop rate insights) beside Trenova's own lane history and the rating engine's price, with the spread and confidence. The rate simulation can use market rates as a scenario.
- **Automatic posting** when the routing guide is exhausted: the waterfall ends, the load posts to boards with a target and max rate from the rate engine, and offers route back into tenders.
- **Backhaul finder** for carriers: for each truck about to go empty, search boards for loads that fit its HOS, equipment and home time, scored with the same dispatch factors and costing, so the answer is "this load adds $412 margin and gets Juan home Friday", not a list of 300 postings.
- **Capacity search** for brokers: search truck postings, rank by carrier vetting status and lane history, and start a spot tender to the best in one action.
- Posts are kept in sync (update, cover, remove on booking) with no orphan postings.

**Scope.**
1. `LoadBoardProvider` port (post, update, remove, search loads, search trucks, rate lookup) with DAT and Truckstop adapters first, then 123Loadboard.
2. Tables: `load_board_postings`, `load_board_search_results` (cached, TTL), `market_rate_snapshots`.
3. Hooks: tender waterfall exhaustion, shipment covered, shipment cancelled.
4. Backhaul scoring service reusing dispatch scoring and costing.
5. Market rate in the rate engine as an optional input with provenance.

**UX and UI.**
- Load detail gets a Market `SectionPanel`: market rate band, Trenova history, engine price, spread; posting state with a Post/Update/Remove action.
- A **Load boards** workspace: saved searches, results table with Trenova's score and margin, `AssistMark` on the recommended rows, one-key "Book" or "Tender".
- Quote builder shows the market band beside the price in real time.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `get_market_rate` | read | | Market rate for a lane, equipment and date |
| `search_load_board_loads` | read | | Scored loads for a truck or lane |
| `search_load_board_trucks` | read | | Scored carriers for a load |
| `list_load_board_postings` | read | | Active posts and their state |
| `post_load_to_boards` | action | Propose | Post with target and max rate |
| `remove_load_board_posting` | action | AutoExecute | Remove on cover or cancel |
| `book_load_board_load` | action | Propose | Book a load for a truck |

- Events: `loadboard.posted`, `loadboard.response_received`, `routing_guide.exhausted`.
- Watchtower source `Coverage` extension: uncovered loads nearing pickup with board activity.
- Templates: **Backhaul Finder** (carriers) and **Capacity Hunter** (brokers), both Propose.
- Report catalog: postings and market rate snapshots; canned report "Our rate versus market by lane".

**Acceptance criteria.**
- Exhausting a seed waterfall posts the load to the DAT sandbox with the engine's target rate; covering it removes the post.
- A backhaul search for a seed truck returns scored results with margin and HOS feasibility explained per row.
- Market rate appears on quotes with its source and date.

**Dependencies.** None; API credentials are required.

**Decisions for Eric.** Which board contracts to obtain first (DAT, Truckstop, both).

---

### G07. Open platform: webhooks, a full API and an MCP server

**Why.** API keys reach only 13 master-data resources and there are no outbound webhooks. Alvys has about 96 REST endpoints, signed webhooks and a read-only MCP server in beta ([webhooks](https://docs.alvys.com/en/api/reference/webhooks/overview), [MCP](https://docs.alvys.com/en/api/guides/mcp)); Rose Rocket has REST with OAuth2 including custom objects ([API](https://roserocket.readme.io/docs)); DataTruck offers APIs and webhooks at no charge. Enterprise buyers and integrators need this, and an MCP server lets every customer's own AI use Trenova, governed.

**Today in code.**
- `runtimePolicy` in `S/core/services/apikeyservice/policy.go` covers 13 resources. `/api/v1` routes in `S/api/router.go`; OpenAPI at `services/tms/docs/openapi-3.*`. GraphQL is the main client API.
- `shared/webhooksig` exists (signing). No `domain/webhook`. The enterprise plan's WS7 proposes `domain/webhook` plus `webhookjobs`.
- GTC streams `cdc:shipments`, `tca:events` and `reporting:cdc` (`services/gtc/config/gtc.yaml`).

**Market bar.** A REST reference, API keys, and a webhook list with a URL and event checkboxes; failures are retried a few times and dropped. Alvys's MCP is read-only.

**Trenova bar.**
- **Every resource** reachable through API keys with scoped permissions (the same permission resources as users), rate limits per key, and usage analytics per key.
- **Webhooks as a product.** Typed event catalog with versioned JSON schemas, per-subscription filters (event type, customer, field conditions), HMAC signatures via `shared/webhooksig` with rotation, exponential retries, a dead-letter queue, replay of any delivery or a time range, a delivery log with request and response, and endpoint health that auto-pauses a failing endpoint and tells the owner.
- **Events sourced from CDC**, not sprinkled in services, so every change is covered consistently (GTC to a webhook dispatcher), plus domain events (tender accepted, invoice posted) from the agent event bus.
- **MCP server that can act.** Expose read tools and action tools over MCP with OAuth. Writes from outside become **governed proposals** in the decision queue under the same tiers, budgets and audit as internal agents. That is the difference from Alvys's read-only beta: a customer's Claude or ChatGPT can manage Trenova, safely.
- **Developer experience**: a developer portal page in settings with interactive API reference generated from OpenAPI, sandbox keys, event samples, a "send test event" button, and code snippets.

**Scope.**
1. API keys: extend `runtimePolicy` to every permission resource (derive it, do not hand-list), scopes per key, per-key rate limits, key expiry and rotation, last-used tracking.
2. `domain/webhook`: subscriptions, event types, deliveries, attempts, endpoint health. `webhookjobs` for dispatch, retry, DLQ sweep and replay. A GTC consumer that turns CDC streams into typed events.
3. Complete REST coverage for the core resources (shipments, stops, customers, carriers, invoices, payments, documents, tenders, rates) with OpenAPI kept in sync and generated clients tested.
4. MCP server (streamable HTTP) with OAuth 2.1 authorization, mapping tools from the agent tool registry filtered by the caller's permissions; action tools create proposals through `proposalrecorder`.
5. Security: SSRF-safe delivery through `shared/httpsafe`, secret storage encrypted, endpoint verification handshake.

**UX and UI.**
- **Developers** settings area: API keys table (scopes, last used, rate), Webhooks (subscriptions table; detail with delivery log, filter builder, health chart, replay), MCP (connected clients, granted scopes, recent proposals from external clients), API reference.
- Delivery log rows open a side sheet with request, response, timing and a Replay button.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `list_webhook_subscriptions` | read | | Subscriptions and health |
| `list_failed_webhook_deliveries` | read | | DLQ with reasons |
| `get_api_key_usage` | read | | Usage and errors per key |
| `replay_webhook_deliveries` | action | ActWithApproval | Replay a delivery or a range |
| `pause_webhook_subscription` | action | AutoExecute | Pause a failing endpoint (reversible) |
| `create_webhook_subscription` | action | Propose | Create or change a subscription (external) |

- Events: `webhook.endpoint_failing`, `api_key.expiring`, `mcp.client_connected`.
- Watchtower source `Platform` (failing endpoints, keys expiring, DLQ growth).
- Template: **Integration Steward** agent.

**Acceptance criteria.**
- Subscribing to `shipment.status_changed` with a customer filter delivers signed events within five seconds of a change; a receiver returning 500 sees retries, then the DLQ, then a successful replay.
- An API key scoped to invoices:read cannot read shipments.
- Claude Desktop connects to the MCP server via OAuth, reads a shipment, and a "place a hold" call appears as a proposal in the decision queue rather than executing.

**Dependencies.** None. G08 builds the MCP client on the same library.

**Decisions for Eric.** Whether the public REST surface is versioned `v1` forever with additive changes (recommended) or date-versioned.

---
## Wave 2: open the external surfaces

---

### G08. Agent extensibility: tenants bring their own tools

**Why.** This extends the lead that matters most. Tenants can already define agents, instructions, guardrails, tiers and triggers over 114 compiled tools, and bring any model provider. The honest gap: a tenant cannot add a tool, so an agent cannot reach the tenant's own systems. Alvys Foundry (announced August 2026, waitlist) builds agents from plain-English goals with a vendor-picked model ([Foundry](https://alvys.com/foundry)); Rose Rocket's Rosie runs vendor-fixed actions; McLeod plans a query-only assistant for Q4 2026 ([TT](https://www.ttnews.com/articles/mcleod-ai-assistant-agents)). Letting tenants add tools under the same governance makes Trenova the only TMS where the AI platform is genuinely open.

**Today in code.**
- Agent definitions in `S/core/domain/agentdefinition/`; tool registries in `agenttoolservice` and `agentquerytoolservice`; governance in `proposalrecorder`, `proposalexecutor`, budgets and earned trust; Desk in `assistantservice`.
- `shared/httpsafe` provides SSRF-safe HTTP.
- Agent events in `S/core/domain/agent/events.go` are a fixed list.

**Market bar.** Vendor-built agents with vendor-built actions; at best a prompt box to customize behaviour.

**Trenova bar.**
- **Custom HTTP tools.** A tenant defines a tool with name, description, JSON parameter schema, method, URL template, headers from encrypted secrets, body template, response mapping (JSONPath to fields), read or action kind, permission resource, and default tier. Calls go through `shared/httpsafe` with allowlisted hosts, timeouts and size limits. Action tools are governed exactly like built-ins: Simulate (a dry-run request or a rendered preview), approvals, budgets, audit.
- **MCP client.** Connect any MCP server (OAuth or token); its tools appear in the tool picker with a trust review step, default to Propose, and can be granted per agent.
- **Describe an agent.** Type what you want ("Watch for loads to Walmart DCs that will miss their appointment and email the customer 2 hours ahead") and Trenova drafts the agent: instructions, triggers, tools, tiers, and a shadow-mode plan, then replays the last 30 days to show what it would have done.
- **Custom events.** Tenants define triggers from any report or saved table view with a condition ("rows appear", "value crosses"), from a webhook inbound URL, or a schedule; agents wake on them.
- **Prompt and definition versioning** with diff, rollback, and replay of a new version against past runs before promotion.
- **Agent library.** Export and import agents as signed bundles; a curated gallery of Trenova-authored and (later) community agents.
- **Desk improvements**: chart artifact from any query result, shared threads with teammates, pinned threads.

**Scope.**
1. `custom_tools` domain (definition, secrets refs, version), executor adapters implementing `AgentQueryTool` and `AgentTool` dynamically, registry merge with built-ins per tenant, permission and tier enforcement.
2. MCP client connections with tool discovery, schema caching, health, and per-tool enablement.
3. Agent drafting as an AI task kind with a deterministic template fallback; replay harness reuse.
4. Custom event sources: report and view conditions evaluated by a Temporal schedule; inbound webhook endpoint per trigger with signature verification.
5. Versioning tables for agent definitions and diffs.
6. Import/export bundle format with signature and a compatibility check.

**UX and UI.**
- Agent Control gets a **Tools** tab: built-in, custom HTTP, and MCP sources in one list with kind, tier and usage. A tool editor with a live "Test call" pane showing request, response and the mapped result.
- "New agent" starts with a single text box ("What should this agent do?"), then shows the drafted agent as editable sections with the replay result.
- Version history with side-by-side diff and "Replay this version".

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `list_agent_tools` | read | | All tools available to the tenant with source and tier |
| `get_agent_run_history` | read | | Runs, outcomes and costs for an agent |
| `draft_agent_definition` | action | Propose | Draft an agent from a description |
| `create_custom_tool` | action | Propose | Define a custom tool (security relevant) |
| `promote_agent_version` | action | ActWithApproval | Promote a version after replay |

- Events: `agent.version_promoted`, `custom_tool.failing`, `mcp_server.disconnected`.
- Watchtower source `AgentPlatform` (failing custom tools, budget exhaustion, agents stuck in shadow with good scores ready to promote).
- Template: **Agent Architect** (helps build and tune other agents).

**Acceptance criteria.**
- A tenant defines a custom HTTP tool to a test endpoint, grants it to an agent, and sees its action calls land in the decision queue with a Simulate preview.
- A private-network URL (169.254.169.254, 10.0.0.0/8) is refused by `httpsafe`.
- An MCP server's tools appear after connection and default to Propose.
- "Describe an agent" produces a working agent that replays against seed history with a readable result.

**Dependencies.** G07 for shared MCP libraries and OAuth.

**Decisions for Eric.** Whether custom tools may use AutoExecute at all. Recommended: allowed for reads only, never for custom actions.

---

### G09. Routed ETA and customer portal

**Why.** Trenova's ETA is a straight line times 1.2 at 50 mph, and it honestly says "Not routed; ignores traffic, weather and dwell". Customers get EDI 214 but no portal. Rose Rocket's customer portal shows status, map, documents and quick quotes ([portal](https://help.roserocket.com/customer-portal)); McLeod's handles quotes, tracking, tenders and documents; Alvys has a branded public tracking page; McLeod predicts ETA from GPS.

**Today in code.**
- `S/core/services/shipmenttracking/`: `Build` produces an `ArrivalEstimate` with `Verdict` and `Basis`; used only by agents.
- `shared/pcmiler` for routed miles; HOS projection exists in dispatch; detention and facility intelligence hold dwell history.
- Public token handlers: `tenderpublichandler`, `rateconfirmationpublichandler`, invoice shared-document download.
- Comments have a "Customer" visibility that nothing delivers. The "portal invitation" template is for drivers (`worker_portal_invitations`).

**Market bar.** A tracking link with a map pin and an ETA from GPS speed; a portal with shipment list, documents and a quote form.

**Trenova bar.**
- **An ETA that explains itself.** Routed with PC*MILER truck routing, then HOS-aware (it knows the driver must take a 10-hour break in Amarillo), then facility-aware (appointment windows and the facility's historical dwell from detention data), then live (current position and speed, traffic where available, weather alerts). Every ETA shows its basis and a confidence band, keeps `Verdict` and `Basis`, and records its history so accuracy is measurable.
- **Slip detection.** When the ETA passes the appointment window or the slack shrinks below a threshold, an event fires hours before the miss, with the cause ("HOS reset required", "Dwell at shipper 3 h over average").
- **Customer portal** by magic link (no passwords unless the customer enables SSO later): shipments with live map and explained ETA, documents (BOL, POD, invoice), quotes (instant price from the rating engine within customer rate agreements, with accept to create an order), order entry from templates, invoices with online payment status and disputes, and a message thread per shipment using the Customer visibility on comments.
- **Proactive updates**: customers subscribe per shipment or account to milestones and ETA changes by email or SMS; messages come from the tenant's brand.
- **Tenant branding**: logo, colours within design tokens, custom domain.

**Scope.**
1. ETA engine in `shipmenttracking`: routing via `shared/pcmiler`, HOS projection reuse, dwell model per facility from detention/stop actuals, live adjustment from G03 positions; `eta_snapshots` history; accuracy metrics.
2. Slip detection job and events.
3. Customer portal domain: `customer_portal_users`, invitations, magic-link sessions (short-lived, device-bound, revocable), per-contact permissions (view, quote, order, pay), audit.
4. Portal app: a new route tree in `client/apps/web` under a public layout or a new `apps/portal`. Shared components from `@trenova/shared`.
5. Notification subscriptions and templates; SMS via `SendSMSWorkflow`.

**UX and UI.**
- Portal home: shipments needing attention first (late, exceptions), then in transit, then delivered; each row shows the ETA with its band.
- Shipment page: map, timeline of milestones, the ETA explanation in one line with a "why" disclosure, documents, messages.
- Quote: origin, destination, date, equipment; a price in under two seconds with what it includes; one click to book.
- Internal: shipment detail shows the same ETA explanation; a portal activity panel shows what the customer viewed.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `get_shipment_eta` | read | | ETA, band, basis and history |
| `list_shipments_at_risk` | read | | Slipping shipments with cause |
| `get_customer_portal_activity` | read | | What a customer viewed or asked |
| `send_customer_eta_update` | action | Propose | Proactive update to a customer (external) |
| `invite_customer_portal_user` | action | Propose | Invite a contact |
| `reply_customer_portal_message` | action | Propose | Reply in a shipment thread |

- Events: `shipment.eta_slipping`, `shipment.eta_recovered`, `portal.quote_requested`, `portal.message_received`.
- Watchtower source extension for slipping shipments.
- Templates: the existing CustomerUpdateDesk gains portal and ETA tools; a new **Customer Concierge** answers portal messages with read tools and drafts replies.

**Acceptance criteria.**
- For a seed load crossing an HOS reset, the ETA includes the reset and says so.
- ETA accuracy (absolute error at 4 h before arrival) is measured and shown on a report.
- A customer contact receives a magic link, sees only their shipments, downloads a POD, and requests a quote that prices from their rate agreement.
- A slip raises an event at least one hour before the appointment in a simulated run.

**Dependencies.** G03 for live positions beyond Samsara.

**Decisions for Eric.** Portal as routes in `apps/web` or a new `apps/portal`. Recommended: a new app, because it has a different auth model, bundle budget and branding.

---

### G10. Carrier portal and onboarding

**Why.** Rose Rocket has a passwordless carrier portal with onboarding packets, tasks and invoices ([carrier portal](https://help.roserocket.com/platform/carrier-portal-rose-rocket)); Alvys sends an emailed onboarding packet with agreement and ACH details; McLeod works through Highway, MyCarrierPortal or RMIS. Trenova's carrier vetting is deeper than all of them (CarrierOK and FMCSA polling, a rule catalog, gates at assignment), but carriers cannot self-onboard or upload a COI.

**Today in code.**
- Carrier fields: `PaymentMethod` (Check or ACHManual), `W9OnFile`, `Is1099Eligible`, `RemitTo*`; `CarrierInsurancePolicy`.
- `carrierintelservice` capability flags and vetting rules; tender and rate confirmation public pages (`PreviewByToken`/`ConfirmByToken`).
- Carrier invoice matching exists.

**Market bar.** A carrier fills a web form, uploads documents, e-signs an agreement, and a person checks it all by hand.

**Trenova bar.**
- **Self-onboarding in ten minutes**: a carrier enters an MC or DOT number; Trenova pre-fills everything public (FMCSA, CarrierOK), runs vetting live, and asks only for what is missing: W-9 (extracted by G04 and TIN-matched where the IRS TIN matching service is available), COI (extracted, limits and expiry checked against the tenant's requirements, certificate holder verified), agreement e-signature, bank details (G11), and contacts.
- **Fraud defence**: identity checks on the person onboarding (email domain versus FMCSA contact, phone verification), VIN and equipment checks where available, and alerts on changes to bank details after onboarding (a classic double-brokering and payment fraud vector) with a cooling-off hold.
- **E-signed rate confirmations** with an audit certificate (signer, time, IP, document hash).
- **Carrier workspace**: offered and awaiting loads (accept or counter), active loads with document upload, invoices with status matched through carrier invoice matching, quick-pay offer and payment status.
- **Continuous compliance**: COI expiry reminders, automatic requests for renewal, and assignment blocked with an explanation when coverage lapses.

**Scope.**
1. Carrier portal users, invitations and magic-link sessions (reuse G09's session model).
2. Onboarding workflow state machine, requirement profiles per tenant, and document requirement checks via G04.
3. E-signature service (build natively with hashing and audit certificate PDF; or integrate a provider, see decisions).
4. Carrier invoice submission feeding carrier invoice matching.
5. Bank detail change controls shared with G11.

**UX and UI.**
- Portal onboarding is a progress stepper with pre-filled sections marked "From FMCSA" and only the missing items highlighted.
- Internal carrier page gets an Onboarding `SectionPanel`: checklist with status per requirement, extracted COI limits against requirements, vetting findings, and approve or request changes.
- Onboarding queue list page with filters and bulk approve for fully passing carriers.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `get_carrier_onboarding_status` | read | | Checklist, findings and blockers |
| `list_carriers_pending_onboarding` | read | | Queue |
| `invite_carrier_to_portal` | action | Propose | Invite by email (external) |
| `approve_carrier_onboarding` | action | ActWithApproval | Approve when all requirements pass |
| `request_carrier_document` | action | Propose | Request a missing or expiring document |
| `hold_carrier_bank_change` | action | AutoExecute | Place a cooling-off hold on a bank detail change |

- Events: `carrier.onboarding_submitted`, `carrier.coi_expiring`, `carrier.bank_details_changed`, `carrier.invoice_submitted`.
- Watchtower source `CarrierCompliance` extension.
- Template: **Carrier Onboarding** agent (Propose, shadow).

**Acceptance criteria.**
- A seed carrier completes onboarding with a real-format COI and W-9; extracted limits are checked against requirements and shown with provenance.
- Changing bank details after approval triggers a hold and a Watchtower item.
- A signed rate confirmation produces a certificate with document hash, signer, IP and time.

**Dependencies.** G04 (extraction). Soft on G11 (bank details).

**Decisions for Eric.** Native e-signature or a provider (DocuSign, Dropbox Sign). Recommended: native for rate cons and agreements, since the audit certificate is straightforward and keeps the flow in Trenova.

---

### G11. Payouts: pay people and carriers, safely

**Why.** Trenova records "paid" and "instant pay" without moving money: no ACH or NACHA file, no quick pay, no 1099s. DataTruck produces NACHA files for Chase and Bank of America and integrates ADP, Stripe and Plaid (sources in the report); Rose Rocket pays through Rainforest, TriumphPay and Comdata; McLeod through TriumphPay, Relay, Comdata and EFS.

**Today in code.**
- Driver settlements (nine pay methods, escrow, advances, deductions, disputes) and carrier settlements exist; remittance and payroll CSV exports exist; `PayWorkerNow` and `MarkPaid` record payments only.
- No bank account fields exist anywhere.

**Market bar.** Export a NACHA file and upload it to the bank; or a payments partner integration with a pay button.

**Trenova bar.**
- **Encrypted bank accounts** for workers and carriers, captured by the payee (portal or driver app) with micro-deposit or instant verification, with change controls (G10's cooling-off hold, notification to the payee's previous contact).
- **Payment runs with dual control**: a run is prepared, previewed (payees, amounts, changes since last run, anomalies such as first-time payees or amounts over history), approved by a second person, then released. Separation of duties is enforced.
- **Rails** through a `PaymentProvider` port: NACHA file generation (per bank's specification, balanced or unbalanced), a direct ACH or RTP provider API, and virtual card or fuel card funding for advances.
- **Quick pay** for carriers with fee schedules and automatic deduction; **instant pay** for drivers where the rail supports it.
- **Positive pay** file for checks, **1099-NEC** generation and e-filing (via an IRS FIRE-compatible file or provider), and year-end TIN checks.
- Everything posts to the GL through journal sources.

**Scope.**
1. `bank_accounts` (encrypted, tokenized display), verification state, change log.
2. `payment_runs`, `payment_run_items`, approvals, release, returns handling (R01, R02 and so on reversing the item and reopening the payable).
3. NACHA writer in `shared/` with validation tests against bank specs; `PaymentProvider` adapters.
4. 1099 generation from settlement and carrier pay data with corrections.
5. Permissions for prepare, approve and release as separate operations.

**UX and UI.**
- **Payments** page: `KpiStrip` (due this week, in current run, awaiting approval, returned), runs table, run detail with anomalies highlighted and a dual-control approval bar.
- Payee detail shows bank account masked with verification state and change history.
- 1099 workspace at year end with a checklist and corrections.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `get_payment_run` | read | | Run with items and anomalies |
| `list_payments_due` | read | | Payables due by date |
| `find_payment_anomalies` | read | | First-time payees, bank changes, outliers |
| `prepare_payment_run` | action | Propose (never Auto) | Build a run for a date |
| `request_bank_verification` | action | Propose | Ask a payee to verify |

No agent tool releases money. Release is a human action with step-up authentication (G12).

- Events: `payment_run.prepared`, `payment.returned`, `bank_account.changed`.
- Watchtower source `Payments` (returns, runs awaiting approval, unverified payees due).
- Template: **Payables Preparer** (Propose only).

**Acceptance criteria.**
- A generated NACHA file validates against a reference parser and the bank's test harness where available.
- A run cannot be released by the person who prepared it.
- A returned payment reverses in the GL and reopens the payable with the return reason.

**Dependencies.** G12 for step-up auth on release. Soft on G10.

**Decisions for Eric.** Which rail first: NACHA file only, or a provider API. Recommended: NACHA first (works with any bank), then one API provider.

---

### G12. Identity: passkeys, MFA, SAML and SCIM

**Why.** Trenova has OIDC SSO and 179 permission resources, but no native MFA, and SAML is modeled but disabled. Rose Rocket has SOC 2 Type II and SSO/SAML on Enterprise; Alvys and McLeod have MFA. Enterprise security questionnaires stop at "Do you support MFA and SAML?".

**Today in code.**
- MFA comes only from OIDC claims; `iam.MFAAuthenticator` exists with read-only endpoints. SAML is rejected in `iamservice`. SCIM is admin configuration only, with no `/scim/v2` server.

**Market bar.** TOTP MFA and SAML SSO on enterprise plans.

**Trenova bar.**
- **Passkeys first** (WebAuthn, platform and roaming authenticators), TOTP as fallback, recovery codes, and admin-enforced policies per role.
- **Step-up authentication** for sensitive actions: releasing payments, changing bank details, creating API keys, changing permissions, exporting large data sets.
- **SAML 2.0 SP** with metadata import, signed assertions, encrypted assertions, JIT provisioning, and group-to-role mapping, next to the existing OIDC.
- **SCIM 2.0 server** (`/scim/v2/Users`, `/Groups`) with Okta, Entra ID and Google compatibility tests.
- **Session security**: device list, revoke, idle and absolute timeouts per policy, new-device alerts.
- **SOC 2 evidence export**: access reviews, MFA coverage, audit log extracts, change history, generated on demand.

**Scope.** WebAuthn and TOTP enrolment and verification; policy engine; step-up middleware for GraphQL and REST; SAML SP; SCIM server with filtering and patch; session management; evidence exports.

**UX and UI.** Security settings with policy toggles and coverage (`KpiStrip`: users with passkeys, TOTP only, no MFA); a user security page (devices, factors, recovery); step-up as a compact dialog that remembers the verification for a short window.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `get_mfa_coverage` | read | | Coverage by role |
| `list_access_review_items` | read | | Users with stale or excessive access |
| `generate_access_review` | action | AutoExecute | Produce a periodic review for approval |
| `revoke_user_session` | action | Propose | Revoke a session (security) |

- Events: `security.mfa_disabled`, `security.new_device_login`, `security.scim_sync_failed`.
- Watchtower source `Security`.
- Template: **Access Reviewer** (quarterly reviews, Propose).

**Acceptance criteria.** Passkey sign-in works in Chrome, Safari and Firefox; Okta and Entra SAML and SCIM pass their integration test suites; step-up blocks payment release without a fresh factor.

**Dependencies.** None.

**Decisions for Eric.** Whether MFA is required by default for all new tenants. Recommended: yes for admin roles.

---

### G13. Safety, claims and CSA

**Why.** Trenova computes a 7-BASIC CSA roll-up from events entered by hand. McLeod imports CSA BASICs from FMCSA and handles accidents, claims and OS&D ([CSA](https://www.truckinginfo.com/articles/latest-mcleod-software-versions-include-csa-analytics)); Rose Rocket has a driver scorecard.

**Today in code.**
- `workersafetyservice` with `SafetyEventKind` Accident, Incident, NearMiss, Citation, Inspection; `csa.go`; `fleetsafety.go`.
- `shared/fmcsa` has a `Basics()` method (percentiles and thresholds) usable for the tenant's own DOT number.
- `servicefailure` has no damage or OS&D type.

**Market bar.** Accident and claim records, imported CSA scores.

**Trenova bar.**
- **Own-DOT SMS import**: BASIC percentiles, thresholds, inspections and violations pulled on FMCSA's refresh cycle, linked to drivers, units and loads.
- **CSA what-if**: see which inspections drive each BASIC and how scores change as violations age out; target coaching where it moves the percentile.
- **DataQs workflow**: flag challengeable violations with reasons, assemble evidence (telematics, DVIR, documents), track submissions and outcomes.
- **Claims and OS&D**: cargo, damage, shortage, refusal and accident claims, fed by G04 OS&D detection, with reserve, insurer, deductible, recovery from carriers or customers, and GL posting.
- **Accident register** meeting 49 CFR 390.15 with preventability decisions.
- **Driver risk score** combining events, telematics harsh events, inspections and hours, explained, with coaching assignments.

**Scope.** FMCSA import job; inspection and violation tables linked to assets; what-if engine; DataQs case model; claims domain with financial postings; risk scoring; coaching tasks.

**UX and UI.** Safety home with `KpiStrip` of BASIC percentiles against thresholds, trend lines, top contributing violations; claim detail with timeline, documents, reserves and recoveries; driver safety tab with the explained score.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `get_csa_scores` | read | | BASICs with contributors |
| `simulate_csa_what_if` | read | | Score projection over time |
| `list_dataqs_candidates` | read | | Challengeable violations |
| `list_open_claims` | read | | Claims with reserves and age |
| `open_claim` | action | ActWithApproval | Create a claim from an event or document |
| `assign_driver_coaching` | action | Propose | Assign coaching |
| `draft_dataqs_challenge` | action | Propose | Assemble a challenge (external) |

- Events: `safety.basic_threshold_crossed`, `safety.inspection_imported`, `claim.opened`.
- Watchtower source `Safety` extension.
- Template: **Safety Analyst**.

**Acceptance criteria.** Import for a real public DOT number matches the SMS site values; a POD OS&D finding opens a claim with the document attached; the what-if projects the score at 6, 12 and 24 months.

**Dependencies.** Soft on G04 and G03.

**Decisions for Eric.** None expected.

---

### G14. Maintenance

**Why.** Trenova has no work orders or preventive maintenance; Samsara DVIRs are ingested and go nowhere; "AtMaintenance" is a status with nothing behind it. DataTruck has a maintenance board and asset health; Alvys and McLeod use partners (Fleetrock, TMT, Cetaris).

**Today in code.**
- `AtMaintenance` equipment status unused; `SequenceTypeWorkOrder` unused.
- `vehicle_inspections` (DVIR) are upserted only. Tractors have no odometer. `dispatcheligibility` has no DVIR check.

**Market bar.** Work orders, PM schedules by miles or date, and DVIR defects listed.

**Trenova bar.**
- **Closed DVIR loop**: a defect creates a work order automatically, the unit is blocked from dispatch by `dispatcheligibility` until a certified repair, and the driver sees the repair certification in the app (G05).
- **PM by miles, engine hours or time**, driven by G03 odometer and engine hours, with due-soon forecasts based on planned loads ("Unit 214 hits its 25,000-mile PM on Thursday's load to Denver; schedule it at the Denver vendor").
- **Fault codes** from telematics with severity and suggested action; recurring codes raise a predictive finding.
- **Work orders** with parts, labour, vendor, warranty and cost posting; vendor bills via G04 extraction.
- **Cost per mile** per unit and fleet, and a keep-or-replace analysis.
- **Planning awareness**: dispatch and the ALNS planner avoid assigning units due for PM inside the planning horizon.

**Scope.** Work order domain and sequence; PM programs; odometer and hours history; fault code handling; eligibility rule; costs and GL; vendor shops.

**UX and UI.** Maintenance board by unit (due, overdue, in shop); unit maintenance tab with timeline; work order detail with `FormSection` and `FormSaveDock`; PM forecast calendar.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `list_maintenance_due` | read | | Units due or overdue |
| `get_unit_maintenance_history` | read | | Work orders, PMs, faults, cost per mile |
| `create_work_order` | action | AutoExecute for DVIR defects, else ActWithApproval | Open a work order |
| `schedule_preventive_maintenance` | action | Propose | Schedule a PM around planned loads |
| `mark_unit_out_of_service` | action | ActWithApproval | Block a unit |

- Events: `maintenance.defect_reported`, `maintenance.pm_due`, `maintenance.fault_critical`.
- Watchtower source `Maintenance`.
- Template: **Shop Planner**.

**Acceptance criteria.** A DVIR defect blocks the unit from dispatch within one minute and creates a work order; PM forecasts use planned-load miles; cost per mile reconciles with the GL.

**Dependencies.** G03 (odometer and faults). Soft on G05 (DVIR in the app).

**Decisions for Eric.** None expected.

---
## Wave 3: extend the lead and reach enterprise depth

Wave 3 briefs are shorter because their shape depends on what waves 1 and 2 build. The agent that picks one up must expand it in its plan to the full wave 1 shape (Today in code verified, Market bar, UX and UI, complete tool table) before building. Parts 1 and 2 apply in full.

---

### G15. Reporting: from Power BI lite to Power BI class

**Why.** The report builder is already a Trenova lead: build from scratch over 70 entities with automatic joins, pivots, parameters, drill-through, conditional formatting, 24-tile dashboards with cross-filtering, schedules, alerts, and reports built from a sentence in Desk. No competitor has an in-product builder at this depth (Rose Rocket copies 15 Luzmo templates; Alvys sells a metrics catalog add-on; DataTruck and McLeod send you to Power BI). What remains: time intelligence, reusable measures, and a way out to warehouses and external BI, which Rose Rocket (Snowflake, BigQuery, Fabric) and McLeod (McLeod IQ star schema) offer ([RR analytics](https://www.roserocket.com/solutions/analytics)).

**Today in code.**
- Report executor: `ComputedOp` is add, subtract, multiply and divide only; `DateBucket`; `TransformOp`. The executor runs on `ReportingConnection`, which points at the same primary database.
- `reportcatalog.yml` curates entities and fields; GTC publishes `reporting:cdc`.
- An expr-lang formula engine already exists for rating formulas (`formulaassistantservice` and the formula template system).

**Market bar.** Template reports with light editing, or a Power BI connection that someone else builds reports in.

**Trenova bar.**
- **Time intelligence**: period over period, year to date, rolling N, same period last year, and fiscal calendars from Trenova's fiscal periods, all as one-click measure modifiers.
- **Measures library**: named, versioned, documented measures ("Revenue per loaded mile", "Deadhead %") defined once with the existing expr-lang formula engine and reused across reports, dashboards, alerts and Desk. Certified measures carry a badge.
- **Richer maths**: percentiles, medians, window functions (rank, running totals, moving averages), conditional aggregates, and safe division.
- **Performance and isolation**: route reporting to a read replica, with an optional columnar store (for example DuckDB over Parquet or ClickHouse) fed by `reporting:cdc` for heavy tenants; query cost limits and caching.
- **Out to anywhere**: scheduled exports to Snowflake, BigQuery, Fabric and S3 as a star schema; a read-only SQL endpoint (Postgres wire) for Power BI, Tableau and Excel with row-level tenant security.
- **Sharing**: share with roles and people, embed a dashboard in the customer portal (G09) with row-level filters, and pixel-perfect PDF dashboards for board packs.
- **More visuals**: combo charts, waterfall, heatmap, funnel, sankey for lane flows, and small multiples.

**Scope.** Measure domain and editor; time intelligence in the query planner; window and percentile functions; replica routing and optional columnar store; warehouse export jobs; SQL endpoint with RLS; sharing and embed tokens; PDF rendering; new chart types.

**UX and UI.** A measure editor with formula autocomplete, live preview on sample data and a plain-language description; a "Compare to" control on every chart; a Share dialog with roles and embed; chart picker with previews rendered from the current data.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `list_measures` | read | | Measures with definitions and certification |
| `run_report_with_measures` | read | | Execute an ad-hoc query using measures and time intelligence |
| `explain_metric_change` | read | | Decompose a change by dimension ("Revenue per mile fell 6%: lane mix 4%, fuel 2%") |
| `create_measure` | action | ActWithApproval | Define a measure |
| `share_dashboard` | action | Propose | Share or embed (can be external) |

- Artifact: a `chart` artifact in Desk if G08 has not added it.
- Template: **Analyst** agent that writes a weekly narrative for each dashboard with explained changes.
- Events: `report.alert_triggered` (exists as alerts; expose as an agent event if not already).

**Acceptance criteria.** "Revenue YTD versus last YTD by customer" is three clicks; a measure changed once updates every report using it; Power BI connects to the SQL endpoint and sees only the tenant's rows; heavy reports no longer touch the primary.

**Dependencies.** None.

**Decisions for Eric.** Columnar store choice. Recommended: start with a read replica; add DuckDB over Parquet for self-hosters and ClickHouse for the hosted service only if measured load requires it.

---

### G16. Accounts payable, budgets and bids

**Why.** McLeod has a full ledger with AP, segment accounts and budgets ([accounting](https://www.mcleodsoftware.com/accounting-factoring-finance/)) and bid management ([bids](https://www.mcleodsoftware.com/bid-management-logistics-brokerage-3pl/)). Rose Rocket has AP and payroll. Trenova's GL has no AP, budgets or segments; `DefaultAPAccountID`, `VendorBillPosted` and `VendorPaymentPosted` are defined and unused.

**Today in code.** The only AP-like code is carrier invoice matching. The GL, fiscal periods and journal sources exist. Rating engine, costing and rate simulation exist for bids.

**Trenova bar.**
- **AP with zero entry**: vendor bills arrive by email or upload, G04 extracts them, lines are coded to GL accounts and equipment or loads by learned rules, three-way matched where there is a PO or work order (G14), approved through routing rules, and paid through G11.
- **Budgets and segments**: budgets by account, department, terminal and equipment class, with variance reporting in the report builder.
- **Bids and RFPs**: import a shipper's lane RFP spreadsheet (any layout, mapped by the assistant), price every lane with the rating engine, costing and market rates (G06), show margin and capacity fit per lane, and export the response in the shipper's format. Awarded lanes become rate agreements automatically.

**AI surface.**

| Tool | Kind | Tier | Purpose |
|---|---|---|---|
| `list_vendor_bills_pending` | read | | Bills awaiting coding or approval |
| `code_vendor_bill` | action | AutoExecute above confidence, else Propose | Apply GL coding |
| `approve_vendor_bill` | action | ActWithApproval | Approve for payment |
| `get_budget_variance` | read | | Variance by segment |
| `price_rfp` | action | AutoExecute | Price an imported RFP (internal draft) |
| `submit_rfp_response` | action | Propose | Send the response (external) |

- Events: `vendor_bill.received`, `budget.variance_exceeded`, `rfp.imported`.
- Watchtower source `Payables`. Templates: **AP Clerk**, **Bid Desk**.

**Acceptance criteria.** A seed repair invoice arrives by email, is coded to the unit's maintenance account, matched to its work order and approved in two clicks; a 500-lane RFP prices in under a minute with per-lane explanations.

**Dependencies.** G01, G04, G06, G11.

**Decisions for Eric.** None expected.

---

### G17. Workflow builder

**Why.** Rose Rocket has a workflow builder, custom objects and formula fields; McLeod has FlowLogix visual workflows ([FlowLogix](https://www.mcleodsoftware.com/flowlogix-truckload-carriers/)). Trenova has custom fields on six resources, saved views and custom agents, but no deterministic rules builder.

**Trenova bar.**
- **One builder, two kinds of step**: deterministic steps (condition, set field, create record, notify, wait, call tool) and agent steps (hand a judgement to an agent with a goal), both using the same tool registry and governance. Deterministic where rules suffice; agents where judgement is needed.
- **Triggers** from agent events, CDC changes, schedules, custom events (G08) and webhooks.
- **Test by replay**: run a workflow against the last 30 days of real events and see every path taken before enabling it.
- **Versioning, run history and per-run trace** with the inputs of every step.
- **Custom fields on every resource** and custom objects with permissions, reportable and tool-accessible automatically.

**AI surface.** `list_workflows`, `get_workflow_run` (read); `draft_workflow` (Propose, from a description), `enable_workflow` (ActWithApproval). Events `workflow.failed`. Watchtower source `Workflows`. The **Agent Architect** template (G08) can build workflows.

**UX and UI.** A canvas with a vertical flow (not free-form spaghetti), each step a card; a side panel for configuration; a replay drawer with a timeline of runs.

**Acceptance criteria.** A workflow "when a load is marked delivered without a POD after 2 hours, request it from the driver, then escalate to the dispatcher after 4 more hours" is built in the UI, replayed against seed history, enabled, and runs.

**Dependencies.** G08.

**Decisions for Eric.** Temporal as the runtime (recommended, already in the stack).

---

### G18. LTL consolidation, terminals and cross-dock

**Why.** McLeod LoadMaster LTL covers cross-dock, linehaul, pickup and delivery and dock scanning ([LTL](https://www.mcleodsoftware.com/who-we-serve/ltl-carriers/)); Rose Rocket does consolidation, leg splitting for cross-dock, manifests and SMC3 rating ([LTL](https://www.roserocket.com/blog/quick-guide-to-ltl-trucking-software)). Trenova's LTL rating is real, but the consolidation tables have no code behind them and there are no terminals.

**Today in code.** Consolidation tables exist unused; `dedicatedlane` is orphaned; LTL rating (freight class, density, deficit) works; ALNS planner exists in `shared/dispatchplanner`.

**Trenova bar.** Terminals and doors; pickup and delivery routes; linehaul schedules; consolidation proposed by the ALNS planner (cube, weight, class compatibility, service commitments) with explained savings; cross-dock with dock scanning from the driver or dock app; manifests; SMC3 or equivalent rating. Decide the fate of `dedicatedlane` (complete it as dedicated fleet contracts or remove it with a migration).

**AI surface.** `propose_consolidation` (Propose), `list_terminal_dock_status` (read), `build_linehaul_manifest` (ActWithApproval), events `consolidation.proposed`, template **Consolidation Planner**.

**Acceptance criteria.** Twenty seed LTL shipments consolidate into fewer moves with the saving and constraints explained; a cross-dock transfer scanned at a door updates both legs.

**Dependencies.** None.

**Decisions for Eric.** Whether LTL carriers are a target segment now; this brief is sized for it.

---

### G19. Canada and tax

**Why.** Rose Rocket files ACE and ACI eManifests through BorderConnect ([ACE](https://help.roserocket.com/borderconnect-how-do-i-file-an-ace-from-within-rose-rocket)); McLeod has multicurrency and GST/HST. Trenova has US states only (though IFTA jurisdictions include provinces), no invoice tax fields, and `TaxExempt` only on the billing profile.

**Trenova bar.** Provinces and territories as first-class jurisdictions; tax engine for GST, HST, QST and PST with registration numbers, exemptions and correct invoice presentation; tax codes synced with G01; realized and unrealized FX gains and losses; ACE and ACI eManifest filing through a provider with status tracking; Canadian HOS already exists and should be surfaced; fr-CA translation.

**AI surface.** `get_invoice_tax_breakdown` (read), `file_emanifest` (Propose), `list_border_crossings_pending` (read); events `emanifest.rejected`; Watchtower source `CrossBorder`.

**Acceptance criteria.** An Ontario-to-Quebec load invoices with correct HST or GST plus QST; an ACE manifest is filed in a provider sandbox and its acceptance tracked; the app runs fully in fr-CA.

**Dependencies.** G01 for tax code sync.

**Decisions for Eric.** eManifest provider (BorderConnect or another).

---

### G20. Responsive web

**Why.** Dispatchers and owners check Trenova from phones. Competitors' web apps are mostly desktop-only; being excellent on a phone is a way to set the standard, not meet it.

**Trenova bar.** Every read view and the most common actions (approve decisions, reply to messages, check a load, approve a payment run with step-up) work well at 390 px wide: tables collapse into cards with the key fields, the decision queue becomes swipe-to-approve, Desk works one-handed, and nothing scrolls sideways. Tablet layouts for dispatch boards.

**Scope.** A responsive audit of every route in `client/apps/web` with a checklist; table-to-card variants in `@trenova/shared` data table; mobile navigation; Playwright visual tests at 390 px and 820 px.

**AI surface.** None new; Desk and the decision queue must be first-class on mobile.

**Acceptance criteria.** The top 30 routes by usage pass the audit and visual tests; no horizontal scroll at 390 px.

**Dependencies.** None.

**Decisions for Eric.** None expected.

---

## Appendix. Evidence index

Competitor claims above come from the sources linked inline and from the full comparison report: [Trenova vs DataTruck, Rose Rocket, Alvys and McLeod](https://claude.ai/artifact/Kzgmuxeri1agVefxDezQyD). Trenova facts were read from the repository on 2026-09-22; agents must re-verify them (rules of engagement, item 2). Two planning documents in the repo are partly stale and should not be trusted without checking code:

- `docs/operations-guides/enterprise-gap-closure-plan.md`: its status table lists WS1, WS3 and WS4 as open, but they have landed.
- `research-docs/trenova-enterprise-tms-gap-analysis-2026-09.md` (2026-09-04): drug and alcohol, IFTA, fuel cards, safety and CSA, PTO accrual, the DispatchAssignment agent and equipment eligibility have landed since.

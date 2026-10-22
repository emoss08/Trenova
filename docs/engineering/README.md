# Engineering docs

Each doc records how a subsystem works and the invariants that compile, pass every test and
still break production. **Before changing a subsystem, find it here and read its doc.** When a
change alters what a doc says, update the doc in the same change; a doc that disagrees with the
code is a bug.

How to write backend code at all (layers, data access, services, APIs, jobs, scale, testing) is
in [backend/](backend/README.md). Frontend rules are in the `trenova-frontend` skill
(`.claude/skills/trenova-frontend/SKILL.md`).

## Platform

| Doc | Read before changing | The rule that bites |
|---|---|---|
| [module-architecture](module-architecture.md) | adding a module, entity, port, service or job; anything exported and generic | a module imports only lower modules' `domain`, `ports`, `<name>contract`; no exported API mentions a concrete generic type with methods |
| [row-level-security](row-level-security.md) | a table, a repository method, a background job, anything before a tenant is known | `dbtx.Read`/`dbtx.Write` + `DBForContext(ctx)`; migrations end with `SELECT trenova_rls.reconcile();`; cross-tenant code is `dbscope.WithSystem` with a reason and allow-listed in `rlslint` |
| [permissions](permissions.md) | a resource or operation, a check, a data scope, a way of granting access, cache invalidation | registry in `domain/permission/resources_*.go`; invalidate by `PermissionChangeNotifier` epoch, never key deletes; admins get new operations from `trenova db migrate` |
| [reference-views](reference-views.md) | showing another kind's record on a screen, a GraphQL type carrying one, the page access manifest | a linked record shows its reference view; more needs `@requiresRead(kind:)`; a page or import change regenerates the manifest (`access:generate`, then refgen) |
| [record-kinds](record-kinds.md) | a kind of record, any table keyed by kind | the permission registry owns the list; never a hand-kept list of kind names |
| [security-audit](security-audit.md) | a mutation that changes who can do what, a new way to read a document, a cross-tenant refusal, deleting audit rows | `SecurityAuditor.RecordChange` after commit; `auth_events`/`audit_entries` are append-only |
| [idempotency](idempotency.md) | the middleware, `pkg/idempotency`, a write that must be safe to retry | a 5xx or transient GraphQL error releases the key |
| [realtime](realtime.md) | `realtimeservice`, `realtimebroker`, the stream endpoint, the browser client, presence | publish with `PublishResourceInvalidation`; `AudienceUserID` for one person |
| [editions](editions.md) | anything the Cloud overlay replaces or extends | public code never imports `internal/cloud`; decide behaviour from an injected port, never `platform.mode` |
| [generated-artifacts](generated-artifacts.md) | a GraphQL schema, a removed column, user-facing text | `projection.yml` needs every field to resolve to a column, relation or declared virtual |
| [graphql-executor](graphql-executor.md) | the gqlexec generator or runtime | `task gqlgen`, never `go tool gqlgen generate` |
| [bun/buncolgen](bun/buncolgen.md) | any repository code | generated column helpers only, never hand-written column names |
| [rust](rust.md) | any Rust code | services build on `trenova-service-kit`; untrusted input is parsed in `trenova-sandbox` |
| [rules-engine](rules-engine.md) | `services/crates/rules`, `pkg/rules`, `client/packages/shared/src/rules`, any caller of the formula language, allocation, report formatting, lanes or rate matrices | one artifact for both hosts: rebuild with `task rust:wasm-rules` and commit both copies, never edit `rules.wasm`; one call per request, never one per item |
| [imaging-worker](imaging-worker.md) | the worker, its proto, `infrastructure/imaging`, `pagerender`, anything that renders or assembles a document; `blob.go`, `storedobject` | no PDF or image library in Go; stored objects go through `storedobject` as streaming sealed blobs |

## Operations domain

| Doc | Read before changing | The rule that bites |
|---|---|---|
| [tracking-events](tracking-events.md) | `trackingevent`, `ReconcileStopActuals`, telematics stop visits, the EDI 214 parser, anything recording a stop actual | people through `RecordReported`, machines through `RecordObserved`; never `RecordStopActual` for a new source |
| [check-calls](check-calls.md) | `checkcall`, the carrier tracking link and page, the no-check-call sweep | arrivals and departures also go through tracking events |
| [eta](eta.md) | `stopeta`, `etaprojection`, `shipmentetaservice`, `etajobs`, `etatrigger`, the route estimator | everything reads the stored `stop_etas` rows; refreshes are debounced per shipment |
| [customer-tracking](customer-tracking.md) | `customertracking*`, public tracking endpoints, status email templates | one `customer_status_notifications` row per shipment, event and stop, sent exactly once |
| [tender-negotiation](tender-negotiation.md) | `tender`, `tenderservice`, `tenderjobs`, tender routes and settings | one workflow per round; a bounce inserts new offer rows; the carrier is paid `AgreedRate()` |
| [exception-queue](exception-queue.md) | `watchtower`, `watchtowerservice`, exception sources, settings or tools | a source never overwrites owner, status or resolution |
| [formula-language-service](formula-language-service.md) | the Formula Studio's editor, its linter, hover, signature help or format command, `analyzeExpression`, a consumer of `formula.*` in the browser | the language lives in `services/crates/rules` and is never re-implemented in TypeScript; call the module only after `useRules()` has loaded it, and show its messages with `translateMessage` |

## Agents and AI

| Doc | Read before changing |
|---|---|
| [agent-runtime](agent-runtime.md) | `agentruntime/`, `assistantservice/`, `jobs/{agentflow,modelcall,agentjobs,assistantjobs,completionjobs}`; reflection ("Learning from the work") |
| [agent-tool-conventions](agent-tool-conventions.md) | adding, renaming or merging a tool, or rewriting its description (no aliases on rename) |
| [agent-write-coverage](agent-write-coverage.md) | any new mutation or write route: it needs a tool, an exemption or a `pending` entry in `writecoverage.yml` |
| [agent-graphql-reads](agent-graphql-reads.md) ([inventory](agent-graphql-reads-inventory.md)) | a read tool, `agentread`, `gqlext`, `reads.yml`, the field guard, an operation a tool reads through |
| [proposal-previews](proposal-previews.md) | `toolpreview`, `proposalpreviewservice`, a tool's `Preview`, how a proposal is decided |
| [ai-tool-safety](ai-tool-safety.md) | generated from tool policies; never edit by hand |
| [agent-extensions](agent-extensions.md) | extensions and how they are gated and metered |
| [agent-portal-automation](agent-portal-automation.md) | `portal_tasks.go`, `portal_sites.go`, `portaljobs`, `portaltools`, `browseruseconnector` |
| [agent-web-watching](agent-web-watching.md) | `web_watches.go`, `webwatchtools`, `webwatchjobs`, `agentwaitservice/webwatch.go`, `shared/exa/monitors.go` |
| [agent-data-analysis](agent-data-analysis.md) | `analysis.go`, `analysis_tool.go`, `e2bconnector`, `shared/e2b` |
| [agent-code-mode](agent-code-mode.md) ([writes](agent-code-mode-writes.md)) | `domain/codemode`, `aiprovider/agentsettings.go`, `scriptbroker.go`, `scripttool.go`, `script_tool.go`, `RunCodeStream` |
| [agent-profiles](agent-profiles.md) | `agentprofileservice`, `agentprofilejobs`, the writing prompt, profile sections of the prompt |
| [agent-waits](agent-waits.md) | `agent_create_wait` and the wait kinds |
| [agent-delegation](agent-delegation.md) | `delegate_task` and the per-agent allowlist |
| [desk-cases](desk-cases.md) | Desk case state, snoozes, waiting on replies, the ready-to-bill checklist |
| [desk-artifacts](desk-artifacts.md) | a Desk artifact kind or resource card (build one with the `desk-artifact` skill) |
| [desk-bench](desk-bench.md) ([gate](desk-bench-gate.md)) | `internal/deskbench`, scenarios; run only against a seeded development database |
| [edi-spec-onboarding](edi-spec-onboarding.md) | `ediconnectiondraft`, `edi_draft_connection`, `edi_apply_connection_draft` |
| [agent-accounting-tools](agent-accounting-tools.md), [agent-rates-tools](agent-rates-tools.md), [agent-workforce-tools](agent-workforce-tools.md), [agent-master-data-tools](agent-master-data-tools.md), [agent-triage-tools](agent-triage-tools.md) | the tools in each area and who holds them |
| [ai-tracing](ai-tracing.md) | a span, a trace attribute or a link column (spans come from activities, never workflow code) |
| [ai-audit-trail](ai-audit-trail.md) | `aiauditservice`, `aiauditjobs`, `domain/aiaudit` (a field change is a `hash_version` change), any table the projector reads |
| [ai-analytics](ai-analytics.md) | `aianalytics*`, or what the completion router or step ledger records |
| [ai-training-export](ai-training-export.md) | `domain/aitraining`, the Cloud training packages, what a correction stores |
| [extraction-fine-tuning](extraction-fine-tuning.md) | the extraction prompt, reply schema, `aicorrection.Score`, `tools/extraction-finetune` |

## Client

| Doc | Read before changing |
|---|---|
| [design-system](design-system.md) | any styling, layout, colour, icon or loading state |
| [web-build](web-build.md) | `vite.config.ts`, `chunk-groups.ts`, the bundle budget, `main.tsx` imports, the service worker, the persisted query cache, the codegen hooks |
| [product-guide](product-guide.md) | a page, a button a guide names, a record link (`recordPath` only) |

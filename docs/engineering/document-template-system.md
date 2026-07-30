# Document & Message Template System — Implementation Handoff

**Status:** Phases 1–6 complete. Phases 7–8 not started.
**Branch:** `master`. Phases 1–6 are committed (unpushed): `e83048c3e`, `fc516f55d`,
then `06eb26612`, `ad6fc87c4`, `f98d90455`, `791d3f4e6`, `40eaf265b`, and the
`AttachNoticePDF` commit. Each was verified to build and `go vet` in isolation
(worktree checkout per commit), so the series is bisectable.

**If you are picking this up cold, read in this order:** §2 (the findings that
will otherwise cost you a day), §11 (the ordered plan), §10 (what to check before
committing, including files other sessions have silently reverted). Then the
section for the phase you are starting.

**Plan of record:** `~/.claude/plans/trenova-has-a-lot-polished-clarke.md` — read it, but prefer
this document where they disagree. Several plan assumptions turned out to be wrong; each is
corrected in place with a note saying so.

---

## 1. Why this exists

Trenova sends documents and emails to customers, drivers, and carriers, and a system
administrator cannot see or change any of them. Every layout and every word is compiled into
the binary:

- The customer invoice PDF was 1,342 lines of imperative `gofpdf` drawing at absolute
  millimeter coordinates (`invoiceservice/invoice_pdf.go`). There was no layout to expose.
- The report PDF layout was a 60-line `html/template` string literal inside
  `reporting/render/pdf.go`.
- Detention notices were `strings.Builder` prose wrapped in `<pre>`, with the HTML branch not
  escaping at all.
- Six email senders and ~30 in-app notification title/message pairs were hardcoded
  `fmt.Sprintf` calls across 18 files.

The goal: organizations author HTML/CSS document templates and email/notification templates in
an admin UI with live preview, versioning, publish, and rollback; documents render through a
network-isolated Gotenberg sidecar; a template can be assigned to a specific customer.

### Decisions already made — do not relitigate

These were settled with the project owner before implementation started.

| Decision | Choice |
|---|---|
| Rendering | HTML + CSS → PDF via a **network-isolated Gotenberg sidecar**. `chromedp` deleted. |
| Documents in scope | Invoice PDF, report PDF, detention notice (email + new PDF). |
| Messages in scope | 6 email senders + ~30 in-app notifications. **Not SMS.** |
| Assignment | Per-customer **whole-template** assignment. No slot merging or partial overrides. Resolution: assigned → org default → built-in. |
| Versioning | `document_templates` + `document_template_versions`, Draft/Active/Archived, publish + rollback. |
| Invoice cutover | **New modern HTML design** as the built-in. `gofpdf` and every `draw*` helper deleted. `buildInvoicePDFData` survives as the context builder. |
| Out of scope | XLSX and CSV customization; SMS. |

---

## 2. Findings that will cost you time if you do not read them

Each of these was discovered by a test failing, not by reading docs. They are the reason
several parts of the code look the way they do.

### 2.1 `html/template` rejects `data:` URIs

Its URL filter allows only `http`, `https`, and `mailto`. A `data:` URI passed as a plain
`string` into `<img src="{{ .LogoDataURI }}">` renders as `#ZgotmplZ`. Since the entire asset
strategy is "inline everything as a data: URI", **every document would have shipped with a
broken logo.**

Fields holding system-generated URIs are therefore typed `template.URL`:
`InvoiceContext.LogoDataURI`, `DetentionNoticeContext.LogoDataURI`,
`AgentEmailContext.LogoDataURI`, `DriverPortalInvitationContext.InviteURL`,
`ReportDeliveryEmailContext.DownloadURL`.

That is safe **only** because those values are machine-generated (base64 from bytes the asset
inliner already decoded, or a URL built from configured base + token) and are never user text.
If you add a `template.URL` field, document why it cannot carry user input.

Consequence: `Registry.HostileProbeContext` deliberately **skips** `template.URL` fields
(`trustedURLType` in `registry.go`). Poisoning them would report a failure that cannot happen
and would inject the probe unescaped into the output being tested.

### 2.2 `ZgotmplZ` is data-driven, not template-shaped

The plan claimed it catches unquoted attributes and `<style>` interpolation. It does neither.
It fires when a *value* cannot be escaped for its position — a dangerous URL scheme, or a CSS
value carrying punctuation. The same template renders cleanly with one value and yields
`ZgotmplZ` with another.

So `Engine.Verify` only catches this if the data you verify against is adversarial. That is
what `HostileProbeContext` is for, and why publishing should check **both** the sample and the
probe.

### 2.3 `html`, `js`, `urlquery`, and `printf` are builtins a FuncMap cannot remove

`{{ .Memo | html }}` parses no matter what you define. None of them can emit unescaped output,
but they let an author choose an escaping context, which is the one decision contextual
auto-escaping must own. They are rejected in `checkFunctions` on the parsed tree, which is
**not** dead code even though undefined functions fail at parse time. Do not delete it.

### 2.4 Fetched and navigated URLs need different rules

`src` is resolved by the renderer while building the page — that is the SSRF boundary, and only
`data:` is allowed. `href` is resolved by the reader later in their own client, so `https` is
both safe and necessary. Conflating them means the driver-portal invitation cannot contain its
invite link. See `fetchingURLAttributes` / `navigationURLAttributes` in `sanitize.go`.

### 2.5 Required paths are per *version*, not per channel

You publish a version, not a channel. A subject line cannot restate a charge table, and
`AgentSubject` only belongs in a subject. The check runs over the **union** of paths across all
of a version's channels, via `templateengine.CheckRequiredPaths`. It is deliberately not part
of `Parse`.

### 2.6 Gotenberg configuration traps

- **Do not set `--chromium-deny-list`.** The default is `^file:(?!//\/tmp/).*`, which already
  denies `file://` outside the staging directory. Overriding it blocks the `index.html`
  Gotenberg itself stages and **every conversion returns 403**.
- The uploaded file **must be named `index.html`**. Any other name returns
  `400 Invalid form data: form file 'index.html' is required`.
- `read_only: true` needs tmpfs for **both** `/tmp` and `/home/gotenberg`. Chromium's crashpad
  handler refuses to start without a writable database directory.
- **Do not set `--chromium-auto-start`.** A pre-warmed long-lived browser was observed to stop
  accepting conversions after ~10 minutes while `/health` still reported chromium `up` — the
  healthcheck cannot see this failure. Per-conversion startup costs little (~200ms renders
  either way). A 20-minute probe after removing it was 10/10 clean. If you ever re-enable it,
  add a probe that actually converts.
- The sidecar has **no published port in production** and must never be reverse-proxied. The
  local compose file publishes `3009` only because the API runs on the host under `air`.

### 2.7 PDF text extraction returns *rendered* text

`go-fitz` gives you what the reader sees. The stylesheet uppercases table headings, so assert
on `AMOUNT`, not `Amount`. This bit me once.

### 2.8 `shared/` must stay CGO-free

`gtc` and `samsara-sim` import it and build without a C toolchain. The only usable Go WebP
decoder needs CGO, so `imageutils` takes an injected `WebPTranscoder` and the tms-side
`assetinliner.NewWebPTranscoder()` supplies it. Do not add `chai2010/webp` to `shared/go.mod`.

### 2.9 Pre-existing bugs this work fixes or exposes

- **Fixed:** report PDF export was broken in Docker. `reporting/render/pdf.go` drove headless
  Chromium via `chromedp`, but `deploy/Dockerfile.tms` is bare alpine with no Chromium binary.
- **Fixed, and the fix has landed:** SSRF on the invoice logo. The
  organization-controlled `Organization.LogoURL` was fetched with
  `http.DefaultClient`. That fetch is deleted; Phase 5's context builder resolves
  the logo through `AssetInliner.ResolveImageDataURI`, which uses
  `shared/httpsafe` and re-checks every resolved address at dial time.
- **Fixed in Phase 6:** two unescaped `<pre>` interpolations in the detention
  notice — `SendNotice` and `BuildNotice` both wrapped operator-entered text as
  markup. Both are gone; see §7.
- **Still open, Phase 8:** `client/apps/web/src/components/elements/pdf-viewer.tsx:9` loads the
  pdfjs worker from `//unpkg.com`. Fails under a strict CSP or air-gapped deploy. `vite.config.ts`
  already `staticCopy`s pdfjs cmaps locally, so switch to a `?url` import.

---

## 3. What is shipped

### Commit `e83048c3e` — Gotenberg sidecar renderer

Verified independently bisectable (Phase 2 stashed, builds and tests clean).

| File | Role |
|---|---|
| `internal/core/ports/services/pdfrenderer.go` | Port. `ErrPDFRendererUnavailable` so a missing renderer is a business error, never a 500. |
| `internal/infrastructure/pdfrender/gotenberg/client.go` | Adapter. Semaphore, retry only on transport/429/502/503/504, PDF size cap, header-injection-safe filenames. |
| `internal/infrastructure/pdfrender/gotenberg/module.go` | fx module. Warns on unreachable renderer rather than failing startup. |
| `internal/core/domain/documenttemplate/enums.go` | `PageSize`, `Orientation`, `Margins` with mm→inch conversion. |
| `internal/infrastructure/reporting/render/pdf.go` | `chromedp` deleted; renders through the port. |
| `internal/infrastructure/config/config.go` | `RendererConfig` + getters. |
| `docker-compose-local.yml` | Hardened `gotenberg` service. |
| `internal/testutil/mocks/mock_PDFRenderer.go` | Hand-written (never run mockery). |

Both security controls are asserted against a real sidecar and checked non-vacuous: the same
template yields `SCRIPT-RAN` against a JS-enabled renderer and `SCRIPT-DID-NOT-RUN` against
ours.

### Commit `fc516f55d` — template engine, registry, starters, inliner

| Package | Tests | Role |
|---|---|---|
| `pkg/templateengine/` | 203 | parse → sanitize → static-analyze → render. Pure; no DB, storage, or network. |
| `internal/core/domain/documenttemplate/` | 102 | Kind registry + context types + variable catalogs. |
| `.../documenttemplate/starters/` | 77 | `go:embed` built-ins for all 8 kinds (21 asset files). |
| `internal/infrastructure/templating/assetinliner/` | 38 | Resolves document images to data: URIs via `httpsafe`. |
| `shared/{money,timeutils,stringutils,imageutils}` | 206 | Extractions with byte-parity tests against the originals. |

**Why context types live in the domain, not in the services that build them:** a template
context is a published contract — it is the variable list an administrator writes against.
`registry_context_test.go` asserts the catalog and the struct match **in both directions** by
reflection, so a field cannot be added without being documented, nor documented without
existing. That test is the highest-value correctness lever in the design; do not weaken it.

**8 registered kinds:** `invoice.pdf`, `invoice.email`, `detention.notice.email`,
`detention.notice.pdf`, `report.pdf`, `report.delivery.email`,
`driverportal.invitation.email`, `agent.request_missing_docs.email`.

Live-verified against the sidecar: the new invoice renders with every collectable field
present, the detention notice carries the figures a charge rests on, the report is landscape,
and `<thead>` repeats across a 4-page invoice.

---

## 4. Phase 3 — domain, repositories, service, complete

### 4.1 Done and verified

**Migration** `internal/infrastructure/postgres/migrations/20260902000000_document_templates.tx.{up,down}.sql`

Applied against the live dev database and tested **up → down → up**. The down migration
recreates the two dormant tables and five legacy enums verbatim — without that,
`migrator.Reset` and a group rollback leave the schema wrong and re-running the forward
migration fails on its `DROP`s.

Confirmed before dropping: in a fully-migrated database (258 migrations, at `20260901010000`),
the dormant `document_templates` and `generated_documents` from
`20251125000001_add_document_templates` hold **0 rows** and are referenced by zero Go, TS, or
SQL. That migration is now superseded.

Tables: `document_templates`, `document_template_versions`,
`document_template_assignments`, `generated_documents`; plus `body_html`,
`template_version_id`, `pdf_document_id` added to `detention_notices`.

Notable schema choices:
- `kind` is `varchar`, **not** a Postgres enum, so adding a kind never needs a migration. The
  Go registry is the single source of truth. Deliberate divergence from house habit; commented.
- Partial unique indexes enforce one draft and one active version per template, and one org
  default per kind — in the database, not just the service.
- `generated_documents.template_version_id` is `ON DELETE RESTRICT`. That is what makes "what
  exactly did we send that customer in March" answerable after a template has been edited,
  which is precisely when an invoice or detention dispute asks.
- CHECK constraints refuse a version with no content in any channel and a `Completed` row with
  no file path. The domain validates the same rules first so callers get a field error rather
  than an opaque constraint violation.

**Domain entities** — `template.go`, `version.go`, `assignment.go`, `generated.go`.
`task generate-columns` run (`pkg/buncolgen/documenttemplate_gen.go`, `fieldmap_gen.go`).
Registered in `pkg/domainregistry/domainregistry.go`.

Every entity diffed against the real schema: **14 / 31 / 10 / 26 columns, exact match both
directions**, no drift.

**Repository ports** — `internal/core/ports/repositories/documenttemplate.go`, compiling.
Four interfaces: `DocumentTemplateRepository`, `DocumentTemplateVersionRepository`,
`DocumentTemplateAssignmentRepository`, `GeneratedDocumentRepository`.

### 4.2 Rest of Phase 3 — done

**Repository implementations** — `internal/infrastructure/postgres/repositories/documenttemplaterepository/`:
`template.go`, `version.go`, `assignment.go`, `generated.go`, `resolve.go`.

`resolve.go` is one round trip over three tiers, built from `buncolgen` fragments
(`resolveTemplateJoin` / `resolveAssignmentJoin`). Two deviations from the SQL sketched above,
both deliberate:

- It reads **`document_template_versions` as the base table** and joins the template, rather
  than the reverse. The join carries `dtpl.active_version_id = dtv.id`, so only a template's own
  live version can resolve — matching on `template_id` alone would let a draft render.
- It **projects columns explicitly into a flat scan struct** instead of scanning the model. Bun
  rejects a column the model has no field for (`model_table_struct.go:300`), and the statement
  returns one such column: whether an assignment matched. Deriving the tier in Go instead was
  rejected because a template that is both the org default *and* assigned would be
  indistinguishable, and the source is what a generated document records.

Verified against the live schema with `EXPLAIN`: the plan uses
`uq_document_template_versions_one_active`, `idx_document_templates_kind`, and
`idx_document_template_assignments_resolution`.

`Publish` archives the outgoing version, activates the incoming one, and repoints
`document_templates.active_version_id` in one transaction — **archive first**, because the
partial unique index allows one Active row per template and is checked per statement.

**Service** — `internal/core/services/documenttemplateservice/`:
`service.go` (resolution, compilation, content hashing), `render.go`, `publish.go` (the gate),
`admin.go` (CRUD, rollback, assignment, audit), `validator.go`.

Port at `internal/core/ports/services/documenttemplate.go`. It grew two members beyond the
sketch, both of which a caller genuinely needs:

- `RenderMessageRequest.FallbackToBuiltIn` — the notification-versus-invoice failure policy is
  the caller's decision, not a rule baked into the service.
- `AssetInliner` — a port rather than a direct import, because resolving an
  organization-controlled URL is an egress decision the core must not own.
  `assetinliner/port.go` adapts the existing `Inliner`.

`PreviewVersion` accepts unsaved editor content (`Content`), a stored `VersionID`, or neither
(the built-in), in that precedence — the editor previews what is on screen, which by definition
has not been saved.

**Permissions** — `ResourceDocumentTemplate` in `resource_gen.go`, registered under
`registerAdministrationResources()` with `standardOpsWithDelete` plus `OpActivate` (labelled
"Publish"), `OpArchive`, `OpAssign`, `OpUnassign`. Two routes in `routeregistry.go`
(`/admin/document-templates` exact, plus a prefix route for the detail surfaces).
`DocumentTemplate: "document_template"` mirrored into `packages/shared/src/types/permission.ts`.

**Seed** — `base/10_document_template_starters.go`, repeatable, depends on `SeedAdminAccount`.
Run against the dev database: 16 drafts across 2 organizations, then re-run — still 16, with
**0 org defaults and 0 active versions**. That is the safety property: a seeded template is a
draft that is never the default, so no run of this seed can change a document a customer
receives.

**fx wiring** — `modulesinfra.TemplatingModule` provides `documenttemplate.NewRegistry`,
`templateengine.New`, and `assetinliner` behind the `services.AssetInliner` port, all singletons.
Repositories, validator, and service registered in the existing modules.
`fx.ValidateApp` passes for **both** the API and the worker graphs.

**Mocks** — five hand-written in `internal/testutil/mocks/`, plus `mock_AssetInliner.go`.
Generated from a one-off script in the session scratchpad rather than mockery, in the same
mockery-testify shape as `mock_PDFRenderer.go`.

**Tests**

- `documenttemplateservice/service_test.go` — the highest-value one is
  `TestBuiltInStartersPassTheirOwnPublishGate`: **all 8 starters** pass the same gate an
  organization's own template must pass. A shipped default that could not be published would be
  content no administrator is allowed to author.
- `documenttemplaterepository/sqlshape_test.go` — asserts the join fragments pin tenant, kind,
  and active version. The risk in generated SQL is not a typo, it is a join that silently widens
  scope.
- `documenttemplaterepository/repository_integration_test.go` — all three resolution tiers
  against real rows including unassign-returns-to-default; both partial unique indexes; the
  `ON DELETE RESTRICT` on `generated_documents`; the `Completed`-needs-a-file CHECK; and the
  down/up migration round trip through the circular `active_version_id` FK.

**One real bug the integration test caught:** the assignment upsert used `Version.Inc(1)`, which
emits a bare `version = version + 1`. Inside `ON CONFLICT DO UPDATE` both the target row and
`excluded` expose `version`, so Postgres refuses it as ambiguous. Now
`Version.SetExpr(Version.Qualified() + " + 1")`. Worth remembering for any other upsert that
bumps a counter — `Inc` is unsafe in a `DO UPDATE` clause.

**Two entities gained a `GetPostgresSearchConfig`** they were missing: `GeneratedDocument` (the
migration builds a search vector the entity never declared, so a search would have scanned) and
`DocumentTemplateAssignment` (needed by `querybuilder.ApplyFilters`).

---

## 5. Phase 4 — GraphQL and preview route, complete

`internal/api/graphql/schema/document_template.graphqls` plus resolvers,
`documenttemplatemapping.go`, and one REST route. Enum and model bindings are in
`gqlgen.yml`; `documenttemplate.{DocumentTemplate,DocumentTemplateVersion,
DocumentTemplateAssignment}` bind directly, so the domain entities *are* the
GraphQL types and there is no parallel model to keep in step.

**Queries:** `documentTemplates`, `documentTemplate`, `documentTemplateVersion`,
`documentTemplateKinds`, `documentTemplateAssignments`,
`customerDocumentTemplateAssignments`, `renderDocumentTemplatePreview`,
`renderMessageTemplatePreview`.

**Mutations:** `create/update/deleteDocumentTemplate`,
`create/update/publish/archive/deleteDocumentTemplateVersion`,
`rollbackDocumentTemplate`, `assign/unassignDocumentTemplate`,
`sendTestMessageTemplate`.

Every resolver opens with `r.requirePermission(...)`. Publish and rollback take
`OpActivate` rather than `OpUpdate`, because editing a draft changes nothing a
customer sees and activating one changes every document the organization sends
from that moment.

### What the plan got wrong here

- **`projection.yml` gates are not a filterability mechanism.** Gates drive
  optional *relation loading*; what is filterable and sortable comes from
  `querybuilder`'s reflected `FieldConfiguration`, which marks every column both.
  The plan's "gate template bodies so they are not filterable or sortable" cannot
  be expressed there. It also turned out to be unnecessary: bodies live on
  `DocumentTemplateVersion`, whose only list endpoint takes a template id and no
  filter, so there is no reachable surface to filter or sort them from. If a
  version connection is ever added, this becomes real work — the mechanism would
  be a custom `FieldConfiguration` via `ApplyFiltersWithConfig`, not projection.yml.
- What projection.yml *did* need: `starterDrifted` as a `virtuals` entry (it is
  computed, not a column), and the four synthetic DTOs — `DocumentTemplateKind`,
  `DocumentTemplateVariable`, `DocumentTemplateDiagnostic`,
  `DocumentTemplatePreview` — registered in `nonProjectionObjects` in
  `projection/gen/generator.go`, since the generator demands a table for every
  object type it does not know to skip.
- `VariableDefinition` has no `Example` field. The plan's editor mock implied
  one; the schema exposes `path`, `type`, `description`, `required`, and nested
  `fields` instead.

### The preview route

`GET /api/v1/document-templates/versions/:versionID/preview.pdf` —
`internal/api/handlers/documenttemplatehandler/`. It renders the stored version
against the kind's **sample** context, never a real customer's: a preview is an
authoring tool, and the sample exercises every declared variable anyway.

`ErrPDFRendererUnavailable` becomes a business error, not a 500 — a deployment
without the sidecar is a condition, not a fault.

**GraphQL never returns PDF bytes.** `DocumentTemplatePreview.pdfUrl` points at
this route, and only for a saved version — unsaved editor content has no id to
address, so the client saves the draft before asking for a print. The GraphQL
preview also ignores `includePdf` deliberately: printing inside a
keystroke-debounced query would burn a conversion nobody reads.

### The security contract, and how it is held

Template markup is authored by an organization. If it were ever executed with
this application's origin it could read app-origin cookies and `localStorage`,
and the editor would become a privilege-escalation path for anyone who can edit
a template. So no endpoint serves template output as a document: GraphQL returns
HTML as a `String` for the client to place in a sandboxed iframe, and the REST
route returns `application/pdf` with `X-Content-Type-Options: nosniff`.

`documenttemplatehandler/security_test.go` holds this. Four of its five tests are
about this one route; the fifth, `TestNoAPIEndpointServesTemplateOutputAsHTML`,
scans every file under `internal/api/handlers` and `internal/api/graphql` for a
`text/html` content type, because a *future* route reintroducing the problem is
the realistic failure and no test scoped to this package would catch it. Two
first-party HTML surfaces are allowlisted by path with a reason: the Swagger UI
shell and the GraphQL playground shell.

`sendTestMessageTemplate` prefixes the subject with `[TEST]` and refuses to send
when the template has blocking diagnostics.

Both REST routes are classified under `FeatureDocumentManagement` in
`platformcatalog/provider_routes.go`, with the matching entry in
`registry_test.go`'s mirrored list — the catalog test fails without both halves.
---

## 6. Phase 5 — invoice cutover, complete

`gofpdf` is gone. `invoice_pdf.go` went from **1,342 lines to 462**: every
`draw*` helper, the logo loader, the 18 millimetre constants, `wrappedPDFLines`,
`fittedImageSize`, `invoicePDFLogo`, and `invoicePDFBox` are deleted, along with
`proHeaderValue` and `invoicePDFMetadataDate`, which only the deleted drawing
code called. The ~28 pure data helpers survive untouched, and
`buildInvoicePDFData` — the function that encodes what actually appears on a
bill — is now the only builder. `Service.storage` went with them: the asset
inliner owns object access now.

`github.com/jung-kurt/gofpdf` is out of `go.mod`. One textual reference remains:
a historical note in `documenttemplate/enums.go`.

### The context builder

`invoiceservice/templatecontext.go`. `ContextBuilder` implements
`services.ContextBuilder` for `invoice.pdf` and is the first entry in the fx
group.

**It depends on repositories, not on `*Service`, and that is load-bearing.** The
invoice service depends on `DocumentTemplateResolver` to render; the template
service depends on the builder group. Routing the builder through `*Service`
closes that loop and `fx.ValidateApp` refuses to start — confirmed by hitting it.
`resolveDeliveryProfile` and `resolveDeliveryOrganization` were therefore split
into package-level `…With(repos, …)` functions that both the service method and
the builder call, so there is still exactly one implementation.

The fx registration provides the concrete `*ContextBuilder` **and** a separate
group entry, because `fx.As` consumes the concrete type and the invoice service
needs it directly on the send path where the invoice and profile are already
loaded (`BuildFrom`).

### The logo, and the SSRF fix landing

`AssetInliner` gained `ResolveImageDataURI`. A logo arrives as a context *field*,
not as markup — by the time `InlineAssets` sees rendered output the value is
already interpolated — so a builder needs a way to resolve one reference.

This matters beyond convenience: `LogoDataURI` is typed `template.URL`, which
html/template does **not** escape. Interpolating a raw organization-supplied URL
there would hand an organization the unescaped-URL slot. Going through the
inliner means the field only ever carries base64 of bytes that were decoded and
validated, and the fetch uses the SSRF-safe client that re-checks every resolved
address at dial time — which is the `invoice_pdf.go:260` bug finally closed
rather than merely relocated.

A logo that cannot be resolved yields an empty string, not an error. The
template's `{{ if }}` drops the image; the invoice still goes out.

### Retry policy

The plan named `ErrTypeTemplateInvalid`; no such thing exists — the convention
here is `temporaltype.ErrorType*`. Added `ErrorTypeTemplateInvalid` plus
`NewTemplateInvalidError`, and gave the PDF workflow its own
`generateInvoicePDFRetryPolicy` instead of borrowing the auto-post one.

`classifyInvoiceRenderError` splits the two failure modes, which is the whole
point: a renderer outage is transient and **should** retry, while a template that
does not compile fails identically five times and only delays the failure
reaching the administrator who has to fix it.

### What happened to the tests

The plan said the ~20 `buildInvoicePDFData*` assertions keep their value and
stay. They did. Eight tests were removed and replaced, all of them testing
something that no longer exists:

| Removed | Why, and what covers it now |
|---|---|
| `…EmbedsStoredPNGLogo`, `…JPEGLogo`, `…WebPLogo`, `…FallsBackWhenLogoUnsupported` | The logo pipeline moved wholesale to `assetinliner`, which has its own tests including WebP transcoding. Re-testing it here would duplicate that. |
| `TestFittedImageSizePreservesAspectRatioWithinBounds` | Renderer geometry. The inliner's `applyIntrinsicDimensions` now bounds images. |
| `TestInvoicePreviewWithFooterFitsSinglePage` | Page fitting is Gotenberg's, and the live starter tests already assert pagination. |
| `TestInvoicePreviewForEntityReturnsPDFResult`, `TestRenderPreviewLoadsBillingControl` | Rewritten against the context, which is stronger — see below. |

`templatecontext_test.go` replaces them with six tests, the important ones being
**BillingControl continuity in both directions**: `DefaultInvoiceTerms`,
`DefaultInvoiceFooter`, `ShowDueDateOnInvoice`, and `ShowBalanceDueOnInvoice`
must still populate `.InvoiceTerms`, `.InvoiceFooter`, `.DueDate`, and
`.BalanceDue` when on, and must still suppress them when off — exactly what the
deleted `invoicePDFShowDueDate` / `ShowBalanceDue` helpers encoded.

`livepdf_integration_test.go` (build tag `integration`, skips without
`RENDERER_GOTENBERG_URL`) prints a real invoice through the real sidecar and
extracts the text a reader sees. It asserts the invoice number, the accessorial
line, `Total USD 2950.00`, `Balance Due USD 2950.00`, and the authored footer are
on the page. Bytes that merely parse as a PDF were never the claim.

Run it with:

```
docker compose -f docker-compose-local.yml up -d gotenberg
RENDERER_GOTENBERG_URL=http://localhost:3009 go test -tags=integration \
  -run TestLive ./internal/core/services/invoiceservice/
```

---

## 7. Phase 6 — detention notice, complete

`detentionservice/templatecontext.go` builds `DetentionNoticeContext` and is
registered for **both** detention kinds: `NewPDFContextBuilder` wraps the same
builder under `detention.notice.pdf`, because a notice PDF is the email's
content on paper and a second builder would be a second place for the numbers in
a dispute to disagree.

### BuildNotice

Now `(s *Service) BuildNotice(ctx, *BuildNoticeParams) (NoticeContent, error)`.
The plan hoped to keep the old pure-function signature; that is not possible once
rendering needs a resolver, a context, and an error path. `ScheduleNotice` and
the snapshot columns needed no rework, which was the actual point.

It sets `FallbackToBuiltIn: true` — a notice is the evidence a detention charge
rests on, so a template mistake must not cost the charge.

`RenderBuiltInNotice(ctx, params)` renders the shipped default with no database,
no resolver, and no organization. The dev seed uses it so seeded notices read
exactly like what a fresh organization sends, and the notice tests use it to
prove the starter reproduces the prose the deleted `strings.Builder` produced.
`noticeSubject`, `noticeOpening`, and `noticeClosing` are deleted — their words
now live only in the starter.

### The XSS is closed, and there was a second one

The plan named the `<pre>` wrap at `SendNotice`. There were two: `BuildNotice`
also wrapped its own text. Both are gone.

- `detention_notices` gained `body_html`, `template_version_id`, and
  `pdf_document_id` on the **entity** — the Phase 3 migration added the columns
  but the struct never got the fields, so nothing could have used them.
- `BuildNotice` snapshots the rendered HTML into `body_html`, and `SendNotice`
  mails that snapshot. **The send path now interpolates nothing.**
- A notice queued before the column existed has no snapshot, so `noticeHTML`
  reconstructs one with `html.EscapeString` rather than reproducing the old bug.

`notice_escaping_test.go` pins both: an operator-entered facility name containing
`<script>` and a shipment reference containing `<img onerror=…>` survive as
escaped, legible text and never as elements.

### One deliberate wording change

The old renderer emitted `Detention charge: 225.00 USD` while the invoice said
`USD 225.00`. The starter uses the shared `moneyString` helper, so both now read
`USD 225.00`. A notice and the charge it becomes finally quote money identically
— which is the stated purpose of the snapshot the notice cites. The test asserts
the new format and says why.

### generated_documents is now written

`generatedRepo` was wired into the template service in Phase 3 and never used,
which meant the table justifying `ON DELETE RESTRICT` was empty.
`RecordGeneratedDocument` on the service fills it, and
`PrepareInvoicePDFUploadActivity` calls it after the invoice bytes are uploaded.

It is deliberately best-effort at the call site: the invoice has already been
rendered and stored, and failing the workflow over a bookkeeping row would turn a
missing audit entry into a missing invoice.

Writing it surfaced a real ordering bug: `GeneratedDocument.Validate` requires a
delivery method, but only `BeforeAppendModel` defaulted it — and that hook runs
at insert, *after* the service validates. Every record would have been rejected.
The service now sets `DeliveryMethodNone` explicitly. **Worth checking for the
same shape elsewhere: any entity whose `Validate` requires a field its
`BeforeAppendModel` defaults is unvalidatable before insert.**

### `DetentionPolicy.AttachNoticePDF` — done

Migration `20260903000000_detention_policy_attach_notice_pdf`, verified up → down
→ up against the dev database. It adds the policy flag (default FALSE, so every
existing policy keeps sending exactly what it sent yesterday) **and backfills a
document type per organization**, because seeding only ever runs for new ones and
the first policy to turn the flag on would otherwise fail to file its notice. The
derived id (`md5(org || 'DETNOTICE')`) makes the backfill idempotent, and the down
step deletes only the types nothing was filed under.

**The plan said document type `DETENTION_NOTICE`. That cannot exist:**
`document_types.code` is `varchar(10)`. The code is `DETNOTICE`, and it lives in
`documenttype.CodeDetentionNotice` because three places have to agree on it — the
seed, the migration, and the send path.

How the pieces fit, and why:

- **The PDF is rendered where the email body is** (`BuildNotice`), from the same
  context object, and carried forward in memory through `ScheduleNotice` →
  `SendNotice`. Rendering it at send time from a fresh context would re-read the
  occurrence, and a notice whose attachment quotes different figures than its body
  is worse than no attachment. `ScheduleNotice` therefore returns a
  `*ScheduledNotice` (notice + PDF) and `SendNotice` takes `*SendNoticeParams`.
- **A PDF render failure drops the attachment, never the notice.** The email is
  the evidence a detention charge rests on.
- **Filing happens after the send**, best-effort, through the same upload session
  → finalize-workflow path every other stored document uses. The send path streams
  the bytes into the session and starts `StoreNoticePDFWorkflow`, which runs the
  finalize child workflow and then `AttachNoticePDFDocumentActivity`. The bytes
  never travel through a Temporal payload — only the session id and the render
  provenance with `PDF`/`HTML` stripped, which one of the new tests pins.
- **Without a Temporal worker** the same code path calls
  `DocumentUploadService.Complete`, which finalizes inline and returns the document
  id. Both routes end in `AttachNoticePDFDocument`, so the notice row and the audit
  row have exactly one writer. `Complete` was added to the port for this; the
  hand-written mock was extended to match.
- `AttachPDFDocument` on the notice repository updates **only** `pdf_document_id`
  and `updated_at`. A provider webhook may be recording delivery on the same row
  while this runs, and a read-modify-write would have the two racing for the same
  optimistic version.

**New port: `services.DetentionNoticeService`.** The detention service now starts
a workflow, so it imports `detentionjobs`; the activities therefore cannot import
the service back. They depend on this two-method port instead — the same shape
`billingjobs` already uses for `services.InvoiceService`. `NoticeSweepResult` and
`AttachNoticePDFDocumentParams` moved to `ports/services/detention.go` with it, and
`api.ServiceModule` gained the binding (both graphs use that module).

The sweep passes the policy it already loaded into `SendOccurrenceNoticeParams`,
so deciding whether to attach costs no second read per occurrence.

Client: `attachNoticePdf` on the `DetentionPolicy` type and input, the zod schema,
the panel defaults, the table operation, and a `SwitchField` next to
"Send the notice automatically". Note gqlgen renders the field as `AttachNoticePDF`
in `gqlmodel` (initialism), not `AttachNoticePdf` — the schema name is
`attachNoticePdf` either way.

## 8. Remaining work

### 8.2 Phase 7 — emails and notifications

Every hardcoded string **moves into** a built-in starter; nothing is duplicated. `Resolve`
returning no row yields `Source: BuiltIn` from the embedded asset, so there is exactly one
render path and no "if no template then hardcoded string" branch anywhere.

**All six email senders are done.** What each one turned out to need:

| Site | Kind | Outcome |
|---|---|---|
| ~~`invoiceservice/delivery.go`~~ | `invoice.email` | **Done.** Precedence preserved exactly: draft snapshot → `CustomerEmailProfile.Subject`/`Comment` (both still via the ad-hoc `renderInvoiceTemplate`) → template → built-in. See the three traps below. |
| ~~`detentionservice/notice.go`~~ | `detention.notice.email` | **Done in Phase 6.** |
| ~~`driverportalservice/invitation.go`~~ | `driverportal.invitation.email` | **Done.** `invitationEmailHTML` and the 4-char `htmlEscape` deleted; the unescaped `inviteURL` is closed. "Expires in 7 days" now reads the invitation row, so an extended invitation quotes its real deadline, formatted in the carrier's zone. |
| ~~`reportjobs/delivery.go` + `digest.go`~~ | `report.delivery.email` | **Done.** Both `strings.Builder` bodies and `renderDigestHTML`/`renderDigestText`/`digestAlign`/`digestToneStyle`/`digestToneColors` are deleted. The digest is context now; the tone travels as a *name*, so the report definition decides what an exception is and the template decides how it looks. |
| ~~`agenttoolservice/request_missing_docs.go`~~ | `agent.request_missing_docs.email` | **Done, inverted as planned.** The tool schema also grew optional `customerName`, `shipmentProNumber`, and `requestedDocuments` — a variable the schema does not declare is a variable no agent will ever fill, which a test now pins. |
| ~30 notification sites, 18 files | `notification.<EventType>` | Not yet registered — see below. |

### What the invoice email turned up

Three things the plan of record did not mention, all load-bearing:

- **`applySendSnapshot` would have frozen the render.** It copies the plan's
  subject and body into `EmailSubjectSnapshot`/`EmailBodySnapshot` — which are the
  *first* tier `resolveSubject` reads. Left alone, the first send pins that render
  forever: a template edit never reaches a re-sent invoice, and the frozen copy
  comes back through the ad-hoc `{number}` engine rather than html/template,
  losing the layout. Template-sourced wording is therefore not frozen; a draft or
  profile comment still is, because that is what makes a re-send reproduce what
  the customer received. If you touch this, `TestSendSnapshotDoesNotFreezeTemplateWording`
  is the guard.
- **Only the template tier produces HTML.** `InvoiceSendPlan` gained `BodyHTML`
  and `FromTemplate` so the send path knows which body it holds; free text still
  goes through `bodyHTML()`'s escape-and-`<br>` wrap.
- **A split invoice re-renders per message** so `PartLabel` is a real value. The
  profile is re-read for that (`splitSendProfile`) rather than carried on the plan,
  because the plan crosses a workflow boundary as JSON.

Also extracted `services.ResolveLogoDataURI`: one place in the system types an
inliner result as `template.URL`, instead of three copies of the same `//nolint:gosec`
reasoning. Invoice, detention, and the agent email all go through it.

### Newly found gap: `report.pdf` never actually cut over

`report.pdf` is a registered kind with a starter, a sample context, and a variable
catalog — and **nothing renders through it.**
`internal/infrastructure/reporting/render/pdf.go` still holds its own
`html/template` literal and its own `pdfTemplateData`. Phase 1 replaced `chromedp`
with the Gotenberg port there, which is what made it look done; the *layout* was
never exposed. An administrator editing "Report Export" in the Phase 8 UI would
see no effect whatsoever.

It was left alone deliberately rather than half-done: `services.ReportRunMeta`
carries no tenant, so the renderer cannot resolve an organization's template
without threading `OrgID`/`BuID` through every caller that builds a render
request (the reporting service, `reportjobs`, the dashboard export). That is a
real piece of work, not a one-liner, and it is the same shape as the invoice
cutover: add the tenant to the meta, render through the resolver with the existing
`ReportContext`, delete `pdfTemplate` and `pdfTemplateData`, and keep
`reportPDFMargins`.

Notification sites: `assignmentservice/service.go` 120/148 · `bankreceiptservice/service.go` 368
· `driverportalservice/actions.go` 80/315 · `driversettlementservice/{expense.go 135/363,
posting.go 65, lifecycle.go 233, transfer.go 65, dispute.go 327}` · `ediservice/alerts.go` 50 ·
`fuelsurchargeservice/eia.go` 303 · `invoiceservice/service.go` 444 ·
`shipmentcommentservice/service.go` 660/720 · `shipmentmoveservice/stopactuals.go` 158 ·
`shipmentservice/billing_readiness.go` 969 · `telematicsservice/alerts.go` 154/192 ·
`compliancejobs/activities.go` 219/259 · `recurringshipmentjobs/activities.go` 242 ·
`reportjobs/{activities.go 681, delivery.go 315/443, dispatch.go 228}`.

Notification kind keys are `"notification." + <the exact EventType string already at the call
site>`, so the registry and the `notifications.event_type` column stay in lockstep. Each needs
a const in `kind_gen.go`, a `Register` call, starter assets, and a context struct with its own
catalog. **Settled with the owner: ~6 context families** (driver, dispatch/ops, report,
comment, settlement, recurring) sharing a context shape via helper builders like
`context_shared.go` already does, rather than 30 bespoke structs for one-line
title/message pairs. All ~30 kinds still register separately, so the registry and the
`notifications.event_type` column stay in lockstep; only the context types and their
catalogs are shared, and the reflection catalog test covers 6 types instead of 30.
Draw family boundaries where the *data* differs, not where the event names do — a
family whose catalog reads "entity name" has been drawn too wide.

### Notifications: survey findings before you write any of it

Read these first; they change the shape of the work from what the plan describes.

- **Most driver notifications share one choke point.** Every `dash.*` event —
  load assigned, load unassigned, PTO reviewed, settlement posted, settlement
  paid, expense reviewed, pay held, dispute resolved, credential expiring —
  goes through `drivernotificationservice.NotifyWithCorrelation`. Rendering there
  covers ~10 of the ~30 sites with one change, and it is the only place that
  already knows the worker. Do that before touching individual services.
- **Some event types are not literals.** Several sites pass a variable or a
  domain enum (`eventType`, `alert.EventType`, `entry.EventType`,
  `sourceEvent.String()`). A kind key of `"notification." + EventType` needs
  every reachable value registered, so enumerate the enum members rather than
  assuming a literal is at the call site. Truly open sets — a telematics
  provider's webhook `event_type` — do not create titled notifications, so they
  are not in scope; verify that before relying on it.
- **`kind_gen.go` has no generator.** The `_gen` suffix marks it as the flat
  declaration list that `registry_coverage_test.go` parses by AST: every constant
  must have a matching `Register` call or the build fails. That makes a partial
  notification commit impossible, which is protective — land a whole family at a
  time.
- **The reflection catalog test cuts both ways with shared contexts.**
  `registry_context_test.go` asserts catalog ↔ struct in both directions per kind,
  so a family sharing one struct needs its catalog built by one helper the whole
  family calls, not copied per kind.
- Failure policy for the whole phase is settled: notifications render with
  `FallbackToBuiltIn: true`. A lost notification is worse than an unstyled one.

`notificationservice.Create` (line 48) is **not** changed — templates resolve at the call site
where the domain data lives, keeping `Create` a dumb writer.

Failure policy: a render failure on a **notification** falls back to the built-in and logs (a
lost notification is worse than an unstyled one); on an **invoice email** it is a hard error
surfaced to the sender.

**Two ad-hoc engines survive by design, documented not migrated:**
`invoiceservice/delivery.go:1562 renderInvoiceTemplate` must keep serving
`EmailSubjectSnapshot`/`EmailBodySnapshot` and `CustomerEmailProfile.Subject/Comment`;
`tablechangealertservice.RenderTemplate` must keep serving `{{new.field}}` over arbitrary CDC
payloads. Both are free-text fields over unbounded, undescribable variable spaces — the
opposite of a registry-backed catalog.

### 8.3 Phase 8 — admin UI

**No new editor, viewer, or diff library needed.** Verified present in
`client/apps/web/package.json`: `@uiw/react-codemirror` 4.25.11 +
`@codemirror/{view,state,language,autocomplete,lint}`, `react-diff-viewer-continued` 4.2.2,
`react-pdf` 10.4.1. **Two new deps only:** `@codemirror/lang-html`, `@codemirror/lang-css`.

Reuse: `components/formula-editor/editor-theme.ts` (brand-matched CM theme),
`formula-editor/expr-language.ts` (the `createCompletions` + `language.data.of({autocomplete})`
pattern), `formula-editor/formula-reference-panel.tsx` (grouped searchable variable picker),
`components/fields/json-editor-field.tsx` (the RHF+CM field wrapper to clone),
`components/elements/pdf-viewer.tsx`, `packages/shared/.../ui/resizable.tsx`,
`hooks/use-debounce.ts`, `ui/segmented-control.tsx`,
`animate-ui/primitives/effects/highlight.tsx`.

**Closest structural analogue is `routes/admin/document-parsing-rules/`, not
`detention-policy`** — it is already a versioned resource with list → detail → tabs → version
detail and per-tab `lazy()`. Copy its shape, including the badge map (`Draft: "warning"`,
`Active: "active"`, `Archived: "secondary"`); swap REST for GraphQL.

Key calls:
- **The list surface is a kind catalog, not a `DataTable`.** Kinds are registry-driven and
  fixed-cardinality; most have zero rows until an admin customizes one, so a table renders
  empty for a feature that is 90% "system default in effect". Each card shows
  `System default` / `Customized · v3 active` / `Draft pending`. Model on
  `admin/home-layouts/page.tsx`. Customer **assignments** are unbounded and do get a real
  `DataTable` on a second tab.
- **Preview: two tiers, never on keystroke.** A content hash feeds two `useDebounce` calls —
  600ms drives HTML preview, 1800ms gates the PDF tier — plus an explicit Render PDF button and
  `Cmd/Ctrl+Enter`. `staleTime: Infinity` with the hash in the query key makes undo an instant
  cache hit. `retry: false`. Stale state keeps the previous render, dimmed + desaturated with a
  2px indeterminate bar — never a spinner over blank. Errors render inline in the preview pane
  with the failing line clickable to scroll+select; never a toast.
- Preview is a GraphQL **query** (mutates nothing), which satisfies the GraphQL-only-writes
  rule cleanly.
- **Email preview iframe must be `sandbox=""`** — no `allow-scripts`, no `allow-same-origin`.
  With `allow-same-origin` a template could read app-origin cookies and `localStorage`, turning
  the editor into a privilege-escalation path.
- Rollback creates a *new* draft seeded from the target and publishes it rather than mutating
  history; the dialog must say so.
- Send-test needs a **new GraphQL mutation** — the existing REST `emailhandler.testSend` is
  bound to an email-profile id, not a template-version id, so it cannot render an unpublished
  draft.
- Customer assignment on both surfaces. The customer-page tab must **not** participate in the
  customer form's RHF state or `FormSaveDock` reports phantom unsaved changes — use
  immediate-save controlled autocompletes per the `Controlled*AutocompleteField` precedent.
- **Replace the two existing disabled nav placeholders**, don't append:
  `navigation.config.ts:1298` ("Document Templates") and `:1204` ("Notification Types" →
  "Message Templates").
- **One** query-key factory in `lib/queries/template.ts`, registered in the
  **`operationsQueries`** merge group. A second factory risks the known TS2589 depth failure.
- Also fix the pdfjs `workerSrc` CDN issue from §2.9.

Project UI rules that apply: every field needs a `description` stating the real-world
consequence; every `<Select>` needs `items`/`options` from `lib/choices.ts`; `useWatch` never
`watch()`; no colored left-border accents; Linear/Vercel-level polish with real motion behind a
`prefers-reduced-motion` guard.

---

## 9. Verification

**Generator order is not optional:**

```
task generate-columns
go generate ./internal/api/graphql/projection/...
task gqlgen
task generate-seeds
pnpm --filter @trenova/graphql codegen
# then REBUILD the Go binary — the persisted-op manifest is go:embed'd, so new
# GraphQL operations 404 until restart. This is the #1 way the integration looks
# broken while being correct.
```

**Gates:**

```bash
# Go
cd services/tms
go build ./... && go test ./...
task lint            # full-repo lint has a large pre-existing backlog; lint your own paths
task gqlgen-check
task test-integration

# Shared
cd shared && go test ./...

# Client
cd client
pnpm --filter @trenova/graphql codegen:check
pnpm lint && pnpm test
npx tsc -b --force   # `pnpm tsc --noEmit` is false-green here
```

**Integration suite** (needs the shared test Postgres, container
`trenova-integration-postgres` on port 55432 — `task test-db-up` starts it):

```bash
TRENOVA_TEST_POSTGRES_DSN="postgres://postgres:postgres@localhost:55432/trenova_test?sslmode=disable" \
APP_ENV=test go test -tags=integration -count=1 \
  ./internal/infrastructure/postgres/repositories/documenttemplaterepository/
```

**Live renderer tests** (they skip without the env var):

```bash
docker compose -f docker-compose-local.yml up -d gotenberg
RENDERER_GOTENBERG_URL=http://localhost:3009 go test -tags integration -run TestLive \
  ./internal/infrastructure/pdfrender/... \
  ./internal/core/domain/documenttemplate/starters/ \
  ./internal/core/services/invoiceservice/
```

The invoice one prints a real invoice through the sidecar and extracts the text a
reader sees. If you change the invoice starter or the context builder, that is
the test that tells you whether a customer can still read what they owe.

**Two gates fail by construction while this work is uncommitted:**
`task gqlgen-check` and `pnpm --filter @trenova/graphql codegen:check` both
regenerate and then diff against git. They report "out of date" until the work is
committed. Both regenerate cleanly — check the diff is only your own changes
rather than assuming the failure is benign.

**Manual verification constraints:** the sandbox kills live servers (exit 144), so verify via
build, unit tests, FX boot (`fx.ValidateApp` — see `internal/bootstrap/app_test.go`), and
`psql`, not `curl`. The Temporal worker is a separate process that `air` does **not** reload;
restart it after touching activities.

**Database:** local Postgres runs as container `db`, database `trenova_go_db`.
`docker exec db psql -U postgres -d trenova_go_db -tAc "..."`.

---

## 10. Current tree state

Phases 1–6 are committed on `master`, **not pushed**. The working tree is clean of
template work.

Green as of handoff: `go build ./...`, `go test -count=1 ./...`, `fx.ValidateApp`
for both the API and worker graphs, `golangci-lint` on the new paths, the live
Gotenberg renders, the migration up → down → up, `npx tsc -b --force`, `pnpm lint`
(0 errors), and the 499 client tests.

Two unrelated uncommitted changes were left alone deliberately:
`client/apps/web/src/hooks/shipment-comments/use-comment-mutations.ts` (a
memoization fix from another session — correct, worth committing) and
`docs/accounting-overhaul-fable-prompt.md` (a markdown formatter reflowed it and
turned one prose paragraph into a nested list item; worth reverting rather than
committing).

### Read this before you commit

**1. Other sessions silently reverted shared files, repeatedly.** The fx wiring,
the gqlgen bindings, the projection manifest, and the seed registration were each
lost at least once while this was being written. Committing Phases 3–6 closed most
of that exposure, but the table still applies to anything you add. The build stays
green when fx wiring vanishes — only `fx.ValidateApp` catches it — and a missing seed
registration shows up nowhere except the integration suite. Check every row:

| File | What must be present |
|---|---|
| `internal/bootstrap/app.go` | `modulesinfra.TemplatingModule` in **both** API and worker options |
| `internal/bootstrap/modules/repositories.go` | the four `documenttemplaterepository.New*` |
| `internal/bootstrap/modules/validators.go` | `documenttemplateservice.NewValidator` |
| `internal/bootstrap/modules/api/services.go` | `documenttemplateservice.New`, the `DocumentTemplateResolver` binding, the `DetentionNoticeService` binding, and **three** `ContextBuilder` group entries (invoice, detention email, detention PDF) |
| `internal/bootstrap/modules/api/handlers.go` | `documenttemplatehandler.New` |
| `internal/api/router.go` | handler field, param, assignment, and `RegisterRoutes` |
| `gqlgen.yml` | the nine `DocumentTemplate*` model/enum bindings |
| `internal/api/graphql/projection/projection.yml` | `virtuals.DocumentTemplateVersion.starterDrifted` |
| `internal/api/graphql/projection/gen/generator.go` | four `DocumentTemplate*` DTOs **and** `ShipmentCommentAttachment` in `nonProjectionObjects` |
| `internal/infrastructure/database/seeds/register_gen.go` | `NewDocumentTemplateStartersSeed()` (regenerate, never hand-edit) |

A one-liner that checks all of them:

```bash
cd services/tms && go test ./internal/bootstrap/    # fx.ValidateApp, both graphs
```

**2. Two files in the diff are not template work.**

- `internal/api/graphql/resolver/shipmentmapping.go` — a two-line import re-sort.
  `HEAD` fails `goimports` on this file; a directory-wide `goimports` fixed it.
  Harmless to keep, harmless to drop.
- `internal/api/graphql/projection/gen/generator.go` — the
  `ShipmentCommentAttachment` entry fixes a **pre-existing break**: the projection
  generator does not run at `HEAD` at all. Verified by stashing this work and
  re-running. Do not drop it or nobody can regenerate specs.

**3. The migration is applied to the local dev database and the starter seed has
run there** (16 drafts, 2 organizations). If you `git stash` this work, roll the
migration back first (`task db-rollback`) or the schema and the code disagree.

**4. Do not run a formatter over a whole directory.** `golines -w <dir>` reflowed
four detention files this work never touched; the noise had to be reverted. Format
by explicit file path.

### The generator order still matters

`task generate-columns` → `go generate ./internal/api/graphql/projection/...` →
`task gqlgen` → `task generate-seeds` → `pnpm --filter @trenova/graphql codegen` →
**rebuild the Go binary**. The client codegen also writes
`services/tms/internal/api/graphql/persisted-documents.json`, which the server
`go:embed`s, so a new or changed operation 404s until the binary is rebuilt.

The mocks are hand-maintained in the mockery-testify shape (copied from
`mock_PDFRenderer.go`); the project forbids running mockery. Repositories with no
generated mock at all — the detention ones — are stubbed inside the `_test.go` that
needs them.

---

## 11. Ordered plan for the next agent

1. ~~`DetentionPolicy.AttachNoticePDF`~~ — **done**, see §7. Phase 6 is closed.
2. ~~**Phase 7 emails**~~ — **done**, all six senders. See §8.2 for what each one
   turned up.
3. **Phase 7 notifications** (§8.2). ~30 sites, grouped into ~6 context families
   (decided — see §8.2).
4. **`report.pdf`** (§8.2). The kind, the starter, and the catalog exist and
   nothing renders through them. Needs the tenant on `ReportRunMeta` first.
5. **Phase 8 admin UI** (§8.3). Everything the server needs is in place; no
   client GraphQL operations exist yet.

Each of 1–3 follows the same four steps: register the kind if it is missing, add
starter assets, write a `ContextBuilder` **depending on repositories rather than
the owning service**, and swap the call site to `RenderMessage`. Copy
`invoiceservice/templatecontext.go` — it is the reference implementation.

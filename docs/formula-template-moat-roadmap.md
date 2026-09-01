# Formula Template Moat Roadmap

Working checklist from the 2026-09-01 review of the formula template engine, service,
and Formula Studio. Goal: formula templates are Trenova's moat, so the Studio must show
authors the same numbers production charges, the approval workflow must be enforced by
the API rather than the UI, and the experience must be system-driven (the system
surfaces what to do next instead of the user having to know).

Items are checked off as they are implemented **and verified** (build + tests green,
manual check in the running app where noted). Each item names the file(s) it lands in.
Work the phases in order; phase 1 is trust and nothing in phases 3–4 is worth shipping
on top of numbers people cannot believe.

Companion doc: [formula-template-roadmap.md](formula-template-roadmap.md) (2026-08-31
items #2–#5, all complete).

---

## Phase 1 — Trust: the Studio must show what production charges

### 1.1 Rate-table lookups in preview and scenarios

`EvaluateWithEnv` passes a nil lookup, so the engine injects a stub that returns `0`
for every `lookup`/`lookup2` call. Any matrix-based template previews wrong and its
approval-gating scenarios prove nothing.

- [x] Thread the tenant `RateTableLookup` into `formula.Service.EvaluateWithEnv` and
      `formulatemplateservice.TestExpression` (`formula/service.go`,
      `formulatemplateservice/service.go` `testExpressionWithEnv`), using
      `ratetablecache.With` so a preview loads tables once
- [x] Scenario runs (`formulatemplateservice/testcases.go` `runSingleCase`) evaluate
      with real tables; a scenario referencing a missing table fails with a readable
      error instead of passing at `$0`
- [x] `lookupOr` swallows only "key not found"; arity, key-type, and missing-table
      errors propagate (`formula/engine/lookup.go`)
- [x] `EvaluatePredicate` keeps the documented stub behaviour, but the stub is opt-in
      by name, not the silent default for every nil provider
- [x] Tests: preview with a lookup returns the matrix value; scenario against a
      missing table fails; `lookupOr` arity error is not swallowed

### 1.2 Approval impact compares against what was actually charged

`ResolveEffectiveTemplate` only diverges from the current row when a scheduled
`effective_from` version exists. Normal saves and approvals never set one, so
`POST /:id/impact` re-rates the pending content against itself and the approve
dialog nearly always reports "pricing-neutral". Same flaw affects backtest's
"current" side.

- [x] "Current" for each shipment = `RatingDetail.Result` when present (what the
      customer was charged), else re-rate with the version recorded in
      `RatingDetail.VersionNumber`, else the last `Active` snapshot
      (`formulatemplateservice/impact.go`, `backtest.go` `resolveEffectiveForShipment`)
- [x] Version repo gains `GetLastActiveSnapshot(templateID)`
      (`formulatemplateversionrepository`)
- [x] Tests: impact with a template whose content changed since the shipment was
      rated reports a non-zero delta; backtest current side uses the recorded version
- [ ] Manual: approve dialog shows movers for a template with edited content

### 1.3 Status is owned by the workflow, not the payload

`Update` permits any `CanTransition` move (InReview→Active included) with update
permission only; `Create` never forces Draft; bulk status flips Inactive→Active
without re-validation; any snapshot (Draft included) can be scheduled onto an
Active template.

- [ ] `Create` forces `Status = Draft` and clears approval fields regardless of
      payload (`formulatemplateservice/service.go`)
- [ ] `Update` rejects any status change in the payload with a field error on
      `status`; Submit/Approve/Reject are the only movers
- [ ] `BulkUpdateStatus` restricted to Active→Inactive; reactivation goes through
      Submit → Approve (or, if kept, re-validates expression + lookup tables and
      snapshots) (`service.go` `BulkUpdateStatus`)
- [ ] `UpdateVersionEffectiveDate` requires the snapshot to be `Active`
      (`formulatemplateservice/effectivedate.go`); Reject and material Update clear
      pending scheduled versions and say so in the audit comment
- [ ] `PATCH /:id` handler stops binding `status` (`formulatemplatehandler/handler.go`)
- [ ] Tests: create-with-Active lands Draft; PUT InReview→Active is 422; scheduling a
      Draft snapshot is 422; reject clears schedules

### 1.4 One rounding step for money

Money is float64 end-to-end; `round()` is binary half-away; agreement rules round
with the agreement's mode; the fallback-template path never rounds and Postgres
truncates to 4dp. Preview ≠ stored ≠ agreement-path to the cent.

- [ ] Template gains `RoundingMode` (half-up, half-even, up, down) and `Precision`
      (default 2) (`domain/formulatemplate/formulatemplate.go`, migration + SQLite
      mirror, GraphQL SDL, client zod schema)
- [ ] `formula.Service.Rate` applies rounding once, after guardrails; preview, test,
      scenarios, backtest, impact, and production all go through `Rate`
      (`formula/service.go`)
- [ ] Decimal-aware `round`, `roundUp`, `roundDown`, `roundTo(x, increment)`,
      `roundHalfEven` in the function library using `shopspring/decimal`
      (`formula/engine/functions.go`, `functionmeta.go`); client fallback list mirrored
- [ ] `bool` results are rejected for rating (`engine.go` `toDecimal`), not coerced
      to `$1`
- [ ] Studio: rounding controls beside Guardrails; preview shows raw vs rounded when
      they differ
- [ ] Tests: half-even vs half-up on `2.675`; production and preview produce the
      same cents for the same input

### 1.5 Timeout race in the engine

After the 5s deadline fires, the caller deletes `__ctx` from `env` while the
abandoned VM goroutine may still read it. Concurrent map access is a fatal runtime
crash.

- [ ] `run()` evaluates against a shallow copy of `env` (or waits for the goroutine
      with a hard cap) and the post-timeout `delete` is removed
      (`formula/engine/engine.go` `run`, `evaluateProgram`)
- [ ] Deadline configurable per caller (interactive vs batch)
- [ ] Test: a deliberately slow expression times out and the process stays healthy
      under `-race`

### 1.6 Null-safe schema fields

Validation substitutes `0` for nullable numbers, so `weight * rate` validates and
then fails on a shipment with no weight, blocking the shipment save.

- [ ] Save-time validation also runs against a nil-shaped env and surfaces the
      outcome as a warning to the author (`formula/engine/environment.go`,
      `formulatemplateservice/service.go` `validateExpression`)
- [ ] `TestExpressionResponse.Warnings` rendered in the preview pane as an amber
      notice with the offending variable and a one-click `coalesce(...)` fix
- [ ] Rating a shipment with a nil-arithmetic failure produces a 422 with the
      variable name, never a 500
- [ ] Test: nullable `weight` with no value yields the warning, not a pass

### 1.7 Error classification and small correctness fixes

- [ ] `SchemaError`/`ComputeError`/`TransformError` map to 422 with the expression
      path (`formula/errors/errors.go`, `internal/api/helpers/classifier.go`)
- [ ] Version repo wraps `sql.ErrNoRows` (`formulatemplateversionrepository.go`
      `GetByTemplateAndVersion`, `UpdateTags`, `UpdateEffectiveDate`); unique
      violations on test-case names map to 409/422
- [ ] `compareVersions` routes errors through `ErrorHandler`
      (`formulatemplatehandler/handler.go`)
- [ ] Permission middleware on `select-options` routes; shipment-read check on
      `backtest` and `impact`
- [ ] `Fork` fails when the requested version does not exist instead of forking
      latest (`service.go` `resolveTemplateSnapshot`)
- [ ] `Rollback` and `CreateVersion` run `validateTemplate`; `CreateVersion` writes
      an audit entry
- [ ] `CountUsages` includes rate agreement rules/accessorials, rate matrices, rate
      quotes, and customer fallback templates
      (`formulatemplaterepository/formulatemplate.go`)
- [ ] Unique index `(organization_id, business_unit_id, name)` and list index
      `(organization_id, business_unit_id, created_at DESC)`; index on
      `accessorial_charges.formula_template_id`
- [ ] Body/field caps: expression length, variable/breakdown counts, `templateIds`
      length on duplicate/bulk (`domain/formulatemplate/formulatemplate.go`)

---

## Phase 2 — Feedback loop: the Studio tells you what is wrong and what is next

### 2.1 Surface real errors

- [ ] Approval dialog routes failures through `handleMutationError` so self-approval,
      failing scenarios, and invalid expressions show the server's message
      (`approval-action-dialog.tsx`)
- [ ] Preview shows an error card when the test request fails; previous result is
      dimmed while pending; last successful run is timestamped
      (`studio/use-live-preview.ts`, `studio/studio-preview-pane.tsx`)
- [ ] Schema fetch failure shows a non-blocking banner ("using built-in reference")
      instead of silently linting against the fallback (`hooks/use-formula-schema.ts`)
- [ ] Validation errors in the collapsed Details section auto-expand it and badge
      the trigger (`studio/studio-editor-pane.tsx`)
- [ ] Save handlers use `mutate` or catch, no more `void onSave()` over a rejecting
      `mutateAsync` (`new/page.tsx`, `[id]/page.tsx`)

### 2.2 Scenarios stay live

- [ ] Scenarios run on the same debounce as preview against the editor candidate;
      header shows a live `n/m passing` badge (`studio/studio-scenarios-pane.tsx`,
      `studio/studio-header.tsx`)
- [ ] Results are marked stale (not cleared) when the expression changes and re-run
      automatically after add/edit/delete
- [ ] "Pin as scenario" button on a green preview card, prefilled from the current
      sample and result (`studio/studio-preview-pane.tsx`)
- [ ] "Use these values" on the resolved-variables view of a real-shipment preview
      copies them into test data or a new scenario

### 2.3 Readiness checklist gates Submit and Approve

- [ ] `ReadinessPanel` computed client-side: lint clean, preview valid, scenarios
      passing, no unsaved changes, description present, rate tables resolvable,
      submitter ≠ current user for Approve (new
      `studio/readiness-panel.tsx`)
- [ ] Submit button disabled while dirty with a tooltip explaining why; Approve
      dialog shows the checklist above the impact panel
- [ ] Server-side `GET /:id/readiness` returning the same checks so the list page
      and notifications can use it (`formulatemplateservice/readiness.go`,
      handler route)

### 2.4 Say what saving will do

- [ ] Saving a material change to an Active or InReview template shows a confirm
      dialog: "This will return the template to Draft and stop it rating shipments
      until re-approved" (`[id]/page.tsx`)
- [ ] Usage chip copy is state-aware; the rollback dialog copy says the template
      returns to Draft on material change (`studio/studio-header.tsx`,
      `version/rollback-confirm-dialog.tsx`)
- [ ] Save toast reports the resulting status when it changed

### 2.5 Schema-driven test data

- [ ] `TestDataEditor` iterates the fetched schema (`useFormulaSchema`) instead of
      the hardcoded fallback list; custom variables get their own section; enum
      fields render as selects (`components/formula-editor/test-data-editor.tsx`)
- [ ] Scenario dialog validates through `formulaTestCaseInputSchema` (name ≤ 100,
      etc.) instead of hand-rolled checks (`studio/scenario-dialog.tsx`)
- [ ] Custom-variable default coercion re-runs when the declared type changes
      (`components/formula-editor/variable-definition-editor.tsx`)
- [ ] Duplicate custom-variable and breakdown names (against each other and the
      schema) produce field errors

### 2.6 Editor correctness and performance

- [ ] Editor theme uses `useResolvedTheme` so system-dark users get the dark
      CodeMirror theme (`components/formula-editor/expression-editor.tsx`)
- [ ] Language/completion extensions live in a `Compartment` reconfigured on
      identifier changes instead of rebuilding every editor on each keystroke
- [ ] Reference pane inserts into the focused editor (main or breakdown mini),
      not always the main one (`studio/formula-studio.tsx` `handleInsert`)
- [ ] Explain panel clears or marks stale when the expression changes
      (`studio/ai/ai-explain-panel.tsx`)

### 2.7 Dead-ends and wiring

- [ ] Lineage dialog nodes navigate to the template (wire `onNavigateToTemplate`
      from `formula-studio.tsx` and `formula-template-table.tsx`)
- [ ] Fork success navigates into the new template's studio
- [ ] Import from inside the studio navigates to the imported template (or the
      list when several)
- [ ] Fork dialog resets its defaults when the target template changes
      (`fork-template-dialog.tsx`)
- [ ] Query keys go through `queries.formulaTemplate.*` everywhere
      (`approval-action-dialog.tsx`, `rollback-confirm-dialog.tsx`,
      `fork-template-dialog.tsx`)
- [ ] Route strings centralised in one `formulaTemplateRoutes` helper

### 2.8 Keyboard, layout, accessibility

- [ ] Shortcuts: `Ctrl/Cmd+S` save, `Ctrl/Cmd+Enter` run preview, `Ctrl/Cmd+K`
      focus reference search, `Ctrl/Cmd+1/2` toggle Preview/Scenarios; shortcut
      hints in tooltips
- [ ] Below ~1100px the right column stacks under the editor as tabs instead of
      nested resizable panes; sheets use `max-w` not fixed widths
- [ ] Hover-only delete buttons get `focus-visible:opacity-100`; icon buttons get
      `aria-label`; labels get `htmlFor`; clickable `div`s in lineage/version
      history become buttons
- [ ] Terminology pass: "Scenarios" everywhere in the UI (API keeps `testCases`),
      "Sample data" as the single term for preview inputs, status shown once,
      reference categories match the schema's category ids

### 2.9 List page carries decision info

- [ ] Columns: in-use count, scenario count with pass state, last approved by/at,
      updated at (`formula-template-columns.tsx`; extend the GraphQL fragment)
- [ ] Filters: pending review, by approver, by source template, by referenced rate
      table (`formulatemplaterepository` `filterQuery`; invalid filter values are
      422, not silently ignored)

---

## Phase 3 — Differentiators: transparency nobody else has

### 3.1 Calculation receipt

Every rating carries a trace that a non-programmer can read.

- [ ] Recording `RateTableLookup` decorator captures `{table, keys, matchedKey or
      band, value}` per call (`formula/engine/lookup.go`)
- [ ] Variable provenance map (schema field, override, declared default, caller
      variable, computed) built alongside the env (`formula/engine/engine.go`
      `Evaluate`, `environment.go`)
- [ ] `Trace` on `EvaluationResult` and `CalculateResponse`: pre-clamp raw value,
      effective version + `EffectiveFrom`, breakdown line results, lookups,
      provenance, evaluation time (`pkg/formulatemplatetypes/engine.go`)
- [ ] Trace carried into `ratetypes.Component.Detail` and
      `shipment.RatingDetail` (replace the intentionally-empty `ResolvedVariables`)
      (`rateengine/pricing.go`, `shipmentcommercial/commercial.go`)
- [ ] Studio preview renders the receipt: variables with source badges, lookup hits
      with the matched band, guardrail clamp, rounding step
- [ ] Shipment "Why this rate" shows the same receipt and links to the template
      version that produced it
- [ ] Func values stripped at the engine boundary, not in the handler

### 3.2 Reviewer sees a diff

- [ ] Endpoint: diff between the last `Active` snapshot and the submitted content
      (`formulatemplateservice` `CompareWithLastApproved`, handler route)
- [ ] Approve dialog shows expression diff (reuse `ExpressionDiff`) plus the
      corrected impact panel; Submit dialog shows the same to the author
- [ ] Notification for reviewers links to the studio with the diff open

### 3.3 Hover-to-evaluate

- [ ] CodeMirror `hoverTooltip` shows each identifier's current value and source
      ("500 mi, from shipment SHP-1234") from the last preview result
      (`components/formula-editor/expr-hover.ts`)
- [ ] Ternary branches that fired in the last run are subtly highlighted; the other
      branch is dimmed

### 3.4 Breakdown reconciliation

- [ ] Preview shows Σ breakdown vs total and an "unallocated" residual row
- [ ] Warning when guardrails clamped the total but breakdown lines still sum to the
      raw amount; optional "scale lines to clamped total" setting on the template
- [ ] Breakdown row errors map back to the matching mini-editor as inline
      diagnostics

### 3.5 Library and starters

- [ ] Starter picker sourced from the standard catalog endpoint instead of the four
      hardcoded starters (`studio/starter-templates.ts` → API)
- [ ] "Start from existing template" option that forks in place
- [ ] AI generate returns two or three proposed scenarios with expected amounts
      computed through `/formula-templates/test`; author accepts them one click
      (`studio/ai/ai-generate-panel.tsx`, `formulaassistantservice`)

### 3.6 Backtest drill-in

- [ ] Rows link to the shipment; summary counts guardrail clamps and evaluation
      failures; CSV export
- [ ] Version picker is a dropdown from `listVersions` with tags and dates, not a
      free number (`formula-template-backtest-tab.tsx`)
- [ ] Backtest sheet shows a skeleton while lazy-loading (`backtest-sheet.tsx`)

---

## Phase 4 — Expressiveness and scale

### 4.1 Function library

- [ ] Variadic `min`/`max`, `avg`; publish expr's string builtins (`startsWith`,
      `contains`, `matches`, slicing) in `DescribeSchema` so the reference pane and
      linter know them (`formula/engine/functionmeta.go`)
- [ ] Lookup key normalisation modes: `trim`, `upper`, `zip3`, and nearest/clamp-to-
      top-band options (`formula/matrix_lookup.go`)
- [ ] `lookupInterp(table, key)` linear interpolation between bands
- [ ] Deficit-weight helper for CWT pricing ("rate as next break if cheaper")

### 4.2 Schema expansion

- [ ] `pickupDate`/`deliveryDate` as expr dates plus per-location timezone (new
      `timezone` column on `locations`, IANA) so `pickupHour`/`isWeekendPickup` are
      local-time correct (`formula/resolver/computed_lane.go`,
      `schema/definitions/shipment.schema.json`)
- [ ] `stops[]` (state, type, appointment window) and `commodities[]` (class,
      dims, hazmat) exposed as arrays; stop shadowing expr's `sum` so `map`/
      `filter`/`sum` work over them
- [ ] Dimensions and freight class for dim-weight formulas; `serviceType`,
      `shipmentType`, `currency`
- [ ] `fuelPrice` as a schema-level computed variable fed by the fuel-price job so
      FSC tables work in preview and scenarios

### 4.3 Performance

- [ ] Process-level rate-table cache keyed by tenant + matrix version stamp,
      invalidated from the rate-matrix write path, replacing per-request full loads
      (`pkg/ratetablecache`, `formula/service.go` `buildLookup`)
- [ ] Compile-cache key hashes declared schema types, not runtime nil-vs-float
      shapes; key computed once per `Evaluate`; size configurable
      (`formula/engine/engine.go`)
- [ ] Boxed `Moves`/`Stops` walk cached once per env build
      (`formula/resolver/computed_lane.go`)
- [ ] Two-axis range-row tables use a sorted index instead of a linear scan

### 4.4 Service-layer structure

- [ ] Split `formulatemplateservice/service.go` into `versions.go`,
      `testexpression.go`, `validator.go`
- [ ] One `templateSnapshot`-based constructor used by Fork, Duplicate, Import, and
      InstallStandards; `buildDuplicateEntity` moves out of the repository
- [ ] Repositories use `buncolgen` column helpers and `DBForContext` everywhere
      (`List`, `CountUsages`, `SelectOptions`)
- [ ] Approval engine records review rounds (reviewer, decision, comment, diff-base
      version) with "request changes" distinct from Reject, and expiry on stale
      submissions (`pkg/approvalworkflow`, migration)

---

## Verification checklist per phase

- Backend: `task lint`, `task test`; `task test-integration` for repo/migration
  items; `task docs-generate` after handler changes; `task gqlgen` after SDL changes
- Client: `pnpm --filter @trenova/web typecheck`, `lint`, `test`;
  `pnpm --filter @trenova/graphql codegen` after fragment changes
- Manual (`task run-watch` + `pnpm dev`): preview a lookup-based template and
  confirm the matrix value; approve a template with edited content and confirm
  movers; PUT a status change and confirm 422; toggle system dark mode and confirm
  the editor follows

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

- [x] `Create` forces `Status = Draft` and clears approval fields regardless of
      payload (`formulatemplateservice/service.go`)
- [x] `Update` rejects any status change in the payload with a field error on
      `status`; Submit/Approve/Reject are the only movers
- [x] `BulkUpdateStatus` restricted to archiving (→Inactive); an archived template
      is reactivated by Submit → Approve (`Inactive → InReview` is now the only
      transition out of Inactive; the Studio labels it "Reactivate via Review")
- [x] `UpdateVersionEffectiveDate` requires the snapshot to be `Active`
      (`formulatemplateservice/effectivedate.go`); Reject and material Update clear
      pending scheduled versions (`versionRepo.ClearScheduled`; the Update audit
      comment carries the count)
- [x] `PATCH /:id` handler stops binding `status` (`formulatemplatehandler/handler.go`)
- [x] Tests: create-with-Active lands Draft; PUT InReview→Active is 422; scheduling a
      Draft snapshot is 422; reject clears schedules

### 1.4 One rounding step for money

Money is float64 end-to-end; `round()` is binary half-away; agreement rules round
with the agreement's mode; the fallback-template path never rounds and Postgres
truncates to 4dp. Preview ≠ stored ≠ agreement-path to the cent.

- [x] Template gains `RoundingMode` (half-up, half-even, up, down) and `Precision`
      (default 2) (`domain/formulatemplate/formulatemplate.go`, migration + SQLite
      mirror, GraphQL SDL, client zod schema)
- [x] `formula.ApplyChargePolicy` (guardrails, then rounding) is the single step
      behind `Rate`, the Studio preview, and scenarios, so production, preview,
      backtest, and impact all land on the same cents (`formula/service.go`).
      Agreement-priced rules still apply the agreement's own rounding on top,
      as the contract's outer policy
- [x] Decimal-aware `round`, `roundUp`, `roundDown`, `roundTo(x, increment)`,
      `roundHalfEven` in the function library using `shopspring/decimal`
      (`formula/engine/functions.go`, `functionmeta.go`); client fallback list mirrored
- [x] `bool` results are rejected for rating (`engine.go` `toDecimal`), not coerced
      to `$1`
- [x] Studio: rounding controls beside Guardrails; preview shows raw vs rounded when
      they differ
- [x] Tests: half-even vs half-up on `2.675`; production and preview produce the
      same cents for the same input

### 1.5 Timeout race in the engine

After the 5s deadline fires, the caller deletes `__ctx` from `env` while the
abandoned VM goroutine may still read it. Concurrent map access is a fatal runtime
crash.

- [x] `run()` evaluates against a shallow copy of `env` and the post-timeout
      `delete` is removed (`formula/engine/engine.go` `run`, `evaluateProgram`)
- [x] Deadline configurable per caller via `engine.WithEvaluationTimeout`; the
      Studio preview runs on a 2s leash, batch paths keep the 5s default
- [x] Test: a blocking function outlives the deadline, the caller gets
      `DeadlineExceeded` on time, the caller's env is untouched after the
      goroutine finishes, and the goroutine exits (`engine_ctx_test.go`; the
      race detector needs gcc, which this machine lacks, so run `-race` in CI)

### 1.6 Null-safe schema fields

Validation substitutes `0` for nullable numbers, so `weight * rate` validates and
then fails on a shipment with no weight, blocking the shipment save.

- [x] `engine.UnguardedNullableFields` re-compiles the expression with each
      referenced nullable field nil-shaped; the preview returns the result as
      `warnings[]` (scope, field, suggestion) and save-time validation logs it
      (`formula/engine/nullable.go`, `formulatemplateservice/service.go`)
- [x] `TestExpressionResponse.warnings` rendered in the preview pane as an amber
      notice per field with a one-click "Use coalesce(field, 0)" fix that rewrites
      the expression or breakdown line in place (`guard-nullable.ts`)
- [x] Rating a shipment whose nullable field is empty yields
      `MissingFieldError` (names the field and the guard) which `Rate` maps to a
      validation error, so it surfaces as a 4xx naming the field instead of a 500
- [x] Tests: unguarded `weight` warns (engine, service, breakdown scope); guarded
      does not; nil `weight` on a real record is a validation error naming it

### 1.7 Error classification and small correctness fixes

- [x] `SchemaError`/`ComputeError`/`TransformError`/`VariableError`/`ResolveError`/
      `MissingFieldError` classify as validation problems (`classifyFormula` in
      `internal/api/helpers/classifier.go`); the engine's own deadline is
      `engine.ErrEvaluationTimeout`, distinct from a caller's context deadline
- [x] Version repo wraps `sql.ErrNoRows` (`formulatemplateversionrepository.go`
      `GetByTemplateAndVersion`, `UpdateTags`, `UpdateEffectiveDate`); unique
      violations on test-case names map to 409/422
- [x] `compareVersions` routes errors through `ErrorHandler`
      (`formulatemplatehandler/handler.go`)
- [x] Permission middleware on `select-options` routes; shipment-read check on
      `backtest` and `impact`
- [x] `Fork` fails when the requested version does not exist instead of forking
      latest (`service.go` `resolveTemplateSnapshot`)
- [x] `Rollback` and `CreateVersion` run `validateTemplate`; `CreateVersion` writes
      an audit entry
- [x] `CountUsages` includes rate agreement rules and rate agreement accessorials
      (the only other tables that reference templates; quotes are records, not
      consumers) and the Studio/rollback dialog label them
      (`formulatemplaterepository/formulatemplate.go`)
- [x] Unique index `(organization_id, business_unit_id, name)` and list index
      `(organization_id, business_unit_id, created_at DESC)`; index on
      `accessorial_charges.formula_template_id` (migration `20261002000000`);
      duplicates pick a free "(Copy N)" name and a name collision on create or
      update is a 409 on `name`
- [x] Field caps: expression ≤ 10,000 chars, ≤ 50 custom variables (breakdowns
      were already capped at 20), `templateIds` 1–100 on duplicate/bulk status
      (`domain/formulatemplate/formulatemplate.go`, service `validateBulkTemplateIDs`)

---

## Phase 2 — Feedback loop: the Studio tells you what is wrong and what is next

### 2.1 Surface real errors

- [x] Approval dialog shows the server's reason inline (self-approval, failing
      scenarios, invalid expression) via `describeApiError`
      (`lib/api-error-message.ts`, `approval-action-dialog.tsx`)
- [x] Preview shows an error card when the test request fails; previous result is
      dimmed while pending; last successful run is timestamped
      (`studio/use-live-preview.ts`, `studio/studio-preview-pane.tsx`)
- [x] Schema fetch failure shows a non-blocking banner ("using built-in reference")
      instead of silently linting against the fallback (`hooks/use-formula-schema.ts`)
- [x] Validation errors in the collapsed Details section auto-expand it and badge
      the trigger (`studio/studio-editor-pane.tsx`)
- [x] Save handlers use `mutate` or catch, no more `void onSave()` over a rejecting
      `mutateAsync` (`new/page.tsx`, `[id]/page.tsx`)

### 2.2 Scenarios stay live

- [x] Scenarios run on the same debounce as preview against the editor candidate;
      header shows a live `n/m passing` badge (`studio/studio-scenarios-pane.tsx`,
      `studio/studio-header.tsx`)
- [x] Results are marked stale (not cleared) when the expression changes and re-run
      automatically after add/edit/delete
- [x] "Pin as scenario" button on a green preview card, prefilled from the current
      sample and result (`studio/studio-preview-pane.tsx`)
- [x] "Use these values" on the resolved-variables view of a real-shipment preview
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

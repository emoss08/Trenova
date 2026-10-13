# Handoff: Formula Studio (create & edit formula templates)

## About the design files
`design/Formula Studio.html` is a **high-fidelity HTML prototype**. Open it in a browser to see layout, copy, states and behavior. It is a design reference, not code to ship. Recreate it in `client/apps/web` with the existing React, Tailwind, Base UI, react-hook-form, CodeMirror and field-component patterns.

- `design/engine.jsx` is a **browser stand-in for the formula engine**: expr tokenizing, lint, evaluation, the charge policy and the backtest. **Do not port it.** Production uses `useLivePreview`, `useLiveScenarios`, `formula-test-request.ts`, the backtest and readiness endpoints, and `expr-lint.ts`.
- `design/editor.jsx` stands in for `ExpressionEditor` (CodeMirror). `fields.css` stands in for the real form components.
- `design/models.jsx` is a copy of the Desk model picker (`desk-model-picker.tsx`).
- `design/chrome.jsx` holds the top bar and the rate trace, plus stand-ins for `ScrollArea`, `DropdownMenu`, `Popover` and `HoverCard` that copy `@trenova/shared/components/ui/*`. Use the real components. `scroll-area.css` styles the ScrollArea stand-in.
- `design/tweaks-panel.jsx` is prototype-only. It switches create/edit, Draft / In review / Active, light/dark, and trace/classic chrome (classic is the previous layout, kept for comparison; build trace).
- Nova calls a live model in the prototype and falls back to canned answers offline. Production uses the formula assistant agent.

Fidelity: **hifi** for structure, copy and states. Visual values come from `packages/shared/src/styles/tokens.css`. The token block at the top of `studio.css` is copied from it.

### Files
- `app.jsx`: shell, rail tabs, save/confirm/submit/backtest dialogs, Nova orchestration, toasts
- `form.jsx`: Details hero, Expression (start card, editor, problems, explain), Inputs, Charge policy, Breakdown
- `chrome.jsx`: top bar (`StudioTop`, `Lifecycle`), rate trace (`RateTrace`), shared-component stand-ins
- `rail.jsx`: Preview, Scenarios (+ `ScenarioWhy`), Reference
- `ai.jsx`: Nova prompts and fallbacks, the proposal diff card, proposed scenarios, Nova pane (thread + composer)
- `data.jsx`: the shipment schema (mirrors `SHIPMENT_VARIABLES` / `AVAILABLE_FUNCTIONS`), standards (mirrors `standard_templates.yaml`), the sample template, scenarios, usage

## Layout
- **Top bar** (`StudioTop`): 52px, bottom border, canvas background.
  - **Left:** back ghost icon button, then **the template name only** (14px/500; placeholder "New rating method" in subtle). No breadcrumb or eyebrow line. A 1px divider follows.
  - **Lifecycle track** (replaces the status pill + version badge). Stages are 7px dots joined by 18px lines.
    - **Create mode:** "New" (brand dot) → Draft → In review → Active.
    - **Past stages:** success check, muted label. The line after a past stage is success at 60% opacity.
    - **Current stage:** a card-colored pill with a 1px border, tone dot (Draft muted / In review info / Active success) and a slow halo pulse, plus a `v{n}` chip.
    - **Active:** a dashed-underlined "N in use" follows, as a `HoverCard`. It lists usages and the edit consequence (copy from `studio-header.tsx`).
    - **Unsaved material edit on Active or In review:** Draft gets a dashed warn ring and an "on save" chip, and the current pill dims.
    - **Archived:** a single "Archived · v{n}" pill.
  - **Right:**
    - **Status actions:** Draft has Submit for review (disabled while dirty). In review has Approve + Request changes; Reject sits in ⋯.
    - **Save button**, which carries the save state: "Create template" (create), "• Save changes ⌘S" with a warn dot (dirty), or quiet "✓ Saved" text (clean).
    - **⋯ More** comes last: a `DropdownMenu` with `align="end"`. Items: Version history (description `v1 – v{n}`), Backtest ("Re-rate recent shipments"), separator, Fork template, Export JSON, Import templates, and on In review a separator + Reject and archive with `color="danger"`.
  - **Responsive:** below 1100px the divider and "in use" hide. Below 760px the lifecycle track hides.
- **Body:** grid `256px | 1fr | minmax(360px,420px)`.
  - **Rate trace:** left column, right border (see "Rate trace").
  - **Form:** centered column, max-width 800px, padding 28px 36px 140px.
  - **Rail:** tabs Preview · Scenarios (n/m pill) · Reference · Nova (right-aligned, ring mark). The active tab has a 2px `fg` underline. The Nova tab label uses the same colors as the other tabs.
- **Responsive:** below 1280px the trace hides. Below 900px the rail stacks under the form and the form scrolls with the page.

## Rate trace
A live column that follows one load (the preview's sample load or picked shipment) through the template, top to bottom. It replaces the section outline and the separate readiness list. All values come from the preview run.
- **Header:**
  - "Rate trace" with a live indicator: success "live" with a breathing dot / "waiting" when empty / "stopped" on error.
  - Below it, the load as a ghost row ("Sample load · 742 mi · 3 stops", or "{shipment} · {lane}"), with a warn "n edited" chip when values are overridden. Clicking it opens Preview.
- **Steps:** a vertical 1px spine joins the step markers.
  - **Markers:** 11px rings. Ok has an inner success dot; warn/hit an inner warn dot; bad a danger ring with a pulsing dot; todo an empty ring.
  - **Current section:** the step for the section scrolled into view gets an `fg` ring.
  - **Header rows:** each step's header row jumps to its form section.
  1. **Reads**, "n values": up to 4 mono rows of name and value, then "+n more in Preview". Warns "n inputs defined but never read".
  2. **Formula**, "n lines": the raw amount (16px mono). On error, "n errors in the expression" or the runtime error in danger. Empty: "Write or draft a formula to start the trace."
  3. **Charge policy:**
     - "Raised to the **$250** minimum" (warn tag) / "Inside $X – $Y" / "No minimum or maximum"
     - then "Rounds half up to 2 places"
  4. **Breakdown**, "n lines" (only when there are lines): a 6px stacked bar of line shares, then up to 4 legend rows with each line's %. Warns "Lines don't add up to the formula".
  5. **Billed:** a card with "Billed", the amount (24px mono), and "on the sample load" / "on {shipment}". It is not on the spine, so the spine stops at the last step above it. Clicking it opens Preview.
- **Amounts:** a short settle animation when they change. **No pulse or beam travels the spine.**
- **Scenarios strip** (sunken; danger-subtle when any fail): "Scenarios hold" / "n scenarios broke" / "No scenarios yet". It shows one 4×10 tick per scenario (pass success, fail danger) and n/m. Clicking it opens the Scenarios tab.
- **Readiness** (pinned to the bottom, top border):
  - Header: "Ready for review" / "Ready to create" / "Ready to approve", with n/m.
  - A segmented 4px meter, one segment per check (pass success / warn / fail danger).
  - Then only the open checks, with their detail, or "Every check passes".
- The column is a `ScrollArea` (`maskVariant="background"`).

## Sections
1. **Details hero**
   - The name is a 26px/500 borderless input, placeholder "Untitled rating method".
   - The description is an auto-growing borderless textarea.
   - Then "Charges" with a Freight / Accessorial segmented control, and a "Schema · Shipment" badge.
   - Failed validation on create tints the field danger-subtle and flashes the outline dot.
2. **Expression**
   - Header buttons: Explain (outline), Ask Nova (Triage hand-off button).
   - **Start card** (create mode, empty expression):
     - A Nova describe field with a "Draft it" ink button.
     - A "Standard library / Your templates" segmented control over a grid of starters (name, mono expression, meta). This replaces `StarterTemplatePicker`.
   - **Editor:** `ExpressionEditor primary` with a line gutter.
     - **Status bar** (sunken): Valid / n errors / "Fails on the preview load", then "n schema · n inputs · n functions", a Ctrl Space hint and Ln/Col.
   - **Problems list** below the editor, one row per issue:
     - Unknown identifier → "Add `x` as input". Infers Boolean for `is*`/`has*`/`requires*`, otherwise Number.
     - Nullable warning → "Use `(x ?? 0)`" (`guard-nullable.ts`).
     - Other errors, and runtime errors on the preview load → "Fix with Nova".
   - **Explain panel:** styled after Triage's decision card (`.nx` + `.why`).
     - Header: Nova ring, "What this charges", "Ask a follow-up", and close.
     - Lead line (14.5px): "On the preview load the formula comes to $X [, under the minimum, so it bills $Y]. Here's how each line gets there."
     - A sunken "why" strip with **one row per statement, including the final line** (marked "=").
     - Hovering a row highlights that statement's range in the editor.
   - **Material-change banner** (info-subtle): shown above the editor when an Active or In review template has unsaved material edits.
3. **Inputs** (replaces `VariableDefinitionEditor`). A table: Name (mono), Type (Number/String/Boolean), Default (number / switch / text), Required (switch), delete.
   - Each row has a borderless description line below it.
   - Tags: "unused" when the expression doesn't reference it, "name taken" when it collides with a schema variable.
   - Header: a Nova "Describe n" button (fills missing descriptions, shown in agent color until edited) and "Add input".
4. **Charge policy:** Minimum / Maximum (`$` NumberField), Rounding (`SegmentedControl` Half up · Half even · Up · Down · None), Decimal places stepper (0–6).
   - A **pipeline strip** on the preview load: Formula $raw → Within range / Raised to minimum → **Billed** (ink chip).
5. **Breakdown** (`BreakdownDefinitionEditor`): each row has Label, mini expression, live amount and delete, plus a small mono key (auto camelCased from the label until edited).
   - Reconcile footer: "Lines explain the total" / unallocated / exceeds / n lines fail.
   - When a guardrail moved the total, the footer warns that the invoice won't match (copy from `studio-preview-pane.tsx`).

## Rail
- **Preview:**
  - Sample load / Real shipment segmented control and a run-time readout in ms. Real shipment shows a picker (`ControlledShipmentAutocompleteField`).
  - **Result card:** "Billed", a 34px mono amount, and the policy notes (guardrail, rounding). Then the breakdown lines and reconcile, then "Receipt · n values read" (expands `ReceiptView`), then "Pin as scenario".
  - **"Values this formula reads"** shows only the variables the expression references. Each value is editable inline with a source tag (shipment info / computed violet / input success / override warn / default / sample). The full list is behind "Show all n shipment variables". A Reset link appears when any value is overridden.
- **Scenarios** (`StudioScenariosPane`):
  - Summary "n/m" (success or danger, "approval is blocked").
  - Header actions: "Suggest edge cases" (Nova) and "Add". Add saves the preview's current values and amount.
  - Rows: pass/fail dot, name, mono inputs, amount.
    - **Failing rows stay compact:** the current amount in danger with the expected amount struck through below it, plus a small "Why?" button with the ring mark. No inline "Expected X, now Y" line and no inline card.
  - **Why it broke** is a `Popover` on the "Why?" button (`align="end"`, 340px, inverted).
    - **Header:** ring mark (spinning while loading), the verdict ("Looks intended" success / "Looks like a regression" danger / "Needs a look" warn), and the signed dollar delta.
    - **Body:** Nova's ≤2-sentence cause; names in backticks render as code chips. Below it, up to 3 sunken trace rows of mono name, was → now.
    - **Footer:** if intended, "Expect $X" is primary with "Fix instead" secondary. Otherwise "Fix with Nova" is primary with a quiet "Expect $X". A faint "vs v{n}" sits at the right; its tooltip names who recorded the scenario.
    - "Fix with Nova" sends the cause to the Nova thread.
    - **Grounding:** the engine evaluates every `let`, the formula, the policy and the billed amount on the scenario inputs, once with the version the scenario was recorded on and once with the draft. It also diffs input defaults that aren't set by the scenario. Nova only explains that trace.
    - **Data:** scenarios need `recordedOnVersion` and `recordedBy`.
  - Scenarios re-run on every edit.
  - **Create mode:** scenarios are held and saved with the template on create. Today they need a saved template, so this is a change.
- **Scroll regions:** the rail body (`maskVariant="card"`), the Nova thread (`background`) and the receipt list (`card`) are `ScrollArea`s.
- **Reference** (`StudioReferencePane`): sticky search (`/`), a Variables / Functions segmented control, then groups.
  - Each row shows name + type and a one-line description; functions also show an example.
  - Clicking a row inserts it at the editor cursor.

## Nova
The assistant is **Nova**. Reuse these existing pieces; don't invent new ones.

| Piece | Source |
|---|---|
| Mark | Nova's gradient ring (Triage `.dm`, conic 259° → 296° → 8°, masked to a ring). Never sparkles or the diamond. |
| Ask Nova / Fix with Nova / Describe / Suggest edge cases / Rewrite buttons | Triage's "Work on it in Desk" hand-off button: outline, ring mark, spectrum conic edge plus a 10% wash on hover, and the ring spins once |
| Thread | Desk thread in compact density (`desk-chat-compact.css`): question as a heading (17 / 15 / long block), narration lines (`turn-status`: past steps faint with a success check, the current step shimmering with a running timer), prose 14/1.65 |
| Composer | `desk-composer.tsx`: 16px card, rainbow `cmp-ring` while busy, a status line with a breathing dot and shimmer, the typewriter placeholder with a Tab hint, round ink send / stop. **Keep only** the page chip ("{template} · n lines · n inputs", read-only) and the model picker. **Drop** attach, the agent picker, dictation and the context meter. |
| Model picker | `desk-model-picker.tsx` as is. Its popover aligns to the composer's right edge. |

- **Empty state:** ring mark, heading ("How should this price a load?" / "What should we change?"), one line of copy, and Desk suggestion rows.
- **Proposal card:**
  - Header "Proposed formula / Proposed change".
  - A line diff against the editor (add success-subtle / del danger-subtle, struck through).
  - "new input" chips for new variables, then the explanation.
  - A schema check line (ok / warn / bad).
  - "Priced by the formula engine": the sample load and 3 real shipments, billed amounts.
  - **Insert into editor** (ink) / Dismiss. After inserting: "In the editor" badge, a note, and Undo (the toast offers Undo too).
  - Inserting merges new variables, and breakdown lines when the proposal has them. It fills the name, description and type only if they're empty.
- **Proposed scenarios card:** rows with mono inputs, the engine-priced expected amount, and Add / Added, plus Add all.

### AI backend needed
- `explainScenario`: the request carries the engine trace (both versions' statement values, policy, billed, input-default diffs). The response is `{ why, verdict: "intended" | "bug" | "unclear" }`. Amounts come only from the trace.
All of these run through the formula assistant agent (`formulaassistantservice`). Amounts are computed by `testexpression.go`, never by the model.
- `explain`: `{ items: [{ statementIndex, says }] }` for each statement, including the final expression (shown last, marked "="). The client adds the lead line from the preview.
- `describeInputs`: `{ name: description }` for variables missing one.
- `suggestScenarios`: inputs only. Each one is priced by the engine and returned as `PricedScenario`.
- `fix`: a normal turn whose prompt carries the lint or runtime error. It returns a `propose_formula` edit.
- `reviewNote`: a summary drafted from the saved-vs-submitted diff and the backtest. It is stored on the review round and shown to the reviewer.

## Dialogs
- **Save and return to Draft?** Shown when saving a material change on Active or In review. Lists the usages; actions are Cancel and "Save and return to Draft". A material save bumps the version.
- **Submit for review:** wide, two columns.
  - Left: readiness list, then reviewer choice.
  - Right: "Note for the reviewer" (Nova-drafted, editable, Rewrite), then "Impact on recent freight": re-rated count, changed count, revenue delta and %, plus the top 3 movers (before → after).
  - Submit is disabled while any check fails.
- **Backtest** (⋯): the same impact block.
- **Toasts:** ink, bottom-center, with an optional Undo.

## Readiness checks (outline + submit)
- Name and description
- Expression is valid (no lint errors, and it runs on the preview load)
- Safe on every shipment (no unguarded nullable fields; warn only)
- Scenarios pass (at least one, all passing; not checked in create mode)
- Breakdown reconciles (warn only)
- Changes saved (edit mode)

These must mirror `formulatemplateservice/readiness.go`.

## Shared components (use the real ones)
| In the mock | Use |
|---|---|
| ⋯ More menu | `DropdownMenu` / `DropdownMenuContent align="end"` / `DropdownMenuItem` (`title`, `description`, `startContent`, `color="danger"`) / `DropdownMenuSeparator` |
| "N in use" | `HoverCard` + `HoverCardContent` (w-64, dark) |
| "Why?" on a failing scenario | `Popover` + `PopoverContent align="end"` (inverted) |
| Every overflow region: form, rate trace, rail body, Nova thread, receipt, dialog bodies, model list, menus | `ScrollArea`. `maskVariant`: `background` on canvas, `card` on cards and dialogs, `popover` inside menus. |

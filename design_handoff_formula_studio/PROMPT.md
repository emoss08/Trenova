Implement the redesigned **Formula Studio**, the create/edit form for formula templates, in the Trenova web client (`client/apps/web`).

Read `design_handoff_formula_studio/README.md` in full, then open `design_handoff_formula_studio/design/Formula Studio.html` in a browser. Use the Tweaks panel to switch between create and edit, Draft / In review / Active, light / dark, and the "trace" / "classic" chrome (classic is the old layout, for comparison only — build "trace").

That file is an HTML design reference: copy its structure, copy and behavior, but rebuild it with the codebase's own components. Don't port the prototype code. In particular:
- `engine.jsx` is a browser stand-in for the formula engine. Every amount in production must come from the server's preview, scenario, backtest and readiness endpoints.
- `chrome.jsx` contains stand-ins for `ScrollArea`, `DropdownMenu`, `Popover` and `HoverCard` so the mock behaves like the app. Use the real ones from `@trenova/shared/components/ui/*`.

## Requirements
1. **Same routes, same form.**
   - Rebuild `routes/formula-template/_components/studio/formula-studio.tsx` and its panes in place. They are used by `formula-template/new/page.tsx` and `formula-template/[id]/page.tsx`.
   - Keep `react-hook-form` with `FormulaTemplateFormValues` and the existing schema.
   - Keep `UnsavedChangesGuard` and `useStudioShortcuts` (⌘S save, ⌘J Nova, `/` reference search).
   - This is a create and edit form, not a detail page.
2. **Layout:** a top bar over three columns:
   - Rate trace, 256px
   - The form, max 800px
   - A tabbed rail with Preview · Scenarios · Reference · Nova, 360–420px
   - Below 1280px the trace hides. Below 900px the rail stacks under the form.
   - Replace the resizable panels with this layout.
3. **Top bar** (README, "Top bar"): back button, the template name only (no breadcrumb or eyebrow), then the lifecycle track (New / Draft → In review → Active).
   - The current stage has a version chip.
   - Active has "N in use" as a `HoverCard`.
   - A material unsaved edit on Active or In review marks Draft as "on save".
   - Right side: status actions, then the Save button, which also shows the save state ("Saved" with a check when clean). The ⋯ `DropdownMenu` comes last.
4. **Rate trace** (README, "Rate trace"): a live column that follows the preview load through the template. It replaces the section outline and the separate readiness list.
   - Steps: Reads → Formula → Charge policy → Breakdown → Billed.
   - Below it: a scenarios strip, then the readiness meter.
   - Each step jumps to its section, and the step for the section in view is highlighted.
   - No travelling pulse or beam animation; only the amounts animate on change.
5. **Use the existing components only.** No raw inputs, selects, toggles, menus or popovers.
   - Use `DropdownMenu`, `Popover` (inverted) and `HoverCard`.
   - Every scrolling region is a `ScrollArea` with the right `maskVariant`. The README lists which goes where.
   - The README's table maps every control to its component.
   - The expression editor stays `ExpressionEditor` (CodeMirror) with `useExprExtensions`, `expr-lint`, `expr-hover` and autocomplete.
   - Breakdown lines use its `variant="mini"`.
6. **Styling:**
   - Use the `tokens.css` utilities only. No shadows; use 1px borders and rings.
   - Primary buttons are `bg-primary` (ink), not brand blue.
   - Cards are `bg-card border rounded-lg`. Support light and dark.
   - The Nova surfaces reuse Desk and Triage styling exactly (README, "Nova"):
     - the Triage hand-off button
     - the gradient ring mark
     - the compact Desk thread and composer
     - the Desk model picker
   - The Nova tab label uses the same colors as the other tabs.
7. **Nova** replaces `FormulaAssistantSheet` with a rail tab built on the shared `PageAssistant` / Desk thread in compact density.
   - Keep `surface: "formula"`, `readDraft` (`formulaDraftFromEditor`) and `propose_formula` edits.
   - The composer shows only the page chip, the model picker and send. Drop attach, agent picker, dictation and the context meter.
   - Proposals render as the diff card in the README. Inserting into the editor can be undone.
8. **New AI features (backend + frontend).** Use the formula assistant agent. Every amount Nova shows must come from the formula engine (`testexpression.go`), never from the model.
   - **Explain:** one item per statement, *including the final expression*, plus a lead line priced on the preview load.
   - **Describe inputs:** fill in missing variable descriptions.
   - **Suggest edge cases:** proposed scenarios (`proposed-scenarios.ts`), priced by the engine.
   - **Fix with Nova:** sends a lint or runtime error to the thread.
   - **Why a scenario broke:** a `Popover` on each failing scenario.
     - The engine evaluates every `let`, the formula, the policy and the billed amount on the scenario's inputs, twice: once with the version the scenario was recorded on, once with the draft. It also diffs input defaults.
     - Nova writes a ≤2-sentence cause from that trace, plus a verdict (intended / regression / unclear).
     - The actions follow the verdict: "Expect $X" or "Fix with Nova".
     - Scenarios must store the template version they were recorded on, and who recorded them.
   - **Reviewer note:** drafted on Submit for review and stored on the review round.
9. **Review flow:**
   - Submit for review opens a dialog with:
     - `ReadinessPanel`
     - a reviewer choice
     - the Nova reviewer note (editable, with Rewrite)
     - the impact from `approval-impact-panel` / backtest
   - Approve / Request changes / Reject keep their permission gates.
   - Saving a material change on an Active or In review template confirms "Save and return to Draft" first.

Work in this order:
1. Layout shell, top bar and rate trace
2. Form sections
3. Rail panes (Preview, Scenarios with "Why it broke", Reference)
4. Nova tab
5. AI endpoints
6. Review dialog

Run typecheck, lint and the existing formula-editor and formula-template tests when you're done. Update `routes.formula-template.json` for es, zh-CN and zh-TW. Placeholder numbers in the mock (usage counts, the backtest, sample shipments, the v3 scenario record) must come from real data.

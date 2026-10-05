# Prompt for Claude Code — Onboarding with Nova

Paste everything below the line into Claude Code, run from the root of the Trenova monorepo (`emoss08/Trenova`, branch `master`). First copy `design_handoff_onboarding/` into the repo root (or anywhere Claude Code can read it).

---

You are rebuilding Trenova's welcome wizard (`/onboarding`) as a full-screen, guided conversation with **Nova**, a scripted setup guide (not AI — every line is a fixed, translatable string). The design reference in `design_handoff_onboarding/` is the source of truth. Match it 1:1 in dark and light: layout, copy, motion, keyboard behaviour, every state.

## Read first
1. `design_handoff_onboarding/README.md` — the full spec: frame, the seven turns with exact copy, each input, review, build/ready, back/edit rules, keyboard, tokens.
2. `design_handoff_onboarding/design/` — the HTML/React prototype. Open `design/index.html` in a browser and click through it (Tweaks: theme, typing speed, first name, restart). Read `app.jsx`/`parts.jsx` for exact behaviour and `onboarding.css` for exact values.
3. The current implementation and its contracts:
   - `client/apps/web/src/routes/onboarding/page.tsx` and `_components/{company,operation,sample-data,review}-step.tsx`
   - `client/apps/web/src/lib/onboarding-form.ts` (steps, `OPERATION_TYPES`, `browserTimezone`, `onboardingFormDefaults`)
   - `client/apps/web/src/types/onboarding.ts` (`onboardingFormSchema`, `toCompleteOnboardingRequest`)
   - `client/apps/web/src/services/onboarding.ts`, `lib/queries/onboarding.ts`, `lib/onboarding-gate.ts`
   - `client/packages/shared/src/styles/tokens.css` (colours, radii, lift, easing, `--dsk-gem-*`)
   - `routes/onboarding/__tests__/onboarding-page.test.tsx`, `docs/product-guide/home/onboarding.md`

## Scope
- **Front end only, unless you find a real gap.** Fields, validation, the `POST /onboarding/complete/` request, the route gate and the post-complete sequence (`setQueryData`, `checkAuth`, `fetchManifest`, invalidate, navigate home) all stay as they are. If something in the design needs data the API doesn't return (for example the user's first name isn't in the auth store), add it the way the repo already does things, and note it.
- Remove the old frame header (logo + Sign out), the `Stepper`, the page title/description and the Back/Continue footer. Delete the four step components or rework them into the new turn inputs, whichever is cleaner.
- Keep react-hook-form + zod as the single form state. Validate each turn with `trigger(step.fields)`. Every string goes through `useT`.

## How to work
1. **Audit first.** Write `docs/onboarding-nova-gap-audit.md` listing every item in the README (frame, progress bar, back button, each turn, each input state, review, build, ready, failure, keyboard, a11y, reduced motion). Mark each one matches, partial or missing, and keep it updated.
2. **Build it** with the repo's components and patterns: `@trenova/shared` `Button`, `UsStateAutocompleteField`, `timezoneGroupedChoices`, icons from `@trenova/shared/components/icons`, and tokens via Tailwind utilities or CSS variables. Where a shared component can't hit the design exactly (for example the borderless composer input or the tile/card buttons), write a small local component under `routes/onboarding/_components/`. Don't restyle the shared one globally.
3. **Typewriter**: a small hook (`useTypewriter(segments, {animate})`) with the exact cadence from the README. Typed turns never re-type. Reduced motion shows text instantly. Announce each full sentence once through an `aria-live="polite"` region, not per character.
4. **Build sequence tied to the real mutation.** Start `onboardingService.complete` when the build begins. Lines advance on the minimum cadence. The last line keeps spinning until the request resolves. On failure, follow the README's failure section (rewind to the first invalid step with the server message; "Try again" for non-field errors).
5. **Back/edit semantics** exactly as specified: the `cur`/`far` rules, Esc, hint after the first answer, review rows that click to edit, editing returns to where you were.
6. **Tests**: rewrite `onboarding-page.test.tsx` to cover:
   - prefilled name and browser timezone first
   - per-turn validation (empty name; bad ZIP; SCAC/DOT patterns)
   - skip for the IDs
   - operation and sample keyboard picks
   - edit from review returns to review
   - Esc/back walks one step and keeps values
   - finish calls `complete` with `toCompleteOnboardingRequest` output
   - server field error rewinds to that step
   - ready → navigate home

   Add unit tests for the step/next logic and the typewriter cadence (fake timers).
7. Update `docs/product-guide/home/onboarding.md` for the new flow (same frontmatter; tasks rewritten around Nova, Edit, Esc and the review lines).
8. Run type checks, lints and tests. Compare every state against the prototype in both themes and at narrow widths (≥360px). The column is fluid up to 640px; tiles wrap.

## Decisions already made (don't re-ask)
- Name: **Nova**, subtitle "Setup guide", with a gradient ring mark (`--dsk-gem-1/2/3`) that spins while typing.
- No header, no rail, no preview. The 2px gradient progress bar on the top edge stays.
- Back button is icon only. The arrow shows at rest; on hover it loops arrow ↔ Trenova logo (Desk's `sbkc`/`sbkl`).
- Timezone: the detected zone goes first and is pre-selected, with no badge or ribbon.
- Review rows have no "Edit" label; the whole row is clickable.
- Primary actions are ink (not blue). Blue is only for selection, focus and meters.
- Sign out is intentionally not on this screen.

## Definition of done
- Every item in the gap audit matches (or has a precise technical reason it doesn't).
- No mock data in shipped code; sample counts come from `SAMPLE_DATA_SET` + `config.freePlan.limits`.
- Tests, type checks and lints pass; the product guide is updated.
- `tweaks-panel.jsx` and the typing-speed control aren't shipped.

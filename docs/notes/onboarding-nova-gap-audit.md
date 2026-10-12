# Onboarding with Nova: gap audit

Each item in `design_handoff_onboarding/README.md`, checked against the shipped screen
(`client/apps/web/src/routes/onboarding/`). The screen was compared with the prototype
side by side at 1280px in light and dark, and at 375px.

**Matches** means the same as the prototype. **Deviates** means it differs on purpose, and the
reason is given.

## Frame

| Item | Status | Notes |
|---|---|---|
| Full-viewport canvas, no header, rail, stepper or preview | Matches | `OnboardingFrame` (logo and Sign out) and `Stepper` are removed. |
| 2px progress bar, four-stop gradient, 900ms settle | Matches | Value is `round(cur / 7 * 86) + 4` while answering, 92% while building or failed, 100% when ready (`onboardingProgress`). |
| Back button, 34×34 at 14/16, icon only | Matches | |
| Back button hover: arrow and logo loop (`sbkc`/`sbkl`) | Matches | |
| Back button on step 1 | Deviates | Hidden. No form comes before onboarding in this flow (signup, then verify email, then here), so there is nowhere to go back to. This is the README's own fallback. |
| Back button hidden once the build starts | Matches | |
| Column: 640px max, 24px gutters, top `clamp(48px, 16vh, 160px)`, bottom 45vh | Matches | |
| Scroller fade, 32px top and 48px bottom | Deviates | Same look, different technique. It is drawn with two canvas-coloured overlays instead of `mask-image`. A mask on the scroller left stale paint after the smooth scroll in Chromium's software renderer. The scroller also gets an opaque canvas background for the same reason. Under 720px the top fade is taller, so text clears the back button. |
| Thin scrollbar; smooth auto-scroll on height change (ResizeObserver) | Matches | |

## Turn anatomy

| Item | Status | Notes |
|---|---|---|
| Nova header: gradient ring mark, "Nova", "Setup guide" | Matches | Ring uses `--dsk-gem-1/2/3` and spins while typing. |
| "Typing" shimmer for 650ms | Matches | |
| Prose: 16px/1.65, `pre-line`, bold at 550 | Matches | |
| Typewriter cadence (12–30ms; +260 `.?!`; +110 `,—`; +200 newline) and blinking caret | Matches | `useTypewriter`, covered by unit tests with fake timers. |
| Input appears 180ms after the last character | Matches | |
| Typed lines never re-type (re-render, back or edit) | Matches | |
| Ask area `rise` animation | Matches | |
| Answer bubble: radius, lift, `send` animation, "Skip for now" muted | Matches | |
| Edit button: always shown on the latest answer, on hover for older ones | Matches | |

## Turns

| Turn | Status | Notes |
|---|---|---|
| Copy for all seven turns, with and without a first name | Matches | The first name is the first word of the auth user's `name`; no API change was needed. |
| Timezone line when the browser's zone is unknown | Matches | Nova drops the "Your browser is set to…" sentence instead of naming a zone it doesn't know. |
| 1 · Composer: float lift, focus ring, entry ring sweep, prefilled name, 34px send button | Matches | |
| 1 · Empty submit: shake plus "Your company name is required." | Matches | Max-length errors use the zod message. |
| 1 · Hint "↵ to answer · esc to go back" | Deviates | Reads "↵ to answer". Esc does nothing on step 1 (see Back button above). |
| 2 · Tiles: grid, local time, staggered rise, hover and selected states | Matches | |
| 2 · Browser zone first and pre-selected; keys 1–7; Enter confirms; 280ms hold | Matches | |
| 2 · Tile labels | Matches | "Eastern time", "Arizona" and so on, as drawn. Zones outside the US keep their `timezoneGroupedChoices` label. |
| 2 · "Other timezone…" | Added | The README's optional fallback. It opens the grouped `SelectField`, so organizations outside the US can still finish. |
| 3 · Address card: grid, placeholders, mono ZIP limited to 5 digits, Enter submits | Matches | |
| 3 · State picker | Matches | Uses the shared `UsStateAutocompleteField`, dressed as the card's inputs through `triggerClassName`. Its chevron is the shared component's own. |
| 3 · Footer errors ("ZIP code needs 5 digits." / "Fill in the highlighted fields.") | Matches | If the server returned a message for one of these fields, that message is shown instead. |
| 4 · IDs: SCAC uppercase letters (max 4), DOT digits (max 8), help text, "2–4 letters." | Matches | |
| 4 · Skip behaviour: ink "Skip for now" when empty; ghost "Skip" plus "Continue" otherwise | Matches | |
| 5 · Operation cards, module chips, keys 1–3, 420ms hold, others fade to 40% | Matches | Values are `asset`, `brokerage` and `both`. |
| 6 · Sample cards, dashed meter grid, limits note | Matches | Rows come from `SAMPLE_DATA_SET` and `config.freePlan.limits`, through `planMeterLabel` and `formatPlanMeterValue`. |
| 5/6 · Pre-selection | Matches | No card is selected until the turn has been answered once, as drawn. The form still holds the existing defaults. |
| 7 · Review: print reveal, two groups, staggered rows, no "Edit" label, row jumps to its turn | Matches | |
| 7 · Footer note, "Looks good, finish setup" with ⌘↵, ⌘/Ctrl+Enter | Matches | |

## Build, ready and failure

| Item | Status | Notes |
|---|---|---|
| "Finish setup" bubble, then Nova's build line | Matches | |
| Narrated lines: spinner, shimmer, check, elapsed time | Matches | |
| Tied to the real request | Matches | `onboardingService.complete` fires on Finish. Each line takes at least 560–1080ms, and the last line spins until the request resolves. |
| Ready card: success ring, drawn check, ripple, 22-particle burst, title and detail | Matches | |
| "Open Trenova" runs the existing post-complete sequence | Matches | `setQueryData` → `checkAuth` + `fetchManifest` → invalidate → `navigate("/")`. |
| Failure: danger dot on the current line, Nova's failure line | Matches | Not drawn in the prototype; built from its visual language. |
| Failure with a field error: rewind to that turn with the server's message | Matches | Rewinds 900ms after Nova finishes typing. The furthest step stays at the review, so a fixed answer goes straight back to it. |
| Failure without a field error: "Try again" | Matches | Any form-wide message from the server is shown above the button. |

## Interactions

| Item | Status | Notes |
|---|---|---|
| `cur`/`far` rules: next is `far > cur ? far : cur + 1` | Matches | `nextOnboardingStep`, unit tested. |
| Edit and review rows keep `far`; Back and Esc set `far = cur − 1` | Matches | |
| "Made a typo?" hint after the first answer only | Matches | |
| Number keys ignored while focus is in a text field | Matches | `numberKeyIndex` in `lib/dom.ts`. |
| Esc ignored when it comes from a popover rendered outside the screen | Matches | |

## Accessibility and motion

| Item | Status | Notes |
|---|---|---|
| Each full Nova line announced once (`aria-live="polite"`) | Matches | A visually hidden live region gets the line once it has finished typing. The prose is `aria-hidden` while it types. |
| Tiles and cards are buttons with `aria-pressed` | Matches | |
| Back has `aria-label="Back"` | Matches | |
| Focus moves to the first input of each new step | Matches | Applies to the text, address and ID turns. Tile, card and review turns don't move focus, because the prototype draws no focus ring there and the keys work from anywhere. |
| Reduced motion | Matches | Typed text shows at once, animations collapse, and the confetti is hidden. |

## Not shipped

- `tweaks-panel.jsx` and the typing-speed control. Typing speed is fixed at 1×.
- No mock data. Sample counts come from `SAMPLE_DATA_SET` and the public config.

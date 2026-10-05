# Handoff: Onboarding — guided setup with Nova

## Overview
Replaces the four-step welcome wizard at `/onboarding` (`client/apps/web/src/routes/onboarding/**`) with a full-screen, conversational setup. A guide named **Nova** types each question; the person answers inline; answers stack up as chat bubbles; a review card confirms everything; a short narrated build plays while the workspace is created; a ready card opens the app.

Nova is **not AI**. Every line is a fixed, translatable string. No model calls.

The data contract is unchanged: same fields, same zod schema (`onboardingFormSchema`), same `POST /onboarding/complete/` request (`CompleteOnboardingRequest`), same gate (`onboarding-gate.ts`). This is a front-end rewrite of `page.tsx` and `_components/*`.

## About the design files
`design/` holds **design references built in HTML/React/CSS**. They are prototypes that show the intended look and behaviour, not production code. Recreate them in `client/apps/web` with the codebase's real stack: React Router, react-hook-form + zod, TanStack Query, `useT` i18n, `@trenova/shared` components and `tokens.css`. All data in the prototype is mock data. Wire it to the existing onboarding service, auth store and public config.

Open `design/index.html` in a browser to click through. The Tweaks panel has theme (dark/light), typing speed, the user's first name, and Restart.

## Fidelity
**High-fidelity.** Final colours (from `tokens.css`), type, spacing, radii, motion and copy. Match it 1:1 in dark and light.

---

## Screen: Onboarding (single full-screen view)

### Frame
- Full viewport, `background: var(--canvas)` (dark: pure black; light: `oklch(0.985 0 0)`). No header, no sidebar, no step rail, no preview panel. The existing `OnboardingFrame` header (logo + Sign out) and the `Stepper` are **removed**.
- **Progress bar**: 2px tall, fixed to the top edge, full width. Track is `--border-subtle`. Fill is a linear gradient, left to right: `oklch(0.72 0.15 225)`, `oklch(0.62 0.2 255)`, `oklch(0.6 0.25 345)`, `oklch(0.72 0.2 45)`. Width transitions over 900ms on `--ease-settle`.
  - Value while answering: `round(currentStep / 7 * 86) + 4` percent.
  - 92% while building, 100% when ready.
- **Back button**: absolute, `top:14px; left:16px`, 34×34, radius `--radius-control` (6px). Icon only, no label. Hover background is `--surface-hover`. See *Back button* under Interactions.
- **Conversation column**: one scroll container filling the viewport. The column is `max-width: 640px`, centred, with 24px side padding. Top padding is `clamp(48px, 16vh, 160px)`. Bottom padding is `45vh`, so the active question sits mid-screen.
  - The scroller has a soft fade mask: transparent → opaque over the first 32px and the last 48px.
  - Thin scrollbar.
  - Auto-scrolls smoothly to the bottom whenever the content height changes (a ResizeObserver on the column).

### Turn anatomy (repeats per step)
Each turn is a `section` with 44px top margin (0 for the first). It contains:

1. **Nova header row**: 22px tall, `gap: 8px`, 12px, `--foreground-subtle`.
   - **Mark**: a 20px circle with `conic-gradient(from 200deg, --dsk-gem-1, --dsk-gem-2, --dsk-gem-3, --dsk-gem-1)` and a 4px inset hole filled with `--canvas` (a ring). It rotates (1.6s linear, infinite) while Nova is "typing".
   - "**Nova**" in 500 weight, `--dsk-fg2`, followed by "Setup guide".
2. **Typing placeholder**: before a line starts, the text "Typing" shows in 14px with a shimmer (a gradient text sweep, 1.8s linear, infinite) for `650ms / speed`.
3. **Message prose**:
   - Margin-top 10px. 16px / 1.65, letter-spacing −0.01em, `--dsk-fg2`.
   - `white-space: pre-line` (`\n\n` makes paragraphs). Bold spans are 550 weight, `--foreground`.
   - Characters are revealed one at a time, followed by a 2×18px caret that blinks at 1s steps. Per-character delay is `(12 + rand*18)ms / speed`, plus:
     - +260ms after `. ? !`
     - +110ms after `, —`
     - +200ms after a newline
   - 180ms after the last character, the input for this step appears.
   - Lines already typed render in full instantly. Never re-type on re-render, back, or edit.
4. **Ask area** (only on the current step, after typing finishes): margin-top 18px, enters with `rise` (520ms `--ease-settle`: translateY 14px, scale .985 → none).
5. **Answer bubble** (once answered):
   - Right-aligned, max-width 80%, padding 9px 14px.
   - Radius `8px 8px 2px 8px` (the tail at bottom-right). `background: --card`, `box-shadow: --lift-ring`. 14px `--foreground`.
   - Enters with `send` (460ms spring: from translateY 12px, scale .9; origin bottom-right).
   - Skipped answers show "Skip for now" in `--foreground-subtle`.
   - An **Edit** ghost button (pencil icon + "Edit", 26px tall, 12px) sits left of the bubble. It is always visible on the most recent answer and shows on hover for older ones (fade + 6px slide, 160/220ms).

### Steps (one turn each, in order)
Copy is exact. `{first}` is the signed-in user's first name; when it's unknown, use the variant without it. `{company}` is the name entered in step 1, rendered bold.

| # | id | Nova says | Input | Form fields |
|---|---|---|---|---|
| 1 | name | "Hi {first}, I'm Nova. I'll help you set up your Trenova workspace. It takes about a minute, and you can change anything later.\n\nTo start, what's your company called?" (no name: "Hi, I'm Nova. …") | Composer | `organization.name` |
| 2 | tz | "Got it, **{company}**. Which timezone should dispatch, appointments and reports use? Your browser is set to **{browser tz label}**." | Timezone tiles | `organization.timezone` |
| 3 | addr | "Where is **{company}** headquartered? This becomes the default origin on new shipments." | Address card | `addressLine1, city, stateId, postalCode` |
| 4 | ids | "Do you have a SCAC code or USDOT number? Both are optional — add them now or skip." | IDs card | `scacCode, dotNumber` |
| 5 | op | "How does **{company}** move freight? This decides which parts of Trenova lead." | Option cards | `operationType` |
| 6 | sample | "Want me to load sample data? Every screen will have something to show, and you can delete any of it at any time." | Option cards | `loadSampleData` |
| 7 | review | "That's everything, {first}. Please check everything below before I create your workspace. If anything's wrong, click the line to fix it." | Review card | — |

Answer bubble text for each step:
- **name**: the name.
- **tz**: the zone label.
- **addr**: `"{street}, {city}, {ST} {zip}"`.
- **ids**: `"SCAC XXXX · USDOT 1234567"` (whichever exist), or "Skip for now".
- **op**: the type label.
- **sample**: "Load sample data" or "Start empty".
- **review**: after Finish, the bubble reads "Finish setup".

#### 1 · Composer (name)
- Card: `--card` background, radius 8px, `box-shadow: --lift-float`, padding `6px 6px 6px 16px`, flex with an 8px gap.
- Focus adds a ring: `--lift-float, 0 0 0 3px var(--ring at ring-opacity)`.
- **Entry ring**: a 1.5px conic-gradient border (masked to the border only) that sweeps around once.
  - Gradient stops: `transparent 0–150°`, `oklch(.72 .15 225/.5) 190°`, `oklch(.62 .2 255) 225°`, `oklch(.5 .22 290) 260°`, `oklch(.6 .25 345) 290°`, `oklch(.72 .2 45) 320°`, `oklch(.85 .17 85) 342°`, `transparent 360°`.
  - It animates `--ang` 0→360° over 2.6s and fades out after 1.6s (1.2s fade). Same as the Desk composer.
- Input: 36px tall, 15px, no border, placeholder "Company name". Autofocus.
- **Prefilled** with `onboardingFormDefaults().organization.name`, which comes from signup.
- Send button: a 34px ink circle with an up-arrow. Disabled (opacity .18) while the input is empty. Hover scale 1.06, active .9 (spring).
- Hint under the composer, 11.5px subtle: "`↵` to answer · `esc` to go back" (keys use the `kbd` style).
- Submitting empty shakes the card (300ms ±4px) and replaces the hint with "Your company name is required." in `--danger-foreground`.
- Use zod messages for the max-length error.

#### 2 · Timezone tiles
- Grid `repeat(auto-fill, minmax(132px, 1fr))`, gap 8px.
- Tile:
  - padding 11px 12px, radius 8px, `--card`, `--lift-ring`.
  - Label 13px/500, then the current local time in that zone, 11.5px Geist Mono, subtle (e.g. "2:41 PM", via `Intl.DateTimeFormat`).
  - Entry is `rise`, staggered 40ms.
  - Hover: `--surface-hover` background with a stronger ring (`--foreground` at 16% in light, 22% in dark).
  - Selected: `box-shadow: 0 0 0 1px var(--brand)` and `background: var(--surface-selected)`.
- **The browser-detected zone is first and pre-selected.** There is no badge or ribbon.
- Order: the detected zone, then Eastern, Central, Mountain, Arizona, Pacific, Alaska, Hawaii.
- Click or keys `1`–`7` select; Enter confirms the current selection. The answer submits 280ms after selection, so the selection state is visible first.
- Source the zone values and labels from `timezoneGroupedChoices` (US group). If the browser zone isn't one of these but *is* in `timezoneGroupedChoices`, still put it first. The prototype has no "other timezone" picker; if the product needs non-US zones, add a quiet "Other timezone…" text button after the grid that opens the existing grouped `SelectField`. Note it in the audit.

#### 3 · Address card
- Card: `--card`, radius 8px, `--lift-ring`, padding 16px.
- Grid columns `1.3fr 1fr .7fr`, gap 12px. Street spans the full row; City, State and ZIP share the second row.
- Field label: 11.5px/500 `--foreground-muted`.
- Inputs:
  - 38px tall, radius 6px, `--field` background, 1px `--border`, 13.5px.
  - Focus: border = foreground at 45%, plus the 3px ring.
  - Error: border = danger at 70%, plus a shake.
- Placeholders: "1200 Industrial Pkwy", "Whitsett", "Select" (state), "27377".
- State uses the existing **`UsStateAutocompleteField`** (it stores `stateId` and keeps the label for display), styled to match.
- ZIP is digits only, max 5; `inputMode="numeric"`, mono.
- Footer row:
  - Left: hint "`↵` to continue", or an error line. Use "ZIP code needs 5 digits." when the ZIP is the only problem, otherwise "Fill in the highlighted fields." Prefer the zod messages from `onboardingFormSchema` per field if you show them inline.
  - Right: ink button "Continue →".
- Enter in any field submits. Validate with `trigger([...addressFields])`.

#### 4 · IDs card
- Same card style, two equal columns.
- **SCAC code** *optional* (the "optional" is a 400-weight faint suffix).
  - Mono, letters only, auto-uppercase, max 4, placeholder "RVFL".
  - Help text: "Your Standard Carrier Alpha Code."
  - Error: "2–4 letters."
- **USDOT number** *optional*.
  - Mono, digits only, max 8, placeholder "3812045".
  - Help text: "Issued by FMCSA."
- Footer:
  - Hint "`↵` to skip" when both are empty, "`↵` to continue" otherwise.
  - Buttons: when both are empty, only the ink "Skip for now →". When either has a value, a ghost "Skip" (clears both and continues) plus the ink "Continue →".

#### 5 · Operation type cards
- Stack with an 8px gap. Card:
  - padding 14px 16px, radius 8px, `--card`, `--lift-ring`. Grid: title + `kbd` number at the right.
  - Title 14px/550.
  - Description 12.5px/1.5 subtle — exact text from `OPERATION_TYPES`.
  - A row of feature chips below: 11px, padding 2px 8px, radius 6px, `--sunken` background with an inset 1px `--border-subtle`.
    - asset: Dispatch, Fleet, Hours of service, Driver pay
    - brokerage: Carrier sourcing, Tenders, Rate confirmations, Carrier settlements
    - both: all eight
- Entry is `rise`, staggered 60ms. Hover and selected match the timezone tiles.
- Click or `1`/`2`/`3` selects. The other cards drop to opacity .4, and the answer submits after 420ms.
- Values are **`asset` | `brokerage` | `both`** (the prototype's internal id `broker` is `brokerage`).

#### 6 · Sample data cards
Same component as step 5, with two options:
- **Load sample data**: "A few customers, locations, equipment and shipments, so every screen has something to show."
  - Below it, a dashed top border (`1px dashed --border`, 12px above and below), then a 3-column meter grid (gap 10px 18px). Each meter shows the label and "n of limit" (mono), over a 3px track with a `--brand` fill that grows over 900ms settle.
  - Rows come from `SAMPLE_DATA_SET` + `config.freePlan.limits` via `planMeterLabel` / `formatPlanMeterValue`. The prototype shows Customers 2/8, Locations 4/25, Workers 1/3, Tractors 1/3, Trailers 1/3, Shipments 2/12.
  - Then 11.5px subtle: "Sample records count toward the free demo's limits, the same as records you create. Deleting one frees its slot."
- **Start empty**: "A clean workspace. Add your own records from day one."
- If the org already has sample data loaded (`state.sampleDataLoaded`), keep today's default (`loadSampleData: false`), but there's no need to hide the option.

#### 7 · Review card
- Card: `--card`, radius 8px, `--lift-ring`. Clip-path reveal from top to bottom (`print`, 700ms settle).
- Two groups, separated by `1px dashed --border`:
  - **COMPANY PROFILE**: Company name, Timezone, Address, SCAC code, USDOT number.
  - **SETUP**: Operation type, Sample data.
- Group header: 10.5px Geist Mono, uppercase, +0.06em, subtle, padding 12px 16px 8px.
- Row:
  - A full-width button, grid `120px 1fr`, padding 8px 16px.
  - Key 12px subtle; value 13px foreground. Empty values show "—" in faint; SCAC/DOT values are mono.
  - Rows print in staggered (`printl`, 120ms + 45ms each).
  - Hover background is `--surface-hover`. **There is no "Edit" label.** Clicking the row jumps to that step.
- Footer:
  - Padding 12px 16px, top border `--border-subtle`, background `color-mix(sunken 55%, card)`.
  - Left: 11.5px subtle "You can change all of this in Organization settings."
  - Right: ink button "Looks good, finish setup" with a `⌘↵` kbd. ⌘/Ctrl+Enter also finishes.

### Build + ready (after Finish)
1. The answer bubble "Finish setup" appears.
2. A new Nova turn: "Creating the workspace for **{company}**. This only takes a moment."
3. Narrated lines appear one at a time (`nl` 420ms settle):
   - Each line is 13.5px/24px with a 20px status column. The current line has a 12px spinner and shimmer text. Done lines have a green check (`--success`) and the elapsed time at the right (11px mono faint).
   - Lines:
     - "Creating **{company}**"
     - "Setting the clock to **{tz}**"
     - "Saving **{City, ST}** as headquarters"
     - "Turning on **{n} modules** for {op type lowercase}"
     - Then either "Loading **11 sample records**" (the sum of `SAMPLE_DATA_SET` quantities) or "Leaving records empty".
     - "Making you the owner"
   - **Tie these to the real request.** Fire `onboardingService.complete` when the build starts.
     - Advance lines on a minimum cadence of 560–1080ms each.
     - Hold the last line spinning until the mutation resolves; never finish the animation before the server does.
     - If the server is faster, keep the cadence.
4. **Ready card** (rises in with a spring, 620ms):
   - `--card` background, radius 8px, `box-shadow: 0 0 0 1px var(--success)`.
   - A 34px success circle with a check drawn in (`stroke-dasharray` 22, 420ms after 260ms), a ripple ring, and a 22-particle confetti burst. Particle colours: gem-1, the indigo/brand blue, gem-3, warning amber, success.
   - Title: "You're all set, {first}" (no first name: "{company} is ready").
   - Sub: "{company} is live. Sample data is loaded and ready to explore." or "{company} is live and ready for your first records."
   - Ink button "Open Trenova →" runs the existing post-complete sequence: `setQueryData` status completed, `checkAuth()`, `fetchManifest()`, invalidate non-onboarding queries, `navigate("/", {replace:true})`.
5. **Failure**: if `complete` rejects:
   - Stop the spinner on the current line and mark it with a danger dot.
   - Nova types: "Something went wrong while I was setting things up. Nothing was lost — let's fix it."
   - Then run the existing `showFirstInvalidStep()` logic, but as a **rewind**: jump the conversation back to the first step holding a field error, with the server's field message shown in that step's error slot.
   - For non-field errors, show a retry button "Try again" under Nova's line.
   - This state isn't drawn in the prototype; match the visual language: danger foreground text, same button styles.

---

## Interactions & behaviour

### Moving between steps
- `submit(patch)` writes the form values. The next step is `far > cur ? far : cur + 1`, where `far` is the furthest step reached.
  - So **editing from the review (or from any later point) returns straight to where you were**.
- Validate each step with `trigger(step.fields)` before advancing.
- **Edit on a bubble** or **click on a review row** sets `cur = thatStep` and keeps `far`. Later turns disappear and re-render (non-animated) after re-submitting.
- **Back** (button or `Esc`) sets `cur = cur − 1` *and* `far = cur − 1`, so going back walks forward step by step again. Values are kept and prefilled.
- After the first answer only, a hint appears right-aligned under the answer (11.5px faint, fades in after 600ms): "Made a typo? Click Edit, or press `esc` to go back."
- Every widget prefills from the current form values.

### Back button
- At rest it shows a **back arrow** (15px, `--foreground`).
- On hover it plays a continuous 3.6s loop with `cubic-bezier(.3,1.3,.5,1)`: the arrow slides out to the left (translateX −130% + fade), the **Trenova logo** (20px, the `logo.webp` already used by the page) slides in from the right and holds, then slides out left while the arrow comes back in from the right.
- Keyframes `sbkc` (on the arrow) and `sbkl` (on the logo) are the same ones as Desk's "Back to Trenova" button, in `design/onboarding.css`. It stops on mouse-out.
- On step 1, Back goes to the form that precedes onboarding (signup/account details). If there's no such route in this flow, hide the button on step 1.
- Hidden (opacity 0, no pointer events) once the build starts.

### Keyboard
- `↵` submits text and form steps.
- `1`–`7` / `1`–`3` / `1`–`2` pick tiles and cards; `↵` confirms the timezone.
- `⌘/Ctrl+↵` finishes on the review.
- `Esc` goes back.
- Ignore number keys while focus is in an input.

### Motion tokens
- `--ease-swift` `cubic-bezier(.2,.8,.2,1)`, `--ease-settle` `cubic-bezier(.16,1,.3,1)`, `--ease-spring` `cubic-bezier(.34,1.4,.64,1)` (already in `tokens.css`).
- Keyframes, all in `design/onboarding.css`: `rise`, `send`, `nl`, `print`, `printl`, `fill`, `shimmer`, `blink`, `confirm`, `ripple`, `draw`, `spark`, `shake`, `grow`, `ang`, `sbkc`, `sbkl`.
- Respect `prefers-reduced-motion`: collapse durations, render typed text instantly, skip confetti.

### Accessibility
- Each turn is a live region (`aria-live="polite"` on the conversation). Announce Nova's full sentence once, not per character.
- Tiles and cards are real buttons with `aria-pressed`.
- The back button has `aria-label="Back"`.
- Focus moves to the first input of each new step.

## State
- **Form**: react-hook-form with `onboardingFormSchema` and `onboardingFormDefaults({state, organizationName, fallbackTimezone: browserTimezone()})`. The values are the same as today.
- **UI state**:
  - `cur` (step index 0–6)
  - `far` (furthest step reached)
  - `seen: Set<number>` (turns already typed)
  - `phase: 'setup' | 'building' | 'ready' | 'failed'`
  - build line index and elapsed times
- `firstName` from the auth store user (whatever field holds the given name). Fall back to the variants without a name.
- Typing speed is a constant 1× in production (the Tweaks slider is for the prototype only).
- The state label for review and bubbles comes from `UsStateAutocompleteField` `onOptionChange` (as `stateLabel` is today), abbreviated to the USPS code in the bubble and the build line.

## Design tokens (map to `client/packages/shared/src/styles/tokens.css`)

| Prototype var | tokens.css |
|---|---|
| `--canvas` | `--canvas` |
| `--card` | `--card` |
| `--raised` | `--raised` |
| `--sunken` | `--sunken` |
| `--field` | `--field` |
| `--hover` | `--surface-hover` |
| `--fg` | `--foreground` |
| `--fg2` | `--dsk-fg2` |
| `--muted` | `--foreground-muted` |
| `--subtle` | `--foreground-subtle` (dark) / `--foreground-muted` (light) |
| `--faint` | `--dsk-faint` |
| `--b-sub` | `--border-subtle` |
| `--b` | `--border` |
| `--b-strong` | `--border-strong` |
| `--ink` | `--ink` |
| `--ink-fg` | `--ink-foreground` |
| `--brand` | `--brand` |
| `--selected` | `--surface-selected` |
| `--ring` | `--ring` at `--ring-opacity` |
| `--success` | `--success` |
| `--danger` | `--danger` |
| `--lift` | `--lift-ring` |
| `--float` | `--lift-float` |
| `--r-c` | `--radius-control` (6px) |
| `--r-s` | `--radius-surface` (8px) |
| Nova ring | `--dsk-gem-1/2/3` |

- **Type**: Geist (UI), Geist Mono (times, SCAC/DOT, kbd, kickers). Sizes used: 10, 10.5, 11, 11.5, 12, 12.5, 13, 13.5, 14, 15, 16px.
- **Elevation**: everything in the page uses a hairline (`--lift-ring`). Only the composer floats (`--lift-float`).
- **Colour use**: brand blue only for the selected state, meters, focus and selection. Green only for success. The rainbow appears only on the composer entry ring and the progress bar.

## Assets
- Trenova logo: use the repo's existing `@/assets/logo.webp` (`design/logo.png` is the same mark).
- Icons (arrow up, arrow left/right, check, pencil): inline SVG in `design/parts.jsx` → `Ic`. Use `@trenova/shared/components/icons` equivalents (`ArrowLeftIcon`, `ArrowRightIcon`, …).

## Files (in `design/`)
- `index.html`: entry point.
- `app.jsx`: step list, copy (`say`), answer text, state machine (`cur`/`far`/`seen`/`phase`), review card, build sequence, ready card, back button, tweaks.
- `parts.jsx`: `Typed` (typewriter), `TextAsk`, `TzAsk`, `AddrAsk`, `IdsAsk`, `Choice`, `Burst`, data (timezones, states, operation types, sample set), icons.
- `onboarding.css`: every value and keyframe.
- `tweaks-panel.jsx`: prototype-only controls; don't ship.

Also update `docs/product-guide/home/onboarding.md` to match the new flow, and rewrite `routes/onboarding/__tests__/onboarding-page.test.tsx`.

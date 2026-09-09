# Handoff: Trenova Auth Flow Redesign (Login → Organization → Role)

## Overview

A complete redesign of the Trenova web auth experience: a two-pane enterprise sign-in that walks the user through **Login → Select Organization → Select Active Roles**, then hands off to the dashboard. The right pane is a single card that morphs between steps; the left pane is an ambient "network" panel where a **session credential assembles itself** as the user progresses.

Target: `client/apps/web/src/routes/auth/` in `emoss08/Trenova` (branch `master`).

## About the Design Files

The files in `prototype/` are **design references written in HTML/JSX-in-browser (Babel standalone)** — they demonstrate the intended look, motion, and behavior. They are **not production code to copy**. Recreate them in the existing web app's environment: React 19 + React Router + Tailwind v4 + the `@trenova/shared` UI primitives (`Button`, `Card`, `Tabs`, `Badge`, `Form`, `InputField`, `SensitiveField`), `motion/react` for animation, and the token set in `client/packages/shared/src/styles/app.css`.

The prototype's `tweaks-panel.jsx` is a **prototyping-only** control panel. Do not port it. Its toggles map to real conditions:

| Tweak | Real condition |
| --- | --- |
| Side panel | viewport ≥ 900px (always on above the breakpoint) |
| Office / Driver tabs | `!tenantMetadata && authStep === "LOGIN"` (existing `showAudienceToggle` logic) |
| SSO providers | `providers.length > 0` from `authService.listProviders(orgSlug)` |
| Organizations 1–3 | `getUserOrganizations()` length; ≤1 skips the organization step |
| Theme | existing dark/light theme provider |
| Motion | `prefers-reduced-motion` |

## Fidelity

**High fidelity.** Colors, typography, spacing, radii, timings and easing below are final. Match them exactly, expressed through existing Tailwind tokens where they already exist.

---

## Design Tokens

All colors are **already in the codebase** at `client/packages/shared/src/styles/app.css`. The design is intentionally **monochrome** — the rainbow logo is the only color on the screen. No accent bars, no brand-blue buttons.

### Mapped to existing tokens (dark)

| Prototype var | Value | Existing token |
| --- | --- | --- |
| `--bg` | `oklch(0 0 0)` | `--background` |
| `--card` | `oklch(0.14 0 0)` | `--card` |
| `--card-2` | `oklch(0.18 0 0)` | `--popover` |
| `--fg` | `oklch(1 0 0)` | `--foreground` / `--primary` |
| `--muted-fg` | `oklch(0.72 0 0)` | `--muted-foreground` |
| `--border` | `oklch(0.26 0 0)` | `--border` |
| `--input` | `oklch(0.32 0 0)` | `--input` |
| `--danger` | `oklch(0.68 0.19 27)` | `--destructive` (lightened for AA on black) |

### New values to add

| Var | Dark | Light | Use |
| --- | --- | --- | --- |
| `--panel` | `oklch(0.055 0 0)` | `oklch(0.965 0 0)` | left panel ground |
| `--border-2` | `oklch(0.20 0 0)` | `oklch(0.94 0 0)` | inner hairlines (receipt rows, option borders) |
| `--field` | `oklch(0.11 0 0)` | `oklch(0.99 0 0)` | input + option row fill |
| `--subtle-fg` | `oklch(0.68 0 0)` | `oklch(0.52 0 0)` | meta text — **verified ≥4.5:1**, do not darken |
| `--ring` | `oklch(1 0 0 / 0.22)` | `oklch(0.145 0 0 / 0.14)` | focus ring |

Light theme: `--bg oklch(0.985 0 0)`, `--card oklch(1 0 0)`, `--fg oklch(0.145 0 0)`, `--muted-fg oklch(0.48 0 0)`, `--border oklch(0.91 0 0)`.

### Typography

**Geist** (UI) + **Geist Mono** (numerics, IDs, keyboard hints, receipt keys). Replaces Inter. Add `@fontsource-variable/geist` and `@fontsource/geist-mono`, or keep `Geist Mono` if already present via `--font-table`.

Critical rule from review: **no wide letter-spacing anywhere.** No `tracking-[0.2em]` uppercase labels. Mono labels sit at `letter-spacing: 0`.

| Role | Size | Weight | Tracking |
| --- | --- | --- | --- |
| body / base | 13.5px | 400 | −0.006em |
| panel headline `.pitch` | 31px | 500 | −0.038em, `line-height:1.14`, `max-width:19ch`, `text-wrap:balance` |
| card title | 17px | 600 | −0.028em |
| card subtitle | 12.5px | 400 | — (`--muted-fg`) |
| option name | 13px | 550 | −0.005em |
| option meta | 11.5px | 400 | — (`--subtle-fg`, single-line ellipsis) |
| metric number | 24px | 500 | −0.035em, `tabular-nums` |
| mono captions (`01 / 03`, receipt keys, footer, lanes) | 10.5–11px | 400 | 0 |
| field label | 11.5px | 500 | — |
| button label | 13px | 550 | — |

### Geometry & motion

- Radii: card `14px`, buttons/inputs `9px`, option rows `10px`, receipt `12px`, avatar `8px`, pills `999px`.
- Card shadow: `0 1px 2px oklch(0 0 0/.5), 0 30px 70px -30px oklch(0 0 0/.9)`.
- Easing: **`cubic-bezier(.2,.8,.2,1)`** everywhere.
- Durations: hover/color `140–180ms`; step enter `300ms`; card height morph `320ms`; segmented knob `320ms`; receipt row fill `420ms`; aura drift `900ms`.
- Step delay simulation in the prototype (`700–900ms`) stands in for the real network call — drive it off mutation `isPending`.

---

## Layout

```
.shell  height:100vh; display:grid;
        grid-template-columns: 1fr clamp(440px, 42%, 560px);
        overflow:hidden
        @media (max-width:900px) → single column, aside display:none

aside   background:var(--panel); border-right:1px solid var(--border-2);
        padding:40px; flex column; justify-content:space-between; overflow:hidden

main    display:grid; grid-template-columns:minmax(0,1fr);
        align-items:center; justify-items:center;
        padding:40px 24px; overflow:auto; scrollbar-gutter:stable

.col    width:min(400px,100%); flex column; gap:16px
```

> The `minmax(0,1fr)` track + `min(400px,100%)` + `scrollbar-gutter:stable` combination is load-bearing — an auto-sized track caused a horizontal scrollbar when the vertical scrollbar appeared on the tallest step. Don't simplify it back to `place-items:center`.

---

## Screen 1 — Left panel (persistent, ≥900px)

**Purpose:** ambient enterprise context + visible proof the session is being assembled.

Top → bottom, `justify-content: space-between`:

1. **Brand lockup** — 24px logo · "Trenova" (14px/600/−0.02em) · `Enterprise` in mono 10.5px `--subtle-fg` behind a `1px solid var(--border)` left divider with `padding-left:10px`.

2. **Middle group** (`flex column; gap:24px`):
   - **Headline:** "Sign in once. The network never stopped moving."
   - **Metrics** (`flex; gap:40px; white-space:nowrap`): `12,480` / "loads in motion" — animates by ±2–5 every 3.2s with a 420ms cubic ease-out count; `98.6%` / "on-time this week".
   - **Lanes** — two rows of pill chips (`5px 10px`, `1px solid var(--border-2)`, mono 10.5px) reading `LAX→PHX ● in transit`. Row 1 drifts left over 34s linear infinite, row 2 reversed over 44s. Content duplicated 2× and translated `-50%`. Masked: `linear-gradient(to right, transparent, #000 8%, #000 82%, transparent)`.
   - **Session credential receipt** (see below).
   - Metrics + lanes are hidden below `620px` viewport height.

3. **Footer** — mono 10.5px: pulsing dot + "Network operational" · `us-west-2` · `v4.12.0`. Dot: 5px circle, opacity `.35 → 1` over 2.6s ease-in-out.

**Ambient layers** (absolute, `pointer-events:none`):
- `.weave` — 64px grid of 5%-foreground hairlines, radial-masked at `18% 28%`.
- `.scan` — 140px-tall soft light band translating from `-160px` to `110vh` over 13s linear infinite. Suppressed under reduced motion.
- `.aura` — 120%-wide radial bloom that **glides as the step changes**: login `0,0` → org `translate(18%,26%)` → role `translate(30%,48%)` → done `translate(24%,40%) scale(1.15)`, 900ms.

### The receipt (signature element)

`max-width:392px`, `1px solid var(--border-2)`, radius 12px, `background: color-mix(in oklch, var(--card) 60%, transparent)`, `backdrop-filter: blur(6px)`. Header row (`1px dashed` divider): "Credential" · state ("Assembling" → "Issued"). Then four `grid-template-columns:78px 1fr` rows separated by dashed hairlines:

| Key | Fills at step | Value |
| --- | --- | --- |
| Identity | after login | `admin@trenova.app` |
| Workspace | after org select | `Trenova Logistics` |
| Roles | after activation | comma-joined role names |
| Session | after activation | `sx_1f4c9ab2` |

- **Pending row:** empty 15px box with a 52px hairline sweeping left→right over 2.6s infinite (`transparent → 45% foreground → transparent`).
- **Filled row:** `opacity 0→1, translateY(5px)→0, blur(2px)→0` over 420ms.
- **On done:** an `AUTHORIZED` stamp — mono 10.5px, `letter-spacing:.04em`, uppercase, `1px solid var(--fg)`, radius 4px, `padding:3px 7px`, `rotate(-4deg)`, entering `scale(1.5)→1` over 380ms, pinned bottom-right.

---

## Screen 2 — Step 1: Sign in

Card `400px`, `--card` bg, `1px solid --border`, radius 14px.

- **Crumbs row** (mono 11px `--subtle-fg`, `space-between`, `margin-bottom:14px`): `01 / 03` (or `01 / 02` when the org step is skipped) · `Secure sign-in`.
- **Title** "Welcome back" · **subtitle** "Don't have an account yet? [Create an account]".
- **Audience segmented control** (replaces the underline tabs). Track: `--field` bg, `1px solid --border-2`, radius 9px, `padding:3px`, 2 equal columns. Knob: `--card-2` + `1px solid --border`, radius 6px, slides `transform`/`width` over 320ms. Items: Building icon + "Office", Truck icon + "Driver", 12.5px/500, inactive `--subtle-fg` → active `--fg`.
- **SSO block** (only when providers exist): full-width 38px outline buttons with Entra / Okta marks, then an "or" rule of hairlines.
- **Fields:** "Email address *" and "Password *" (asterisk in `--danger`). Control: `--field` bg, `1px solid --border`, radius 9px, input padding `10px 12px`. Focus: border → `color-mix(in oklch, var(--fg) 50%, transparent)` + `0 0 0 3px var(--ring)`. Password has a `show`/`hide` text button (11.5px/500) and a "Forgot?" link right-aligned on the label row.
- **Error:** `.bad` sets a danger border and runs a 260ms `translateX(±3px)` shake; message renders below in 11.5px danger.
- **Submit:** full-width 40px, `--fg` fill / `--bg` text, radius 9px. Busy → spinner + "Verifying credentials".
- **Driver tab** swaps the form for a short line of copy and a "Continue to Dash →" button (routes to `/dash/login`).
- Below the card: legal line, 11.5px, centered, `text-wrap:balance`.

**Validation:** email must contain `@`; password ≥6 chars. Errors clear on any keystroke. Keep the real `loginRequestSchema` + zodResolver.

## Screen 3 — Step 2: Select organization

- Crumbs: `02 / 03` · `N available`.
- Title "Select organization" · subtitle "Choose the workspace for this session."
- **Option rows** (`gap:8px`): 32px mono avatar tile (3-letter org code) · name 13px/550 + location 11.5px `--subtle-fg` · `Current` pill · `⌘1`-style shortcut badge that fades in on hover/focus · 18px circular check mark on the right.
  - Rest: `--field` bg, `1px solid --border-2`. Hover: `--border` + 4% foreground wash. Selected: border `color-mix(fg 60%)`, bg 7% foreground wash, mark fills `--fg` with the check scaling `0.5→1` over 240ms. Active press: `scale(.994)`.
- Continue button; busy → "Opening workspace".
- **Tray:** "← Back" on the left, `↑↓ move  ⌘↵ continue` hints on the right.
- Skipped entirely when the user has ≤1 organization (go straight to roles).

## Screen 4 — Step 3: Select active roles

Same row anatomy, **multi-select**, shield icon instead of the code tile.

- Crumbs: `03 / 03` · **live permission tally** that counts up/down over 420ms as roles toggle (`214 / 68 / 41` per role).
- Subtitle: "Scope this session at {org}. You can switch later without signing out."
- Rows carry a `System` / `Custom` pill.
- Button label is dynamic: `Activate N roles` → disabled "Select at least one role" when empty; busy → "Issuing credential".
- Tray hints: `⌘1–3 toggle  ⌘↵ activate`.

## Screen 5 — Handoff state

Centered in the card, `padding:38px 24px`, `gap:16px`:
- 44px ring with a checkmark drawn via `stroke-dasharray:22` over 460ms (140ms delay), plus a halo ring expanding `0.8→1.25` and fading over 900ms.
- "Entering Trenova Logistics" (14px/550) and "1 role · 214 permissions" (mono 11.5px `--subtle-fg`).
- A 2px determinate bar filling 0→100% over 1.6s, then navigate to `/`.

---

## Interactions & Behavior

- **Card height morph.** One card; content swaps per step. A `ResizeObserver` on the inner content sets an explicit `height` on the wrapper, transitioned at 320ms. New content enters with `opacity 0→1, translateY(8px)→0` over 300ms. In the real app, `motion/react`'s `AnimatePresence mode="wait"` + a height-animated container is equivalent.
- **Border beam on submit.** While a step is pending, a conic gradient sweeps the card's 1px border:
  ```css
  @property --beam { syntax:"<angle>"; initial-value:0deg; inherits:false }
  .beamring{position:absolute;inset:-1px;border-radius:15px;padding:1px;
    background:conic-gradient(from var(--beam),transparent 0deg,var(--fg) 26deg,transparent 64deg);
    mask:linear-gradient(#000 0 0) content-box,linear-gradient(#000 0 0);
    mask-composite:exclude;opacity:0;transition:opacity 240ms}
  .card.busy .beamring{opacity:.85;animation:spinbeam 1.3s linear infinite}
  @keyframes spinbeam{to{--beam:360deg}}
  ```
  The repo already declares `--border-beam-angle` + `border-beam-spin` in `app.css` — reuse those names.
- **Keyboard.** `↑ ↓` moves focus between option rows (wrapping); `⌘/Ctrl + 1…3` selects an org or toggles a role; `⌘/Ctrl + ↵` advances. Listener is scoped to the org/role steps. Shortcut badges only appear on hover/focus.
- **Reduced motion.** A global `@media (prefers-reduced-motion: reduce)` collapses all durations to `0.01ms`; the scanline is not rendered.
- **Responsive.** <900px: panel hidden, card centers with a small logo+wordmark above it. <620px height: metrics and lanes hidden.

## State

```ts
step: "login" | "org" | "role" | "done"
audience: "office" | "driver"
email, password, revealPassword, error
busy            // mutation isPending
orgId           // default = isCurrent ?? first
roleIds: string[]   // default = system role
sessionId       // from the auth response, displayed on the receipt
```

Transitions: `login --(orgs>1)--> org --> role --> done --> navigate("/")`; `login --(orgs<=1)--> role`. Back from `org` resets to login; back from `role` returns to `org` (or login when skipped). Wire to the existing services — `authService.login`, `apiService.userService.getUserOrganizations()`, `switchOrganization`, `usePermissionStore.fetchManifest()`.

## Assets

- `logo.webp` — from `client/apps/web/src/assets/logo.webp` (unchanged; the only color in the design).
- `entra.svg`, `okta.svg` — from `client/apps/web/src/assets/integrations/logos/`. The Okta dark mark is inverted in dark mode (`filter: invert(1) brightness(1.7)`); use the existing light/dark pair instead.
- Icons (building, truck, shield, check, arrow, chevron) — draw from **lucide-react**, which the app already uses: `BuildingIcon`, `TruckIcon`, `ShieldIcon`, `CheckIcon`, `ArrowRightIcon`, `ArrowLeftIcon`.

## Files

```
prototype/index.html   all CSS: tokens, layout, every component style, keyframes
prototype/app.jsx      flow, state, keyboard handling, receipt, lanes, metrics
prototype/tweaks-panel.jsx   prototyping harness — DO NOT PORT
prototype/logo.webp, entra.svg, okta.svg
screenshots/01-sign-in.png, 02-organization.png, 03-roles.png, 04-authorized.png
```

Screenshots were captured at 924×540, below the 620px height threshold — the panel's metrics and lane chips are hidden in them by design. Run the prototype in a taller window to see those.

Open `prototype/index.html` in a browser to interact with the full flow.

## Upstream files being replaced

| Prototype screen | Repo file |
| --- | --- |
| Shell + left panel | `client/apps/web/src/routes/auth/page.tsx` |
| Card, step orchestration, audience toggle | `client/apps/web/src/routes/auth/_components/auth-form.tsx` |
| Step 1 | `client/apps/web/src/routes/auth/_components/login-form.tsx` |
| Step 2 | `client/apps/web/src/routes/auth/_components/organization-selection.tsx` |
| Step 3 | **new** — `role-selection.tsx` (no upstream component exists yet) |
| Tokens | `client/packages/shared/src/styles/app.css` |

# Handoff: Free-demo banner

## Overview
A full-width alert bar pinned to the **very top of the Trenova TMS app, above the app header** (above the sidebar toggle / back-forward / breadcrumb row). It tells demo workspaces how much of the free demo remains and rotates between two limits:

1. `Free demo` · **30** days left. After that the workspace becomes read-only.
2. `Free demo` · **250** shipments left. After that the workspace becomes read-only.

The numbers are live values (days remaining, shipments remaining) — not hardcoded.

## About the design files
`Demo Banner.html` is a **design reference built in HTML** (vanilla HTML/CSS/JS), not production code. Recreate it in the Trenova web client (`client/apps/web`, React + Tailwind + shared tokens in `client/packages/shared/src/styles`) using the codebase's existing patterns. The header and "Plan & usage" page shown under the banner are context only — don't rebuild them.

## Fidelity
**High-fidelity.** Match colors, timing, easing and motion closely.

## Layout
- Container: full width, `min-height: 36px`, `padding: 0 16px`, flex row, `align-items: center`, `gap: 10px`, `overflow: hidden`, `isolation: isolate`, `position: relative`. Rendered as the first child of the app shell, before the header.
- Left to right: **chip** → **message** → flex spacer → **CTA link**.
- Message wraps on narrow widths (`flex-wrap: wrap`, `column-gap: .3em`).

## Components

### Background
- Base: `--panel` (white / `oklch(1 0 0)`).
- Tint layer (::before, z −2): `linear-gradient(110deg, oklch(0.62 0.2 255/.09), oklch(0.55 0.22 290/.07) 35%, oklch(0.62 0.24 345/.06) 65%, oklch(0.72 0.2 45/.07))`.
- Bottom hairline (::after): 1px, `linear-gradient(90deg, c1,c2,c3,c4,c5,c1)`, `background-size: 200% 100%`, opacity .55, animates `background-position → -200% 0` over 8s linear infinite.

### Floating icons (background)
- 22 small line icons (truck, package, clock, map pin, route, calendar — lucide-style, stroke 2) in a `pointer-events:none` layer at z −1.
- Each: size random 10–15px, vertical offset random 3–21px, color cycles c1…c5, opacity random .35–.6.
- Motion: drift right→left across the full width (`translateX(100vw+20px) → -30px`), duration random 22–40s linear infinite, negative random delay so they start spread out. Inner SVG bobs: `translateY(-3px) rotate(-8deg) ↔ translateY(3px) rotate(8deg)`, 1.6–3s ease-in-out alternate.
- Mask so icons never sit behind the text: `mask-image: linear-gradient(90deg, transparent 0, transparent 38%, #000 70%, #000 92%, transparent)`.
- On each message switch, drift speeds up 4× for 1s (`.pulse` class), then eases back.
- This style matches the Desk model picker's "Auto" row (logo rain + conic ring) — reuse those primitives if they already exist in the codebase.

### "Free demo" chip
- Height 22px, pill (`border-radius: 999px`), padding `0 8px 0 3px`, gap 6px, bg panel, Inter 11.5px / 600, color `--fg`.
- Animated border: 1px conic-gradient ring `conic-gradient(from var(--ang), c1,c2,c3,c4,c5,c1)` with `@property --ang` rotating 0→360deg over 6s linear infinite (ring = ::before at inset −1px, panel fill = ::after at inset 0).
- Icon tile: 16px circle, `linear-gradient(135deg, c2, c3 50%, c4)`, white 10px icon (stroke 2.4). Clock for "days", package for "shipments". Swap: outgoing `opacity 0, scale(.3) rotate(-120deg)`; incoming back to identity; `transform .6s var(--spring)`, `opacity .3s`.
- Ring pulse on switch: 1.5px c3 ring at inset −2px, `scale 1 → 1.9`, `opacity .8 → 0`, 1s settle.

### Message
- Inter 12.5px, color `--fg2` `oklch(0.32 0 0)`.
- Rotating phrase (`[number] days left.` / `[number] shipments left.`) sits in a fixed-height (1.5em) wrapper whose **width animates** to the incoming phrase width (`.55s settle`) so the trailing text glides.
- Static trailing text: "After that the workspace becomes read-only."
- Number: Geist Mono 12.5px / 600, tabular-nums, `--fg`. **Odometer**: each digit is a 0–9 column in a 1.5em `overflow:hidden` window (vertical fade mask 25%/75%); it rolls to its value with `transform 1.1s cubic-bezier(.2,.9,.25,1.08)`, staggered right-to-left 90ms per digit, base delay 150ms. Columns reset to 0 when the phrase leaves so they roll again next time.
- Unit word ("days"/"shipments"): 600 weight, `--fg`; on enter gets a one-time gradient shine (`linear-gradient(90deg, fg 0 35%, c2 45%, c4 55%, fg 65% 100%)`, size 300%, `background-clip:text`, position 100%→0 over 1.6s, 0.5s delay).

### CTA
- "Plan & usage" + right arrow (12px). Height 24px, padding `0 10px`, radius 7px, Inter 12px/500, `--fg`, bg `oklch(1 0 0/.7)`, `box-shadow: 0 0 0 1px oklch(0.55 0.22 290/.18)`.
- Hover: bg panel, `box-shadow: 0 0 0 1px oklch(0.55 0.22 290/.4), 0 2px 8px oklch(0.55 0.22 290/.15)`, arrow `translateX(2px)` (.3s spring).
- Navigates to Settings → Plan & usage.

## Interactions & behavior
- Rotation: show message 1 on mount (after fonts load), then switch every **5.2s**, looping.
- Switch sequence: outgoing words animate out staggered 40ms (`translateY(-70%)`, blur 4px, opacity 0, .35s ease-in); after 260ms incoming words animate in staggered 60ms (`translateY(70%)`, blur 4px → none, .6s settle); width animates; odometer rolls; icon swaps; ring pulses; icons speed up.
- **Hover on the banner pauses rotation**; mouseleave resumes.
- `role="status"`, `aria-live="polite"`; odometer has `aria-label` = full number.
- `prefers-reduced-motion: reduce`: no drift/bob/ring spin/hairline flow, near-instant word swaps, no digit roll.
- Not dismissible (persistent while the workspace is on the demo).

## State
- `daysLeft: number`, `shipmentsLeft: number` — from the org's plan/usage data (wherever Plan & usage reads them).
- `index` (0|1), `paused` (hover).
- Show only when the org is on the free demo. Edge cases to decide with product: singular ("1 day left", "1 shipment left"); if one limit is already reached, show only the other; at 0 the workspace is read-only (likely a different banner).

## Design tokens
- `--fg oklch(0.16 0 0)`, `--fg2 oklch(0.32 0 0)`, `--panel oklch(1 0 0)`, `--border oklch(0.9 0 0)`
- Accent ramp: `c1 oklch(0.75 0.14 220)`, `c2 oklch(0.62 0.2 255)`, `c3 oklch(0.55 0.22 290)`, `c4 oklch(0.62 0.24 345)`, `c5 oklch(0.72 0.2 45)`
- `--spring cubic-bezier(.2,.9,.25,1.25)`, `--settle cubic-bezier(.2,.8,.2,1)`
- Fonts: Inter (UI), Geist Mono (numbers)
- Dark mode: swap panel/fg to the existing dark tokens; keep the accent ramp, raise tint alphas slightly.

## Assets
Icons are simple lucide-style line icons — use `lucide-react` (Clock, Package, Truck, MapPin, Route, Calendar, ArrowRight) as the codebase already does.

## Files
- `Demo Banner.html` — the reference prototype (CSS + JS inline).

---

## Prompt for Claude Code

```
Implement the free-demo banner described in design_handoff_demo_banner/README.md, using Demo Banner.html as the visual/motion reference.

- Build it as a React component (e.g. components/demo-banner.tsx) in client/apps/web, styled with the project's Tailwind setup and shared tokens; put keyframes/@property in the shared CSS where other animations live.
- Mount it as the first element of the authenticated app shell, ABOVE the top header, full width.
- Read daysLeft and shipmentsLeft from the same plan/usage data the Settings → Plan & usage page uses; render only when the org is on the free demo. Handle singular wording and hide a message whose limit isn't applicable.
- Match the README exactly: rotating messages every 5.2s with staggered blur-rise word transitions, width-animated phrase wrapper, odometer digits, clock↔package icon swap with ring pulse, conic-ring "Free demo" chip, drifting lucide icons masked away from the text, moving gradient hairline, hover-to-pause, prefers-reduced-motion fallback, role="status" aria-live="polite".
- The CTA links to the Plan & usage settings route.
- Don't rebuild the header or Plan & usage page from the reference — they're context only.
```

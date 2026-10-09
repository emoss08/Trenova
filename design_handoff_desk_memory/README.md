# Handoff: Desk memory cards

## Overview
This handoff redesigns the two memory notes Desk shows under a reply:

1. **Ask**: the person is asked first. Copy is "Keep these steps for everyone?" or "Remember this for next time?".
2. **Saved / learned**: the memory has already been kept. Label is "Learned steps", "Learned from this conversation" or "Saved to memory".

The goals are a compact, minimal design that matches the rest of Desk, with every value taken from `tokens.css`.

## About the design files
`design/Memory cards.html` is a **design reference built in HTML**. It is not production code. Rebuild it inside the existing component, `client/apps/web/src/components/desk-chat/memory/desk-memory-notes.tsx` (`DeskMemorySaved`), using the codebase's React, i18n (`useT`), mutations and shared UI. Open the file in a browser to see both cards in light and dark side by side. Every control in it can be clicked.

## Fidelity
**High-fidelity.** Match the colours, type, spacing, radii and motion exactly. All colours are existing tokens; no new colour is introduced.

## Components

### Shared shell (`.dk-mem-card`)
- Container: `--dsk-card` background, 8px radius (`--radius-surface`).
- Border: 1px `--dsk-b` ring on the Ask card, 1px `--dsk-b-sub` ring once saved. Draw it as an inset box-shadow ring. No drop shadow.
- Placement: `margin-top: 14px` below the reply prose.
- Entrance: fade + `translateY(4px)` over 420ms with `cubic-bezier(.16,1,.3,1)` (`--dk-settle`).
- **Header row**: 38px tall, `padding: 0 6px 0 11px`, flex, `gap: 8px`, `align-items: center`.
  - Icon: the existing Desk `memory` bookmark glyph at 14px, stroke 2, in `--accent-violet`. There is no tile behind it.
  - Icon fill: the bookmark's body fills with `--accent-violet-subtle` once kept or saved. The Ask state leaves it unfilled.
  - Title: 12.5px Geist, `white-space: nowrap`. `<b>` runs are weight 500 in `--dsk-fg`. Meta runs are 400 in `--dsk-subtle` and truncate with an ellipsis.
  - Actions: flush right, `gap: 2px`.
- **Body**: `padding: 0 14px 12px 33px`, so the steps line up with the title text, not the icon.

### Steps list
- Rendering: `<ol>` with no list style. Each `li` is a grid of `16px | 1fr`.
- Counter: Geist Mono 500, 11px, line-height 20px, `--dsk-faint`.
- Text: 13px, line-height 1.55, `--dsk-fg2`, `text-wrap: pretty`. Rows are separated by `gap: 5px`.
- Inline code: Geist Mono 500 at `.86em`, `padding: 1px 5px`, 4px radius, `--dsk-sunken` background, 1px `--dsk-b-sub` border, `--dsk-fg` text. This matches `.prose code`.
- Splitting: procedure memories are split into steps. Parse `content` on `/\s*\d+\.\s+/`, or on newlines when the content uses them. Content that doesn't split renders as a single unnumbered paragraph.
- **Scroll**: wrap the list in the shared `<ScrollArea>` (`@trenova/shared/components/ui/scroll-area`) with `max-height: 172px`. When content overflows, fade the top and bottom edges with a 14px mask, showing each fade only on the side that has more content.

### Ask card: "Keep these steps for {who}?"
- **Title**: one plain sentence. Copy by kind:
  - Learned procedure: "Keep these steps for {who}?"
  - Learned fact: "Keep this for {who}?"
  - Person asked first: "Remember this for {who}?"
- **{who}**: the only trigger for the scope popover. It is the word itself, with no chevron and no pill.
  - Style: weight 500, `--dsk-fg`, dotted underline in `--dsk-b-strong` with a 3px offset. The underline turns to `currentColor` on hover and while open.
  - Word per scope: `User` → "just you", `Role` → the role name, `Agent` → "everyone".
  - Keep `scopeLabel()` for the longer label, which the popover uses.
- **Popover**: the shared `Popover`, opening below the word and aligned to its start (−10px). Width 248px, padding 4px, 8px radius, `--dsk-raised` background, `--lift-float`.
  - Rows: `14px | 1fr` grid, `padding: 7px 8px`, 6px radius, `--dsk-hover` on hover. Each row has a check icon (visible only on the selected row), a title (12.5px, 500, `--dsk-fg`) and a description underneath (11.5px, `--dsk-subtle`):
    - **Just you**: "Only in your conversations"
    - **{role.name}**, one row per writable team: "Your team"
    - **Everyone**: "Anyone using this agent". Shown when the memory is agent-scoped or agent scope is allowed.
  - Accessibility: `role="radiogroup"` and `aria-checked`.
- **Actions**, right side of the header:
  - "Not now": ghost button, 26px tall, 9px horizontal padding, 6px radius, 12px/500, `--dsk-muted`. Hover changes it to `--dsk-hover` background and `--dsk-fg` text.
  - "Keep": ink button, `--dsk-ink` background, `--dsk-ink-fg` text, 11px horizontal padding, `--ink-hover` on hover.
  - Both buttons use `white-space: nowrap` and press to `scale(.96)`.
- **Editing**:
  - Clicking the steps edits them in place. Use an auto-sized `<textarea>` (one step per line) styled exactly like the list: same font, same leading, **`--dsk-card` background, no border, no ring**. Nothing changes visually apart from the caret.
  - Hover before editing shows a faint `--dsk-hover` wash at 60%.
  - Keys: Enter saves, Shift+Enter adds a line, Esc cancels.
  - On save, join the lines back as `1. … 2. …` (or keep the content's original format).
- **Note line** under the steps: 11.5px, `--dsk-faint`, reading "From the retry that worked · Why ›".
  - Use `card.reason`'s source, or "From this conversation" as the fallback.
  - "Why" is a `--dsk-subtle` button whose 9px chevron rotates 90° when open.
- **Why panel**:
  - Opening: grid-rows `0fr → 1fr` over 300ms, `--dk-settle`.
  - Layout: `margin-top: 8px; padding-top: 9px`, 1px dashed `--dsk-b` top border, then a `62px | 1fr` grid with `gap: 6px 10px` at 12px/1.5. `dt` is `--dsk-subtle`; `dd` is `--dsk-fg2`.
  - Content: reuse `memoryWhy()` and `DeskMemoryWhy`, with restyled classes.
- **Kept state**: clicking "Keep" calls `confirmDeskMemory`.
  - The body collapses (grid-rows `1fr → 0fr` plus opacity, 360ms).
  - The header swaps to **"Kept for {who}"** (500) followed by "· Desk will follow these next time" (subtle), with an "Undo" link on the right.
  - The icon fills and pops (scale 1 → 1.28 → 1, 520ms, `cubic-bezier(.34,1.4,.64,1)`).
- **Declined**: "Not now" calls `dismissDeskMemory`. The body collapses, the header reads "Not kept" with the icon in `--dsk-faint`, and an "Undo" link returns to Ask.
- **Undo links**: 12px, `--dsk-muted`, underline in `--dsk-b-strong` with a 2px offset. On hover: `--dsk-fg` text and `currentColor` underline.

### Saved / learned card
- **Header**: filled icon, then **"Learned steps"** (or "Learned from this conversation" / "Saved to memory", as today), then a subtle meta run: "{Who} · {memoryDay}". Who uses the capitalised short word: "Just you", the role name, "Everyone".
- **Actions**: Edit (pencil) and Undo (counter-clockwise arrow) as 26×26 icon buttons, 6px radius, `--dsk-subtle`. Hover changes them to `--dsk-hover` background and `--dsk-fg` icon.
  - They are **hidden (opacity 0) until the card is hovered or focused within**, with a 160ms fade.
  - Edit opens the same in-place textarea; Undo is the existing remove mutation.
- **Body**: the same steps list and `ScrollArea`.
- **Note line**:
  - If `card.replaces` exists: "Replaces earlier steps · Why ›".
  - Otherwise: "Saved {day} from “{source}” · Why ›".
- **Why panel**: Why = `card.reason`. Before = the replaced memory's content, split into steps and rendered at 12px in `--dsk-faint` with a `--dsk-b-strong` strike-through (inline code in `--dsk-subtle`, no fill).
- **Removed, replaced and declined** keep their existing single-line forms but adopt the new header row: 38px, icon in `--dsk-faint`, the text, and an "Undo" link.

## State
Keep `MemorySaveState` (`saved | ask | edit | removed | declined | replaced`) and all existing mutations. The only additions are local UI state:
- `whyOpen: boolean`
- `editing: boolean` (replaces the separate `edit` render branch with in-place editing in both cards)
- the popover's `open`

The scope popover writes the existing `audience` / `scopeName` state.

## Design tokens
All values come from `client/packages/shared/src/styles/tokens.css`. This list uses the `--dsk-*` mapping; light first, then dark.

- **Surfaces**
  - card: `oklch(1 0 0)` / `oklch(0.145 0 0)`
  - raised: `oklch(1 0 0)` / `oklch(0.178 0 0)`
  - sunken and hover: `oklch(0.961 0 0)` / `oklch(0.218 0 0)` sunken, `oklch(0.239 0 0)` hover
- **Text**
  - fg: `oklch(0.205 0 0)` / `oklch(0.946 0 0)`
  - fg2: `oklch(0.301 0 0)` / `oklch(0.87 0 0)`
  - muted: `oklch(0.42 0 0)` / `oklch(0.709 0 0)`
  - subtle: `oklch(0.51 0 0)` / `oklch(0.709 0 0)`
  - faint: `oklch(0.65 0 0)` / `oklch(0.545 0 0)`
- **Borders**
  - b-sub: `oklch(0.94 0 0)` / `oklch(0.239 0 0)`
  - b: `oklch(0.922 0 0)` / `oklch(0.301 0 0)`
  - b-strong: `oklch(0.836 0 0)` / `oklch(0.39 0 0)`
- **Ink**: `oklch(0.205 0 0)` with hover `oklch(0.341 0 0)` and fg `oklch(1 0 0)`; inverted in dark (`0.946`, hover `0.845`, fg `0.145`).
- **Memory accent**: `--accent-violet` (hue 306), `oklch(0.55 0.185 306)` / `oklch(0.72 0.157 306)`; subtle `oklch(0.952 0.029 306)` / `oklch(0.285 0.064 306)`.
  - Replace the hard-coded `--dk-mem: oklch(0.62 0.17 300)` in `desk-memory.css` with `var(--accent-violet)`.
- **Radii**: 6px for controls and menu rows, 8px for the card and popover, 4px for inline code.
- **Elevation**: none, except `--lift-float` on the popover.
- **Type**
  - Geist: 12.5 header, 13 steps, 12 buttons and the Why panel, 11.5 note.
  - Geist Mono: 11 counters, `.86em` code.
- **Motion**
  - `--dk-settle` `cubic-bezier(.16,1,.3,1)`; spring `cubic-bezier(.34,1.4,.64,1)`.
  - Durations: 140ms hovers, 160ms action fade, 200ms chevron, 300ms Why, 360ms collapse, 420ms entrance, 520ms icon pop.
  - Disable all of these under `prefers-reduced-motion`.

## Assets
None new. Icons are the existing Desk `memory`, `chevR` and edit/undo glyphs from `desk-icons.tsx`, plus a check.

## Files
- `design/Memory cards.html`: the reference, with both cards in light and dark, all interactive.
- `PROMPT.md`: the implementation brief for Claude Code.

# Handoff: Shipments page redesign (v2)

A complete redesign of the Shipments page (`client/apps/web/src/routes/shipment/page.tsx`). The current page stacks ten KPI boxes, an unconfigured map, five side panels, the table and three bottom modules. The new page puts **the table first**. Around it there is only what a dispatcher can act on: a streamed AI briefing, a capacity strip (drivers and/or carriers), and a floating Brief/Activity panel with an action queue and a live watchlist. Every number on the page leads to an action. If you can't act on it, it doesn't belong here.

---

## Read this first — mandate for Claude Code

**You have full range and full control.** This redesign needs **backend and client changes**. Change anything you need to: Go services, GraphQL/REST schema, database migrations, queries, analytics endpoints, permissions, shared packages, `data-table.tsx` itself, routes, stores and tests. Delete or replace existing shipment-page components (`kpi-rail`, `right-stack`, `bottom-modules`, `saved-views-bar`, the map panel's placement, and so on) when the new design makes them obsolete. Do not work around existing code. Fix it properly.

### Engineering rules (non-negotiable)

- **DRY.** One source of truth for each concept. Stage and status derivation, coverage resolution (driver vs. carrier), HOS formatting, money and duration formatting, filter predicates and shortcut labels each live in exactly one module and are imported everywhere else. The prototype duplicates logic in places (for example `stageOf` in `shell.jsx` and `groupOf` in `board.jsx`). Collapse those duplicates.
- **SOLID.**
  - *Single responsibility:* each block (briefing, capacity strip, action queue, watchlist sections, expanded-row sections) is its own component with its own data hook.
  - *Open/closed:* org type (asset / brokerage / both) and integration availability (AI provider, ELD/HOS, maps) are **capabilities** that components read. Don't hard-code `if (org === "brokerage")` branches across the tree. Use a strategy/registry: a `CoverageProvider` for drivers and one for carriers, composed for "both".
  - *Liskov / interface segregation:* drivers and carriers implement one `CapacityUnit` interface (id, display name, initials, hue, availability, ring metric, best-load matching, assign/tender action). The strip and expanded row should never care which one they're rendering.
  - *Dependency inversion:* UI depends on hooks/interfaces, never on transport details.
- **Use `client/apps/web/src/components/data-table/data-table.tsx`.** The shipments table must be built on the existing `DataTable`: columns, sorting (`data-table-sort-builder`), filtering (`data-table-filter-builder` + `data-table-filter-chips`), pagination (`_components/data-table-pagination`), column visibility (`data-table-view-options` / `data-table-display-menu`), saved views (`data-table-config-manager`), selection (`data-table-selection-column`, `data-table-dock` for the bulk bar), and the context menu. If something the design needs is missing, **extend `DataTable` generically** (for example row grouping with sticky collapsible group headers and per-group aggregates, a single-row expand slot, a keyboard row cursor). Don't fork it. Every other list page should be able to use those features too.
- **Existing patterns win** where they conflict with the mock on small details (Kbd component, Button variants, i18n via `useT`, `formatShortcut`/`isMacPlatform` from `@trenova/shared/lib/shortcuts`, tokens from `tokens.css`).
- Respect `prefers-reduced-motion` (`app.css` already collapses animations).
- Write tests alongside new hooks and DataTable extensions. Follow existing `__tests__` conventions.

### About the design files

`design/` contains an **HTML/React (Babel) prototype with mock data**. It is a design reference for look and behavior, not production code. Recreate it inside Trenova's real React + Tailwind + Base UI environment using `@trenova/shared` components and the tokens in `client/packages/shared/src/styles/tokens.css`. Open `design/index.html`. The **Tweaks** toolbar toggle switches AI on/off, org type, ELD/HOS integration, board volume (quiet / busy / high volume, 480 rows), theme and density.

### Fidelity

**High fidelity.** Layout, hierarchy, copy, spacing, interactions and motion are intentional. Colors already use `tokens.css` values (see Design tokens). **No drop shadows anywhere.** Separation comes from 1px borders and surface steps only. Avoid wrapping content in cards. Content sits on the surface, separated by hairlines.

---

## Capabilities (drive conditional UI from these, nowhere else)

| Capability | Source | Effect when OFF |
|---|---|---|
| `ai` | org has an AI provider configured (existing `page-assistant-availability`) | No briefing sentence. The Brief tab is renamed **Overview**, the queue is labeled **Exceptions** with manual actions, there are no drafted messages or fit scores, and a dashed note links to Integrations. |
| `orgType` | organization settings: `asset` \| `brokerage` \| `both` (add this if it doesn't exist) | **asset**: drivers only. **brokerage**: carriers only (tender flows, "Booked" status label, carrier in the Coverage column). **both**: Drivers/Carriers tabs in the strip, and the expanded row offers drivers then "Or tender to a carrier". |
| `hos` | ELD/telematics integration connected | HOS rings, "· 7:48 HOS" sublines, the drive-time bar and HOS queue items are hidden. "Connect ELD for hours of service" link appears under the driver strip. |
| `maps` | Google Maps runtime config (existing) | The Map view shows the "isn't connected" message and a Connect button. |

There is **no "Nova" branding** in the app (Nova is signup-only). The assistant is just "Assistant". The assistant chat already exists in the bottom-right corner, so this page has **no chat UI**. Ctrl/⌘+J opens that existing assistant.

---

## Layout

```
┌ App shell (existing: org switcher, module switcher, Ctrl K search, sidebar) ─────────────┐
│ Page head: "Shipments"                                          [↻]  [+ New shipment]   │
│ ── header block (.hdx, padding 12px 20px 14px, gap 14px) ─────────────────────────────── │
│ AI briefing sentence (AI only, full width, streamed)                                     │
│ Capacity strip: [summary 260px │ scrolling avatar dock with edge fades]                  │
│ ── board (full width) ────────────────────────────────────────────────────────────────── │
│ Toolbar: [Search……] [Filter] [Sort]      [Table|Timeline|Map] [Group] [Columns] [⇩] [Views] [▣] │
│ Table: edge-to-edge (no outer card), sticky header, grouped rows, inline expand          │
│ Footer: ⌨  Showing 1 to 10 of 480 results          Rows per page [10]  ‹ Page 1 of 48 ›  │
└──────────────────────────────────────────────────────────────── floating panel ┐ ────────┘
                                                        (top 60px, right 12px, bottom 12px,
                                                         width min(372px, 100% − 28px))
```

- Workspace grid: `auto` (page head) / `auto` (header block) / `minmax(0,1fr)` (board).
- **Floating side panel** (Brief / Activity): absolutely positioned over the workspace. Background `--raised`, `1px solid --border-strong`, radius 12px, **no shadow**. Enter animation translateX(16px) scale(.98) → none, 320ms `--ease-settle`. Toggled from the toolbar's panel button and its own ×. On narrow containers (≤820px) it spans the width with 8px insets.

---

## 1. AI briefing (AI only)

- One sentence, full width, 15px / 1.6, color `--foreground` (all white in dark). No icon, no timestamp.
- **Streams in**: reveal 2–4 characters every ~28ms. Each word fades from `blur(3px)`/opacity 0 over 420ms, with a 2px blinking caret until done.
- Clickable phrases are underlined (1px `--border-strong`; hover fills with brand at 22%). Clicking one applies a table filter (late / moving / uncovered).
- Example copy (busy): "14 loads today. 5 are moving on schedule, 3 are running late behind the storm over Iowa, and 2 still need a driver. I've drafted fixes in the brief."
- The coverage noun follows `orgType`: "a driver" / "a carrier" / "coverage". Pluralization must be correct (1 load / N loads).
- **Backend:** add a briefing endpoint that returns structured segments `{text, filter?}` so the client can stream and link them. Cache per org per ~5 minutes and invalidate on shipment events.

## 2. Capacity strip

Left summary (260px, right hairline) + right dock.

- **Drivers mode:** big number "24" + "drivers ready for 69 uncovered loads". A 5px bar split into ready (`--success`), within 2h (success at 40%) and short (amber hatched). Legend below. Action:
  - **both**: "Tender 37 to carriers". Tenders the overflow loads and switches to the Carriers tab.
  - **asset**: "Review 37 you can't cover". Filters the table to uncovered loads.
- **Carriers mode:** "12 carriers posting trucks for 68 untendered loads". Bar for awaiting acceptance vs. not tendered, plus "avg $2.50/mi". Action "Tender all 68 to best matches", then "Everything is tendered".
- **both:** a segmented control "Drivers 24 | Carriers 12" replaces the label.
- **Dock:** horizontal scroll with left/right arrow buttons, grouped "Ready now | Within 2h" (drivers) or "Trucks posted | Usually accept" (carriers), each group with a vertical label.
  - Edge fades of 24px on the left and 48px on the right, plus matching padding. The left fade shows only after scrolling and the right fade hides at the end.
  - **Avatar:** 30px (circle for drivers, rounded square for carriers) inside a 40px SVG ring. The ring shows HOS remaining out of 11h for drivers (amber below 4h) or acceptance % for carriers. Initials must be perfectly centered.
  - A green status dot shows on drivers ready now; on carriers it carries a truck-count badge. Below the avatar: first name and city / "free 13:40" / "$2.31/mi".
- **Click an avatar** to open a popover (340px, portal, no shadow) with the name and meta, then the "Best load for X" / "Best loads to tender" lists. The primary row has an ink button, the alternate a plain one.
  - **Assign** (driver) or **Tender** (carrier) commits. The avatar animates out (scale .7, translateY −10px, 320ms) and the counts update.
  - Esc, an outside click or scrolling closes the popover.
- **Backend:** add an available-capacity query (drivers with status, location and HOS; carriers with posted trucks on open lanes, acceptance rate and rate). Add a best-match service that ranks uncovered loads per unit (deadhead distance, HOS fit, lane history; for carriers, rate and acceptance). Add a bulk tender mutation.
- **Scale:** must perform with hundreds of drivers/carriers. Virtualize the dock.

## 3. Board / table (built on `DataTable`)

**Toolbar (padding 10px 16px 10px 20px, gap 8px):** search field (flex 0 1 300px, min 120px; the only shrinkable item; "/" focuses it) with removable filter tokens and a quick-filter dropdown (Late, Uncovered, Moving, Delivering today, Reefer, Low margin, each with counts) · **Filter** (faceted checkboxes: Status, Equipment, Tender, Customer, with counts; badge shows the number active) · **Sort** (field list, click again to flip asc/desc, Reset) · spacer · view switch **Table / Timeline / Map** · **Group** toggle · **Columns** (visibility; Lane locked; Reset) · Export · Views · panel toggle. Active buttons get a brand-subtle tint. Below 1180px container width, the labels collapse to icons. Display and Views hide below 1000px. **Buttons never shrink.**

**Table:** edge-to-edge, top border only, no outer radius. Header: 32px, 11px uppercase, letter-spacing .02em, `--muted-foreground`, 1px dividers between headers, sort affordance on every column (faint until hover). The checkbox column is separate (36px). Row height is 32px compact (default) or 52px comfortable; compact shows one line per cell.

Columns (match `shipment-columns.tsx`): Lane (city → city plus an inline progress bar), Status, Tender, Billing, PRO / BOL, Order, Customer, Coverage, ETA, *Pickup appt (hidden by default)*, *Delivery appt (hidden by default)*, Revenue ($ plus $/mi), Margin, and an actions column (expand chevron + ⋯).

- **Status badge:** 22px, radius 5, 1px border at 30% of the tone, subtle background, 6px dot. Tones: In transit = brand, Delayed = danger, Assigned/Booked = teal, New = warning, Delivered = success.
- **Coverage:** avatar + name + unit. Carrier = rounded-square avatar + carrier name + driver/MC. Tendered = dashed faded avatar + "Tendered · awaiting". Uncovered = amber "⚠ Needs coverage".
- **Grouping (extend `DataTable`):** sticky group header rows (top 32px) with collapse chevron, color square, label, count and revenue sum, right-aligned. Groups: Needs attention (late), Needs coverage, Moving, Scheduled, Delivered. Grouping must compose with sort, filters and pagination (pagination is over the flattened, grouped order; group counts show group totals, not page counts).
- **Row menu (⋯):** Edit (E), Duplicate, Copy link (⌘/Ctrl L), Transfer ownership, Open full record, a divider, then **Cancel shipment** in danger color. Wire it to the existing `RowAction<Shipment>[]`.
- **Pagination footer:** "Showing **1** to **10** of **480** results" · Rows per page [10/25/50] · ‹ Page **1** of **48** ›, plus a keyboard-shortcuts button. Use `DataTablePagination`.
- **Timeline view:** a bar per shipment across 04:00–24:00 with a red "now" line. Late loads get a hatched red extension from the appointment to the new ETA. Uncovered/new pickups are dashed. Click a bar to expand that row.
- **Map view:** use the existing map panel when `maps` is on; otherwise show the not-connected state.

**Selection / bulk bar:** use `data-table-dock`. Actions: Tender, Assign (labeled "Tender to carrier" in brokerage), Export, clear.

## 4. Expanded row (replaces the current 4-column `expanded-row.tsx`)

Clicking a row (or pressing Enter) toggles inline expansion. Only one row is open at a time, and the arrow/J/K keys move the expansion. The panel is sticky-left and sized to the visible scroll width, so it never scrolls sideways. Background: row tint `color-mix(brand 7%)`. **No inner cards.**

Grid: `minmax(0,1fr) 340px`. Main column on the left, a "next step + quick actions" column on the right with a left hairline. Below 900px they stack.

- **Route track:** origin node, a dashed line with a solid progress fill (brand / danger / success), a hatched red slip stretch when late, a pulsing truck marker at the current position, and the destination node. Below it, three columns:
  - Pickup: facility name, `City · window`, status ("Departed 06:12").
  - Middle: the current situation and "x mi done · y to go · ping 3m".
  - Delivery: facility name, appointment, status ("Will miss · +2h 40m" in danger).
- **Money line** (dashed top hairline): Revenue (bold, with $/mi), Linehaul, Fuel, Accessorials, Est. cost, Margin (colored). **Doc pills** on the right: Rate con, BOL, POD (done / waiting / due) + Upload.
- **Next step (right column, exactly one):**
  - Uncovered:
    - Suggested drivers (asset/both). With AI: fit %. Without: "Nearest drivers" by distance. Hover reveals "Assign".
    - Carriers with quote, $/mi and acceptance % plus "Tender" (brokerage, or "Or tender to a carrier" in both).
  - Late: with AI, a drafted delay notice plus "Send to {customer}" and "Edit". Without AI: "{delta} behind · {reason}" plus "Notify customer".
  - Covered: driver (and/or carrier) with a message button, plus the HOS bar (`hos` only).
  - Just assigned: a success state ("Assigned" / "Tendered", "dispatch sheet sent" / "rate con sent").
- **Quick actions** (2-column list, 30px rows, kbd hints): Edit (E), Duplicate, Transfer ownership, Copy link (⌘/Ctrl L), Add comment, **Cancel shipment** (danger). There is no "Ask assistant" button here.
- A "Collapse · Esc" button is centered at the bottom.
- Keep the existing lazy-loaded blocks (`route-timeline-block`, `financials-block`, `document-stack`, `quick-actions-block`, `telematics-forms-block`) where they fit, restyled to this layout. Lift duplicated pieces into shared components.

## 5. Floating panel — Brief tab

The tab label is **Brief** (AI) or **Overview** (no AI), with a danger count badge of open actions. The second tab is **Activity**.

### 5a. Suggested actions / Exceptions (one-at-a-time queue)

- Header: "SUGGESTED ACTIONS" + "2 of 5". A segmented progress bar of 3px segments: done = success, current = foreground, rest = 10% foreground.
- **Current item:** a tone dot (danger / warning / teal / brand) + kind ("Coverage", "Delay notice", "Hours of service", "Tender") + due time; title (15px/600); reason (12.5px muted); impact facts in mono separated by dots ("$2,900 · 11 mi out · 96% fit").
- **Buttons:** primary (teal for AI-approve, ink for manual) with a **⌘/Ctrl ↵** hint; Review; **Later** with an **Alt L** hint (moves the item to the back). These shortcuts mirror the existing approval box keybinds in `keybinds.config.ts`.
- New items slide in (translateX 12px → 0, 360ms).
- **Up next:** one-line rows (dot, title, due), clickable.
- After acting: an "Undo" line ("✓ Marcus Bell assigned to S2610-0431 · Undo").
- **Empty:** "All caught up · N handled this shift".
- **Org-aware content:** brokerage turns "Assign driver X" into "Tender {lane} to {carrier}" with quote, acceptance and margin. HOS items are dropped when `hos` is off or the org is a brokerage.
- **Backend:** a suggestions service that produces typed actions `{kind, shipmentId, title, reason, impact[], primaryAction, reviewTarget, due}`. With AI, the titles and reasons can come from the model. Without AI, use rule-based exceptions with manual actions.

### 5b. Watchlist (built for high volume, aggregate first)

- **Today's deliveries:** "412 / 438 on time". An 18-column hourly histogram (06–24) with stacked delivered / scheduled / late segments, a "now" line, and an axis.
  - Hovering a column shows its counts in the caption. Clicking it filters the table to that hour.
  - Below: the 3 worst-late rows (delta · city · customer, click to expand that row) + "Review all N late".
- **Uncovered pickups:** total uncovered revenue + load count. Four window tiles (< 2h [danger-tinted], 2–6h, Later today, Tomorrow+), each with count and revenue; clicking filters the table. Then "Next pickup in 1:29:58 · lane · [Cover]" (live countdown, red under 1h).
- **Detention accruing:** running total (two decimals, thousands separators) + stop count. The top 3 longest waits show as rows with a growing fill bar, driver/carrier, elapsed time, $ and a **Bill** button. "Review all N" filters the table. A billed amount freezes.
- **Ready to bill:** total + count. The top 4 customers get bars (count, $), then "+N more customers". A **hold-to-transfer** button (900ms press fills green, then "✓ Transferred to billing"). "Review first" filters the table.
- **Backend:** aggregate endpoints for histogram buckets, uncovered windows, detention accrual (facility dwell from telematics or arrival events, billable after free time) and billing-ready totals. Push updates over the existing realtime channel. The client must not compute these over the full shipment list.

### 5c. Activity tab

A vertical timeline feed (7px nodes: brand = arrival, danger = delay, teal = assistant/automation), the actor in bold, and relative time in mono.

---

## Keyboard (register in `config/keybinds.config.ts` under a new "Shipments" group)

| Keys | Action |
|---|---|
| J / ↓, K / ↑ | Move row cursor (moves the expansion if a row is open) |
| Enter | Expand / collapse the row |
| Esc | Collapse → clear selection → clear cursor |
| X | Toggle selection of the cursor row |
| E | Edit shipment |
| Alt C | Copy PRO number |
| ⌘/Ctrl L | Copy link |
| / | Focus table search |
| ⌘/Ctrl ↵ / Alt L | Approve / Later on the current queue item |
| ⌘/Ctrl K, B, J, / | Existing global: palette, sidebar, assistant, shortcuts dialog |

Ignore keys in inputs and dialogs (use the existing `isTypingTarget` / `isWithinDialog`). Use `formatShortcut` for labels (⌘ on Mac, Ctrl elsewhere).

---

## Motion

Eases from tokens: `--ease-swift` cubic-bezier(.2,.8,.2,1), `--ease-settle` (.16,1,.3,1), `--ease-spring` (.34,1.4,.64,1).

- Rows: rise 3px over 300ms, staggered 18ms (cap at 14).
- Expansion: translateY(−6px) → 0, 280ms.
- Popovers: translateY(−4px) scale(.98) → none, 180–220ms.
- Success state: scale .97 → 1, spring.
- No looping animation except the "live" indicators (the truck ping and the countdown). Nothing may cause whole-page re-renders on a timer: isolate ticking clocks in leaf components.

## Design tokens (from `tokens.css`; dark values shown, light in the prototype CSS)

- Surfaces: canvas `oklch(0 0 0)`, card `oklch(.145 0 0)`, raised `oklch(.178 0 0)`.
- Borders: subtle `oklch(.239 0 0)`, border `oklch(.26 0 0)`, strong `oklch(.321 0 0)`.
- Foreground `oklch(.946 0 0)`, muted `oklch(.709 0 0)`, subtle `oklch(.65 0 0)`.
- Brand `oklch(.717 .148 258)`, danger `oklch(.626 .193 23)`, warning `oklch(.817 .164 76)`, success `oklch(.645 .145 147)`, teal accent `oklch(.72 .089 182)`, each with its `-subtle` / `-foreground` pair.
- Radius: control 6px, surface 8px; the floating panel is 12px.
- Type: Geist (UI), Geist Mono (numbers, IDs, kbd). Sizes: 11 (labels, uppercase .02em), 12–12.5 (body), 13 (rows), 15 (briefing, queue title), 17 (page title), 18–26 (big figures, −0.03 to −0.04em).

## Files in `design/`

- `index.html`: entry point (open it in a browser).
- `app.jsx`: state, keyboard, capability flags (Tweaks), assign/tender flows.
- `shell.jsx`: app shell, page head, filter field and query parsing, Capacity strip.
- `board.jsx`: toolbar, grouping, pagination, Filter/Sort/Columns menus, timeline, map state.
- `rows.jsx`: column config, cells, row menu, expanded row, shortcuts dialog.
- `rail.jsx`: floating panel, briefing, action queue, watchlist, activity.
- `data.jsx`: mock shipments (busy / quiet / 480-row high volume), drivers, carriers, coverage resolution.
- `ship.css`: all styles (token-mapped).
- `icons.jsx`: inline icons (replace with `@trenova/shared/components/icons`).

Mock data shapes in `data.jsx` show the fields each view needs (`HOURS[id] = [pickup, delivery, slippedEta?]`, `det` minutes, `tenderTo`, and so on). Treat them as a starting point for the backend contracts above, not as the final schema.

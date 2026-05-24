# Phase 1 — Shipments Table

The operational core of the workspace. Build this first; everything else drills into it.

## Goal

Recreate the shipments table from `design/app.jsx` (`TablePanel`, `ShipmentTable`, `ShipmentRow`, `ExpandedRow`, `ViewSegments`, `FilterRow`, `TableFooter`) and the timeline view (`design/timeline.jsx`) inside the Trenova client. After this phase, a dispatcher should be able to scan all shipments, filter by saved view + status chips, expand a row inline, and toggle Table↔Timeline.

## Reuse from the codebase first

Search **before building**. Likely existing primitives:

- `client/src/components/ui/Table` or `data-table/` — column definitions, sort headers, sticky header, virtualized body
- `client/src/components/ui/Badge` or `Pill` — for `StatusPill`
- `client/src/components/ui/Button` — for filter/sort/columns/save-view buttons
- `client/src/components/ui/Tabs` — for the saved-views segment row
- `client/src/components/ui/Tooltip` — for the `lastEvent` hover on the status cell
- `client/src/hooks/useShipments` (or equivalent) — data
- `client/src/lib/format.ts` — currency, percentage, miles

If `<DataTable>` exists, this whole phase reduces to column config + cell renderers + the Timeline view component. **Do not hand-roll a `<table>` if a primitive exists.**

## Build new (only if not in the codebase)

| Component | Purpose |
|---|---|
| `<ShipmentTable>` | Wrapper around the table primitive with the column config below |
| `<LaneCell>` | Origin code → dest code, progress bar, miles + commodity |
| `<StatusCell>` | Status pill + last-event timestamp with tooltip |
| `<EtaCell>` | ETA + HOS-left line, color by `etaStatus` |
| `<RevenueCell>` | Revenue + RPM stacked, right-aligned |
| `<MarginCell>` | Margin %, color-graded (≥20 success, ≥15 warning, else danger) |
| `<DriverCell>` | Driver + tractor/trailer, or "Needs driver" when unassigned |
| `<ExpandedRow>` | Inline detail panel — see below |
| `<SavedViewsBar>` | Tabbed segment row above the table |
| `<FilterChipRow>` | Filter chips with toggle behavior, plus Filter / Sort / Columns buttons |
| `<TimelineView>` | Gantt-style alternate view (see `design/timeline.jsx`) |

## Column configuration

| # | Column | Width | Align | Source field(s) | Cell |
|---|---|---|---|---|---|
| 1 | Lane | 20% | left | `originCode`, `destCode`, `progress`, `miles`, `commodity` | `<LaneCell>` |
| 2 | Status | 14% | left | `status`, `lastEvent`, `lastEventAt` | `<StatusCell>` |
| 3 | PRO / BOL | 11% | left | `pro`, `bol` | mono, two lines |
| 4 | Customer | 13% | left | `customer`, `weight` | text + mono small |
| 5 | Driver / Equip | 13% | left | `driver`, `tractor`, `trailer` | `<DriverCell>` |
| 6 | ETA | 10% | left | `eta`, `etaStatus`, `hosLeft` | `<EtaCell>` |
| 7 | Revenue | 10% | right | `revenue`, `rpm` | `<RevenueCell>` |
| 8 | Margin | 7% | right | `margin` | `<MarginCell>` |
| 9 | actions | 2% | left | — | row menu (`⋯`) |

Header row uses `.label` style (10px / 600 / uppercase / 0.08em / `--fg-subtle`).

Row height is driven by `--row-h` (density-aware): 28 / 36 / 44 px. Cell padding `var(--pad-y) var(--pad-x)`.

## Lane cell — detail

```
[ TERM-LA ]  →  [ DC-CHI ]  ▰▰▰▰▰▱▱▱▱▱  62%
2,012mi · Industrial parts
```

- Origin/dest codes: mono tabular, 11.5px, weight 600
- Arrow `→` at `--fg-subtle`
- Progress bar (`.lane-bar`): 4px tall, rounded, `flex:1; max-width:80px`, fill colored by row state:
  - `is-done` (Delivered) → success
  - `is-late` (At Risk) → danger
  - `is-warn` (Detention) → warning
  - default → brand
- Percent label after bar: 9.5px mono tabular, `--fg-subtle`
- Subline: `{miles}mi · {commodity}` at 9.5px mono `--fg-subtle`
- When the row is `highlighted` (hover from map), the origin code switches to `var(--brand)` color

## Status cell — detail

`StatusPill` mapping (already in `design/components.jsx`):

| status | pill class | icon |
|---|---|---|
| Delivered | `pill-soft-success` | check |
| In Transit | `pill-soft-brand` | route |
| At Risk | `pill-soft-danger` | alert |
| Detention | `pill-soft-warning` | clock |
| Loading | `pill-soft-muted` | dots |
| Pending | `pill-soft-muted` | clock |

After the pill, show `lastEventAt` (e.g. "12 min ago") in 9.5px mono `--fg-subtle`, max-width 100px with ellipsis. Wrap it in a Tooltip showing `lastEvent`.

## ETA cell — detail

- Line 1: `eta` text, mono tabular, weight 500. Color by `etaStatus`:
  - `late` → danger, `watch` → warning, `delivered` → success, `pending` → fg-subtle, default → fg
- Line 2: `HOS {hosLeft}` at 9.5px `--fg-subtle`, omitted when `hosLeft === "—"`

## Margin cell — detail

`<span>{margin}%</span>` colored:
- `≥ 20` → success
- `≥ 15` → warning
- else → danger

Weight 600.

## Driver cell — detail

When `driver === "—"`:
```
[!] Needs driver        (var(--warning), 11px, with alert icon)
```
Otherwise:
```
M. Alvarez
T-1184 · 53' DRY        (mono, 9.5px, --fg-subtle)
```

## Row interactions

| Event | Behavior |
|---|---|
| `click` row | Toggle expanded (one row at a time — clicking a different row replaces the expanded one) |
| `mouseenter` row | Call `onHover(shipmentId)` so the map can highlight the matching truck |
| `mouseleave` row | `onHover(null)` |
| `click` action `⋯` | Open row menu (do NOT propagate to row click) |
| highlighted (from map) | `background: color-mix(in oklch, var(--brand) 7%, transparent)` |
| expanded | `background: color-mix(in oklch, var(--brand) 5%, transparent)` |

Hover background: `color-mix(in oklch, var(--fg) 4%, transparent)`. Transition: `background 80ms ease`. Cursor: pointer.

## Expanded row — detail

`design/app.jsx` line ~837. The expanded row spans all 9 columns (`<td colSpan={9}>`) and contains a 3-column layout:

1. **Stops timeline** (left, ~40%) — vertical list of stops with timestamps, status icons, and notes
2. **Documents + financials** (middle, ~30%) — POD / BOL links, accessorials, fuel surcharge, line haul, total
3. **Communications + actions** (right, ~30%) — last 3 events, comments preview, action buttons (Tender, Reassign, Add note, Open detail)

Background: `color-mix(in oklch, var(--brand) 3%, var(--card))`. Padding: 12px 16px. Animation: `fade-in` (200ms opacity 0→1).

> **Read `design/app.jsx` lines 837–947 for the exact layout.** Recreate using the codebase's Card / Button / Link primitives.

## Saved-views bar (`<ViewSegments>`)

Horizontal scrollable row of buttons. Each button:
```
{label}  [count]
```
- Active: weight 600, `--fg`, count chip uses `--brand` text on `--brand-soft` background, **2px brand underline** at the bottom
- Inactive: weight 500, `--fg-muted`, count chip on `color-mix(--fg 6%, transparent)`
- Padding 5px 10px, 11.5px font, gap 6px between label + count

Saved views (mock):
```ts
[
  { id: "all",       label: "All shipments",  count: 142 },
  { id: "transit",   label: "In transit",     count: 58 },
  { id: "at-risk",   label: "At risk",        count: 9 },
  { id: "unassigned",label: "Unassigned",     count: 5 },
  { id: "delivering-today", label: "Delivering today", count: 23 },
  { id: "detention", label: "Detention",      count: 7 },
]
```

Right side of the bar: a Table↔Timeline toggle (segmented control, 2 buttons with icon+label) and a "Save view" ghost button with `+` icon. Persist `viewMode` to `localStorage["trenova.viewMode"]`.

## Filter row

```
[Filter] [Sort]  |  [chip] [chip] [chip] [chip]    142 of 142 results   [Columns]
```

- Filter / Sort / Columns: `.btn` style, 26px tall, with an icon
- Chips are `<button>` with `pill pill-soft-{color}` when on, `pill pill-soft-muted` when off. When on, append " ×" so the user knows clicking again removes it. Border becomes `1px solid color-mix(in oklch, var(--{color}) 30%, transparent)`.
- Default chips:
  - `at-risk` → danger
  - `reefer` → brand
  - `today` → muted
  - `hot` → warning
- Right-aligned counter: `{filtered} of 142 results`, mono 10.5px `--fg-muted`

## Timeline view

Alternate to the table — see `design/timeline.jsx`. Out-of-scope for the first ship of Phase 1 if it stretches the timeline. Acceptable to land Table-only and add Timeline as 1.b. The toggle should still appear, with Timeline showing a "Coming soon" placeholder if not yet implemented.

When you do build it, key behaviors:
- Each row = one driver
- A horizontal time track per row (clamped to visible window — `overflow:hidden` so blocks can't escape)
- Shipment blocks colored by `etaStatus` (moving=success, dwell=danger, loading=info, unassigned=fg-subtle)
- Blocks that start before the visible window show a leading `‹` indicator
- Click a block = same as clicking the table row (expand)

## Table footer

Single row, 11px mono, `--fg-muted`. Shows: result count, last refresh time, optional pagination. See `design/app.jsx` line ~935.

## State this phase introduces

```ts
const [expandedId, setExpandedId] = useState<string | null>(null);
const [highlightId, setHighlightId] = useState<string | null>(null);  // from map hover
const [selectedView, setSelectedView] = useState<string>("all");
const [filters, setFilters] = useState<string[]>([]);                 // chip ids
const [viewMode, setViewMode] = useState<"table" | "timeline">("timeline");
```

`expandedId` and `highlightId` are mutually exclusive surfaces (expand wins visually). `highlightId` is set externally — exported up so Phase 3 (map) can both read and write it.

## Acceptance criteria

- [ ] Table renders all 9 columns with correct alignment and density
- [ ] Saved-view tabs filter the underlying dataset
- [ ] Filter chips toggle on/off and AND with the saved view
- [ ] Click row expands inline; clicking another row swaps; clicking the same row collapses
- [ ] Hover row sets `highlightId` (verify in dev tools / Phase 3 will consume it)
- [ ] Status, ETA, and margin colors match the design tokens exactly
- [ ] Density classes (`compact` / `cozy` / `comfortable`) on the wrapper change row height + padding without re-render glitches
- [ ] All numeric cells use tabular nums + Geist Mono
- [ ] Last-event timestamp shows tooltip with full event text
- [ ] Action `⋯` menu does not bubble row click
- [ ] `localStorage["trenova.viewMode"]` persists Table↔Timeline choice
- [ ] Lighthouse / a11y: header cells use `<th scope="col">`, expand has aria-expanded, row is `role="button"` if interactive

## Files to study

- `design/app.jsx` lines 637–947 — `TablePanel`, `ViewSegments`, `FilterRow`, `ShipmentTable`, `ShipmentRow`, `ExpandedRow`, `TableFooter`
- `design/components.jsx` — `StatusPill`, helper styling
- `design/data.jsx` — `SHIPMENTS`, `SAVED_VIEWS` shapes
- `design/timeline.jsx` — Timeline view (Phase 1.b)

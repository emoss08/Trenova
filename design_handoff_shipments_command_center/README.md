# Handoff: Shipments Command Center

Operations workspace for dispatchers — a multi-pane workspace combining KPIs, a live map, an exception inbox, an unassigned-loads queue, an HOS watchlist, a shipments table (with timeline view), and supporting modules (lane heatmap, customer mix, tomorrow's pickups, activity feed, comments).

## Read this first — implementation rules

> **The HTML files in `design/` are design references, not production code.** They show *what* the result should look like and *how* it should behave. They are intentionally written as a single Babel-transpiled HTML prototype with mock data so the team can review interactions without standing up a backend.
>
> **Do not copy the HTML/JSX into the codebase wholesale.** Recreate the designs inside Trenova's existing React + Tailwind environment using the components and patterns that already exist there.

### Reuse over reinvention

Before building anything new, **search the codebase** for an existing equivalent. The Trenova client already has a UI library and conventions in place — use them. Likely locations to search:

- `client/src/components/ui/` — primitives (Button, Badge, Card, Input, Select, Tabs, Sheet, Dropdown, Tooltip)
- `client/src/components/data-table/` — the existing table primitive (column config, sorting, pagination, expand)
- `client/src/components/` — feature components (Map, Sidebar, etc.)
- `client/src/styles/app.css` — design tokens (colors, radii, typography). The mock's CSS variables in `design/index.html` (`--brand`, `--success`, `--card`, `--border`, etc.) are deliberately matched to this file. **Use the codebase's tokens, not the mock's.**
- `client/src/hooks/` — data hooks (`useShipments`, `useDrivers`, etc.)
- `client/src/lib/` — utilities (formatters, classnames, query client)

If a primitive exists, **use it as-is**. If it's close but missing a feature, extend it. Only build a new primitive if nothing comparable exists.

### Fidelity

**High-fidelity.** Colors, typography, spacing, density tokens, micro-interactions, hover states, and information hierarchy in the mocks are intentional. Match them. Where the codebase's existing components dictate slightly different choices (e.g. a different Badge variant), follow the codebase — consistency with what shipped beats consistency with the mock.

---

## Phased rollout

This is a large surface. **Ship it in phases, table first.** Each phase is independently demoable.

| Phase | What you build | Why this order | Doc |
|---|---|---|---|
| **1. Shipments table** | Table + Timeline view, filter chips, saved views, expandable row, row→map hover plumbing | This is the operational core — dispatchers spend most of their time here. Makes everything else have a place to drill into. | [`phase_1_table.md`](./phase_1_table.md) |
| **2. KPI rail** | 5 KPI card variants (Hero, Ring, GoalBar, Stat, Watchlist) above the table | Adds at-a-glance status without depending on the heavier modules. Self-contained, low coupling. | [`phase_2_kpi_rail.md`](./phase_2_kpi_rail.md) |
| **3. Live map panel** | Map with truck pins, route overlays, hover sync with table, map-style toggle | Spatial context for the table. Requires the table's selection/hover state to already be live. | [`phase_3_map.md`](./phase_3_map.md) |
| **4. Right-stack modules** | Unassigned queue (drag to assign), Exceptions, HOS watch, with reorder + hide + persist | The fastest-moving collaborative surface. Drag-to-assign needs the table+map to already exist as drop reference. | [`phase_4_right_stack.md`](./phase_4_right_stack.md) |
| **5. Bottom modules** | Lane heatmap, Customer mix / Tomorrow pickups (tabbed), Activity feed | Lower-priority analytical context. Build last. | [`phase_5_bottom_modules.md`](./phase_5_bottom_modules.md) |
| **6. Polish** | Tweaks panel, density modes, theme toggle, accent hue, layout variants, time window, saved-view persistence | Cross-cutting concerns; layer on once the surface is stable. | [`phase_6_polish.md`](./phase_6_polish.md) |

Each phase doc lists: scope, components to reuse from the codebase, components to build new, data contracts, interactions, and acceptance criteria.

---

## Design tokens (already in `client/src/styles/app.css`)

The mock's CSS variables mirror these — **read tokens from app.css in the real implementation, do not redeclare**.

```
/* Surfaces */
--bg, --bg-elev, --card, --card-2
/* Borders */
--border, --border-2
/* Text */
--fg, --fg-muted, --fg-subtle
/* Brand + semantic */
--brand, --brand-soft
--success, --success-soft
--warning, --warning-soft
--danger,  --danger-soft
--info
/* Radius */
--radius: 6px
/* Density (set on a wrapping element) */
--row-h, --pad-y, --pad-x, --kpi-h, --kpi-h-sm
```

### Typography

- **UI font**: `Inter`, with `font-feature-settings: "ss01", "cv11"`
- **Mono / numeric**: `Geist Mono`, with `font-variant-numeric: tabular-nums` on numeric cells
- **Label style** (`.label`): `10px / 600 / uppercase / 0.08em letter-spacing / var(--fg-subtle)`
- **Numeric headlines**: tabular nums, `letter-spacing: -0.02em`, weight 600
- **Body table**: 11.5px

### Density modes

Three modes set as a class on the workspace root:

| Class | row-h | pad-y | pad-x | kpi-h | kpi-h-sm |
|---|---|---|---|---|---|
| `.density-compact` | 28 | 4 | 8 | 96 | 86 |
| `.density-cozy` (default) | 36 | 6 | 10 | 108 | 96 |
| `.density-comfortable` | 44 | 10 | 12 | 120 | 106 |

---

## Data contracts

The mocks read from `design/data.jsx`. These shapes are the contract — wire to the real APIs but keep these field names where reasonable, or document the mapping.

### Shipment
```ts
{
  id: string;              // "SHP-2026-1042"
  pro: string;             // "PRO-984512"
  bol: string;             // "BOL-2026-0042"
  customer: string;
  commodity: string;
  weight: string;          // "42,180 lb" — pre-formatted
  origin: string;          // "Long Beach, CA"
  originCode: string;      // "TERM-LA"
  dest: string;
  destCode: string;
  miles: number;
  progress: number;        // 0..100
  status: "Delivered" | "In Transit" | "At Risk" | "Detention" | "Loading" | "Pending";
  etaStatus: "ontime" | "watch" | "late" | "delivered" | "pending";
  eta: string;             // "Apr 22, 18:30" or "—"
  driver: string;          // "M. Alvarez" or "—"
  tractor: string;         // "T-1184"
  trailer: string;         // "53' DRY"
  hosLeft: string;         // "06:42" or "—"
  revenue: number;
  rpm: number;             // dollars per mile
  margin: number;          // percentage 0..100
  lastEvent: string;       // tooltip text
  lastEventAt: string;     // "12 min ago"
  lat: number; lon: number;            // current truck position
  originLat: number; originLon: number;
  destLat: number; destLon: number;
}
```

### Driver
```ts
{
  id: string;       // "D-204"
  name: string;
  tractor: string;
  hosLeft: string;
  lane: string;     // "LAX → CHI"
  lat: number; lon: number;
  status: "moving" | "dwell" | "loading" | "unassigned";
  load: string | null;  // shipment id
}
```

### Unassigned load (lighter shape for the queue)
```ts
{
  id: string; lane: string; customer: string;
  pickup: string; revenue: number; miles: number;
  equip: string; priority: "high" | "med" | "low";
}
```

The other mock shapes (`SERIES`, `ACTIVITY`, `CUSTOMERS`, `LANES`, `PICKUPS_TOMORROW`, `HOS_AT_RISK`, `SAVED_VIEWS`) are documented inline in `design/data.jsx`.

---

## Files in this bundle

```
design_handoff_shipments_command_center/
├── README.md                        ← you are here
├── phase_1_table.md                 ← start here for implementation
├── phase_2_kpi_rail.md
├── phase_3_map.md
├── phase_4_right_stack.md
├── phase_5_bottom_modules.md
├── phase_6_polish.md
└── design/
    ├── index.html                   ← entry point — open in a browser to see the full design
    ├── app.jsx                      ← page shell, KPI rail, table panel, expanded row, modules
    ├── components.jsx               ← Sparkline, Ring, Bar, StatusPill, KPI variants, NavItem, HeatCell
    ├── data.jsx                     ← all mock data + types
    ├── icons.jsx                    ← inline SVG icon set (replace with codebase's icon library)
    ├── map.jsx                      ← MapPanel: pins, routes, controls, legend
    ├── timeline.jsx                 ← timeline (gantt) view of the table
    └── tweaks-panel.jsx             ← floating Tweaks UI (Phase 6)
```

To explore the design locally: `open design/index.html` (no build needed — it's Babel-transpiled in the browser).

---

## Out of scope for this handoff

- **Backend / GraphQL schema**: shapes above are display contracts; design the real API to fit your domain.
- **Real-time transport**: how truck positions get to the client (websocket, polling, etc.) is your call.
- **Mapping library choice**: the mock fakes a map with SVG. Use whatever the codebase already integrates (Mapbox GL, MapLibre, Leaflet) — see Phase 3 for what the panel needs to do.
- **Auth / permissions**: dispatcher vs. read-only viewer is not differentiated in the mock.

---

## Questions to confirm before Phase 1

1. Does the codebase already have a `<DataTable>` primitive with sortable columns, expandable rows, and column visibility? If so, Phase 1 is mostly column config + cell renderers.
2. Are saved views server-persisted or local-only?
3. Is the table virtualized for >1k rows? (Mock assumes <500.)
4. Does an `Inter` + `Geist Mono` font stack already exist in the build? (Both are free Google fonts.)

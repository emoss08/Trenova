# Phase 2 — KPI Rail

A 12-column rail of purpose-built KPI cards above the table. **Five card variants** — pick the right shape for the metric instead of forcing every KPI into one card.

## Reuse from the codebase first

- `client/src/components/ui/Card` — base card chrome (border, radius, background)
- `client/src/components/ui/Tooltip` — for sparkline / segment hover
- `client/src/lib/format.ts` — currency, percentage, deltas
- `client/src/components/charts/Sparkline` — if it exists, prefer it over the mock's

## The 5 card variants

All variants share a common header (`<KpiHeader>`): label on the left (uppercase 10px, with optional 11px icon), Delta chip on the right. See `design/components.jsx` for reference implementations.

### 1. `<KpiHero>` — bigger number + segmented breakdown bar + sparkline footer
**Use for:** the 1-2 most important numbers (revenue, active shipments).
**Span:** 3 cols. Height: `var(--kpi-h)`.
- 28px number, mono tabular, weight 600
- Optional `breakdown={[{label, value, color}]}` → renders a stacked horizontal bar with a 4-item legend below
- Optional `sparkData` → 88×24 sparkline in the footer (mutually exclusive with breakdown)
- Sub line at 10.5px `--fg-subtle`

### 2. `<KpiRing>` — value + filled ring + target
**Use for:** percentages with a target (on-time %, tender accept %).
**Span:** 2 cols. Height: `var(--kpi-h)`.
- 42×42 ring on the left, value beside it (22px mono)
- Ring color: `success` when at/above target, `warning` when below
- "Target 96%" line in 9.5px uppercase `--fg-subtle`

### 3. `<KpiGoalBar>` — actual vs. target as horizontal bar
**Use for:** ratios where lower is better (empty mile %).
**Span:** 2 cols. Height: `var(--kpi-h)`.
- 22px number
- 6px-tall fill bar with a 2px tick marker at the target position
- Fill color: `success` when actual ≤ target, else `warning`

### 4. `<KpiStat>` — compact number-forward, no chart
**Use for:** simple counts where the number IS the message (at-risk, unassigned, ready to dispatch).
**Span:** 2 cols. Height: `var(--kpi-h-sm)` (smaller).
- Tone-colored 6px dot before the icon to indicate severity (`danger`, `warning`, `brand`)
- 26px number
- Delta chip on the right of the header

### 5. `<KpiWatchlist>` — stacked mini-list of items
**Use for:** when individual rows matter more than a single number (HOS near limit, detention dwell).
**Span:** 3 cols. Height: `var(--kpi-h-sm)`.
- 3 mini rows, each: tone dot + identifier (mono 11px) + meta time (mono 10.5px tone-colored)
- First row gets a faint `color-mix(--fg 4%)` background as "most urgent"
- Header right shows total count

## Layout — the 12-col grid

```
┌────────── HERO 3 ─────────┬────────── HERO 3 ─────────┬─ RING 2 ──┬─ GOAL 2 ──┬─ RING 2 ──┐
│ Revenue today  $36.4K     │ Active shipments  142     │ On-time   │ Empty mile│ Tender    │
│ ▆▆▆▆▆▆▆▆ sparkline        │ ▰▰▰▰▰░░░░ breakdown bar   │ 94.2% O   │ 11.8%  ━━ │ 94.1% O   │
│ RPM $2.18 · margin 22.4%  │ in-transit · at-risk ...  │ tgt 96%   │ tgt 10%   │ tgt 95%   │
└───────────────────────────┴───────────────────────────┴───────────┴───────────┴───────────┘
┌─ STAT 2 ──┬─ STAT 2 ──┬─ STAT 2 ──┬───── WATCHLIST 3 ─────┬───── WATCHLIST 3 ─────┐
│ At-risk   │Unassigned │Ready disp.│ HOS near limit        │ Detention dwell > 2h  │
│ ● 9       │ ● 5       │ ● 12      │ • D-211 J. Park  2:15 │ • SHP-1040  3h 38m    │
│ 4 ETA slip│ $8,650 wait│ 5 unassgn │ • D-176 K.W.    4:30 │ • SHP-1041  2h 22m    │
└───────────┴───────────┴───────────┴───────────────────────┴───────────────────────┘
```

CSS:
```css
.kpi-rail {
  display: grid;
  grid-template-columns: repeat(12, minmax(0, 1fr));
  gap: 8px;
  padding: 4px 16px 12px;
}
.kpi-card { grid-column: span var(--span); /* 2 or 3 */ }
```

## Visibility toggle

The whole rail is hidden when `tweaks.showKpis === false` (Phase 6). Build it as a single component that the page wraps in `{tweaks.showKpis && <KpiRail />}`.

## Time window

The rail accepts `timeWindow: "today" | "24h" | "7d"`. For Phase 2, just propagate it down — the actual data swap can stub against different SERIES arrays. In production, this becomes a query param to the analytics endpoint.

## Acceptance criteria

- [ ] All 5 variants render with correct hierarchy
- [ ] No left-edge accent bars (the old design had these — they read as form errors)
- [ ] Ring color flips success↔warning at the target threshold
- [ ] Goal bar tick lands exactly at `target/max * 100%`
- [ ] Watchlist's first row has the subtle highlight
- [ ] Delta chip color follows `deltaTone` prop, not just sign — sometimes a negative delta is good (empty mile % going down)
- [ ] Sparkline renders crisp at 1× and 2× DPR
- [ ] Cards respect density tokens (`--kpi-h` / `--kpi-h-sm`)

## Files to study

- `design/components.jsx` lines 73–195 — all 5 KPI components, `KpiHeader`, `Delta`, `SegmentedBar`
- `design/app.jsx` lines 242–323 — `KpiRail` composition + per-card props
- `design/data.jsx` `SERIES` — sparkline data shapes

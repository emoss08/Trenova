# Phase 5 — Bottom Modules

Lower-priority analytical / collaborative context that lives below the table.

Three modules:

1. **Lane Heatmap** — a 4×4 origin-region × dest-region grid showing load volume
2. **Customer Mix / Tomorrow's Pickups** — tabbed module
3. **Activity Feed** — recent system + user events

Layout: 3-column grid below the table, equal widths.

## Reuse from the codebase first

- `client/src/components/ui/Tabs` — for the Customer Mix / Pickups toggle
- `client/src/components/ui/Avatar` — for activity feed authors
- `client/src/lib/format.ts` — relative time

## Lane Heatmap

A 4×4 grid: rows = origin region, columns = dest region. Each cell shows the load count, with cell background opacity proportional to count.

```
       │  West  Midwest  South  Northeast
─────────────────────────────────────────────
West   │   ·     14       8       3
Midwest│   9      ·       11      6
South  │   4      7       ·       12
Northe.│   2      5       9       ·
```

- Cell: 26px tall, fontSize 11, mono tabular
- Background: `color-mix(in oklch, var(--brand) {t * 100}%, transparent)` where `t = value / max`
- Text color flips white when `t > 0.55`, else `--fg`
- Diagonal cells (same region in/out): `·` placeholder
- Border: `1px solid var(--border-2)` on each cell

Click a cell → set `selectedView` to a synthetic "Lane: West→Midwest" view and scroll the table. Tooltip on hover shows full count + percent of total.

Use the existing `<HeatCell>` from `design/components.jsx` as reference.

## Customer Mix / Tomorrow's Pickups

Tabbed card. Default tab: **Customers**.

### Customers tab

Top 5 customers by today's revenue:
```
Acme Manufacturing       $41,200    34%   18 loads   ▲ 1.2pp
FreshHaul Foods          $22,800    19%   11 loads   ▼ 0.4pp
Range Logistics          $18,300    15%    9 loads   ▲ 0.8pp
GlobalTrade Inc.         $14,600    12%    7 loads   —
Peak Distribution        $12,400    10%    6 loads   ▲ 2.1pp
```

- Customer name: 11.5px weight 500
- Revenue: 11px mono tabular weight 600
- Share + loads: 10px `--fg-subtle`
- Trend: delta chip
- Optional: a 4px-tall horizontal bar showing share-of-revenue under each row

### Tomorrow's Pickups tab

Time-sorted list of tomorrow's scheduled pickups. Each row:
```
06:00  Acme Manufacturing       TERM-LA → DC-CHI    M. Alvarez   [scheduled]
07:30  FreshHaul Foods          COLD-SEA → DC-PDX   L. Mendez    [scheduled]
...
```

Status pills: `scheduled` (muted), `confirmed` (success), `tentative` (warning).

## Activity Feed

Stream of recent events. Each item:
```
[● sev] {time}   {text}                 {who}
```

- Severity dot color per `sev`: `danger`, `success`, `brand`, `info`, default → `--fg-subtle`
- Time (11px mono): "now", "2 min", "14 min", "1h" — relative
- Text: 11.5px, supports @mentions in `--brand`
- Author handle: 10.5px mono `--fg-subtle`
- Hover row: faint background tint, "reply" action appears
- Click a system alert → expand the related shipment in the table

Cap visible at 8–10; "Show more" expands. Internal scroll with `max-height: 240px`.

## Acceptance criteria

- [ ] Heatmap cells show correct intensity, diagonal is muted
- [ ] Heatmap click jumps to filtered table view
- [ ] Tabs swap content without re-mounting the parent card
- [ ] Activity feed shows last 10 items, with proper severity dots
- [ ] @mentions render with brand color and (eventually) link to the user

## Files to study

- `design/app.jsx` `LaneHeatmap` (~982), `CustomerMix` (~1022), `ActivityFeed` (~950)
- `design/components.jsx` `HeatCell`
- `design/data.jsx` `LANES`, `CUSTOMERS`, `PICKUPS_TOMORROW`, `ACTIVITY`

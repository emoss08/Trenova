# Phase 4 — Right-Stack Modules

Three stacked modules to the right of the map: **Unassigned Queue**, **Exceptions Inbox**, **HOS Watch**. Reorderable, hideable, persistent.

## Reuse from the codebase first

- `client/src/components/ui/Card` — module chrome
- `client/src/components/ui/DropdownMenu` — for "Add panel" picker
- `client/src/components/ui/ScrollArea` — internal scrolling
- `client/src/hooks/useDragAndDrop` (or `dnd-kit`) — for drag-to-assign + drag-to-reorder

If `dnd-kit` is already in the project, use it. The mock uses native HTML5 drag-and-drop — fine for a prototype, less ergonomic in production for keyboard accessibility.

## Shared module chrome

Each module is a `<Card>` with a header and a body that internally scrolls when content overflows. Total stack height matches the map panel's height (so the table can sit cleanly below). Internally each module has a fixed share of that height (e.g. 33% each), with the card body using `overflow-y: auto`.

Header structure:
```
[≡ drag handle] Module title    [— hide]
```

- Drag handle: 11px icon, cursor `grab`, hover background slight tint. Initiates panel reorder.
- Title: 11.5px / weight 600
- Right side: count chip (e.g. "5") + the hide button (`−` icon, ghost button)

When ALL modules in the stack are hidden, show an empty placeholder with a centered button: **"+ Add panel"** that opens a dropdown listing the hidden modules. When SOME are hidden, show a small "+ Add panel" link in the top-right of the stack.

State to persist (localStorage):
```ts
{
  "trenova.rightStack.order": ["unassigned", "exceptions", "hos"],
  "trenova.rightStack.hidden": []
}
```

## Module 1 — Unassigned Queue

List of unassigned loads (`UNASSIGNED` shape in `data.jsx`). Each item is **draggable** and can be dropped on a driver row in the HOS Watch (or in the future, on the map / timeline).

Each item:
```
SHP-2026-1035   [HIGH]
TERM-HOU → CUST-NOL
Acme Manufacturing · Apr 22, 14:00
$1,850 · 348mi · 53' DRY
```

- Identifier: 11px mono weight 600
- Priority pill: `pill-soft-danger` for "high", `pill-soft-warning` for "med", `pill-soft-muted` for "low"
- Lane: 11.5px, mono codes
- Meta: 10px `--fg-subtle`
- Drag visual: while dragging, opacity 0.5, cursor `grabbing`. The drop targets get `outline: 2px dashed var(--brand); outline-offset: -2px; background: var(--brand-soft)`.

When dropped on a driver, fire `onAssign({shipmentId, driverId})` and add the shipment id to `recentlyAssigned` so it can flash green for 1.5s before disappearing from the queue.

## Module 2 — Exceptions Inbox

Filterable list of shipments needing dispatcher attention (at-risk, detention, reefer alerts, weather slips, missing docs).

Each item:
```
[icon] {SHP-id}   {summary}
       {customer} · {time-ago}    [chip: severity]
```

- Severity → border-left color in the row (use `box-shadow: inset 3px 0 var(--danger)` to avoid the "form-error" feel of a separate accent div)
- Click → expand the matching row in the table (mark exception as triaged)
- "Triage" + "Snooze" actions on hover

Tab strip at the top filters by category: `All` | `ETA slip` | `Reefer` | `Weather` | `Missing docs`.

## Module 3 — HOS Watch

Drivers nearing HOS limits. Each row:
```
[● tone] D-211 J. Park    02:15 left    [SHP-2026-1041]
         Force 30m break by 19:45
```

- Sorted by `hosLeft` ascending (most urgent first)
- Row is a **drop target** for the unassigned queue — drag a load onto a driver to assign it
- Click → expand the driver's current shipment in the table

## Reordering behavior

- Drag a header up/down to reorder. While dragging, show a 2px `--brand` line above the slot the dragged module would land in.
- Reorder triggers persistence + animation (50ms slide of siblings).
- Keyboard support (a11y): focused header, `Space` to grab, `↑/↓` to move, `Space` to drop, `Esc` to cancel.

## Acceptance criteria

- [ ] All three modules render with the same chrome
- [ ] Internal scroll works without breaking the outer layout
- [ ] Drag a load from Unassigned onto a driver in HOS → assignment fires + flash
- [ ] Drag a header to reorder, persists across reload
- [ ] Hide / Add panel works, persists across reload
- [ ] When all panels hidden, the "Add panel" empty state is centered and obvious
- [ ] Stack height matches map height; table sits flush below

## Files to study

- `design/app.jsx` `UnassignedQueue` (line ~473), `HosWatch` (line ~573), and the right-stack composition in the page-level layout
- `design/data.jsx` `UNASSIGNED`, `DRIVERS`, `HOS_AT_RISK`

# Workspace rail redesign

A proposal to replace the Workspace and Classic layouts with one shell. The prototype lives on
this branch as a third `sidebarVariant` (`rail`) so it can be compared against both in the same
session; every screenshot in `workspace-rail/` is the real app rendered against the mock API
(`client/apps/web/e2e/mock-api`), not a mockup.

## The shape

```
┌────┬──────────────┬──────────────────────────────────────────────┐
│ TL │ Search    ⌘K │ ⇤ ‹ › Shipments ⓘ            ⟳ [New shipment] ☆│  ← the page's own 44px bar
│────│ Shipments    │──────────────────────────────────────────────│
│ ⌂  │  Shipments   │                                              │
│ 💬3│  Recurring   │                                              │
│ 📥 │  Orders      │                page                          │
│ ⬚  │  Service f. 2│                                              │
│────│ Configuration│                                              │
│ 🚚2│  Shipment t. │                                              │
│ ⋯  │  ...         │                                              │
│────│              │                                              │
│ 🔔 │              │                                              │
│ ⚙  │              │                                              │
│ 👤 │              │                                              │
└────┴──────────────┴──────────────────────────────────────────────┘
  48px   224px (200–320, dragged)
```

- **The rail** never folds and never scrolls the page: organization mark at the top, one tile
  per module in its domain with hairlines between domains, settings and you at the bottom. The
  tile you are in is filled and marked with an ink bar on the rail's edge; a count on a tile's
  corner is how much work waits behind it, in the attention tone.
- **The panel** holds search and the module's pages, grouped the way the module groups them,
  configuration last; on Home it lists what needs you, pins and recent pages. It folds with ⌘B
  and the fold button on the page bar, and its width is dragged from its edge and remembered.
- **The page bar** is the page's own title bar: fold, history, trail, title, description,
  actions, then the star behind a hairline. There is no global header. The trail skips Home and
  any crumb that repeats the title.

## What it buys

| Measure | Workspace | Classic | Rail |
| --- | ---: | ---: | ---: |
| Chrome above the page body | 116px | 84px | 44px |
| Chrome to the left, panel open | 224px | 256px | 272px |
| Chrome to the left, folded | 0px | 0px | 48px |
| Clicks to any other module | 2 | 1–2 | 1 |

72px back at the top is about 2.4 rows at the comfortable row height. The 48px spent on the
rail is deliberate: folding today loses the product map; folding here keeps every module one
hover away.

## New behaviour

- Attention counts on the rail tiles, from the same `moduleAttention` map the modules menu used.
- With the panel folded, resting on a tile opens that module's page list as a flyout.
- `G` then a letter jumps to a module (`G S` Shipments, `G B` Billing, `G H` Home); the letter
  shows in the tile's tooltip and the chord is ignored while a field has focus.
- The panel resizes between 200 and 320px by drag, arrow keys, and double-click to reset.
- Below 768px the rail and panel come out together as one sheet.

## Left alone

Triage and the Desk are full-screen rooms and draw their own rail; a room goes through today's
shell unchanged. Home's greeting bar and the admin header draw their own top bar and are
untouched. The palette, settings dialog, notification sheet and assistant dock are reused as they
are; only their triggers moved onto the rail.

## Screenshots

| | |
| --- | --- |
| ![Board, light](workspace-rail/board-light.webp) | ![Board, dark](workspace-rail/board-dark.webp) |
| ![Detail, light](workspace-rail/detail-light.webp) | ![Detail, dark](workspace-rail/detail-dark.webp) |
| ![Folded](workspace-rail/folded-dark.webp) | ![Flyout](workspace-rail/flyout-light.webp) |
| ![Tooltip](workspace-rail/tooltip-dark.webp) | ![List page](workspace-rail/list-light.webp) |
| ![Formula templates](workspace-rail/formula-templates-dark.webp) | ![Home](workspace-rail/home-light.webp) |
| ![AI control](workspace-rail/ai-control-dark.webp) | ![Wide panel](workspace-rail/wide-panel-light.webp) |
| ![1000px](workspace-rail/narrow-1000-light.webp) | ![Palette](workspace-rail/palette-dark.webp) |
| ![Layout menu](workspace-rail/layout-menu-light.webp) | ![Phone sheet](workspace-rail/phone-sheet-dark.webp) ![Phone page](workspace-rail/phone-light.webp) |

Before, for comparison: ![Workspace today](workspace-rail/before-workspace-light.webp)
![Classic today](workspace-rail/before-classic-light.webp)
![Modules menu today](workspace-rail/before-modules-menu.webp)

## Reproducing the screenshots

```sh
cd client/apps/web
MOCK_PALETTE=1 node e2e/mock-api/server.mjs   # terminal 1
pnpm exec vite                                # terminal 2
```

Then set `navigation-storage` in local storage to
`{"state":{"sidebarVariant":"rail","sidebarCollapsed":false},"version":0}`, or pick Rail under
Sidebar layout in the user menu. The mock returns no sidebar preferences and an empty attention
summary, so the rail counts were captured with those two operations answered from the capture
script.

## Where the prototype lives

`client/apps/web/src/components/navigation/variants/rail/`: `module-rail.tsx`, `rail-panel.tsx`,
`rail-page-bar.tsx`, `rail-shell.tsx`, `rail-shortcuts.ts`, `use-go-to-chords.ts`. The store
gained `railPanelWidth`; `PageHeader` gained `leading` and `trailing` slots the shell fills;
`NotificationSheet` takes a `className` so the bell can be a rail tile.

## Open decisions

1. Ship Rail as the only layout and retire Workspace and Classic (recommended), or keep three.
2. Which modules carry a chord letter; People, Payroll, Detention and EDI have none for now.
3. Whether the bell stays on the rail or returns to each page bar beside the star.
4. Whether the panel keeps its quiet module heading or drops it.

## What implementing takes

- Make Rail the default and remove the other two shells, their preference and their tests.
- Give Home's greeting bar and the admin header the page bar's leading slot.
- Tests for the chords, the resize bounds, the breadcrumb trimming, the flyout only when
  folded, and the sheet never opening on first paint.
- Translations for the new strings, the chords in the keyboard shortcuts dialog, the
  design-system "Page anatomy" section, and a product-guide regeneration.

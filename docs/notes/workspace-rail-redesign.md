# Workspace rail redesign

The design note for the shell that replaced the Workspace and Classic layouts. It was first
built as a third layout so the three could be compared in one session, then shipped as the only
one; the Workspace and Classic shells, their preference and the modules dropdown on pages are
gone. Every screenshot in `workspace-rail/` is the real app rendered against the mock API
(`client/apps/web/e2e/mock-api`), not a mockup. The living description is "Shell anatomy" in
[design-system.md](../engineering/design-system.md).

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

The mock returns no sidebar preferences and an empty attention summary, so the rail counts were
captured with those two operations answered from the capture script.

## Where it lives

`client/apps/web/src/components/navigation/`: `rail-shell.tsx`, `module-rail.tsx`,
`rail-panel.tsx`, `home-panel.tsx`, `rail-page-bar.tsx`, `rail-shortcuts.ts`,
`use-go-to-chords.ts`; `room-header.tsx` is the header a room keeps. The store holds
`railPanelWidth`; `PageHeader` has `leading` and `trailing` slots the shell fills;
`NotificationSheet` takes a `className` so the bell can be a rail tile. The chords are listed
in `config/keybinds.config.ts` so the shortcuts dialog shows them.

## Still open

- People, Payroll, Detention and EDI carry no chord letter.
- Rooms (Triage, the Formula Studio) sit beside the module rail too; the rail carries their
  fold tile, and on a phone a room gets one strip that opens the module sheet.

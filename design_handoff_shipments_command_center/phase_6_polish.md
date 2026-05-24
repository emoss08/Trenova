# Phase 6 — Polish: Tweaks panel, theme, density, layouts

Cross-cutting concerns to layer on once the surface is stable.

## Tweaks panel (floating)

A floating panel at the bottom-right of the workspace lets the user toggle:

- **Theme** — Light / Dark
- **Density** — Compact / Cozy / Comfortable (sets `density-*` class on the workspace root)
- **Layout** — `Multi-pane` (default — KPIs + map + right-stack + table) / `Table-first` (table hero, map collapsed) / `Map-first` (map hero, table collapsed)
- **Map style** — Street / Satellite / Dark tactical
- **Show/hide KPI cards** — `tweaks.showKpis` boolean
- **Time window** — Today / 24h / 7d
- **Accent hue** — degrees on the OKLCH hue wheel (slider 0–360, default 263 = brand blue)

Only show the panel when "Tweaks" is toggled on in the toolbar — see the protocol in the design system.

In production, most of these become **user preferences** persisted server-side (theme, density, layout). Map style + time window + accent hue are demo-only — the real product probably ships a single accent color and only allows theme / density / layout switching in user settings.

## Theme toggle

Light vs. dark — `class="dark"` on the workspace root or `<html>`. All tokens in `client/src/styles/app.css` already have light + dark variants, so this is a one-line toggle.

Persist to `localStorage["trenova.theme"]` and respect `prefers-color-scheme` on first load when no preference is set.

## Density modes

```css
.density-compact { --row-h: 28px; --pad-y: 4px; --pad-x: 8px; --kpi-h: 96px; --kpi-h-sm: 86px; }
.density-cozy    { --row-h: 36px; --pad-y: 6px; --pad-x: 10px; --kpi-h: 108px; --kpi-h-sm: 96px; }
.density-comfortable { --row-h: 44px; --pad-y: 10px; --pad-x: 12px; --kpi-h: 120px; --kpi-h-sm: 106px; }
```

The class goes on the workspace root. Every component reads its sizing from these CSS variables — **do not hardcode pixel values** in cells, KPI cards, or module headers.

## Layout variants

Three top-level grid templates:

```css
/* multi-pane (default) */
.layout-multi {
  grid-template-columns: 1fr minmax(320px, 380px);
  /* row 1: map | right-stack ; row 2: table spans both cols */
}

/* table-first */
.layout-table {
  /* table is full-width hero; map collapses to a 200px strip on the right */
}

/* map-first */
.layout-map {
  /* map fills 60vh hero; table is below as a collapsible drawer */
}
```

Persist to `localStorage["trenova.layout"]`.

## Accent hue

Override `--brand` and `--brand-soft` via inline style on the root:

```ts
const hue = tweaks.accentHue ?? 263;
root.style.setProperty('--brand', `oklch(0.55 0.22 ${hue})`);
root.style.setProperty('--brand-soft', `oklch(0.55 0.22 ${hue} / 0.10)`);
```

Tweaks-only — production ships a single brand value.

## Saved-view persistence

`selectedView` should be persisted server-side per user (not localStorage) once the auth model is in place. For the prototype, localStorage is fine.

## Acceptance criteria

- [ ] Theme toggle flips all tokens with no flash of wrong theme on load
- [ ] Density change updates row height + KPI height live with no layout shift artifacts
- [ ] Layout switch persists and restores correctly
- [ ] Tweaks panel toggle in toolbar shows/hides the panel cleanly
- [ ] All Tweaks values persist to localStorage (or user prefs)
- [ ] Accent hue updates `--brand` everywhere it's referenced

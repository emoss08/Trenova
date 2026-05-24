# Phase 3 — Live Map Panel

A live map showing trucks, routes, origin/dest pins, with hover-sync to the table.

## Reuse from the codebase first

The mock fakes a map with SVG. **In production, use whatever mapping library the codebase already integrates.** Likely:

- `client/src/components/Map` — if there's a wrapper around Mapbox GL / MapLibre / Leaflet
- `client/src/lib/geo.ts` — projection / distance helpers

If nothing exists yet, **Mapbox GL JS** or **MapLibre GL** is the recommended choice — both support custom layers, fast pin rendering, and dark-mode styles.

## What the panel must do

1. **Plot every active shipment** as either:
   - a route polyline from `(originLat, originLon)` → `(lat, lon)` → `(destLat, destLon)` (current segment solid, completed dashed)
   - a truck pin at `(lat, lon)`
2. **Plot pickup + delivery pins** at origin/dest of each shipment (small dots with hover label)
3. **Color trucks/routes** by `status` / `etaStatus`:
   - moving / in-transit → `--brand`
   - at-risk → `--danger`
   - dwell / detention → `--warning`
   - delivered → `--success`
   - loading → `--info`
4. **Hover sync with the table:**
   - When the user hovers a row, `highlightId` becomes that shipment's id; the map should pulse / enlarge that pin and pan-to (without zooming)
   - When the user hovers a pin, the same `highlightId` is set so the table row highlights
5. **Click a pin** → `onSelect(shipmentId)` (caller decides — typically expand the row in the table)
6. **Map style toggle** (top-right control): `street` | `satellite` | `dark-tactical`. Persist to `tweaks.mapStyle`.
7. **Legend** (bottom-left, fixed): chips for the 5 status colors. Constrained to bottom-left, doesn't extend under the map controls. Has a border so it reads as a discrete chip strip.

## Panel chrome

- Card with no padding, fixed height that responds to layout mode:
  - `large` (map-first layout): 520px
  - default (split layout): 420px
- Border and radius from the `Card` primitive
- Header bar (top, 36px): title "Live Map", count chip ("9 at-risk · 58 in-transit"), spacer, then map-style toggle, then full-screen button
- Body: the map fills remaining height
- Bottom-left overlay: legend (see above)

## Drawing the routes correctly

A common bug in the early mock: every route drew from a hard-coded LA→Chicago endpoint, so trucks based in Cleveland visually started in California. **Each route must use the shipment's actual `originLat/Lon` → `destLat/Lon`.** Don't reuse one set of coordinates across rows.

## Acceptance criteria

- [ ] Each shipment renders at its actual coordinates
- [ ] Hovering a table row pulses + pans to the matching pin
- [ ] Hovering a pin highlights the matching table row
- [ ] Clicking a pin expands the table row
- [ ] Map style toggle works and persists
- [ ] Legend stays in bottom-left and doesn't overlap controls

## Files to study

- `design/map.jsx` — the mock implementation (SVG-based; informational for the hover/select contracts)
- `design/app.jsx` `MapPanel` invocation — props it expects

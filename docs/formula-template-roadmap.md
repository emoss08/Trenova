# Formula Template Roadmap

Working checklist for the next block of formula template improvements, following the
2026-08-31 hardening + Formula Studio effort. Items are checked off as they are
implemented **and verified** (build + tests green).

## #2 — Standard templates for production organizations

The seeder is development-only, so production organizations start with zero templates.
An idempotent install endpoint gives every org the vetted standard library.

- [x] Move the standard template catalog to an embedded, runtime-usable location
      (single source of truth shared with the dev seeder) —
      `formulatemplateservice/standardcatalog`
- [x] `POST /formula-templates/install-standards` — idempotent by name: creates
      missing standards (validated, Active, version snapshot, audit), skips existing
- [x] Client: "Install Standard Templates" action on the table with confirm dialog
- [x] Tests: installs missing / skips existing / fills only gaps; every standard
      template compiles against the shipment schema (validated at install)

## #3 — Saved test scenarios per template ("formula unit tests")

Named input/expected pairs stored with the template, re-runnable on every edit, and
enforced at approval time.

- [x] Domain + migration: `formula_template_test_cases` (name, description, variable
      values, expected amount, tolerance) — Postgres + SQLite mirrors
- [x] Repository (port + postgres) with tenant scoping
- [x] Service: test-case CRUD (audited) + `RunTestCases` against saved content or
      candidate editor content (guardrails included)
- [x] Approval gate: Approve fails when the template has scenarios and any fail
- [x] Handler routes: list/create/update/delete/run
- [x] Client: Scenarios view in the studio (run-all with pass/fail results,
      add/edit/delete, "save current preview as scenario")
- [x] Tests: run pass/fail/tolerance/guardrail semantics, candidate override,
      approval gate (blocked + green)

## #4 — Date/time and lane variables

The schema has no temporal or geographic data, so weekend/after-hours surcharges and
lane-based pricing cannot be expressed.

- [x] Computed temporal variables from the first pickup stop: `pickupDayOfWeek`,
      `pickupHour`, `pickupMonth`, `isWeekendPickup` (UTC — locations carry no
      timezone; documented in the schema descriptions)
- [x] Lane variables from first/last stop locations: `origin.city/state/zip`,
      `destination.city/state/zip` (never error: missing stops/locations yield
      empty values so mid-entry shipments still price)
- [x] Schema properties + preloads so rating callers hydrate stop locations
      (`Moves.Stops.Location` + `.State` added to x-data-source preloads)
- [x] Client fallback variable list + sample test data updated (Origin/Destination
      categories + temporal defaults)
- [x] Tests: each computed resolver (actuals vs scheduled, missing location, ordering)

## #5 — Multi-axis lookup

`lookup()` only addresses single-axis matrices; class tariffs (weight break × zone)
need two keys.

- [x] Engine: `lookup2(table, rowKey, colKey)` and `lookup2Or(...)` with per-axis
      exact/range matching (all four mode combinations; half-open bands)
- [x] Rate matrix loading extended to two-axis matrices for the lookup provider
      (`GetLookupData` axis filter now `IN (1, 2)`; three-plus-axis still excluded)
- [x] Save-time validation understands the new functions (`ExtractLookupTableRefs`
      separates arities; arity-mismatch errors point at the right function);
      reserved-name guard extended to `lookup2`/`lookup2Or`
- [x] Function metadata + docs + client fallback list updated (flows through the
      schema endpoint automatically; AI assistant prompt/table list annotated too)
- [x] Tests: two-axis exact/range matching, misses, validation, engine integration

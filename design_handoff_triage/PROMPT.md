You are implementing the Triage redesign (which replaces `/inbox`) in the Trenova repo (emoss08/Trenova, branch master).

Read, in this order:
1. `/CLAUDE.md` and `/AGENTS.md`. Follow every convention in them. Before touching those areas, also read: `docs/engineering/design-system.md`, `generated-artifacts.md`, `row-level-security.md`, `realtime.md`, `idempotency.md`, `product-guide.md` and `agent-write-coverage.md`.
2. `design_handoff_triage/README.md`. It is the spec.
3. `design_handoff_triage/design/index.html`, opened in a browser. Use the Tweaks panel to see every state (live, first day, no mailbox, desk off, end of day, phone, light theme). It is a reference, not code to copy.

**Authority.** You have full control of the codebase: frontend, GraphQL schema, Go domain, ports, services, repositories, workers, Temporal jobs, migrations (Postgres and SQLite) and seeds. Many features in README §6 need new backend work. Build it. My one expectation is that the result **matches the design**. When the existing code and the design disagree, change the code. When a server rule makes the design impossible, keep the rule and write the conflict in `docs/triage-redesign-notes.md`.

**Engineering bar.** This is a production, multi-tenant application, so build for scale.
- **DRY and SOLID, strictly.**
  - One triage-actions hook serves the buttons, keyboard and context menus.
  - One context-menu component serves every surface.
  - One rule evaluator is shared by the processor, the preview and the suggestions.
  - Shared UI and utilities go in `@trenova/shared`; backend utilities go in `shared/`.
  - Follow the hexagonal layout. Do not create a parallel architecture.
- **Frontend performance:**
  - lazy-load routes and heavy panels;
  - virtualize every long list with `src/components/virtual-rows.tsx`;
  - TanStack Query with prefetch on hover and for the next J/K message;
  - optimistic updates with rollback that power Undo;
  - SSE patches the cache;
  - memoize rows, use stable callbacks and split contexts so typing never re-renders the list;
  - animate transforms and opacity only, pause animation off-screen, and respect `prefers-reduced-motion`.
- **Backend performance:**
  - indexed cursor queries (due, received, lane, owner, thread, snooze);
  - dataloaders for every per-row field, and no N+1;
  - one `CountBreakdown` aggregate;
  - a per-tenant Redis cache for the Briefing aggregate, rule previews and sender hints, invalidated on write;
  - background work (rules, conflicts, event snoozes, reply-waits) runs in idempotent workers or Temporal and never in the list's request path;
  - every new tenant table is under RLS.
- **Design rules:**
  - tokens only; `pnpm lint:design` passes;
  - icons from `@trenova/shared/components/icons` (draw missing ones into the custom set);
  - copy through `useT()`;
  - keys through `src/config/keybinds.config.ts`;
  - floating surfaces fully opaque;
  - follow the Nova vs Desk rules in README §1.
- **Every new capability ships with:**
  - a domain type and ports;
  - a service;
  - schema and resolver (`task gqlgen`), then `pnpm --filter @trenova/graphql codegen`;
  - migrations (up and down, both databases);
  - an authz check;
  - write-coverage;
  - tests next to the existing ones;
  - seeds updated so the dev data shows every state in the prototype.

Work in this order and keep master green after each step:
1. Backend contracts and data: threads, due/targets, owner, snooze (time and event), waiting-on-them, rules and the evaluator, corrections, conflicts, source health, the Briefing aggregate, reply send and templates, presence.
2. The list and reader rebuilt on the new contracts: virtualized list, reader, actions panel, action bar, keys, the shared context menu.
3. Briefing, including first day and end of day.
4. Rules.
5. Peeks, the document viewer, the Desk hand-off, connect a source.
6. Product guide regeneration, the `/inbox` → Triage redirect, and the light-theme and narrow-screen pass.

Before each step, list the files you will touch. After each step run:
- the client `typecheck`, `lint`, `lint:design` and `test`;
- `task test` and `task lint` in `services/tms`;
- `task gqlschema-diff`.

Profile the queue with a seeded tenant of at least 50k messages, and record list query times and render times in the notes file.

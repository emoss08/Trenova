You are implementing the new Analytics tab in AI control in the Trenova repo (emoss08/Trenova, branch master).

Read, in this order:
1. `/CLAUDE.md` and `/AGENTS.md`. Follow every convention in them. Also read `docs/engineering/design-system.md`, `generated-artifacts.md` and `row-level-security.md`.
2. `design_handoff_analytics/README.md`. It is the spec.
3. `design_handoff_analytics/design/index.html?tab=analytics`, opened in a browser, and `design_handoff_analytics/screenshots/`. Hover chips and bars, switch every control, open both sheets, and set Tweaks › Providers to "none". It is a reference, not code to copy.

**Authority.** You may change the frontend, GraphQL schema, Go domain, ports, services, repositories, workers and migrations (Postgres and SQLite). The result must **match the design**. When a server rule makes the design impossible, keep the rule and write the conflict in `docs/ai-analytics-notes.md`.

**Engineering bar.** Production, multi-tenant.
- DRY and SOLID. Metric definitions (README §5) live once on the server; the client only formats. One highlight context serves every chart. One `analytics-model.ts` builds the sentences and Nova summary.
- Follow the hexagonal module layout in `internal/modules/agent/`. Don't create a parallel architecture.
- Backend: aggregates from indexed queries over runs, steps and usage (tenant, time, provider, model, agent, task); no N+1; a per-tenant Redis cache for the aggregate keyed by scope, short TTL, invalidated on new runs. If raw-row aggregation over 30 days is slow, add a rollup table filled by an idempotent worker. Every new tenant table is under RLS.
- Frontend: lazy-load the tab; TanStack Query keyed by scope with prefetch on hover of range buttons; memoize charts; animate transforms and opacity only; respect `prefers-reduced-motion`; charts resize with a ResizeObserver.
- Design rules: tokens only, `pnpm lint:design` passes; icons from `@trenova/shared/components/icons`; copy through `useT()`; floating surfaces fully opaque; scope held in the URL.
- Every new capability ships with domain type, ports, service, schema and resolver (`task gqlgen`, then `pnpm --filter @trenova/graphql codegen`), migrations up and down for both databases, an authz check, tests next to existing ones, and seeds that show every state in the prototype (including a same-model-on-two-providers pair, a failing local provider, and a turned-off deployment).

Work in this order and keep master green after each step:
1. Telemetry: tool-call cause and latency; model-call TTFT, provider error kind, fell-to-next.
2. The `AgentAnalytics` query, tool-failure examples query, provider events and CSV export.
3. Tab shell: Nova summary, scope bar, figures, comparing chips, empty states.
4. Leaderboard and trade-offs (scatter, same model different provider, "Put X first").
5. Run time and over time.
6. Tool calls (heatmap, whose mistake) and routing and judgment (task fit, proposals, provider health), plus both sheets.
7. Light theme and narrow-screen pass; product guide regeneration.

Before each step, list the files you will touch. After each step run the client `typecheck`, `lint`, `lint:design` and `test`; `task test` and `task lint` in `services/tms`; and `task gqlschema-diff`.

Profile the aggregate on a seeded tenant with at least 500k runs over 30 days and record query times in the notes file.

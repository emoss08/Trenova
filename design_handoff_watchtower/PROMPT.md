You are implementing the redesigned **Watchtower** page of Desk in the Trenova repo (emoss08/Trenova, branch master). The rest of Desk v2 is already implemented. Touch only Watchtower and whatever it needs.

Read, in this order:
1. `/CLAUDE.md` and `/AGENTS.md`. Follow every convention in them. Also read `docs/engineering/design-system.md`, `realtime.md` and `idempotency.md`.
2. `design_handoff_watchtower/README.md`. It is the spec.
3. `design_handoff_watchtower/design/index.html`, opened in a browser (toggle dark/light at the top right). `watchtower.jsx` and `watchtower.css` are a reference, not code to copy.

Where it lives: `client/apps/web/src/routes/desk/_components/watchtower/` (`watchtower-feed.tsx`, `item-row.tsx`, `severity.ts`). Replace the current feed and slide-over panel with the queue + card layout. It deliberately mirrors the Decisions flow (`routes/desk/_components/decisions/decision-flow.tsx`). Extract the shared queue, card, action-bar and progress primitives from Decisions into one component set and use it in both places. Do not duplicate them.

Requirements:
- Match the design exactly:
  - deadline lanes (Act now / Today / Coming up / FYI);
  - the "On it" strip with AgentRings;
  - the deadline clock ring;
  - At stake pills, the record button and the trail;
  - the agent block with its step bar;
  - the Hand off split button and agent menu;
  - Ask about this, which opens a Desk conversation with the item as page context;
  - the card exit animations;
  - the All clear state;
  - the ≤900px and ≤640px container breakpoints.
- One `useWatchtowerActions` hook drives the buttons and the keys (J/K/↑/↓/H/E/S, registered in `src/config/keybinds.config.ts`).
- Done and Snooze: optimistic updates with rollback and an Undo toast. Hand off assigns the agent immediately. Agent step progress and new signals arrive over SSE and patch the TanStack Query cache.
- Backend: add whatever the item contract is missing, following the hexagonal layout:
  - `dueAt`, `dueLabel`, `impact[]`, `trail[]`, `suggestedAgentId`;
  - agent ownership with step and status;
  - seen state;
  - snooze.

  Each capability ships with:
  - domain types and ports;
  - a service;
  - schema and resolver (`task gqlgen`, then `pnpm --filter @trenova/graphql codegen`);
  - migrations (up and down, both databases);
  - authz;
  - RLS on new tenant tables;
  - write-coverage;
  - tests.

  Use indexed cursor queries ordered by `dueAt`, and dataloaders for agent and subject fields.
- Rules:
  - design tokens only; `pnpm lint:design` passes;
  - icons from `@trenova/shared/components/icons`;
  - copy through `useT()`;
  - floating menus fully opaque;
  - animate only transform and opacity;
  - respect `prefers-reduced-motion`;
  - virtualize the queue with `src/components/virtual-rows.tsx`.
- Update the dev seeds so every state in the prototype appears:
  - each lane;
  - working, needs and done agents;
  - overdue detention;
  - an item with a trail;
  - an item with no deadline.
- Keep the sidebar count and the Today page's "Critical in Watchtower" link correct.

When the existing code and the design disagree, change the code. When a server rule makes the design impossible, keep the rule and note it in `docs/watchtower-redesign-notes.md`.

Before you start, list the files you will touch. When you finish, run the client `typecheck`, `lint`, `lint:design` and `test`, then `task test`, `task lint` and `task gqlschema-diff` in `services/tms`.

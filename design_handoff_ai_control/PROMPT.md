You are implementing the AI control redesign in the Trenova repo (emoss08/Trenova, branch master).

Read, in this order:
1. /CLAUDE.md and /AGENTS.md at the repo root. Follow every convention there.
2. design_handoff_ai_control/README.md. It is the spec.
3. Open design_handoff_ai_control/design/index.html in a browser. It is an HTML prototype, a design reference, not code to copy. Use the Tweaks panel (toolbar toggle) to see every state: no providers, many providers, paused, empty memory, each editor, unsaved, and conflict.

You have full permission to change anything: the frontend, the GraphQL schema, Go services, repositories, and migrations (both Postgres and SQLite). Keep the existing architecture. Do not invent a parallel one.

Hard rules:
- Recreate the design using the existing stack in client/apps/web: React, TanStack Router and Query, react-hook-form + zod, the shared UI in client/packages/shared, and tokens.css. Do not port the prototype's CSS or its hand-written components.
- Every list or table uses the existing client/apps/web/src/components/data-table/data-table.tsx. Change columns only; do not write a new table.
- Build ONE edit-sheet shell and ONE save-flow hook (spec §5). The agent builder, provider editor, tool-rule editor, policy editor, and the existing memory, extension and sweep dialogs all compose them.
- Every keyboard shortcut goes through client/apps/web/src/config/keybinds.config.ts.
- All copy goes through useT() / i18n, matching i18n/glossary.json.
- Every new backend capability in spec §8 ships with: a domain type, a port (core/ports/services + repositories), a service, a GraphQL resolver, Postgres and SQLite migrations (up and down), and tests next to the existing ones.
- AI-dependent features (Draft it, Tighten, Try it, Tune-ups narration) degrade gracefully when no provider is on. The spec states each fallback.

Work in this order and keep main green after each step:
  (1) backend contracts for §8.1–8.4: versioning, dry run, validation, and the routing preview;
  (2) the shared edit-sheet and save flow;
  (3) Overview;
  (4) Agents list and agent builder;
  (5) Providers and the provider editor;
  (6) the remaining tabs' panels moved onto the shared shell;
  (7) Tune-ups (§8.10).

Before each step, list the files you will touch. After each step, run the client typecheck/lint/tests and `go test ./...` for services/tms.

When something in the spec conflicts with what the code actually does, trust the code and the server's rules (for example agent-form-schema.ts superRefine). Note the conflict in docs/ai-control-redesign-notes.md.

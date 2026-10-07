# AI control — edit panels (handoff for Claude Code)

You may change anything: frontend, API, schema. Keep it DRY: **one shell, one draft hook, one set of fields**. No editor owns its own chrome.

## Frontend architecture
- `EditSheet` (prototype: `forms.jsx`) → `client/apps/web/src/components/edit-sheet/`. Props: `sections[] {id, label, icon, keys[], warn?, right?, body}`, `form`, `fields` (labels + formatters for the change review), `onSave`, `create`, `invalid`, `aside`, `banner`, `conflict`, `footerStart`.
  - Owns: section rail + scroll-spy, per-section dirty/warn dots, sticky footer (`N unsaved changes` → review popover with per-field undo, Discard, Save with ⌘/Ctrl+S), close guard ("Discard N changes?"), Esc handling, saving/saved states.
  - Back it with the existing react-hook-form + zod setup: `changed` = `formState.dirtyFields`, per-field undo = `resetField(name)`, review popover reads `defaultValues` vs `getValues()`. Do **not** write a new form library.
- Field primitives (`F`, `Txt`, `Area`, `Sel`, `SwRow`, `Chips`, `Callout`) should be thin wrappers over the existing form controls, not new components. The existing `data-table` stays as is.
- Editors are just section lists: `AgentEditor`, `ProviderEditor`, `ToolRuleEditor`. New editors (memory, extension, sweep settings) should be the same: a schema plus sections.
- Open editors through one route-level store (`?edit=agent:billing`, `?edit=provider:new&preset=ollama`) so they deep-link and survive a refresh.
- Keys come from `keybinds.config.ts`: add `editor.save` (⌘/Ctrl+S), `editor.close` (Esc), `agent.edit` (E), `editor.try` (⌘/Ctrl+Enter).

## Backend changes needed
1. **Optimistic concurrency.** Every editable entity returns `version`, and `PUT` requires `If-Match`. On 409, return the other person's diff and who/when, so the conflict bar can show "Load theirs" or "Keep mine".
2. **Agent versions.** Add an `agent_config_versions` table (snapshot, author, summary, created_at). `GET /agents/:id/versions` and `POST /agents/:id/versions/:v/restore`, where restore loads the version into a draft and does not save.
3. **Dry run.** `POST /agents/dry-run` with `{ draftConfig, prompt }` runs in simulation mode against live data, writes nothing, and streams steps with each tool's outcome (runs, asks first, proposes, not held, simulated). Saved test prompts are stored per agent.
4. **Shadow report.** `GET /agents/:id/shadow-report?days=14` returns recorded count, the share that matches what people actually did, would-be rejections, and would-be failures. Show it when moving Shadow → Live.
5. **Instruction lint.** `POST /agents/lint-instructions` flags capabilities the instructions mention but the agent's tools can't do. The prototype uses keyword rules; the server can use the tool registry.
6. **Provider model list.** `GET /providers/models?kind&baseUrl` proxies the provider's own model list (context window, price where known).
7. **Test a draft.** `POST /providers/test` takes an unsaved config and returns latency, the raw error, and a likely-cause hint.
8. **Route impact preview.** `POST /providers/route-preview` takes a draft and returns, per task, where it goes before and after. The frontend can also compute this with `routeOf`; keep the server as the source of truth.
9. **Key rotation.** Keep the previous key working for 24h after a replace (`rotation_expires_at`). Track `last_used_at` and `added_by`.
10. **Provider limits.** `timeout_seconds`, `max_concurrent`, `monthly_cap_usd`, and `on_cap: next|stop` are enforced in the router.
11. **Agent limits.** `runs_per_day`, `monthly_budget_usd`, `max_steps`, and `pinned_provider_id`, which skips routing when set.
12. **Tool rule changes require a reason.** It goes to the audit trail. The response includes affected agents with their effective tier before and after.
13. **Removal.** A confirm with the typed name. Keep runs and audit rows, and withdraw open proposals.

## Tweaks (prototype only)
- Open panel: jump straight to any editor.
- Editor state: `clean`, `unsaved` (prefilled draft changes), or `conflict` (someone else saved).
- Try it panel open: show the dry-run side panel by default.

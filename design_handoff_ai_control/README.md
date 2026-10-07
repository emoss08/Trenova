# Handoff: AI control redesign (Overview → Audit trail, editors, agent builder)

> Paste `PROMPT.md` into Claude Code. This README is the spec it points to.

## 1. Overview
The redesign covers `/agent-control` (labelled **AI control** in Settings › AI & Automation). It was rebuilt with the same modern patterns as Desk v2 and Shipments v2:
- content sits directly on the page, separated by thin rules;
- one sentence from Nova sums up each tab;
- each panel focuses on one action;
- AI features fall back gracefully when no provider is on.

Scope:
- all ten tabs: Overview, Agents, Providers, Extensions, Memory, Retrieval, Safety, Quality, Activity, Audit trail;
- a shared edit sheet used by every editor;
- a full-screen **agent builder** that replaces the agent form sheet.

## 2. About the design files
`design/` holds an **HTML prototype**: React 18 via Babel in the browser, with demo data. It is a design reference for look and behaviour; **do not ship it**. Recreate it in `client/apps/web` with the libraries already in use.

To view it, open `design/index.html`. The **Tweaks** panel switches between demo states:
- Providers: configured / many / none;
- Extensions catalog;
- Memory: recorded / empty;
- All agents paused;
- Panels: open any editor directly; Editor state (clean / unsaved / conflict); Try it panel open;
- Theme: dark / light.

Keys: `1`–`9` switch tabs, `/` focuses search, `N` creates something new, `E` edits, `⌘/Ctrl+S` saves, `Esc` closes.

## 3. Fidelity
**High fidelity.** Colours, spacing, type, motion and copy are final. Take the values from the existing tokens, not the prototype's CSS. The prototype's `ai.css` mirrors `client/packages/shared/src/styles/tokens.css` (neutral greys, brand hue 258).

## 4. Where it lives in the repo
| Area | Existing files to change |
| --- | --- |
| Route and tabs | `client/apps/web/src/routes/agent-control/` (page, `ai-control-tabs.ts`, `_components/agent-control-form.tsx`, `agent-control-options.ts`, `ai-readiness.ts`) |
| Overview | `_components/overview/overview-tab.tsx`, `agents-glance.tsx`, `usage-by-feature.tsx`, `recent-failures.tsx` |
| Agents list | `_components/agents/agents-tab.tsx`, `agent-rows.tsx`, `agent-roster.ts`, `trigger-meta.ts`, `track-record.tsx`, `scorecard.tsx` |
| Agent builder (replaces form) | `_components/agents/agent-form.tsx`, `agent-panel.tsx`, `agent-form-schema.ts`, `tool-catalog.ts`, `tool-picker-dialog.tsx`, `tool-summary.tsx`, `budget.tsx`, `delegates-field.tsx`, `delegates.ts`, `identity-picker.tsx`, `template-picker.tsx`, `template-fill.ts`, `prompt-preview-sheet.tsx`, `agent-access-section.tsx`, `agent-access-save.ts` |
| Providers | `_components/providers/provider-rows.tsx` and its sheet/form |
| Extensions | `_components/extensions/extension-roster.ts`, `extension-settings-dialog.tsx`. Marketplace cards reuse `routes/admin/integrations/_components/integration-catalog.tsx` (`CatalogItemCard`) and `packages/shared/src/components/ui/magic-card.tsx` |
| Memory / Retrieval / Safety / Quality / Activity / Audit | `_components/{memory,retrieval,safety,quality,activity,audit}/*` |
| Shared | `components/data-table/data-table.tsx`, `config/keybinds.config.ts`, `components/fields/*`, `lib/graphql/agent-control.ts`, `lib/graphql/agent-definition*`, `lib/graphql/agent-access*`, `types/assistant.ts` |
| Backend | `services/tms/internal/core/domain/agent/*`, `core/domain/tenant/agentcontrol.go`, `core/ports/{services,repositories}/agentdefinition.go`, `aiprovider.go`, `agentrun*.go`, `agentruntime.go`, `core/services/agentquerytoolservice/*`, `api/graphql/**` (loaders: `agentdefinitionstatsloader.go`, `agentrunloader.go`), migrations in `infrastructure/postgres/migrations` **and** `infrastructure/sqlite/migrations`, `pkg/buncolgen/*_gen.go` (regenerate) |
| Product docs | `docs/product-guide/admin/agent-control.md`: update to match |

## 5. Shared building blocks (DRY)
Build these once, then compose them everywhere. The prototype names are in parentheses.

1. **`useEditFlow`** (`forms.jsx`), a hook on top of react-hook-form:
   - `changed` = `formState.dirtyFields`; undoing one field = `resetField(name)`;
   - close guard ("Discard N unsaved changes?" → *Keep editing* / *Discard and close*);
   - `⌘/Ctrl+S` saves, `Esc` closes (popovers swallow `Esc` first);
   - states: saving (spinner) and saved (green "Saved" for 1.8 s).
2. **`SaveBar`**: shows "● N unsaved changes ▾". Clicking it opens a review popover listing each change as `label · old → new`, with an undo icon per row. Then *Discard* and **Save ⌘S**. In create mode the idle text reads "Nothing is created until you save". A validation message sits inline (amber, with an alert icon).
3. **`EditSheet`**: a right-hand sheet 680px wide (`min(680px, 100% − 16px)`), inset 8px, radius 12, background `--canvas`.
   - Header: icon (36), 18px/600 title, 12.5px subtitle.
   - Body: one column of sections, each with a 13.5px/600 heading. A changed section gets a 6px brand dot in the left gutter; an amber dot means a warning.
   - Field rows: label column 190px (label 12px/500, hint 11.5px muted below it), then the control.
   - The footer slides in only when the form is dirty, in create mode, or after a save.
   - Optional slots: `conflict` (§5.5), `banner` (for example a test failure, on danger-sub), `aside`.
4. **Field primitives**, thin wrappers over the existing `components/fields/*`: text (34px high, radius 7), textarea with a counter, select, switch row, chips multi-select, callout (info / warn / danger / ok tones, with an optional action button), and a typed-confirm dialog ("Type the agent's name to confirm").
5. **ConflictBar**: shown when a save returns 409. "**Sarah Alvarez** saved changes 2 min ago while you were editing." with *See theirs* / *Load theirs* / **Keep mine**. Uses the brand-sub background.

## 6. Screens
Shared page chrome: the existing app shell with a sticky tab bar, a sliding 2px underline that tracks the active tab, and a muted count on each tab. **No status badge in the page title.**

**Nova summary** (`Nova`, `tables.jsx`), used at the top of every tab:
- A spinning conic ring (16px; it only spins while something is working) with "Nova · <tab>".
- One 19px/1.5 sentence in `--muted`, with `<b>` in `--fg`. Inline links underline and fill with brand at 22% on hover; danger and warn variants exist.
- A control on the right.

**Figures row** (`Figs`): 3–5 cells separated by rules. Label 11.5 muted; value 26/600 mono with tracking −0.04em; sub-line 11.5.

### 6.1 Overview
- **Nova sentence.** Example: "**14 agents** are on, **3** working right now. *7 proposals* wait on a person in Watchtower, and *Workstation vLLM* can't connect. *1 task has* nowhere to go."
  - Links: Watchtower, the provider sheet, and Providers › Routing.
  - Control: a **hold-to-pause** button (900 ms fill, warn colour) with the note "They keep running in shadow". While paused it becomes **Resume agents**. With no provider it becomes **Connect a provider**.
- **Figures.** Model calls over 7 days, with 7 mini bars (failed portion in danger at the top of each bar); median response; tokens; spend. Spend with no prices shows a dashed **+ Set prices** button that opens the provider editor.
- **Provider failure strip.** Shown only when an on provider is failing.
  - Provider mark with a pulsing danger dot; "**Workstation vLLM can't connect**" and "24 failed calls overnight · its tasks fall to the next provider in line".
  - Actions: *Test again*, *Edit connection*, and **×** (dismissed until that provider fails again, with Undo in the toast). It disappears when a test passes.
- **Tune-ups** (main column). Configuration suggestions taken from the last 30 days of runs. These are not operational work; that stays in Watchtower.
  - Each row: mark or tile; title with a small icon; one line of evidence; a gain pill (ok / warn / brand / neutral); then dismiss (×) and the action.
  - Applying or dismissing collapses the row (260 ms grid-rows animation). A dismissal is snoozed for 30 days.
  - Rule kinds:
    - raise a tool's tier after N clean approvals;
    - reorder providers when an earlier one fails and a later one succeeds;
    - take an agent out of shadow when its match rate is ≥ 85% over at least 20 recorded proposals;
    - assign an uncovered task to a provider that already serves a suitable model;
    - turn off an agent idle for 14 days or more.
  - When the list is empty: "Nothing to tune".
- **Usage by feature** uses the shared data-table. Columns: Feature, Calls (with an inline bar and a red failed segment), Failed, Tokens, Spend, Median. Filter: "With failures".
- **Side column (340px).**
  - **Agents:** "working now" rows (tile with a pulsing dot, name, a shimmering "what it's doing…"), then the roster mosaic (28px tiles; off agents greyed; shadow agents with a dashed outline; amber badge for pending proposals) and a live / shadow / off legend.
  - **Organization-wide:** one row per setting (earned autonomy, learning, monthly allowance, share corrections), plus Routing "N of 10 covered". Everything opens the **Policy editor** (EditSheet). When earned autonomy is turned on, it previews which tools would move up a tier on save.
- **No-provider state:** Nova sentence, then a 3-step setup with provider preset buttons.

### 6.2 Agents
- **Nova sentence:** "14 of 15 agents are on, and 3 are working right now. *7 proposals* wait… 3 agents run in *shadow* and have recorded 61 proposals nobody has seen". Control: Review in Desk.
- **Working now:** a row of cards.
- **Toolbar:** search, filter segment (All / Waiting / Shadow / Off), "N of M on", and **New agent** (N) with a template menu.
- **Groups:** Chat / Scheduled / On an event, each with a column header: Agent · Last 14 days · Approved · Waiting · Tools.
- **Rows** (56px):
  - tile (32);
  - name with Shadow / Simulation / System tags;
  - a second line starting with one fact (next run, event, or role access), replaced by a shimmering activity line while running;
  - 14-day run bars with the total;
  - an approval ring and %, or "31 recorded" for shadow agents;
  - an amber waiting pill (opens Desk);
  - a tier bar (read / propose / act) and the tool count;
  - Run now or Ask in Desk, and an enable switch.
- No status dots on tiles. Clicking a row opens the read sheet, with an **Edit** button (E).

### 6.3 Agent builder (full screen, replaces `agent-form.tsx` and `agent-panel.tsx`)
Covers the whole viewport (`inset: 0`, edge to edge, no outer padding).

**Header** (56px):
- ‹ Agents / tile / name;
- **mode pill** (Live green dot / Shadow dashed ring / Simulation violet), which opens a menu with descriptions;
- **Enabled** switch;
- **Try it** toggle (teal);
- ⋯ menu: Duplicate, Export as JSON, Runs and proposals, Remove (typed confirm; disabled for system agents);
- the SaveBar, then ×.

**Body:** a 240px left rail, the canvas, and a 400px Try it panel when it is open.

**Rail:**
- a checklist: Identity, Instructions, When it runs, Tools and autonomy, Budget and model, Memory and handoffs, How it's doing (edit only), Try it.
- each item shows a status (ok = green filled check, warn = amber "!", todo = dashed ring), its label, and a one-line summary.
- bottom: a readiness ring ("N left before going live") and the version link (a popover with history and Restore, which loads into the draft only).

**Canvas** (padding 40/48):
- **Identity:** a 64px tile with an edit badge on hover (popover with 16 icons and 10 hues); an inline 30px/600 name input and a 15px description input.
- **Summary sentence** on a faint panel. Each clause is clickable and jumps to its section. Example: "Answers anyone in Desk, reads with 11 tools, does 2 things on its own, asks a person before 7 and only proposes 3. It runs in shadow…".
- **Instructions:**
  - toolbar: insert-variable buttons (`{{organization}}`, `{{user.name}}`, `{{user.role}}`, `{{today}}`), a token estimate, and **Tighten** (AI; hidden with no provider);
  - an editor that highlights `{{vars}}` (a transparent textarea over a highlighted backdrop, growing with its content, 14/1.75);
  - lint rows: "Mentions email, but it holds no tool that sends email." with **+ Give it Reply to customer**;
  - a **Never** list (`guardrails`): numbered rows you can edit inline, plus an add row (Enter).
- **When it runs:**
  - four cards: Someone asks / On a schedule / Something happens / Keeps watch (`triggerMode` Chat / Scheduled / Event / Continuous);
  - Scheduled: preset chips, a **7-day strip** (one track per day with a dot per run, today marked, a "now" line, and "N runs in the next 7 days"), a cron input and a timezone select;
  - Event: event chips;
  - Continuous: Every (1 / 5 / 15 min, 1 h; at least 60 s) and At most N runs at once;
  - Scheduled and Continuous: **Stop after** (`endsAt`);
  - access: Everyone / Specific roles plus role chips. If the agent is open to everyone and holds outside-reaching tools, warn with **Limit to roles**.
- **Tools and autonomy**, a bordered "bench" (radius 14):
  - **Left panel:** a toolbar (Add tools first, then filter search, then All / Changes / Reads with counts; it wraps).
  - Column header: Tool · Freedom · Daily limit.
  - **The list scrolls at a fixed 420px height.** Groups by resource with sticky headers (`groupToolsByResource`). Each row: egress dot, name and reach, a three-way **Propose / Ask first / Automatic** control (locked past the tool's own max; striped and amber when above the ceiling), a daily limit input ("No limit … /day", `toolDailyLimits`, change tools only), and remove.
  - Footer, collapsible: "Also held · N". It shows *Every agent* (core / `grantedToEveryAgent`) and *With your tools* (`impliedReads`, "for <tool>").
  - **Right panel (270px):** Ceiling (radio list with descriptions); "How its N changes run" (distribution bar and legend); **Data access** Internal / Restricted (`dataAccessCeiling`); **Proposals expire after** (`decisionTimeoutSeconds`: 1 h, 4 h, 1 d, 3 d, 7 d).
  - **Add tools dialog:** 820×640 with a search box, group nav with chosen counts, and tick rows showing reads/changes, reach, max tier and outside text. It writes through immediately; *Done* closes it.
- **Budget and model:** four big-number cells with −/+ steppers.
  - Monthly budget (empty means no cap; a spend bar from `BudgetStatus`); Runs per day (0 means no cap); Run timeout (minutes → `runTimeoutSeconds`); Tool calls per run (`maxToolCalls`).
  - Preferred provider (Automatic, or a provider).
  - Replies as: *A conversation* / *A report* (`outputMode`).
- **Memory and handoffs:** Learns from its work (disabled with a reason when the organization turns it off); Memory in the prompt (`memoryTokenBudget`, 1,000–16,000, empty = default); **Tell it about** chips (`contextProviders`, none = all; **use the real enum labels from `types/assistant.ts` — the prototype's five are placeholders**); **Can ask** grid (`delegateIds`, Chat only, at most `MAX_DELEGATES` = 8).
- **How it's doing** (edit only): Runs / Approved / Cost / Time saved (`scorecard.tsx`) and the streak bar (`track-record.tsx`).
- **Try it panel** (Desk-style):
  - empty state with three suggested prompts;
  - each run shows the question, the agent line, steps with outcome tags (Runs / Asks a person first / Proposes only / Recorded / Simulated / Not held · skipped), the answer streaming word by word, and a meta line ("1.4s · 2.1k tokens · ~$0.004 · nothing written");
  - after the draft changes, the last run shows "You changed the draft since this run — **Run again**";
  - composer: Enter sends.
  - Backed by §8.2.
- **Create flow:** opens on "**What should it do?**":
  - 36px heading, a large prompt box with example chips, and **Draft it ⌘↵**. While drafting, steps shimmer ("Naming it…", "Writing instructions…", …), then the builder opens with the filled sections glowing briefly.
  - With no provider: "Connect a provider to draft with AI", and the templates below still work (Desk agent / Scheduled report / Event watcher / Start blank).
  - New agents default to **Shadow**; the save label reads "Create in shadow".

### 6.4 Providers
- **Nova sentence** about who is taking work, with a one-click fix: *Test X* or *Add X key*.
- **"The chain":** compact rows that are always used (48px, drag to reorder). Each row: 01-style number with a grip, mark with a status dot, name and model, "first for N" or "backup", 7-day mini bars, success %, median, test icon, switch. Off providers fold into "Show N off".
- **Routing grid** (tasks × providers): the task column is sticky left and "Goes to" sticky right; its shadow is **black in dark mode**.
- **Provider editor** (EditSheet):
  - Connection, with a URL check ("This address is on your own network. Turn on Private network…" + fix);
  - Model, with **Fetch models** (a list with context window and price);
  - API key: masked current key with added-by and last-used; replace; "Keep old key working 24 h";
  - What it handles: task chips and a **route impact** list ("When you save: Billing diagnosis · Workstation vLLM → Anthropic");
  - Access: Trusted, Private network;
  - Limits and price: timeout, concurrency, monthly cap with *Hand to next* / *Stop*, input and output price;
  - Remove;
  - footer **Test draft**, with the result inline and a danger banner showing the raw error and a hint;
  - in create mode the save label becomes "Add and turn on" after a passing test.

### 6.5 Other tabs
Each tab is a Nova sentence, a Figures row and the shared **data-table** with tab-specific columns. Their panels move onto EditSheet: memory new/edit, extension settings, quality sweep settings, the tool-rule editor (requires a reason when the maximum changes, and shows affected agents before → after), and the read sheets for runs, proposals and exceptions. The Extensions marketplace uses `CatalogItemCard` with the MagicCard hover, and its featured card has a Desk-style "How agents use it" loop.

## 7. Interactions and motion
- Easings: swift `cubic-bezier(.2,.8,.2,1)`, settle `cubic-bezier(.16,1,.3,1)`, spring `cubic-bezier(.34,1.4,.64,1)`.
- Durations: rise 300 ms (translateY 4px → 0); sheet in 340 ms; popover 180 ms; row collapse 260 ms; hold-to-pause 900 ms.
- The shimmer text gradient loops every 1.6 s. Respect `prefers-reduced-motion`.
- Toasts: bottom centre, ink background, optional **Undo** (5 s with Undo, 2.6 s without).
- Empty, loading and error states are drawn for every tab; see the Tweaks states.

## 8. Backend work (required)
Each item needs: a domain type, ports, a service, GraphQL, Postgres and SQLite migrations (up and down), regenerated `buncolgen`, and tests.

1. **Optimistic concurrency.** `agent_definitions.version` already exists; enforce it on save and return 409 with the other person's diff, author and time. Add `version` to `ai_providers` and the tenant agent-control settings.
2. **Dry run.** Add `dryRunAgent(draft: SaveAgentDefinitionRequest, prompt: String!)`, streaming like the assistant. It runs in simulation mode against live data and writes nothing. Each step returns `{tool, outcome: Runs|AskFirst|Propose|Recorded|Simulated|NotHeld}`. Store saved test prompts per agent.
3. **Agent versions.** A new table `agent_definition_versions` (snapshot JSON, author, summary, created_at). Queries: `agentVersions(id)`. `restoreAgentVersion` returns a draft and never saves directly.
4. **Shadow report.** `agentShadowReport(id, days)` returns recorded count, match %, would-reject count and would-fail count.
5. **Instruction lint.** Instructions that mention a capability the agent's tools can't do. Use the tool registry, not keywords.
6. **AI drafting.** `draftAgentFromDescription(text)` returns a `SaveAgentDefinitionRequest`; `tightenInstructions(text)`. Both are hidden when no provider covers AssistantChat.
7. **Providers:**
   - `providerModels(kind, baseUrl)` proxies the provider's model list;
   - `testProviderDraft(input)` returns latency, the raw error and a hint;
   - `routePreview(draft)` returns each task's provider before and after;
   - key rotation: `rotation_expires_at`, `last_used_at`, `added_by`;
   - limits: `timeout_seconds`, `max_concurrent`, `monthly_cap_usd`, `on_cap` (next | stop), enforced in the router.
8. **Tool rule changes** require a `reason`, stored in the audit trail. The response includes affected agents with their effective tier before and after.
9. **Overview figures:** a 7-day series of calls and failures, median and p95 latency, tokens in and out, and spend. Use the existing usage tables.
10. **Tune-ups:** `aiTuneUps` returns typed suggestions `{kind, subjectId, evidence, gain, action}`, plus `applyTuneUp(id)` and `dismissTuneUp(id, days=30)`. Kinds and thresholds are in §6.1. Computed nightly and cached per tenant.
11. **Provider failure dismissal:** a per-user dismissal keyed by provider and last failure time, so the strip comes back on a new failure.
12. **Removal:** keep runs and audit rows, withdraw open proposals.

## 9. Design tokens (from `tokens.css`; use the variables)
| Token | Dark | Light |
| --- | --- | --- |
| canvas | oklch(0 0 0) | oklch(0.985 0 0) |
| card / raised | 0.145 / 0.178 | 1 / 1 |
| fg / muted / subtle / faint | 0.946 / 0.709 / 0.65 / 0.46 | 0.205 / 0.51 / 0.53 / 0.7 |
| border-sub / border / strong | 0.239 / 0.26 / 0.321 | 0.94 / 0.922 / 0.836 |
| brand | oklch(0.717 0.148 258) | oklch(0.56 0.207 258) |
| ok / warn / danger | 0.645 0.145 147 / 0.817 0.164 76 / 0.626 0.193 23 | 0.535 0.127 147 / 0.817 0.164 76 / 0.581 0.206 25 |
| teal (Try it) / violet (Simulation) | 0.72 0.089 182 / 0.72 0.13 295 | 0.55 0.093 182 / 0.55 0.16 295 |

- Type: **Geist** and **Geist Mono**, with tabular numbers.
- Sizes: base 13; Nova sentence 19; builder name 30; create heading 36; big figures 26–28.
- Radius: controls 6–8, cards and bench 12–14, sheets 12.
- **No drop shadows. No left accent bars.** Use borders and tints.

## 10. Assets
Icons come from the existing icon set (lucide-style strokes in the prototype's `icons.jsx`). Provider marks and agent tiles are generated from initials or icon plus hue, as in `components/agent-identity/agent-tile.tsx`. `logo.png` is the org logo placeholder.

## 11. Prototype file map (`design/`)
`index.html` · `app.jsx` (state, actions, tweaks) · `shell.jsx` (chrome, Sheet, Modal, Menu, Seg, Switch) · `forms.jsx` (**useDraft, useEditFlow, SaveBar, EditSheet, fields**) · `overview.jsx` (Overview, Tune-ups, PolicyEditor) · `agents.jsx` · `agent-kit.jsx` (constants, draft mapping) · `agent-builder.jsx` · `agent-tools.jsx` (tool bench and picker) · `agent-try.jsx` (create flow, week strip, Try it) · `providers.jsx` · `edit-provider.jsx` (provider editor, tool rule editor, editor router) · `extensions.jsx` · `memory.jsx` · `retrieval.jsx` · `safety.jsx` · `tables.jsx` (Nova, Figs, data-table recreation) · `rest.jsx` (Quality, Activity, Audit) · `data*.jsx` (demo data only) · `ai.css`–`ai4.css` · `PANELS.md` (earlier backend notes, merged into §8).

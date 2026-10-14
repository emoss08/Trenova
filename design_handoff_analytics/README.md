# Handoff: AI control › Analytics (new tab)

> Paste `PROMPT.md` into Claude Code. This README is the spec it points to.

## 1. What Analytics is
A tab in AI control that answers one question: **which model, on which provider, should do which job?** It compares every model (or provider) the organization runs on completion, speed, tool-call accuracy, cost, and how people judged its proposals, then links each finding to the setting that acts on it (routing, provider, tool rules).

Principles:
- **Comparisons, not counts.** Every chart ranks or contrasts groups. A single total never stands alone.
- **Say it in words first.** Each leaderboard card opens with a sentence ("GPT-5.1 finishes 93 of every 100 runs; Llama 70B finishes 80"). Nova's summary at the top names the best model, the same-model-different-provider gap, and the worst tool-caller.
- **Separate the model from the connection.** Provider errors (rate limits, timeouts, auth) happen before the model answers and are always shown by provider. Tool-call failures are split into the model's mistakes and the tool's or a rule's.
- **Small samples don't rank.** Groups under 30 runs (20 calls for a tool cell) show "few" and sit at the end, unranked.
- **Every mark opens something.** A bar, dot, row or cell opens the detail sheet, or goes to Providers, Routing, Tool rules or Activity.

## 2. The prototype
Open `design/index.html?tab=analytics` (or press the Analytics tab). It's the full AI control prototype; Analytics lives in:

| File | Covers |
| --- | --- |
| `analytics.jsx` | Tab layout, scope bar, figures, tool heatmap, task fit, proposals, provider health, detail sheets |
| `analytics-charts.jsx` | Comparing chips, leaderboard cards, scatter, same-model card, run-time ranges, over-time lines, "whose mistake" chart |
| `analytics-data.jsx` | Demo data and every metric definition (`AN_MET`, `AN_WHY`, `AN_PERR`, `AN_OUT`) |
| `analytics.css` | Styling (reference only) |

Shared shell pieces it uses (`Nova`, `Figs`, `Seg`, `Sheet`, `KV`, `Mark`) are in `shell.jsx` / `forms.jsx` and already exist in the codebase from the AI control handoff.

**It is a reference for look and behaviour, not code to port.** Don't copy its CSS or hand-written charts.

Tweaks panel: `Providers: none` shows the empty state; `Theme` toggles light.

Screenshots of every section are in `screenshots/` (01–11).

## 3. Fidelity
High fidelity. Layout, hierarchy, copy and interaction are final.
- Colour, radius, type, elevation from `client/packages/shared/src/styles/tokens.css`; `pnpm lint:design` passes.
- **Series colours:** one stable hue per model and per provider, `oklch(0.72 0.15 <hue>)` (see `AN_HUE`). Add them as a chart palette token set, assigned by stable key, not by sort order.
- Outcome, failure-cause and provider-error colours are semantic (completed / handed off / stopped / failed; model vs tool causes). Map to existing status tokens where they read correctly.
- Floating surfaces (tooltips, selects, sheet) are fully opaque.
- Icons from `@trenova/shared/components/icons`. Copy through `useT()`, sentence case, matching `i18n/glossary.json`.

## 4. Layout (top to bottom)
1. **Nova summary** with Export (CSV of the current scope) and "Updated n min ago". Up to three sentences, each conditional (see `AnalyticsTab`): best completer (≥30 runs); same model on two providers with the completion gap; worst tool-caller if ≥2× the rest, with its top cause. Names are links that open the sheet or provider.
2. **Scope bar:** range (24 hours / 7 days / 30 days), Compare by (Model / Provider), Agent select, Task select, Clear. All in the URL.
3. **Figures (5):** Runs, Completed, Median run, Tool calls failed (red above 3%), Spend. Numbers count up on change.
4. **Comparing chips:** one per group with run count. Hover highlights that group in every chart on the page (shared highlight context); click hides it from charts (at least one stays). "Show all" resets.
5. **Leaderboard:** six cards, 3×2 — Completed, Median run time, Tool calls that failed, Cost per completed run, Approved as proposed, Time to first token. Each: title, definition, "Higher/Lower is better", the sentence, vertical bars sorted best first, best one marked, "few" groups greyed at the end.
6. **Trade-offs:** scatter (Up: Completed / Approved / Rated helpful; Across: run time / cost / TTFT / tool failures), dot size = runs, shaded "Best here" quadrant from medians, collision-avoiding labels, tooltip. Beside it, when comparing by model and one model runs on two providers: **Same model, different provider** butterfly chart with "X is better on n of 5" and "Put X first" (reorders routing).
7. **Run time:** per group, p25–p75 bar, line to p95, median tick, on a log axis (1s–2m); outcome switch re-sorts rows with animated reordering; outcome-mix bar per row; an insight line when failed runs take ≥2× completed.
8. **Over time:** lines per group for Completed / Tool calls failed / Median run time, buckets per range (24 hourly, 28 six-hourly, 30 daily), provider event markers ("vLLM stopped answering"), crosshair tooltip ranked by value. Gaps where a deployment didn't run.
9. **Tool calls:** heatmap tool × group, cell = failure share (0 → 12%+), sorted worst tool first; click a cell for its sheet. Beside it, **Whose mistake was it**: per group, model causes left, tool/rule causes right, stacked by cause.
10. **Routing and judgment:** **Which model fits which task** heatmap of completion per task (only tasks the group was routed to), "Goes here now" outline on the current route, check on the best; when another deployment beats the route by ≥5 points, "Use X +n" links to routing. Hidden when a task filter is set. Then **What people did with its proposals** (approved / with changes / rejected stacks) and **Provider health** (error stack by kind, rate, fell-to-next count, first-token median; row opens the provider).

Sheets:
- **Group sheet:** runs, completed, tool failure share and model share, approved, rated helpful, cost (or Local), first token; run time by outcome table; top 4 tools it gets wrong; buttons to its providers and its runs.
- **Tool cell sheet:** failures by cause with whose fault; three recent failures with call, message, agent and time; links to Tool rules and Failed runs.

States: no providers → "Nothing to measure yet" + Connect a provider. Filters match nothing → "No runs match". Deployments turned off show "last ran Oct 3".

## 5. Metric definitions (contract)
From `AN_MET` — implement once, server-side, and reuse:
- **Completed** = runs that finished without handing off or failing ÷ runs. Outcomes: completed, handed to a person, stopped at a limit, failed.
- **Median run time** = p50 of completed runs; run-time chart needs p25/p50/p75/p95 per outcome.
- **Tool calls that failed** = failed tool calls ÷ all tool calls. Causes: model — wrong arguments, called a tool it doesn't hold, repeated the same call (loop guard); tool/rule — tool returned an error, blocked by a tool rule, tool timed out.
- **Cost per completed run** = provider charge ÷ completed runs; local providers are "Local", not $0.
- **Approved as proposed** = proposals accepted unchanged ÷ proposals decided.
- **Time to first token** = median, weighted by model calls.
- **Rated helpful** = thumbs up ÷ ratings.
- **Provider errors** (before the model answers): rate limited, timed out, couldn't connect, access denied, server error; plus "fell to next" = calls that failed over to the next provider in routing.
- A group is a model (aggregating its deployments) or a provider. A **deployment** = model on provider.

## 6. Where it lives in the repo
**Frontend (`client/apps/web`):**
- New `src/routes/agent-control/_components/analytics/` (tab, cards, charts, sheets, `analytics-model.ts` for formatting and the sentences).
- Tab registration next to the other AI control tabs; nearest existing pattern: `quality/extraction/provider-trends-panel.tsx`, `provider-trend-model.ts`, `overview/use-ai-control-stats.ts`.
- Charts: use the chart primitives already in `@trenova/shared` if present; otherwise add small SVG chart components there (bars, scatter, range, line, heatmap). No new chart library unless one is already a dependency.
- Reuse the existing Nova summary, figures, segmented control, sheet and KV components from the AI control redesign.

**Backend (`services/tms`)**, existing sources:
- Runs and steps: `internal/modules/agent/domain/agent/agentrun.go`, `agentrunstep.go`, `agentrunevent.go`; repos under `internal/modules/agent/adapters/postgres/agentrun*repository/`.
- Usage and cost: `internal/modules/agent/ports/aiusageports/`, `adapters/postgres/aiusagerepository/aiusage.go`, resolver `internal/api/graphql/resolver/aiusageresolver/`.
- Tool-call parsing: `internal/infrastructure/agentcompletion/modeladapter/`.

New work:
- Record per tool call: tool, outcome, **cause** (the six above), latency. Record per model call: provider, model, TTFT, provider error kind, fell-to-next.
- One `AgentAnalytics` query: input range, groupBy (model|provider), agentId?, task?; returns groups and deployments with the aggregates in §5, tool matrix, task matrix, proposal decisions, provider errors, and a bucketed time series. Plus `AgentAnalyticsToolFailures(deployment, tool)` for the cell sheet's recent examples.
- Provider events for the trend markers (restart, lost access, turned off, stopped answering) from existing provider health/audit data.
- CSV export of the same query.

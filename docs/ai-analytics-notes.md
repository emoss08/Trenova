# AI control › Analytics: implementation notes

The handoff (`design_handoff_analytics/`) asked for these notes. Where a server rule or a
repository convention made the design impossible as drawn, the rule was kept and the conflict is
written down here. It also records the query profile on a tenant with 500,000 runs.

How the feature is built (the rollup, the refresh job, the query, the cache and routing) is in
[engineering/ai-analytics.md](engineering/ai-analytics.md).

## Where the build differs from the handoff, and why

**Section headings are sentence case.** The prototype sets "LEADERBOARD", "TRADE-OFFS" and the
other headings in uppercase with letter spacing. The design system forbids
`uppercase tracking-wider` on section labels, so they read "Leaderboard", "Trade-offs" and so on.
The detail sheets keep the kit's existing `sh-k` heading style, which is shared with every other
AI control sheet.

**"Use X +n" routes the task.** Screenshots 08a and 08b show a right-hand column of
"Route to X +n" buttons. The README and `analytics.jsx` instead put "Use X +n" under the task name,
in place of "Goes to X". The screenshots were taken from an earlier prototype, so the build follows
the README and the current prototype code.

The prototype only navigates to routing and shows a toast. The build performs the routing change,
behind a confirmation, through `routeAITask`. That mutation applies the same `PlanRouteTo` rules
the suggestion was computed with:

- The task moves to the suggested provider.
- Providers ahead of it in routing stop taking the task.
- Every other task stays where it was.
- An audit entry records the change.

A person without the right to edit providers sees the link disabled, with a tooltip that says why.

**No suggestion to a provider that is off.** In screenshot 08a, "Route to Google Gemini +11" is
offered while Gemini is turned off. The server suggests a route only when routing can actually
reach the target:

- The target is enabled.
- It still serves the same model.
- No provider ahead of it would lose its only task.

Gemini's row therefore shows the check on its cell but no "Use" link.

**Small samples.** As the README specifies:

- Groups under 30 runs, and tool cells under 20 calls, are marked "few" and left unranked.
- A "Use X" suggestion also needs the current route and the candidate to each have 30 runs on that
  task.
- "Best for the task" is only marked when at least two groups can be ranked, as in the prototype.

On a leaderboard a "few" bar is dashed with a faint value and keeps its series name, so the reader
can still tell which group it is.

**Provider error kinds.** The handoff names five kinds. Calls are now classified where they are
recorded:

| Kind | Recorded when |
| --- | --- |
| Rate limited | HTTP 429 |
| Access denied | HTTP 401 or 403 |
| Couldn't connect | Network failure |
| Timed out | The call timed out |
| Server error | Retryable 5xx, plus anything else the provider returned that Trenova could not use |

Rows recorded before this change are mapped by message pattern.

Two failure classes are stored but are not provider errors:

- A model that declined to answer.
- A call whose caller went away (cancelled).

Both count towards calls and fail the call, but neither is in the "Errors" bar.

**"Local" cost.** A group with no priced runs shows "Local" instead of $0, as the README asks. A
hosted provider whose prices were never configured is also unpriced, so it also reads "Local".
Pricing is per provider, and the provider editor already asks for it.

**Series colours.** There are twelve hues, 30° apart, at `oklch(0.72 0.15 h)`, added as
`--series-1` … `--series-12` in both themes. Four hues trim their chroma to stay inside sRGB, which
`pnpm lint:design` checks:

| Hue | Chroma |
| --- | --- |
| 258 | 0.144 |
| 198 | 0.12 |
| 228 | 0.138 |
| 168 | 0.142 |

Slots are assigned on the server (`aianalytics.AssignPaletteSlots`) over every model and provider
seen in the last 30 days. A model therefore keeps its colour whatever the window, comparison, agent
or task. A deployment takes its provider's hue.

When two keys hash to the same slot, the later one in hash order moves to the next free slot. So a
newly connected model can, rarely, move one existing model to a different hue.

**Chart primitives.** The README suggests adding small chart components to `@trenova/shared`. The
generic pieces went there:

- `useElementWidth`, built on ResizeObserver.
- `useCountUp`.
- `useReducedMotion`.

The charts themselves stay in `routes/agent-control/_components/analytics/`, because each one is
specific to its card (the label collision rules, the butterfly, the range bars). Nothing else in
the product draws them yet. Moving one to shared is mechanical when a second page needs it.

**"Its runs" and "Failed runs".** These links open Activity › Runs, filtered by the selected agent
and, for failed runs, by status. The runs table cannot filter by canonical model or provider, so
the link does not narrow to the group.

**Over-time buckets** follow the README: 24 hourly, 28 six-hourly and 30 daily points, in the
viewer's time zone. The server works out where each bucket starts (daily and 6-hour buckets on
the local calendar, so a day with a clock change is 23 or 25 hours) and returns the list with the
answer. The query buckets runs with `width_bucket` over the same list, and the chart draws it as
given, so a run is never drawn in the neighbouring bucket.

**Event markers.**

- Markers name a provider by its short name, as the prototype does ("vLLM", "Bedrock").
- The design's "vLLM restarted" is the server's "vLLM recovered": the first hour back
  under 20% failed calls after an hour at 50% or more.
- "Lost access" is the same transition when most failures were access denied.
- "Turned off" and "switched model" come from the provider's settings history.

**Query names.** The README names the query `AgentAnalytics`. It is
`agentAnalytics(input): AgentAnalytics!`, and the cell sheet's query is
`agentAnalyticsToolFailures(input, seriesKey, toolName)`.

**Databases.** The prompt asks for migrations for Postgres and SQLite. Trenova runs on Postgres
only. Its migrations are `20261231009250_ai_analytics.tx.{up,down}.sql` and
`20261231009260_ai_analytics_indexes.{up,down}.sql` (concurrent indexes on the source tables).

**Evaluation calls count towards provider health.** Calls made while evaluating an agent (the
judge's calls in an evaluation run) are not runs, so they are left out of runs, spend and every
per-run figure. They are real calls to the provider, though, so they stay in provider health:
calls, errors, error rate and first-token time. A provider that times out on an evaluation batch
is a provider that times out.

**Numbers in sentences.** The server writes percentages and ratios ("86%", "2.5×") into the
Nova sentences. In every locale Trenova ships (en-US, es-419, zh-CN, zh-TW) those are written the
same way, so they need no locale formatting. Counts inside plural messages (`#`) are grouped the
way the locale writes numbers ("21,375 model calls"), identically in the Go and browser
formatters. The client formats money with the shared currency formatter, so es-419 reads
"USD 0.053".

**What stays on the client.** Every threshold that decides what a figure *means* (ranking,
"few", alarms, "quiet", suggested routes, the same-model pair) is decided on the server and sent
as a flag. The client keeps presentation only: the colour scale of a heatmap cell and where a bar
tops out.

**Tool latency.** Each tool step now records when it started and settled, in milliseconds, and
model calls record first-token time and when they finished. The design has no tool-latency chart,
so tool latency is recorded but not drawn.

## Seed

`AIAnalytics` (development) writes a month of analytics for the default organization. It also adds
an "Office vLLM" provider that serves the same Llama model as the workstation. The seed shows every
state in the prototype:

- The same model on two providers, both with enough runs to compare. This drives the same-model
  card, the connection-gap sentence and "Put Office vLLM first".
- A failing local provider. The workstation times out, fails to connect, stopped answering this
  morning, and limped 13 days ago. Its failed runs wait out the 60-second timeout.
- A turned-off deployment. Hosted Anthropic lost access, was switched off six days ago, and reads
  "last ran …" with "off".
- Every outcome and every failure cause.
- Accepted, modified and rejected proposals.
- Thumbs up and down.
- A task with a better route: Assistant chat goes to the workstation, and Office vLLM finishes more of it.

The seed writes the rollup tables directly and marks the tenant refreshed through the moment it
ran. The refresh job therefore only rebuilds from then on, and the seeded month stays until
retention removes it. Its proposals, decisions and ratings hang off lightweight runs, backdated so
the refresh never revisits them. `Down` removes those runs and clears the rollups, which the
refresh rebuilds from the real sources.

## Profile: 500,000 runs

**Environment.** Profiled in the development container against PostgreSQL 16 with default
settings (`shared_buffers` 128 MB, `work_mem` 4 MB), on the migration's schema and indexes.

**Data.** One tenant had 500,000 work units across 7 deployments and 6 tasks, with:

- 866,738 tool rows.
- 272,160 hourly provider rows (720 hours × 7 deployments × 6 tasks × 3 agents × 3 error kinds).

A second tenant held another 500,000 units, so every read had to filter. The generated runs span
about six days, so the 7- and 30-day windows both hold all 500,000. The 24-hour window holds
86,300.

**Method.** Timings are the median of three warm runs of the whole `Analytics` call. Fifteen reads
run four at a time. The Redis cache is off, so these are the uncached times a person sees after
each five-minute refresh.

| Scope | Runs in scope | Whole query |
| --- | ---: | ---: |
| 30 days, by model, every agent | 500,000 | 1.40 s |
| 30 days, by provider, every agent | 500,000 | 1.40 s |
| 30 days, by model, one agent | 66,667 | 0.75 s |
| 7 days, by model, every agent | 500,000 | 1.08 s |
| 7 days, by model, one agent | 66,667 | 0.57 s |
| 24 hours, by model, every agent | 86,300 | 0.31 s |
| 24 hours, by model, one agent | 11,507 | 0.20 s |

Repeated runs of the whole profile differ by up to 10% (the 30-day, every-agent query measured
1.40 s and 1.53 s on two runs), so read the figures as a scale rather than exact.

The slowest reads at 30 days, every agent, by model:

| Read | Time | What it does |
| --- | ---: | --- |
| Trend | 0.90 s | Exact median per series per day: sorts every run once |
| Outcome durations, per deployment | 0.74 s | Exact p25/p50/p75/p95 per outcome |
| First-token histograms | 0.70 s | Sums 64-bucket histograms across hourly rows |
| Outcome durations, per model | 0.56 s | As above, per group |
| Tool cells | 0.34 s | Tool calls by deployment, tool and cause |
| Totals | 0.33 s | Counts and the overall percentiles |
| Deployments, task cells, provider calls, provider hours, palette | 0.21–0.31 s each | Grouped counts |
| Decisions, ratings | ~1 ms | Indexed joins on decided proposals and ratings |

Two changes came out of profiling:

- **Trend: 2.07 s → 0.90 s.** Converting every run to the viewer's time zone dominated the trend.
  It now buckets with `width_bucket` over bucket starts the service computes once per request.
- **Tool cells: 1.12 s → 0.34 s.** Tool rows carry their unit's model, provider, agent and task, so
  the tool reads no longer join 867,000 tool rows to 500,000 units.

The whole 30-day query went from 2.37 s to 1.40 s.

**Remaining cost.** What is left is sorting for exact percentiles. A larger transaction-local
`work_mem` kept the sorts in memory but saved under 10%, so it was not adopted. Median run time is
a headline figure, so it stays exact rather than estimated from a histogram. First-token times,
which are only ever read as medians across hours, use the mergeable histogram.

**Repeat reads.** These come from Redis. The cache key holds the tenant, a per-tenant epoch, the
window, the comparison, the agent, the task and the time zone, with a 2-minute TTL. The refresh
job bumps the epoch whenever it folds in new runs, so a cached answer never outlives the data it
was built from. Hovering a range button prefetches that range.

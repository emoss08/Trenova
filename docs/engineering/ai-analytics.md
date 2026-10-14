# AI analytics

How AI control's Analytics tab answers "which model, on which provider, should do which job":

- the telemetry it reads
- the rollup a worker keeps
- the one query every chart reads
- the cache in front of that query
- the routing change a finding can make

Read this before changing `aianalyticsservice`, `aianalyticsrepository`, `aianalyticsjobs`, the
`aianalytics` domain, `aianalytics.graphqls`, or what `completionrouter`, `runstepledger` or
`agentruntime` records about a model call or a tool call. Design decisions and the query profile
are in [../ai-analytics-notes.md](../ai-analytics-notes.md).

## The shape of it

```
completionrouter ─▶ ai_usage_records (provider, model, task, error class, first token, recorded_at_ms)
runstepledger    ─▶ agent_run_steps  (tool, verdict, cause, started_at_ms, settled_at_ms)
agent runtime    ─▶ agent_runs / assistant_turns (status, fingerprint.stoppedAtLimit) + agent_exceptions
                                     │
            ai-analytics-refresh (every 5 min, per tenant, Temporal)
                                     ▼
 ai_work_units ── one row per run, assistant turn or standalone call: outcome, duration, calls, cost, tools
 ai_work_unit_tools ── a unit's tool calls by tool and cause, with the unit's model, provider, agent, task
 ai_provider_hourly_stats ── calls per hour × provider × model × task × agent × provider error,
                             fell-to-next, 64-bucket first-token histogram
 ai_analytics_refresh_state ── how far each tenant is folded in
                                     │
                    AnalyticsService.Analytics (15 reads, 4 at a time)
                                     ▼
          Redis (tenant epoch + scope, 2 min) ─▶ agentAnalytics GraphQL ─▶ Analytics tab
```

| Piece | Where |
|---|---|
| Domain (units, rollups, enums, palette, histogram, thresholds) | `internal/modules/agent/domain/aianalytics` |
| Ports and views | `agent/ports/aianalyticsports` (`AnalyticsRepository`, `AnalyticsService`, `AnalyticsCache`) |
| Service | `agent/app/aianalyticsservice` (`service.go`, `assemble.go`, `tasks.go`, `narrative.go`, `refresh.go`) |
| Repository and SQL | `agent/adapters/postgres/aianalyticsrepository` (`refreshsql.go`, `readsql.go`) |
| Cache | `infrastructure/redis/repositories/aianalyticscache.go` |
| Refresh workflow | `agent/jobs/aianalyticsjobs` (schedule `ai-analytics-refresh`, `RefreshAIAnalyticsWorkflow`) |
| Routing change | `aiprovider.PlanRouteTo`, `AIProviderService.RouteTask`, mutation `routeAITask` |
| GraphQL | `schema/aianalytics.graphqls`, `resolver/aianalyticsresolver` (permission `AgentRun` Read) |
| Web | `routes/agent-control/_components/analytics/`, query keys `queries.aiAnalytics` |
| Migrations | `20261231009250_ai_analytics.tx`, `20261231009260_ai_analytics_indexes` |
| Development seed | `seeds/development/38_ai_analytics.go` |

## What is recorded

- **Model calls.** Each attempt writes an `ai_usage_records` row. `completionrouter.classifyError`
  names the failure:

  | Class | When |
  |---|---|
  | `rate_limited` | HTTP 429 |
  | `auth` | HTTP 401 or 403 |
  | `provider_unavailable` | A retryable 5xx |
  | `network` | The connection failed |
  | `timeout` | The call timed out |
  | `refused` | The model declined |
  | `cancelled` | The caller went away |
  | `provider_error` | Anything else |

  `first_token_ms` and `recorded_at_ms` give the first token and when the call ended.
- **Tool calls.** Each tool step settles with a verdict. `runstepledger` adds the cause where the
  runtime knows it: wrong arguments, a tool the agent does not hold, the loop guard, a rule, a
  timeout, or a tool error. `aianalytics.ClassifyToolFailure` reads it, or derives it from the
  verdict. `started_at_ms` and `settled_at_ms` give the latency.
- **Run outcomes.** An agent run hands off when it raised an exception. It is stopped when its
  fingerprint says `stoppedAtLimit` (the tool budget ran out). It fails when its status is Failed.
  Otherwise it completed. An assistant turn reads its own status.

## The rollup

`RefreshAIAnalyticsWorkflow` runs every five minutes and fans out over every organization. `Refresh`, for each one:

1. Starts from the stored `refreshed_through`, less a 15-minute overlap, or backfills 30 days on a
   tenant's first refresh.
2. Folds in at most fourteen one-day chunks per cycle. Each chunk carries a millisecond
   `refresh_stamp`.
3. Upserts units for runs, turns and standalone calls (keyed per trace). It rebuilds those units'
   tool rows, and rebuilds the provider hours in the chunk's window.
4. Gives every new model name its canonical key (`aianalytics.CanonicalModel`), so
   `meta-llama/Llama-3.3-70B-Instruct` and `llama-3.3-70b-versatile` are one model. The key is
   applied to units, hours and tool rows.
5. Prunes everything older than 90 days.
6. Bumps the tenant's cache epoch when it wrote anything, including when a later step of the
   same cycle failed, so rows already saved never hide behind a cached answer.

Every write transaction of the refresh first takes a transaction-scoped advisory lock on the
tenant (`pg_advisory_xact_lock`). An activity retried while its first attempt is still running
waits for it instead of colliding on the hourly table's primary key.

Every table is a tenant table under FORCE row-level security, and the migration ends with
`trenova_rls.reconcile()`. The rollup is derived data: deleting it is safe, and the next refresh
rebuilds the last 30 days from the sources.

## The query

`agentAnalytics(input: {window, compareBy, agentDefinitionId, task, timezone})`.

**Grouping.** Everything is read per **deployment** (canonical model × provider) and folded into
**groups** (models or providers) in Go. Counts, costs, decisions, ratings and first-token
histograms add up, so they are read once. Percentiles do not, so outcome durations are read per
group and per deployment, and the trend is read per group.

**Thresholds.** The service owns every one, in `aianalytics/metrics.go`:

| Rule | Threshold |
|---|---|
| A group ranks | 30 runs |
| A tool cell ranks | 20 calls |
| A route is suggested | 5 points ahead of the current route |
| A tool-call outlier is named | 2× the rest, among at least 3 ranked groups |
| A run-time note appears | 10 failed runs, at 2× the completed median |
| A connection gap is named | 3 points |
| Failed tool calls read as a problem | over 3% |
| A provider's error rate reads as a problem | over 5% |
| Calls that fell to the next provider read as a warning | over 50 |
| A series or deployment says when it last ran | quiet for over a day before the window ends |

The view carries each of these as a flag (`toolFailureAlarm`, `errorAlarm`, `fellToNextAlarm`,
`quiet`), so the client only draws them. What the client still owns is presentation: the colour
scale of a heatmap cell and where a bar tops out.

**Same model, two providers.** `sharedModelPair` picks the model run on two providers with the
widest gap in completion between its best and worst deployment, among deployments that rank.
The view names it as `sameModel` (the model and both deployment keys), and both the same-model
card and the connection-gap sentence read that pair, so they always talk about the same one. The
sentence appears only when the gap is at least 3 points; the card shows whenever a pair exists.

**Short names.** Every provider carries a `shortName` for chart labels and event markers:
`stringutils.DistinctShortNames` takes the last word of the name ("AWS Bedrock" → "Bedrock"),
falls back to the first when two providers share it ("Workstation vLLM" and "Office vLLM" →
"Workstation", "Office"), and to the full name after that.

**Narrative.** `summarize` and `runTimeNote` build the Nova sentences as segments the client
renders. Segments that name a series or a provider link to it. Each sentence is one catalog
message in the viewer's language, with tags marking the linked and bold parts
(`"Failed runs on <p>{0}</p> take <b>{1}</b> as long as its completed ones."`).
`i18n.TranslateRich` splits the translated message at its tags, and each tag becomes a segment, so
a translator can move a name without breaking its link. Event labels and the fallback names
("Unknown model", "Removed provider") are translated the same way. Never build a sentence by
joining fragments.

**Events** come from two places: provider settings history (turned off or on, model switched) and
hourly health transitions (stopped answering, lost access, recovered).

**Colours.** Every series carries a `colorSlot`, 0–11. It is assigned from the hash of the model
key or provider id, over everything seen in the last 30 days, so a model keeps its colour whatever
is filtered. The client draws `var(--series-<slot + 1>)`.

**Trend buckets.** Buckets are 1 hour, 6 hours or 1 day, in the viewer's time zone: 24, 28 or 30
of them. `aianalytics.BucketStarts` works out the instant each bucket starts. Daily and 6-hour
buckets start on the local calendar, so a day with a clock change is 23 or 25 hours long. Hourly
buckets follow the zone's offset, so a half-hour zone's hours start on its own half hour. The trend
query assigns each run to the latest start at or before it with `width_bucket` over that list, and
the view returns the same list as `trendBuckets`. The client draws exactly those buckets, so the
axis and the data cannot disagree about where a day begins.

**Tool failures.** `agentAnalyticsToolFailures` returns a cell's failures by cause, and the eight
newest failed tool steps with their messages and agents.

## Cache

`AnalyticsCache` keys an answer by:

- organization and business unit
- the tenant's epoch
- window, comparison, agent, task and time zone
- the viewer's language, because the sentences are translated on the server

Entries live for two minutes. The epoch (`INCR`, kept seven days) is bumped by:

- the refresh, whenever it writes;
- every AI provider write: create, update, patch, delete, reorder, assign a task and route a task.

A new run is therefore visible after the next refresh, and a routing change made from the page
is visible on its next read. The cache fails open: a Redis error is logged, and the query is read
from Postgres. `until` and every "quiet" flag are as of when the answer was built, so they can be
up to two minutes old.

## Loading the tab

- The tab is a `lazyComponent` of the AI control page. Its code is preloaded when the pointer or
  focus reaches the tab, as every AI control tab's is.
- While its code loads, the page draws `AnalyticsTabSkeleton`, and while the query runs the
  tab draws `AnalyticsBodySkeleton`. Both live in `analytics-skeleton.tsx`, which the page
  imports up front and which draws only from `ai-control.css`, so it stands in before
  `analytics.css` arrives.
- The series and tool-failure sheets are `lazyComponent`s, mounted the first time one opens
  and preloaded with `whenIdle` once the analytics are on screen.
- No component here opts out of the React Compiler: nothing is virtualized, and nothing reads
  or writes a ref during render.

## Routing from the page

- **"Use X +n"** on a task row calls `routeAITask(task, providerId)`. `PlanRouteTo` refuses a target
  that is missing or off, and refuses a move that would leave a provider ahead of it with no task.
  Otherwise the task moves to the target, and providers ahead of it stop taking that task.
  `AIProviderService.RouteTask` applies the plan in one transaction and writes an audit entry. The
  write is exempt in `writecoverage.yml` as agent administration.
- **"Put X first"** on the same-model card reorders providers with `reorderAIProviders`, moving the
  better deployment's provider just ahead of the other.

## Testing it

- `aianalyticsservice` tests build `readings` by hand and assert the assembled view: grouping,
  "few", the narrative, task fit, provider health, the cache epoch, alarms, short names and the
  same-model pair.
- `aianalytics` tests cover bucket starts across a clock change and in a half-hour zone.
- `aianalyticsrepository/refresh_integration_test.go` (`-tags integration`) refreshes real rows into
  a container database.
- The client's `analytics/__tests__/analytics-model.test.ts` covers formatting, ranking, sentences,
  label placement, the trend and the same-model pair.
- The mock API (`e2e/mock-api/handlers/aianalytics.mjs`) ports the service's rules over a month of
  units, for screenshots.

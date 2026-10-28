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
 ai_work_units ── one row per run, assistant turn or standalone call: outcome, duration, calls,
                  tokens (uncached input, cache read, cache write, output), cost, tools
 ai_work_unit_tools ── a unit's tool calls by tool and cause, with the unit's model, provider, agent, task
 ai_provider_hourly_stats ── calls per hour × provider × model × task × agent × provider error,
                             fell-to-next, 64-bucket first-token histogram
 ai_analytics_refresh_state ── how far each tenant is folded in
                                     │
                    AnalyticsService.Analytics (17 reads, heaviest first, 4 at a time)
                                     ▼
          Redis (tenant epoch + scope, 2 min) ─▶ agentAnalytics GraphQL ─▶ Analytics tab
```

| Piece | Where |
|---|---|
| Domain (units, rollups, enums, palette, histogram, thresholds) | `internal/modules/agent/domain/aianalytics` |
| Ports and views | `agent/ports/aianalyticsports` (`AnalyticsReader`, `AnalyticsRefresher`, `AnalyticsService`, `AnalyticsCache`) |
| Service | `agent/app/aianalyticsservice` (`service.go`, `assemble.go`, `tasks.go`, `narrative.go`, `refresh.go`) |
| Repository and SQL | `agent/adapters/postgres/aianalyticsrepository` (`refreshsql.go`, `readsql.go`) |
| Cache | `infrastructure/redis/repositories/aianalyticscache.go` |
| Refresh workflow | `agent/jobs/aianalyticsjobs` (schedule `ai-analytics-refresh`, `RefreshAIAnalyticsWorkflow`) |
| Routing change | `aiprovider.PlanRouteTo`, `AIProviderService.RouteTask`, mutation `routeAITask` |
| GraphQL | `schema/aianalytics.graphqls`, `resolver/aianalyticsresolver` (permission `AgentRun` Read) |
| Web | `routes/agent-control/_components/analytics/`, query keys `queries.aiAnalytics` |
| Migrations | `20261231009250_ai_analytics.tx`, `20261231009260_ai_analytics_indexes`, `20261231009290_ai_analytics_refresh_indexes`, `20261231009950_ai_work_unit_cache_tokens.tx`, `20261231010210_ai_work_unit_advisor.tx` |
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
2. Asks whether anything changed since then (`HasActivity`): an agent run or assistant turn
   updated, or a usage record written. Three `EXISTS` probes answer it from tenant-leading
   indexes (`idx_agent_runs_tenant_updated`, `idx_assistant_turns_tenant_updated` and the usage
   records' own). A quiet tenant is marked caught up without rebuilding anything, so most of the
   five-minute cycles over an idle organization cost three index probes and one write.
3. Folds in at most fourteen one-day chunks per cycle. Each chunk carries a millisecond
   `refresh_stamp`, and the activity heartbeats the point it reached after every chunk (heartbeat
   timeout two minutes), so a stuck chunk is retried rather than waited out.
4. Upserts units for runs, turns and standalone calls (keyed per trace). A unit whose columns are
   all unchanged is not rewritten (`ON CONFLICT … WHERE … IS DISTINCT FROM`), so the overlap
   rewrites nothing and leaves no dead tuples. It rebuilds the changed units' tool rows, and
   rebuilds the provider hours in the chunk's window from every call but indexing and retrieval
   calls: agents, the assistant, standalone completions and evaluations all count toward a
   provider's health.
5. Gives every new model name its canonical key (`aianalytics.CanonicalModel`), so
   `meta-llama/Llama-3.3-70B-Instruct` and `llama-3.3-70b-versatile` are one model. The key is
   applied to units, hours and tool rows.
6. Prunes everything older than 90 days, at most once an hour (the first refresh of each hour),
   the Desk's agent suggestions (`agent_suggestions`) with it.
7. Bumps the tenant's cache epoch when it wrote anything, including when a later step of the
   same cycle failed, so rows already saved never hide behind a cached answer.

Every write transaction of the refresh first takes a transaction-scoped advisory lock on the
tenant (`pg_advisory_xact_lock`). An activity retried while its first attempt is still running
waits for it instead of colliding on the hourly table's primary key.

Every table is a tenant table under FORCE row-level security, and the migration ends with
`trenova_rls.reconcile()`. The rollup is derived data: deleting it is safe, and the next refresh
rebuilds the last 30 days from the sources.

- **Tokens.** A unit's prompt is stored in three parts whatever the protocol reported:
  `input_tokens` is the input the provider's cache did not serve, beside `cache_read_tokens`
  and `cache_write_tokens`. Anthropic reports cache reads and writes beside `input_tokens`;
  the OpenAI protocols count cached tokens inside it. The refresh reads each usage row's own
  `provider_kind` (kept on the row, so a deleted provider changes nothing) and takes the cached
  part out of the OpenAI rows, so a Claude unit and an OpenAI unit compare on the same terms (a
  Haiku turn used to read "4 in"). The usage rows keep what the provider reported. Units worked
  out before this split counted OpenAI's cached input in `input_tokens`; migration
  `20261231009950_ai_work_unit_cache_tokens` sets every organization's `refreshed_through` to
  0, so the next refresh rebuilds the whole retention window (90 days, 14 days a cycle) from the
  usage rows and no unit keeps the old meaning. `cost_usd` is priced per call by the provider's prices, with
  cache reads and writes at their own rates (the entered price, else the model's list rate from
  `aiprovider.ListPrice`, else the protocol's multiple) and Haiku 5.5's long-prompt rates above
  100,000 prompt tokens (per executor iteration when the call consulted the advisor). A call
  that consulted Anthropic's advisor carries the advisor's cost inside `cost_usd`, while its
  token columns are the executor's alone (the advisor's are in the usage row's `advisor_*`
  columns). Nothing here divides a cost by tokens: costs are read per run and per completed
  run, and tokens beside them, so the advisor's share needs no split. A per-token figure added
  later must take `advisor_cost_usd` out of the cost, or add the advisor's tokens, first.
  A call answered through a provider's batch API (an unattended suite's judge, see
  "Evaluations" in [agent-runtime.md](agent-runtime.md#evaluations)) is a usage row with
  `batched` true and `cost_usd` already at the batch discount (`aiprovider.BatchDiscount`,
  half of every token price, through `Provider.BatchCostForTask`); a per-token figure must
  read `batched` before comparing its rate with an unbatched row's. Its id is derived from the
  batch and the item (`pulid.Derived`) and it is written with `CreateOrSkip`, so reading the
  same results again records nothing twice. An item the provider never ran (expired,
  canceled) has no row; one it ran and failed has a failed row, as a synchronous attempt does.
  A unit keeps the advisor's part apart as well: `advisor_calls`, `advisor_prompt_tokens`
  (input, cache reads and cache writes together), `advisor_output_tokens` and
  `advisor_cost_usd`, summed from the usage rows' `advisor_*` columns. `cost_usd` still includes
  the advisor's cost; `advisor_cost_usd` is its part. Migration
  `20261231010210_ai_work_unit_advisor` added them and set every organization's
  `refreshed_through` to 0, so the next refresh rebuilds the retention window with them.

## The query

`agentAnalytics(input: {window, compareBy, agentDefinitionId, task, timezone})`.

**Tokens.** Every series and the totals carry `tokens` (`uncachedInput`, `cacheRead`,
`cacheWrite`, `output`, as `Decimal` because a window's sum can pass 32 bits) and
`cacheReadShare`, cache reads over every prompt token. They also carry `advisor` (`calls`,
`promptTokens`, `outputTokens`, `costUsd`; `aianalyticsports.AdvisorTotals`), what consulting
Anthropic's advisor came to, zero where no provider turned it on.

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
is filtered. The client draws `var(--series-<slot + 1>)`. The keys behind the assignment are read
once per tenant epoch and cached for an hour (`AnalyticsCache.Palette`), since every scope of the
same epoch shares them.

**Trend buckets.** Buckets are 1 hour, 6 hours or 1 day, in the viewer's time zone: 24, 28 or 30
of them. `aianalytics.BucketStarts` works out the instant each bucket starts. Daily and 6-hour
buckets start on the local calendar, so a day with a clock change is 23 or 25 hours long. Hourly
buckets follow the zone's offset, so a half-hour zone's hours start on its own half hour. The trend
query assigns each run to the latest start at or before it with `width_bucket` over that list, and
the view returns the same list as `trendBuckets`. A run before the first start is left out
rather than folded into the first bucket. The client draws exactly those buckets, so the axis and
the data cannot disagree about where a day begins.

**Hourly rows.** Provider hours, calls and first-token histograms are read from the first whole
hour inside the window, so a 24-hour window never counts the part of an hour before it. A
deployment's `lastCalledAt` is the end of the last hour it was called in, capped at `until`.
The first-token histogram query drops empty buckets before it aggregates; most of an hour's 64
buckets are empty, and filtering them first made the read three times faster at 500,000 runs.

**Task fit with one agent.** Routing is set per organization, so when the view is narrowed to one
agent the task rows show that agent's completion but suggest no route: a move justified by one
agent's runs would change every agent's provider.

**Tool failures.** `agentAnalyticsToolFailures` returns a cell's failures by cause, and the eight
newest failed tool steps with their messages and agents. The providers that served the cell come
from the same tool rows, so the sheet reads two queries and the provider list, not the whole
deployment table. The examples query picks the newest candidate units from the tool rollup first
and reads their steps only, rather than scanning every step in the window.

**Agent suggestions.** `suggestions` counts what became of the agents the Desk named for
people's questions (see "Matching a question to an agent" in
[agent-runtime.md](agent-runtime.md)), by the agent named, read straight from
`agent_suggestions` as the ratings are read from `ai_feedback`: switched to, sent to,
moved to without asking, kept the person's own agent, and passed over. A suggestion still
on screen for under half an hour (`aianalytics.SuggestionSettleSeconds`) is not counted
yet; one left longer counts as passed over. The taken rate is switched, sent to and moved
to over all of them. The section is absent when the view is narrowed to a task, since a
suggestion has none, or when nothing was suggested; narrowed to an agent it counts that
agent's. A new suggestion is visible once the two-minute cache entry lapses: recording one
does not bump the epoch.

## Cache

`AnalyticsCache` keys an answer by:

- organization and business unit
- the tenant's epoch
- window, comparison, agent and task
- a digest of the trend's bucket starts (`aianalytics.BucketsDigest`), not the time zone name:
  zones that start their buckets at the same instants share an entry, and a zone name is
  validated (64 characters at most, cached `time.LoadLocation`) before anything is read
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
- Hovering a series highlights it in every card without re-rendering any of them. Marks carry
  their key as `data-series`; `HighlightProvider` (`highlight.tsx`) holds the hovered key, sets
  `data-hot` on its own element and writes one rule, `[data-series="<key>"]{--an-hot:1}`, into a
  constructed stylesheet adopted by the document (not an inline `<style>`, which a strict
  `style-src` refuses). `analytics.css` turns `--an-hot` into opacity, the chip border and the
  trade-off dot's outline. Cards read only the stable `useSetHot()`, so a hover re-renders the
  provider's element and nothing else.

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
  "few", the narrative, task fit, provider health, the cache epoch, alarms, short names, the
  same-model pair and the agent suggestions.
- `aianalytics` tests cover bucket starts across a clock change and in a half-hour zone.
- `aianalyticsrepository/refresh_integration_test.go` (`-tags integration`) refreshes real rows into
  a container database.
- The client's `analytics/__tests__/analytics-model.test.ts` covers formatting, ranking, sentences,
  label placement, the trend and the same-model pair; `highlight.test.tsx` covers the highlight
  rule, its escaping, and that hovering re-renders no mark.
- The mock API (`e2e/mock-api/handlers/aianalytics.mjs`) ports the service's rules over a month of
  units, for screenshots.

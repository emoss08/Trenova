// AI analytics: agentAnalytics, agentAnalyticsToolFailures and routeAITask, aggregated from the month
// of units in fixtures/aianalytics.mjs by the rules aianalyticsservice uses (assemble.go, tasks.go,
// narrative.go): series and deployments busiest first, under 30 runs (20 calls for a tool cell)
// marked few and never ranked, a task's best only among two ranked series, a route suggestion only
// to a ranked deployment five points ahead of the current route that routing can reach, and the
// summary's three clauses and the run-time note.
import {
  ANALYTICS_PROVIDERS,
  FAILURE_MESSAGES,
  ROUTING_ORDER,
  analyticsEvents,
  createAnalyticsState,
} from "../fixtures/aianalytics.mjs";

const DAY = 86400;
const HOUR = 3600;
const WINDOW_SECONDS = { Day: DAY, Week: 7 * DAY, Month: 30 * DAY };
const BUCKET_SECONDS = { Day: HOUR, Week: 6 * HOUR, Month: DAY };
const MIN_RUNS_TO_RANK = 30;
const MIN_CALLS_TO_RANK = 20;
const MIN_ROUTE_GAIN = 5;
const MIN_CONNECTION_GAP = 3;
const MIN_MISTAKE_RATIO = 2;
const MIN_GROUPS_FOR_OUTLIER = 3;
const MIN_FAILED_RUNS_TO_NOTE = 10;
const TOOL_FAILURE_ALARM_RATE = 0.03;
const PROVIDER_ERROR_ALARM_RATE = 0.05;
const FELL_TO_NEXT_ALARM_CALLS = 50;
const QUIET_SECONDS = DAY;
const EVENT_LABELS = {
  TurnedOff: "{0} turned off",
  TurnedOn: "{0} turned on",
  StoppedAnswering: "{0} stopped answering",
  LostAccess: "{0} lost access",
  Recovered: "{0} recovered",
};
const MAX_TOOLS = 12;
const CAUSES = [
  "WrongArguments",
  "ToolTimedOut",
  "RepeatedCall",
  "BlockedByRule",
  "ToolError",
  "ToolNotHeld",
];
const MODEL_CAUSES = new Set(["WrongArguments", "RepeatedCall", "ToolNotHeld"]);
const PROVIDER_ERRORS = [
  "RateLimited",
  "TimedOut",
  "CouldNotConnect",
  "AccessDenied",
  "ServerError",
];
const OUTCOMES = ["Completed", "HandedOff", "Stopped", "Failed"];
const LABELS = {
  "llama-3.3-70b": ["Llama 3.3 70B", "Llama 70B"],
  "gpt-5.1-mini": ["GPT-5.1 mini", "5.1 mini"],
  "qwen-2.5-14b": ["Qwen 2.5 14B", "Qwen 14B"],
  "gpt-5.1": ["GPT-5.1", "GPT-5.1"],
  "claude-sonnet-5": ["Claude Sonnet 5", "Sonnet 5"],
  "gemini-3-flash": ["Gemini 3 Flash", "Gem. Flash"],
};
const CAUSE_WORDS = {
  WrongArguments: "wrong arguments",
  ToolTimedOut: "tools timing out",
  RepeatedCall: "repeating the same call",
  BlockedByRule: "calls a rule blocked",
  ToolError: "tools returning errors",
  ToolNotHeld: "calling tools it doesn't hold",
};

function analyticsOf(state) {
  state.ai.analytics ??= createAnalyticsState(state.anchor);
  return state.ai.analytics;
}

/** The last word of each name, else the first, else the whole name: whichever no other shares (stringutils.DistinctShortNames). */
function distinctShortNames(names) {
  const candidates = names.map((name) => {
    const trimmed = name.trim();
    const words = trimmed.split(/\s+/);
    if (words.length < 2) return [trimmed];
    const first = words[0];
    const last = words[words.length - 1];
    return first.toLowerCase() === last.toLowerCase() ? [last, trimmed] : [last, first, trimmed];
  });
  const level = names.map(() => 0);
  for (let round = 0; round < 3; round += 1) {
    const claims = new Map();
    candidates.forEach((list, index) => {
      const want = list[level[index]].toLowerCase();
      claims.set(want, (claims.get(want) ?? 0) + 1);
    });
    let moved = false;
    candidates.forEach((list, index) => {
      if (claims.get(list[level[index]].toLowerCase()) > 1 && level[index] < list.length - 1) {
        level[index] += 1;
        moved = true;
      }
    });
    if (!moved) break;
  }
  return candidates.map((list, index) => list[level[index]]);
}

let shortNames = null;
function shortNameOf(id) {
  if (!shortNames) {
    const providers = Object.values(ANALYTICS_PROVIDERS);
    const names = distinctShortNames(providers.map((provider) => provider.name));
    shortNames = new Map(providers.map((provider, index) => [provider.id, names[index]]));
  }
  return shortNames.get(id);
}

function providerRef(id) {
  const provider = ANALYTICS_PROVIDERS[id];
  if (!provider) {
    const name = id ? "Removed provider" : "No provider";
    return {
      id: id || null,
      name,
      shortName: name,
      kind: null,
      baseUrl: "",
      enabled: false,
      removed: true,
    };
  }
  return {
    id: provider.id,
    name: provider.name,
    shortName: shortNameOf(provider.id) ?? provider.name,
    kind: provider.kind,
    baseUrl: provider.baseUrl,
    enabled: provider.enabled,
    removed: false,
  };
}

function canonical(model) {
  return (
    Object.keys(LABELS).find((key) =>
      model
        .toLowerCase()
        .replaceAll(":", "-")
        .replace("qwen2", "qwen-2")
        .includes(key.replace("claude-", "")),
    ) ?? model
  );
}

const PALETTE_SLOTS = 12;

function fnv1a(key) {
  let hash = 0x811c9dc5;
  for (const byte of new TextEncoder().encode(key)) {
    hash = Math.imul(hash ^ byte, 0x01000193) >>> 0;
  }
  return hash >>> 0;
}

/** The server's palette: keys in hash order take their hash's slot, or the next free one. */
function paletteSlots(keys) {
  const ordered = [...new Set(keys)].sort((a, b) => fnv1a(a) - fnv1a(b) || a.localeCompare(b));
  const taken = new Set();
  const out = new Map();
  for (const key of ordered) {
    let slot = fnv1a(key) % PALETTE_SLOTS;
    for (let step = 0; step < PALETTE_SLOTS && taken.has(slot); step += 1)
      slot = (slot + 1) % PALETTE_SLOTS;
    taken.add(slot);
    out.set(key, slot);
  }
  return out;
}

const modelLabel = (key) => LABELS[key] ?? [key || "Unknown model", key || "Unknown model"];
const rate = (part, whole) => (whole > 0 ? part / whole : null);
const sum = (items, value) => items.reduce((total, item) => total + value(item), 0);
const deploymentKey = (unit) => `${unit.modelKey}|${unit.providerId}`;

function percentile(sorted, p) {
  if (sorted.length === 0) return null;
  const rank = (sorted.length - 1) * p;
  const low = Math.floor(rank);
  const high = Math.ceil(rank);
  return Math.round(sorted[low] + (sorted[high] - sorted[low]) * (rank - low));
}

function durations(units) {
  const sorted = units.map((unit) => unit.durationMs).sort((a, b) => a - b);
  return {
    count: sorted.length,
    p25: percentile(sorted, 0.25),
    p50: percentile(sorted, 0.5),
    p75: percentile(sorted, 0.75),
    p95: percentile(sorted, 0.95),
  };
}

function groupKeyOf(compareBy) {
  return compareBy === "Provider" ? (unit) => unit.providerId : (unit) => unit.modelKey;
}

function scope(state, input, now) {
  const since = now - (WINDOW_SECONDS[input.window] ?? WINDOW_SECONDS.Month);
  const units = analyticsOf(state).units.filter(
    (unit) =>
      unit.finishedAt >= since &&
      (!input.agentDefinitionId || unit.agentId === input.agentDefinitionId) &&
      (!input.task || unit.task === input.task),
  );
  return { since, units };
}

function groupBy(items, key) {
  const groups = new Map();
  for (const item of items) {
    const value = key(item);
    if (!groups.has(value)) groups.set(value, []);
    groups.get(value).push(item);
  }
  return groups;
}

const money = (value) => Number(value.toFixed(6)).toString();
const failuresOf = (units) => units.flatMap((unit) => unit.tools.flatMap((tool) => tool.failures));

function totalsOf(units) {
  const count = (outcome) => units.filter((unit) => unit.outcome === outcome).length;
  const completed = count("Completed");
  const toolCalls = sum(units, (unit) => sum(unit.tools, (tool) => tool.calls));
  const failures = failuresOf(units);
  const modelFailures = failures.filter((cause) => MODEL_CAUSES.has(cause)).length;
  const priced = units.filter((unit) => unit.cost !== null);
  const cost = sum(priced, (unit) => unit.cost);
  const completedDurations = durations(units.filter((unit) => unit.outcome === "Completed"));
  return {
    runs: units.length,
    modelCalls: sum(units, (unit) => unit.modelCalls),
    completed,
    handedOff: count("HandedOff"),
    stopped: count("Stopped"),
    failed: count("Failed"),
    completedRate: rate(completed, units.length),
    medianRunMs: completedDurations.p50,
    p95RunMs: completedDurations.p95,
    toolCalls,
    toolCallsFailed: failures.length,
    toolFailuresModel: modelFailures,
    toolFailureRate: rate(failures.length, toolCalls),
    modelShareOfToolFailures: rate(modelFailures, failures.length),
    costUsd: money(cost),
    pricedRuns: priced.length,
    costPerCompletedRun: priced.length > 0 && completed > 0 ? money(cost / completed) : null,
    toolFailureAlarm: (rate(failures.length, toolCalls) ?? 0) > TOOL_FAILURE_ALARM_RATE,
  };
}

/** One series' figures from its units; who it is comes from identify. */
function seriesFrom(key, members, identity) {
  const totals = totalsOf(members);
  const runTimes = OUTCOMES.map((outcome) => ({
    outcome,
    stats: durations(members.filter((unit) => unit.outcome === outcome)),
  }))
    .filter((entry) => entry.stats.count > 0)
    .map(({ outcome, stats }) => ({
      outcome,
      runs: stats.count,
      p25Ms: stats.p25,
      medianMs: stats.p50,
      p75Ms: stats.p75,
      p95Ms: stats.p95,
    }));
  const decisions = members.filter((unit) => unit.decision);
  const accepted = decisions.filter((unit) => unit.decision === "Accepted").length;
  const modified = decisions.filter((unit) => unit.decision === "Modified").length;
  const rated = members.filter((unit) => unit.rating !== null);
  const helpful = rated.filter((unit) => unit.rating > 0).length;
  const firstTokens = members.map((unit) => unit.firstTokenMs).sort((a, b) => a - b);
  const failures = failuresOf(members);
  const providers = [...groupBy(members, (unit) => unit.providerId).entries()]
    .sort((a, b) => b[1].length - a[1].length)
    .map(([id]) => providerRef(id));
  return {
    key,
    ...identity,
    providers: identity.providers ?? providers,
    few: members.length < MIN_RUNS_TO_RANK,
    runs: totals.runs,
    completed: totals.completed,
    handedOff: totals.handedOff,
    stopped: totals.stopped,
    failed: totals.failed,
    completedRate: totals.completedRate,
    medianRunMs: totals.medianRunMs,
    p95RunMs: totals.p95RunMs,
    modelCalls: totals.modelCalls,
    toolCalls: totals.toolCalls,
    toolCallsFailed: totals.toolCallsFailed,
    toolFailuresModel: totals.toolFailuresModel,
    toolFailureRate: totals.toolFailureRate,
    modelMistakeRate: totals.modelShareOfToolFailures,
    toolFailuresByCause: CAUSES.map((cause) => ({
      cause,
      blame: MODEL_CAUSES.has(cause) ? "Model" : "Tool",
      calls: failures.filter((item) => item === cause).length,
    })).filter((item) => item.calls > 0),
    costUsd: totals.costUsd,
    pricedRuns: totals.pricedRuns,
    costPerCompletedRun: totals.costPerCompletedRun,
    firstTokenMedianMs: percentile(firstTokens, 0.5),
    lastRanAt: Math.max(...members.map((unit) => unit.finishedAt)),
    runTimes,
    decided: decisions.length,
    accepted,
    modified,
    rejected: decisions.length - accepted - modified,
    approvalRate: rate(accepted, decisions.length),
    rated: rated.length,
    helpful,
    helpfulRate: rate(helpful, rated.length),
  };
}

const busiestFirst = (list) =>
  list.sort((a, b) => b.runs - a.runs || a.label.localeCompare(b.label));

function seriesOf(units, compareBy, palette) {
  return busiestFirst(
    [...groupBy(units, groupKeyOf(compareBy)).entries()].map(([key, members]) => {
      if (compareBy === "Provider") {
        const ref = providerRef(key);
        return seriesFrom(key, members, {
          label: ref.name,
          shortLabel: ref.shortName,
          modelKey: null,
          providerId: key,
          colorSlot: palette.providers.get(key) ?? 0,
          models: [...new Set(members.map((unit) => modelLabel(unit.modelKey)[0]))],
        });
      }
      const [label, shortLabel] = modelLabel(key);
      return seriesFrom(key, members, {
        label,
        shortLabel,
        modelKey: key,
        providerId: null,
        colorSlot: palette.models.get(key) ?? 0,
        models: [],
      });
    }),
  );
}

function deploymentsOf(units, palette) {
  return busiestFirst(
    [...groupBy(units, deploymentKey).entries()].map(([key, members]) => {
      const { modelKey, providerId } = members[0];
      const ref = providerRef(providerId);
      return seriesFrom(key, members, {
        label: ref.name,
        shortLabel: ref.shortName,
        modelKey,
        providerId,
        colorSlot: palette.providers.get(providerId) ?? 0,
        models: [modelLabel(modelKey)[0]],
        providers: [providerRef(providerId)],
      });
    }),
  );
}

/** The zone's offset from UTC at an instant, in seconds. */
function offsetAt(at, timezone) {
  const parts = Object.fromEntries(
    new Intl.DateTimeFormat("en-US", {
      timeZone: timezone,
      hourCycle: "h23",
      year: "numeric",
      month: "numeric",
      day: "numeric",
      hour: "numeric",
      minute: "numeric",
      second: "numeric",
    })
      .formatToParts(new Date(at * 1000))
      .map((part) => [part.type, Number(part.value)]),
  );
  const local =
    Date.UTC(parts.year, parts.month - 1, parts.day, parts.hour, parts.minute, parts.second) / 1000;
  return local - at;
}

/** The start of the bucket an instant falls in, at the zone's local boundaries (aianalytics.BucketStarts). */
function alignBucket(at, size, timezone) {
  const step = size / HOUR;
  if (size > HOUR && 24 % step === 0) {
    const local = new Date((at + offsetAt(at, timezone)) * 1000);
    const hour = local.getUTCHours() - (local.getUTCHours() % step);
    const guess =
      Date.UTC(local.getUTCFullYear(), local.getUTCMonth(), local.getUTCDate(), hour) / 1000;
    let start = guess - offsetAt(guess, timezone);
    start = guess - offsetAt(start, timezone);
    if (start <= at) return start;
  }
  const rest = (((at + offsetAt(at, timezone)) % size) + size) % size;
  return at - rest;
}

function bucketStarts(window, timezone, until) {
  const size = BUCKET_SECONDS[window] ?? DAY;
  const count = Math.round((WINDOW_SECONDS[window] ?? WINDOW_SECONDS.Month) / size);
  const starts = [];
  let at = alignBucket(until, size, timezone);
  while (starts.length < count) {
    starts.push(at);
    at = alignBucket(at - 1, size, timezone);
  }
  return starts.reverse();
}

/** The bucket a run lands in: the latest start at or before it, the first for anything earlier. */
function bucketOf(starts, at) {
  let index = 0;
  while (index + 1 < starts.length && starts[index + 1] <= at) index += 1;
  return starts[index];
}

function trendOf(units, compareBy, starts) {
  const key = groupKeyOf(compareBy);
  const groups = groupBy(units, (unit) => `${key(unit)}|${bucketOf(starts, unit.finishedAt)}`);
  return [...groups.entries()]
    .map(([group, members]) => {
      const [seriesKey, bucket] = group.split("|");
      const completed = members.filter((unit) => unit.outcome === "Completed");
      const toolCalls = sum(members, (unit) => sum(unit.tools, (tool) => tool.calls));
      const failures = failuresOf(members).length;
      return {
        seriesKey,
        bucketStart: Number(bucket),
        runs: members.length,
        completedRate: rate(completed.length, members.length),
        toolFailureRate: rate(failures, toolCalls),
        medianRunMs: durations(completed).p50,
      };
    })
    .sort((a, b) => a.bucketStart - b.bucketStart || a.seriesKey.localeCompare(b.seriesKey));
}

function toolsOf(units, compareBy) {
  const key = groupKeyOf(compareBy);
  const tools = new Map();
  for (const unit of units) {
    for (const tool of unit.tools) {
      const row = tools.get(tool.name) ?? {
        name: tool.name,
        calls: 0,
        failed: 0,
        cells: new Map(),
      };
      const cell = row.cells.get(key(unit)) ?? { seriesKey: key(unit), calls: 0, failed: 0 };
      cell.calls += tool.calls;
      cell.failed += tool.failures.length;
      row.calls += tool.calls;
      row.failed += tool.failures.length;
      row.cells.set(key(unit), cell);
      tools.set(tool.name, row);
    }
  }
  const ranks = (row) => row.calls >= MIN_CALLS_TO_RANK;
  return [...tools.values()]
    .map((row) => ({ ...row, failureRate: rate(row.failed, row.calls) }))
    .sort(
      (a, b) =>
        Number(ranks(b)) - Number(ranks(a)) ||
        (b.failureRate ?? 0) - (a.failureRate ?? 0) ||
        b.calls - a.calls ||
        a.name.localeCompare(b.name),
    )
    .slice(0, MAX_TOOLS)
    .map((row) => ({
      name: row.name,
      calls: row.calls,
      failed: row.failed,
      failureRate: row.failureRate,
      cells: [...row.cells.values()]
        .map((cell) => ({
          ...cell,
          few: cell.calls < MIN_CALLS_TO_RANK,
          failureRate: rate(cell.failed, cell.calls),
        }))
        .sort((a, b) => a.seriesKey.localeCompare(b.seriesKey)),
    }));
}

function routeFor(routing, task) {
  return (
    ROUTING_ORDER.find((id) => ANALYTICS_PROVIDERS[id].enabled && routing[id]?.includes(task)) ??
    null
  );
}

function canRouteTo(routing, task, providerId) {
  const provider = ANALYTICS_PROVIDERS[providerId];
  if (!provider?.enabled) return false;
  const index = ROUTING_ORDER.indexOf(providerId);
  return (
    ROUTING_ORDER.slice(0, index).every(
      (id) =>
        !(ANALYTICS_PROVIDERS[id].enabled && routing[id]?.includes(task)) || routing[id].length > 1,
    ) && routeFor(routing, task) !== providerId
  );
}

const completion = (members) =>
  members.filter((unit) => unit.outcome === "Completed").length / Math.max(1, members.length);

function tasksOf(units, compareBy, routing) {
  const key = groupKeyOf(compareBy);
  const rows = [];
  for (const [task, members] of groupBy(units, (unit) => unit.task)) {
    const cells = [...groupBy(members, key).entries()]
      .map(([seriesKey, cellUnits]) => {
        const completed = cellUnits.filter((unit) => unit.outcome === "Completed").length;
        return {
          seriesKey,
          few: cellUnits.length < MIN_RUNS_TO_RANK,
          runs: cellUnits.length,
          completed,
          completedRate: rate(completed, cellUnits.length),
        };
      })
      .sort((a, b) => a.seriesKey.localeCompare(b.seriesKey));
    const ranked = cells
      .filter((cell) => !cell.few)
      .sort((a, b) => b.completedRate - a.completedRate || b.runs - a.runs);
    const best = ranked.length >= 2 ? ranked[0] : null;
    const currentId = routeFor(routing, task);
    const currentKey = currentId
      ? compareBy === "Provider"
        ? currentId
        : canonical(ANALYTICS_PROVIDERS[currentId].model)
      : null;

    let suggestion = null;
    const currentUnits = currentId ? members.filter((unit) => unit.providerId === currentId) : [];
    if (currentUnits.length >= MIN_RUNS_TO_RANK) {
      const candidates = [...groupBy(members, deploymentKey).values()]
        .filter((list) => list[0].providerId !== currentId && list.length >= MIN_RUNS_TO_RANK)
        .sort((a, b) => completion(b) - completion(a) || b.length - a.length);
      for (const candidate of candidates) {
        const gain = (completion(candidate) - completion(currentUnits)) * 100;
        if (gain < MIN_ROUTE_GAIN) break;
        const { providerId, modelKey } = candidate[0];
        if (
          !canRouteTo(routing, task, providerId) ||
          canonical(ANALYTICS_PROVIDERS[providerId].model) !== modelKey
        )
          continue;
        suggestion = {
          provider: providerRef(providerId),
          seriesKey: key(candidate[0]),
          gainPoints: gain,
        };
        break;
      }
    }
    rows.push({
      task,
      current: currentId ? providerRef(currentId) : null,
      currentSeriesKey: currentKey,
      bestSeriesKey: best?.seriesKey ?? null,
      cells,
      suggestion,
      _runs: members.length,
    });
  }
  return rows.sort((a, b) => b._runs - a._runs).map(({ _runs, ...row }) => row);
}

function providerHealthOf(state, units, since, until, input) {
  const health = new Map();
  const entry = (providerId, modelKey) => {
    const key = `${modelKey}|${providerId}`;
    if (!health.has(key))
      health.set(key, {
        key,
        provider: providerRef(providerId),
        modelKey,
        model: modelLabel(modelKey)[0],
        lastCalledAt: 0,
        calls: 0,
        errors: 0,
        fellToNext: 0,
        firstTokens: [],
        kinds: new Map(),
      });
    return health.get(key);
  };
  for (const unit of units) {
    const item = entry(unit.providerId, unit.modelKey);
    item.calls += unit.modelCalls;
    item.firstTokens.push(unit.firstTokenMs);
    item.lastCalledAt = Math.max(item.lastCalledAt, unit.finishedAt - (unit.finishedAt % HOUR));
  }
  for (const error of analyticsOf(state).errors) {
    if (
      error.at < since ||
      (input.task && error.task !== input.task) ||
      (input.agentDefinitionId && error.agentId !== input.agentDefinitionId)
    )
      continue;
    const item = entry(error.providerId, error.model);
    item.calls += 1;
    item.errors += 1;
    item.fellToNext += error.fell ? 1 : 0;
    item.kinds.set(error.kind, (item.kinds.get(error.kind) ?? 0) + 1);
    item.lastCalledAt = Math.max(item.lastCalledAt, error.at - (error.at % HOUR));
  }
  return [...health.values()]
    .map((item) => ({
      key: item.key,
      provider: item.provider,
      modelKey: item.modelKey,
      model: item.model,
      lastCalledAt: item.lastCalledAt || null,
      quiet: item.lastCalledAt > 0 && until - item.lastCalledAt > QUIET_SECONDS,
      calls: item.calls,
      errors: item.errors,
      errorRate: rate(item.errors, item.calls),
      errorAlarm: (rate(item.errors, item.calls) ?? 0) > PROVIDER_ERROR_ALARM_RATE,
      fellToNext: item.fellToNext,
      fellToNextAlarm: item.fellToNext > FELL_TO_NEXT_ALARM_CALLS,
      firstTokenMedianMs: percentile(
        item.firstTokens.sort((a, b) => a - b),
        0.5,
      ),
      errorsByKind: PROVIDER_ERRORS.filter((kind) => item.kinds.has(kind)).map((kind) => ({
        kind,
        calls: item.kinds.get(kind),
      })),
    }))
    .sort((a, b) => b.calls - a.calls || a.key.localeCompare(b.key));
}

const seg = (text, extra = {}) => ({
  text,
  strong: false,
  tone: null,
  target: null,
  seriesKey: null,
  providerId: null,
  ...extra,
});
const percent = (value) => `${Math.round(value * 100)}%`;

/** The model on two providers with the widest gap in completion (narrative.go sharedModelPair). */
function sharedModelPair(deployments) {
  let widest = null;
  for (const [model, list] of groupBy(
    deployments.filter((item) => !item.few && item.modelKey && item.providerId),
    (item) => item.modelKey,
  )) {
    if (list.length < 2) continue;
    const sorted = [...list].sort(
      (a, b) =>
        (b.completedRate ?? 0) - (a.completedRate ?? 0) ||
        b.runs - a.runs ||
        a.key.localeCompare(b.key),
    );
    const gap =
      ((sorted[0].completedRate ?? 0) - (sorted[sorted.length - 1].completedRate ?? 0)) * 100;
    if (!widest || gap > widest.gap || (gap === widest.gap && model < widest.model))
      widest = { model, better: sorted[0], worse: sorted[sorted.length - 1], gap };
  }
  return widest;
}

function summaryOf(input, series, deployments, compareBy) {
  const phrase =
    { Day: "Over the last 24 hours", Week: "Over the last 7 days", Month: "Over the last 30 days" }[
      input.window
    ] ?? "Over the last 30 days";
  if (series.length === 0)
    return [
      seg(
        `${phrase}, no agent run, conversation or AI call finished, so there is nothing to compare yet.`,
      ),
    ];
  const out = [];
  const ranked = series.filter((item) => !item.few);
  const leader = [...ranked].sort((a, b) => b.completedRate - a.completedRate)[0];
  if (leader) {
    out.push(
      seg(`${phrase}, `),
      seg(leader.label, { target: "Series", seriesKey: leader.key }),
      ...(ranked.length === 1
        ? [
            seg(" completed "),
            seg(percent(leader.completedRate), { strong: true }),
            seg(" of its runs."),
          ]
        : [
            seg(" completed the most of its runs — "),
            seg(percent(leader.completedRate), { strong: true }),
            seg("."),
          ]),
      seg(" "),
    );
  } else {
    out.push(
      seg(`${phrase}, `),
      seg(series[0].label, { target: "Series", seriesKey: series[0].key }),
      seg(
        ` did the most work, but nothing has run often enough yet to say which ${compareBy === "Provider" ? "provider" : "model"} finishes more of its runs.`,
      ),
    );
  }

  const pair = sharedModelPair(deployments);
  const widest = pair && pair.gap >= MIN_CONNECTION_GAP ? pair : null;
  if (widest) {
    out.push(
      seg(modelLabel(widest.model)[0], { strong: true }),
      seg(` completed ${percent(widest.better.completedRate)} on `),
      seg(widest.better.label, { target: "Provider", providerId: widest.better.providerId }),
      seg(` but ${percent(widest.worse.completedRate)} on `),
      seg(widest.worse.label, {
        target: "Provider",
        providerId: widest.worse.providerId,
        tone: "danger",
      }),
      seg(". It's the same model, so the gap comes from the connection. "),
    );
  }

  if (ranked.length >= MIN_GROUPS_FOR_OUTLIER) {
    const share = (item) => (item.toolCalls > 0 ? item.toolCallsFailed / item.toolCalls : 0);
    const worst = ranked
      .filter((item) => item.toolCalls > 0)
      .sort((a, b) => share(b) - share(a))[0];
    const rest = ranked.filter((item) => item !== worst);
    const restCalls = sum(rest, (item) => item.toolCalls);
    const restFailed = sum(rest, (item) => item.toolCallsFailed);
    if (worst && worst.toolCallsFailed > 0 && restCalls > 0 && restFailed > 0) {
      const ratio = share(worst) / (restFailed / restCalls);
      if (ratio >= MIN_MISTAKE_RATIO) {
        const top = [...worst.toolFailuresByCause].sort((a, b) => b.calls - a.calls)[0];
        const clause = top ? `, mostly ${CAUSE_WORDS[top.cause]}.` : ".";
        out.push(
          seg(worst.label, { target: "Series", seriesKey: worst.key }),
          seg(" gets tool calls wrong "),
          seg(`${ratio.toFixed(1)}×`, { strong: true }),
          seg(` as often as the rest${clause}`),
        );
      }
    }
  }
  const last = out[out.length - 1];
  last.text = last.text.trimEnd();
  return out;
}

function runTimeNoteOf(deployments) {
  let slowest = null;
  for (const deployment of deployments) {
    const completed = deployment.runTimes.find((item) => item.outcome === "Completed");
    const failed = deployment.runTimes.find((item) => item.outcome === "Failed");
    if (
      !completed ||
      !failed ||
      completed.runs < MIN_RUNS_TO_RANK ||
      failed.runs < MIN_FAILED_RUNS_TO_NOTE ||
      !completed.medianMs
    )
      continue;
    const ratio = failed.medianMs / completed.medianMs;
    if (ratio >= 2 && (!slowest || ratio > slowest.ratio)) slowest = { deployment, ratio, failed };
  }
  if (!slowest) return [];
  const provider = ANALYTICS_PROVIDERS[slowest.deployment.providerId];
  const out = [
    seg("Failed runs on "),
    seg(provider.name, { target: "Provider", providerId: provider.id, tone: "warn" }),
    seg(" take "),
    seg(`${slowest.ratio.toFixed(1)}×`, { strong: true }),
    seg(" as long as its completed ones."),
  ];
  if (slowest.failed.medianMs >= 0.8 * provider.timeout * 1000)
    out.push(seg(` Most of them wait out the ${provider.timeout}-second timeout first.`));
  return out;
}

function agentAnalytics(state, input) {
  if (state.scenario.aiAnalytics === "error") throw new Error("analytics read failed");
  if (state.scenario.aiAnalytics === "hold") return { __hold: true };
  const now = state.now();
  const { since, units } = scope(state, input, now);
  const compareBy = input.compareBy ?? "Model";
  const bucketSeconds = BUCKET_SECONDS[input.window] ?? DAY;
  const analytics = analyticsOf(state);
  const palette = {
    models: paletteSlots(analytics.units.map((unit) => unit.modelKey)),
    providers: paletteSlots(analytics.units.map((unit) => unit.providerId)),
  };
  const markQuiet = (list) =>
    list.map((item) => ({ ...item, quiet: now - item.lastRanAt > QUIET_SECONDS }));
  const series = markQuiet(seriesOf(units, compareBy, palette));
  const deployments = markQuiet(deploymentsOf(units, palette));
  const starts = bucketStarts(input.window, input.timezone || "UTC", now);
  const pair = sharedModelPair(deployments);
  return {
    window: input.window,
    compareBy,
    since,
    until: now,
    bucketSeconds,
    trendBuckets: starts,
    refreshedAt: analytics.refreshedAt,
    summary: summaryOf(input, series, deployments, compareBy),
    runTimeNote: runTimeNoteOf(deployments),
    totals: totalsOf(units),
    series,
    deployments,
    trend: trendOf(units, compareBy, starts),
    events: analyticsEvents(state.anchor)
      .filter((event) => event.at >= since)
      .map(({ providerId, ...event }) => {
        const provider = providerRef(providerId);
        return {
          ...event,
          label: EVENT_LABELS[event.kind].replace("{0}", provider.shortName),
          provider,
        };
      }),
    tools: toolsOf(units, compareBy),
    tasks: tasksOf(units, compareBy, analytics.routing),
    providers: providerHealthOf(state, units, since, now, input),
    sameModel: pair
      ? { modelKey: pair.model, betterKey: pair.better.key, worseKey: pair.worse.key }
      : null,
  };
}

function toolFailures(state, input, seriesKey, toolName) {
  const { units } = scope(state, input, state.now());
  const key = groupKeyOf(input.compareBy ?? "Model");
  const members = units.filter((unit) => key(unit) === seriesKey);
  const causes = new Map();
  let calls = 0;
  const recent = [];
  for (const unit of [...members].reverse()) {
    for (const tool of unit.tools) {
      if (tool.name !== toolName) continue;
      calls += tool.calls;
      for (const cause of tool.failures) {
        causes.set(cause, (causes.get(cause) ?? 0) + 1);
        if (recent.length < 8) {
          const agent = state.ai.agents.find((candidate) => candidate.node.id === unit.agentId);
          const message =
            cause === "WrongArguments"
              ? (FAILURE_MESSAGES.WrongArguments[toolName] ??
                "The arguments did not match the tool's schema")
              : FAILURE_MESSAGES[cause];
          recent.push({
            unitId: unit.id,
            source: unit.source,
            toolName,
            message,
            agentId: agent?.node.id ?? null,
            agentName: agent?.node.name ?? null,
            at: unit.finishedAt,
          });
        }
      }
    }
  }
  const failed = sum([...causes.values()], (value) => value);
  return {
    seriesKey,
    toolName,
    calls,
    failed,
    causes: [...causes.entries()]
      .map(([cause, count]) => ({
        cause,
        blame: MODEL_CAUSES.has(cause) ? "Model" : "Tool",
        calls: count,
      }))
      .sort((a, b) => b.calls - a.calls || a.cause.localeCompare(b.cause)),
    recent,
    providers: [...groupBy(members, (unit) => unit.providerId).entries()]
      .sort((a, b) => b[1].length - a[1].length)
      .map(([id]) => providerRef(id)),
  };
}

function routeTask(state, task, providerId) {
  const analytics = analyticsOf(state);
  if (!ANALYTICS_PROVIDERS[providerId]) throw new Error("AI provider not found");
  if (!canRouteTo(analytics.routing, task, providerId))
    throw new Error("This provider can't take the task");
  const index = ROUTING_ORDER.indexOf(providerId);
  for (const id of ROUTING_ORDER.slice(0, index)) {
    analytics.routing[id] = (analytics.routing[id] ?? []).filter((candidate) => candidate !== task);
  }
  if (!analytics.routing[providerId].includes(task)) analytics.routing[providerId].push(task);
  return ROUTING_ORDER.map((id, order) => ({
    id,
    tasks: analytics.routing[id],
    priority: (order + 1) * 10,
    version: 5,
  }));
}

export const AI_ANALYTICS_HANDLERS = {
  AgentAnalytics: (state, v) => {
    const analytics = agentAnalytics(state, v.input);
    return analytics.__hold ? analytics : { agentAnalytics: analytics };
  },
  AgentAnalyticsToolFailures: (state, v) => ({
    agentAnalyticsToolFailures: toolFailures(state, v.input, v.seriesKey, v.toolName),
  }),
  RouteAITask: (state, v) => ({ routeAITask: routeTask(state, v.task, v.providerId) }),
};

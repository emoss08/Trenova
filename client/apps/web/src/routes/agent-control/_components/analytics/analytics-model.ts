import { escapeCsvValue, spreadsheetSafeText } from "@/lib/data-table-export";
import type {
  AgentAnalytics,
  AIAnalyticsSeries,
  AIAnalyticsTrendPoint,
} from "@/lib/graphql/ai-analytics";
import type { RichTags, RichTranslateFn } from "@trenova/shared/i18n/rich";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  AiAnalyticsCompareBy,
  AiAnalyticsWindow,
  AiProviderErrorKind,
  AiTask,
  AiToolFailureCause,
  AiWorkOutcome,
} from "@trenova/graphql/generated/graphql";
import { formatNumber } from "@trenova/shared/i18n/format";
import { formatCurrency, formatPercent } from "@trenova/shared/lib/utils";
import { createElement, type ReactNode } from "react";

export const ANALYTICS_WINDOWS = [
  "Day",
  "Week",
  "Month",
] as const satisfies readonly AiAnalyticsWindow[];
export const ANALYTICS_COMPARE = [
  "Model",
  "Provider",
] as const satisfies readonly AiAnalyticsCompareBy[];

export const WINDOW_OPTIONS: readonly { value: AiAnalyticsWindow; label: string }[] = [
  { value: "Day", label: "24 hours" },
  { value: "Week", label: "7 days" },
  { value: "Month", label: "30 days" },
];

export const WINDOW_PHRASES: Record<AiAnalyticsWindow, { label: string }> = {
  Day: { label: "the last 24 hours" },
  Week: { label: "the last 7 days" },
  Month: { label: "the last 30 days" },
};

export const OUTCOMES: readonly { outcome: AiWorkOutcome; label: string; title: string }[] = [
  { outcome: "Completed", label: "Completed", title: "Completed" },
  { outcome: "HandedOff", label: "Handed off", title: "Handed to a person" },
  { outcome: "Stopped", label: "Stopped", title: "Stopped at a limit" },
  { outcome: "Failed", label: "Failed", title: "Failed" },
];

export type Blame = "Model" | "Tool";

/** Why a tool call failed, the model's mistakes first, in the order the charts stack them. */
export const CAUSES: readonly { cause: AiToolFailureCause; label: string; blame: Blame }[] = [
  { cause: "WrongArguments", label: "Wrong arguments", blame: "Model" },
  { cause: "ToolNotHeld", label: "Called a tool it doesn't hold", blame: "Model" },
  { cause: "RepeatedCall", label: "Repeated the same call", blame: "Model" },
  { cause: "ToolError", label: "Tool returned an error", blame: "Tool" },
  { cause: "BlockedByRule", label: "Blocked by a tool rule", blame: "Tool" },
  { cause: "ToolTimedOut", label: "Tool timed out", blame: "Tool" },
];

export const PROVIDER_ERRORS: readonly { kind: AiProviderErrorKind; label: string }[] = [
  { kind: "RateLimited", label: "Rate limited" },
  { kind: "TimedOut", label: "Timed out" },
  { kind: "CouldNotConnect", label: "Couldn't connect" },
  { kind: "AccessDenied", label: "Access denied" },
  { kind: "ServerError", label: "Server error" },
];

/** Each series' hue: the palette slot the server holds for its model or provider, by key. */
export function seriesColors(
  items: readonly { key: string; colorSlot: number }[],
): ReadonlyMap<string, string> {
  return new Map(items.map((item) => [item.key, `var(--series-${item.colorSlot + 1})`]));
}

export type SeriesColors = ReadonlyMap<string, string>;

export function percent(value: number | null | undefined, digits = 0): string {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return "—";
  }

  return formatPercent(value * 100, digits);
}

/** A failure share: one decimal under ten percent, under 0.1% said as such. */
export function failureShare(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return "—";
  }
  if (value === 0) {
    return "0";
  }
  if (value < 0.001) {
    return "<0.1%";
  }

  return percent(value, value < 0.1 ? 1 : 0);
}

/** A run time in the unit a person reads it: 4.1s, 13s, 1m 09s. */
export function formatRunTime(ms: number | null | undefined): string {
  if (ms === null || ms === undefined || !Number.isFinite(ms) || ms < 0) {
    return "—";
  }
  const seconds = ms / 1000;
  if (seconds >= 60) {
    const whole = Math.round(seconds);
    return `${Math.floor(whole / 60)}m ${String(whole % 60).padStart(2, "0")}s`;
  }
  if (seconds < 10) {
    return `${seconds.toFixed(1)}s`;
  }

  return `${Math.round(seconds)}s`;
}

/** A latency: 820ms under a second, 1.42s above. */
export function formatLatency(ms: number | null | undefined): string {
  if (ms === null || ms === undefined || !Number.isFinite(ms) || ms < 0) {
    return "—";
  }
  if (ms < 1000) {
    return `${Math.round(ms)}ms`;
  }

  return `${(ms / 1000).toFixed(2)}s`;
}

/** A cost per run, to a tenth of a cent: runs are cheap, so cents alone would read $0.00. */
export function formatCost(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return "—";
  }

  return formatNumber(value, {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 3,
    maximumFractionDigits: 3,
  });
}

export function formatSpend(value: number): string {
  return formatCurrency(value);
}

/** "1 in 25" for a rate; a rate of nothing is "none". */
export function oneIn(t: TranslateFn, rate: number): string {
  if (rate <= 0) {
    return t("none");
  }

  return t("1 in {0}", Math.max(2, Math.round(1 / rate)));
}

function amount(value: string | null | undefined): number | null {
  if (value === null || value === undefined) {
    return null;
  }
  const parsed = Number(value);

  return Number.isFinite(parsed) ? parsed : null;
}

/** A series nothing it served carried a charge for runs on hardware the organization owns. */
export function isLocal(series: AIAnalyticsSeries): boolean {
  return series.runs > 0 && series.pricedRuns === 0;
}

export function costPerCompleted(series: AIAnalyticsSeries): number | null {
  if (isLocal(series)) {
    return 0;
  }

  return amount(series.costPerCompletedRun);
}

export type MetricKey =
  | "completed"
  | "runTime"
  | "toolFailures"
  | "cost"
  | "approval"
  | "firstToken"
  | "helpful";

export type Metric = {
  key: MetricKey;
  label: string;
  description: string;
  higherIsBetter: boolean;
  share: boolean;
  value: (series: AIAnalyticsSeries) => number | null;
  format: (value: number) => string;
};

/** Every figure the charts compare, formatted once. The definitions themselves are the server's. */
export const METRICS: Record<MetricKey, Metric> = {
  completed: {
    key: "completed",
    label: "Completed",
    description: "Runs that finished without handing off or failing",
    higherIsBetter: true,
    share: true,
    value: (series) => series.completedRate ?? null,
    format: (value) => percent(value),
  },
  runTime: {
    key: "runTime",
    label: "Median run time",
    description: "Completed runs, from the first token to the last tool call",
    higherIsBetter: false,
    share: false,
    value: (series) => series.medianRunMs ?? null,
    format: formatRunTime,
  },
  toolFailures: {
    key: "toolFailures",
    label: "Tool calls that failed",
    description: "Out of every tool call the model made",
    higherIsBetter: false,
    share: true,
    value: (series) => (series.toolCalls > 0 ? (series.toolFailureRate ?? 0) : null),
    format: (value) => percent(value, 1),
  },
  cost: {
    key: "cost",
    label: "Cost per completed run",
    description: "What the provider charged, divided by runs that finished",
    higherIsBetter: false,
    share: false,
    value: costPerCompleted,
    format: formatCost,
  },
  approval: {
    key: "approval",
    label: "Approved as proposed",
    description: "Proposals a person accepted without changing",
    higherIsBetter: true,
    share: true,
    value: (series) => series.approvalRate ?? null,
    format: (value) => percent(value),
  },
  firstToken: {
    key: "firstToken",
    label: "Time to first token",
    description: "How long before the model starts answering",
    higherIsBetter: false,
    share: false,
    value: (series) => series.firstTokenMedianMs ?? null,
    format: formatLatency,
  },
  helpful: {
    key: "helpful",
    label: "Rated helpful",
    description: "Thumbs up from the people who rated an answer",
    higherIsBetter: true,
    share: true,
    value: (series) => series.helpfulRate ?? null,
    format: (value) => percent(value),
  },
};

export const LEADERBOARD: readonly MetricKey[] = [
  "completed",
  "runTime",
  "toolFailures",
  "cost",
  "approval",
  "firstToken",
];
export const TRADEOFF_UP: readonly MetricKey[] = ["completed", "approval", "helpful"];
export const TRADEOFF_ACROSS: readonly MetricKey[] = [
  "runTime",
  "cost",
  "firstToken",
  "toolFailures",
];
export const SAME_MODEL_METRICS: readonly MetricKey[] = [
  "completed",
  "runTime",
  "toolFailures",
  "firstToken",
  "approval",
];

/** A metric's value as shown: a cost of nothing is a local provider's, not a free one. */
export function display(t: TranslateFn, metric: Metric, value: number): string {
  if (metric.key === "cost" && value === 0) {
    return t("Local");
  }

  return metric.format(value);
}

/** A metric's value is a better one than another's. */
export function beats(metric: Metric, value: number, other: number): boolean {
  return metric.higherIsBetter ? value >= other : value <= other;
}

export type BoardEntry = { series: AIAnalyticsSeries; value: number; best: boolean };

/** A leaderboard: ranked series best first, then the ones with too few runs, unranked. */
export function board(series: readonly AIAnalyticsSeries[], metric: Metric): BoardEntry[] {
  const entries = series
    .map((item) => ({ series: item, value: metric.value(item) }))
    .filter(
      (entry): entry is { series: AIAnalyticsSeries; value: number } =>
        entry.value !== null && Number.isFinite(entry.value),
    )
    .sort(
      (left, right) =>
        Number(left.series.few) - Number(right.series.few) ||
        (metric.higherIsBetter ? right.value - left.value : left.value - right.value),
    );

  return entries.map((entry, index) => ({
    ...entry,
    best: index === 0 && !entry.series.few && entries.length > 1,
  }));
}

export const BOLD: RichTags = { b: (children) => createElement("b", null, children) };

/**
 * The sentence over a leaderboard card: the best series, and the worst ranked one beside it.
 * Each is one whole message whose `<b>` tags mark the series names, so a translator can move
 * the names along with the words.
 */
export function boardSentence(
  t: TranslateFn,
  rt: RichTranslateFn,
  metric: Metric,
  entries: readonly BoardEntry[],
): ReactNode {
  const ranked = entries.filter((entry) => !entry.series.few);
  const best = ranked[0];
  if (!best) {
    return null;
  }
  const worst = ranked.length > 1 ? ranked[ranked.length - 1] : undefined;
  const top = best.series.shortLabel;
  const low = worst?.series.shortLabel;
  const value = worst?.value ?? 0;

  switch (metric.key) {
    case "completed": {
      const own = Math.round(best.value * 100);
      return worst
        ? rt(
            "<b>{0}</b> finishes {1} of every 100 runs; <b>{2}</b> finishes {3}.",
            BOLD,
            top,
            own,
            low,
            Math.round(value * 100),
          )
        : rt("<b>{0}</b> finishes {1} of every 100 runs.", BOLD, top, own);
    }
    case "runTime": {
      const own = metric.format(best.value);
      return worst
        ? rt(
            "<b>{0}</b> usually finishes in {1}; <b>{2}</b> takes {3}.",
            BOLD,
            top,
            own,
            low,
            metric.format(value),
          )
        : rt("<b>{0}</b> usually finishes in {1}.", BOLD, top, own);
    }
    case "toolFailures": {
      if (best.value <= 0) {
        return worst && value > 0
          ? rt(
              "<b>{0}</b> gets no tool calls wrong; <b>{1}</b> gets {2} wrong.",
              BOLD,
              top,
              low,
              oneIn(t, value),
            )
          : rt("<b>{0}</b> gets no tool calls wrong.", BOLD, top);
      }
      const own = oneIn(t, best.value);
      return worst
        ? rt(
            "<b>{0}</b> gets {1} tool calls wrong; <b>{2}</b> gets {3} wrong.",
            BOLD,
            top,
            own,
            low,
            oneIn(t, value),
          )
        : rt("<b>{0}</b> gets {1} tool calls wrong.", BOLD, top, own);
    }
    case "cost": {
      const other = display(t, metric, value);
      if (isLocal(best.series)) {
        return worst
          ? rt(
              "<b>{0}</b> runs on your own hardware, so a run costs nothing; <b>{1}</b> costs {2}.",
              BOLD,
              top,
              low,
              other,
            )
          : rt("<b>{0}</b> runs on your own hardware, so a run costs nothing.", BOLD, top);
      }
      const own = metric.format(best.value);
      return worst
        ? rt(
            "<b>{0}</b> costs {1} per finished run; <b>{2}</b> costs {3}.",
            BOLD,
            top,
            own,
            low,
            other,
          )
        : rt("<b>{0}</b> costs {1} per finished run.", BOLD, top, own);
    }
    case "approval": {
      const own = percent(best.value);
      return worst
        ? rt(
            "People accept {1} of <b>{0}</b>'s proposals as written, and {3} of <b>{2}</b>'s.",
            BOLD,
            top,
            own,
            low,
            percent(value),
          )
        : rt("People accept {1} of <b>{0}</b>'s proposals as written.", BOLD, top, own);
    }
    case "firstToken": {
      const own = metric.format(best.value);
      return worst
        ? rt(
            "<b>{0}</b> starts answering in {1}; <b>{2}</b> in {3}.",
            BOLD,
            top,
            own,
            low,
            metric.format(value),
          )
        : rt("<b>{0}</b> starts answering in {1}.", BOLD, top, own);
    }
    case "helpful":
      return rt("<b>{0}</b> is rated helpful {1} of the time.", BOLD, top, percent(best.value));
  }
}

/** Round, readable steps from zero to just past the largest value. */
export function niceTicks(max: number, count = 5): { top: number; ticks: number[] } {
  const raw = Math.max(max, Number.EPSILON) / count;
  const magnitude = 10 ** Math.floor(Math.log10(raw));
  const step = [1, 2, 2.5, 5, 10].map((m) => m * magnitude).find((s) => s >= raw) ?? raw;
  const top = Math.ceil(max / step) * step;
  const ticks: number[] = [];
  for (let value = 0; value <= top + step / 1e6; value += step) {
    ticks.push(Number(value.toFixed(6)));
  }

  return { top, ticks };
}

/** Share ticks in fives from a floor just under the lowest value up to 100%. */
export function shareAxis(values: readonly number[]): { low: number; ticks: number[] } {
  const lowest = values.length ? Math.min(...values) : 0;
  const low = Math.max(0, Math.floor((lowest - 0.03) * 20) / 20);
  const ticks: number[] = [];
  for (let value = low; value <= 1 + 1e-6; value += 0.05) {
    ticks.push(Number(value.toFixed(4)));
  }

  return { low, ticks };
}

export function median(values: readonly number[]): number {
  if (values.length === 0) {
    return 0;
  }
  const sorted = [...values].sort((left, right) => left - right);

  return sorted[Math.floor(sorted.length / 2)] ?? 0;
}

export type LabelPlacement = {
  key: string;
  cx: number;
  cy: number;
  r: number;
  width: number;
  left: boolean;
  y: number;
};

type Box = { x0: number; x1: number; y0: number; y1: number };

const LABEL_GAP = 7;
const LABEL_HALF_HEIGHT = 8;

function labelBox(label: LabelPlacement): Box {
  const x0 = label.left
    ? label.cx - label.r - LABEL_GAP - label.width
    : label.cx + label.r + LABEL_GAP;

  return {
    x0,
    x1: x0 + label.width,
    y0: label.y - LABEL_HALF_HEIGHT,
    y1: label.y + LABEL_HALF_HEIGHT,
  };
}

function overlaps(left: Box, right: Box): boolean {
  return left.x0 < right.x1 && right.x0 < left.x1 && left.y0 < right.y1 && right.y0 < left.y1;
}

/**
 * Where each dot's label goes: to its right unless that leaves the plot, flipped to the
 * other side when it would cover a label or dot, and then nudged down until it is clear.
 */
export function placeLabels(
  dots: readonly { key: string; cx: number; cy: number; r: number; width: number }[],
  plotRight: number,
): LabelPlacement[] {
  const labels = dots
    .map((dot) => ({ ...dot, left: dot.cx + dot.r + LABEL_GAP + dot.width > plotRight, y: dot.cy }))
    .sort((left, right) => left.cy - right.cy);
  const dotBoxes = labels.map((dot) => ({
    x0: dot.cx - dot.r - 2,
    x1: dot.cx + dot.r + 2,
    y0: dot.cy - dot.r - 2,
    y1: dot.cy + dot.r + 2,
  }));
  const blocked = (label: LabelPlacement, index: number) => {
    const box = labelBox(label);
    return (
      labels.slice(0, index).some((other) => overlaps(box, labelBox(other))) ||
      dotBoxes.some((dot, other) => other !== index && overlaps(box, dot))
    );
  };
  labels.forEach((label, index) => {
    if (!blocked(label, index)) {
      return;
    }
    label.left = !label.left;
    if (!blocked(label, index)) {
      return;
    }
    label.left = !label.left;
    for (let attempt = 0; attempt < 10 && blocked(label, index); attempt += 1) {
      label.y += 9;
    }
  });

  return labels;
}

/** The run-time axis: log scale from one second to three minutes, ticked where people count. */
export const RUN_TIME_TICKS_S = [1, 2, 5, 10, 30, 60, 120] as const;
const RUN_TIME_CEILING_S = 180;

export function runTimePosition(ms: number): number {
  const seconds = Math.max(1, ms / 1000);

  return Math.max(0, Math.min(1, Math.log(seconds) / Math.log(RUN_TIME_CEILING_S)));
}

export function runTimeTick(seconds: number): string {
  return seconds >= 60 ? `${seconds / 60}m` : `${seconds}s`;
}

export function outcomeCount(series: AIAnalyticsSeries, outcome: AiWorkOutcome): number {
  switch (outcome) {
    case "Completed":
      return series.completed;
    case "HandedOff":
      return series.handedOff;
    case "Stopped":
      return series.stopped;
    case "Failed":
      return series.failed;
  }
}

export function runTimeOf(series: AIAnalyticsSeries, outcome: AiWorkOutcome) {
  return series.runTimes.find((runTime) => runTime.outcome === outcome) ?? null;
}

/** Rows ordered for an outcome: series that had such runs by median, the rest after. */
export function runTimeOrder(
  series: readonly AIAnalyticsSeries[],
  outcome: AiWorkOutcome,
): string[] {
  return [...series]
    .sort((left, right) => {
      const a = runTimeOf(left, outcome);
      const b = runTimeOf(right, outcome);
      return Number(Boolean(b)) - Number(Boolean(a)) || (a?.medianMs ?? 0) - (b?.medianMs ?? 0);
    })
    .map((item) => item.key);
}

/** How dark a failure cell is: none at zero, full at twelve percent and above. */
export const FAILURE_CEILING = 0.12;

export function failureIntensity(rate: number | null | undefined): number {
  if (rate === null || rate === undefined || rate <= 0) {
    return 0;
  }

  return Math.min(1, rate / FAILURE_CEILING);
}

/** How green a task cell is: none at 60% completed, full at 98%. */
export function completionIntensity(rate: number | null | undefined): number {
  if (rate === null || rate === undefined) {
    return 0;
  }

  return Math.max(0, Math.min(1, (rate - 0.6) / 0.38));
}

export type TrendMetric = "completed" | "toolFailures" | "runTime";

export const TREND_METRICS: readonly { value: TrendMetric; label: string; metric: MetricKey }[] = [
  { value: "completed", label: "Completed", metric: "completed" },
  { value: "toolFailures", label: "Tool calls failed", metric: "toolFailures" },
  { value: "runTime", label: "Median run time", metric: "runTime" },
];

function trendValue(point: AIAnalyticsTrendPoint, metric: TrendMetric): number | null {
  switch (metric) {
    case "completed":
      return point.completedRate ?? null;
    case "toolFailures":
      return point.toolFailureRate ?? null;
    case "runTime":
      return point.medianRunMs ?? null;
  }
}

/** Each series' value per bucket, null where it did not run; points land on their own bucket. */
export function trendLines(
  trend: readonly AIAnalyticsTrendPoint[],
  metric: TrendMetric,
  keys: readonly string[],
  buckets: readonly number[],
): Map<string, (number | null)[]> {
  const out = new Map<string, (number | null)[]>(keys.map((key) => [key, buckets.map(() => null)]));
  const slots = new Map(buckets.map((start, index) => [start, index] as const));
  for (const point of trend) {
    const line = out.get(point.seriesKey);
    const slot = slots.get(point.bucketStart);
    const value = trendValue(point, metric);
    if (!line || slot === undefined || value === null || !Number.isFinite(value)) {
      continue;
    }
    line[slot] = value;
  }

  return out;
}

/**
 * Where an instant falls on the bucket axis, as a fractional bucket index: 2.5 is halfway
 * through the third bucket. Buckets need not be the same length (a day with a clock change).
 */
export function bucketPosition(
  buckets: readonly number[],
  at: number,
  bucketSeconds: number,
): number {
  const count = buckets.length;
  if (count === 0) {
    return 0;
  }
  let index = 0;
  while (index + 1 < count && (buckets[index + 1] ?? 0) <= at) {
    index += 1;
  }
  const start = buckets[index] ?? 0;
  const end = buckets[index + 1] ?? start + bucketSeconds;

  return index + Math.min(1, Math.max(0, (at - start) / (end - start || bucketSeconds)));
}

/**
 * The row each label goes on so none overlaps another: a label takes the highest row whose
 * last label ends, with a gap, before it starts, and opens a new row when none has room.
 */
export function stackLabels(
  spans: readonly { start: number; end: number }[],
  gap: number,
): number[] {
  const rowEnds: number[] = [];

  return spans.map((span) => {
    const row = rowEnds.findIndex((end) => end + gap <= span.start);
    if (row >= 0) {
      rowEnds[row] = span.end;
      return row;
    }
    rowEnds.push(span.end);
    return rowEnds.length - 1;
  });
}

/** An SVG path through the points, lifting the pen over gaps. */
export function linePath(
  values: readonly (number | null)[],
  x: (index: number) => number,
  y: (value: number) => number,
): string {
  let path = "";
  let pen = false;
  values.forEach((value, index) => {
    if (value === null) {
      pen = false;
      return;
    }
    path += `${pen ? "L" : "M"}${x(index).toFixed(1)},${y(value).toFixed(1)}`;
    pen = true;
  });

  return path;
}

export type WhoseRow = {
  series: AIAnalyticsSeries;
  model: { cause: AiToolFailureCause; share: number }[];
  tool: { cause: AiToolFailureCause; share: number }[];
  modelTotal: number;
  toolTotal: number;
};

/** Each series' failed tool calls as shares of all its calls, split by whose mistake. */
export function whoseRows(series: readonly AIAnalyticsSeries[]): WhoseRow[] {
  return series
    .map((item) => {
      const calls = Math.max(1, item.toolCalls);
      const share = (cause: AiToolFailureCause) =>
        (item.toolFailuresByCause.find((count) => count.cause === cause)?.calls ?? 0) / calls;
      const model = CAUSES.filter((entry) => entry.blame === "Model").map((entry) => ({
        cause: entry.cause,
        share: share(entry.cause),
      }));
      const tool = CAUSES.filter((entry) => entry.blame === "Tool").map((entry) => ({
        cause: entry.cause,
        share: share(entry.cause),
      }));
      const sum = (parts: { share: number }[]) =>
        parts.reduce((total, part) => total + part.share, 0);
      return { series: item, model, tool, modelTotal: sum(model), toolTotal: sum(tool) };
    })
    .sort((left, right) => right.modelTotal - left.modelTotal);
}

/** The deployment a series is drawn under: its model, or its provider. */
export function groupKeyOf(compareBy: AiAnalyticsCompareBy, deployment: AIAnalyticsSeries): string {
  return compareBy === "Provider" ? (deployment.providerId ?? "") : (deployment.modelKey ?? "");
}

export type SameModelRow = {
  metric: Metric;
  left: number;
  right: number;
  max: number;
  leftWins: boolean;
};

export type SameModelPair = {
  left: AIAnalyticsSeries;
  right: AIAnalyticsSeries;
  rows: SameModelRow[];
  wins: number;
};

/**
 * The model the server picked as running on two providers (the widest gap in completion), its
 * better deployment on the left, and how the two compare on the figures the connection moves.
 * Nothing when either deployment is hidden from the charts.
 */
export function sameModelPair(
  deployments: readonly AIAnalyticsSeries[],
  sameModel: AgentAnalytics["sameModel"],
  isHidden: (deployment: AIAnalyticsSeries) => boolean,
): SameModelPair | null {
  if (!sameModel) {
    return null;
  }
  const left = deployments.find((deployment) => deployment.key === sameModel.betterKey);
  const right = deployments.find((deployment) => deployment.key === sameModel.worseKey);
  if (!left || !right || isHidden(left) || isHidden(right)) {
    return null;
  }
  const rows: SameModelRow[] = [];
  for (const key of SAME_MODEL_METRICS) {
    const metric = METRICS[key];
    const a = metric.value(left);
    const b = metric.value(right);
    if (a === null || b === null) {
      continue;
    }
    rows.push({
      metric,
      left: a,
      right: b,
      max: Math.max(a, b) || 1,
      leftWins: beats(metric, a, b),
    });
  }

  return { left, right, rows, wins: rows.filter((row) => row.leftWins).length };
}

/** The hidden series that are still on the page; all of them hidden shows them all again. */
export function hiddenAmong(
  hidden: readonly string[],
  series: readonly { key: string }[],
): readonly string[] {
  const keys = new Set(series.map((item) => item.key));
  const kept = hidden.filter((key) => keys.has(key));

  return series.length > 0 && kept.length >= series.length ? [] : kept;
}

/**
 * The task the scope filters on: none when no task is chosen or the chosen one is not a task
 * the catalog knows, and undefined while the catalog that decides it is still loading.
 */
export function knownTask(
  task: string | null,
  labels: ReadonlyMap<AiTask, string>,
  loaded: boolean,
  failed: boolean,
): AiTask | null | undefined {
  if (!task) {
    return null;
  }
  if (labels.has(task as AiTask)) {
    return task as AiTask;
  }

  return loaded || failed ? null : undefined;
}

/** Why a scope with runs elsewhere shows nothing, naming the filters that are set. */
export function emptyScopeMessage(
  t: TranslateFn,
  scope: { window: string; agent: boolean; task: boolean },
): string {
  if (scope.agent && scope.task) {
    return t("Nothing ran for this agent and task in {0}.", scope.window);
  }
  if (scope.agent) {
    return t("Nothing ran for this agent in {0}.", scope.window);
  }
  if (scope.task) {
    return t("Nothing ran for this task in {0}.", scope.window);
  }

  return t("Nothing ran in {0}.", scope.window);
}

/** The routing order with one provider moved to just ahead of another. */
export function moveAhead(ids: readonly string[], mover: string, ahead: string): string[] | null {
  const from = ids.indexOf(mover);
  const target = ids.indexOf(ahead);
  if (from < 0 || target < 0 || from < target) {
    return null;
  }
  const next = ids.filter((id) => id !== mover);
  next.splice(next.indexOf(ahead), 0, mover);

  return next;
}

const EXPORT_HEADERS: readonly { label: string }[] = [
  { label: "Group" },
  { label: "Providers" },
  { label: "Runs" },
  { label: "Completed %" },
  { label: "Handed off" },
  { label: "Stopped" },
  { label: "Failed" },
  { label: "Median run (ms)" },
  { label: "Slowest 5% (ms)" },
  { label: "Tool calls" },
  { label: "Tool calls failed %" },
  { label: "Model's mistakes" },
  { label: "Spend (USD)" },
  { label: "Per completed run (USD)" },
  { label: "Approved as proposed %" },
  { label: "Rated helpful %" },
  { label: "First token (ms)" },
];

function csvCell(value: string | number): string {
  return escapeCsvValue(
    typeof value === "number" ? String(value) : (spreadsheetSafeText(value) ?? ""),
  );
}

function rateCell(value: number | null | undefined): string {
  return value === null || value === undefined ? "" : (value * 100).toFixed(2);
}

/** The scope as a spreadsheet: one row per group, then one per deployment. */
export function analyticsCsv(analytics: AgentAnalytics, t: TranslateFn): string {
  const row = (series: AIAnalyticsSeries, label: string) => [
    label,
    series.providers.map((provider) => provider.name).join("; "),
    series.runs,
    rateCell(series.completedRate),
    series.handedOff,
    series.stopped,
    series.failed,
    series.medianRunMs ?? "",
    series.p95RunMs ?? "",
    series.toolCalls,
    rateCell(series.toolFailureRate),
    series.toolFailuresModel,
    series.costUsd,
    series.costPerCompletedRun ?? "",
    rateCell(series.approvalRate),
    rateCell(series.helpfulRate),
    series.firstTokenMedianMs ?? "",
  ];
  const rows = [
    ...analytics.series.map((series) => row(series, series.label)),
    ...analytics.deployments.map((series) =>
      row(series, `${series.models[0] ?? ""} · ${series.label}`),
    ),
  ];

  return [EXPORT_HEADERS.map((header) => t(header.label)), ...rows]
    .map((cells) => cells.map(csvCell).join(","))
    .join("\n");
}

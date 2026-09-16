import type { Insight, InsightMetric } from "@/types/insight";

/**
 * One point in a finding's history: what a metric read at one refresh.
 */
export type TrendPoint = {
  detectedAt: number;
  value: number;
};

/**
 * Which way a finding has moved since it was first seen.
 *
 * "Unknown" is a real answer and is kept distinct from "steady". A finding seen
 * once has no direction, and drawing it as flat would assert stability nobody
 * has observed.
 */
export type TrendDirection = "worsening" | "improving" | "steady" | "unknown";

export type MetricTrend = {
  metricKey: string;
  label: string;
  points: TrendPoint[];
  direction: TrendDirection;
  /** The change from the oldest reading to the newest, in the metric's own unit. */
  change: number | null;
  /** The metric as it reads now, carried so a caller can format the series. */
  current: InsightMetric;
};

/**
 * Builds the series for one metric across a finding's history.
 *
 * History arrives newest-first because that is how it is read back; a chart is
 * drawn oldest-first, so the order is reversed here rather than in the
 * component, where it would be one more thing to get wrong.
 *
 * A refresh that did not carry this metric is skipped rather than plotted as
 * zero. Detectors add and drop metrics between releases, and a zero would draw a
 * collapse that never happened.
 */
export function buildTrend(
  current: Insight,
  history: readonly Insight[],
  metricKey: string,
): MetricTrend | null {
  const metric = (current.metrics ?? []).find((entry) => entry.key === metricKey);
  if (!metric) {
    return null;
  }

  const points: TrendPoint[] = [];

  for (const past of [...history].reverse()) {
    const value = readMetric(past, metricKey);
    if (value !== null) {
      points.push({ detectedAt: past.detectedAt, value });
    }
  }

  const currentValue = parseDecimal(metric.value);
  if (currentValue !== null) {
    points.push({ detectedAt: current.detectedAt, value: currentValue });
  }

  return {
    metricKey,
    label: metric.label,
    points,
    direction: directionOf(points, metric),
    change: changeOf(points),
    current: metric,
  };
}

/**
 * Reads the direction a metric has travelled, in terms of whether that is good.
 *
 * The sign alone does not say: rising detention and rising on-time percentage
 * are opposite news. The metric carries which way is bad, and that is what
 * decides the word.
 */
function directionOf(points: readonly TrendPoint[], metric: InsightMetric): TrendDirection {
  if (points.length < 2) {
    return "unknown";
  }

  const first = points[0];
  const last = points[points.length - 1];
  if (!first || !last) {
    return "unknown";
  }

  const delta = last.value - first.value;
  if (delta === 0 || metric.direction === "Neutral") {
    return "steady";
  }

  const worse = metric.direction === "HigherIsWorse" ? delta > 0 : delta < 0;

  return worse ? "worsening" : "improving";
}

function changeOf(points: readonly TrendPoint[]): number | null {
  if (points.length < 2) {
    return null;
  }

  const first = points[0];
  const last = points[points.length - 1];
  if (!first || !last) {
    return null;
  }

  return last.value - first.value;
}

/**
 * Normalises a series to 0–1 for drawing, and reports whether it is flat.
 *
 * A flat series has no range to normalise against. Dividing by zero would put
 * every point at NaN and draw nothing; returning the midline draws the truth,
 * which is a level line.
 */
export function normalizePoints(points: readonly TrendPoint[]): number[] {
  if (points.length === 0) {
    return [];
  }

  const values = points.map((point) => point.value);
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min;

  if (range === 0) {
    return values.map(() => 0.5);
  }

  return values.map((value) => (value - min) / range);
}

/**
 * Which metric a finding's trend should be drawn from.
 *
 * The first metric is the detector's own headline number — the percentage, the
 * amount, the count it led with — so it is the one worth charting. Falling back
 * to nothing rather than guessing keeps a malformed finding from drawing a chart
 * of something arbitrary.
 */
export function primaryTrendMetricKey(insight: Insight): string | null {
  const first = (insight.metrics ?? [])[0];

  return first ? first.key : null;
}

function readMetric(insight: Insight, metricKey: string): number | null {
  const metric = (insight.metrics ?? []).find((entry) => entry.key === metricKey);

  return metric ? parseDecimal(metric.value) : null;
}

function parseDecimal(raw: string | null | undefined): number | null {
  if (raw === null || raw === undefined || raw.trim() === "") {
    return null;
  }

  const value = Number(raw);

  return Number.isFinite(value) ? value : null;
}

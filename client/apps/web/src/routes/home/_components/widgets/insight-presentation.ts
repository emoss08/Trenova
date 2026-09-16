import type { Insight, InsightMetric, InsightSeverity } from "@/types/insight";

/**
 * How a metric's value should read.
 *
 * Values arrive as decimal strings because a currency amount that has been
 * through a JS float is no longer the amount. Formatting parses once, at the
 * edge, and anything unparseable renders as an em dash rather than NaN — a card
 * that says "NaN unbilled" is worse than one that admits it does not know.
 */
export function formatMetricValue(metric: InsightMetric): string {
  const value = parseDecimal(metric.value);
  if (value === null) {
    return "—";
  }

  switch (metric.unit) {
    case "Currency":
      return value.toLocaleString(undefined, {
        style: "currency",
        currency: "USD",
        maximumFractionDigits: 0,
      });
    case "Percent":
      return `${trimNumber(value)}%`;
    case "Days":
      return `${trimNumber(value)}d`;
    case "Hours":
      return `${trimNumber(value)}h`;
    case "Miles":
      return `${Math.round(value).toLocaleString()} mi`;
    default:
      return Math.round(value).toLocaleString();
  }
}

/**
 * The change between a metric and its baseline, as the reader should see it.
 *
 * Returns null when there is nothing to compare against, so a caller renders
 * nothing rather than an arrow pointing at zero.
 */
export function formatMetricChange(metric: InsightMetric): string | null {
  if (!metric.baseline) {
    return null;
  }

  const value = parseDecimal(metric.value);
  const baseline = parseDecimal(metric.baseline);
  if (value === null || baseline === null) {
    return null;
  }

  const delta = value - baseline;
  if (delta === 0) {
    return null;
  }

  const sign = delta > 0 ? "+" : "−";
  const magnitude = formatMetricValue({ ...metric, value: String(Math.abs(delta)) });

  return `${sign}${magnitude}`;
}

/** Whether a change is movement in the wrong direction, for colouring. */
export function isChangeAdverse(metric: InsightMetric): boolean {
  if (!metric.baseline || metric.direction === "Neutral") {
    return false;
  }

  const value = parseDecimal(metric.value);
  const baseline = parseDecimal(metric.baseline);
  if (value === null || baseline === null) {
    return false;
  }

  return metric.direction === "HigherIsWorse" ? value > baseline : value < baseline;
}

/**
 * The prose to show, and whether it came from a model.
 *
 * A card headed "insight" invites more trust than a table, so the reader is
 * told which sentences a model wrote. The numbers beside them never came from
 * one, which is exactly why the distinction is worth drawing rather than hiding.
 */
export function insightBody(insight: Insight): { text: string; generated: boolean } {
  const narrative = insight.narrative.trim();
  if (narrative !== "" && insight.narrated) {
    return { text: narrative, generated: true };
  }

  return { text: narrative, generated: false };
}

const SEVERITY_RANK: Record<InsightSeverity, number> = {
  Critical: 3,
  Warning: 2,
  Info: 1,
};

/**
 * Orders findings the way a person should read them: most urgent first, and
 * newest first within a severity.
 *
 * The server already sorts this way. Doing it again here means a client that
 * merges a refreshed list, or renders an optimistic dismissal, cannot end up
 * showing a critical finding below an informational one.
 */
export function sortInsights(insights: readonly Insight[]): Insight[] {
  return [...insights].sort((left, right) => {
    const bySeverity = SEVERITY_RANK[right.severity] - SEVERITY_RANK[left.severity];
    if (bySeverity !== 0) {
      return bySeverity;
    }

    return right.detectedAt - left.detectedAt;
  });
}

/**
 * Whether the numbers have aged past the point of being worth trusting.
 *
 * A stale insight is still shown, labelled, rather than hidden: the condition it
 * describes was real, and a refresh that has not run is a different problem from
 * a problem that has gone away. Hiding it would quietly report "all clear".
 */
export function isStale(insight: Insight, now: number): boolean {
  return insight.staleAt > 0 && now > insight.staleAt;
}

/**
 * The metrics a card shows without being opened.
 *
 * Two is the limit because a home tile is scanned, not studied, and the third
 * number is where a card stops being a headline and becomes a report.
 */
export function primaryMetrics(insight: Insight): InsightMetric[] {
  return (insight.metrics ?? []).slice(0, 2);
}

/**
 * Reads a decimal string, treating an empty or unreadable one as absent.
 *
 * Number("") is 0, which is finite, so relying on Number alone turns a missing
 * figure into a confident zero — "0% on time", "$0 unbilled". Those read as
 * findings rather than as gaps, which is the opposite of what a blank value
 * means.
 */
function parseDecimal(raw: string | null | undefined): number | null {
  if (raw === null || raw === undefined || raw.trim() === "") {
    return null;
  }

  const value = Number(raw);

  return Number.isFinite(value) ? value : null;
}

function trimNumber(value: number): string {
  const rounded = Math.round(value * 10) / 10;

  return Number.isInteger(rounded) ? String(rounded) : rounded.toFixed(1);
}

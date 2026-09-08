import { csaBasicTone, type CSATone } from "@trenova/shared/lib/csa";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";

// Structural shapes rather than the generated types, so the same maths serves
// the page and a fixture written from the schema by hand.
export type RatingCountLike = { rating: string; workers: number };
export type BasicLike = {
  basic: string;
  violations: number;
  events: number;
  weightedScore: number;
  outOfService: number;
  inferred: boolean;
};
export type KindLike = {
  kind: string;
  events: number;
  points: number;
  preventable: number;
  outOfService: number;
  open: number;
};
export type TrendPointLike = {
  periodStart: number;
  events: number;
  accidents: number;
  preventable: number;
  citations: number;
  inspections: number;
  outOfService: number;
  points: number;
};
export type TerminalLike = {
  fleetCodeId?: string | null;
  code: string;
  workers: number;
  atRisk: number;
  watch: number;
  averageScore: number;
};

/** The safety ratings in the order the scale runs, best first. */
export const SAFETY_RATING_ORDER = ["Excellent", "Good", "Watch", "AtRisk"] as const;

export type RatingSegment = { rating: string; label: string; workers: number };

const RATING_LABELS: Record<string, string> = {
  Excellent: "Excellent",
  Good: "Good",
  Watch: "Watch",
  AtRisk: "At risk",
};

/**
 * The roster split by rating, always all four in scale order. A rating with
 * nobody in it stays on the legend at zero: a bar that drops "At risk" when it
 * empties reads as a category that does not exist.
 */
export function ratingSegments(ratings: readonly RatingCountLike[]): RatingSegment[] {
  const counts = new Map(ratings.map((row) => [row.rating, row.workers]));
  return SAFETY_RATING_ORDER.map((rating) => ({
    rating,
    label: RATING_LABELS[rating] ?? rating,
    workers: counts.get(rating) ?? 0,
  }));
}

export type BasicStanding<B extends BasicLike = BasicLike> = {
  basic: B;
  tone: CSATone;
  /** Width against the fleet's own worst category, 0 to 100. */
  share: number;
};

/**
 * Each BASIC read against the fleet's own worst one. The share is relative
 * and deliberately not called a percentile: the FMCSA ranks a carrier against
 * its peers, and this data cannot.
 */
export function basicStandings<B extends BasicLike>(basics: readonly B[]): BasicStanding<B>[] {
  const worst = basics.reduce((peak, row) => Math.max(peak, row.weightedScore), 0);
  return basics.map((basic) => ({
    basic,
    tone: csaBasicTone(basic.weightedScore, worst),
    share:
      worst > 0 && basic.weightedScore > 0
        ? Math.max(3, Math.min(100, Math.round((basic.weightedScore / worst) * 100)))
        : 0,
  }));
}

export type EventTotals = {
  events: number;
  preventable: number;
  open: number;
  outOfService: number;
  points: number;
};

export function eventTotals(kinds: readonly KindLike[]): EventTotals {
  const totals: EventTotals = { events: 0, preventable: 0, open: 0, outOfService: 0, points: 0 };
  for (const kind of kinds) {
    totals.events += kind.events;
    totals.preventable += kind.preventable;
    totals.open += kind.open;
    totals.outOfService += kind.outOfService;
    totals.points += kind.points;
  }
  return totals;
}

export type TrendRow = {
  periodStart: number;
  /** Short month label for the axis, in UTC because the buckets are. */
  label: string;
  events: number;
  accidents: number;
  preventable: number;
  citations: number;
  inspections: number;
  outOfService: number;
  points: number;
};

/** The trend as chart rows, oldest first, each month labelled once. */
export function trendRows(trend: readonly TrendPointLike[]): TrendRow[] {
  return [...trend]
    .sort((a, b) => a.periodStart - b.periodStart)
    .map((point) => ({
      ...point,
      label: formatUnixInUserTimezone(point.periodStart, { month: "short", timezone: "UTC" }),
    }));
}

/**
 * The month with the most events, and how the window's second half compares
 * with its first. Two points are not a trend, so short windows read as flat.
 */
export function trendPeak(trend: readonly TrendPointLike[]): TrendPointLike | null {
  return trend.reduce<TrendPointLike | null>(
    (peak, point) => (peak === null || point.events > peak.events ? point : peak),
    null,
  );
}

export type TerminalStanding<T extends TerminalLike = TerminalLike> = {
  terminal: T;
  /** Drivers flagged at risk or on watch, out of the terminal's drivers. */
  flagged: number;
  share: number;
};

/** Terminals with the most flagged drivers first, each with its share of the fleet. */
export function terminalStandings<T extends TerminalLike>(
  terminals: readonly T[],
  totalWorkers: number,
): TerminalStanding<T>[] {
  return terminals
    .map((terminal) => ({
      terminal,
      flagged: terminal.atRisk + terminal.watch,
      share:
        totalWorkers > 0 && terminal.workers > 0
          ? Math.max(2, Math.min(100, Math.round((terminal.workers / totalWorkers) * 100)))
          : 0,
    }))
    .sort(
      (a, b) =>
        b.terminal.atRisk - a.terminal.atRisk ||
        b.flagged - a.flagged ||
        b.terminal.workers - a.terminal.workers,
    );
}

export const WINDOW_MONTHS = ["3", "6", "12", "24"] as const;
export type WindowMonths = (typeof WINDOW_MONTHS)[number];

export function isWindowMonths(value: string): value is WindowMonths {
  return (WINDOW_MONTHS as readonly string[]).includes(value);
}

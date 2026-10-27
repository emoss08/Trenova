import { formatAnalysisValue } from "@/lib/analysis-values";
import type { AnalysisChart } from "@/types/analysis";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/** The key each point carries its place along the axis under, formatted for a person. */
export const CHART_LABEL_KEY = "__label";

/** The key the numeric x of a scatter point is carried under. */
export const CHART_X_KEY = "__x";

export type AnalysisChartPoint = Record<string, string | number | null>;

/** A series is keyed by its place, so two series a script named alike stay apart. */
export function chartSeriesKey(index: number): string {
  return `s${index}`;
}

/**
 * A chart's spec as Recharts reads it: one point per value along the axis, its label formatted
 * by the axis's kind and each series' number under the series' key. A value a series lacks is
 * null, a gap rather than a zero. A scatter point also carries its x as a number.
 */
export function analysisChartPoints(chart: AnalysisChart, t: TranslateFn): AnalysisChartPoint[] {
  return chart.x.values.map((value, index) => {
    const point: AnalysisChartPoint = {
      [CHART_LABEL_KEY]: formatAnalysisValue(value, chart.x.kind, t),
    };
    if (chart.kind === "scatter") {
      point[CHART_X_KEY] = typeof value === "number" && Number.isFinite(value) ? value : null;
    }
    chart.series.forEach((series, seriesIndex) => {
      point[chartSeriesKey(seriesIndex)] = series.values[index] ?? null;
    });
    return point;
  });
}

/**
 * The slices of a pie: its one series' positive values by their label. A pie cannot draw a
 * negative or missing slice, so those are left out and counted for the reader.
 */
export function analysisPieSlices(points: AnalysisChartPoint[]): {
  slices: AnalysisChartPoint[];
  left: number;
} {
  const key = chartSeriesKey(0);
  const slices = points.filter((point) => {
    const value = point[key];
    return typeof value === "number" && value > 0;
  });

  return { slices, left: points.length - slices.length };
}

/** Whether a chart has anything to draw: a series with at least one number. */
export function chartHasValues(chart: AnalysisChart): boolean {
  return chart.series.some((series) => series.values.some((value) => value !== null));
}

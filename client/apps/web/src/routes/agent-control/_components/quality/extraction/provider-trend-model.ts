import type { ExtractionProviderTrend } from "@/lib/graphql/extraction-eval";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";

export type TrendState = "drifting" | "steady" | "gathering";

export const TREND_STATE: Record<TrendState, BadgeAttrProps> = {
  drifting: {
    phase: "attention",
    text: "Drifting",
    description: "Last week fell further below its own recent weeks than allowed",
  },
  steady: {
    phase: "complete",
    text: "Steady",
    description: "Last week held up against its own recent weeks",
  },
  gathering: {
    phase: "draft",
    text: "Not enough data",
    description: "Too few confirmed fields last week or in the weeks before to judge",
  },
};

export function trendState(trend: ExtractionProviderTrend): TrendState {
  if (!trend.comparable) return "gathering";
  return trend.drifting ? "drifting" : "steady";
}

/** Weekly accuracy for the sparkline, leaving out weeks with nothing scored. */
export function trendPoints(trend: ExtractionProviderTrend): number[] {
  return trend.weeks.filter((week) => week.scored > 0).map((week) => week.accuracy);
}

/** Last week against the baseline, as the fraction `formatDelta` expects; null until comparable. */
export function trendChange(trend: ExtractionProviderTrend): number | null {
  return trend.comparable ? -trend.dropPoints / 100 : null;
}

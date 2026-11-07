import type { DispatchForecastQuality } from "@/lib/graphql/dispatch-console";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatNumber } from "@trenova/shared/i18n/format";

/**
 * Where the freight forecast stands: never fitted, or fitted with the status its last test
 * earned. Plans use it only when it is Ready.
 */
export type ForecastStanding = "Unfitted" | DispatchForecastQuality["status"];

export function forecastStanding(quality: DispatchForecastQuality): ForecastStanding {
  return quality.fittedAt ? quality.status : "Unfitted";
}

export type ForecastStandingTone = "neutral" | "warning" | "success";

export const FORECAST_STANDING_TONE: Record<ForecastStanding, ForecastStandingTone> = {
  Unfitted: "neutral",
  InsufficientHistory: "neutral",
  NoSkill: "warning",
  Ready: "success",
};

/** A share from 0 to 1 as a whole percentage in the reader's language: "82%". */
export function formatShare(share: number): string {
  return formatNumber(share, { style: "percent", maximumFractionDigits: 0 });
}

/** Loads to one decimal, as an average miss or a shortfall reads: "1.4". */
export function formatLoads(loads: number): string {
  return formatNumber(loads, { minimumFractionDigits: 1, maximumFractionDigits: 1 });
}

/**
 * The forecast's standing in a heading and a sentence, the same on Dispatch controls and in
 * the console's outlook, so the two never describe one forecast differently.
 */
export function forecastStandingCopy(
  quality: DispatchForecastQuality,
  t: TranslateFn,
): { title: string; description: string } {
  switch (forecastStanding(quality)) {
    case "Unfitted":
      return {
        title: t("Not fitted yet"),
        description: t("The forecast is fitted every night from your shipment history."),
      };
    case "InsufficientHistory":
      return {
        title: t("Not enough history yet"),
        description: t(
          "The forecast needs 8 weeks and 200 loads of history; it has {0, plural, one {# week} other {# weeks}} and {1, plural, one {# load} other {# loads}}.",
          quality.historyWeeks,
          quality.historyLoads,
        ),
      };
    case "NoSkill":
      return {
        title: t("Not better than last week yet"),
        description: t(
          "Tested on your last {0, plural, one {week} other {# weeks}}, it missed by {1} loads per area and slot, against {2} for repeating last week. Plans ignore it until it does better; it is tested again every night.",
          quality.weeksTested,
          formatLoads(quality.meanAbsoluteError),
          formatLoads(quality.naiveMeanAbsoluteError),
        ),
      };
    case "Ready":
      return {
        title: t("Ready"),
        description: t(
          "Tested on your last {0, plural, one {week} other {# weeks}}: {1} of actual counts fell inside the forecast range; {2} more accurate than repeating last week.",
          quality.weeksTested,
          formatShare(quality.intervalCoverage),
          formatShare(quality.skill),
        ),
      };
  }
}

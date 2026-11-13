import type { DispatchForecastQuality } from "@/lib/graphql/dispatch-console";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatNumber } from "@trenova/shared/i18n/format";

/**
 * Where the freight forecast stands: never fitted, or fitted with the status its last test
 * earned. Plans use it only when it is Ready.
 */
export type ForecastStanding = "Unfitted" | DispatchForecastQuality["status"];

/** What the forecast needs before it is tested; the server's MinHistoryWeeks and MinHistoryLoads. */
export const FORECAST_MIN_WEEKS = 8;
export const FORECAST_MIN_LOADS = 200;

/** How often a page asks again while the history is being re-read, and for how long. */
export const FORECAST_REFIT_POLL_MS = 4_000;
export const FORECAST_REFIT_POLL_FOR_MS = 120_000;

/**
 * The refetch interval for a read that carries the forecast's standing: quick while the server
 * says it is refitting, so the page turns over when the fit lands, then back to the page's own
 * pace. The quick pace is held for two minutes at most, so a fit that keeps failing is not
 * asked for every few seconds for as long as the page is open.
 */
export function createRefitPoll(idleMs: number | false, now: () => number = Date.now) {
  let since: number | null = null;

  return (refitting: boolean | undefined): number | false => {
    if (!refitting) {
      since = null;
      return idleMs;
    }
    since ??= now();
    return now() - since < FORECAST_REFIT_POLL_FOR_MS ? FORECAST_REFIT_POLL_MS : idleMs;
  };
}

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
        description: quality.refitting
          ? t("Your shipment history is being read for the first time. This takes about a minute.")
          : t("The forecast is fitted every night from your shipment history."),
      };
    case "InsufficientHistory":
      return {
        title: t("Not enough history yet"),
        description: t(
          "The forecast is tested once the last 35 weeks hold {0, plural, one {# week} other {# weeks}} with loads and {1, plural, one {# load} other {# loads}} picked up.",
          FORECAST_MIN_WEEKS,
          FORECAST_MIN_LOADS,
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
          "Tested on your last {0, plural, one {week} other {# weeks}}: {1} of days landed inside the forecast range; {2} more accurate than repeating last week.",
          quality.weeksTested,
          formatShare(quality.intervalCoverage),
          formatShare(quality.skill),
        ),
      };
  }
}

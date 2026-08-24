import type { MarginVerdict, ShopOption, ShopResult, ShopStrategy } from "../types/rate";

/**
 * Presenting a shopping result.
 *
 * The ranking itself is the server's, and this deliberately does not re-do it:
 * a screen that sorted the options its own way would show a different winner
 * from the one the stored result names, and the stored result is what somebody
 * cites when asked why a carrier was offered the load.
 *
 * What lives here is the reading of it — which number the chosen strategy is
 * actually deciding on, how a margin should be coloured, and what to say about
 * an option that could not be priced.
 */

/** How a strategy is described where somebody picks one. */
const STRATEGY_LABEL: Record<ShopStrategy, string> = {
  LeastCost: "Cheapest",
  BestMargin: "Best margin",
  GuideRank: "Routing guide order",
  FastestAccept: "Fastest to accept",
};

const STRATEGY_EXPLANATION: Record<ShopStrategy, string> = {
  LeastCost: "Ranked by what the carrier charges.",
  BestMargin: "Ranked by what is left after paying them, which is not the same order as cheapest.",
  GuideRank: "Kept in the routing guide's own order, so a committed carrier is offered it first.",
  FastestAccept: "Ranked by the shortest offer window, for a load that has to move now.",
};

export function shopStrategyLabel(strategy: ShopStrategy): string {
  return STRATEGY_LABEL[strategy] ?? strategy;
}

export function shopStrategyExplanation(strategy: ShopStrategy): string {
  return STRATEGY_EXPLANATION[strategy] ?? "";
}

/**
 * How a margin should read.
 *
 * A breach is the one that has to stand out, because it is the only state where
 * somebody has a decision to make. A load with no sell price yet is neutral
 * rather than alarming: quoting cost before quoting the customer is ordinary.
 */
export type MarginTone = "breach" | "healthy" | "thin" | "unknown";

/**
 * Below this share of revenue a margin is worth looking at even where no
 * contract set a floor. It is a display threshold only — nothing is blocked by
 * it, and the contract's own floor is the number that decides anything.
 */
export const THIN_MARGIN_PERCENT = 10;

export function marginTone(margin: MarginVerdict, hasSellPrice: boolean): MarginTone {
  if (!hasSellPrice) return "unknown";
  if (margin.belowFloor || margin.abovePayCeiling) return "breach";
  if (margin.percent < THIN_MARGIN_PERCENT) return "thin";

  return "healthy";
}

/**
 * What to say about one option, in the words the panel shows.
 *
 * The server's own note wins whenever it set one: it knows why the option
 * ranked where it did, and paraphrasing it here would put two different
 * explanations of the same decision in front of the same person.
 */
export function shopOptionNote(option: ShopOption, hasSellPrice: boolean): string {
  if (option.note) return option.note;

  if (!optionIsPriced(option)) {
    return "No carrier agreement covers this lane.";
  }

  if (!hasSellPrice) {
    return "This shipment has no sell price yet, so there is no margin to measure.";
  }

  return "";
}

/**
 * Whether a contract actually put a number on this carrier.
 *
 * Mirrors `services.ShopOption.Priced`. An unpriced option is shown rather than
 * hidden — somebody needs to see that the carrier was considered — but it can
 * never be chosen.
 */
export function optionIsPriced(option: ShopOption): boolean {
  return (
    option.outcome === "Rated" ||
    option.outcome === "FormulaFallback" ||
    option.outcome === "ManualOverride"
  );
}

/** The options somebody could actually assign the load to. */
export function assignableOptions(result: ShopResult | undefined): ShopOption[] {
  return (result?.options ?? []).filter(optionIsPriced);
}

/**
 * The number the chosen strategy is deciding on, so the column it ranked by can
 * be marked as such.
 */
export function decidingValue(option: ShopOption, strategy: ShopStrategy): string {
  switch (strategy) {
    case "BestMargin":
      return `${option.margin.percent}%`;
    case "GuideRank":
      return option.guideRank > 0 ? `#${option.guideRank}` : "—";
    case "FastestAccept":
      return offerWindowLabel(option.offerTtlSeconds);
    case "LeastCost":
      return String(option.cost);
    default:
      return String(option.cost);
  }
}

/**
 * How long a carrier has to accept, in the units somebody thinks in.
 *
 * A guide that never set one reads as unknown rather than as zero minutes,
 * which would look like an offer that has already expired.
 */
export function offerWindowLabel(seconds: number): string {
  if (!seconds || seconds <= 0) return "—";

  if (seconds < 3600) {
    const minutes = Math.round(seconds / 60);

    return `${minutes} min`;
  }

  const hours = seconds / 3600;
  const rounded = Math.round(hours * 10) / 10;

  return `${rounded} hr`;
}

/**
 * Whether the sell side has a price to measure margin against.
 *
 * A shipment quoted at nothing is not a shipment sold for nothing, and every
 * margin shown against it would be a hundred percent.
 */
export function hasSellPrice(result: ShopResult | undefined): boolean {
  const total = result?.sellTotal;

  return total !== null && total !== undefined && total > 0;
}

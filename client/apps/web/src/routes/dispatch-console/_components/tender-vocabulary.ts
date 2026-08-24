import { formatDurationFromSeconds } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import {
  TENDER_MODE_LABEL,
  type TenderMode,
  type TenderStatus,
} from "@trenova/shared/types/tender";

/**
 * The board summary a move card chip renders. Mirrors DispatchBoardTenderSummary
 * from the console query.
 */
export type TenderChipSummary = {
  status: TenderStatus;
  mode: TenderMode;
  currentRank: number;
  offerCount: number;
  currentCarrierName: string;
  currentOfferExpiresAt?: number | null;
};

export type TenderChipTone = "info" | "active" | "attention";

export type TenderChipMeta = {
  label: string;
  tone: TenderChipTone;
};

/**
 * Condenses a live tender into the one line a board card has room for. Active
 * waterfalls lead with where the ladder stands; the failure states lead with
 * the word that demands attention.
 */
export function tenderChipMeta(summary: TenderChipSummary): TenderChipMeta {
  switch (summary.status) {
    case "Active": {
      const position =
        summary.offerCount > 0 ? ` ${summary.currentRank}/${summary.offerCount}` : "";
      const carrier = summary.currentCarrierName ? ` · ${summary.currentCarrierName}` : "";
      return { label: `${TENDER_MODE_LABEL[summary.mode]}${position}${carrier}`, tone: "info" };
    }
    case "Accepted":
      return { label: "Tender accepted", tone: "active" };
    case "NeedsReview":
      return { label: "Tender needs review", tone: "attention" };
    case "Exhausted":
      return { label: "Tender exhausted", tone: "attention" };
    case "Canceled":
      return { label: "Tender canceled", tone: "attention" };
  }
}

/**
 * A decimal-string rate rendered the way the console prints money, with the
 * per-mile qualifier when the method is distance-based. An unparseable string
 * renders verbatim rather than as NaN.
 */
export function formatOfferRate(rate: string, rateMethod: string): string {
  const amount = Number(rate);
  if (!Number.isFinite(amount)) return rate;
  return rateMethod === "PerMile" ? `${formatCurrency(amount)}/mi` : formatCurrency(amount);
}

/**
 * A compact countdown to an offer expiry. Unknown expiries render as nothing
 * rather than a bogus clock; an elapsed one reads as expired so a dispatcher
 * knows the waterfall is about to advance.
 */
export function formatOfferCountdown(
  expiresAt: number | null | undefined,
  nowSeconds: number,
): string {
  if (expiresAt == null || expiresAt <= 0) {
    return "";
  }
  const remaining = expiresAt - nowSeconds;
  if (remaining <= 0) {
    return "expired";
  }
  return `${formatDurationFromSeconds(remaining)} left`;
}

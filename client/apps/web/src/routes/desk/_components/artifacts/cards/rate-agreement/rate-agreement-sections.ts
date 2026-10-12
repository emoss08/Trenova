import { Resource } from "@trenova/shared/types/permission";

/**
 * The agreement card's sections built from records other than the agreement,
 * by the resource each reads: the loads it has priced are rate quotes. Its own
 * module, so the skeleton reads it without the card's code.
 */
export const RATE_AGREEMENT_SECTIONS = {
  pricing: Resource.RateQuote,
} as const;

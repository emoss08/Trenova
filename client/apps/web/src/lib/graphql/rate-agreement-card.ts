import { DeskRateAgreementPricingDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

/** How much freight an agreement has priced: the loads it rated, and when it last rated one. */
export type RateAgreementPricing = { loads: number; lastRatedAt: number | null };

/**
 * The loads a rate agreement has priced. A quote rated for a shipment is one
 * load priced; quotes, what-ifs and simulations price nothing and are left out.
 */
export async function fetchRateAgreementPricing(
  rateAgreementId: string,
  options?: { signal?: AbortSignal },
): Promise<RateAgreementPricing> {
  const data = await requestGraphQL({
    document: DeskRateAgreementPricingDocument,
    operationName: "DeskRateAgreementPricing",
    variables: {
      input: {
        first: 1,
        fieldFilters: [
          { field: "rateAgreementId", operator: "eq", value: rateAgreementId },
          { field: "purpose", operator: "eq", value: "Rating" },
          { field: "outcome", operator: "eq", value: "Rated" },
        ],
        sort: [{ field: "ratedAt", direction: "desc" }],
      },
    },
    signal: options?.signal,
  });
  const latest = data.rateQuotes.edges[0]?.node ?? null;
  return {
    loads: data.rateQuotes.totalCount ?? data.rateQuotes.edges.length,
    lastRatedAt: latest?.ratedAt ?? null,
  };
}

import { fetchRateAgreementPricing } from "@/lib/graphql/rate-agreement-card";
import { nextLaneOffset } from "@/lib/rate-agreement";
import { infiniteQueryOptions } from "@tanstack/react-query";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const rateAgreement = createQueryKeys("rateAgreement", {
  /** One agreement with its accessorials and fuel binding, without its lanes. */
  get: (rateAgreementId: string) => ({
    queryKey: [rateAgreementId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.rateAgreementService.getWithoutLanes(rateAgreementId, { signal }),
  }),
  /**
   * The key the agreement's lanes are read under, a page at a time, most
   * specific first; the reader pages with `useInfiniteQuery`.
   */
  lanes: (rateAgreementId: string) => ({
    queryKey: [rateAgreementId],
  }),
  /** The agreement's versions, each the terms as they stood when it was cut. */
  versions: (rateAgreementId: string) => ({
    queryKey: [rateAgreementId],
    queryFn: async () => apiService.rateAgreementService.listVersions(rateAgreementId),
  }),
  /** The loads the agreement has priced, and when it last priced one. */
  pricing: (rateAgreementId: string) => ({
    queryKey: [rateAgreementId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchRateAgreementPricing(rateAgreementId, { signal }),
  }),
});

/**
 * An agreement's live lanes, most specific first, read `pageSize` at a time:
 * the first page carries how many lanes it prices and how many it retired.
 */
export function rateAgreementLanesQuery(rateAgreementId: string, pageSize: number) {
  return infiniteQueryOptions({
    queryKey: [...rateAgreement.lanes(rateAgreementId).queryKey, pageSize],
    queryFn: ({ pageParam, signal }) =>
      apiService.rateAgreementService.listLanes(
        rateAgreementId,
        { limit: pageSize, offset: pageParam },
        { signal },
      ),
    initialPageParam: 0,
    getNextPageParam: nextLaneOffset,
  });
}

import {
  fetchCarrierEligibility,
  fetchCarrierIntel,
  fetchCarrierSettlements,
  fetchDeskCarrier,
} from "@/lib/graphql/carrier-card";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import { infiniteQueryOptions } from "@tanstack/react-query";

export const carrier = createQueryKeys("carrier", {
  card: (carrierId: string) => ({
    queryKey: ["card", carrierId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchDeskCarrier(carrierId, { signal }),
  }),
  eligibility: (carrierId: string) => ({
    queryKey: ["eligibility", carrierId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchCarrierEligibility(carrierId, { signal }),
  }),
  intel: (carrierId: string) => ({
    queryKey: ["intel", carrierId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchCarrierIntel(carrierId, { signal }),
  }),
  settlements: (carrierId: string) => ({
    queryKey: ["settlements", carrierId],
  }),
});

/** The carrier's settlements, newest first, a page at a time. */
export function carrierSettlementsQuery(carrierId: string, pageSize: number) {
  return infiniteQueryOptions({
    queryKey: [...carrier.settlements(carrierId).queryKey, pageSize],
    queryFn: ({ pageParam, signal }) =>
      fetchCarrierSettlements(carrierId, { first: pageSize, after: pageParam }, { signal }),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => (last.hasNextPage && last.endCursor ? last.endCursor : undefined),
  });
}

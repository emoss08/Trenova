import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/**
 * Ranked detention analytics are grouped rollups, so a generous ceiling keeps
 * the page's derived totals honest while still bounding the aggregation.
 */
export const DETENTION_STATS_LIMIT = 100;

export const detention = createQueryKeys("detention", {
  desk: () => ({
    queryKey: ["desk"],
    queryFn: async ({ signal }) => apiService.detentionService.desk({ signal }),
  }),
  byShipment: (shipmentId: string) => ({
    queryKey: ["byShipment", shipmentId],
    queryFn: async ({ signal }) => apiService.detentionService.byShipment(shipmentId, { signal }),
  }),
  occurrence: (id: string) => ({
    queryKey: ["occurrence", id],
    queryFn: async ({ signal }) => apiService.detentionService.getOccurrence(id, { signal }),
  }),
  facilities: (from: number, to: number) => ({
    queryKey: ["facilities", from, to],
    queryFn: async ({ signal }) =>
      apiService.detentionAnalyticsService.facilities(
        {
          from,
          to,
          limit: DETENTION_STATS_LIMIT,
        },
        { signal },
      ),
  }),
  customers: (from: number, to: number) => ({
    queryKey: ["customers", from, to],
    queryFn: async ({ signal }) =>
      apiService.detentionAnalyticsService.customers(
        {
          from,
          to,
          limit: DETENTION_STATS_LIMIT,
        },
        { signal },
      ),
  }),
  waivers: (from: number, to: number) => ({
    queryKey: ["waivers", from, to],
    queryFn: async ({ signal }) =>
      apiService.detentionAnalyticsService.waivers({ from, to }, { signal }),
  }),
  disputePacket: (id: string) => ({
    queryKey: ["disputePacket", id],
    queryFn: async ({ signal }) => apiService.detentionService.disputePacket(id, { signal }),
  }),
  policyById: (id: string) => ({
    queryKey: ["policyById", id],
    queryFn: async ({ signal }) => apiService.detentionPolicyService.getById(id, { signal }),
  }),
});

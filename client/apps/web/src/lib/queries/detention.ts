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
    queryFn: async () => apiService.detentionService.desk(),
  }),
  byShipment: (shipmentId: string) => ({
    queryKey: ["byShipment", shipmentId],
    queryFn: async () => apiService.detentionService.byShipment(shipmentId),
  }),
  occurrence: (id: string) => ({
    queryKey: ["occurrence", id],
    queryFn: async () => apiService.detentionService.getOccurrence(id),
  }),
  facilities: (from: number, to: number) => ({
    queryKey: ["facilities", from, to],
    queryFn: async () =>
      apiService.detentionAnalyticsService.facilities({
        from,
        to,
        limit: DETENTION_STATS_LIMIT,
      }),
  }),
  customers: (from: number, to: number) => ({
    queryKey: ["customers", from, to],
    queryFn: async () =>
      apiService.detentionAnalyticsService.customers({
        from,
        to,
        limit: DETENTION_STATS_LIMIT,
      }),
  }),
  waivers: (from: number, to: number) => ({
    queryKey: ["waivers", from, to],
    queryFn: async () => apiService.detentionAnalyticsService.waivers({ from, to }),
  }),
  disputePacket: (id: string) => ({
    queryKey: ["disputePacket", id],
    queryFn: async () => apiService.detentionService.disputePacket(id),
  }),
  policyById: (id: string) => ({
    queryKey: ["policyById", id],
    queryFn: async () => apiService.detentionPolicyService.getById(id),
  }),
});

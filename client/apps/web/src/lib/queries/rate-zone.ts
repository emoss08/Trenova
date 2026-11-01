import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const rateZone = createQueryKeys("rateZone", {
  /**
   * A zone with every place it covers, as its form edits it. The version is in the
   * key so a save that lands refetches instead of re-seating the form from a stale
   * record.
   */
  record: (rateZoneId: string, version?: number) => ({
    queryKey: [rateZoneId, version],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.rateZoneService.getById(rateZoneId, { signal }),
  }),
});

import {
  fetchDeskCarrierSettlement,
  fetchDeskDriverSettlement,
} from "@/lib/graphql/settlement-card";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const settlement = createQueryKeys("settlement", {
  driverCard: (settlementId: string) => ({
    queryKey: ["driver-card", settlementId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchDeskDriverSettlement(settlementId, { signal }),
  }),
  carrierCard: (settlementId: string) => ({
    queryKey: ["carrier-card", settlementId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchDeskCarrierSettlement(settlementId, { signal }),
  }),
});

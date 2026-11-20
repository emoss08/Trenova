import { fetchCarrierSettlementControl } from "@/lib/graphql/carrier-settlement";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const carrierSettlementControl = createQueryKeys("carrierSettlementControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchCarrierSettlementControl({ signal }),
  }),
});

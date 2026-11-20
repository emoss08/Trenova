import { fetchSettlementControl } from "@/lib/graphql/driver-settlement";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const settlementControl = createQueryKeys("settlementControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) => fetchSettlementControl({ signal }),
  }),
});

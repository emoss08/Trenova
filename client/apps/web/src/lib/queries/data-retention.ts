import { getDataRetention } from "@/services/data-retention";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const dataRetention = createQueryKeys("dataRetention", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async ({ signal }) => getDataRetention({ signal }),
  }),
});

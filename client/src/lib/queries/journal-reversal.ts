import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const journalReversal = createQueryKeys("journalReversal", {
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async () => apiService.journalReversalService.getById(id),
  }),
});

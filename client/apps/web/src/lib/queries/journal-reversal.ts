import { journalReversalService } from "@/services/journal-reversal";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const journalReversal = createQueryKeys("journalReversal", {
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async () => journalReversalService.getById(id),
  }),
});

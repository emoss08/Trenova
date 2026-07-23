import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const manualJournal = createQueryKeys("manualJournal", {
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async () => apiService.manualJournalService.getById(id),
  }),
});

import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const journalEntry = createQueryKeys("journalEntry", {
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async () => apiService.journalEntryService.getById(id),
  }),
  bySource: (sourceType: string, sourceId: string) => ({
    queryKey: ["bySource", sourceType, sourceId],
    queryFn: async () => apiService.journalEntryService.getBySource(sourceType, sourceId),
  }),
});

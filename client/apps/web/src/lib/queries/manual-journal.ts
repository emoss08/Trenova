import { fetchDeskManualJournal } from "@/lib/graphql/journal-cards";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const manualJournal = createQueryKeys("manualJournal", {
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async ({ signal }) => apiService.manualJournalService.getById(id, { signal }),
  }),
  card: (id: string) => ({
    queryKey: ["card", id],
    queryFn: async ({ signal }) => fetchDeskManualJournal(id, { signal }),
  }),
});

import { fetchDeskManualJournal } from "@/lib/graphql/journal-cards";
import { manualJournalService } from "@/services/manual-journal";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const manualJournal = createQueryKeys("manualJournal", {
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async () => manualJournalService.getById(id),
  }),
  card: (id: string) => ({
    queryKey: ["card", id],
    queryFn: async ({ signal }) => fetchDeskManualJournal(id, { signal }),
  }),
});

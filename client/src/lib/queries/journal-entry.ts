import {
  fetchJournalEntriesBySource,
  fetchJournalEntry,
  fetchJournalSourceByObject,
} from "@/lib/graphql/journal-entry";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const journalEntry = createQueryKeys("journalEntry", {
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async () => fetchJournalEntry(id),
  }),
  bySource: (sourceType: string, sourceId: string) => ({
    queryKey: ["bySource", sourceType, sourceId],
    queryFn: async () => fetchJournalEntriesBySource(sourceType, sourceId),
  }),
  sourceByObject: (sourceType: string, sourceId: string) => ({
    queryKey: ["sourceByObject", sourceType, sourceId],
    queryFn: async () => fetchJournalSourceByObject(sourceType, sourceId),
  }),
});

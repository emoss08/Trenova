import {
  fetchJournalEntriesBySource,
  fetchJournalEntry,
  fetchJournalSourceByObject,
} from "@/lib/graphql/journal-entry";
import { fetchJournalReviewSummary } from "@/lib/graphql/journal-review";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const journalEntry = createQueryKeys("journalEntry", {
  get: (id: string) => ({
    queryKey: ["get", id],
    queryFn: async ({ signal }) => fetchJournalEntry(id, { signal }),
  }),
  bySource: (sourceType: string, sourceId: string) => ({
    queryKey: ["bySource", sourceType, sourceId],
    queryFn: async ({ signal }) => fetchJournalEntriesBySource(sourceType, sourceId, { signal }),
  }),
  reviewSummary: () => ({
    queryKey: ["reviewSummary"],
    queryFn: async ({ signal }) => fetchJournalReviewSummary({ signal }),
  }),
  sourceByObject: (sourceType: string, sourceId: string) => ({
    queryKey: ["sourceByObject", sourceType, sourceId],
    queryFn: async ({ signal }) => fetchJournalSourceByObject(sourceType, sourceId, { signal }),
  }),
});

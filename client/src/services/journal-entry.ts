import { api } from "@/lib/api";
import type { JournalEntry } from "@/types/journal-entry";

export class JournalEntryService {
  async getById(id: string) {
    return api.get<JournalEntry>(`/accounting/journal-entries/${id}/`);
  }

  async getBySource(sourceObjectType: string, sourceObjectId: string) {
    return api.get<JournalEntry[]>(
      `/accounting/journal-entries/source/${sourceObjectType}/${sourceObjectId}/`,
    );
  }
}

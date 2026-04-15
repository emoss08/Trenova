import { api } from "@/lib/api";
import type { ManualJournal } from "@/types/manual-journal";

export class ManualJournalService {
  async getById(id: string) {
    return api.get<ManualJournal>(`/accounting/manual-journals/${id}/`);
  }

  async createDraft(data: Partial<ManualJournal>) {
    return api.post<ManualJournal>("/accounting/manual-journals/drafts/", data);
  }

  async updateDraft(id: string, data: Partial<ManualJournal>) {
    return api.put<ManualJournal>(`/accounting/manual-journals/drafts/${id}/`, data);
  }

  async submit(id: string) {
    return api.post<ManualJournal>(`/accounting/manual-journals/${id}/submit/`);
  }

  async approve(id: string) {
    return api.post<ManualJournal>(`/accounting/manual-journals/${id}/approve/`);
  }

  async post(id: string) {
    return api.post<ManualJournal>(`/accounting/manual-journals/${id}/post/`);
  }

  async reject(id: string, reason: string) {
    return api.post<ManualJournal>(`/accounting/manual-journals/${id}/reject/`, {
      rejectionReason: reason,
    });
  }

  async cancel(id: string, reason: string) {
    return api.post<ManualJournal>(`/accounting/manual-journals/${id}/cancel/`, {
      cancelReason: reason,
    });
  }
}

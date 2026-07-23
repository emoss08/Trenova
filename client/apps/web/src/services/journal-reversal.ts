import { api } from "@/lib/api";
import type { JournalReversal } from "@/types/journal-reversal";

export class JournalReversalService {
  async getById(id: string) {
    return api.get<JournalReversal>(`/accounting/journal-reversals/${id}/`);
  }

  async create(data: Partial<JournalReversal>) {
    return api.post<JournalReversal>("/accounting/journal-reversals/", data);
  }

  async approve(id: string) {
    return api.post<JournalReversal>(`/accounting/journal-reversals/${id}/approve/`);
  }

  async post(id: string) {
    return api.post<JournalReversal>(`/accounting/journal-reversals/${id}/post/`);
  }

  async reject(id: string, reason: string) {
    return api.post<JournalReversal>(`/accounting/journal-reversals/${id}/reject/`, {
      rejectionReason: reason,
    });
  }

  async cancel(id: string, reason: string) {
    return api.post<JournalReversal>(`/accounting/journal-reversals/${id}/cancel/`, {
      cancelReason: reason,
    });
  }
}

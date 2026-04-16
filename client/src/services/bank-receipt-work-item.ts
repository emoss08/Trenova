import { z } from "zod";
import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import { bankReceiptWorkItemSchema, type BankReceiptWorkItem } from "@/types/bank-receipt-work-item";

const workItemListSchema = z.array(bankReceiptWorkItemSchema);

export class BankReceiptWorkItemService {
  async list(params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params).toString()}` : "";
    const response = await api.get<BankReceiptWorkItem[]>(
      `/accounting/bank-receipt-work-items/${query}`,
    );
    return safeParse(workItemListSchema, response, "BankReceiptWorkItemList");
  }

  async getById(id: string) {
    return api.get<BankReceiptWorkItem>(`/accounting/bank-receipt-work-items/${id}/`);
  }

  async assign(id: string, userId: string) {
    return api.post<BankReceiptWorkItem>(
      `/accounting/bank-receipt-work-items/${id}/assign/`,
      { userId },
    );
  }

  async startReview(id: string) {
    return api.post<BankReceiptWorkItem>(
      `/accounting/bank-receipt-work-items/${id}/start-review/`,
    );
  }

  async resolve(id: string, data: { resolutionType: string; resolutionNote: string }) {
    return api.post<BankReceiptWorkItem>(
      `/accounting/bank-receipt-work-items/${id}/resolve/`,
      data,
    );
  }

  async dismiss(id: string, data: { resolutionNote: string }) {
    return api.post<BankReceiptWorkItem>(
      `/accounting/bank-receipt-work-items/${id}/dismiss/`,
      data,
    );
  }
}

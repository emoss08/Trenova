import { api } from "@/lib/api";
import type { BankReceipt, MatchSuggestion, ReconciliationSummary } from "@/types/bank-receipt";

export class BankReceiptService {
  async getSummary() {
    return api.get<ReconciliationSummary>("/accounting/bank-receipts/summary/");
  }

  async getExceptions(params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params).toString()}` : "";
    return api.get<BankReceipt[]>(`/accounting/bank-receipts/exceptions/${query}`);
  }

  async getById(id: string) {
    return api.get<BankReceipt>(`/accounting/bank-receipts/${id}/`);
  }

  async getSuggestions(id: string) {
    return api.get<MatchSuggestion[]>(`/accounting/bank-receipts/${id}/suggestions/`);
  }

  async importReceipts(data: FormData) {
    return api.post<BankReceipt[]>("/accounting/bank-receipts/", data);
  }

  async match(id: string, customerPaymentId: string) {
    return api.post<BankReceipt>(`/accounting/bank-receipts/${id}/match/`, {
      customerPaymentId,
    });
  }
}

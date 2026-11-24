import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  bankReceiptCardSchema,
  type BankReceipt,
  type BankReceiptCard,
  type MatchSuggestion,
  type ReconciliationSummary,
} from "@/types/bank-receipt";

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

  /** The receipt with its matched payment and queue entry, as the Desk card draws it. */
  async getCard(id: string, options: { signal?: AbortSignal } = {}): Promise<BankReceiptCard> {
    const response = await api.get<BankReceiptCard>(`/accounting/bank-receipts/${id}/card/`, {
      signal: options.signal,
    });

    return safeParse(bankReceiptCardSchema, response, "Bank receipt card");
  }

  async getSuggestions(id: string) {
    return api.get<MatchSuggestion[]>(`/accounting/bank-receipts/${id}/suggestions/`);
  }

  async match(id: string, paymentId: string) {
    return api.post<BankReceipt>(`/accounting/bank-receipts/${id}/match/`, { paymentId });
  }
}

export const bankReceiptService = new BankReceiptService();

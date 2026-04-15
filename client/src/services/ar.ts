import { api } from "@/lib/api";
import type { AgingSummary } from "@/types/ar-aging";
import type { AROpenItem } from "@/types/ar-open-items";
import type { CustomerLedgerEntry } from "@/types/customer-ledger";
import type { CustomerStatement } from "@/types/customer-statement";

export class ARService {
  async getAging(params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params).toString()}` : "";
    return api.get<AgingSummary>(`/accounting/accounts-receivable/aging/${query}`);
  }

  async getCustomerLedger(customerId: string, params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params).toString()}` : "";
    return api.get<CustomerLedgerEntry[]>(
      `/accounting/accounts-receivable/customers/${customerId}/ledger/${query}`,
    );
  }

  async getOpenItems(params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params).toString()}` : "";
    return api.get<AROpenItem[]>(`/accounting/accounts-receivable/open-items/${query}`);
  }

  async getCustomerStatement(customerId: string, params?: Record<string, string>) {
    const query = params ? `?${new URLSearchParams(params).toString()}` : "";
    return api.get<CustomerStatement>(
      `/accounting/accounts-receivable/customers/${customerId}/statement/${query}`,
    );
  }
}

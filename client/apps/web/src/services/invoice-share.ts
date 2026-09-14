import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  invoiceShareCandidateListSchema,
  invoiceShareListSchema,
  shareInvoiceResultSchema,
  type ShareInvoicePayload,
} from "@trenova/shared/types/invoice-share";

export class InvoiceShareService {
  public async list(invoiceId: string) {
    const response = await api.get(`/billing/invoices/${invoiceId}/shares/`);
    const parsed = await safeParse(invoiceShareListSchema, response, "InvoiceShareList");
    return parsed.shares;
  }

  public async candidates(invoiceId: string, query: string, signal?: AbortSignal) {
    const params = new URLSearchParams({ limit: "10", offset: "0" });
    const trimmed = query.trim();
    if (trimmed) {
      params.set("query", trimmed);
    }
    const response = await api.get(
      `/billing/invoices/${invoiceId}/shares/candidates/?${params.toString()}`,
      { signal },
    );
    const parsed = await safeParse(
      invoiceShareCandidateListSchema,
      response,
      "InvoiceShareCandidateList",
    );
    return parsed.results;
  }

  public async share(invoiceId: string, payload: ShareInvoicePayload) {
    const response = await api.post(`/billing/invoices/${invoiceId}/shares/`, {
      userIds: payload.userIds,
      note: payload.note.trim(),
      tab: payload.tab,
    });
    return safeParse(shareInvoiceResultSchema, response, "ShareInvoiceResult");
  }
}

import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import { createLimitOffsetResponse } from "@/types/server";
import { invoiceSchema, type Invoice } from "@/types/invoice";

const invoiceListSchema = createLimitOffsetResponse(invoiceSchema);

export class InvoiceService {
  public async list(params?: Record<string, string>) {
    const endpoint = params
      ? `/billing/invoices/?${new URLSearchParams(params).toString()}`
      : "/billing/invoices/";
    const response = await api.get(endpoint);
    return safeParse(invoiceListSchema, response, "InvoiceList");
  }

  public async getById(id: string) {
    const response = await api.get<Invoice>(`/billing/invoices/${id}/`);
    return safeParse(invoiceSchema, response, "Invoice");
  }

  public async post(id: string) {
    const response = await api.post<Invoice>(`/billing/invoices/${id}/post/`, {});
    return safeParse(invoiceSchema, response, "Invoice");
  }
}

import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { createLimitOffsetResponse } from "@trenova/shared/types/server";
import {
  invoiceEmailAttemptSchema,
  generateInvoicePdfResultSchema,
  invoiceSchema,
  invoiceSendPlanSchema,
  invoiceSendResultSchema,
  updateInvoiceDraftSchema,
  type Invoice,
  type UpdateInvoiceDraft,
} from "@trenova/shared/types/invoice";

const invoiceListSchema = createLimitOffsetResponse(invoiceSchema);
const invoiceEmailAttemptListSchema = createLimitOffsetResponse(invoiceEmailAttemptSchema);

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

  public async createFromShipments(shipmentIds: string[]) {
    const response = await api.post<Invoice>("/billing/invoices/from-shipments/", { shipmentIds });
    return safeParse(invoiceSchema, response, "Invoice");
  }

  public async updateDraft(id: string, data: UpdateInvoiceDraft) {
    const payload = updateInvoiceDraftSchema.parse(data);
    const response = await api.patch<Invoice>(`/billing/invoices/${id}/`, payload);
    return safeParse(invoiceSchema, response, "Invoice");
  }

  public async generatePdf(id: string) {
    const response = await api.post(`/billing/invoices/${id}/generate-pdf/`, {});
    return safeParse(generateInvoicePdfResultSchema, response, "GenerateInvoicePdfResult");
  }

  public async getSendPlan(id: string) {
    const response = await api.get(`/billing/invoices/${id}/send-plan/`);
    return safeParse(invoiceSendPlanSchema, response, "InvoiceSendPlan");
  }

  public async send(id: string) {
    const response = await api.post(`/billing/invoices/${id}/send/`, {});
    return safeParse(invoiceSendResultSchema, response, "InvoiceSendResult");
  }

  public async resend(id: string) {
    const response = await api.post(`/billing/invoices/${id}/resend/`, {});
    return safeParse(invoiceSendResultSchema, response, "InvoiceSendResult");
  }

  public async listEmailAttempts(id: string, params?: { limit?: number; offset?: number }) {
    const search = params ? new URLSearchParams(toStringParams(params)).toString() : "";
    const query = search ? `?${search}` : "";
    const response = await api.get(`/billing/invoices/${id}/email-attempts/${query}`);
    return safeParse(invoiceEmailAttemptListSchema, response, "InvoiceEmailAttempts");
  }

  public async post(id: string) {
    const response = await api.post<Invoice>(`/billing/invoices/${id}/post/`, {});
    return safeParse(invoiceSchema, response, "Invoice");
  }
}

function toStringParams(params: { limit?: number; offset?: number }) {
  return Object.fromEntries(
    Object.entries(params)
      .filter(([, value]) => value !== undefined)
      .map(([key, value]) => [key, String(value)]),
  );
}

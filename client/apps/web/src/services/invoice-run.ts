import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { createLimitOffsetResponse } from "@trenova/shared/types/server";
import {
  commitInvoiceRunResultSchema,
  invoiceRunSchema,
  type AdjustMembershipInput,
  type InvoiceRun,
  type PreviewInvoiceRunInput,
} from "@trenova/shared/types/invoice-run";
import {
  openStatementListSchema,
  openStatementSchema,
  type OpenStatement,
} from "@trenova/shared/types/statement";

const invoiceRunListSchema = createLimitOffsetResponse(invoiceRunSchema);

export class InvoiceRunService {
  public async list(params?: Record<string, string>) {
    const endpoint = params
      ? `/billing/invoice-runs/?${new URLSearchParams(params).toString()}`
      : "/billing/invoice-runs/";
    const response = await api.get(endpoint);
    return safeParse(invoiceRunListSchema, response, "InvoiceRunList");
  }

  public async getById(id: string) {
    const response = await api.get<InvoiceRun>(`/billing/invoice-runs/${id}/`);
    return safeParse(invoiceRunSchema, response, "InvoiceRun");
  }

  public async preview(input: PreviewInvoiceRunInput) {
    const response = await api.post<InvoiceRun>("/billing/invoice-runs/preview/", input);
    return safeParse(invoiceRunSchema, response, "InvoiceRun");
  }

  public async adjustMembership(id: string, input: AdjustMembershipInput) {
    const response = await api.patch<InvoiceRun>(
      `/billing/invoice-runs/${id}/membership/`,
      input,
    );
    return safeParse(invoiceRunSchema, response, "InvoiceRun");
  }

  public async commit(id: string) {
    const response = await api.post(`/billing/invoice-runs/${id}/commit/`, {});
    return safeParse(commitInvoiceRunResultSchema, response, "CommitInvoiceRunResult");
  }

  public async cancel(id: string, reason: string) {
    const response = await api.post<InvoiceRun>(`/billing/invoice-runs/${id}/cancel/`, {
      reason,
    });
    return safeParse(invoiceRunSchema, response, "InvoiceRun");
  }

  /** Every statement customer's open period, derived live from the billing queue. */
  public async listStatements() {
    const response = await api.get("/billing/statements/");
    return safeParse(openStatementListSchema, response, "OpenStatementList");
  }

  public async getStatement(customerId: string) {
    const response = await api.get<OpenStatement>(`/billing/statements/${customerId}/`);
    return safeParse(openStatementSchema, response, "OpenStatement");
  }

  /**
   * Bills an open period. The reason is recorded on the run and is required only
   * when the period has not closed yet; held billing-queue items stay approved
   * and uninvoiced for the next period.
   */
  public async billStatementNow(
    customerId: string,
    reason: string,
    heldBillingQueueItemIds: string[] = [],
  ) {
    const response = await api.post(`/billing/statements/${customerId}/bill/`, {
      reason,
      exclude: heldBillingQueueItemIds.map((billingQueueItemId) => ({
        billingQueueItemId,
        reason: "Held back by the biller before this statement billed",
      })),
    });
    return safeParse(commitInvoiceRunResultSchema, response, "CommitInvoiceRunResult");
  }
}
